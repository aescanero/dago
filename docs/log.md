# dago Project Log

> Append-only chronological record of project operations.
> Each entry records what was done, when, and with what result.
>
> Format: `## [YYYY-MM-DD] type | Description`
> Types: `init`, `sprint`, `adr`, `spec`, `deploy`, `fix`, `decision`, `lint`
>
> Parseable with: `grep "^## \[" docs/log.md | tail -10`

---

## [2026-04-20] init | dago project initialised

**Artifacts created:**
- CLAUDE.md with instructions for Claude Code
- 20 ADRs (001-020) covering architecture, stack, processes
- Skills catalog with 22 skills and 6 agents
- Specs: OpenAPI 3.1, AsyncAPI 3.0, 13 JSON Schemas for patterns
- 4+1 documentation (Kruchten) with placeholders
- Claude Code configuration (.claude/settings.json, rules, commands, agents)
- Setup script (setup-dago.sh)

**Decisions made:**
- Monorepo with single Go module (8 backend services + 1 frontend)
- Spec Driven Development with 4 specs as sources of truth
- Communication: Valkey events for orchestration, HTTP for support
- Custom OAuth 2.1 (auth-server) + tag-based ABAC
- Three-layer agent memory (working + episodic + semantic)
- 7 node patterns + 5 flow patterns with JSON Schema
- Packages as distribution unit (workflow + skills + tools + UI)
- AG-UI + A2UI for frontend communication
- shadcn/ui + Module Federation for UI
- Reduced sprints with full traceability
- Conventional Commits + semver + automatic changelog (git-cliff)

**Status:** Definition phase complete. Ready for first sprint.

---

## [2026-04-27] sprint | SPRINT-001: Go monorepo bootstrap

**Planned artifacts:**
- `docs/sprints/SPRINT-001-bootstrap-monorepo.md`

**Scope:** Monorepo base infrastructure: go.mod, Makefile,
docker-compose (PostgreSQL 16 + pgvector + Valkey 8), .golangci.yml,
complete directory structure (libs/, adapters/, services/ ×8),
compilable Go stubs, atlas.hcl, pipeline smoke tests.

**TODOs:** 9 (infra ×7, test ×1, docs ×1) — see sprint document.

**Status:** completed

---

## [2026-04-30] sprint | SPRINT-001: completed

**Result:** All TODOs implemented. 13/13 smoke tests pass.
`make ci` exits 0. 8 binaries in bin/. Compilable monorepo.

**Artifacts created:**
- `go.mod` + `go.sum` (module github.com/aescanero/dago, Go 1.25)
- `Makefile` (20 targets)
- `docker-compose.yml` (pgvector:pg16 + valkey:8, healthchecks)
- `.golangci.yml` (17 linters, ADR-003 + ADR-004)
- `atlas.hcl` (minimal configuration)
- Package stubs: libs/ (4), adapters/ (5), services/ ×8 cmd/main.go
- `tests/smoke/build_test.go` (5 tests, build tag: smoke)

**Verifications:**
- `go mod verify` → OK
- `go build ./...` → 0 errors
- `go vet ./...` → 0 issues
- `golangci-lint run ./...` → 0 errors
- `make test-smoke` → 13/13 PASS
- `make ci` → 0 (lint + build-all + test)
- `bin/` → 8 binaries

---

## [2026-04-27] sprint | SPRINT-002: Ent schemas — Graph, Node, Execution

**Planned artifacts:**
- `docs/sprints/SPRINT-002-ent-schemas-graph-node-execution.md`

**Scope:** Ent schemas for Graph (graph definition), Node (vertex with
7 ADR-016 patterns) and Execution (execution instance, ADR-015 working memory).
`go generate ./ent`, Atlas migration, unit tests with in-memory SQLite,
integration tests against real PostgreSQL.

**TODOs:** 10 (spec ×1, data ×4, test ×2, infra ×2, docs ×1) — see sprint document.

**Blocks:** SPRINT-003 (orchestrator), SPRINT-015 (memory).

**Status:** completed

---

## [2026-05-04] sprint | SPRINT-002: PR #3 merged to main

**PR:** https://github.com/aescanero/dago/pull/3 — squash-merged.
**Reviewer verdict:** APPROVED. Non-blocking observations propagated to SPRINT-003.
**main** is now at commit `0953baf` with full Ent data model for Graph, Node, Execution.

---

## [2026-05-03] sprint | SPRINT-002: completed

**Result:** All 10 TODOs implemented. 14/14 tests pass (12 unit + 2 integration).
`go build ./...` and `go vet ./...` exit 0. Migration applied to PostgreSQL.

