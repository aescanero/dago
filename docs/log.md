# Log del Proyecto dago

> Registro cronológico append-only de operaciones del proyecto.
> Cada entrada registra qué se hizo, cuándo, y con qué resultado.
>
> Formato: `## [YYYY-MM-DD] tipo | Descripción`
> Tipos: `init`, `sprint`, `adr`, `spec`, `deploy`, `fix`, `decision`, `lint`
>
> Parseable con: `grep "^## \[" docs/log.md | tail -10`

---

## [2026-04-20] init | Proyecto dago inicializado

**Artefactos creados:**
- CLAUDE.md con instrucciones para Claude Code
- 20 ADRs (001-020) cubriendo arquitectura, stack, procesos
- Skills catalog con 22 skills y 6 agentes
- Specs: OpenAPI 3.1, AsyncAPI 3.0, 13 JSON Schemas de patrones
- Documentación 4+1 (Kruchten) con placeholders
- Configuración Claude Code (.claude/settings.json, rules, commands, agents)
- Script de setup (setup-dago.sh)

**Decisiones tomadas:**
- Monorepo con un solo módulo Go (8 servicios backend + 1 frontend)
- Spec Driven Development con 4 specs como fuentes de verdad
- Comunicación: eventos Valkey para orquestación, HTTP para soporte
- OAuth 2.1 propio (auth-server) + ABAC por etiquetas
- Memoria de agentes en 3 capas (working + episodic + semantic)
- 7 patrones de nodo + 5 patrones de flujo con JSON Schema
- Paquetes como unidad de distribución (workflow + skills + tools + UI)
- AG-UI + A2UI para comunicación frontend
- shadcn/ui + Module Federation para UI
- Sprints reducidos con trazabilidad completa
- Conventional Commits + semver + changelog automático (git-cliff)

**Estado:** Fase de definición completada. Listo para primer sprint.

---

## [2026-04-27] sprint | SPRINT-001: Bootstrap del monorepo Go

**Artefactos planificados:**
- `docs/sprints/SPRINT-001-bootstrap-monorepo.md`

**Alcance:** Infraestructura base del monorepo: go.mod, Makefile,
docker-compose (PostgreSQL 16 + pgvector + Valkey 8), .golangci.yml,
estructura de directorios completa (libs/, adapters/, services/ ×8),
stubs Go compilables, atlas.hcl, smoke tests del pipeline.

**TODOs:** 9 (infra ×7, test ×1, docs ×1) — ver documento de sprint.

**Estado:** completado

---

## [2026-04-30] sprint | SPRINT-001: completado

**Resultado:** Todos los TODOs implementados. 13/13 smoke tests pasan.
`make ci` finaliza en 0. 8 binarios en bin/. Monorepo compilable.

**Artefactos creados:**
- `go.mod` + `go.sum` (módulo github.com/aescanero/dago, Go 1.25)
- `Makefile` (20 targets)
- `docker-compose.yml` (pgvector:pg16 + valkey:8, healthchecks)
- `.golangci.yml` (17 linters, ADR-003 + ADR-004)
- `atlas.hcl` (configuración mínima)
- Package stubs: libs/ (4), adapters/ (5), services/ ×8 cmd/main.go
- `tests/smoke/build_test.go` (5 tests, build tag: smoke)

**Verificaciones:**
- `go mod verify` → OK
- `go build ./...` → 0 errores
- `go vet ./...` → 0 problemas
- `golangci-lint run ./...` → 0 errores
- `make test-smoke` → 13/13 PASS
- `make ci` → 0 (lint + build-all + test)
- `bin/` → 8 binarios

---

## [2026-04-27] sprint | SPRINT-002: Schemas Ent — Graph, Node, Execution

**Artefactos planificados:**
- `docs/sprints/SPRINT-002-ent-schemas-graph-node-execution.md`

