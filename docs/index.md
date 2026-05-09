# dago Project Index

> This file is updated automatically after each sprint.
> It is the navigable map of the project — Claude Code reads it first
> to orient itself before diving into specific files.

## Architectural Decision Records (ADRs)

| ADR | Title | Status |
|-----|-------|--------|
| [001](docs/adr/ADR-001-arquitectura-hexagonal.md) | Hexagonal Architecture | Accepted (revised) |
| [002](docs/adr/ADR-002-tdd.md) | TDD (Red-Green-Refactor) | Accepted |
| [003](docs/adr/ADR-003-clean-code.md) | Clean Code | Accepted |
| [004](docs/adr/ADR-004-go-lenguaje.md) | Go + style guides | Accepted |
| [005](docs/adr/ADR-005-sdlc-github-flow.md) | GitHub Flow + semver + changelog | Accepted (revised) |
| [006](docs/adr/ADR-006-gin-framework-http.md) | Gin (5 HTTP services) | Accepted (revised) |
| [007](docs/adr/ADR-007-postgresql.md) | PostgreSQL + Ent + Atlas | Accepted (revised) |
| [008](docs/adr/ADR-008-valkey.md) | Valkey (events, cache, sessions) | Accepted (revised) |
| [009](docs/adr/ADR-009-react-typescript-frontend.md) | React 19 + TypeScript + Vite | Accepted (revised) |
| [010](docs/adr/ADR-010-api-versioning-openapi.md) | URL path versioning + OpenAPI 3.1 | Accepted |
| [011](docs/adr/ADR-011-asyncapi-eventos-valkey.md) | AsyncAPI 3.0 + Event-Carried State | Accepted (revised) |
| [012](docs/adr/ADR-012-oauth21-autenticacion.md) | OAuth 2.1 + tag-based ABAC | Accepted (revised) |
| [013](docs/adr/ADR-013-monorepo.md) | Monorepo (single go.mod, 8 services) | Accepted (revised) |
| [014](docs/adr/ADR-014-comunicacion-eventos.md) | Communication: events + HTTP | Accepted (revised) |
| [015](docs/adr/ADR-015-memoria-agentes.md) | Agent memory (3 layers + dreaming) | Accepted |
| [016](docs/adr/ADR-016-patrones-orquestacion.md) | Orchestration patterns (flow + node) | Accepted |
| [017](docs/adr/ADR-017-paquetes.md) | Packages as distribution unit | Accepted |
| [018](docs/adr/ADR-018-agui-a2ui.md) | AG-UI + A2UI (UI protocols) | Accepted |
| [019](docs/adr/ADR-019-design-system-microfrontales.md) | shadcn/ui + Module Federation | Accepted |
| [020](docs/adr/ADR-020-sprints-trazabilidad.md) | Reduced sprints + full traceability | Accepted |

## Sprints

| Sprint | Title | Status |
|--------|-------|--------|
| [SPRINT-001](sprints/SPRINT-001-bootstrap-monorepo.md) | Go monorepo bootstrap | completed |
| [SPRINT-002](sprints/SPRINT-002-ent-schemas-graph-node-execution.md) | Ent schemas: Graph, Node, Execution | completed |
| [SPRINT-003](sprints/SPRINT-003-api-rest-orchestrator.md) | Orchestrator REST API: graph CRUD + executions | completed |
| [SPRINT-004](sprints/SPRINT-004-auth-server-jwt-basico.md) | Auth-server: argon2id local login, JWT RS256, JWKS, middleware | completed |
| [SPRINT-005](sprints/SPRINT-005-dashboard-bootstrap-pkce.md) | Dashboard: React 19 + shadcn/ui + PKCE + OpenAPI types + GraphsPage | completed |
| [SPRINT-006](sprints/SPRINT-006-dashboard-feature-grafos.md) | Dashboard: Graph feature — list, detail, create, edit | completed |
| [SPRINT-007](sprints/SPRINT-007-eventbus-valkey-adapter.md) | Event Bus adapter — Valkey Streams + consumer groups + DLQ | completed |
| [SPRINT-008](sprints/SPRINT-008-llm-adapter-anthropic.md) | LLM adapter — LLMClient port + Anthropic + Ollama/Mixtral + Fake | completed |
| [SPRINT-009](sprints/SPRINT-009-executor-llm-call.md) | Executor: llm_call pattern handler, node.execute.requested consumer | completed |
| [SPRINT-010](sprints/SPRINT-010-orchestrator-state-machine.md) | Orchestrator: state machine, graph validation, publish events, transition, complete | completed |
| [SPRINT-011](sprints/SPRINT-011-docker-compose.md) | docker-compose full-stack containerization: Dockerfiles, profiles, health endpoints, runbook | completed |

