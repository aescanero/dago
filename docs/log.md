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

**Status:** completed

---

## [2026-05-06] sprint | SPRINT-004: completed

**Result:** All 16 TODOs implemented. Unit tests, contract tests, and integration tests pass.
`go build ./...` exits 0.

**Artifacts created:**
- `specs/schemas/auth.yaml` — RegisterInput, LoginInput, TokenResponse, UserResponse, JWKSResponse
- `specs/paths/auth.yaml` — register, login, JWKS (3 endpoints)
- `specs/openapi.yaml` — updated with auth path and schema refs
- `ent/schema/user.go`, `ent/schema/org_unit.go` — Ent schemas with Sensitive(), UUID, TIMESTAMPTZ
- `ent/` (generated) — EntClient regenerated including User and OrgUnit types
- `migrations/20260429000000_add_user_org_unit.up.sql` — SQL migration for users and org_units tables
- `libs/domain/user.go` — User + Credentials types (no crypto imports)
- `libs/domain/token.go` — Claims, ClaimsAttrs, TokenPair, scope constants
- `libs/domain/errors.go` — added ErrInvalidCredentials
- `libs/ports/auth.go` — PasswordHasher, TokenIssuer, TokenValidator, UserRepository ports
- `adapters/auth/argon2id.go` — OWASP 2023 argon2id hasher with PHC format + ConstantTimeCompare
- `adapters/auth/jwt_issuer.go` — RS256 JWT issuer with ABAC attrs
- `adapters/auth/jwks_validator.go` — RSAValidator (static public key, for tests/dev)
- `adapters/auth/jwks_http_validator.go` — JWKSHTTPValidator (lazy HTTP fetch + 5min cache)
- `adapters/auth/jwks_endpoint.go` — PublicKeyToJWKSJSON helper
- `adapters/auth/keygen.go` — MustGenerateRSAKeyPair (dev only)
- `adapters/auth/ent_user_repo.go` — EntUserRepository implementing UserRepository
- `services/auth-server/internal/usecase/register.go` — RegisterUser use case
- `services/auth-server/internal/usecase/login.go` — LoginUser use case (timing-safe, no email reveal)
- `services/auth-server/internal/handler/auth.go` — Register, Login, JWKS handlers
- `services/auth-server/internal/handler/auth_handler_test.go` — 4 handler unit tests
- `services/auth-server/internal/router/router.go` — Gin router
- `services/auth-server/testutil/server.go` — test server builder (for contract and integration tests)
- `services/auth-server/cmd/main.go` — full wiring with graceful shutdown, RSA key loading
- `services/orchestrator/internal/middleware/auth.go` — JWT middleware (bypass when AUTH_REQUIRED=false)
- `services/orchestrator/internal/middleware/auth_middleware_test.go` — 4 middleware unit tests
- `services/orchestrator/internal/router/router.go` — updated to register auth middleware + /api/v1/protected group
- `tests/testutil/fakes/user_repo.go` — InMemoryUserRepository
- `tests/contract/auth_contract_test.go` — 6 contract tests (build tag: contract)
- `tests/integration/auth_integration_test.go` — 1 end-to-end integration test (build tag: integration)

**Verifications:**
- `go build ./...` → 0 errors
- `go test ./tests/unit/auth/...` → 11/11 PASS (argon2id: 5, JWT: 6)
- `go test ./services/auth-server/internal/handler/...` → 4/4 PASS
- `go test ./services/orchestrator/internal/middleware/...` → 4/4 PASS
- `go test -tags=contract ./tests/contract/...` → 6 auth + 8 graph PASS
- `go test -tags=integration ./tests/integration/...` → 1/1 PASS

