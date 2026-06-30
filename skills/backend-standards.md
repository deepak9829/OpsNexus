# Backend Standards — OpsNexus Go Services

These are the working standards for every Go service in `services/`. Read this before writing your first line of Go in this repo.

---

## 1. Project Layout

Every service follows this layout. Do not invent new top-level directories without a discussion.

```
services/{name}/
  cmd/
    server/
      main.go          # Entry point: load config, wire deps, start server
  internal/
    domain/            # Pure business types and sentinel errors
    ports/             # Repository and service interfaces
    application/       # Business logic (use cases)
    adapters/
      http/            # HTTP handlers, middleware, routes
        handlers/
        middleware/
        router.go
      mysql/           # GORM repository implementations
      mongodb/         # mongo-driver implementations
      dynamodb/        # AWS SDK implementations
  pkg/                 # Exported packages safe to import from other services
  config/
    config.go          # Config struct populated from env vars via Viper
  go.mod
  go.sum
  Makefile
  Dockerfile
```

`internal/` is enforced by the Go compiler — nothing outside this service can import from it. `pkg/` is for utilities explicitly intended to be shared (e.g., a shared JWT parser package).

---

## 2. Error Handling

### Wrap errors with context

Every error returned from a function must be wrapped with the operation context so the call chain is readable in logs.

```go
// CORRECT: wraps with context
func (s *CaseService) CreateCase(ctx context.Context, req CreateCaseRequest) (*domain.Case, error) {
    user, err := s.users.FindByID(ctx, req.TenantID, req.UserID)
    if err != nil {
        return nil, fmt.Errorf("creating case: finding user %s: %w", req.UserID, err)
    }
    c := domain.NewCase(req.TenantID, req.Title, user.ID)
    if err := s.cases.Save(ctx, c); err != nil {
        return nil, fmt.Errorf("creating case: saving: %w", err)
    }
    return c, nil
}

// WRONG: loses context
if err != nil {
    return nil, err
}

// WRONG: swallows error
user, _ := s.users.FindByID(ctx, tenantID, userID)
```

### Check domain errors with errors.Is

```go
// CORRECT
if errors.Is(err, domain.ErrUserNotFound) {
    return http.StatusNotFound, "user not found"
}

// WRONG: string comparison is fragile
if err.Error() == "user not found" { ... }
```

### Never panic in business logic

`panic` is only acceptable in `main.go` during startup (e.g., failed to parse config, failed to connect to DB). Everywhere else, return an error.

---

## 3. Context Propagation

Context is the first argument in every function that does I/O or calls another service. No exceptions.

```go
// CORRECT
func (r *MySQLUserRepo) FindByID(ctx context.Context, tenantID, userID string) (*domain.User, error)

// WRONG: no context
func (r *MySQLUserRepo) FindByID(tenantID, userID string) (*domain.User, error)
```

Never use `context.Background()` in business logic or handlers. `context.Background()` is only valid in:
- `main.go` during startup
- Test setup

In handlers, extract the request context:
```go
func (h *CaseHandler) Create(c *fiber.Ctx) error {
    ctx := c.UserContext() // fiber's request context
    result, err := h.service.CreateCase(ctx, req)
    // ...
}
```

Pass timeouts via context when calling external services:
```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
result, err := externalClient.Call(ctx, params)
```

---

## 4. Logging

Use `zap` (uber-go/zap). Never use `fmt.Printf`, `log.Println`, or `fmt.Println` in production code paths.

```go
// CORRECT: structured logging
logger.Info("case created",
    zap.String("tenantID", tenantID),
    zap.String("caseID", c.ID),
    zap.String("userID", userID),
)

logger.Error("failed to create case",
    zap.Error(err),
    zap.String("tenantID", tenantID),
    zap.String("requestID", requestID),
)

// WRONG: unstructured
fmt.Printf("case created: %s\n", caseID)
log.Println("error:", err)
```

Log levels:
- `Debug`: verbose detail for development (disabled in production)
- `Info`: normal lifecycle events (service started, request processed, job completed)
- `Warn`: recoverable unexpected states (retry triggered, slow query, deprecation hit)
- `Error`: failures that impacted the caller (request failed, data not saved)
- `Fatal`: only in `main.go`; kills the process