## Specs (sources of truth)

| Spec | Location | Status | Governs |
|------|----------|--------|---------|
| OpenAPI 3.1 | [specs/openapi.yaml](specs/openapi.yaml) | base structure | Orchestrator REST API |
| ↳ schemas/graph.yaml | [specs/schemas/graph.yaml](specs/schemas/graph.yaml) | implemented (SPRINT-003) | GraphInput, GraphResponse, GraphListResponse |
| ↳ schemas/execution.yaml | [specs/schemas/execution.yaml](specs/schemas/execution.yaml) | implemented (SPRINT-003) | ExecutionInput, ExecutionResponse |
| ↳ paths/graphs.yaml | [specs/paths/graphs.yaml](specs/paths/graphs.yaml) | implemented (SPRINT-003) | Graph CRUD (6 endpoints) |
| ↳ paths/executions.yaml | [specs/paths/executions.yaml](specs/paths/executions.yaml) | implemented (SPRINT-003) | Execution start |
| ↳ schemas/auth.yaml | [specs/schemas/auth.yaml](specs/schemas/auth.yaml) | implemented (SPRINT-005) | RegisterInput, LoginInput, TokenResponse, UserResponse, TokenRequest, AuthorizeParams |
| ↳ paths/auth.yaml | [specs/paths/auth.yaml](specs/paths/auth.yaml) | implemented (SPRINT-005) | Register, Login, JWKS, GET/POST /authorize, POST /token (6 endpoints) |
| AsyncAPI 3.0 | [specs/asyncapi.yaml](specs/asyncapi.yaml) | implemented (SPRINT-007, SPRINT-009, SPRINT-010) | Valkey Streams: 10 channels + 5 orchestrator ops (node.execute.requested publish, node.executed/failed consume, graph.completed/failed publish) |
| Graph Schema | [specs/patterns/graph.json](specs/patterns/graph.json) | implemented | Graph structure |
| Edge patterns (5) | [specs/patterns/edges/](specs/patterns/edges/) | implemented | sequential, conditional, parallel, loop, interrupt |
| Node patterns (7) | [specs/patterns/nodes/](specs/patterns/nodes/) | implemented | llm_call, tool_use, react, reflection, router, guardrail, subgraph |

## Services

| Service | Type | Responsibility | Location | Status |
|---------|------|----------------|----------|--------|
| orchestrator | Events + HTTP | Core: graphs, state, coordination, API, WebSocket AG-UI; state machine, ValidateGraph, NodeResultConsumer | services/orchestrator/ | implemented (SPRINT-003, SPRINT-010) |
| executor | Events | Worker: agentic node patterns (llm_call implemented) | services/executor/ | partial (SPRINT-009) |
| router | Events | Worker: deterministic/llm/hybrid routing | services/router/ | planned |
| planner | Events | NL → graph | services/planner/ | planned |
| auth-server | HTTP | OAuth 2.1: local login, JWT RS256, JWKS, PKCE authorize/token | services/auth-server/ | implemented (SPRINT-005) |
| catalog | HTTP | Package catalogue, versioning | services/catalog/ | planned |
| mcp-registry | HTTP | MCP server registry + broker | services/mcp-registry/ | planned |
| agent-registry | HTTP | A2A Agent Cards, discovery | services/agent-registry/ | planned |
| dashboard | Frontend | React 19 + shadcn/ui + PKCE + graphs feature (list, detail, create, edit, archive) | dashboard/ | implemented (SPRINT-006) |

## Documentation

