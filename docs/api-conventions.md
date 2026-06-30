# API Conventions — OpsNexus

All OpsNexus REST APIs follow these conventions. Consistency is more important than local optimization — follow these patterns even when they feel slightly verbose for a particular endpoint.

---

## URL Structure

```
/api/v1/{resource}
/api/v1/{resource}/{id}
/api/v1/{resource}/{id}/{sub-resource}
/api/v1/{resource}/{id}/{sub-resource}/{sub-id}
```

Examples:
```
GET  /api/v1/cases                          # list cases
POST /api/v1/cases                          # create case
GET  /api/v1/cases/abc-123                  # get case by ID
PUT  /api/v1/cases/abc-123                  # replace case (full update)
PATCH /api/v1/cases/abc-123                 # partial update
DELETE /api/v1/cases/abc-123               # delete case
GET  /api/v1/cases/abc-123/comments        # list comments for a case
POST /api/v1/cases/abc-123/comments        # add comment to case
GET  /api/v1/cases/abc-123/comments/def-456 # get specific comment
```

**URL naming rules:**
- Use kebab-case for multi-word resource names: `/api/v1/case-comments`, not `/api/v1/caseComments` or `/api/v1/case_comments`
- Use plural nouns for collections: `/api/v1/cases`, not `/api/v1/case`
- Use nouns, not verbs in URLs: `POST /api/v1/cases/{id}/close` not `POST /api/v1/closeCases/{id}`
  - Exception: actions that don't map cleanly to CRUD (e.g., `POST /api/v1/cases/{id}/assign`)

---

## HTTP Methods

| Method | Purpose | Body | Idempotent |
|--------|---------|------|-----------|
| GET | Retrieve resource(s) | None | Yes |
| POST | Create resource or trigger action | JSON | No |
| PUT | Replace entire resource | JSON | Yes |
| PATCH | Partial update | JSON (only changed fields) | No |
| DELETE | Remove resource | None (or optional body) | Yes |

**GET requests never have a request body.** Pass filter parameters as query strings.

---

## Response Envelopes

All responses use a consistent envelope. Never return a bare object or bare array.

### Success — Single Resource

```json
{
  "data": {
    "id": "abc-123",
    "tenantId": "tenant-456",
    "title": "Server is down in prod",
    "status": "open",
    "priority": "critical",
    "createdAt": "2024-01-15T10:30:00Z",
    "updatedAt": "2024-01-15T10:30:00Z"
  },
  "meta": {
    "requestId": "req-789",
    "timestamp": "2024-01-15T10:30:01Z"
  }
}
```

### Success — Collection with Pagination

```json
{
  "data": [
    { "id": "abc-123", "title": "...", "status": "open" },
    { "id": "abc-124", "title": "...", "status": "in_progress" }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 145,
    "totalPages": 8,
    "requestId": "req-789",
    "timestamp": "2024-01-15T10:30:01Z"
  }
}
```

### Error

```json
{
  "error": {
    "code": "CASE_NOT_FOUND",
    "message": "the requested case was not found"
  },
  "meta": {
    "requestId": "req-789",
    "timestamp": "2024-01-15T10:30:01Z"
  }
}
```

### Validation Error (multiple field errors)

```json
{
  "error": {
    "code": "CASE_VALIDATION_FAILED",
    "message": "request validation failed",
    "fields": {
      "title": "title is required",
      "priority": "priority must be one of: low, medium, high, critical"
    }
  },
  "meta": {
    "requestId": "req-789",
    "timestamp": "2024-01-15T10:30:01Z"
  }
}
```

---

## Error Codes

Error codes follow the convention: `{SERVICE}_{NOUN}_{PROBLEM}`

| Component | Values |
|-----------|--------|
| SERVICE | `AUTH`, `CASE`, `DOC`, `WORKFLOW`, `NOTIF` |
| NOUN | `TOKEN`, `USER`, `TENANT`, `CASE`, `DOCUMENT`, `FORM`, `WORKFLOW`, `PERMISSION` |
| PROBLEM | `EXPIRED`, `INVALID`, `NOT_FOUND`, `CONFLICT`, `FORBIDDEN`, `VALIDATION_FAILED`, `RATE_LIMITED` |

Examples:
```
AUTH_TOKEN_EXPIRED           JWT has passed its expiry
AUTH_TOKEN_INVALID           JWT is malformed or signature invalid
AUTH_USER_NOT_FOUND          User doesn't exist
AUTH_TENANT_NOT_FOUND        Tenant doesn't exist
AUTH_PERMISSION_FORBIDDEN    User's role doesn't allow this action
CASE_NOT_FOUND               Case doesn't exist or tenant mismatch
CASE_VALIDATION_FAILED       Request body failed validation
CASE_CONFLICT                Concurrent modification conflict
DOC_DOCUMENT_TOO_LARGE       File exceeds size limit
DOC_DOCUMENT_TYPE_INVALID    File type not allowed
WORKFLOW_NOT_FOUND           Workflow template doesn't exist
INTERNAL_ERROR               Unhandled server error (no internal details)
```

