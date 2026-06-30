# Local Development Guide — OpsNexus

Everything you need to get OpsNexus running on your machine from scratch. Follow these steps in order on first setup. After that, the daily workflow is covered in the final section.

---

## Prerequisites

Install these before starting. Exact minimum versions matter — older versions may work but are not tested.

| Tool | Minimum Version | Install |
|------|----------------|---------|
| Go | 1.22 | https://go.dev/dl/ |
| Node.js | 20 LTS | https://nodejs.org or `nvm install 20` |
| Docker Desktop | Latest | https://docs.docker.com/get-docker/ |
| golangci-lint | 1.57+ | `brew install golangci-lint` or see https://golangci-lint.run/usage/install/ |
| make | any | Pre-installed on macOS/Linux |

Verify your setup:
```bash
go version          # should show go1.22.x or higher
node --version      # should show v20.x.x or higher
docker --version    # should show Docker version 24.x.x or higher
golangci-lint --version
```

---

## First-Time Setup

### 1. Clone the repository

```bash
git clone https://github.com/your-org/opsnexus.git
cd opsnexus
```

### 2. Set up Go workspace

The monorepo uses Go workspaces so that local services can reference each other without publishing to a registry:

```bash
# This file should already exist in the repo, but if not:
go work sync
```

### 3. Install Go dependencies for each service

```bash
# Install all at once from repo root
make go-tidy-all

# Or manually per service
cd services/auth && go mod tidy
cd services/cases && go mod tidy
cd services/documents && go mod tidy
cd services/workflows && go mod tidy
cd services/notifications && go mod tidy
```

### 4. Install frontend dependencies

```bash
# Install all at once
make npm-install-all

# Or manually per app
cd apps/web && npm install
cd apps/admin && npm install
```

### 5. Copy and configure environment files

Each service and app has a `.env.example`. Copy it to `.env` and fill in values:

```bash
# Services
cp services/auth/.env.example services/auth/.env
cp services/cases/.env.example services/cases/.env
cp services/documents/.env.example services/documents/.env
cp services/workflows/.env.example services/workflows/.env
cp services/notifications/.env.example services/notifications/.env

# Frontend apps
cp apps/web/.env.example apps/web/.env.local
cp apps/admin/.env.example apps/admin/.env.local
```

For local development, the default values in `.env.example` work without modification. The database connection strings point to Docker Compose services. Do not change them unless you have a reason.

The one value you must set:
```bash
# In services/auth/.env
JWT_SECRET=some-random-string-at-least-32-characters-long
```

### 6. Start infrastructure with Docker Compose

```bash
make dev-up
```

This starts:
- MySQL 8.0 on port 3306
- MongoDB 7.0 on port 27017
- LocalStack (DynamoDB + S3) on port 4566

Wait ~30 seconds for all services to be healthy:
```bash
docker compose ps
# All services should show "healthy"
```

If a service shows "unhealthy" or "starting", wait another 30 seconds and check again.

### 7. Run database migrations

```bash
make migrate-all
```

This runs the numbered SQL migration files for each service against the local MySQL instance. MongoDB indexes are created on service startup.

### 8. Verify the setup

```bash
make test-unit-all
# All tests should pass
```

If tests fail at this point, check the troubleshooting section.

---

## Starting Services

### All services at once

```bash
# Start all Go services in the background
make services-start

# Start all frontend dev servers
make frontend-start
```

### Individual services

Each service can be started independently. Run from the repo root or the service directory:

```bash
# Auth Service
cd services/auth && go run ./cmd/server
# or: make run-auth

# Case Service
cd services/cases && go run ./cmd/server
# or: make run-cases

# Document Service
cd services/documents && go run ./cmd/server
# or: make run-documents

# Workflow Service
cd services/workflows && go run ./cmd/server
# or: make run-workflows

# Notification Service
cd services/notifications && go run ./cmd/server
# or: make run-notifications
```

Default ports:

