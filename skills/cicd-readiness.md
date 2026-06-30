# CI/CD Readiness Standards — OpsNexus

Standards for keeping the codebase in a continuously deployable state. Every commit to `main` should be deployable to production given the right approval. These standards define what "deployable" means.

---

## 1. Build Readiness Checklist

Before opening a PR, verify all of these pass locally:

**Go services (run in each service directory):**
- [ ] `go build ./...` — compiles with no errors or warnings
- [ ] `go vet ./...` — passes with no issues
- [ ] `golangci-lint run ./...` — passes (see `.golangci.yml` at project root for config)
- [ ] `go test ./...` — all unit tests pass
- [ ] `go test -tags=integration ./...` — all integration tests pass (requires `make dev-up`)

**Frontend apps (run in each app directory):**
- [ ] `npm run type-check` — TypeScript compiles with no errors
- [ ] `npm run lint` — ESLint passes with no errors (warnings are acceptable but should be resolved)
- [ ] `npm test -- --run` — all Vitest tests pass
- [ ] `npm run build` — production build succeeds

**Repository-level:**
- [ ] `make build-all` — builds all services and apps
- [ ] `make test-all` — runs all unit tests
- [ ] `make lint-all` — runs all linters

If any of these fail, the code is not ready for PR.

---

## 2. Docker Build Rules

### Multi-stage builds

Every Go service Dockerfile uses a multi-stage build to minimize image size and eliminate build tools from the runtime image:

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /app/server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/server .

# Run as non-root
RUN addgroup -g 1001 appgroup && adduser -u 1001 -G appgroup -D appuser
USER appuser

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:${SERVER_PORT:-8080}/health || exit 1

EXPOSE 8080
ENTRYPOINT ["./server"]
```

### Image size targets

- Go services: < 30MB (multi-stage + alpine achieves this)
- Frontend: served from S3+CloudFront, so container size is not a concern for the final artifact

### No secrets in images

Never `COPY` `.env` files, certificates, or credentials into an image. Secrets are injected at runtime via environment variables from AWS Secrets Manager (production) or Docker Compose env files (local dev).

Scan images with `docker scout` or `trivy` before publishing to ECR.

### Image tagging

Images are tagged with the git SHA for full traceability:

```bash
IMAGE_TAG=$(git rev-parse --short HEAD)
docker build -t opsnexus/auth-service:${IMAGE_TAG} -t opsnexus/auth-service:latest .
```

`latest` is only used for local development convenience. In CI and production, always reference the SHA-tagged image.

---

## 3. Environment Configuration

### Environment variable contract

Every service has a `.env.example` file in its directory that documents all supported environment variables, their purpose, whether they are required, and their default values:

```bash
# .env.example — Auth Service
# Copy to .env and fill in values for local development. Never commit .env.

# Server
SERVER_PORT=8080                          # default: 8080
SERVER_READ_TIMEOUT_SECONDS=30            # default: 30
SERVER_WRITE_TIMEOUT_SECONDS=30           # default: 30

# Database
MYSQL_DSN=root:password@tcp(localhost:3306)/opsnexus_auth?parseTime=true  # REQUIRED

# JWT
JWT_SECRET=                               # REQUIRED — at least 32 random bytes
JWT_EXPIRY_SECONDS=86400                  # default: 86400 (24h)

# Logging
LOG_LEVEL=info                            # default: info; options: debug, info, warn, error
```

When you add a new environment variable:
1. Add it to `.env.example` with a comment
2. Add it to `config/config.go`
3. Update the Kubernetes deployment manifest (or note it for the infrastructure team)
4. Update the AWS Secrets Manager path documentation

### Never commit credentials

`.env` files are in `.gitignore`. If you accidentally commit a credential, treat it as compromised immediately — rotate it, then remove it from git history.

### Production secrets

In production, secrets come from AWS Secrets Manager. The application startup reads a single secret JSON blob:

```json
{
  "MYSQL_DSN": "...",
  "JWT_SECRET": "...",
  "MONGO_URI": "..."
}
```

This is injected as environment variables by the EKS pod's init container or via External Secrets Operator. The application code does not change — it reads from environment variables regardless of the source.

---

## 4. Health Check Contract

Every service must satisfy this contract:

```
GET /health
Authorization: not required

