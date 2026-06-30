# Testing Standards — OpsNexus

Testing is part of the Definition of Done. No feature is complete without tests. This document is the reference for how we test, what we test, and what "good" looks like.

---

## 1. Test Pyramid

| Layer | Target % | Scope | Speed |
|-------|---------|-------|-------|
| Unit | 70% | Application layer logic, React components, pure functions | < 1ms each |
| Integration | 20% | DB adapters, HTTP handlers with real infrastructure | < 500ms each |
| E2E | 10% | Critical user flows end-to-end in a browser | < 30s each |

Do not invert this pyramid. Expensive E2E tests should cover only the most critical flows. Unit tests should cover the business logic exhaustively.

---

## 2. Go Unit Test Rules

### Mock interfaces with simple structs

Do not use generated mocks unless the interface has more than 8 methods. Simple struct implementations are readable and require no extra tooling.

```go
// CORRECT: simple struct mock
type mockCaseRepo struct {
    cases   map[string]*domain.Case
    saveErr error
    findErr error
}

func (m *mockCaseRepo) FindByID(_ context.Context, tenantID, id string) (*domain.Case, error) {
    if m.findErr != nil {
        return nil, m.findErr
    }
    key := tenantID + ":" + id
    c, ok := m.cases[key]
    if !ok {
        return nil, domain.ErrNotFound
    }
    return c, nil
}

func (m *mockCaseRepo) Save(_ context.Context, c *domain.Case) error {
    if m.saveErr != nil {
        return m.saveErr
    }
    key := c.TenantID + ":" + c.ID
    m.cases[key] = c
    return nil
}
```

### Test package conventions

- Use `package foo_test` (black-box testing) for application and port packages — test the exported API only
- Use `package foo` (white-box testing) when you need to test unexported functions — should be rare

### Assertions

Use `testify`:
- `require.NoError(t, err)` — stops the test immediately if the assertion fails (fatal)
- `assert.Equal(t, expected, actual)` — records failure but continues (non-fatal)

Use `require` when subsequent lines in the test depend on the assertion being true (e.g., after asserting no error, then accessing the returned value). Use `assert` when you want to check multiple independent conditions.

```go
func TestCaseService_CreateCase_ValidInput_ReturnsCase(t *testing.T) {
    repo := &mockCaseRepo{cases: map[string]*domain.Case{}}
    service := NewCaseService(repo, mockNotifier{})

    result, err := service.CreateCase(context.Background(), application.CreateCaseInput{
        TenantID: "tenant-1",
        UserID:   "user-1",
        Title:    "Server is down",
        Priority: "critical",
    })

    require.NoError(t, err)        // fatal — if err != nil, the next line panics
    assert.NotEmpty(t, result.ID)  // non-fatal — continue checking
    assert.Equal(t, "Server is down", result.Title)
    assert.Equal(t, "tenant-1", result.TenantID)
    assert.Equal(t, domain.PriorityCritical, result.Priority)
}
```

### Table-driven tests

Use table-driven tests whenever a function has multiple distinct input/outcome combinations:

```go
func TestValidateCreateCaseInput(t *testing.T) {
    tests := []struct {
        name    string
        input   application.CreateCaseInput
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid input",
            input:   CreateCaseInput{TenantID: "t1", UserID: "u1", Title: "Valid Title", Priority: "high"},
            wantErr: false,
        },
        {
            name:    "empty title",
            input:   CreateCaseInput{TenantID: "t1", UserID: "u1", Title: "", Priority: "high"},
            wantErr: true,
            errMsg:  "title is required",
        },
        {
            name:    "title too long",
            input:   CreateCaseInput{TenantID: "t1", UserID: "u1", Title: strings.Repeat("x", 201), Priority: "high"},
            wantErr: true,
            errMsg:  "title must be at most 200 characters",
        },
        {
            name:    "invalid priority",
            input:   CreateCaseInput{TenantID: "t1", UserID: "u1", Title: "Title", Priority: "urgent"},
            wantErr: true,
            errMsg:  "invalid priority",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateCreateCaseInput(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Test naming convention

`TestUnitOfWork_Scenario_ExpectedOutcome`

Examples:
- `TestAuthService_Login_ValidCredentials_ReturnsToken`
- `TestAuthService_Login_WrongPassword_ReturnsInvalidPasswordError`
- `TestAuthService_Login_UserNotFound_ReturnsNotFoundError`
- `TestCaseRepo_FindByID_WrongTenant_ReturnsNotFound`

---

## 3. Integration Test Rules

### Build tag

Every integration test file must start with the build tag:

```go
//go:build integration