| Service | Port |
|---------|------|
| auth-service | 8081 |
| case-service | 8082 |
| document-service | 8083 |
| workflow-service | 8084 |
| notification-service | 8085 |
| apps/web (Vite) | 3000 |
| apps/admin (Vite) | 3001 |

### Frontend dev servers

```bash
cd apps/web && npm run dev      # http://localhost:3000
cd apps/admin && npm run dev    # http://localhost:3001
```

### Starting with hot reload (Air)

Install `air` for Go hot reload:
```bash
go install github.com/cosmtrek/air@latest
```

Then from a service directory:
```bash
air
```

Air watches for file changes and restarts the server. The `.air.toml` configuration is in each service directory.

---

## Running All Services with Docker Compose

To run the full stack including Go services in Docker:

```bash
make dev-stack-up
```

This builds and runs:
- All infrastructure (MySQL, MongoDB, LocalStack)
- All Go services

Note: Docker Compose services do not hot-reload on code changes. Use this for integration testing or to verify Docker builds. For active development, run services as local processes.

---

## Database Access

### MySQL

```bash
# Connect via docker compose
docker compose exec mysql mysql -u root -ppassword opsnexus_auth

# Or with any MySQL client
mysql -h 127.0.0.1 -P 3306 -u root -ppassword

# Databases
# opsnexus_auth       — Auth Service
# opsnexus_cases      — Case Service
# opsnexus_workflows  — Workflow Service
```

### MongoDB

```bash
# Connect via docker compose
docker compose exec mongodb mongosh mongodb://localhost:27017/opsnexus_documents

# Or with mongosh directly
mongosh mongodb://localhost:27017

# Database
# opsnexus_documents  — Document Service
```

Useful MongoDB commands:
```javascript
use opsnexus_documents
db.documents.find({tenantId: "your-tenant-id"}).limit(10)
db.documents.countDocuments()
```

### LocalStack DynamoDB

LocalStack runs on port 4566 and emulates AWS DynamoDB locally:

```bash
# Install AWS CLI if not already installed
pip install awscli-local  # awslocal wrapper

# List tables
awslocal dynamodb list-tables --endpoint-url http://localhost:4566

# Scan a table (use sparingly — for debugging only)
awslocal dynamodb scan \
  --endpoint-url http://localhost:4566 \
  --table-name notifications

# Query a table
awslocal dynamodb query \
  --endpoint-url http://localhost:4566 \
  --table-name notifications \
  --key-condition-expression "pk = :pk" \
  --expression-attribute-values '{":pk":{"S":"tenant-1#user-1"}}'
```

The LocalStack data does not persist between `make dev-down` and `make dev-up` restarts. Tables are created by the Notification Service on startup.

---

## Running Tests

### Unit tests (no infrastructure required)

```bash
# All services
make test-unit-all

# Single service
cd services/auth && go test ./...

# With coverage
cd services/auth && go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration tests (requires `make dev-up`)

```bash
# All services
make test-integration-all

# Single service
cd services/auth && go test -tags=integration ./...

# Required environment variables (set automatically by make targets)
export TEST_MYSQL_DSN="root:password@tcp(localhost:3306)/opsnexus_test?parseTime=true"
export TEST_MONGO_URI="mongodb://localhost:27017/opsnexus_test"
export TEST_DYNAMODB_ENDPOINT="http://localhost:4566"
```

The integration test database (`opsnexus_test`) is separate from the dev database. It is created automatically and cleared between test runs.

### Frontend tests

```bash
# All apps
make test-frontend-all

# Single app
cd apps/web && npm test
cd apps/web && npm test -- --run  # single run (no watch mode)
cd apps/web && npm test -- --coverage  # with coverage report
```

### E2E tests (requires full stack running)

```bash
# Start the full stack first
make dev-stack-up

# Run Playwright tests
cd tests/e2e && npx playwright test

# Run specific test file
npx playwright test auth.spec.ts

