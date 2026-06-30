# Test Strategy — OpsNexus

This document describes the overall testing approach for OpsNexus: what we test, how we test it, what tools we use, and what the acceptance bar looks like.

---

## Philosophy

Testing in OpsNexus serves three goals:
1. **Confidence that the software works.** Tests catch regressions before they reach users.
2. **Tenant isolation verification.** Multi-tenancy is the system's most critical invariant. Tests specifically verify that cross-tenant data leakage is impossible.
3. **Living documentation.** Well-named tests describe the system's intended behavior.

We do not write tests to hit a coverage number. We write tests because a bug in production is more expensive than a test. Untested code paths are risks, not features.

---

## Test Pyramid

```
         ▲
        ╱ ╲
       ╱E2E╲          10% — Critical user journeys via real browser
      ╱─────╲
     ╱ Integ ╲         20% — DB adapters, HTTP handlers, real infra
    ╱─────────╲
   ╱   Unit    ╲       70% — Application logic, React components
  ╱─────────────╲
```

| Layer | Target | Focus | Speed |
|-------|--------|-------|-------|
| Unit | 70% | Application logic, React components, pure functions | < 1ms each |
| Integration | 20% | DB adapters, HTTP handlers, multi-tenant isolation | < 500ms each |
| E2E | 10% | Critical user journeys end-to-end | 5–30s each |

Over-indexing on E2E tests creates a slow, flaky test suite. Over-indexing on unit tests only gives confidence in isolated logic, not in the system as a whole. The pyramid is the target balance.

---

## Tools

### Go Backend

| Tool | Purpose |
|------|---------|
| `testing` stdlib | Test runner |
| `testify/assert` | Non-fatal assertions |
| `testify/require` | Fatal assertions (stop test on failure) |
| `testify/mock` | Generated mocks (only for large interfaces) |
| `go test -tags=integration` | Integration test build tag |
| `k6` | Load testing and performance baselines |

### React Frontend

| Tool | Purpose |
|------|---------|
| Vitest | Test runner (faster Jest alternative) |
| @testing-library/react | Component test utilities |
| @testing-library/user-event | Simulated user interactions |
| MSW (Mock Service Worker) | HTTP request mocking |
| Playwright | E2E browser automation |

---

## What MUST Be Tested

These are non-negotiable. A feature is not done without them.

### Application Layer (Go)
- Every use case function (create, update, delete, transition) has unit tests
- Every validation function has table-driven tests covering valid, boundary, and invalid inputs
- Every error case is explicitly tested (not just the happy path)
- Domain rules (state machine transitions, permission checks) are unit tested

### Repository Layer (Go)
- Every repository method has an integration test (real DB, no mocks)
- Every repository method has a multi-tenant isolation test: wrong tenant returns not-found or empty, never another tenant's data
- CRUD operations round-trip correctly (save then find returns the same data)

### HTTP Handlers (Go)
- Authentication: unauthenticated request returns 401
- Authorization: authenticated-but-wrong-role request returns 403
- Validation: malformed or missing fields return 400 with field-level errors
- Success: valid request returns 2xx with correct response shape

### React Components
- Loading state is rendered
- Error state is rendered with a user-friendly message
- Success/populated state is rendered correctly
- User interactions (clicks, form submissions) trigger correct mutations
- Form validation messages appear for invalid inputs

