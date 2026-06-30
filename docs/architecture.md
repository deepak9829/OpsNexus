# OpsNexus — Architecture Document

## System Overview

OpsNexus is a multi-tenant operations management platform for service businesses. It enables organizations (tenants) to manage support cases, collect information via structured forms, store and version documents, automate repetitive workflows, and receive notifications about system events.

The system is built as a set of independently deployable Go microservices behind a unified REST API, backed by a React frontend. Multi-tenancy is enforced at every layer: every piece of data is scoped to a tenant, and the system guarantees that no tenant can see or modify another tenant's data.

---

## 3-Tier Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                      PRESENTATION TIER                           │
│                                                                  │
│   apps/web/          apps/admin/                                 │
│   (Tenant Users)     (Platform Admins)                           │
│   React + TypeScript + TanStack Query + Vite                     │
└──────────────────────────┬───────────────────────────────────────┘
                           │ HTTPS / REST API
                           │ Bearer JWT + X-Tenant-ID header
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│                      APPLICATION TIER                            │
│                                                                  │
│  ┌────────────┐ ┌──────────┐ ┌────────────┐ ┌──────────────┐   │
│  │    Auth    │ │  Cases   │ │ Documents  │ │  Workflows   │   │
│  │  Service   │ │ Service  │ │  Service   │ │   Service    │   │
│  │  :8081     │ │  :8082   │ │   :8083    │ │   :8084      │   │
│  └────────────┘ └──────────┘ └────────────┘ └──────────────┘   │
│                                                                  │
│  ┌────────────────┐                                              │
│  │  Notification  │                                              │
│  │   Service      │                                              │
│  │   :8085        │                                              │
│  └────────────────┘                                              │
│                                                                  │
│  All services implement Clean/Hexagonal Architecture             │
│  All services require JWT auth except /health                    │
└──────────────────────────┬───────────────────────────────────────┘
                           │
        ┌──────────────────┼────────────────────────────┐
        ▼                  ▼                            ▼
┌───────────────┐  ┌───────────────┐        ┌──────────────────┐
│  DATA TIER    │  │  DATA TIER    │        │   DATA TIER      │
│               │  │               │        │                  │
│  MySQL 8.0    │  │  MongoDB 7    │        │  DynamoDB        │
│               │  │               │        │  (LocalStack)    │
│  auth DB      │  │  documents DB │        │                  │
│  cases DB     │  │               │        │  notifications   │
│  workflows DB │  │               │        │  table           │
└───────────────┘  └───────────────┘        └──────────────────┘
```

**Invariant:** The Presentation tier never communicates directly with the Data tier. All reads and writes go through the Application tier.

---

## Service Responsibilities

### Auth Service (port 8081)

**Domain:** Identity, authentication, authorization, and tenant management.

**Responsibilities:**
- User registration, login, logout
- JWT token issuance and validation
- Tenant creation and configuration
- Role-based access control (RBAC): `super_admin`, `admin`, `agent`, `viewer`
- Password reset flows
- Session management

**Storage:** MySQL — tables: `tenants`, `users`, `sessions`, `roles`, `role_assignments`

**Critical paths:** Every other service depends on the Auth Service to validate JWTs on startup. The Auth Service is the trust anchor for the entire system.

---

### Case Service (port 8082)

**Domain:** Support case lifecycle management.

**Responsibilities:**
- Create, update, close, and reopen support cases
- Case assignment to agents
- Case comments and activity log
- Case priority and status transitions
- Case search and filtering

**Storage:** MySQL — tables: `cases`, `case_comments`, `case_activities`

**Dependencies:** Calls Auth Service to validate user existence when assigning cases. Publishes events to the Notification Service when case status changes.

---

### Document Service (port 8083)

**Domain:** Form templates, form submissions, and document versioning.

**Responsibilities:**
- Create and manage form templates
- Accept and store form submissions from clients
- Upload, version, and retrieve documents (PDFs, images, etc.)
- Generate presigned S3 upload/download URLs

**Storage:** MongoDB — collections: `forms`, `form_submissions`, `documents`, `document_versions`
**File storage:** S3 (LocalStack locally, AWS S3 in production)

---

### Workflow Service (port 8084)

**Domain:** Automated multi-step workflow definitions and execution.

**Responsibilities:**
- Define workflow templates (sequences of steps with conditions)
- Trigger workflow instances from case events or manual initiation
- Track step completion and transition to next steps
- Handle approval steps (human-in-the-loop)

**Storage:** MySQL — tables: `workflows`, `workflow_steps`, `workflow_instances`, `workflow_step_instances`

**Dependencies:** Reads case data to determine workflow eligibility. Publishes notifications on workflow completion/failure.

---

### Notification Service (port 8085)

**Domain:** Event-driven notifications delivery.

**Responsibilities:**
- Receive notification events from other services
- Fan out to in-app, email, and (future) SMS channels
- Track notification delivery status
- Store notification preferences per user

**Storage:** DynamoDB — tables: `notifications`, `notification_preferences`

**Note:** This service is event-driven. Other services POST to it to fire notifications; it doesn't query other services.

---

## Data Flow Diagrams

### User Login Flow

```
Browser                 Auth Service           MySQL
  │                         │                    │
  ├─POST /api/v1/auth/login─►│                    │
  │  {email, password,       │                    │
  │   tenantId}              │                    │
  │                         ├─SELECT user WHERE──►│
  │                         │  email=? AND         │
  │                         │  tenant_id=?         │
  │                         │◄─────────────────────┤
  │                         │                    │
  │                         ├─verify bcrypt hash  │
  │                         ├─issue JWT           │
  │                         │  {sub, tenantId,    │
  │                         │   role, exp}        │
  │◄────────────────────────┤                    │
  │  {data: {token, user}}  │                    │
