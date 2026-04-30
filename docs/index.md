# Índice del Proyecto dago

> Este fichero se actualiza automáticamente tras cada sprint.
> Es el mapa navegable del proyecto — Claude Code lo lee primero
> para orientarse antes de profundizar en ficheros específicos.

## Decisiones arquitectónicas (ADRs)

| ADR | Título | Estado |
|-----|--------|--------|
| [001](docs/adr/ADR-001-arquitectura-hexagonal.md) | Arquitectura Hexagonal | Aceptado (revisado) |
| [002](docs/adr/ADR-002-tdd.md) | TDD (Red-Green-Refactor) | Aceptado |
| [003](docs/adr/ADR-003-clean-code.md) | Clean Code | Aceptado |
| [004](docs/adr/ADR-004-go-lenguaje.md) | Go + guías de estilo | Aceptado |
| [005](docs/adr/ADR-005-sdlc-github-flow.md) | GitHub Flow + semver + changelog | Aceptado (revisado) |
| [006](docs/adr/ADR-006-gin-framework-http.md) | Gin (5 servicios HTTP) | Aceptado (revisado) |
| [007](docs/adr/ADR-007-postgresql.md) | PostgreSQL + Ent + Atlas | Aceptado (revisado) |
| [008](docs/adr/ADR-008-valkey.md) | Valkey (eventos, caché, sesiones) | Aceptado (revisado) |
| [009](docs/adr/ADR-009-react-typescript-frontend.md) | React 19 + TypeScript + Vite | Aceptado (revisado) |
| [010](docs/adr/ADR-010-api-versioning-openapi.md) | URL path versioning + OpenAPI 3.1 | Aceptado |
| [011](docs/adr/ADR-011-asyncapi-eventos-valkey.md) | AsyncAPI 3.0 + Event-Carried State | Aceptado (revisado) |
| [012](docs/adr/ADR-012-oauth21-autenticacion.md) | OAuth 2.1 + ABAC por etiquetas | Aceptado (revisado) |
| [013](docs/adr/ADR-013-monorepo.md) | Monorepo (un solo go.mod, 8 servicios) | Aceptado (revisado) |
| [014](docs/adr/ADR-014-comunicacion-eventos.md) | Comunicación: eventos + HTTP | Aceptado (revisado) |
| [015](docs/adr/ADR-015-memoria-agentes.md) | Memoria de agentes (3 capas + dreaming) | Aceptado |
| [016](docs/adr/ADR-016-patrones-orquestacion.md) | Patrones de orquestación (flujo + nodo) | Aceptado |
| [017](docs/adr/ADR-017-paquetes.md) | Paquetes como unidad de distribución | Aceptado |
| [018](docs/adr/ADR-018-agui-a2ui.md) | AG-UI + A2UI (protocolos de UI) | Aceptado |
| [019](docs/adr/ADR-019-design-system-microfrontales.md) | shadcn/ui + Module Federation | Aceptado |
| [020](docs/adr/ADR-020-sprints-trazabilidad.md) | Sprints reducidos + trazabilidad | Aceptado |

## Sprints

| Sprint | Título | Estado |
|--------|--------|--------|
| [SPRINT-001](sprints/SPRINT-001-bootstrap-monorepo.md) | Bootstrap del monorepo Go | completado |
| [SPRINT-002](sprints/SPRINT-002-ent-schemas-graph-node-execution.md) | Schemas Ent: Graph, Node, Execution | planificado |
| [SPRINT-003](sprints/SPRINT-003-api-rest-orchestrator.md) | API REST orchestrator: CRUD grafos + ejecuciones | planificado |
| [SPRINT-004](sprints/SPRINT-004-auth-server-jwt-basico.md) | Auth-server: login local argon2id, JWT RS256, JWKS, middleware | planificado |
| [SPRINT-005](sprints/SPRINT-005-dashboard-bootstrap-pkce.md) | Dashboard: React 19 + shadcn/ui + PKCE + tipos OpenAPI + GraphsPage | planificado |
| [SPRINT-006](sprints/SPRINT-006-dashboard-feature-grafos.md) | Dashboard: Feature grafos — listado, detalle, creación y edición | planificado |
| [SPRINT-007](sprints/SPRINT-007-eventbus-valkey-adapter.md) | Adaptador Event Bus — Valkey Streams + consumer groups + DLQ | planificado |
| [SPRINT-008](sprints/SPRINT-008-llm-adapter-anthropic.md) | Adaptador LLM — Puerto LLMClient + Anthropic + Ollama/Mixtral + Fake | planificado |
| [SPRINT-009](sprints/SPRINT-009-executor-llm-call.md) | Executor: handler patrón llm_call, consumer node.execute.requested | planificado |
| [SPRINT-010](sprints/SPRINT-010-orchestrator-state-machine.md) | Orchestrator: state machine, validar grafo, publicar eventos, transicionar, completar | planificado |