### Security (all new endpoints)
- Unauthenticated → 401
- Expired JWT → 401
- Tampered JWT → 401
- Valid JWT, wrong tenant → 403 or empty (not other tenant's data)
- Valid JWT, insufficient role → 403
- Valid JWT, correct tenant and role → 200/201

---

## Integration Test Requirements

Integration tests run against real infrastructure started by `make dev-up`. They are tagged `//go:build integration` and excluded from the default `go test ./...` run.

### What qualifies as an integration test

- Tests that execute SQL queries against a real MySQL instance
- Tests that read/write to a real MongoDB collection
- Tests that interact with LocalStack DynamoDB
- Tests that start an HTTP server and make real HTTP requests against it
- Tests that verify multiple components work together (HTTP handler → service → repository → DB)

### Requirements per integration test

- Uses `//go:build integration` build tag
- Gets connection info from environment variables (`TEST_MYSQL_DSN`, `TEST_MONGO_URI`, `TEST_DYNAMODB_ENDPOINT`)
- Skips cleanly with `t.Skip(...)` if the env var is not set
- Creates test data with a unique `tenantID` per run to prevent test interference
- Cleans up its data in a `t.Cleanup()` callback that runs even on test failure
- Does not depend on data created by other tests (each test is self-contained)

---

## Critical E2E Test Scenarios

E2E tests use Playwright and run against a fully deployed local stack (`make dev-stack-up`).

### Authentication Flow

1. Unauthenticated user visits `/` → redirected to `/login`
2. User enters invalid credentials → error message shown
3. User enters valid credentials → redirected to dashboard
4. User clicks logout → redirected to `/login`, token cleared
5. User with expired token visits a protected page → redirected to `/login`

### Case Lifecycle

1. Authenticated agent creates a new case with title, priority, description
2. Case appears in the case list
3. Agent opens the case, adds a comment
4. Admin assigns the case to an agent
5. Agent transitions case status from `open` → `in_progress` → `resolved`
6. Case shows in resolved filter, not in open filter
7. Admin closes the case
8. Closed case cannot be re-opened by a viewer

### Document Upload

1. Agent opens a case
2. Agent uploads a PDF document (< 10MB)
3. Document appears in the case's document list
4. Agent clicks download — file downloads correctly
5. Agent uploads an updated version
6. Version history shows both versions
7. Attempt to upload a file > 50MB → error message shown

### Tenant Creation (Admin Flow)

1. Super admin logs in to admin portal
2. Super admin creates a new tenant (name, domain, plan)
3. Super admin creates an admin user for the new tenant
4. New tenant admin logs in at tenant URL
5. New tenant has empty case list (no data from other tenants)
6. New tenant admin creates a case — it exists only in their tenant's context

---

## Security Test Coverage

Security tests live in `tests/security/` and are run as part of the integration test suite.

### Authentication Tests

```
TestSecurity_NoAuthHeader_Returns401
TestSecurity_ExpiredJWT_Returns401
TestSecurity_TamperedJWT_Returns401
TestSecurity_MalformedJWT_Returns401
TestSecurity_WrongSigningKey_Returns401
```

### Authorization Tests

```
TestSecurity_ViewerCannotCreateCase
TestSecurity_AgentCannotDeleteTenant
TestSecurity_AdminCannotAccessOtherTenant
TestSecurity_SuperAdminCanAccessAllTenants
```

### Tenant Isolation Tests

```
TestSecurity_TenantA_CannotReadTenantB_Cases
TestSecurity_TenantA_CannotWriteToTenantB_Cases
TestSecurity_TenantA_CannotReadTenantB_Documents
TestSecurity_UserFromTenantA_WithTenantBHeader_IsRejected
```

### Input Validation Tests

```
TestSecurity_SQLInjection_InCaseTitleQueryParam
TestSecurity_SQLInjection_InFilterParams
TestSecurity_OversizedJSONPayload_Returns413
TestSecurity_OversizedFileUpload_Returns413
TestSecurity_XSSAttempt_InCaseTitle_IsSanitized
```

---

## Performance Baselines

Load tests run with k6 from `tests/load/`. Run against staging before any release affecting data access.

### Baseline Targets

| Scenario | Concurrency | p50 Target | p95 Target | p99 Target |
|---------|------------|-----------|-----------|-----------|
| POST /api/v1/auth/login | 50 | < 200ms | < 500ms | < 1000ms |
| GET /api/v1/cases (list) | 50 | < 100ms | < 300ms | < 600ms |
| GET /api/v1/cases/{id} | 100 | < 50ms | < 150ms | < 300ms |
| POST /api/v1/cases | 20 | < 150ms | < 400ms | < 800ms |
| PATCH /api/v1/cases/{id} | 20 | < 150ms | < 400ms | < 800ms |
| POST /api/v1/documents/{id}/upload (5MB) | 10 | < 1000ms | < 2000ms | < 4000ms |
| GET /api/v1/audit-logs | 20 | < 150ms | < 400ms | < 800ms |

### Running Load Tests

```bash
# Install k6: https://k6.io/docs/get-started/installation/

# Run a single scenario
k6 run tests/load/cases-list.js \
  --env BASE_URL=http://localhost:8082 \
  --env TENANT_ID=your-test-tenant-id \
  --env AUTH_TOKEN=your-test-token

# Run with output to console and file
k6 run tests/load/login.js --out json=results/login-$(date +%Y%m%d).json
```

Performance regressions (any p95 increases by > 50%) block deployment to production.

---

## Test Environment Setup

### Local Development

```bash
# 1. Start infrastructure
make dev-up

# 2. Set integration test env vars
export TEST_MYSQL_DSN="root:password@tcp(localhost:3306)/opsnexus_test?parseTime=true"
export TEST_MONGO_URI="mongodb://localhost:27017/opsnexus_test"
export TEST_DYNAMODB_ENDPOINT="http://localhost:4566"
export TEST_AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"

# 3. Run tests
make test-all
```

### CI Environment

GitHub Actions provides the infrastructure via service containers:

```yaml
services:
  mysql:
    image: mysql:8.0
    env:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: opsnexus_test
    options: >-
      --health-cmd="mysqladmin ping"
      --health-interval=10s
      --health-timeout=5s
      --health-retries=5

  mongodb:
    image: mongo:7.0
    options: >-
      --health-cmd="mongosh --eval 'db.adminCommand({ping: 1})'"
      --health-interval=10s
      --health-timeout=5s
      --health-retries=5

  localstack:
    image: localstack/localstack:latest
    env:
      SERVICES: dynamodb,s3
```

---

## CI Test Stages

Tests are gated — each stage must pass before the next starts:

| Stage | Trigger | Tests | Blocks Merge |
|-------|---------|-------|-------------|
| Lint | Every PR | golangci-lint, ESLint | Yes |
| Type Check | Every PR | npm run type-check | Yes |
| Unit Tests | Every PR | go test ./..., npm test -- --run | Yes |
| Build | Every PR | go build, npm run build, docker build | Yes |
| Integration Tests | Merge to main | go test -tags=integration, security tests | Yes (blocks deploy) |
| E2E Tests | Merge to main | Playwright critical paths | Yes (blocks deploy) |
| Load Tests | Release tag | k6 performance baselines | Yes (blocks production deploy) |

Total CI time target:
- PR checks (stages 1–4): < 5 minutes
- Full pipeline (stages 1–7): < 20 minutes