# Run with UI (interactive)
npx playwright test --ui
```

---

## Common Troubleshooting

### Port already in use

```bash
# Find what's using port 8081
lsof -i :8081
# or
sudo ss -tulpn | grep 8081

# Kill the process
kill -9 <PID>
```

If MySQL's port 3306 is in use, you likely have a local MySQL instance running:
```bash
# macOS (Homebrew)
brew services stop mysql
```

### MySQL connection refused

1. Check Docker Compose is running: `docker compose ps`
2. Check MySQL is healthy: `docker compose logs mysql | tail -20`
3. Wait 30 more seconds — MySQL takes time on first start
4. Try connecting manually: `docker compose exec mysql mysqladmin ping -u root -ppassword`

If MySQL shows "unable to connect: Access denied", the password in your `.env` doesn't match the one Docker Compose set up. The default is `password`. Check `docker-compose.yml` for the `MYSQL_ROOT_PASSWORD` value.

### MongoDB connection issues

```bash
docker compose logs mongodb | tail -20
# Check for "Listening on port 27017"

# Test connection
docker compose exec mongodb mongosh --eval "db.adminCommand('ping')"
```

### LocalStack DynamoDB issues

```bash
docker compose logs localstack | tail -30
# Check for "Ready."

# Test connectivity
curl http://localhost:4566/_localstack/health | jq .

# The Notification Service creates its DynamoDB tables on startup.
# If tables are missing, restart the notification service.
```

### Go build errors

```bash
# Sync workspace
go work sync

# Clear module cache if modules seem corrupted
go clean -modcache
make go-tidy-all

# Check for conflicting versions
go mod graph | grep "package-name"
```

### Frontend build errors

```bash
# Clear node_modules and reinstall
cd apps/web
rm -rf node_modules
npm install

# Check Node version
node --version  # must be 20+
nvm use 20      # if using nvm
```

### Database migration failures

```bash
# Check migration status
make migrate-status

# Re-run migrations manually
make migrate-all

# If a migration is in a dirty state (partially applied):
# Check the schema_migrations table
docker compose exec mysql mysql -u root -ppassword opsnexus_auth \
  -e "SELECT * FROM schema_migrations;"
# Fix the dirty flag if needed (advanced — ask before doing this)
```

---

## Development Workflow

### Daily workflow

```bash
# 1. Start infrastructure
make dev-up

# 2. Start services you're working on
make run-auth        # in terminal 1
make run-cases       # in terminal 2

# 3. Start frontend
cd apps/web && npm run dev  # in terminal 3

# 4. Run tests in watch mode
cd services/auth && go test ./... -count=1  # re-run manually
cd apps/web && npm test     # vitest in watch mode

# 5. When done
make dev-down
```

### Makefile quick reference

```bash
make dev-up              # Start Docker infrastructure
make dev-down            # Stop Docker infrastructure
make dev-stack-up        # Start full stack including Go services in Docker
make migrate-all         # Run all pending DB migrations
make run-{service}       # Run a specific service locally (e.g., make run-auth)
make build-all           # Build all services and frontend apps
make test-unit-all       # Run all unit tests
make test-integration-all # Run all integration tests
make test-all            # Run unit + integration tests
make lint-all            # Run all linters
make typecheck-all       # Run TypeScript type check on all apps
make docker-build-all    # Build all Docker images
make go-tidy-all         # go mod tidy on all services
make npm-install-all     # npm install on all apps
```

### Making a code change

1. Create a feature branch: `git checkout -b feat/your-feature-name`
2. Make changes
3. Run the relevant tests: `go test ./...` and/or `npm test -- --run`
4. Run linters: `golangci-lint run ./...` and/or `npm run lint`
5. Check TypeScript if you touched frontend: `npm run type-check`
6. Verify the full build: `make build-all`
7. Open a PR

Before opening a PR, work through the Definition of Done checklist in `skills/definition-of-done.md`.