**Artifacts created:**
- `ent/schema/graph.go`, `node.go`, `execution.go` — 3 Ent schemas
- `ent/` full generated client (27+ files) — `go generate ./ent`
- `cmd/migrate/main.go` — migration generation tool
- `migrations/20260503191126_init_graph_node_execution.up.sql` — SQL migration
- `migrations/20260503191126_init_graph_node_execution.down.sql` — rollback SQL
- `tests/unit/schema/` — 12 unit tests (graph, node, execution)
- `tests/integration/schema_integration_test.go` — 2 PostgreSQL integration tests
- `go.mod` updated: `github.com/lib/pq v1.12.3`

**Verifications:**
- `go build ./...` → 0 errors
- `go vet ./...` → 0 issues
- `go test ./tests/unit/schema/...` → 12/12 PASS
- `go test -tags=integration ./tests/integration/...` → 2/2 PASS
- `psql \dt` → graphs, nodes, executions tables confirmed
- FK nodes.graph_nodes → graphs.id and executions.graph_executions → graphs.id active
- jsonb, uuid, timestamptz column types verified

**Decisions:** Atlas lint Pro-only since v0.38; used `schema.Diff` API instead of CLI `ent://` URL.

---

## [2026-04-27] sprint | SPRINT-003: Orchestrator REST API — graph CRUD + executions

**Planned artifacts:**
- `docs/sprints/SPRINT-003-api-rest-orchestrator.md`

**Scope:** Full OpenAPI spec for 6 endpoints (POST/GET/GET list/PUT/DELETE
graphs + POST executions). Domain types in `libs/domain/`, ports in
`libs/ports/`. Use cases, Gin handlers, Ent adapter. Contract, unit, and smoke tests.

**TODOs:** 14 (spec ×2, domain ×2, test ×4, impl ×4, docs ×1 + smoke ×1).

**Key decisions:** DELETE archives (no physical delete); PUT only for `draft` graphs;
POST /executions creates as `pending` without publishing events (that is SPRINT-004).

**Blocks:** SPRINT-004 (Valkey events), SPRINT-dashboard-001 (frontend).
**Blocked by:** SPRINT-001 (go.mod + Gin), SPRINT-002 (Ent schemas).

**Status:** completed

---

## [2026-05-05] sprint | SPRINT-003: completed

**Result:** All 14 TODOs implemented. Tests pass (unit, contract, smoke).
`go build ./...` exits 0.

**Artifacts created:**
- `specs/schemas/graph.yaml`, `specs/schemas/execution.yaml` — OpenAPI schemas
- `specs/paths/graphs.yaml`, `specs/paths/executions.yaml` — 6 endpoints
- `specs/openapi.yaml` — updated with path and schema refs
- `libs/domain/graph.go`, `execution.go`, `errors.go` — domain types (no infrastructure imports)
- `libs/ports/storage.go` — GraphRepository + ExecutionRepository interfaces
- `tests/testutil/fakes/graph_repo.go`, `execution_repo.go` — in-memory fakes
- `services/orchestrator/internal/usecase/graph.go`, `execution.go`, `validate.go` — 6 use cases
- `services/orchestrator/internal/handler/graph.go`, `execution.go`, `errors.go` — Gin handlers
- `services/orchestrator/internal/router/router.go` — Gin router
- `services/orchestrator/testutil/server.go` — test server builder
- `adapters/storage/graph_repo.go` — EntGraphRepository + EntExecutionRepository
- `services/orchestrator/cmd/main.go` — full main with graceful shutdown
- `tests/contract/graphs_contract_test.go` — 8 contract tests (build tag: contract)
- `tests/smoke/api_smoke_test.go` — 2 smoke tests (build tag: smoke)

**Verifications:**
- `go build ./...` → 0 errors
- `go test ./services/orchestrator/internal/usecase/...` → 9/9 PASS
- `go test ./services/orchestrator/internal/handler/...` → 6/6 PASS
- `go test -tags=contract ./tests/contract/...` → 8/8 PASS
- `go test -tags=smoke ./tests/smoke/...` → 2/2 PASS