```

### Authenticated Case Fetch

```
Browser           Auth Middleware     Case Service       MySQL
  │                    │                   │               │
  ├─GET /api/v1/cases──►│                   │               │
  │  Bearer: <jwt>      │                   │               │
  │  X-Tenant-ID: <id>  │                   │               │
  │                     ├─validate JWT      │               │
  │                     ├─extract tenantId  │               │
  │                     ├─set ctx locals────►               │
  │                     │                  ├─SELECT cases──►│
  │                     │                  │  WHERE          │
  │                     │                  │  tenant_id=?   │
  │                     │                  │◄───────────────┤
  │◄────────────────────────────────────────┤               │
  │  {data: [...], meta: {total, page}}     │               │
```

### Document Upload Flow

```
Browser          Document Service          S3/LocalStack
  │                    │                       │
  ├─POST /api/v1/documents/{caseId}/upload──►  │
  │  multipart/form-data                       │
  │                    ├─validate file type/size│
  │                    ├─generate storageKey   │
  │                    ├─store metadata────────►│
  │                    │  (MongoDB)             │
  │                    ├─PUT object─────────────►│
  │                    │  (S3)                  │
  │◄───────────────────┤                       │
  │  {data: {documentId, downloadUrl}}         │
```

---

## Inter-Service Communication

At this stage, inter-service communication is synchronous HTTP.

All requests between services include these headers:
- `Authorization: Bearer <service-to-service JWT>` — services use their own service account token
- `X-Tenant-ID: <uuid>` — tenant context is always propagated
- `X-User-ID: <uuid>` — the original user causing the operation
- `X-Request-ID: <uuid>` — for distributed tracing correlation

```
Cases Service ──────────────────────────────► Auth Service
  POST /api/v1/internal/users/validate        (validate user exists before assigning case)

Cases Service ──────────────────────────────► Notification Service
  POST /api/v1/notifications                  (case status change events)

Workflow Service ────────────────────────────► Cases Service
  PATCH /api/v1/cases/{id}/status             (workflow step updates case status)

Workflow Service ────────────────────────────► Notification Service
  POST /api/v1/notifications                  (workflow completion events)
```

Timeout: 5 seconds. Retries: 3 with exponential backoff (1s, 2s, 4s). Circuit breaking is a future concern.

---

## Security Model

### Authentication

All endpoints (except `/health` and `/api/v1/auth/login`, `/api/v1/auth/register`) require a valid JWT.

JWT payload:
```json
{
  "sub": "user-uuid",
  "tenantId": "tenant-uuid",
  "role": "agent",
  "exp": 1705320600,
  "iat": 1705234200
}
```

JWTs are signed with HS256 using a secret from environment config. The signing secret is shared between services to allow each service to independently validate tokens without an Auth Service round-trip on every request.

### Tenant Isolation

Every piece of user data is scoped to a tenant. The tenant ID is extracted from the validated JWT, never from the request body. A request claiming to be for tenant A with a JWT for tenant B is rejected.

Repository layer enforcement: every DB query includes `WHERE tenant_id = $tenantID`. There is no escape hatch.

### RBAC

| Role | Permissions |
|------|------------|
| `super_admin` | All actions across all tenants (platform management only) |
| `admin` | All actions within their tenant |
| `agent` | Create and manage cases; upload documents; run workflows |
| `viewer` | Read-only access to cases, documents, workflow status |

RBAC is enforced in the application layer, not the database. The role is extracted from the JWT and checked in the service before any mutation.

---

## Internal Architecture per Service (Clean/Hexagonal)

Every Go service follows this pattern:

```
domain/         Pure business entities. No imports beyond stdlib.
  ├── user.go
  ├── errors.go
  └── ...

ports/          Interfaces (contracts) for repos and external services.
  ├── user_repository.go
  ├── token_issuer.go
  └── ...

application/    Business logic. Imports only domain and ports.
  ├── auth_service.go
  ├── user_service.go
  └── ...

adapters/       Concrete implementations. Only layer with framework deps.
  ├── http/
  │   ├── handlers/
  │   ├── middleware/
  │   └── router.go
  ├── mysql/
  │   └── user_repository.go
  └── ...
```

Dependency direction: `adapters → application → ports ← domain`. Never reversed.

---

## Future State: AWS Deployment Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  AWS Region (us-east-1)                                             │
│                                                                     │
│  CloudFront ──► S3 (React apps)                                     │
│                                                                     │
│  Route 53 ──► ALB ──► EKS Cluster                                   │
│                          ├── auth-service Deployment                │
│                          ├── case-service Deployment                │
│                          ├── document-service Deployment            │
│                          ├── workflow-service Deployment            │
│                          └── notification-service Deployment        │
│                                                                     │
│  RDS MySQL 8.0 (Multi-AZ)                                           │
│  Amazon DocumentDB (MongoDB-compatible)                             │
│  Amazon DynamoDB                                                    │
│  Amazon S3 (document storage)                                       │
│  Amazon SNS + SQS (async notification delivery)                     │
│  AWS Secrets Manager (all service credentials)                      │
│  Amazon ECR (container registry)                                    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

Current local development components map directly to these AWS services without application code changes:
- Docker MySQL 8 → RDS MySQL 8
- Docker MongoDB 7 → DocumentDB
- LocalStack DynamoDB → DynamoDB
- LocalStack S3 → S3
- In-process notifications → SNS + SQS
