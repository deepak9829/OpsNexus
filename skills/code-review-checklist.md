# Code Review Checklist — OpsNexus

Use this checklist on every PR review. Not every item applies to every PR, but consider each one. Items marked as blocking must be resolved before merge. Items marked as non-blocking are suggestions that should be addressed but don't block merge.

---

## Architecture & Design

- [ ] **[BLOCKING]** Does the change respect clean architecture layer boundaries? (domain has no framework deps, application has no adapter deps, adapters are the only place GORM/Fiber/Mongo imports live)
- [ ] **[BLOCKING]** Are new interfaces defined in `ports/`, not leaked from adapters? Adapter types must not appear in application or domain layer signatures.
- [ ] **[BLOCKING]** Does any new MySQL/MongoDB table/collection include `tenant_id`? If the data belongs to a tenant, the field is mandatory.
- [ ] **[BLOCKING]** Are there any direct DB calls from the application layer? Application must call repository interfaces, never GORM or mongo-driver directly.
- [ ] Does any new service-to-service communication follow the established pattern (HTTP with propagated headers, via gateway)?
- [ ] If a new service is added, does it have its own `go.mod` and follow the standard directory layout?

---

## Security

- [ ] **[BLOCKING]** Are all new HTTP endpoints registered behind the auth middleware?
- [ ] **[BLOCKING]** Is the tenant ID extracted from the authenticated JWT/context (not from user-supplied request body) and applied to every data access?
- [ ] **[BLOCKING]** Is user input validated before it's used in any query, command, or business logic?
- [ ] **[BLOCKING]** Are passwords, tokens, API keys, or secrets ever logged, returned in a response, or stored in plaintext?
- [ ] Does the response payload expose more fields than the caller needs? (Principle of least exposure)
- [ ] Are admin-only endpoints guarded by a role check, not just an auth check?
- [ ] Does any new file upload path enforce file type and size limits?

---

## Error Handling

- [ ] **[BLOCKING]** Are errors wrapped with context at each layer? (`fmt.Errorf("doing X: %w", err)`)
- [ ] **[BLOCKING]** Are domain sentinel errors checked with `errors.Is()`, not string matching?
- [ ] **[BLOCKING]** Does the HTTP handler map domain errors to the appropriate HTTP status codes? (`ErrNotFound` → 404, `ErrPermissionDenied` → 403, etc.)
- [ ] Are 500-level errors logged with full detail server-side while only returning a generic message to the client?
- [ ] Are there any swallowed errors (`_, err :=` or missing error check)?

---

## Tenant Isolation

- [ ] **[BLOCKING]** Does every new repository method accept `tenantID` as a parameter?
- [ ] **[BLOCKING]** Does every DB query include a `tenant_id` filter?
- [ ] Is there an integration test that verifies a tenant cannot access another tenant's data through this code path?

---

## Testing

- [ ] **[BLOCKING]** Is there at least one unit test for every new business logic function in the application layer?
- [ ] Are both the happy path and at least one error/edge case path tested?
- [ ] Do test names follow the `TestUnit_Scenario_ExpectedOutcome` pattern?
- [ ] If a new DB repository method was added, is there an integration test for it?
- [ ] Do new React components have tests for loading, error, and success states?
- [ ] Is the test coverage roughly maintained (new code has tests, no significant drops)?

---

## Code Quality

- [ ] Are exported functions, types, and methods documented with Go doc comments if their behavior isn't self-evident?
- [ ] Is there any commented-out code? (Remove it; git history preserves it if needed later)
- [ ] Are there `// TODO:` comments that represent tracked work? If so, is there a corresponding Jira/GitHub issue reference? `// TODO: fix this later` without a ticket is not acceptable.
- [ ] Is there code duplication that should be extracted into a shared function or package?
- [ ] Does the change introduce any new dependencies? If so, are they justified, well-maintained, and license-compatible?
- [ ] Is the change overly complex for the problem being solved? (Simplest correct solution is preferred)

---

## Go-Specific

- [ ] Is `context` passed as the first argument to all functions that do I/O?
- [ ] Is `context.Background()` used anywhere outside of `main.go` or test setup? (Flag this — it should be the request context)
- [ ] Are goroutines properly bounded? (No goroutine leaks — every goroutine has a defined exit condition)
- [ ] Are `defer` statements in the right place (within the function scope they clean up, not in a loop)?
- [ ] Does graceful shutdown still work with any new server resources (DB connections, goroutines)?

---

## Frontend-Specific

- [ ] **[BLOCKING]** Are all props typed with explicit TypeScript interfaces? (No `any`, no implicit `any`)
- [ ] Is loading state handled? (Skeleton, spinner, or disabled state)
- [ ] Is error state handled with a user-friendly message? (Not a raw error object, not a blank screen)
- [ ] Is the empty/zero-data state handled? (Not undefined behavior when the API returns an empty array)
- [ ] Is server state managed with TanStack Query? (No `useEffect + fetch`)
- [ ] Are all form fields labeled and accessible?
- [ ] Is query cache invalidated after mutations that change list data?
- [ ] Are there any `any` types introduced? ESLint should catch this, but do a spot check.

---

## OpenAPI Contract

- [ ] If a new endpoint was added, is the contract in `contracts/{service}/openapi.yaml` updated?
- [ ] If an existing endpoint's request/response shape changed, is the contract updated?
- [ ] Do error responses use the standard error envelope (`{"error": {"code": "...", "message": "..."}}`)?
- [ ] Are new enum values documented in the contract?

---

## Operations

- [ ] If new environment variables are required, are they added to `.env.example` with a description?
- [ ] If the deployment configuration changes (new port, new volume, new resource requirement), is the relevant Docker Compose / Kubernetes manifest updated?
- [ ] If the change adds a significant new query or DB access pattern, is performance considered? (Index added if needed, N+1 avoided)
- [ ] Does the service still pass its health check with this change?

---

## PR Hygiene

- [ ] Is the PR description clear about what changed and why?
- [ ] Is the PR a reasonable size? PRs over ~500 lines of changed code are hard to review effectively — consider splitting.
- [ ] Does the PR title follow a clear format? (`feat: add case comments API`, `fix: correct tenant isolation in case queries`)
- [ ] Are there any merge conflicts to resolve?
