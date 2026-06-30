.PHONY: help dev-up dev-down dev-up-all build-all test-all lint-all fmt-all clean \
        migrate-auth migrate-tenant migrate-workflow migrate-all seed-all \
        fe-install fe-dev-portal fe-dev-admin fe-test-all fe-build-all \
        check-tools generate-mocks

# ============================================================
# Variables
# ============================================================
SERVICES  := auth-service tenant-service workflow-service document-service notification-service
FRONTENDS := customer-portal admin-console

# ============================================================
# Help — auto-generated from ## comments
# ============================================================
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

# ============================================================
# Infrastructure
# ============================================================
dev-up: ## Start infrastructure (MySQL, MongoDB, LocalStack)
	docker compose up -d mysql mongodb localstack
	@echo "Waiting for services to be healthy..."
	@sleep 5

dev-down: ## Stop all docker services and remove volumes
	docker compose down -v

dev-up-all: ## Start infrastructure + all app services
	docker compose --profile app up -d

# ============================================================
# Go — Build / Test / Lint / Format
# ============================================================
build-all: ## Build all Go services
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		cd services/$$svc && go build ./... && cd ../..; \
	done

test-all: ## Run all tests (race detector + coverage)
	@for svc in $(SERVICES); do \
		echo "Testing $$svc..."; \
		cd services/$$svc && go test ./... -v -race -coverprofile=coverage.out && cd ../..; \
	done

lint-all: ## Lint all Go services with golangci-lint
	@for svc in $(SERVICES); do \
		echo "Linting $$svc..."; \
		cd services/$$svc && golangci-lint run ./... && cd ../..; \
	done

fmt-all: ## Format all Go code with gofmt
	@for svc in $(SERVICES); do \
		cd services/$$svc && gofmt -w . && cd ../..; \
	done

# ============================================================
# Migrations
# ============================================================
migrate-auth: ## Run auth service migrations
	cd services/auth-service && go run cmd/migrate/main.go

migrate-tenant: ## Run tenant service migrations
	cd services/tenant-service && go run cmd/migrate/main.go

migrate-workflow: ## Run workflow service migrations
	cd services/workflow-service && go run cmd/migrate/main.go

migrate-all: migrate-auth migrate-tenant migrate-workflow ## Run all SQL migrations

# ============================================================
# Seed
# ============================================================
seed-all: ## Seed databases with sample data
	@for svc in auth-service tenant-service workflow-service; do \
		echo "Seeding $$svc..."; \
		cd services/$$svc && go run cmd/seed/main.go && cd ../..; \
	done

# ============================================================
# Frontend
# ============================================================
fe-install: ## Install frontend dependencies
	cd frontend/customer-portal && npm install
	cd frontend/admin-console && npm install

fe-dev-portal: ## Start customer portal dev server (port 3000)
	cd frontend/customer-portal && npm run dev

fe-dev-admin: ## Start admin console dev server (port 3001)
	cd frontend/admin-console && npm run dev

fe-test-all: ## Run frontend tests (Vitest)
	cd frontend/customer-portal && npm test -- --run
	cd frontend/admin-console && npm test -- --run

fe-build-all: ## Build frontend apps for production
	cd frontend/customer-portal && npm run build
	cd frontend/admin-console && npm run build

# ============================================================
# LocalStack / AWS
# ============================================================
setup-localstack: ## Create DynamoDB tables in LocalStack
	bash scripts/setup-localstack.sh

# ============================================================
# Tooling
# ============================================================
check-tools: ## Verify required tools are installed
	@which go       || (echo "ERROR: Go not installed — https://go.dev/dl/" && exit 1)
	@which docker   || (echo "ERROR: Docker not installed — https://docs.docker.com/get-docker/" && exit 1)
	@which node     || (echo "ERROR: Node.js not installed — https://nodejs.org/" && exit 1)
	@which golangci-lint || echo "WARN: golangci-lint not installed — run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
	@go version
	@node --version
	@docker --version
	@echo "Tool check complete."

generate-mocks: ## Generate mocks for all services (requires mockery/moq)
	@for svc in $(SERVICES); do \
		cd services/$$svc && go generate ./... && cd ../..; \
	done

# ============================================================
# Clean
# ============================================================
clean: ## Clean build artifacts and coverage files
	@for svc in $(SERVICES); do \
		cd services/$$svc && go clean && rm -f coverage.out && cd ../..; \
	done
	@rm -rf frontend/customer-portal/dist frontend/admin-console/dist
	@echo "Clean complete."