## Specs (fuentes de verdad)

| Spec | Ubicación | Estado | Gobierna |
|------|-----------|--------|---------|
| OpenAPI 3.1 | [specs/openapi.yaml](specs/openapi.yaml) | estructura base | API REST del orchestrator |
| ↳ paths/graphs.yaml | [specs/paths/graphs.yaml](specs/paths/graphs.yaml) | planificado (SPRINT-003) | CRUD grafos |
| ↳ paths/executions.yaml | [specs/paths/executions.yaml](specs/paths/executions.yaml) | planificado (SPRINT-003) | inicio ejecuciones |
| AsyncAPI 3.0 | [specs/asyncapi.yaml](specs/asyncapi.yaml) | estructura base | Eventos Valkey (13 tipos) |
| Graph Schema | [specs/patterns/graph.json](specs/patterns/graph.json) | implementado | Estructura de grafos |
| Edge patterns (5) | [specs/patterns/edges/](specs/patterns/edges/) | implementado | sequential, conditional, parallel, loop, interrupt |
| Node patterns (7) | [specs/patterns/nodes/](specs/patterns/nodes/) | implementado | llm_call, tool_use, react, reflection, router, guardrail, subgraph |

## Servicios

| Servicio | Tipo | Responsabilidad | Ubicación |
|----------|------|-----------------|-----------|
| orchestrator | Eventos + HTTP | Core: grafos, estado, coordinación, API, WebSocket AG-UI | services/orchestrator/ |
| executor | Eventos | Worker: patrones de nodo agénticos | services/executor/ |
| router | Eventos | Worker: routing deterministic/llm/hybrid | services/router/ |
| planner | Eventos | NL → grafo | services/planner/ |
| auth-server | HTTP | OAuth 2.1, Identity Broker, ABAC | services/auth-server/ |
| catalog | HTTP | Catálogo de paquetes, versionado | services/catalog/ |
| mcp-registry | HTTP | Registry + broker MCP | services/mcp-registry/ |
| agent-registry | HTTP | Agent Cards A2A, discovery | services/agent-registry/ |
| dashboard | Frontend | React 19 + shadcn/ui + Module Federation | dashboard/ |

## Documentación

| Documento | Ubicación | Contenido |
|-----------|-----------|-----------|
| CLAUDE.md | [CLAUDE.md](CLAUDE.md) | Instrucciones para Claude Code |
| Skills & Agentes | [docs/skills-catalog.md](docs/skills-catalog.md) | 22 skills + 6 agentes |
| Sprints | [docs/sprints/](docs/sprints/) | Documentos de sprint (trazabilidad) |
| Vista Escenarios | [docs/views/scenarios/](docs/views/scenarios/) | +1: Casos de uso end-to-end |
| Vista Lógica | [docs/views/logical/](docs/views/logical/) | Componentes, dominio, patrones |
| Vista Procesos | [docs/views/process/](docs/views/process/) | Flujos, concurrencia, eventos |
| Vista Desarrollo | [docs/views/development/](docs/views/development/) | Código, build, estándares |
| Vista Física | [docs/views/physical/](docs/views/physical/) | Infraestructura, despliegue |
| Changelog | [CHANGELOG.md](CHANGELOG.md) | Historial de releases (generado) |
| Log | [docs/log.md](docs/log.md) | Historial cronológico de operaciones |

## Protocolos

| Protocolo | Relación | Uso en dago |
|-----------|----------|-------------|
| MCP | Agente ↔ Herramientas | executor → mcp-registry → tools |
| A2A | Agente ↔ Agente | agent-registry → Agent Cards |
| AG-UI | Agente ↔ Usuario | orchestrator → dashboard (WebSocket) |
| A2UI | Agente → UI | nodo → dashboard (JSON declarativo) |
| OAuth 2.1 | Auth | auth-server → todos los servicios (JWT/JWKS) |

