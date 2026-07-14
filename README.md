# OpsNexus

A multi-tenant **Service Operations Hub** — a production-grade Go + React monorepo that provides centralized workflow management, document handling, tenant administration, and real-time notifications for B2B SaaS teams.

---

## What it does

OpsNexus lets organizations onboard multiple tenants into a single platform where each tenant gets isolated:

- **Authentication & RBAC** — JWT-based auth with per-tenant role assignments
- **Tenant management** — onboarding, configuration, billing tier enforcement
- **Workflow engine** — multi-step approval workflows with state machine transitions
- **Document store** — versioned document upload, retrieval, and metadata search
- **Notification hub** — real-time and async notification delivery with full audit trail

---

## Architecture Overview

OpsNexus runs in two modes: locally via Docker Compose or Kubernetes, and in production on AWS EKS.

### AWS Deployment

> Full interactive diagram: [`docs/aws-architecture.html`](docs/aws-architecture.html)

```
Internet
  ├── Route 53 (opsnexus.site)
  │     ├── app.dev.opsnexus.site   → CloudFront → S3 (customer-portal/)
  │     ├── admin.dev.opsnexus.site → CloudFront → S3 (admin-console/)
  │     └── api.dev.opsnexus.site   → API Gateway (Regional, ap-south-1)
  │                                       │
  │                                 Lambda JWT Authorizer (Node.js 20.x)
  │                                       │
  │                                   VPC Link → NLB (:30080)
  │
  └── VPC 10.0.0.0/16  ·  ap-south-1  ·  3 AZs
        │
        ├── Public subnets (.1–.3/24)
        │     Internet Gateway · NAT Gateway (1a) · NLB
        │
        ├── EKS private subnets (.32, .64, .128 /19)
        │     EKS v1.36 · opsnexus-dev-eks
        │       ├── System nodes: 2× t3.medium (on-demand)
        │       ├── Karpenter v1.13: c/m/r families, spot+OD, al2023@latest
        │       └── Namespace: opsnexus
        │             Traefik (NodePort :30080) → ClusterIP routing
        │             auth :8081  tenant :8082  workflow :8083
        │             document :8084  notification :8085 (IRSA → DynamoDB)
        │             External Secrets Operator (IRSA → Secrets Manager)
        │
        └── DB private subnets (.200–.202/24)
              RDS MySQL 8.0 (db.t3.small) · DocumentDB 5.0 (db.t3.medium)

Managed services (outside VPC):
  DynamoDB ×2 (notifications, audit_logs, on-demand)
  ECR ×5 repos (multi-arch amd64+arm64)
  Secrets Manager ×4 · KMS ×5
  SQS + EventBridge (Karpenter spot interruption handling)
  IRSA roles: karpenter-controller · eso · notification-service
  API GW usage plans: Basic 10 rps · Pro 100 rps · Enterprise 1000 rps
```

### Local service graph

```
customer-portal (:3000)     admin-console (:3001)
         └──────────────┬──────────────┘
                        ▼
      auth    tenant   workflow  document  notification
    (:8081) (:8082)  (:8083)  (:8084)    (:8085)
       │       │        │        │            │
     MySQL   MySQL    MySQL   MongoDB      DynamoDB
                                          (LocalStack)
```

All services communicate over the `opsnexus-net` Docker bridge network. Each service owns its datastore — no shared DB connections.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Build Go services |
| Node.js | 20+ | Build React frontends |
| Docker Desktop | 24+ | Docker Compose + Kubernetes |
| kubectl | 1.28+ | Kubernetes CLI |
| AWS CLI | v2 | LocalStack table setup |

---

## Local Deployment — Option 1: Docker Compose

The simplest way to run everything locally. All services, databases, and frontends run as Docker containers.

### 1. Configure environment

```bash
cp .env.example .env
# Required: set LOCALSTACK_AUTH_TOKEN (get it from app.localstack.cloud)
# Optional: change JWT_SECRET to a 32+ char random string
```

### 2. Start infrastructure (MySQL, MongoDB, LocalStack)

```bash
docker compose up -d
```

### 3. Create DynamoDB tables

```bash
bash scripts/setup-localstack.sh
```

### 4. Start all application services

```bash
docker compose --profile app up -d
```

### 5. Start frontends

```bash
cd frontend/customer-portal && npm install && npm run dev   # http://localhost:3000
cd frontend/admin-console   && npm install && npm run dev   # http://localhost:3001
```

### Service ports (Docker Compose)