| Document | Location | Content |
|----------|----------|---------|
| CLAUDE.md | [CLAUDE.md](CLAUDE.md) | Instructions for Claude Code |
| Skills & Agents | [docs/skills-catalog.md](docs/skills-catalog.md) | 22 skills + 6 agents |
| Sprints | [docs/sprints/](docs/sprints/) | Sprint documents (traceability) |
| Scenarios view | [docs/views/scenarios/](docs/views/scenarios/) | +1: End-to-end use cases |
| Logical view | [docs/views/logical/](docs/views/logical/) | Components, domain, patterns |
| Process view | [docs/views/process/](docs/views/process/) | Flows, concurrency, events |
| Development view | [docs/views/development/](docs/views/development/) | Code, build, standards |
| Physical view | [docs/views/physical/](docs/views/physical/) | Infrastructure, deployment |
| Changelog | [CHANGELOG.md](CHANGELOG.md) | Release history (generated) |
| Log | [docs/log.md](docs/log.md) | Chronological operation history |

## Protocols

| Protocol | Relationship | Use in dago |
|----------|-------------|-------------|
| MCP | Agent ↔ Tools | executor → mcp-registry → tools |
| A2A | Agent ↔ Agent | agent-registry → Agent Cards |
| AG-UI | Agent ↔ User | orchestrator → dashboard (WebSocket) |
| A2UI | Agent → UI | node → dashboard (declarative JSON) |
| OAuth 2.1 | Auth | auth-server → all services (JWT/JWKS) |

## Adapters

| Adapter | Port | Implementation | Status | Sprint |
|---------|------|----------------|--------|--------|
| Graph/Execution Storage | libs/ports/storage.go | adapters/storage/graph_repo.go | implemented | [SPRINT-003](sprints/SPRINT-003-api-rest-orchestrator.md) |
| Auth (argon2id + JWT + JWKS) | libs/ports/auth.go | adapters/auth/ | implemented | [SPRINT-004](sprints/SPRINT-004-auth-server-jwt-basico.md) |
| User Storage (Ent) | libs/ports/auth.go | adapters/auth/ent_user_repo.go | implemented | [SPRINT-004](sprints/SPRINT-004-auth-server-jwt-basico.md) |
| Event Bus Valkey | libs/ports/eventbus.go | adapters/eventbus/valkey/ | implemented | [SPRINT-007](sprints/SPRINT-007-eventbus-valkey-adapter.md) |
| LLM Anthropic | libs/ports/llm.go | adapters/llm/anthropic/ | implemented | [SPRINT-008](sprints/SPRINT-008-llm-adapter-anthropic.md) |
| LLM Ollama (Mixtral) | libs/ports/llm.go | adapters/llm/ollama/ | implemented | [SPRINT-008](sprints/SPRINT-008-llm-adapter-anthropic.md) |
| LLM Fake | libs/ports/llm.go | adapters/llm/fake/ | implemented | [SPRINT-008](sprints/SPRINT-008-llm-adapter-anthropic.md) |

## Dashboard (SPRINT-006)

| Directory | Description |
|-----------|-------------|
| `dashboard/src/api/` | OpenAPI-generated types (types.gen.ts) + API client |
| `dashboard/src/auth/` | PKCE module, AuthProvider, ProtectedRoute, AuthCallback |
| `dashboard/src/components/ui/` | shadcn/ui base components (Button, Badge, Table, Form, etc.) |
| `dashboard/src/features/graphs/` | Graphs feature: pages, components, hooks, schemas, templates |
| `dashboard/src/layouts/` | AppLayout (sidebar + header + dark mode) |
| `dashboard/src/pages/` | NotFoundPage |

### Graphs feature (`features/graphs/`)

| Sub-directory | Contents |
|---------------|----------|
| `pages/` | GraphsPage (list+filter), GraphDetailPage (tabs), GraphCreatePage, GraphEditPage |
| `components/` | GraphTable, GraphForm, GraphStatusBadge, NodePatternBadge, EdgeTypeBadge, GraphDefinitionViewer, GraphArchiveDialog |
| `hooks/` | useGraphs, useGraph, useCreateGraph, useUpdateGraph, useArchiveGraph |
| `schemas/` | graphFormSchema (Zod), formToGraphInput |
| `lib/` | definition-templates (4 presets: empty, llm_call, react_agent, router_handlers) |