**Alcance:** Schemas Ent de Graph (definición de grafo), Node (vértice
con 7 patrones de ADR-016) y Execution (instancia de ejecución, working
memory de ADR-015). `go generate ./ent`, migración Atlas, tests unitarios
con SQLite en memoria y tests de integración contra PostgreSQL real.

**TODOs:** 10 (spec ×1, data ×4, test ×2, infra ×2, docs ×1) — ver documento de sprint.

**Bloquea:** SPRINT-003 (orchestrator), SPRINT-015 (memoria).

**Estado:** planificado

---

## [2026-04-27] sprint | SPRINT-003: API REST orchestrator — CRUD grafos y ejecuciones

**Artefactos planificados:**
- `docs/sprints/SPRINT-003-api-rest-orchestrator.md`

**Alcance:** Spec OpenAPI completa para 6 endpoints (POST/GET/GET list/PUT/DELETE
grafos + POST ejecuciones). Tipos de dominio en `libs/domain/`, puertos en
`libs/ports/`. Casos de uso, handlers Gin, adaptador Ent. Tests de contrato,
unitarios y smoke.

**TODOs:** 14 (spec ×2, domain ×2, test ×4, impl ×4, docs ×1 + smoke ×1).

**Decisiones clave:** DELETE archiva (no borra físicamente); PUT solo grafos
`draft`; POST /executions crea en `pending` sin publicar evento (eso es SPRINT-004).

**Bloquea:** SPRINT-004 (eventos Valkey), SPRINT-dashboard-001 (frontend).
**Bloqueado por:** SPRINT-001 (go.mod + Gin), SPRINT-002 (Ent schemas).

**Estado:** planificado

---

## [2026-04-29] sprint | SPRINT-004: Auth-server básico — login local, JWT RS256, JWKS, middleware

**Artefactos planificados:**
- `docs/sprints/SPRINT-004-auth-server-jwt-basico.md`

**Alcance:** Ent schemas User+OrgUnit, login local con argon2id (OWASP),
emisión JWT RS256 con claims ABAC, endpoint `/.well-known/jwks.json`,
adaptador JWKS validator, middleware JWT para orchestrator (bypass en dev).

**TODOs:** 16 (spec ×1, data ×2, domain ×1, test ×4, impl ×6, docs ×1 + integration ×1).

**Decisiones clave:** LOGIN no revela si email existe (same error para not found
y wrong password); UserResponse nunca incluye password_hash; middleware en modo
bypass (AUTH_REQUIRED=false) no afecta rutas SPRINT-003.

**Paralelo a:** SPRINT-002, SPRINT-003 (solo depende de SPRINT-001).
**Bloquea:** SPRINT-005 (rutas protegidas orchestrator), SPRINT-ABAC.

**Estado:** planificado

---

## [2026-04-29] sprint | SPRINT-005: Dashboard bootstrap — React 19, shadcn/ui, PKCE, tipos OpenAPI

**Artefactos planificados:**
- `docs/sprints/SPRINT-005-dashboard-bootstrap-pkce.md`

**Alcance:** Extensión auth-server con GET/POST /authorize + POST /token (PKCE completo).
Bootstrap dashboard: Vite + React 19 + TypeScript strict + Tailwind + shadcn/ui + dark mode.
Generación tipos TypeScript desde OpenAPI (`openapi-typescript`). Módulo PKCE con Web Crypto
API (tokens en memoria, nunca localStorage). GraphsPage con TanStack Query.
Tests con Vitest + RTL + MSW.

**TODOs:** 13 (spec ×1, infra ×3, test ×4, impl ×3, docs ×1 + smoke ×1).

**Decisiones clave:** PKCE S256 con Web Crypto nativa (sin dependencias). `code_verifier`
en sessionStorage solo durante el redirect (se limpia). Códigos de autorización en
sync.Map in-memory con TTL (limitación documentada: no distribuido).