package mysql_test
```

This ensures `go test ./...` (no tags) never runs integration tests unintentionally. Integration tests require Docker Compose to be running.

### Infrastructure

Integration tests connect to real infrastructure started by `make dev-up`:
- MySQL: connection string from `TEST_MYSQL_DSN` env var
- MongoDB: connection string from `TEST_MONGO_URI` env var
- DynamoDB (LocalStack): endpoint from `TEST_DYNAMODB_ENDPOINT` env var

```go
func testDB(t *testing.T) *gorm.DB {
    t.Helper()
    dsn := os.Getenv("TEST_MYSQL_DSN")
    if dsn == "" {
        t.Skip("TEST_MYSQL_DSN not set — skipping integration test")
    }
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    require.NoError(t, err)
    runMigrations(t, db)
    t.Cleanup(func() { cleanupTestData(t, db) })
    return db
}
```

### Cleanup

Each integration test cleans up its data. Use `t.Cleanup()` to guarantee cleanup even if the test fails. Use unique tenant IDs or row identifiers per test run to prevent interference between parallel tests.

```go
func TestMySQLCaseRepo_Save_NewCase_Persists(t *testing.T) {
    db := testDB(t)
    repo := mysql.NewCaseRepository(db)
    tenantID := "test-tenant-" + uuid.New().String() // unique per run

    c := &domain.Case{
        ID:       uuid.New().String(),
        TenantID: tenantID,
        Title:    "integration test case",
    }

    t.Cleanup(func() {
        db.Where("tenant_id = ?", tenantID).Delete(&mysql.CaseModel{})
    })

    err := repo.Save(context.Background(), c)
    require.NoError(t, err)

    // Verify directly in DB
    var saved mysql.CaseModel
    err = db.Where("id = ? AND tenant_id = ?", c.ID, tenantID).First(&saved).Error
    require.NoError(t, err)
    assert.Equal(t, c.Title, saved.Title)
}
```

### Multi-tenant isolation integration tests

For every repository, there must be an integration test that verifies cross-tenant isolation:

```go
func TestMySQLCaseRepo_FindByID_WrongTenant_ReturnsNotFound(t *testing.T) {
    db := testDB(t)
    repo := mysql.NewCaseRepository(db)
    tenantA := "tenant-a-" + uuid.New().String()
    tenantB := "tenant-b-" + uuid.New().String()

    // Create case under tenant A
    caseID := insertTestCase(t, db, tenantA, "Tenant A's case")
    t.Cleanup(func() {
        db.Where("tenant_id IN ?", []string{tenantA, tenantB}).Delete(&mysql.CaseModel{})
    })

    // Try to access it as tenant B — must not see it
    _, err := repo.FindByID(context.Background(), tenantB, caseID)
    assert.ErrorIs(t, err, domain.ErrNotFound)
}
```

---

## 4. React Test Rules

### Test from the user's perspective

```tsx
// CORRECT: tests what the user sees and does
test('shows error when form submitted without title', async () => {
  render(<CreateCaseForm onSuccess={vi.fn()} />);

  await userEvent.click(screen.getByRole('button', { name: 'Create Case' }));

  expect(screen.getByRole('alert')).toHaveTextContent('Title must be at least 3 characters');
});

// WRONG: tests implementation details
test('sets titleError state when title is empty', () => {
  const { result } = renderHook(() => useCreateCaseForm());
  act(() => result.current.setTitle(''));
  expect(result.current.titleError).toBe('Title must be at least 3 characters');
});
```

### API mocking with MSW

Use Mock Service Worker to intercept HTTP requests in tests. Do not mock `axios` or `fetch` directly.

```ts
// test/mocks/handlers.ts
import { http, HttpResponse } from 'msw';

export const handlers = [
  http.get('/api/v1/cases', () => {
    return HttpResponse.json({
      data: [{ id: 'case-1', title: 'Test Case', priority: 'high' }],
      meta: { total: 1, page: 1, limit: 20, totalPages: 1 },
    });
  }),
  http.post('/api/v1/cases', () => {
    return HttpResponse.json({ data: { id: 'case-new', title: 'New Case' } }, { status: 201 });
  }),
];
```

### Render with all providers

```tsx
// test/utils.tsx
export function renderWithProviders(ui: ReactElement, options?: RenderOptions) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },    // Don't retry on failure in tests
      mutations: { retry: false },
    },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <AuthProvider initialUser={mockUser}>
            {children}
          </AuthProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );
  }

  return render(ui, { wrapper: Wrapper, ...options });
}
```

### Test all three states

Every component that fetches data must have tests for:
1. Loading state: skeleton or spinner is shown
2. Error state: error message is shown, action to retry is available
3. Success state: data is rendered correctly

```tsx
test('shows loading skeleton while cases are fetching', () => {
  server.use(http.get('/api/v1/cases', () => new Promise(() => {}))); // never resolves
  renderWithProviders(<CasesList />);
  expect(screen.getByTestId('cases-skeleton')).toBeInTheDocument();
});