### Environment variables (dashboard/.env.development)

| Variable | Value | Purpose |
|----------|-------|---------|
| `VITE_API_URL` | `http://localhost:8080` | Orchestrator REST API base URL |
| `VITE_AUTH_URL` | `http://localhost:8081` | Auth-server base URL |
| `VITE_AUTH_CLIENT_ID` | `dago-dashboard` | OAuth client identifier |
| `VITE_AUTH_REDIRECT_URI` | `http://localhost:5173/auth/callback` | PKCE callback URL |

## Development infrastructure

| Artifact | Location | Description |
|----------|----------|-------------|
| Makefile | `Makefile` | Unified build: `make ci`, `make bootstrap`, `make build-all` |
| Docker Compose | `docker-compose.yml` | Full stack: postgres + valkey + 8 Go services + dashboard (4 profiles) — SPRINT-011 |
| Dockerfile.service | `Dockerfile.service` | Shared multi-stage Go build (SERVICE build arg) — SPRINT-011 |
| Dashboard Dockerfile | `dashboard/Dockerfile` | Multi-stage React 19 build + nginx — SPRINT-011 |
| Linter | `.golangci.yml` | goimports, errcheck, funlen, wrapcheck (ADR-003, ADR-004) |
| Atlas config | `atlas.hcl` | Ent → PostgreSQL migrations (ADR-007) |
| Env vars | `.env.example` | All service env vars for local development — SPRINT-011 |
| Deploy runbook | `docs/deploy/docker-compose-runbook.md` | Prereqs, quick-start, profiles, secrets, troubleshooting — SPRINT-011 |

_Artifacts created in SPRINT-001. Status: completed (2026-04-30)._

## Go structure (created in SPRINT-001)

| Directory | Go package | Description |
|-----------|-----------|-------------|
| `libs/domain/` | `domain` | Domain types and business logic |
| `libs/ports/` | `ports` | Port interfaces (hexagonal architecture) |
| `libs/schemas/` | `schemas` | JSON Schema validation helpers |
| `libs/utils/` | `utils` | Shared utilities |
| `adapters/storage/` | `storage` | Database adapter (Ent) |
| `adapters/eventbus/` | `eventbus` | Event bus adapter (Valkey Streams) |
| `adapters/auth/` | `auth` | Authentication adapter |
| `adapters/llm/` | `llm` | LLM provider adapters |
| `adapters/metrics/` | `metrics` | Metrics adapter |
| `services/*/cmd/main.go` | `main` | Entry points for the 8 services |

## Domain — Ent Schemas

_Ent schemas implemented in SPRINT-002. Ent client generated. Migration applied to PostgreSQL._

| Entity | Ent Schema | Status | Sprint |
|--------|------------|--------|--------|
| Graph | ent/schema/graph.go | implemented | [SPRINT-002](sprints/SPRINT-002-ent-schemas-graph-node-execution.md) |
| Node | ent/schema/node.go | implemented | [SPRINT-002](sprints/SPRINT-002-ent-schemas-graph-node-execution.md) |
| Execution | ent/schema/execution.go | implemented | [SPRINT-002](sprints/SPRINT-002-ent-schemas-graph-node-execution.md) |
| Episode | ent/schema/episode.go | pending | SPRINT-015 (memory) |
| SemanticFact | ent/schema/semantic_fact.go | pending | SPRINT-015 (memory, pgvector) |
| User | ent/schema/user.go | implemented | [SPRINT-004](sprints/SPRINT-004-auth-server-jwt-basico.md) |
| OrgUnit | ent/schema/org_unit.go | implemented | [SPRINT-004](sprints/SPRINT-004-auth-server-jwt-basico.md) |
| Package | ent/schema/package.go | pending | SPRINT-catalog |
| McpServer | ent/schema/mcp_server.go | pending | SPRINT-mcp |
| AgentCard | ent/schema/agent_card.go | pending | SPRINT-a2a |