**Bloquea:** SPRINT-006 (graph editor), SPRINT-007 (execution monitor AG-UI).
**Bloqueado por:** SPRINT-003 (GET /api/v1/graphs), SPRINT-004 (JWT emisión).

**Estado:** planificado

---

## [2026-04-29] sprint | SPRINT-006: Dashboard feature grafos — listado, detalle, creación, edición

**Artefactos planificados:**
- `docs/sprints/SPRINT-006-dashboard-feature-grafos.md`

**Alcance:** Feature module `dashboard/src/features/graphs/`. Componentes shadcn/ui adicionales
(form, textarea, alert-dialog, breadcrumb, alert, tabs, switch, select, tooltip, scroll-area,
collapsible). Schema Zod con 4 templates ADR-016. 5 hooks TanStack Query. Badges de estado y
patrón. GraphDefinitionViewer (colapsible). GraphForm (React Hook Form + Zod). 4 páginas
(GraphsPage mejorada, GraphCreatePage, GraphDetailPage 3 tabs, GraphEditPage). Rutas actualizadas.

**TODOs:** 15 (infra ×1, spec ×1, test ×4, impl ×7, smoke ×1, docs ×1).

**Decisiones clave:** DELETE archiva vía AlertDialog (409 muestra "tiene ejecuciones activas");
edición bloqueada para grafos no-draft (Alert en lugar de form); filtro de estado persiste en
URL query params; GraphDefinitionViewer colapsible sin editor pesado.

**Bloquea:** SPRINT-007 (execution monitor AG-UI).
**Bloqueado por:** SPRINT-005 (dashboard bootstrap, PKCE, hooks base).

**Estado:** planificado

---

## [2026-04-29] sprint | SPRINT-007: Adaptador Event Bus — Valkey Streams + consumer groups

**Artefactos planificados:**
- `docs/sprints/SPRINT-007-eventbus-valkey-adapter.md`

**Alcance:** Puerto `EventPublisher`/`EventConsumer` en `libs/ports/eventbus.go`. Tipos de
dominio `Event`+`EventAuth` en `libs/domain/events.go`. Adaptador Valkey Streams en
`adapters/eventbus/valkey/`: publisher (XADD + XGROUP CREATE MKSTREAM), consumer
(XREADGROUP + ACK/NACK), pending recovery (XAUTOCLAIM), envelope CloudEvents con campo auth,
DLQ tras MaxRetries (default 3). Spec AsyncAPI con 7 canales. Tests de integración con
Testcontainers (6 casos).

**TODOs:** 11 (spec ×1, domain ×1, port ×1, test ×1, impl ×4, infra ×2, docs ×1).

**Decisiones clave:** cliente `valkey-io/valkey-go` (no go-redis, ADR-008); envelope único
campo JSON en el stream entry; DLQ tras MaxRetries con XACK para purgar del pending;
XAUTOCLAIM para recovery de consumers caídos; tests de integración con build tag `integration`
separados del target `make ci`.

**Paralelo a:** SPRINT-002, SPRINT-003, SPRINT-004 (solo depende de SPRINT-001).
**Bloquea:** orchestrator (publicar `execution.requested`), executor y router (consumir eventos).

**Estado:** planificado

---

## [2026-04-29] sprint | SPRINT-008: Adaptador LLM — Puerto LLMClient + Anthropic + Ollama/Mixtral

**Artefactos planificados:**
- `docs/sprints/SPRINT-008-llm-adapter-anthropic.md`

**Alcance:** Puerto `LLMClient` en `libs/ports/llm.go`. Tres errores de dominio nuevos
(`ErrUnauthorized`, `ErrRateLimited`, `ErrProviderUnavailable`). Adaptador Anthropic
en `adapters/llm/anthropic/` (7 tests con `httptest.NewServer`). Adaptador Ollama
(API OpenAI-compatible) en `adapters/llm/ollama/` con modelo default `mixtral` y
`convertFinishReason` (6 tests con `httptest.NewServer`). `FakeLLMClient` determinista.
Dependencias `anthropic-sdk-go` y `go-openai` en go.mod. 14 tests unitarios en total.