| Service | Port |
|---------|------|
| Auth Service | 8081 |
| Tenant Service | 8082 |
| Workflow Service | 8083 |
| Document Service | 8084 |
| Notification Service | 8085 |
| Customer Portal | 3000 |
| Admin Console | 3001 |
| MySQL | 3306 |
| MongoDB | 27017 |
| LocalStack (DynamoDB) | 4566 |

### Seed demo data (optional)

```bash
bash scripts/seed-data.sh
```

### Default credentials

| Account | Email | Password |
|---------|-------|----------|
| Admin | admin@opsnexus.com | Admin123! |
| User | sarah.chen@opsnexus.com | Pass1234! |

### Scale down

```bash
docker compose --profile app down   # stop app services only
docker compose down                 # stop everything including infra
```

---

## Local Deployment — Option 2: Kubernetes (Docker Desktop)

Runs the full stack on a single-node Kubernetes cluster using Docker Desktop's built-in Kubernetes engine. Kustomize manifests live in `k8s/`.

### 1. Enable Kubernetes in Docker Desktop

Open Docker Desktop → **Settings** → **Kubernetes** → check **Enable Kubernetes** → **Apply & Restart**. Wait ~2 minutes until the status bar shows Kubernetes running.

### 2. Switch kubectl context

```bash
kubectl config use-context docker-desktop
kubectl get nodes   # should show docker-desktop node in Ready state
```

### 3. Configure environment

```bash
cp .env.example .env
# Required: set LOCALSTACK_AUTH_TOKEN in .env
```

### 4. Build service images

Docker Desktop Kubernetes shares the local Docker daemon, so locally built images are available to the cluster immediately.

```bash
docker build -t opsnexus-auth-service:latest         ./services/auth-service
docker build -t opsnexus-tenant-service:latest        ./services/tenant-service
docker build -t opsnexus-workflow-service:latest      ./services/workflow-service
docker build -t opsnexus-document-service:latest      ./services/document-service
docker build -t opsnexus-notification-service:latest  ./services/notification-service
```

### 5. Inject secrets

Reads `LOCALSTACK_AUTH_TOKEN` and other credentials from `.env` and creates a Kubernetes Secret — nothing sensitive is written to git-tracked files.

```bash
bash k8s/inject-secrets.sh
```

### 6. Deploy with Kustomize

```bash
kubectl apply -k k8s/overlays/local
```

### 7. Wait for pods to be ready

```bash
kubectl get pods -n opsnexus -w
```

All 9 pods should reach `Running` status within ~2 minutes (LocalStack Pro may take slightly longer on first start).

### Service ports (Kubernetes NodePort)

| Service | NodePort |
|---------|----------|
| Auth Service | 30081 |
| Tenant Service | 30082 |
| Workflow Service | 30083 |
| Document Service | 30084 |
| Notification Service | 30085 |

Frontends can be pointed at `localhost:30081`–`30085` instead of the Docker Compose ports.

### Scale down

```bash
# Suspend all workloads (preserves namespace and configs)
kubectl scale deployment --all -n opsnexus --replicas=0

# Delete everything in the namespace
kubectl delete namespace opsnexus
```

---

## Kubernetes Manifest Structure

```
k8s/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── secrets.yaml              ← placeholder only, gitignored
│   ├── infrastructure/
│   │   ├── mysql/                ← PV, PVC, ConfigMap (init SQL), Deployment, Service
│   │   ├── mongodb/              ← PV, PVC, Deployment, Service
│   │   └── localstack/           ← PV, PVC, Deployment, Service, init Job
│   └── services/
│       ├── auth-service/
│       ├── tenant-service/
│       ├── workflow-service/
│       ├── document-service/
│       └── notification-service/
├── overlays/
│   └── local/                    ← NodePort patch, local image settings
├── inject-secrets.sh             ← creates k8s Secret from .env
└── kubeadm-setup.sh              ← optional: full kubeadm single-node setup
```

---

## Directory Structure

