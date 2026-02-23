# Infraplane

**Describe your app. Get a cloud.**

Infraplane is a cloud infrastructure platform that turns natural language into production-ready infrastructure. Point it at your codebase, pick AWS or GCP, and get a complete hosting plan — with Terraform configs, cost estimates, compliance enforcement, and one-click deployment — in under a minute.

It works two ways:
- **Web dashboard** — a guided onboarding wizard and full management UI with deploy, monitor, and review workflows
- **Claude Code + MCP** — infrastructure generation happens in real-time as you build

---

## Quick Demo

```
1. Browse to http://localhost:5173/onboard
2. Choose AWS or GCP
3. Select your project folder
4. Wait ~60 seconds
5. Get: detected resources, hosting plan, cost estimate, and per-resource Terraform HCL
6. Click Deploy — review the generated Terraform — approve and apply
```

---

## How It Works

```
Your Code ──► Infraplane Analyzer ──► LLM (Claude Sonnet 4.5) ──► Resources + Plan + Terraform
                    │                         │
                    │ Extracts infra files     │ Interprets needs,
                    │ (Dockerfile, k8s,        │ maps to cloud services,
                    │  Terraform, CI/CD...)     │ generates Terraform HCL
                    │                         │
                    ▼                         ▼
              16 file types            Cloud-agnostic Resource model
              auto-detected            with AWS + GCP provider mappings
```

When you register an application, Infraplane:

1. **Scans** your codebase for infrastructure signals (Dockerfiles, `docker-compose.yml`, Kubernetes manifests, Terraform files, CI/CD workflows, `package.json`, `requirements.txt`, and more)
2. **Analyzes** those files with an LLM to understand what your app actually needs
3. **Creates** cloud-agnostic resources — a `database`, a `cache`, a `queue` — independent of any provider
4. **Maps** each resource to concrete cloud services: RDS vs. Cloud SQL, ElastiCache vs. Memorystore
5. **Generates** a hosting plan with architecture recommendations and monthly cost estimates
6. **Produces** a complete Terraform configuration in a single LLM call — all resources properly wired together with shared infrastructure (VPC, subnets, IAM) declared once
7. **Deploys** with a two-phase review flow: generate Terraform → review per-resource HCL → approve and apply

---

## Features

### Onboarding Wizard
A 4-step guided flow: pick a provider → browse your code → wait for analysis → get your full hosting plan with resources, costs, and Terraform configs.

### Codebase Analysis
The analyzer extracts 16+ infrastructure file types from your project and feeds them to the LLM for intelligent resource detection. No manual tagging required. Analysis history is tracked per application with full run logs.

### Two-Phase Deploy with Terraform Review
Deployments pause after Terraform generation so you can review before applying. The UI shows expandable per-resource HCL blocks — one for each cloud resource — with Approve & Apply or Reject buttons. The deploy pipeline streams real-time SSE events through each step: initialize → generate Terraform → review → validate credentials → apply.

### Single-Call Terraform Generation
All resources for an application are sent to the LLM in one call, producing a complete, cohesive Terraform configuration where shared infrastructure (VPC, subnets, firewall rules, IAM) is declared once and all resources are properly wired together. The LLM also returns a per-resource HCL breakdown for the review UI. A deduplication safety net catches any remaining duplicate blocks.

### Live Resource Discovery
Discover what's already running in your cloud account. Infraplane generates targeted CLI commands (`gcloud`, `aws`), executes them in a secure sandbox, and maps the results back to your application. Also supports GCP Cloud Asset Inventory for comprehensive project-wide scans.

### Infrastructure Topology Graphs
Visualize your application's infrastructure as an interactive directed graph — compute nodes, databases, caches, queues, and the edges between them — rendered with React Flow and dagre layout.

### Hosting Plans & Cost Estimates
LLM-generated architecture recommendations with monthly cost breakdowns by resource category (compute, database, storage, networking, etc.).