**TODOs:** 16 (domain ×1, port ×1, test ×3, impl ×7, infra ×3, docs ×1).

**Decisiones clave:** tests con `httptest.NewServer` sin credenciales reales en CI;
`go-openai` para Ollama (reutilizable para OpenAI nativo en sprints futuros);
`NewOllamaClient` sin error de retorno (BaseURL con default válido); errores de dominio
como centinelas para `errors.Is`; `FakeLLMClient` sin mock frameworks (ADR-003).

**Paralelo a:** SPRINT-002, SPRINT-003, SPRINT-004, SPRINT-007 (solo depende de SPRINT-001).
**Bloquea:** executor (`llm_call`, `react`, `reflection`), router LLM.

**Estado:** planificado

---

## [2026-04-30] sprint | SPRINT-009: Executor — Handler del patrón llm_call

**Artefactos planificados:**
- `docs/sprints/SPRINT-009-executor-llm-call.md`

**Alcance:** Consumer del stream `node.execute.requested`. Handler `LLMCallHandler`
para el patrón `llm_call` según `specs/patterns/nodes/llm_call.json`. Evaluador
simplificado de `input_mapping`/`output_mapping` (paths: `state.variables.<name>`,
`state.messages[-1].content`, `output.content`, `output.stop_reason`). `Dispatcher`
por patrón. Publisher de `node.executed` y `node.execute.failed`. Wiring en
`services/executor/main.go`. Operaciones AsyncAPI del executor. 10 tests unitarios con
`FakeLLMClient`; 1 test de integración con Valkey real (build tag `integration`).

**TODOs:** 13 (spec ×2, test ×4, impl ×5, infra ×1, docs ×1).

**Decisiones clave:** selección de LLMClient por provider en `main.go` (no en el handler);
ACK de errores no-retryable (el `node.execute.failed` ya fue publicado); `fakePublisher`
inline en tests hasta que otro handler lo necesite; evaluador de paths limitado a los
casos de uso de SPRINT-009.

**Bloqueado por:** SPRINT-007 (eventbus), SPRINT-008 (LLMClient + errores de dominio).
**Bloquea:** executor patrones `tool_use`, `react`, `reflection`.

**Estado:** planificado

---

## [2026-04-29] sprint | SPRINT-010: Orchestrator state machine

**Artefactos planificados:**
- `docs/sprints/SPRINT-010-orchestrator-state-machine.md`

**Alcance:** Validar grafos con `dominikbraun/graph` (solo aristas `sequential`).
Extender `StartExecution`: validar → publicar `node.execute.requested` → estado `running`.
`ExecutionStateMachine` con `HandleNodeExecuted` y `HandleNodeExecuteFailed`.
Consumer `node_result` consume `node.executed` y `node.execute.failed`. `ErrRetryable`
para NACK. `UpdateExecution` en port + adaptador Ent. 4 tests unitarios + 2 integración.

**TODOs:** 17 (spec ×2, test ×4, domain ×1, port ×1, impl ×6, infra ×1, docs ×1 + integración ×1).

**Decisiones clave:** Solo aristas `sequential` (conditional/parallel/loop/interrupt → 422
GRAPH_VALIDATION_ERROR). Timeout por nodo excluido (documentado como TODO futuro).
`StartExecution` pasa directo a `running` (no `pending`). Checkpointing antes de publicar
siguiente evento. `CanTransitionTo` para idempotencia.

**Bloqueado por:** SPRINT-003 (repos, use cases), SPRINT-007 (event bus), SPRINT-009 (executor).
**Bloquea:** SPRINT-011 (executor tool_use), SPRINT-015 (memoria episódica).

**Estado:** planificado
