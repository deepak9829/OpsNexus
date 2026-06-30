# Architecture Standards — OpsNexus

These standards are enforced on every PR. Violations block merge. When in doubt, read the Dependency Rule first; it resolves most ambiguity.

---

## 1. 3-Tier Enforcement

OpsNexus is a hard 3-tier system:

| Tier | Responsibility | Examples |
|------|---------------|---------|
| **Presentation** | Render UI, collect user input, call API | React apps in `apps/web/`, `apps/admin/` |
| **Application** | Business logic, orchestration, auth, validation | Go services in `services/` |
| **Data** | Persist and retrieve data | MySQL, MongoDB, DynamoDB, S3 |

**Rule:** Presentation never talks directly to Data. The browser never holds a MySQL connection string, never calls DynamoDB, never reads from S3 presigned URLs that bypass the application tier's auth checks.

All data access flows through the Application tier. Every read and write goes through a Go service. No exceptions.

If a feature seems to require direct DB access from the frontend, the correct fix is to add an endpoint to the relevant service, not to bypass the tier boundary.

---

## 2. Service Boundaries

Each service is an autonomous unit with its own schema ownership:

- **Auth Service** owns: `users`, `tenants`, `sessions`, `roles` tables in MySQL
- **Case Service** owns: `cases`, `case_comments`, `case_activities` tables in MySQL
- **Document Service** owns: `forms`, `form_submissions`, `documents`, `document_versions` collections in MongoDB
- **Workflow Service** owns: `workflows`, `workflow_steps`, `workflow_instances` tables in MySQL
- **Notification Service** owns: `notifications`, `notification_preferences` tables in DynamoDB

**Prohibited:**
- Cross-service DB joins (no `JOIN cases c ON c.user_id = users.id` running across two services)
- Direct table access into another service's schema
- Shared ORM models between services

**Correct approach:** If the Case Service needs a user's display name, it calls `GET /api/v1/users/{id}` on the Auth Service, caches if needed, and renders the result. The Case Service never queries the `users` table directly.

---

## 3. Clean/Hexagonal Architecture Pattern

Every Go service follows this internal directory layout:

```
services/{name}/
  internal/
    domain/       # Pure business entities. Zero external dependencies.
    ports/        # Interfaces for repos and services. No implementations.
    application/  # Business logic. Implements use cases. Uses ports only.
    adapters/     # Implementations: HTTP handlers, MySQL repos, MongoDB repos, DynamoDB repos.
  cmd/
    server/
      main.go     # Wire up adapters, inject into application, start server.
```

### domain/

Contains Go structs and business rules, nothing else.

```go
// domain/user.go — CORRECT
package domain

import (
    "errors"
    "time"
)

var (
    ErrUserNotFound    = errors.New("user not found")
    ErrEmailTaken      = errors.New("email already registered")
    ErrInvalidPassword = errors.New("invalid password")
)

type User struct {
    ID        string
    TenantID  string
    Email     string
    Role      Role
    CreatedAt time.Time
}

func (u *User) CanManageTenant() bool {
    return u.Role == RoleAdmin || u.Role == RoleSuperAdmin
}
```

Never import `gorm`, `mongo-driver`, `fiber`, `gin`, `aws-sdk-go`, or any infrastructure package in `domain/`.

### ports/

Pure Go interfaces. No struct implementations live here.

```go
// ports/user_repository.go
package ports

import (
    "context"
    "opsnexus/services/auth/internal/domain"
)

type UserRepository interface {
    FindByID(ctx context.Context, tenantID, userID string) (*domain.User, error)
    FindByEmail(ctx context.Context, tenantID, email string) (*domain.User, error)
    Save(ctx context.Context, user *domain.User) error
    Delete(ctx context.Context, tenantID, userID string) error
}

type PasswordHasher interface {
    Hash(password string) (string, error)
    Verify(hash, password string) error
}
```

Note: `tenantID` is always a parameter. No query ever lacks tenant scope.

### application/

Business logic only. Calls ports. Never calls adapters directly.

```go
// application/auth_service.go — CORRECT
package application

import (
    "context"
    "fmt"
    "opsnexus/services/auth/internal/domain"
    "opsnexus/services/auth/internal/ports"
)

type AuthService struct {
    users   ports.UserRepository
    hasher  ports.PasswordHasher
    tokens  ports.TokenIssuer
}

func NewAuthService(users ports.UserRepository, hasher ports.PasswordHasher, tokens ports.TokenIssuer) *AuthService {
    return &AuthService{users: users, hasher: hasher, tokens: tokens}
}

func (s *AuthService) Login(ctx context.Context, tenantID, email, password string) (string, error) {
    user, err := s.users.FindByEmail(ctx, tenantID, email)
    if err != nil {
        return "", fmt.Errorf("login: finding user: %w", err)
    }
    if err := s.hasher.Verify(user.PasswordHash, password); err != nil {
        return "", domain.ErrInvalidPassword
    }
    token, err := s.tokens.Issue(ctx, user)
    if err != nil {
        return "", fmt.Errorf("login: issuing token: %w", err)
    }
    return token, nil
}
```

Never import `gorm`, `fiber`, `gin`, `net/http`, or any adapter package in `application/`.

### adapters/

Concrete implementations. This is the only place framework dependencies live.

```
adapters/
  http/          # Fiber/Gin handlers, middleware, routing
  mysql/         # GORM-backed repository implementations
  mongodb/       # mongo-driver repository implementations
  dynamodb/      # AWS SDK DynamoDB implementations
```

---

## 4. Dependency Rule

Dependencies flow inward only:

```
adapters → application → ports ← domain
```