### Compliance Frameworks
Apply compliance frameworks (SOC 2, HIPAA, PCI-DSS, CIS) to your application. The LLM receives framework-specific rules when generating Terraform HCL and enforces them with inline comments referencing each rule.

### Migration Planning
Generate a step-by-step plan to move your application between AWS and GCP, including service mappings, data migration strategies, and new Terraform configurations.

### GCP Credential Management
Upload GCP service account keys directly in the UI. Infraplane validates credentials, auto-detects the project ID, and lists available projects.

### Deploy Log Reconnection
Navigate away during a deploy and come back — the UI reconnects to the SSE stream and replays all events from where you left off. Works for both active streams and completed deployments.

### MCP Integration
All features are available as MCP tools for Claude Code. Infraplane can run as an MCP server over stdio, so Claude Code can create resources, generate plans, and discover infrastructure in real-time as you build.

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Claude Code                              │
│               (MCP over stdio transport)                      │
└────────────────────────┬─────────────────────────────────────┘
                         │
┌────────────────────────▼─────────────────────────────────────┐
│                   Infraplane Server                           │
│                                                               │
│   ┌────────────┐   ┌────────────┐   ┌──────────────────────┐ │
│   │ MCP Server │   │  REST API  │   │    LLM Engine        │ │
│   │ (12 tools) │   │(35 endpts) │   │ (Claude Sonnet 4.5)  │ │
│   └──────┬─────┘   └──────┬─────┘   └──────────┬───────────┘ │
│          │                │                     │             │
│          └────────────────┼─────────────────────┘             │
│                           │                                   │
│                  ┌────────▼────────┐                          │
│                  │  Service Layer  │                          │
│                  │  (7 services)   │                          │
│                  └────────┬────────┘                          │
│                           │                                   │
│          ┌────────────────┼────────────────┐                  │
│          ▼                ▼                ▼                  │
│   ┌────────────┐   ┌──────────┐   ┌────────────┐            │
│   │ PostgreSQL │   │ Analyzer │   │  Executor  │            │
│   │ (pgx v5)   │   │ (16 file │   │ (secure    │            │
│   │            │   │  types)  │   │  CLI runs) │            │
│   └────────────┘   └──────────┘   └────────────┘            │
│                                                               │
│   ┌──────────────────────────────────────────────────────┐   │
│   │  Provider Adapters: AWS │ GCP │ Terraform Generator  │   │
│   │                         (with HCL deduplication)      │   │
│   └──────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────┐
│                    React Dashboard (Vite)                      │
│  ┌─────────┐ ┌────────┐ ┌──────────┐ ┌────────┐ ┌─────────┐ │
│  │ Onboard │ │  Apps   │ │ App      │ │ Deploy │ │Migration│ │
│  │ Wizard  │ │  List   │ │ Detail   │ │ Board  │ │ Planner │ │
│  └─────────┘ └────────┘ └──────────┘ └────────┘ └─────────┘ │
│  ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌────────────────┐  │
│  │InfraGraph│ │LiveResources│ │ Deploy  │ │   Terraform    │  │
│  │(ReactFlow)│ │  Table    │ │  Log    │ │   Review       │  │
│  └──────────┘ └───────────┘ └──────────┘ └────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

### Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26 |
| Database | PostgreSQL 16 |
| Frontend | React 19, TypeScript, Tailwind CSS |
| Build | Vite 6 |
| LLM | Claude Sonnet 4.5 via Anthropic API |
| MCP | [mcp-go](https://github.com/mark3labs/mcp-go) v0.43 |
| DB Driver | [pgx](https://github.com/jackc/pgx) v5 |
| Router | [chi](https://github.com/go-chi/chi) v5 |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) v4 |
| State | [TanStack Query](https://tanstack.com/query) v5 |
| Graphs | [React Flow](https://reactflow.dev/) + [dagre](https://github.com/dagrejs/dagre) |
| Testing | [testcontainers-go](https://github.com/testcontainers/testcontainers-go), Vitest |
| Container Runtime | [Colima](https://github.com/abiosoft/colima) |

---

## Getting Started

### Prerequisites

- **Go 1.26+** — `brew install go`
- **Node.js 20+** — `brew install node`
- **Colima** (Docker runtime) — `brew install colima docker`
- **Anthropic API Key** — for LLM features

### Quick Start

```bash
# Clone
git clone https://github.com/matthewdriscoll/infraplane.git
cd infraplane

# Environment
cp .env.example .env
# Add your ANTHROPIC_API_KEY to .env

# Start Docker + PostgreSQL
colima start --cpu 2 --memory 4
docker compose up -d postgres

# Backend
make deps
make migrate
make dev          # Starts on :8080

# Frontend (separate terminal)
make web          # Starts on :5173
```

Open [http://localhost:5173/onboard](http://localhost:5173/onboard) to try the onboarding wizard.

### Connecting Claude Code via MCP

```json
{
  "mcpServers": {
    "infraplane": {
      "command": "/path/to/infraplane",
      "args": [],
      "env": {
        "DATABASE_URL": "postgres://infraplane:infraplane@localhost:5432/infraplane?sslmode=disable",
        "ANTHROPIC_API_KEY": "sk-ant-xxxxx"
      }
    }
  }
}
```

---

## REST API

All endpoints are prefixed with `/api`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/applications/onboard` | Full onboarding: register + analyze + plan |
| `POST` | `/applications` | Register an application |
| `GET` | `/applications` | List all applications |
| `GET` | `/applications/{name}` | Get application details |
| `DELETE` | `/applications/{name}` | Delete an application |
| `POST` | `/applications/{name}/reanalyze` | Re-analyze source code |
| `POST` | `/applications/{name}/analyze-upload` | Analyze uploaded files |
| `POST` | `/applications/{name}/resources` | Add a resource (LLM-powered) |
| `GET` | `/applications/{name}/resources` | List resources |
| `DELETE` | `/resources/{id}` | Remove a resource |
| `POST` | `/resources/{id}/terraform` | Generate Terraform HCL |
| `POST` | `/applications/{name}/hosting-plan` | Generate hosting plan |
| `POST` | `/applications/{name}/migration-plan` | Generate migration plan |
| `GET` | `/applications/{name}/plans` | List plans |
| `POST` | `/applications/{name}/graph` | Generate infrastructure graph |
| `GET` | `/applications/{name}/graph` | Get latest graph |
| `POST` | `/applications/{name}/live-resources` | Discover live resources |
| `POST` | `/applications/{name}/deploy` | Create deployment |
| `GET` | `/applications/{name}/deployments` | List deployments |
| `GET` | `/deployments/{id}` | Get deployment status |
| `GET` | `/deployments/{id}/stream` | Stream deployment events (SSE) |
| `GET` | `/deployments/{id}/approve` | Approve and apply deployment (SSE) |
| `POST` | `/deployments/{id}/reject` | Reject deployment |
| `POST` | `/gcp/credentials` | Upload GCP credentials |
| `GET` | `/gcp/credentials/status` | Check GCP credential status |
| `GET` | `/gcp/projects` | List GCP projects |
| `POST` | `/applications/{name}/validate-target` | Validate deploy target credentials |
| `GET` | `/applications/{name}/analysis-runs` | List analysis run history |
| `GET` | `/health` | Health check |

---

## MCP Tools

12 tools available when running as an MCP server:

| Tool | Description | LLM |
|------|-------------|:---:|
| `register_application` | Register app with auto-detection from source path | ✦ |
| `list_applications` | List all registered applications | |
| `get_application` | Get app details with resources | |
| `add_resource` | Describe a resource in natural language | ✦ |
| `remove_resource` | Remove a resource | |
| `get_hosting_plan` | Generate hosting plan with cost estimates | ✦ |
| `plan_migration` | Generate cross-provider migration plan | ✦ |
| `deploy` | Trigger deployment | |
| `get_deployment_status` | Check deployment status | |
| `generate_graph` | Generate infrastructure topology graph | ✦ |
| `discover_live_resources` | Discover running cloud resources | ✦ |
| `generate_terraform_hcl` | Generate Terraform HCL for a resource | ✦ |

✦ = LLM-powered operation

---

## Core Domain Model

### The Resource Abstraction

The central concept is a cloud-agnostic `Resource`. When you say "I need a PostgreSQL database," Infraplane creates:

```go
Resource{
    Kind: "database",
    Name: "user-db",
    Spec: {"engine": "postgres", "version": "16"},
    ProviderMappings: {
        "aws": {ServiceName: "RDS",       Config: {"instance_class": "db.t3.micro"}},
        "gcp": {ServiceName: "Cloud SQL", Config: {"tier": "db-f1-micro"}},
    },
}
```

Each resource maps to concrete services on both providers. At deploy time, all resources are sent to the LLM in a single call to produce one cohesive Terraform config with shared infrastructure wired correctly.

### Supported Resource Kinds

| Kind | AWS | GCP |
|------|-----|-----|
| `compute` | ECS, EKS, Lambda, EC2 | Cloud Run, GKE, Cloud Functions |
| `database` | RDS, Aurora, DynamoDB | Cloud SQL, Firestore, Spanner |
| `storage` | S3 | Cloud Storage |
| `cache` | ElastiCache | Memorystore |
| `queue` | SQS, SNS | Pub/Sub, Cloud Tasks |
| `cdn` | CloudFront | Cloud CDN |
| `network` | VPC, ALB | VPC, Cloud Load Balancing |
| `secrets` | Secrets Manager, SSM | Secret Manager |
| `policy` | IAM Roles & Policies | IAM Service Accounts & Bindings |

### Entity Relationships

```
Application ──┬── Resources ── ProviderMappings (AWS + GCP)
              ├── Deployments (two-phase: generate → review → apply)
              ├── InfrastructurePlans (hosting or migration, cost estimates)
              ├── InfraGraphs (topology: nodes + edges)
              └── AnalysisRuns (codebase scan history)
```

---

## Deployment Pipeline

Infraplane uses a two-phase deployment model with a human review gate:

```
Phase 1 (automatic):
  Initialize workspace
  → Generate complete Terraform config (single LLM call, all resources)
  → Pause at "awaiting_approval" status

  ┌─────────────────────────────────┐
  │  Terraform Review UI            │
  │  ┌───────────────────────────┐  │
  │  │ ▶ database: user-db       │  │
  │  │   (Cloud SQL — 42 lines)  │  │
  │  ├───────────────────────────┤  │
  │  │ ▶ cache: session-cache    │  │
  │  │   (Memorystore — 28 lines)│  │
  │  └───────────────────────────┘  │
  │  [Approve & Apply] [Reject]     │
  └─────────────────────────────────┘

Phase 2 (after approval):
  Validate cloud credentials
  → Terraform init + apply
  → Stream apply output in real-time
  → Mark succeeded or failed
```

Deployments that are left in `awaiting_approval` are automatically cleaned up on server restart.

---

## Project Structure

```
infraplane/
├── cmd/infraplane/main.go              # Entry point: MCP or HTTP mode
├── internal/
│   ├── domain/                         # Core models (zero external deps)
│   │   ├── application.go              # Application entity + status enum
│   │   ├── resource.go                 # Cloud-agnostic resource model
│   │   ├── deployment.go               # Deployment tracking + two-phase status
│   │   ├── plan.go                     # Infrastructure plans + cost estimates
│   │   ├── graph.go                    # Topology graph (nodes + edges)
│   │   ├── live_resource.go            # Live cloud resource tracking
│   │   ├── provider.go                 # Cloud provider enum
│   │   └── errors.go                   # Domain error types
│   ├── llm/                            # LLM integration
│   │   ├── anthropic.go                # Anthropic SDK client (Sonnet 4.5)
│   │   ├── client.go                   # Client interface
│   │   ├── prompts.go                  # Prompt templates (8 tasks incl. full-config)
│   │   └── mock.go                     # Mock client for tests
│   ├── service/                        # Business logic (7 services)
│   │   ├── application.go              # CRUD + auto-detect + onboarding
│   │   ├── resource.go                 # LLM-powered resource management
│   │   ├── planner.go                  # Hosting + migration planning
│   │   ├── graph.go                    # Topology graph generation
│   │   ├── discovery.go                # Live resource discovery
│   │   ├── deployment.go               # Two-phase deploy (Execute → Review → Resume)
│   │   ├── infra.go                    # Single-call Terraform generation + provider orchestration
│   │   └── eventstore.go              # In-memory SSE event store with pause/resume
│   ├── repository/                     # Data access layer
│   │   ├── interfaces.go               # Repository interfaces
│   │   ├── postgres/                   # PostgreSQL implementations (pgx v5)
│   │   └── mock/                       # In-memory mocks for unit tests
│   ├── analyzer/                       # Codebase analyzer (16+ file types)
│   ├── executor/                       # Secure CLI executor (read-only)
│   ├── compliance/                     # Compliance framework registry
│   ├── cloud/gcp/                      # GCP Cloud Asset Inventory
│   ├── provider/                       # Cloud provider adapters
│   │   ├── aws/                        # AWS adapter
│   │   ├── gcp/                        # GCP adapter
│   │   └── terraform/                  # Terraform HCL generator + deduplication
│   ├── mcp/                            # MCP server (12 tools)
│   └── api/                            # REST API (35 endpoints, chi router)
├── migrations/                         # 11 PostgreSQL migrations
├── web/                                # React + TypeScript frontend
│   ├── src/pages/                      # 5 pages
│   ├── src/components/                 # 11 components
│   ├── src/api/client.ts               # API client
│   ├── src/hooks/useApi.ts             # TanStack Query hooks
│   └── src/lib/directoryPicker.ts      # File System Access API
├── docker-compose.yml                  # PostgreSQL for local dev
├── Makefile                            # Build, test, dev commands
└── .env.example                        # Environment template
```

---

## Development

### Make Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/infraplane` |
| `make dev` | Start PostgreSQL + run server |
| `make test` | Unit tests (fast, no Docker) |
| `make test-integration` | Integration tests (requires Colima) |
| `make test-all` | All tests |
| `make migrate` | Run database migrations |
| `make migrate-down` | Rollback last migration |
| `make web` | Start frontend dev server |
| `make fmt` | Format and vet Go code |
| `make deps` | Tidy Go dependencies |
| `make clean` | Remove build artifacts |

### Testing

```bash
make test              # Unit tests — fast, no Docker
make test-integration  # Integration tests — needs Colima running
make test-all          # Everything
```

**Unit tests** use mock repositories and a mock LLM client. **Integration tests** spin up real PostgreSQL containers via testcontainers-go, run migrations, execute tests, and tear down.

### Environment Variables

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `ANTHROPIC_API_KEY` | Yes | — | Anthropic API key |
| `PORT` | No | `8080` | REST API port |
| `MCP_MODE` | No | `stdio` | MCP transport mode |

### Database

11 migrations manage the schema:

| Migration | Table / Change |
|-----------|-------|
| 001 | `applications` |
| 002 | `resources` (JSONB specs + provider mappings) |
| 003 | `deployments` |
| 004 | `plans` (hosting/migration + cost estimates) |
| 005 | `source_path` column on applications |
| 006 | `graphs` (topology nodes + edges) |
| 007 | Cascade deletes from application → resources |
| 008 | `compliance_frameworks` column on applications |
| 009 | `plan_id` column on deployments |
| 010 | `deploy_target` JSONB column on deployments |
| 011 | `analysis_runs` table |

Key decisions: UUID primary keys, JSONB for flexible schemas, cascading deletes from application → resources, indexed foreign keys.

### Docker / Colima

```bash
brew install colima docker
colima start --cpu 2 --memory 4
docker compose up -d postgres
```

Testcontainers need `TESTCONTAINERS_RYUK_DISABLED=true` under Colima — the Makefile sets this automatically.

---

## License

MIT
