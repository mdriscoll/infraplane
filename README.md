# Infraplane

A cloud infrastructure abstraction platform that lets developers define and manage infrastructure through natural language via [Claude Code](https://claude.com/claude-code) and [MCP](https://modelcontextprotocol.io/) (Model Context Protocol). Infraplane provides a cloud-agnostic resource layer that makes applications easily deployable across AWS and GCP.

## What is Infraplane?

Infraplane sits between your application code and cloud providers. Instead of writing Terraform or CloudFormation by hand, you describe what your application needs in plain English — "I need a PostgreSQL database for user data" — and Infraplane's LLM engine (Claude Opus 4.6) translates that into concrete infrastructure definitions for your chosen cloud provider.

### The Problem

- Developers spend significant time writing and maintaining infrastructure-as-code
- Moving applications between cloud providers requires rewriting IaC from scratch
- Infrastructure decisions are made in isolation from the application code being written
- There's no single pane of glass for tracking what's deployed where

### The Solution

Infraplane provides:
- **Real-time IaC generation** as you build your application with Claude Code
- **Cloud-agnostic resource abstraction** — define once, deploy to AWS or GCP
- **LLM-powered infrastructure intelligence** — recommendations, hosting plans, and migration strategies
- **A deployment dashboard** for visibility and control

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Claude Code                          │
│              (Developer's AI assistant)                   │
└──────────────────────┬──────────────────────────────────┘
                       │ MCP (Model Context Protocol)
                       │ stdio transport
                       ▼
┌─────────────────────────────────────────────────────────┐
│                  Infraplane Server                        │
│                                                          │
│  ┌──────────┐   ┌───────────┐   ┌─────────────────┐    │
│  │ MCP      │   │ REST API  │   │ LLM Engine      │    │
│  │ Server   │   │ (Dashboard│   │ (Claude Opus 4.6)│    │
│  │ (Tools)  │   │  Backend) │   │                  │    │
│  └────┬─────┘   └─────┬─────┘   └────────┬────────┘    │
│       │               │                   │              │
│       └───────────────┼───────────────────┘              │
│                       │                                  │
│              ┌────────▼────────┐                         │
│              │  Service Layer  │                         │
│              │  (Business      │                         │
│              │   Logic)        │                         │
│              └────────┬────────┘                         │
│                       │                                  │
│         ┌─────────────┼─────────────┐                    │
│         ▼             ▼             ▼                    │
│  ┌────────────┐ ┌──────────┐ ┌──────────┐              │
│  │ PostgreSQL │ │ AWS      │ │ GCP      │              │
│  │ Repository │ │ Adapter  │ │ Adapter  │              │
│  └────────────┘ └──────────┘ └──────────┘              │
└─────────────────────────────────────────────────────────┘
```

### Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.26+ |
| Database | PostgreSQL 16 |
| Frontend | React + TypeScript (Vite) |
| LLM | Claude Opus 4.6 via Anthropic API |
| MCP SDK | [mcp-go](https://github.com/mark3labs/mcp-go) |
| DB Driver | [pgx](https://github.com/jackc/pgx) v5 |
| HTTP Router | [chi](https://github.com/go-chi/chi) v5 |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) v4 |
| Integration Tests | [testcontainers-go](https://github.com/testcontainers/testcontainers-go) |
| Container Runtime | [Colima](https://github.com/abiosoft/colima) (Docker-compatible) |

## User Journeys

### 1. Real-Time Infrastructure Generation

While you build your application with Claude Code, Infraplane defines infrastructure on-the-fly:

```
You: "I need a PostgreSQL database for user data"
Claude Code → MCP → Infraplane:
  → LLM analyzes the request
  → Creates abstract Resource(kind=database, spec={engine: postgres})
  → Generates provider mappings:
      AWS → RDS (db.t3.micro, PostgreSQL 16)
      GCP → Cloud SQL (db-f1-micro, PostgreSQL 16)
  → Returns confirmation with both provider options
```

### 2. Application Registry

List all registered applications with deployment status, provider, and latest git commit:

```
$ infraplane list-applications

NAME          PROVIDER  STATUS      LATEST DEPLOY  BRANCH
my-api        aws       deployed    abc123f        main
frontend-app  gcp       provisioned -              -
new-service   aws       draft       -              -
```

### 3. Deployment Dashboard

A React-based web dashboard where you can:
- View all registered applications and their resources
- Deploy the latest commit from `main` or any branch
- View deployment history with Terraform plans
- Monitor deployment status in real-time

### 4. Cloud Migration Planning

Request a migration plan to move your application between providers:

```
You: "Generate a plan to migrate my-api from AWS to GCP"
Infraplane:
  → Analyzes all resources (RDS → Cloud SQL, S3 → Cloud Storage, etc.)
  → Generates step-by-step migration plan
  → Estimates cost differences
  → Produces new Terraform configurations for the target provider
```

### 5. Hosting Recommendations

As you build, Infraplane analyzes your application's resource requirements and suggests optimal hosting strategies:

```
You: "What's the best way to deploy this application?"
Infraplane:
  → Analyzes your resources (database, cache, compute, CDN)
  → Recommends architecture (ECS Fargate vs. EKS vs. Lambda)
  → Provides estimated monthly costs
  → Generates complete Terraform configuration
```

## Project Structure

```
infraplane/
├── cmd/
│   └── infraplane/
│       └── main.go                 # Entry point: MCP server + REST API
├── internal/
│   ├── domain/                     # Core domain models (zero external deps)
│   │   ├── application.go          # Application entity
│   │   ├── resource.go             # Cloud-agnostic resource abstraction
│   │   ├── deployment.go           # Deployment tracking
│   │   ├── plan.go                 # Infrastructure & migration plans
│   │   ├── provider.go             # Cloud provider enum (AWS, GCP)
│   │   ├── errors.go               # Domain error types
│   │   └── domain_test.go          # 29 unit tests for all domain logic
│   ├── llm/                        # LLM integration (Claude Opus 4.6)
│   │   ├── client.go               # Anthropic API client wrapper
│   │   ├── prompts.go              # Prompt templates for reasoning tasks
│   │   └── prompts/                # Long-form prompt templates (.txt)
│   ├── service/                    # Business logic layer
│   │   ├── application.go          # App CRUD operations
│   │   ├── resource.go             # Resource management + LLM analysis
│   │   ├── deployment.go           # Deployment orchestration
│   │   └── planner.go              # LLM-powered hosting & migration planning
│   ├── repository/                 # Data access layer
│   │   ├── interfaces.go           # Repository interfaces (ports)
│   │   ├── postgres/               # PostgreSQL implementations
│   │   │   ├── db.go               # Connection pool setup
│   │   │   ├── application.go      # Application CRUD
│   │   │   ├── resource.go         # Resource CRUD with JSONB
│   │   │   ├── deployment.go       # Deployment tracking
│   │   │   ├── plan.go             # Plan storage
│   │   │   ├── testhelper_test.go   # Testcontainers setup
│   │   │   └── *_test.go           # 25 integration tests
│   │   └── mock/                   # In-memory mocks for unit testing
│   │       ├── repositories.go     # All 4 mock repos
│   │       └── repositories_test.go # 4 CRUD test suites
│   ├── provider/                   # Cloud provider adapters
│   │   ├── adapter.go              # CloudProviderAdapter interface
│   │   ├── aws/                    # AWS implementation
│   │   ├── gcp/                    # GCP implementation
│   │   └── terraform/              # Terraform HCL generation
│   ├── mcp/                        # MCP server + tool definitions
│   │   ├── server.go               # MCP server setup
│   │   └── tools.go                # Tool handler implementations
│   └── api/                        # REST API for dashboard
│       ├── router.go               # HTTP router
│       ├── handlers.go             # REST handlers
│       └── middleware.go           # Auth, CORS, logging
├── migrations/                     # PostgreSQL schema migrations
│   ├── 001_create_applications.*   # Applications table
│   ├── 002_create_resources.*      # Resources table (JSONB specs)
│   ├── 003_create_deployments.*    # Deployment tracking
│   └── 004_create_plans.*          # Infrastructure plans
├── web/                            # React + TypeScript frontend
│   └── src/
│       ├── pages/                  # Dashboard pages
│       ├── components/             # UI components
│       └── api/                    # API client
├── docker-compose.yml              # PostgreSQL for local dev
├── Makefile                        # Build, test, and dev commands
├── go.mod
└── .env.example                    # Environment variable template
```

## Core Domain Model

### The Resource Abstraction

The key innovation is the **cloud-agnostic resource model**. When a developer says "I need a database," Infraplane creates an abstract `Resource` that maps to concrete cloud services:

```go
// A developer says: "I need a PostgreSQL database for user data"
// Infraplane creates:
Resource{
    Kind: "database",
    Name: "user-db",
    Spec: {"engine": "postgres", "version": "16"},
    ProviderMappings: {
        "aws": {
            ServiceName:  "RDS",
            Config:       {"instance_class": "db.t3.micro"},
            TerraformHCL: "resource \"aws_db_instance\" ...",
        },
        "gcp": {
            ServiceName:  "Cloud SQL",
            Config:       {"tier": "db-f1-micro"},
            TerraformHCL: "resource \"google_sql_database_instance\" ...",
        },
    },
}
```

### Supported Resource Kinds

| Kind | AWS Service | GCP Service |
|------|-----------|-------------|
| `database` | RDS, Aurora, DynamoDB | Cloud SQL, Firestore, Spanner |
| `compute` | ECS, EKS, Lambda, EC2 | Cloud Run, GKE, Cloud Functions |
| `storage` | S3 | Cloud Storage |
| `cache` | ElastiCache | Memorystore |
| `queue` | SQS, SNS | Pub/Sub, Cloud Tasks |
| `cdn` | CloudFront | Cloud CDN |
| `network` | VPC, ALB | VPC, Cloud Load Balancing |

### Entity Relationships

```
Application (1) ──── (N) Resource
     │                      │
     │                      └── ProviderMappings (AWS, GCP configs + Terraform)
     │
     ├──── (N) Deployment (tracks git commit, status, Terraform plan)
     │
     └──── (N) InfrastructurePlan (hosting or migration, LLM-generated)
```

## MCP Tools

Infraplane exposes the following tools via MCP for Claude Code to call:

| Tool | Description | LLM-Powered |
|------|-------------|:-----------:|
| `register_application` | Register a new app with name, git repo, preferred provider | |
| `add_resource` | Describe a resource need in natural language; LLM maps to cloud services | Yes |
| `remove_resource` | Remove a resource from an application | |
| `list_applications` | List all registered applications with status | |
| `get_application` | Get detailed app info including all resources | |
| `get_hosting_plan` | LLM analyzes app + resources, recommends hosting strategy | Yes |
| `plan_migration` | LLM generates migration plan between cloud providers | Yes |
| `deploy` | Trigger deployment to a provider from a git branch/commit | |
| `get_deployment_status` | Check deployment progress and history | |

## LLM Integration

Infraplane uses Claude Opus 4.6 as its reasoning engine for all "smart" operations:

### How It Works

1. **Resource Analysis**: When a developer describes an infrastructure need, the LLM interprets the natural language, identifies the resource kind, generates a specification, and maps it to concrete cloud services with Terraform.

2. **Hosting Plans**: The LLM analyzes all resources for an application and recommends an optimal deployment architecture, considering cost, performance, reliability, and scalability.

3. **Migration Plans**: Given a source and target provider, the LLM generates a step-by-step migration plan with service mappings, data migration strategies, and new Terraform configurations.

4. **Terraform Generation**: All IaC is generated by the LLM as Terraform HCL, ensuring it follows best practices for the target provider.

### Configuration

Set the `ANTHROPIC_API_KEY` environment variable:

```bash
cp .env.example .env
# Edit .env and add your Anthropic API key
```

## Getting Started

### Prerequisites

- **Go 1.26+**: `brew install go`
- **Colima** (Docker runtime): `brew install colima docker`
- **Node.js 20+** (for frontend): `brew install node`
- **Anthropic API Key**: Required for LLM-powered features

### Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/matthewdriscoll/infraplane.git
cd infraplane

# 2. Copy environment template
cp .env.example .env
# Edit .env with your ANTHROPIC_API_KEY

# 3. Start Colima (Docker runtime)
colima start --cpu 2 --memory 4

# 4. Start PostgreSQL
docker compose up -d postgres

# 5. Install Go dependencies
make deps

# 6. Run database migrations
make migrate

# 7. Start the server
make dev
```

### Connecting to Claude Code

Once the MCP server is running, configure Claude Code to connect:

```json
// In your Claude Code MCP settings
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

## Development

### Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the binary to `bin/infraplane` |
| `make dev` | Start PostgreSQL + run the server |
| `make test` | Run unit tests only (fast, no Docker needed) |
| `make test-integration` | Run integration tests (requires Colima/Docker) |
| `make test-all` | Run all tests |
| `make migrate` | Run database migrations |
| `make migrate-down` | Rollback last migration |
| `make web` | Start frontend dev server |
| `make fmt` | Format and vet Go code |
| `make deps` | Install/tidy Go dependencies |
| `make clean` | Remove build artifacts |

### Running Tests

```bash
# Unit tests only (fast, no Docker needed)
make test

# Integration tests (spins up PostgreSQL via testcontainers)
make test-integration

# All tests
make test-all
```

### Test Architecture

The project follows strict TDD with two test layers:

**Unit Tests** (`-short` flag skips integration):
- Domain model validation (table-driven tests)
- Mock repository CRUD operations
- Service layer logic (with mock repos + mock LLM client)
- MCP tool handlers (with mock services)

**Integration Tests** (`-run Integration`):
- PostgreSQL repositories tested against real database via [testcontainers-go](https://github.com/testcontainers/testcontainers-go)
- Each test suite spins up a fresh Postgres container, runs migrations, executes tests, and tears down
- MCP end-to-end tests (in-process client/server with real service + test DB)

### Database Migrations

Migrations live in `migrations/` and are managed by [golang-migrate](https://github.com/golang-migrate/migrate):

```
migrations/
├── 001_create_applications.up.sql    # Applications table
├── 001_create_applications.down.sql
├── 002_create_resources.up.sql       # Resources with JSONB specs
├── 002_create_resources.down.sql
├── 003_create_deployments.up.sql     # Deployment tracking
├── 003_create_deployments.down.sql
├── 004_create_plans.up.sql           # Infrastructure plans
└── 004_create_plans.down.sql
```

Key schema decisions:
- **JSONB columns** for resource specs and provider mappings — flexible schema for diverse resource types
- **UUID primary keys** — globally unique, safe for distributed systems
- **Cascading deletes** on resources when an application is deleted
- **Indexed foreign keys** for efficient lookups

### Docker / Colima Setup

This project uses Colima as a lightweight Docker alternative on macOS:

```bash
# Install
brew install colima docker

# Start (2 CPUs, 4GB RAM)
colima start --cpu 2 --memory 4

# Verify
docker info
```

Testcontainers require the `TESTCONTAINERS_RYUK_DISABLED=true` environment variable when using Colima (the Makefile handles this automatically).

## Implementation Roadmap

### Phase 1: Foundation (Complete)
- [x] Project scaffolding (Go module, Makefile, Docker Compose)
- [x] Domain models (Application, Resource, Deployment, InfrastructurePlan)
- [x] Cloud-agnostic resource abstraction with provider mappings
- [x] PostgreSQL migrations for all tables
- [x] Repository interfaces + PostgreSQL implementations
- [x] Mock repositories for unit testing
- [x] Integration tests with testcontainers-go (25 tests)
- [x] Unit tests for domain models (29 tests)

### Phase 2: LLM Integration
- [ ] Anthropic API client wrapper (Claude Opus 4.6)
- [ ] Prompt templates for resource analysis, hosting plans, migration plans
- [ ] Structured output parsing (JSON from LLM responses)
- [ ] Unit tests with mocked HTTP responses
- [ ] LLM client interface + mock for service testing

### Phase 3: Services + MCP Server
- [ ] Application service (CRUD operations)
- [ ] Resource service (add/remove + LLM-powered analysis)
- [ ] Planner service (hosting and migration plans)
- [ ] MCP server with all 9 tools registered
- [ ] MCP integration tests (in-process client/server)

### Phase 4: REST API
- [ ] REST endpoints mirroring MCP tools
- [ ] CORS, logging, error handling middleware
- [ ] API integration tests

### Phase 5: Deployment Service
- [ ] Terraform generation via LLM
- [ ] AWS adapter (credential validation, Terraform apply)
- [ ] GCP adapter (credential validation, Terraform apply)
- [ ] Deployment orchestration service
- [ ] Provider adapter tests

### Phase 6: Frontend
- [ ] React app scaffolding (Vite, Router, TanStack Query, Tailwind)
- [ ] Application list and detail pages
- [ ] Deployment dashboard with branch selection
- [ ] Migration planner page
- [ ] Frontend component tests

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `ANTHROPIC_API_KEY` | Yes | - | Anthropic API key for Claude Opus 4.6 |
| `PORT` | No | `8080` | REST API server port |
| `MCP_MODE` | No | `stdio` | MCP transport mode |

### Example `.env`

```bash
DATABASE_URL=postgres://infraplane:infraplane@localhost:5432/infraplane?sslmode=disable
ANTHROPIC_API_KEY=sk-ant-xxxxx
PORT=8080
MCP_MODE=stdio
```

## Contributing

This project uses test-driven development. Every feature must have:
- **Unit tests** for business logic (using mock repositories and mock LLM client)
- **Integration tests** for cross-domain flows (using testcontainers-go for PostgreSQL)

Before submitting a PR:
```bash
make fmt          # Format and vet
make test         # Unit tests pass
make test-integration  # Integration tests pass
```

## License

TBD