**Decisions:**
- Unit tests for handlers and middleware co-located in respective internal packages (Go's internal rule).
- `golang-jwt/jwt/v5` added as direct dependency.
- Atlas CLI not available in environment — migration SQL written manually.
- JWKSHTTPValidator has lazy caching (5min TTL) for production; RSAValidator (static key) for tests.
- Orchestrator middleware registered only on empty `/api/v1/protected` group — SPRINT-005 applies it to routes.

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

---

## [2026-05-07] sprint | SPRINT-005: Dashboard bootstrap — React 19, shadcn/ui, PKCE, OpenAPI types

**Artifacts created:**
- `docs/sprints/SPRINT-005-dashboard-bootstrap-pkce.md` (pre-existing)
- `specs/paths/auth.yaml` — GET/POST `/authorize`, POST `/token` endpoints
- `specs/schemas/auth.yaml` — `TokenRequest`, `AuthorizeParams` schemas
- `specs/openapi.yaml` — wired new paths and schemas
- `services/auth-server/internal/oauth/code_store.go` — `InMemoryCodeStore` + `AuthorizationCode`
- `services/auth-server/internal/handler/oauth.go` — `OAuthHandler` (PKCE flow)
- `services/auth-server/internal/router/router.go` — wired OAuth routes
- `services/auth-server/cmd/main.go` — wired `OAuthHandler` dependencies
- `services/auth-server/testutil/server.go` — updated to pass `OAuthHandler`
- `dashboard/` — complete React 19 + TypeScript + Vite + Tailwind 4 scaffold
- `dashboard/src/auth/` — PKCE module, AuthProvider, useAuth, ProtectedRoute, AuthCallback
- `dashboard/src/components/ui/` — Button, Badge, Skeleton, Card, Table, Avatar
- `dashboard/src/api/` — OpenAPI-generated types stub + createApiClient
- `dashboard/src/hooks/useGraphs.ts` — TanStack Query v5 hook
- `dashboard/src/pages/GraphsPage.tsx` — graph listing with pagination
- `dashboard/src/layouts/AppLayout.tsx` — sidebar + header + dark mode
- `dashboard/src/App.tsx`, `main.tsx` — routing + provider wiring
- `dashboard/scripts/smoke.sh` — build/type-check/lint/test pipeline
- `Makefile` — `gen-api-types`, `dashboard-check` targets

**Tests:** 15 passing (5 PKCE, 5 AuthProvider, 5 GraphsPage).

**Key decisions:**
- `InMemoryCodeStore` uses `sync.RWMutex` (not `sync.Map`) for iteration under lock during cleanup.
- Authorization codes stored by `SHA-256(code_plaintext)` — plaintext never persisted.
- Auth code struct stores `Email` for user lookup in PostToken (avoids re-auth).
- `redirect_uri` validated as `http://localhost` prefix only (TODO(SPRINT-ABAC): validate against registered clients).
- PKCE token stored in `useRef` (never in localStorage), state in `useState`.
- `GraphsPage` tests use `vi.mock("@/hooks/useGraphs")` for isolated component testing (MSW used in AuthProvider tests).
- `types.gen.ts` is committed as the frontend↔backend contract; regenerate with `make gen-api-types`.
- `@radix-ui/react-slot` added as a runtime dependency for shadcn/ui Button.

**Status:** completed

---

## SPRINT-006 — Dashboard: Graphs feature (2026-05-07)

**Objective:** Implement the graphs feature in the dashboard: list with status filter, detail page with tabs, creation and edit with validated forms, archive action — connecting to the 5 CRUD endpoints from SPRINT-003.

**Files created/modified:**
- `dashboard/src/api/types.gen.ts` — updated with full GraphInput, GraphResponse, GraphListResponse, Pagination types
- `dashboard/src/components/ui/` — 14 new shadcn/ui components: input, textarea, label, form, alert, alert-dialog, breadcrumb, tabs, switch, select, tooltip, scroll-area, collapsible, separator
- `dashboard/src/features/graphs/schemas/graph-form.schema.ts` — Zod schema + formToGraphInput
- `dashboard/src/features/graphs/lib/definition-templates.ts` — 4 templates (empty, llm_call, react_agent, router_handlers)
- `dashboard/src/features/graphs/hooks/` — useGraphs, useGraph, useCreateGraph, useUpdateGraph, useArchiveGraph (all using direct fetch() + useAuth)
- `dashboard/src/features/graphs/components/` — GraphStatusBadge, NodePatternBadge, EdgeTypeBadge, GraphTable, GraphDefinitionViewer, GraphForm, GraphArchiveDialog
- `dashboard/src/features/graphs/pages/` — GraphsPage, GraphDetailPage, GraphCreatePage, GraphEditPage
- `dashboard/src/features/graphs/index.ts` — barrel export
- `dashboard/src/App.tsx` — added routes: /graphs/new, /graphs/:id, /graphs/:id/edit
- `dashboard/src/test/setup.ts` — added jsdom polyfills for Radix UI (hasPointerCapture, setPointerCapture, releasePointerCapture, scrollIntoView, ResizeObserver)
- `dashboard/scripts/smoke.sh` — extended with graph bundle presence checks
- `docs/sprints/SPRINT-006-dashboard-feature-grafos.md` — manual E2E procedure added

**Deleted:**
- `dashboard/src/pages/GraphsPage.tsx` (replaced by features/graphs/pages/GraphsPage.tsx)
- `dashboard/src/pages/__tests__/GraphsPage.test.tsx` (replaced by feature tests)

**Tests:** 31 passing (5 PKCE, 5 AuthProvider, 2 useArchiveGraph, 8 GraphDetailPage, 6 GraphForm, 5 GraphCreatePage).

**Key decisions:**
- All mutation hooks use direct `fetch()` calls (not openapi-fetch) for testability with MSW v2.
- `zodResolver(schema) as any` cast required due to TS strict incompatibility with react-hook-form generics.
- `userEvent.type()` cannot type `{` — use `fireEvent.change()` for JSON textarea content in tests.
- `<Toaster />` must be in the test wrapper for sonner toasts to appear in RTL tests.
- 422 field errors rendered as inline `<Alert>` (not toast) to satisfy acceptance criteria.
- Smoke bundle check greps for JSX string literals (survive minification) rather than component names.

**Status:** completed

---

## 2026-05-08 — SPRINT-007: Event Bus Adapter — Valkey Streams

**Branch:** claude/sleepy-hamilton-oSncI
**Status:** completed

**What was done:**
- TODO #1: Updated `specs/asyncapi.yaml` with 7 CloudEvents 1.0 channels for orchestration (graph.execution.requested, node.execution.started/completed/failed, graph.execution.completed/failed, dago.dlq). Added Valkey stream bindings and consumer group definitions per ADR-011.
- TODO #2: Created `libs/domain/events.go` — pure domain types `Event`, `EventAuth`, and 7 event type constants. No infrastructure imports.
- TODO #3: Created `libs/ports/eventbus.go` — `EventPublisher`, `EventConsumer`, `EventHandler` interfaces; `PublishOptions`, `ConsumeOptions` types.
- TODO #4: Wrote 6 integration tests in `adapters/eventbus/valkey/integration_test.go` (build tag: `integration`) using Testcontainers (`valkey/valkey:8`).
- TODO #5: Implemented `adapters/eventbus/valkey/envelope.go` — CloudEvents JSON serialization/deserialization via single `envelope` field.
- TODO #6: Implemented `adapters/eventbus/valkey/publisher.go` — `ValkeyPublisher` with XADD and idempotent XGROUP CREATE MKSTREAM.
- TODO #7+8: Implemented `adapters/eventbus/valkey/consumer.go` — `ValkeyConsumer` with XREADGROUP, XACK, DLQ after MaxRetries, and XAUTOCLAIM for pending recovery.
- TODO #9: Added `github.com/valkey-io/valkey-go v1.0.74` and `github.com/testcontainers/testcontainers-go v0.42.0` to `go.mod`.
- TODO #10: Added `make test-integration-eventbus` target to Makefile.
- TODO #11: Updated `docs/index.md` (SPRINT-007 → completed, AsyncAPI → implemented, Event Bus adapter → implemented) and this log.

**Files created/modified:**
- `specs/asyncapi.yaml` — 7 orchestration channels
- `libs/domain/events.go` — new
- `libs/ports/eventbus.go` — new
- `adapters/eventbus/valkey/envelope.go` — new
- `adapters/eventbus/valkey/publisher.go` — new
- `adapters/eventbus/valkey/consumer.go` — new
- `adapters/eventbus/valkey/integration_test.go` — new (integration tag)
- `go.mod` / `go.sum` — valkey-go + testcontainers added
- `.env.example` — Valkey variables appended
- `Makefile` — test-integration-eventbus target added
- `docs/index.md` — SPRINT-007 completed, AsyncAPI implemented, Event Bus adapter implemented
- `docs/log.md` — this entry

**Key decisions:**
- `RecoverPending` accepts an explicit `EventHandler` parameter (matches test expectations and makes the API clear).
- `PendingCount(ctx, stream, group)` is a concrete method (not in port interface) for test verification.
- `moveToDLQ` preserves original event ID + auth context; `dago.dlq` payload contains `original_id`, `original_type`, `original_source`, `original_stream`, `retry_count`.
- BUSYGROUP errors on XGROUP CREATE are silently ignored (idempotent setup).

**Status:** completed

---

## 2026-05-09 — SPRINT-008: LLM Adapter — LLMClient Port, Anthropic and Ollama/Mixtral

**Sprint:** SPRINT-008 | **Branch:** claude/kind-hawking-3WrBD

**Objective:** Implement the `LLMClient` port and two concrete adapters (Anthropic + Ollama/Mixtral) plus a deterministic fake for tests.

**Operations performed:**
- TODO #1: Added `ErrUnauthorized`, `ErrRateLimited`, `ErrProviderUnavailable` to `libs/domain/errors.go`.
- TODO #2: Created `libs/ports/llm.go` — `LLMClient` interface with `Message`, `ToolDefinition`, `LLMRequest`, `LLMResponse`, `ToolUse` types.
- TODO #9/#12: Added `github.com/anthropics/anthropic-sdk-go v1.41.0` and `github.com/sashabaranov/go-openai v1.41.2` to `go.mod`; appended Anthropic + Ollama vars to `.env.example`.
- TODO #3: Wrote 7 Red tests in `adapters/llm/anthropic/client_test.go` (httptest.NewServer mocks).
- TODO #4: Wrote 1 Red test in `adapters/llm/fake/client_test.go` (FIFO queue + call registry).
- TODO #13: Wrote 6 Red tests in `adapters/llm/ollama/client_test.go` (httptest.NewServer mocks).
- TODO #5: Implemented `adapters/llm/fake/client.go` — `FakeLLMClient` with FIFO Responses + Calls registry.
- TODO #6: Implemented `adapters/llm/anthropic/convert.go` — `toAnthropicMessages`, `toAnthropicTools`, `fromAnthropicResponse`.
- TODO #7: Implemented `adapters/llm/anthropic/errors.go` — `mapAnthropicError` mapping HTTP codes to domain errors.
- TODO #8: Implemented `adapters/llm/anthropic/client.go` — `AnthropicClient` with `NewAnthropicClient` and `Complete`.
- TODO #14: Implemented `adapters/llm/ollama/convert.go` — `toOpenAIMessages`, `toOpenAITools`, `fromOpenAIResponse`, `ConvertFinishReason`.
- TODO #15: Implemented `adapters/llm/ollama/client.go` + `errors.go` — `OllamaClient` using go-openai SDK.
- TODO #10/#16: Added `make test-llm` target to `Makefile`; added as dependency of `test`; `./adapters/llm/...` covers all 3 sub-packages.
- TODO #11: Updated `docs/index.md` (SPRINT-008 → completed, LLM adapters → implemented) and this log.

**Files created/modified:**
- `libs/domain/errors.go` — 3 new errors added
- `libs/ports/llm.go` — new
- `adapters/llm/anthropic/client.go` — new
- `adapters/llm/anthropic/convert.go` — new
- `adapters/llm/anthropic/errors.go` — new
- `adapters/llm/anthropic/client_test.go` — new (7 tests)
- `adapters/llm/fake/client.go` — new
- `adapters/llm/fake/client_test.go` — new (1 test)
- `adapters/llm/ollama/client.go` — new
- `adapters/llm/ollama/convert.go` — new
- `adapters/llm/ollama/errors.go` — new
- `adapters/llm/ollama/client_test.go` — new (6 tests)
- `go.mod` / `go.sum` — anthropic-sdk-go + go-openai added
- `.env.example` — Anthropic + Ollama variables appended
- `Makefile` — test-llm target added
- `docs/index.md` — SPRINT-008 completed, LLM adapters implemented
- `docs/log.md` — this entry

**Key decisions:**
- `ConvertFinishReason` is exported so `TestOllamaConvertFinishReason` can call it directly without round-tripping through the HTTP mock.
- `mapOllamaError` handles both `*openai.APIError` and `*openai.RequestError` for HTTP 500 → `ErrProviderUnavailable` (go-openai returns RequestError when the response body is not the standard OpenAI error format).
- `NewAnthropicClient` passes `option.WithoutEnvironmentDefaults()` so tests never pick up real API keys from env.
- `NewOllamaClient` never returns an error because BaseURL always has a valid default.

**Test results:** `make test-llm` → 14 tests (7 Anthropic + 1 Fake + 6 Ollama), all pass, no network or real credentials needed.

**Status:** completed

---

## 2026-05-09 — SPRINT-009: Executor — llm_call pattern handler

**Sprint:** SPRINT-009
**Branch:** claude/adoring-mccarthy-FSwgJ
**Status:** completed

**Objective:** Implement the executor service as an event worker for the `llm_call` pattern: consumes `node.execute.requested`, builds the LLM request with input mapping, calls `LLMClient.Complete`, applies output mapping, and publishes `node.executed` or `node.execute.failed`.

**Files created/modified:**
- `specs/asyncapi.yaml` — 3 new channels (nodeExecuteRequested, nodeExecuted, nodeExecuteFailed), 3 operations, 3 data schemas (NodeExecuteRequestedData, NodeExecutedData, NodeExecuteFailedData)
- `specs/patterns/nodes/llm_call.json` — enriched descriptions for defaults and supported paths
- `libs/domain/events.go` — StreamNodeExecuteRequested, StreamNodeExecuted, StreamNodeExecuteFailed, EventTypeNodeExecute* constants
- `adapters/llm/fake/client.go` — extended FakeLLMClient with Errors []error field for test error injection
- `services/executor/internal/mapping/input.go` — new: ApplyInputMapping (state.variables.*, state.messages[-1].content)
- `services/executor/internal/mapping/input_test.go` — new: 6 unit tests
- `services/executor/internal/mapping/output.go` — new: ApplyOutputMapping (output.content, output.stop_reason → state.variables.*)
- `services/executor/internal/mapping/output_test.go` — new: 6 unit tests
- `services/executor/internal/handler/node_handler.go` — new: NodeHandler interface + data types
- `services/executor/internal/handler/llm_call.go` — new: LLMCallHandler.Handle (TDD Green)
- `services/executor/internal/handler/llm_call_test.go` — new: 7 unit tests
- `services/executor/internal/handler/dispatcher.go` — new: Dispatcher.Dispatch
- `services/executor/internal/consumer/node_execute.go` — new: NodeExecuteConsumer.Run (ACK/NACK by retryability)
- `services/executor/internal/consumer/node_execute_test.go` — new: 1 integration test (build tag: integration)
- `services/executor/cmd/main.go` — implemented: env config, LLM provider selection, wiring, graceful shutdown
- `Makefile` — test-executor target added
- `.env.example` — Executor variables appended
- `docs/index.md` — SPRINT-009 completed, executor partial, asyncapi updated
- `docs/log.md` — this entry

**Key decisions:**
- FakeLLMClient extended with `Errors []error` (drained before Responses) so handler tests can inject ErrRateLimited, ErrProviderUnavailable, ErrUnauthorized without network.
- Consumer ACKs non-retryable errors (failure event already published) and NACKs retryable ones (ErrRateLimited, ErrProviderUnavailable) to leave them in the Valkey PEL for retry.
- `fakePublisher` defined as a package-private type in `llm_call_test.go`; not exported until a second handler requires it.
- input/output mapping packages are pure domain code — zero infrastructure imports.
- Dispatcher returns an error for unsupported patterns; consumer ACKs these after logging to avoid queue blockage.

**Test results:** 12 mapping tests + 7 handler tests = 19 unit tests, all pass, no network or real credentials needed.

**Status:** completed

---

## [2026-05-09] sprint | SPRINT-010: Orchestrator state machine — Submit, validate, execute, transition, complete

**Scope:** Connect orchestrator with Valkey event bus: validate graph on submission,
publish `node.execute.requested` for the entry node, consume `node.executed` /
`node.execute.failed`, update state and transition until completion or failure.
Only sequential edges supported. Per-node timeout excluded (documented as future TODO).

**TODOs completed:** 17/17 (spec ×2, domain ×1, port ×1, test ×4, impl ×7, infra ×1, docs ×1)

**Status:** completed

---

## [2026-05-09] sprint | SPRINT-010: completed

**Result:** All 17 TODOs implemented. 13 test suites pass.
`go build ./...`, `go vet ./...`, `golangci-lint run ./...` all exit 0.

**Artifacts created:**
- `specs/asyncapi.yaml` — 5 orchestrator operations + GraphCompletedData + GraphFailedData schemas
- `specs/paths/executions.yaml` — 422 GRAPH_VALIDATION_ERROR documented
- `libs/domain/errors.go` — ErrGraphValidation, ErrRetryable
- `libs/domain/graph.go` — GraphDefinition, NodeDefinition, EdgeDefinition
- `libs/domain/events.go` — StreamGraphCompleted, StreamGraphFailed constants
- `libs/ports/storage.go` — UpdateExecution in ExecutionRepository interface
- `adapters/storage/graph_repo.go` — UpdateExecution implemented
- `tests/testutil/fakes/` — UpdateExecution in InMemoryExecutionRepository; new InMemoryPublisher
- `services/orchestrator/internal/statemachine/` — graph_validator.go, traversal.go, execution_sm.go + tests
- `services/orchestrator/internal/consumer/node_result.go` — NodeResultConsumer
- `services/orchestrator/internal/usecase/execution.go` — StartExecution extended
- `services/orchestrator/internal/handler/errors.go` — 422 GRAPH_VALIDATION_ERROR
- `services/orchestrator/cmd/main.go` — full wiring with graceful shutdown
- `go.mod` — github.com/dominikbraun/graph v0.23.0 (Apache-2.0)
- `docs/views/process/execution_state.md` — execution state diagram
- `docs/index.md`, `docs/log.md` — updated

**Key decisions:**
- Only `sequential` edges in this sprint; other types → ErrGraphValidation (documented as known limitation).
- Per-node timeout excluded; context is propagated but no per-node deadline (documented as future TODO).
- ErrRetryable sentinel in libs/domain/ keeps consumer NACK logic adapter-independent.
- CanTransitionTo prevents double transitions (Valkey at-least-once delivery).
- StartExecution goes directly to `running` (not `pending`) since first event publication is synchronous.

**Test results:** 13 test suites, all pass (10 new statemachine/usecase/handler tests + 3 integration stubs).

**Verifications:**
- `go build ./...` → 0 errors
- `go vet ./...` → 0 issues
- `golangci-lint run ./...` → 0 issues
- `go test ./...` → 13/13 suites pass

---

## [2026-05-09] sprint | SPRINT-011: docker-compose full-stack containerization

**Planned artifacts:**
- `docs/sprints/SPRINT-011-docker-compose.md`

**Scope:** Working `docker-compose.yml` for the complete dago stack (2 infra + 8 Go
services + 1 dashboard). Single `Dockerfile.service` with `SERVICE` build arg.
Dashboard `Dockerfile` + nginx SPA config. `.env.example` with all variables.
Atlas init-container for migrations. 4 compose profiles. `/health` endpoints on
all Gin services. Makefile targets. Smoke test script. Runbook.

**TODOs:** 10 (audit ×1, impl ×5, test ×1, docs ×3).

**Status:** completed

---

## [2026-05-09] sprint | SPRINT-011: completed

**Result:** All 10 TODOs implemented.

**Artifacts created:**
- `Dockerfile.service` — shared multi-stage Go build (golang:1.25-alpine → alpine:3.20)
- `dashboard/Dockerfile` — multi-stage React 19 build (node:20-alpine → nginx:1.27-alpine)
- `dashboard/nginx.conf` — SPA routing (`try_files $uri /index.html`)
- `.env.example` — all 25+ env vars grouped by service with defaults and descriptions
- `docker-compose.yml` — extended with 11 services across 4 profiles (infra/backend/frontend/all)
- `services/orchestrator/internal/router/router.go` — added `GET /health`
- `services/auth-server/internal/router/router.go` — added `GET /health`
- `services/catalog/cmd/main.go` — minimal Gin server with `GET /health` on :8082
- `services/mcp-registry/cmd/main.go` — minimal Gin server with `GET /health` on :8083
- `services/agent-registry/cmd/main.go` — minimal Gin server with `GET /health` on :8084
- `Makefile` — 5 compose targets: compose-infra, compose-up, compose-down, compose-logs, compose-ps
- `scripts/smoke-test-compose.sh` — full stack smoke test (start → health checks → valkey ping → teardown)
- `docs/deploy/docker-compose-runbook.md` — prerequisites, quick-start, profiles, secrets, troubleshooting
- `docs/index.md`, `docs/log.md` — updated

**Key decisions:**
- Alpine:3.20 runtime (not distroless/static) to enable `wget` health checks in docker-compose.
- Stub services (catalog, mcp-registry, agent-registry) get minimal Gin /health so health checks pass.
- Atlas migrate init-container ensures schema migrations run before orchestrator boots.
- Profiles: infra/backend/frontend/all — postgres+valkey included in both infra and backend profiles.