**Decisions:**
- Unit tests co-located within internal packages (Go's internal rule prevents external imports).
- `services/orchestrator/testutil/server.go` (non-internal) exposes test server builder for contract tests.
- Gin moved to direct dependency (added via `go get github.com/gin-gonic/gin`).
- `mapDomainError()` covers all 4 domain errors; default → 500 INTERNAL_ERROR.

---

## [2026-04-29] sprint | SPRINT-004: Basic auth-server — local login, JWT RS256, JWKS, middleware

**Planned artifacts:**
- `docs/sprints/SPRINT-004-auth-server-jwt-basico.md`

**Scope:** Ent schemas User+OrgUnit, local login with argon2id (OWASP),
JWT RS256 issuance with ABAC claims, `/.well-known/jwks.json` endpoint,
JWKS validator adapter, JWT middleware for orchestrator (bypass in dev).

**TODOs:** 16 (spec ×1, data ×2, domain ×1, test ×4, impl ×6, docs ×1 + integration ×1).

**Key decisions:** LOGIN does not reveal whether email exists (same error for not found
and wrong password); UserResponse never includes password_hash; middleware in bypass mode
(AUTH_REQUIRED=false) does not affect SPRINT-003 routes.

**Parallel to:** SPRINT-002, SPRINT-003 (only depends on SPRINT-001).
**Blocks:** SPRINT-005 (protected orchestrator routes), SPRINT-ABAC.

**Status:** planned

---

## [2026-04-29] sprint | SPRINT-005: Dashboard bootstrap — React 19, shadcn/ui, PKCE, OpenAPI types

**Planned artifacts:**
- `docs/sprints/SPRINT-005-dashboard-bootstrap-pkce.md`

**Scope:** auth-server extension with GET/POST /authorize + POST /token (full PKCE).
Dashboard bootstrap: Vite + React 19 + TypeScript strict + Tailwind + shadcn/ui + dark mode.
TypeScript type generation from OpenAPI (`openapi-typescript`). PKCE module with Web Crypto
API (tokens in memory, never localStorage). GraphsPage with TanStack Query.
Tests with Vitest + RTL + MSW.

**TODOs:** 13 (spec ×1, infra ×3, test ×4, impl ×3, docs ×1 + smoke ×1).

**Key decisions:** PKCE S256 with native Web Crypto (no dependencies). `code_verifier`
in sessionStorage only during redirect (cleaned up). Authorization codes in
in-memory sync.Map with TTL (documented limitation: not distributed).

**Blocks:** SPRINT-006 (graph editor), SPRINT-007 (execution monitor AG-UI).
**Blocked by:** SPRINT-003 (GET /api/v1/graphs), SPRINT-004 (JWT issuance).

**Status:** planned

---

## [2026-04-29] sprint | SPRINT-006: Dashboard graph feature — list, detail, create, edit

**Planned artifacts:**
- `docs/sprints/SPRINT-006-dashboard-feature-grafos.md`

**Scope:** Feature module `dashboard/src/features/graphs/`. Additional shadcn/ui components
(form, textarea, alert-dialog, breadcrumb, alert, tabs, switch, select, tooltip, scroll-area,
collapsible). Zod schema with 4 ADR-016 templates. 5 TanStack Query hooks. Status and
pattern badges. GraphDefinitionViewer (collapsible). GraphForm (React Hook Form + Zod). 4 pages
(improved GraphsPage, GraphCreatePage, GraphDetailPage 3 tabs, GraphEditPage). Updated routes.

**TODOs:** 15 (infra ×1, spec ×1, test ×4, impl ×7, smoke ×1, docs ×1).

**Key decisions:** DELETE archives via AlertDialog (409 shows "has active executions");
editing blocked for non-draft graphs (Alert instead of form); status filter persists in
URL query params; collapsible GraphDefinitionViewer without heavy editor.

**Blocks:** SPRINT-007 (execution monitor AG-UI).
**Blocked by:** SPRINT-005 (dashboard bootstrap, PKCE, base hooks).

**Status:** planned

---

## [2026-04-29] sprint | SPRINT-007: Event Bus adapter — Valkey Streams + consumer groups

**Planned artifacts:**
- `docs/sprints/SPRINT-007-eventbus-valkey-adapter.md`

**Scope:** `EventPublisher`/`EventConsumer` port in `libs/ports/eventbus.go`. Domain
types `Event`+`EventAuth` in `libs/domain/events.go`. Valkey Streams adapter in
`adapters/eventbus/valkey/`: publisher (XADD + XGROUP CREATE MKSTREAM), consumer
(XREADGROUP + ACK/NACK), pending recovery (XAUTOCLAIM), CloudEvents envelope with auth field,
DLQ after MaxRetries (default 3). AsyncAPI spec with 7 channels. Integration tests with
Testcontainers (6 cases).

**TODOs:** 11 (spec ×1, data ×1, impl ×1, test ×1, impl ×4, infra ×2, docs ×1).

**Key decisions:** `valkey-io/valkey-go` client (not go-redis, ADR-008); single JSON field
envelope in stream entry; DLQ after MaxRetries with XACK to purge from pending;
XAUTOCLAIM for recovery of crashed consumers; integration tests with `integration` build tag
separate from `make ci`.

**Parallel to:** SPRINT-002, SPRINT-003, SPRINT-004 (only depends on SPRINT-001).
**Blocks:** orchestrator (publish `execution.requested`), executor and router (consume events).

**Status:** planned

---

## [2026-04-29] sprint | SPRINT-008: LLM adapter — LLMClient port + Anthropic + Ollama/Mixtral

**Planned artifacts:**
- `docs/sprints/SPRINT-008-llm-adapter-anthropic.md`

**Scope:** `LLMClient` port in `libs/ports/llm.go`. Three new domain errors
(`ErrUnauthorized`, `ErrRateLimited`, `ErrProviderUnavailable`). Anthropic adapter
in `adapters/llm/anthropic/` (7 tests with `httptest.NewServer`). Ollama adapter
(OpenAI-compatible API) in `adapters/llm/ollama/` with default model `mixtral` and
`convertFinishReason` (6 tests with `httptest.NewServer`). Deterministic `FakeLLMClient`.
`anthropic-sdk-go` and `go-openai` dependencies in go.mod. 14 unit tests total.

**TODOs:** 16 (data ×1, impl ×1, test ×3, impl ×7, infra ×3, docs ×1).

**Key decisions:** tests with `httptest.NewServer` without real credentials in CI;
`go-openai` for Ollama (reusable for native OpenAI in future sprints);
`NewOllamaClient` with no error return (BaseURL has valid default); domain errors
as sentinels for `errors.Is`; `FakeLLMClient` without mock frameworks (ADR-003).

**Parallel to:** SPRINT-002, SPRINT-003, SPRINT-004, SPRINT-007 (only depends on SPRINT-001).
**Blocks:** executor (`llm_call`, `react`, `reflection`), LLM router.

**Status:** planned

---

## [2026-04-30] sprint | SPRINT-009: Executor — llm_call pattern handler

**Planned artifacts:**
- `docs/sprints/SPRINT-009-executor-llm-call.md`

**Scope:** `node.execute.requested` stream consumer. `LLMCallHandler` for the `llm_call`
pattern per `specs/patterns/nodes/llm_call.json`. Simplified `input_mapping`/`output_mapping`
evaluator (paths: `state.variables.<name>`, `state.messages[-1].content`, `output.content`,
`output.stop_reason`). Pattern-based `Dispatcher`. Publisher of `node.executed` and
`node.execute.failed`. Wiring in `services/executor/main.go`. Executor AsyncAPI operations.
10 unit tests with `FakeLLMClient`; 1 integration test with real Valkey (build tag `integration`).

**TODOs:** 13 (spec ×2, test ×4, impl ×5, infra ×1, docs ×1).

**Key decisions:** LLMClient provider selection in `main.go` (not in the handler);
ACK for non-retryable errors (`node.execute.failed` already published); inline `fakePublisher`
in tests until another handler needs it; path evaluator limited to SPRINT-009 use cases.

**Blocked by:** SPRINT-007 (eventbus), SPRINT-008 (LLMClient + domain errors).
**Blocks:** executor patterns `tool_use`, `react`, `reflection`.

**Status:** planned

---

## [2026-04-29] sprint | SPRINT-010: Orchestrator state machine

**Planned artifacts:**
- `docs/sprints/SPRINT-010-orchestrator-state-machine.md`

**Scope:** Graph validation with `dominikbraun/graph` (only `sequential` edges).
Extend `StartExecution`: validate → publish `node.execute.requested` → status `running`.
`ExecutionStateMachine` with `HandleNodeExecuted` and `HandleNodeExecuteFailed`.
`node_result` consumer consumes `node.executed` and `node.execute.failed`. `ErrRetryable`
for NACK. `UpdateExecution` in port + Ent adapter. 4 unit tests + 2 integration tests.

**TODOs:** 17 (spec ×2, test ×4, data ×1, impl ×1, impl ×6, infra ×1, docs ×1 + integration ×1).

**Key decisions:** Only `sequential` edges (conditional/parallel/loop/interrupt → 422
GRAPH_VALIDATION_ERROR). Per-node timeout excluded (documented as future TODO).
`StartExecution` goes directly to `running` (not `pending`). Checkpointing before publishing
next event. `CanTransitionTo` for idempotency.

**Blocked by:** SPRINT-003 (repos, use cases), SPRINT-007 (event bus), SPRINT-009 (executor).
**Blocks:** SPRINT-011 (executor tool_use), SPRINT-015 (episodic memory).

**Status:** planned