Never log passwords, tokens, PII (email addresses, phone numbers, names), or internal stack traces in Info/Warn. Log them at Debug only when explicitly enabled for troubleshooting.

---

## 5. Configuration

All configuration comes from environment variables. No hardcoded values in code. No config files committed to the repo (use `.env.example` as documentation).

```go
// config/config.go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
}

type ServerConfig struct {
    Port         int    `mapstructure:"SERVER_PORT"`
    ReadTimeout  int    `mapstructure:"SERVER_READ_TIMEOUT_SECONDS"`
    WriteTimeout int    `mapstructure:"SERVER_WRITE_TIMEOUT_SECONDS"`
}

type DatabaseConfig struct {
    MySQLDSN string `mapstructure:"MYSQL_DSN"`
}

type JWTConfig struct {
    Secret        string `mapstructure:"JWT_SECRET"`
    ExpirySeconds int    `mapstructure:"JWT_EXPIRY_SECONDS"`
}

func Load() (*Config, error) {
    viper.AutomaticEnv()
    viper.SetDefault("SERVER_PORT", 8080)
    viper.SetDefault("SERVER_READ_TIMEOUT_SECONDS", 30)
    viper.SetDefault("SERVER_WRITE_TIMEOUT_SECONDS", 30)
    viper.SetDefault("JWT_EXPIRY_SECONDS", 86400)

    cfg := &Config{}
    if err := viper.Unmarshal(cfg); err != nil {
        return nil, fmt.Errorf("loading config: %w", err)
    }
    return cfg, nil
}
```

Required variables (those without defaults) must be validated at startup:
```go
cfg, err := config.Load()
if err != nil || cfg.JWT.Secret == "" {
    log.Fatal("invalid config: JWT_SECRET is required")
}
```

---

## 6. HTTP Handlers

One handler method per endpoint. Handlers are thin: validate input, call application service, map result, return response.

```go
// adapters/http/handlers/case_handler.go
type CaseHandler struct {
    service ports.CaseService
    logger  *zap.Logger
}

func (h *CaseHandler) Create(c *fiber.Ctx) error {
    // 1. Extract and validate input
    var req CreateCaseRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(http.StatusBadRequest).JSON(errorResponse("CASE_INVALID_BODY", "invalid request body"))
    }
    if err := validate(req); err != nil {
        return c.Status(http.StatusBadRequest).JSON(errorResponse("CASE_VALIDATION_FAILED", err.Error()))
    }

    // 2. Extract tenant/user from context (set by auth middleware)
    tenantID := c.Locals("tenantID").(string)
    userID := c.Locals("userID").(string)

    // 3. Call application layer
    ctx := c.UserContext()
    cas, err := h.service.CreateCase(ctx, application.CreateCaseInput{
        TenantID: tenantID,
        UserID:   userID,
        Title:    req.Title,
        Priority: req.Priority,
    })
    if err != nil {
        return mapDomainError(c, err, h.logger)
    }

    // 4. Return response
    return c.Status(http.StatusCreated).JSON(successResponse(toCaseDTO(cas)))
}
```

Never put business logic in handlers. If a handler is longer than ~60 lines, extract logic into the application layer.

---

## 7. Repository Pattern

Repositories handle data access only. No business logic, no validation, no email sending.

```go
// adapters/mysql/user_repository.go — CORRECT
func (r *MySQLUserRepo) FindByEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
    var model UserModel
    err := r.db.WithContext(ctx).
        Where("tenant_id = ? AND email = ?", tenantID, email).
        First(&model).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, domain.ErrUserNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("finding user by email: %w", err)
    }
    return toDomainUser(model), nil
}
```

Key rules:
- Map `gorm.ErrRecordNotFound` to `domain.ErrNotFound` (never let GORM errors escape the adapter)
- Always pass and use `ctx`
- Always include `tenant_id` in WHERE clauses
- Return domain types, not ORM model types
- Keep mapper functions (`toDomainUser`, `toUserModel`) private to the adapter package

---

## 8. Testing Standards

### Unit tests (application layer)

Use simple mock structs implementing port interfaces. Do not use mockery-generated mocks unless the interface is very large.