test('shows error message when cases fetch fails', async () => {
  server.use(http.get('/api/v1/cases', () => HttpResponse.error()));
  renderWithProviders(<CasesList />);
  expect(await screen.findByRole('alert')).toHaveTextContent('Failed to load cases');
});

test('renders cases when fetch succeeds', async () => {
  renderWithProviders(<CasesList />);
  expect(await screen.findByText('Test Case')).toBeInTheDocument();
});
```

---

## 5. Security Test Cases

The following checklist applies to every new service endpoint. These tests must exist before a feature is considered done.

- [ ] **Unauthenticated request returns 401** — request with no Authorization header
- [ ] **Invalid token returns 401** — request with a malformed JWT
- [ ] **Expired token returns 401** — request with JWT where `exp` is in the past
- [ ] **Tampered signature returns 401** — JWT with valid structure but invalid signature
- [ ] **Wrong tenant isolation returns 403 or empty** — authenticated user from tenant B cannot see tenant A's data; must return 403 (for direct ID access) or empty collection (for list), never tenant A's data
- [ ] **Admin endpoint rejects non-admin token** — authenticated user with `role: user` cannot call admin endpoints; returns 403
- [ ] **SQL/NoSQL injection attempt** — query params containing `' OR 1=1 --` or similar must not affect results beyond a validation error
- [ ] **Oversized JSON payload rejected** — payload > 10MB returns 413
- [ ] **Oversized file upload rejected** — file > 50MB (or configured limit) returns 413

---

## 6. Performance Test Baselines

Using k6 for load tests (located in `tests/load/`):

| Endpoint | Condition | p95 Target |
|---------|-----------|-----------|
| POST /api/v1/auth/login | 50 concurrent users | < 500ms |
| GET /api/v1/cases | 50 concurrent users | < 300ms |
| GET /api/v1/cases/{id} | 50 concurrent users | < 150ms |
| POST /api/v1/cases | 20 concurrent users | < 400ms |
| POST /api/v1/documents/{id}/upload (5MB file) | 10 concurrent users | < 2000ms |
| GET /api/v1/audit-logs | 20 concurrent users | < 400ms |

These are baselines for staging. Production targets may be tighter. Run load tests before any release that changes data access patterns.

---

## 7. Test Naming Convention

All test names follow the pattern: `TestUnitOfWork_Scenario_ExpectedOutcome`

For React tests, use plain English descriptions that read like requirements:
- `shows loading skeleton while cases are fetching`
- `displays error message when case creation fails`
- `redirects to case detail after successful creation`
- `disables submit button while form is submitting`

---

## 8. Test Environment Setup

```bash
# Start infrastructure for integration tests
make dev-up

# Verify infrastructure is ready
docker compose ps

# Set env vars for integration tests
export TEST_MYSQL_DSN="root:password@tcp(localhost:3306)/opsnexus_test?parseTime=true"
export TEST_MONGO_URI="mongodb://localhost:27017/opsnexus_test"
export TEST_DYNAMODB_ENDPOINT="http://localhost:4566"

# Run unit tests only
go test ./...

# Run integration tests (requires infrastructure)
go test -tags=integration ./...

# Run frontend unit tests
npm test -- --run

# Run Playwright E2E tests (requires all services running)
npx playwright test
```

---

## 9. CI Test Stages

Tests run in this order on every PR:

1. **Lint** — `golangci-lint run` and `npm run lint` (fast, no infrastructure)
2. **Type check** — `npm run type-check` (fast, no infrastructure)
3. **Unit tests** — `go test ./...` and `npm test -- --run` (fast, no infrastructure)
4. **Build** — `go build ./...` and `npm run build` (moderate, no infrastructure)
5. **Integration tests** — `go test -tags=integration ./...` (requires Docker services in CI)
6. **E2E tests** — Playwright against the full stack (slowest, runs on merge to main only)

PRs that fail stages 1–4 are blocked from merge. Stages 5–6 failures are tracked but handled separately for speed.