---

## HTTP Status Codes

| Status | When to use |
|--------|------------|
| 200 OK | Successful GET, PUT, PATCH |
| 201 Created | Successful POST that created a resource |
| 204 No Content | Successful DELETE |
| 400 Bad Request | Malformed request body, missing required fields |
| 401 Unauthorized | Missing or invalid auth token |
| 403 Forbidden | Valid token but insufficient permissions |
| 404 Not Found | Resource doesn't exist |
| 409 Conflict | Concurrent modification, duplicate unique field |
| 413 Payload Too Large | Request body or file exceeds size limit |
| 422 Unprocessable Entity | Request is syntactically valid but semantically wrong (e.g., referencing a non-existent parent entity) |
| 429 Too Many Requests | Rate limit exceeded |
| 500 Internal Server Error | Unexpected server error |
| 503 Service Unavailable | Service or dependency temporarily down |

---

## Pagination

All list endpoints support pagination via query parameters:

```
GET /api/v1/cases?page=2&limit=20
```

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `page` | integer | 1 | — | 1-based page number |
| `limit` | integer | 20 | 100 | Items per page |

Response meta for paginated responses:
```json
"meta": {
  "page": 2,
  "limit": 20,
  "total": 145,
  "totalPages": 8
}
```

When `total` is expensive to compute (e.g., large aggregations), it may be omitted and `totalPages` set to `-1` to signal "unknown". This should be documented per-endpoint in the OpenAPI spec.

---

## Filtering and Sorting

Filtering via query parameters. Parameter names match the field names in the response body (camelCase):

```
GET /api/v1/cases?status=open&priority=critical&assignedTo=user-123
GET /api/v1/cases?createdAfter=2024-01-01T00:00:00Z
```

Sorting:
```
GET /api/v1/cases?sortBy=createdAt&sortOrder=desc
```

| Parameter | Values | Default |
|-----------|--------|---------|
| `sortBy` | Field name | `createdAt` |
| `sortOrder` | `asc`, `desc` | `desc` |

Not all fields are sortable. The OpenAPI spec documents which sort fields are supported.

---

## Authentication

All endpoints except `/health`, `POST /api/v1/auth/login`, and `POST /api/v1/auth/register` require authentication.

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

Token expiry: 24 hours (configurable via `JWT_EXPIRY_SECONDS`).

On token expiry, the client receives `401 AUTH_TOKEN_EXPIRED` and must re-authenticate.

---

## Tenant Context

All non-auth endpoints require the tenant context header:

```
X-Tenant-ID: tenant-uuid-here
```

This header is validated against the tenant ID in the JWT. If they don't match, the request is rejected with 403.

The Auth Service login response includes the tenant ID:
```json
{
  "data": {
    "token": "...",
    "user": { "id": "...", "tenantId": "tenant-uuid-here", ... }
  }
}
```

The frontend stores both the token and tenant ID and sends them on every request.

---

## Request Tracing

Every request should include a trace ID for debugging:

```
X-Request-ID: client-generated-uuid
```

If not provided by the client, the server generates one. The `X-Request-ID` is:
- Added to all log entries for the request
- Returned in the response `meta.requestId` field
- Propagated to downstream service calls

---

## Data Types and Formats

| Type | Format | Example |
|------|--------|---------|
| Timestamps | ISO 8601 UTC | `2024-01-15T10:30:00Z` |
| Dates (no time) | ISO 8601 date | `2024-01-15` |
| All IDs | UUID v4 string | `abc12345-1234-1234-1234-abc123456789` |
| Enums | lowercase string | `"status": "in_progress"` |
| JSON field names | camelCase | `"createdAt"`, `"tenantId"` |
| Booleans | JSON boolean | `"isActive": true` (not `"1"` or `"yes"`) |
| Currency | Integer cents | `"amountCents": 1999` (not `19.99`) |
| File sizes | Integer bytes | `"sizeBytes": 1048576` |

---

## Content Type

All request and response bodies use `Content-Type: application/json`. File uploads use `Content-Type: multipart/form-data`.

---

## Rate Limiting

Rate limits are applied per tenant per endpoint:

| Tier | Limit |
|------|-------|
| Default | 100 requests/minute |
| Auth endpoints | 20 requests/minute |
| File upload | 10 requests/minute |

When a rate limit is exceeded:
```
HTTP 429 Too Many Requests
Retry-After: 45
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1705234260

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "rate limit exceeded, retry after 45 seconds"
  }
}
```

---

## OpenAPI Specification

The authoritative API contract lives in `contracts/{service}/openapi.yaml`. The implementation must match the spec. If they diverge, the spec wins and the implementation is fixed.

Specs follow OpenAPI 3.1. All requests and responses are defined with JSON Schema. Enum values, required fields, and field descriptions are always complete.

When adding or changing an endpoint:
1. Update `contracts/{service}/openapi.yaml` first
2. Review the spec change in the PR
3. Implement to match the spec
4. Write tests that verify the shape matches the spec (response validation middleware in integration tests)
