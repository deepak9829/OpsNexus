# Integration Tests

Integration tests verify that service adapters work correctly against real infrastructure (databases, caches, message queues). They are intentionally excluded from the default `go test ./...` run and must be opted into explicitly.

## Prerequisites

All integration tests require the Docker Compose infrastructure to be running. Start it from the repo root:

```bash
docker compose -f docker-compose.yml up -d mysql redis rabbitmq
```

Wait for services to be healthy before running tests:

```bash
docker compose ps   # all relevant containers should show "healthy"
```

## Build Tag

Every integration test file is guarded with the build tag:

```go
//go:build integration
```

This prevents them from running during normal unit test passes.

## Running Integration Tests

```bash
# Run all integration tests
go test -tags=integration ./tests/integration/...

# Run with verbose output
go test -tags=integration -v ./tests/integration/...

# Run a single test function
go test -tags=integration -v -run TestUserRepository_CreateAndFind ./tests/integration/

# Run with timeout (integration tests can be slow)
go test -tags=integration -timeout 120s ./tests/integration/...
```

## Environment Variables

| Variable           | Default                                                                          | Description                       |
|--------------------|----------------------------------------------------------------------------------|-----------------------------------|
| `TEST_MYSQL_DSN`   | `auth_user:auth_pass@tcp(localhost:3306)/auth_db?charset=utf8mb4&parseTime=True&loc=UTC` | Auth service DB connection string |
| `TEST_WF_MYSQL_DSN`| `workflow_user:workflow_pass@tcp(localhost:3307)/workflow_db?charset=utf8mb4&parseTime=True&loc=UTC` | Workflow service DB DSN           |

Override them at runtime if your local ports differ:

```bash
TEST_MYSQL_DSN="root:secret@tcp(localhost:13306)/auth_db?parseTime=True" \
  go test -tags=integration -v ./tests/integration/...
```

## CI

In CI, the integration tests run in a separate job that spins up Docker Compose services before invoking `go test`. See `.github/workflows/integration.yml`.

## Test Isolation

Each test creates its own data using random UUIDs and cleans up after itself in a `t.Cleanup` callback. Tests within a package may run in parallel when they do not share mutable state.