## Adaptadores

| Adaptador | Puerto | Implementación | Estado | Sprint |
|-----------|--------|----------------|--------|--------|
| Event Bus Valkey | libs/ports/eventbus.go | adapters/eventbus/valkey/ | planificado | [SPRINT-007](sprints/SPRINT-007-eventbus-valkey-adapter.md) |
| LLM Anthropic | libs/ports/llm.go | adapters/llm/anthropic/ | planificado | [SPRINT-008](sprints/SPRINT-008-llm-adapter-anthropic.md) |
| LLM Ollama (Mixtral) | libs/ports/llm.go | adapters/llm/ollama/ | planificado | [SPRINT-008](sprints/SPRINT-008-llm-adapter-anthropic.md) |
| LLM Fake | libs/ports/llm.go | adapters/llm/fake/ | planificado | [SPRINT-008](sprints/SPRINT-008-llm-adapter-anthropic.md) |

## Infraestructura de desarrollo

| Artefacto | Ubicación | Descripción |
|-----------|-----------|-------------|
| Makefile | `Makefile` | Build unificado: `make ci`, `make bootstrap`, `make build-all` |
| Docker Compose | `docker-compose.yml` | PostgreSQL 16 + pgvector + Valkey 8 |
| Linter | `.golangci.yml` | goimports, errcheck, funlen, wrapcheck (ADR-003, ADR-004) |
| Atlas config | `atlas.hcl` | Migraciones Ent → PostgreSQL (ADR-007) |
| Variables | `.env.example` | Variables de entorno para desarrollo local |

_Artefactos creados en SPRINT-001. Estado: completado (2026-04-30)._

## Estructura Go (creada en SPRINT-001)

| Directorio | Paquete Go | Descripción |
|------------|-----------|-------------|
| `libs/domain/` | `domain` | Tipos y lógica de negocio del dominio |
| `libs/ports/` | `ports` | Interfaces de puerto (arquitectura hexagonal) |
| `libs/schemas/` | `schemas` | Helpers de validación JSON Schema |
| `libs/utils/` | `utils` | Utilidades compartidas |
| `adapters/storage/` | `storage` | Adaptador de base de datos (Ent) |
| `adapters/eventbus/` | `eventbus` | Adaptador de event bus (Valkey Streams) |
| `adapters/auth/` | `auth` | Adaptador de autenticación |
| `adapters/llm/` | `llm` | Adaptadores de proveedores LLM |
| `adapters/metrics/` | `metrics` | Adaptador de métricas |
| `services/*/cmd/main.go` | `main` | Puntos de entrada para los 8 servicios |

## Dominio — Schemas Ent

_Nota: Los schemas Ent se crean en SPRINT-002. El directorio `ent/schema/` existe pero está vacío._

| Entidad | Schema Ent | Estado | Sprint |
|---------|------------|--------|--------|
| Graph | ent/schema/graph.go | planificado | [SPRINT-002](sprints/SPRINT-002-ent-schemas-graph-node-execution.md) |
| Node | ent/schema/node.go | planificado | [SPRINT-002](sprints/SPRINT-002-ent-schemas-graph-node-execution.md) |
| Execution | ent/schema/execution.go | planificado | [SPRINT-002](sprints/SPRINT-002-ent-schemas-graph-node-execution.md) |
| Episode | ent/schema/episode.go | pendiente | SPRINT-015 (memoria) |
| SemanticFact | ent/schema/semantic_fact.go | pendiente | SPRINT-015 (memoria, pgvector) |
| User | ent/schema/user.go | planificado | [SPRINT-004](sprints/SPRINT-004-auth-server-jwt-basico.md) |
| OrgUnit | ent/schema/org_unit.go | planificado | [SPRINT-004](sprints/SPRINT-004-auth-server-jwt-basico.md) |
| Package | ent/schema/package.go | pendiente | SPRINT-catalog |
| McpServer | ent/schema/mcp_server.go | pendiente | SPRINT-mcp |
| AgentCard | ent/schema/agent_card.go | pendiente | SPRINT-a2a |
