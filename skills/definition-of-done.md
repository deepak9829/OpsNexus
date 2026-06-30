# Definition of Done — OpsNexus

A feature or bug fix is **Done** only when every item in this checklist is satisfied. "Done" means it could be deployed to production right now without manual intervention, breakage, or a follow-up ticket.

This is not aspirational — every item is a hard requirement. If something needs to be deferred, create a ticket, link it to the PR, and document the deferral explicitly. Don't silently leave DoD items incomplete.

---

## 1. Code

- [ ] Implementation matches the OpenAPI contract spec in `contracts/{service}/openapi.yaml`
- [ ] No `TODO` comments that don't have a corresponding issue/ticket reference
- [ ] No commented-out code
- [ ] No debug logging left in (`fmt.Printf`, hardcoded `log.Println`, console.log in production paths)
- [ ] Code review checklist has been completed by the reviewer
- [ ] All reviewer-blocking comments are resolved

---

## 2. Tests

- [ ] Unit tests exist for all new business logic in the application layer
- [ ] Integration test exists for any new DB adapter method
- [ ] Component test exists for any new React component (loading, error, success states)
- [ ] Multi-tenant isolation test exists for any new data access path
- [ ] All tests pass: `make test-all`
- [ ] No tests were deleted or skipped without explicit justification in a comment

---

## 3. Security

- [ ] All new HTTP endpoints are behind auth middleware
- [ ] Tenant isolation is verified on every new data access (tenant ID from JWT, not request body)
- [ ] User input is validated before use
- [ ] No sensitive data (passwords, tokens, PII) in logs or API responses
- [ ] Security test cases from the testing standards checklist pass for any new endpoint

---

## 4. API Contracts

- [ ] OpenAPI spec updated if any endpoint was added or changed
- [ ] Error responses use the standard error envelope (`{"error": {"code": "...", "message": "..."}}`)
- [ ] Success responses use the standard envelope (`{"data": ..., "meta": {...}}`)
- [ ] Pagination is implemented for any list endpoint returning variable-length results

---

## 5. Error Handling

- [ ] All error cases are handled — no unhandled errors, no swallowed errors
- [ ] Errors are logged at the appropriate level on the server
- [ ] User-facing error messages are friendly and actionable (no internal error details, no stack traces)
- [ ] HTTP status codes are semantically correct (404 for not found, 403 for forbidden, not all 500s)
- [ ] Domain errors are mapped to HTTP codes at the adapter boundary, not deeper

---

## 6. Documentation & Configuration

- [ ] `.env.example` updated if any new environment variable was added or an existing one changed
- [ ] If a significant pattern was introduced or changed, the relevant skill doc is updated (or a follow-up ticket is created and linked)
- [ ] If a new service was created, the `docs/architecture.md` is updated to reflect it
- [ ] Docker Compose is updated if the service's dependencies changed

---

## 7. Build

- [ ] `make build-all` passes cleanly (no errors, no warnings treated as errors)
- [ ] `make test-all` passes (all unit tests)
- [ ] `make lint-all` passes (golangci-lint and ESLint)
- [ ] `npm run type-check` passes for all affected frontend apps
- [ ] Docker image builds successfully: `docker build .` in the service directory

---

## 8. Local Dev Verification

- [ ] Feature works end-to-end with `make dev-up` + services running locally
- [ ] The change doesn't break any currently working features (tested manually or by existing tests)
- [ ] If a DB migration was added, it applies cleanly from a fresh DB and from an existing DB with data

---

## 9. Review

- [ ] At least one team member (not the author) has reviewed the code
- [ ] The reviewer has worked through the code review checklist
- [ ] All blocking items from the code review are resolved
- [ ] PR description explains what changed, why, and how to test it

---

## Things That Are NOT in the DoD (Tracked Separately)

The following are tracked and important, but do not block a PR from merging:

| Item | Where Tracked |
|------|-------------|
| Performance benchmarks vs. baselines | Load test results board |
| Accessibility audit against WCAG 2.1 AA | Accessibility backlog |
| Production deployment | Deployment board / release process |
| AWS infrastructure changes | Infra planning board |
| Observability (dashboards, alerts) | Platform backlog |

If a feature is performance-sensitive, run the load tests and document the results in the PR, even if they don't block merge.

---

## DoD Quick Reference

When submitting a PR, self-review against this summary:

```
Code:        Matches spec. Clean. No debug leftovers.
Tests:       Unit, integration, component. Multi-tenant isolation verified.
Security:    Auth + tenant isolation + input validation on every new endpoint.
Contracts:   OpenAPI updated if endpoints changed.
Errors:      All cases handled. Friendly user messages. Correct status codes.
Config:      .env.example updated. No hardcoded values.
Build:       make build-all, make test-all, make lint-all all pass.
Local:       Works end-to-end. Migration applies cleanly.
Review:      Code review checklist complete. Blocking items resolved.
```
