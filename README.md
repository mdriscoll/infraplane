# Infraplane

**Describe your app. Get a cloud.**

Infraplane is a cloud infrastructure platform that turns natural language into production-ready infrastructure. Point it at your codebase, pick AWS or GCP, and get a complete hosting plan — with Terraform configs, cost estimates, and a full resource inventory — in under a minute.

It works two ways:
- **Web dashboard** — a guided onboarding wizard and full management UI
- **Claude Code + MCP** — infrastructure generation happens in real-time as you build

---

## Quick Demo

```
1. Browse to http://localhost:5173/onboard
2. Choose AWS or GCP
3. Select your project folder
4. Wait ~60 seconds
5. Get: detected resources, hosting plan, cost estimate, and per-resource Terraform HCL
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
6. **Produces** production-ready Terraform HCL for every resource, on demand

---

## Features

### Onboarding Wizard
A 4-step guided flow: pick a provider → browse your code → wait for analysis → get your full hosting plan with resources, costs, and Terraform configs.

### Codebase Analysis
The analyzer extracts 16+ infrastructure file types from your project and feeds them to the LLM for intelligent resource detection. No manual tagging required.

### Per-Resource Terraform Generation
Select any resource from your plan and generate production-ready Terraform HCL for your chosen provider. Copy to clipboard and deploy.

### Live Resource Discovery
Discover what's already running in your cloud account. Infraplane generates targeted CLI commands (`gcloud`, `aws`), executes them in a secure sandbox, and maps the results back to your application. Also supports GCP Cloud Asset Inventory for comprehensive project-wide scans.

### Infrastructure Topology Graphs
Visualize your application's infrastructure as an interactive directed graph — compute nodes, databases, caches, queues, and the edges between them — rendered with React Flow and dagre layout.

### Hosting Plans & Cost Estimates
LLM-generated architecture recommendations with monthly cost breakdowns by resource category (compute, database, storage, networking, etc.).

### Migration Planning
Generate a step-by-step plan to move your application between AWS and GCP, including service mappings, data migration strategies, and new Terraform configurations.

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
│   │ (11 tools) │   │ (18 endpts)│   │ (Claude Sonnet 4.5)  │ │
│   └──────┬─────┘   └──────┬─────┘   └──────────┬───────────┘ │
│          │                │                     │             │
│          └────────────────┼─────────────────────┘             │
│                           │                                   │
│                  ┌────────▼────────┐                          │
│                  │  Service Layer  │                          │
│                  │  (6 services)   │                          │
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
│   └──────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────┐
│                    React Dashboard (Vite)                      │
│  ┌─────────┐ ┌────────┐ ┌──────────┐ ┌────────┐ ┌─────────┐ │
│  │ Onboard │ │  Apps   │ │ App      │ │ Deploy │ │Migration│ │
│  │ Wizard  │ │  List   │ │ Detail   │ │ Board  │ │ Planner │ │
│  └─────────┘ └────────┘ └──────────┘ └────────┘ └─────────┘ │
│  ┌──────────────┐ ┌───────────────┐ ┌────────────────────┐   │
│  │ InfraGraph   │ │ LiveResources │ │ Terraform Viewer   │   │
│  │ (React Flow) │ │ Table         │ │ (per-resource HCL) │   │
│  └──────────────┘ └───────────────┘ └────────────────────┘   │
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
| `POST` | `/applications/{name}/deploy` | Deploy application |
| `GET` | `/applications/{name}/deployments` | List deployments |
| `GET` | `/deployments/{id}` | Get deployment status |
| `GET` | `/health` | Health check |

---

## MCP Tools

11 tools available when running as an MCP server:

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

Each resource maps to concrete services on both providers. Terraform HCL is generated on demand per resource.

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
              ├── Deployments (git commit, status, Terraform plan)
              ├── InfrastructurePlans (hosting or migration, cost estimates)
              └── InfraGraphs (topology: nodes + edges)
```

---

## Project Structure

```
infraplane/
├── cmd/infraplane/main.go              # Entry point: MCP or HTTP mode
├── internal/
│   ├── domain/                         # Core models (zero external deps)
│   │   ├── application.go              # Application entity + status enum
│   │   ├── resource.go                 # Cloud-agnostic resource model
│   │   ├── deployment.go               # Deployment tracking
│   │   ├── plan.go                     # Infrastructure plans + cost estimates
│   │   ├── graph.go                    # Topology graph (nodes + edges)
│   │   ├── live_resource.go            # Live cloud resource tracking
│   │   ├── provider.go                 # Cloud provider enum
│   │   └── errors.go                   # Domain error types
│   ├── llm/                            # LLM integration
│   │   ├── anthropic.go                # Anthropic SDK client (Sonnet 4.5)
│   │   ├── client.go                   # Client interface
│   │   ├── prompts.go                  # Prompt templates (7+ tasks)
│   │   └── mock.go                     # Mock client for tests
│   ├── service/                        # Business logic (6 services)
│   │   ├── application.go              # CRUD + auto-detect + onboarding
│   │   ├── resource.go                 # LLM-powered resource management
│   │   ├── planner.go                  # Hosting + migration planning
│   │   ├── graph.go                    # Topology graph generation
│   │   ├── discovery.go                # Live resource discovery
│   │   └── deployment.go               # Deployment orchestration
│   ├── repository/                     # Data access layer
│   │   ├── interfaces.go               # Repository interfaces
│   │   ├── postgres/                   # PostgreSQL implementations (pgx v5)
│   │   └── mock/                       # In-memory mocks for unit tests
│   ├── analyzer/                       # Codebase analyzer (16+ file types)
│   ├── executor/                       # Secure CLI executor (read-only)
│   ├── cloud/gcp/                      # GCP Cloud Asset Inventory
│   ├── provider/                       # Cloud provider adapters
│   │   ├── aws/                        # AWS adapter
│   │   ├── gcp/                        # GCP adapter
│   │   └── terraform/                  # Terraform HCL generator
│   ├── mcp/                            # MCP server (11 tools)
│   └── api/                            # REST API (18 endpoints, chi router)
├── migrations/                         # 6 PostgreSQL migrations
├── web/                                # React + TypeScript frontend
│   ├── src/pages/                      # 5 pages
│   ├── src/components/                 # 6 components
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

6 migrations manage the schema:

| Migration | Table |
|-----------|-------|
| 001 | `applications` |
| 002 | `resources` (JSONB specs + provider mappings) |
| 003 | `deployments` |
| 004 | `plans` (hosting/migration + cost estimates) |
| 005 | `source_path` column on applications |
| 006 | `graphs` (topology nodes + edges) |

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