```go
// internal/application/auth_service_test.go
type mockUserRepo struct {
    users map[string]*domain.User
    err   error
}

func (m *mockUserRepo) FindByEmail(_ context.Context, _, email string) (*domain.User, error) {
    if m.err != nil {
        return nil, m.err
    }
    u, ok := m.users[email]
    if !ok {
        return nil, domain.ErrUserNotFound
    }
    return u, nil
}

func TestAuthService_Login_ValidCredentials_ReturnsToken(t *testing.T) {
    repo := &mockUserRepo{
        users: map[string]*domain.User{
            "test@example.com": {ID: "u1", Email: "test@example.com", PasswordHash: "$bcrypt$..."},
        },
    }
    service := NewAuthService(repo, realHasher, realTokenIssuer)
    token, err := service.Login(context.Background(), "tenant1", "test@example.com", "correct-password")
    require.NoError(t, err)
    assert.NotEmpty(t, token)
}
```

### Integration tests (adapter layer)

```go
//go:build integration

package mysql_test

func TestMySQLUserRepo_FindByEmail_ExistingUser_Returns(t *testing.T) {
    db := testDB(t) // connects to TEST_MYSQL_DSN, runs migrations, registers cleanup
    repo := mysql.NewUserRepository(db)

    // Arrange
    testUser := insertTestUser(t, db, "tenant-a", "found@example.com")

    // Act
    user, err := repo.FindByEmail(context.Background(), "tenant-a", "found@example.com")

    // Assert
    require.NoError(t, err)
    assert.Equal(t, testUser.ID, user.ID)
}
```

Run unit tests with: `go test ./...`
Run integration tests with: `go test -tags=integration ./...`

### Table-driven tests for multiple scenarios

```go
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid password", "Str0ng!Pass", false},
        {"too short", "abc", true},
        {"no uppercase", "weakpass1!", true},
        {"no number", "WeakPasswd!", true},
        {"empty", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePassword(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## 9. Naming Conventions

| Thing | Convention | Example |
|-------|-----------|---------|
| Interface | No `I` prefix | `UserRepository` not `IUserRepository` |
| Constructor | `New` prefix | `NewAuthService`, `NewUserHandler` |
| Implementation | Unexported if only one | `mysqlUserRepo` |
| Error sentinels | `Err` prefix | `ErrUserNotFound`, `ErrPermissionDenied` |
| Test names | `TestUnit_Scenario_Outcome` | `TestAuthService_Login_InvalidPassword_ReturnsError` |
| Config struct fields | Match env var name | `MySQLDSN string \`mapstructure:"MYSQL_DSN"\`` |

---

## 10. Go Module Per Service

Each service has its own `go.mod`. Services are independently deployable and independently versioned.

For local development across services, use a `go.work` file at the repo root:

```
go 1.22

use (
    ./services/auth
    ./services/cases
    ./services/documents
    ./services/workflows
    ./services/notifications
    ./pkg/shared
)
```

Never add a replace directive that points outside the monorepo. Dependencies must be real versioned modules.

---

## 11. Graceful Shutdown

Every service must handle SIGTERM and SIGINT cleanly. The pattern:

```go
// cmd/server/main.go
func main() {
    cfg, err := config.Load()
    // ... setup ...

    app := fiber.New()
    // ... routes ...

    // Start server in goroutine
    go func() {
        if err := app.Listen(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
            logger.Fatal("server error", zap.Error(err))
        }
    }()

    // Block until signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit

    logger.Info("shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := app.ShutdownWithContext(ctx); err != nil {
        logger.Error("shutdown error", zap.Error(err))
    }

    // Close DB connections
    db.Close()
    logger.Info("shutdown complete")
}
```

Kubernetes sends SIGTERM before killing the pod. 30 seconds is the standard grace period.

---

## 12. Health Check

Every service exposes `GET /health` with no authentication required.

```go
// Response when healthy
// HTTP 200
{
  "status": "ok",
  "service": "auth-service",
  "version": "1.2.3"
}

// Response when degraded (DB down)
// HTTP 503
{
  "status": "degraded",
  "service": "auth-service",
  "checks": {
    "database": "connection refused"
  }
}
```

The health handler checks:
1. DB connectivity (ping)
2. Any critical external dependencies

The Dockerfile HEALTHCHECK and Kubernetes readiness probe both point to `/health`.