```
OpsNexus/
├── go.work                         # Go workspace (links all 5 services)
├── Makefile                        # Top-level dev commands
├── docker-compose.yml              # All infra + app services
├── .env.example                    # Environment variable documentation
│
├── services/
│   ├── auth-service/               # JWT auth, users, RBAC
│   │   ├── cmd/server/main.go
│   │   ├── cmd/migrate/main.go
│   │   ├── cmd/seed/main.go
│   │   ├── internal/
│   │   └── go.mod
│   ├── tenant-service/             # Tenant onboarding & config
│   ├── workflow-service/           # Approval workflow engine
│   ├── document-service/           # Document storage & retrieval
│   └── notification-service/       # Async notifications & audit
│
├── frontend/
│   ├── customer-portal/            # React app for end-users (:3000)
│   └── admin-console/              # React app for ops admins (:3001)
│
├── k8s/                            # Kubernetes manifests (Kustomize)
│   ├── base/                       # Base layer (all environments)
│   ├── overlays/local/             # Docker Desktop local overlay
│   ├── inject-secrets.sh           # Creates k8s Secret from .env
│   └── kubeadm-setup.sh            # Optional kubeadm cluster setup
│
├── contracts/                      # OpenAPI / AsyncAPI specs
│   ├── auth-service.yaml
│   ├── tenant-service.yaml
│   ├── workflow-service.yaml
│   ├── document-service.yaml
│   └── notification-service.yaml
│
├── docs/                           # Architecture diagrams, ADRs
└── scripts/
    ├── mysql-init.sql              # DB/user creation on first run
    ├── setup-localstack.sh         # DynamoDB table provisioning
    └── seed-data.sh                # Demo data seeding
```

---

## Running Tests

```bash
# All Go service tests (with race detector + coverage)
make test-all

# Single service
cd services/auth-service && go test ./... -v -race -coverprofile=coverage.out

# All frontend tests (Vitest)
make fe-test-all

# Single frontend
cd frontend/customer-portal && npm test
```

---

## API Contracts

OpenAPI 3.1 specs live in `/contracts/`. Each service exposes its spec at `/swagger/` in development mode. Use these specs to generate client SDKs or test with tools like Postman / Hoppscotch.

```
contracts/
├── auth-service.yaml           # /api/v1/auth, /api/v1/users
├── tenant-service.yaml         # /api/v1/tenants
├── workflow-service.yaml       # /api/v1/workflows, /api/v1/steps
├── document-service.yaml       # /api/v1/documents
└── notification-service.yaml   # /api/v1/notifications
```

---

## Project Skills / Instructions

Claude Code project instructions are in `/skills/CLAUDE.md`. These encode the conventions used throughout this codebase:

- Package layout (Clean Architecture per service)
- Error handling patterns
- Testing conventions (table-driven tests, testify)
- Migration file naming
- PR checklist

---

## Technology Choices

| Choice | Rationale |
|--------|-----------|
| **Go 1.22** | Excellent concurrency primitives, small Docker images, fast builds. Workspace mode (`go.work`) lets all 5 services live in one repo without publishing modules. |
| **MySQL 8.0** | Battle-tested ACID compliance for transactional data (auth, tenant config, workflows). Per-service databases enforce bounded contexts. |
| **DocumentDB 5.0** | MongoDB-compatible API, AWS-managed, multi-AZ capable. Flexible schema for documents with varying metadata shapes. |
| **DynamoDB (on-demand)** | Single-digit millisecond reads for notifications and audit logs; scales to zero cost at rest. |
| **React + Vite** | Vite's instant HMR dramatically speeds up frontend iteration. React ecosystem maturity for complex admin UIs. |
| **EKS + Karpenter** | Managed control plane reduces ops burden; Karpenter provisions right-sized nodes on demand (c/m/r families, spot+OD) rather than pre-provisioning fixed node groups. |
| **Traefik as ingress** | Single NodePort entry point into the cluster; path-based routing to all 5 services without per-service load balancers. Deployed via Helm, configured with IngressRoutes. |
| **External Secrets Operator** | Secrets Manager values are projected into Kubernetes Secrets at runtime via IRSA — no credentials ever touch CI or git. |
| **API Gateway + Lambda authorizer** | Centralises JWT validation at the edge; 300 s cache means the authorizer Lambda is not called on every request. Usage plans enforce per-tier rate limits (10 / 100 / 1000 rps). |
| **Terraform (12 modules)** | Each AWS concern (vpc, subnets, routing, eks, rds, ecr, …) is a self-contained module. S3-backed remote state with per-environment workspaces. |
| **Docker Compose profiles** | `profiles: ["app"]` keeps infra and app containers decoupled — developers can run services locally against Dockerized infra without rebuilding images on every change. |
| **golangci-lint** | Aggregates 60+ linters in a single fast pass; catches bugs (errcheck, staticcheck) and style issues before review. |

---

## Contributing

1. Fork the repo and create a feature branch: `git checkout -b feat/your-feature`
2. Follow the coding conventions in `/skills/CLAUDE.md`
3. Run `make fmt-all lint-all test-all` before opening a PR
4. Open a PR against `main` — CI will run the full test suite

---

## License

MIT — see [LICENSE](LICENSE).