200 OK
{
  "status": "ok",
  "service": "auth-service",
  "version": "abc1234"
}

503 Service Unavailable (when DB is unreachable)
{
  "status": "degraded",
  "service": "auth-service",
  "version": "abc1234",
  "checks": {
    "database": "dial tcp: connection refused"
  }
}
```

The health endpoint checks all critical dependencies:
- Database connectivity (MySQL ping, MongoDB ping, DynamoDB describe table)
- Any other required external services

Response time for `/health` must be under 200ms. If the DB ping takes longer, timeout it.

### Dockerfile HEALTHCHECK

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:${SERVER_PORT:-8080}/health || exit 1
```

### Kubernetes probes (future)

```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  failureThreshold: 3

livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30
  failureThreshold: 3
```

---

## 5. Database Migration Strategy

### Migration must be backward compatible

A deployment consists of two phases:
1. Run migrations (via init container or CI step)
2. Deploy new application code

Since there's a window where old code is still running after migrations are applied, migrations must be backward compatible:
- **Adding a column**: add it as nullable or with a default — old code ignores it, new code uses it
- **Renaming a column**: add the new column, deploy code that writes both, then remove the old column in a subsequent migration
- **Removing a column**: stop reading it in code first, deploy, then remove it in migration
- **Never** rename or remove a column in the same deployment as the code change that depends on it

### Migration runner

```go
// cmd/server/main.go — run migrations on startup
func runMigrations(db *sql.DB) error {
    m, err := migrate.New(
        "file://migrations",
        dsn,
    )
    if err != nil {
        return fmt.Errorf("creating migrator: %w", err)
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("running migrations: %w", err)
    }
    return nil
}
```

Test the migration against a copy of the production schema before any production deployment. This is currently a manual step until CI is fully configured.

---

## 6. GitHub Actions Pipeline Design (to be implemented)

The following pipeline design is documented here so implementation is straightforward when we wire up GitHub Actions. The design is final; only the implementation is pending.

### On every PR

```yaml
name: PR Checks

on: [pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: make lint-all

  typecheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: make typecheck-all

  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: make test-unit-all

  build:
    runs-on: ubuntu-latest
    needs: [lint, typecheck, unit-tests]
    steps:
      - uses: actions/checkout@v4
      - run: make docker-build-all
        env:
          VERSION: ${{ github.sha }}
```

### On merge to main

```yaml
name: Main Branch CI/CD

on:
  push:
    branches: [main]

jobs:
  # All PR checks re-run, plus:

  integration-tests:
    runs-on: ubuntu-latest
    services:
      mysql:
        image: mysql:8.0
        env: { MYSQL_ROOT_PASSWORD: password, MYSQL_DATABASE: opsnexus_test }
      mongodb:
        image: mongo:7.0
      localstack:
        image: localstack/localstack:latest
    steps:
      - uses: actions/checkout@v4
      - run: make test-integration-all

  push-to-ecr:
    runs-on: ubuntu-latest
    needs: [integration-tests]
    steps:
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: us-east-1
      - name: Push to ECR
        run: make docker-push-all
        env:
          VERSION: ${{ github.sha }}

  deploy-staging:
    runs-on: ubuntu-latest
    needs: [push-to-ecr]
    environment: staging
    steps:
      - run: make deploy-staging
        env:
          VERSION: ${{ github.sha }}
          KUBE_CONFIG: ${{ secrets.STAGING_KUBE_CONFIG }}
```

### On git tag (production release)

```yaml
name: Production Release

on:
  push:
    tags: ['v*']

jobs:
  deploy-production:
    runs-on: ubuntu-latest
    environment: production  # Requires manual approval in GitHub
    steps:
      - run: make deploy-production
        env:
          VERSION: ${{ github.ref_name }}
          KUBE_CONFIG: ${{ secrets.PROD_KUBE_CONFIG }}
```

### Required secrets (to be configured in GitHub)

| Secret | Purpose |
|--------|---------|
| `AWS_ROLE_ARN` | IAM role for ECR push |
| `STAGING_KUBE_CONFIG` | Kubeconfig for staging EKS cluster |
| `PROD_KUBE_CONFIG` | Kubeconfig for production EKS cluster |
| `SLACK_WEBHOOK` | Deployment notifications |