- `adapters` depends on `application` and `ports`
- `application` depends on `ports` and `domain`
- `domain` has zero external dependencies

**Never reverse this.** `domain` must not import `application`. `application` must not import `adapters`. If you find yourself needing to import an outer layer from an inner layer, the fix is to introduce a new port interface, not to break the rule.

A fast check: run `grep -r "gorm\|fiber\|gin\|mongo-driver\|aws-sdk" internal/domain/ internal/application/` in any service. It must return empty.

---

## 5. Error Handling

Domain errors are typed sentinel errors defined in `domain/`:

```go
var (
    ErrNotFound        = errors.New("not found")
    ErrPermissionDenied = errors.New("permission denied")
    ErrConflict        = errors.New("conflict")
    ErrValidation      = errors.New("validation failed")
)
```

Application layer wraps and propagates:

```go
return fmt.Errorf("creating case: %w", err)
```

HTTP adapter is the only place domain errors map to HTTP status codes:

```go
// adapters/http/handlers/case_handler.go
func mapError(err error) (int, ErrorResponse) {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        return http.StatusNotFound, ErrorResponse{Code: "CASE_NOT_FOUND", Message: "case not found"}
    case errors.Is(err, domain.ErrPermissionDenied):
        return http.StatusForbidden, ErrorResponse{Code: "CASE_ACCESS_DENIED", Message: "access denied"}
    case errors.Is(err, domain.ErrValidation):
        return http.StatusBadRequest, ErrorResponse{Code: "CASE_VALIDATION_FAILED", Message: err.Error()}
    default:
        // Log the full error internally; never expose internals
        logger.Error("unhandled error", zap.Error(err))
        return http.StatusInternalServerError, ErrorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"}
    }
}
```

Stack traces and internal error strings never reach the API response body.

---

## 6. Tenant Isolation

Multi-tenancy is non-negotiable. Every piece of data in OpsNexus belongs to a tenant.

**Rules:**
- Every MySQL table that holds tenant data includes a `tenant_id VARCHAR(36) NOT NULL` column with an index
- Every MongoDB document includes a `tenantId` field
- Every DynamoDB item includes `tenantId` as part of the partition key or a required attribute
- Every repository method receives `tenantID` as a parameter
- No query runs without a tenant filter

**Code pattern:**
```go
// CORRECT
func (r *MySQLCaseRepo) FindByID(ctx context.Context, tenantID, caseID string) (*domain.Case, error) {
    var c Case
    err := r.db.WithContext(ctx).
        Where("tenant_id = ? AND id = ?", tenantID, caseID).
        First(&c).Error
    // ...
}

// WRONG — missing tenant filter
func (r *MySQLCaseRepo) FindByID(ctx context.Context, caseID string) (*domain.Case, error) {
    var c Case
    err := r.db.WithContext(ctx).Where("id = ?", caseID).First(&c).Error
    // ...
}
```

Test this with multi-tenant integration tests: create two tenants A and B, create a case under A, then verify that querying with tenant B's ID returns not-found, not A's case.

---

## 7. API Design

**URL structure:**
```
/api/v1/{resource}                    # collection
/api/v1/{resource}/{id}               # single resource
/api/v1/{resource}/{id}/{sub-resource} # sub-collection
```

- kebab-case for URL segments: `/api/v1/case-comments`, not `/api/v1/caseComments`
- camelCase for JSON field names: `{"createdAt": "..."}`, not `{"created_at": "..."}`
- Plural nouns for collections: `/api/v1/cases`, not `/api/v1/case`

**Response envelope (success):**
```json
{
  "data": { ... },
  "meta": {
    "requestId": "uuid",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

**Response envelope (collection with pagination):**
```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 145,
    "totalPages": 8,
    "requestId": "uuid"
  }
}
```

**Response envelope (error):**
```json
{
  "error": {
    "code": "AUTH_TOKEN_EXPIRED",
    "message": "your session has expired, please log in again"
  }
}
```

**OpenAPI-first:** The OpenAPI spec in `contracts/{service}/openapi.yaml` is the source of truth. Implement to match the spec. If the spec and implementation disagree, the spec wins and the implementation is fixed.

---

## 8. Service Communication

At this stage, services do not call each other directly within the backend. Inter-service calls route through the API gateway pattern (a single entry point that proxies to services).

When a service-to-service call is needed:
- Use HTTP with standard request headers
- Always propagate `X-Tenant-ID`, `X-User-ID`, and `X-Request-ID` headers
- Use the shared internal HTTP client from `pkg/httpclient/` which adds these headers automatically
- Treat the called service as an external dependency: handle errors, timeouts (default 5s), and retries (max 3, exponential backoff)

No gRPC, no message queues between services yet. That is a future concern.

---

## 9. Future AWS Deployment Mapping

This table guides infrastructure decisions and keeps local development choices reversible:

| Component | Local Dev | AWS Target |
|-----------|-----------|-----------|
| React apps | Vite dev server | S3 + CloudFront |
| Go services | Local binary / Docker Compose | EKS pods (one Deployment per service) |
| MySQL | Docker Compose MySQL 8 | RDS MySQL 8 (Multi-AZ in prod) |
| MongoDB | Docker Compose MongoDB 7 | Amazon DocumentDB |
| DynamoDB | LocalStack | AWS DynamoDB |
| File storage | LocalStack S3 | AWS S3 |
| Notifications | In-process mock | SNS + SQS |
| Secrets | `.env` files | AWS Secrets Manager |
| Container registry | Local Docker | Amazon ECR |
| Load balancing | — | AWS ALB |
| Service mesh | — | AWS App Mesh (future) |

Design decisions that conflict with this mapping must be explicitly justified and documented.
