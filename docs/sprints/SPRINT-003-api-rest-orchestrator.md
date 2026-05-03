# SPRINT-003: Orchestrator REST API — CRUD graphs and executions

## Metadata

- **Start date:** 2026-04-29 (after completing SPRINT-002)
- **Estimated end date:** 2026-05-01
- **Status:** planned
- **ADRs applied:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-006, ADR-010, ADR-016
- **Affected specs:**
  - `specs/openapi.yaml` — add `$ref` to the new paths
  - `specs/paths/graphs.yaml` — new (6 graph endpoints)
  - `specs/paths/executions.yaml` — new (1 execution endpoint)
  - `specs/schemas/graph.yaml` — new
  - `specs/schemas/execution.yaml` — new
- **Planning agent:** planner
- **Reviewed by:** pending
- **Blocks:** SPRINT-004 (events), SPRINT-dashboard-001 (frontend)
- **Blocked by:** SPRINT-001 (go.mod, Gin), SPRINT-002 (Ent schemas)

## Sprint Objective

Implement the orchestrator REST API for the dashboard: 5 graph CRUD endpoints
and 1 execution start endpoint, with the complete flow
Spec → Tests → Domain → Use Cases → Handlers → Adapter.

At the end there is a functional Gin API that complies with the OpenAPI spec, with
contract tests, use case unit tests, and handler unit tests. The
endpoint POST /executions creates records in PostgreSQL but does not yet publish
events (added in SPRINT-004).

## Scope

### Included

**OpenAPI Spec (source of truth — ADR-010 rule 6):**
- `specs/paths/graphs.yaml` with the 6 graph endpoints.
- `specs/paths/executions.yaml` with the execution endpoint.
- `specs/schemas/graph.yaml`, `specs/schemas/execution.yaml`.
- `specs/openapi.yaml` updated with `$ref` to the new paths.

**Domain (`libs/domain/`):**
- `libs/domain/graph.go` — `Graph`, `GraphStatus` types, constants.
- `libs/domain/execution.go` — `Execution`, `ExecutionStatus` types.
- `libs/domain/errors.go` — domain errors (`ErrNotFound`,
  `ErrConflict`, `ErrValidation`, `ErrInvalidGraphStatus`).

**Ports (`libs/ports/`):**
- `libs/ports/storage.go` — `GraphRepository` and
  `ExecutionRepository` interfaces.

**Use cases (`services/orchestrator/internal/usecase/`):**
- `CreateGraph`, `GetGraph`, `ListGraphs`, `UpdateGraph`, `ArchiveGraph`.
- `StartExecution`.

**Gin handlers (`services/orchestrator/internal/handler/`):**
- `GraphHandler` — 5 endpoints.
- `ExecutionHandler` — 1 endpoint.
- `mapDomainError()` — translates domain errors to HTTP (ADR-006).

**Router (`services/orchestrator/internal/router/`):**
- `NewRouter()` — composes middlewares and registers routes under `/api/v1/`.

**Storage adapter (`adapters/storage/`):**
- `EntGraphRepository` — implements `GraphRepository` with Ent.
- `EntExecutionRepository` — implements `ExecutionRepository` with Ent.

**Tests:**
- `tests/contract/graphs_contract_test.go` — validates responses against
  the OpenAPI spec with `go-openapi`.
- `tests/unit/usecase/graph_usecase_test.go` — use cases with fakes.
- `tests/unit/usecase/execution_usecase_test.go` — same.
- `tests/unit/handler/graph_handler_test.go` — handlers with mock
  `httptest.NewRecorder`.
- `tests/unit/handler/execution_handler_test.go` — same.

**Server entry in `services/orchestrator/cmd/main.go`:**
- Real `main()` that initializes Ent, repositories, use cases, router
  and starts the server on the port configured by env var.

### Excluded

- JWT auth middleware (SPRINT-auth): endpoints do not require a token
  in this sprint. Middleware is added later.
- Valkey event publishing when creating Execution (SPRINT-004).
- Real graph execution (SPRINT-004 onwards).
- WebSocket AG-UI (SPRINT-005).
- GET /executions and GET /executions/{id} endpoints (SPRINT-004).
- PATCH /graphs/{id}/status endpoint for status transitions.
- Semantic validation of the graph definition against ADR-016
  (node patterns, edges) — implemented in SPRINT-004 when
  real execution starts.
- TypeScript client generation from OpenAPI (SPRINT-dashboard-001).

## Dependencies

- **SPRINT-001 completed:** `go.mod` with Gin, directory structure,
  `services/orchestrator/cmd/main.go` stub.
- **SPRINT-002 completed:** Ent schemas for Graph, Node, Execution
  generated and migrated in PostgreSQL.

## Behavior Contracts

### C1 — `POST /api/v1/graphs` — successful creation

```
Given: Valid JSON body with name, semver version ("1.0.0"), entry_node and definition
When: POST /api/v1/graphs with Content-Type application/json
Then: HTTP 201, body with GraphResponse schema
      id is UUID v4, status = "draft"
      created_at and updated_at present in RFC3339
```

### C2 — `POST /api/v1/graphs` — invalid version

```
Given: JSON body with version = "v1" (not semver)
When: POST /api/v1/graphs
Then: HTTP 422, ErrorResponse body with code = "VALIDATION_ERROR"
      No row is persisted
```

### C3 — `DELETE /api/v1/graphs/:id` — archived (not physically deleted)

```
Given: Graph with status="draft" and no active executions
When: DELETE /api/v1/graphs/:id
Then: HTTP 204 without body
      The graph status in DB changes to "archived"
      The row is not physically deleted (soft delete)
```

```
Given: Graph with executions in status="running"
When: DELETE /api/v1/graphs/:id
Then: HTTP 409, ErrorResponse with code = "GRAPH_HAS_ACTIVE_EXECUTIONS"
```

## API Design

### Endpoints

| Method | Path | Handler | Status codes |
|--------|------|---------|--------------|
| `POST` | `/api/v1/graphs` | `CreateGraph` | 201, 400, 409, 422, 500 |
| `GET` | `/api/v1/graphs` | `ListGraphs` | 200, 500 |
| `GET` | `/api/v1/graphs/:id` | `GetGraph` | 200, 404, 500 |
| `PUT` | `/api/v1/graphs/:id` | `UpdateGraph` | 200, 400, 404, 409, 422, 500 |
| `DELETE` | `/api/v1/graphs/:id` | `ArchiveGraph` | 204, 404, 409, 500 |
| `POST` | `/api/v1/executions` | `StartExecution` | 201, 400, 404, 422, 500 |

### Design decisions

**DELETE archives instead of physically deleting.** Sets
`status=archived`. Returns 409 if the graph has active executions
(`status=running`). Reason: preserve execution history
associated with the graph (ADR-015).

**PUT only updates graphs in `draft` status.** Returns 409 if the
graph is `active` or `archived`. To activate a graph there is a
dedicated endpoint (out of scope for this sprint).

**POST /executions creates the Execution in `pending`.** Does not start
real execution or publish an event (SPRINT-004). The response includes
the execution ID so the client can subscribe via
WebSocket (SPRINT-005).

### API Schemas

**GraphInput** (creation and update request):
```yaml
type: object
required: [name, version, entry_node, definition]
properties:
  name:
    type: string
    maxLength: 255
  version:
    type: string
    pattern: '^\d+\.\d+\.\d+$'
  description:
    type: string
  entry_node:
    type: string
    maxLength: 255
  definition:
    type: object          # free graph JSON (nodes + edges)
  memory_config:
    type: object
    properties:
      semantic_search: {type: boolean}
      episode_context: {type: integer, minimum: 0}
```

**GraphResponse** (individual response):
```yaml
type: object
required: [id, name, version, entry_node, definition, status, created_at, updated_at]
properties:
  id:       {type: string, format: uuid}
  name:     {type: string}
  version:  {type: string}
  description: {type: string}
  entry_node: {type: string}
  definition: {type: object}
  memory_config: {type: object}
  status:   {type: string, enum: [draft, active, archived]}
  created_at: {type: string, format: date-time}
  updated_at: {type: string, format: date-time}
```

**GraphListResponse** (paginated response):
```yaml
type: object
required: [items, pagination]
properties:
  items:
    type: array
    items: {$ref: '#/components/schemas/GraphResponse'}
  pagination: {$ref: '#/components/schemas/Pagination'}
```

**ExecutionInput** (start request):
```yaml
type: object
required: [graph_id]
properties:
  graph_id:
    type: string
    format: uuid
  variables:
    type: object     # initial graph variables
    default: {}
```

**ExecutionResponse** (response):
```yaml
type: object
required: [id, graph_id, status, variables, messages, node_results, created_at, updated_at]
properties:
  id:            {type: string, format: uuid}
  graph_id:      {type: string, format: uuid}
  status:        {type: string, enum: [pending, running, completed, failed, cancelled, interrupted]}
  current_node:  {type: string}
  variables:     {type: object}
  messages:      {type: array, items: {type: object}}
  node_results:  {type: object}
  error:         {type: string}
  started_at:    {type: string, format: date-time}
  completed_at:  {type: string, format: date-time}
  created_at:    {type: string, format: date-time}
  updated_at:    {type: string, format: date-time}
```

**ErrorResponse** (already in openapi.yaml):
```yaml
type: object
required: [code, message]
properties:
  code:    {type: string}   # GRAPH_NOT_FOUND, GRAPH_NOT_DRAFT, etc.
  message: {type: string}
  details: {type: array, items: {type: object}}
```

### Domain error → HTTP mapping

| Domain error | HTTP | `code` in ErrorResponse |
|-------------|------|-------------------------|
| `ErrNotFound` | 404 | `GRAPH_NOT_FOUND`, `EXECUTION_NOT_FOUND` |
| `ErrConflict` | 409 | `GRAPH_DUPLICATE_VERSION`, `GRAPH_NOT_DRAFT`, `GRAPH_HAS_ACTIVE_EXECUTIONS` |
| `ErrValidation` | 422 | `VALIDATION_ERROR` |
| `ErrInvalidGraphStatus` | 409 | `GRAPH_NOT_FOUND_OR_ACTIVE` |
| Other error | 500 | `INTERNAL_ERROR` |

## TODOs

### 1. [spec] Write `specs/schemas/graph.yaml` and `specs/schemas/execution.yaml`

- **Agente:** @developer
- **Description:** Create reusable OpenAPI schemas for
  `GraphInput`, `GraphResponse`, `GraphListResponse`,
  `ExecutionInput`, `ExecutionResponse`. Follow the design in the
  previous section and the error format already defined in `openapi.yaml`.

  Each schema goes in its own file in `specs/schemas/` referenced
  from `specs/openapi.yaml` with `$ref`.

- **Acceptance criteria:** `swagger-cli validate specs/openapi.yaml`
  (or equivalent) passes without errors. Each schema has complete `required`,
  `properties` and `description`.
- **Dependencies:** none
- **Commit:** `spec(openapi): add Graph and Execution schemas [SPRINT-003 #1]`

### 2. [spec] Write `specs/paths/graphs.yaml` and `specs/paths/executions.yaml`

- **Agente:** @developer
- **Description:** Create OpenAPI path items for the 6 endpoints.
  Each endpoint documents: parameters, request body, responses (all
  status codes), references to the schemas from TODO #1 and to
  the existing `ErrorResponse`.

  Add in `specs/openapi.yaml` under `paths:`:
  ```yaml
  paths:
    /graphs:
      $ref: './paths/graphs.yaml#/graphs'
    /graphs/{id}:
      $ref: './paths/graphs.yaml#/graphsId'
    /executions:
      $ref: './paths/executions.yaml#/executions'
  ```

  Pagination parameters for `GET /graphs`:
  - `page` (integer, default 1)
  - `per_page` (integer, default 20, max 100)
  - `status` (string, optional, filter by status)

- **Acceptance criteria:** `swagger-cli validate specs/openapi.yaml`
  passes. All endpoints have all their status codes documented
  with request/response examples.
- **Dependencies:** #1
- **Commit:** `spec(openapi): add graphs and executions path items [SPRINT-003 #2]`

### 3. [domain] Implement domain types in `libs/domain/`

- **Agente:** @developer
- **Description:** Create pure domain types, without Ent or Gin dependencies.
  They follow ADR-001 (domain is the center) and ADR-007
  rule 7 (Ent types do not leave the adapter).

  **`libs/domain/graph.go`:**
  ```go
  // GraphStatus represents the status of a graph.
  type GraphStatus string

  const (
      GraphStatusDraft    GraphStatus = "draft"
      GraphStatusActive   GraphStatus = "active"
      GraphStatusArchived GraphStatus = "archived"
  )

  // Graph is the definition of an execution graph (not an instance).
  type Graph struct {
      ID           uuid.UUID
      Name         string
      Version      string
      Description  string
      EntryNode    string
      Definition   json.RawMessage
      MemoryConfig json.RawMessage
      Status       GraphStatus
      CreatedAt    time.Time
      UpdatedAt    time.Time
  }

  // IsDraft returns true if the graph can be modified.
  func (g *Graph) IsDraft() bool { return g.Status == GraphStatusDraft }

  // IsActive returns true if the graph can start executions.
  func (g *Graph) IsActive() bool { return g.Status == GraphStatusActive }
  ```

  **`libs/domain/execution.go`:**
  ```go
  // ExecutionStatus represents the status of an execution.
  type ExecutionStatus string

  const (
      ExecutionStatusPending     ExecutionStatus = "pending"
      ExecutionStatusRunning     ExecutionStatus = "running"
      ExecutionStatusCompleted   ExecutionStatus = "completed"
      ExecutionStatusFailed      ExecutionStatus = "failed"
      ExecutionStatusCancelled   ExecutionStatus = "cancelled"
      ExecutionStatusInterrupted ExecutionStatus = "interrupted"
  )

  // Execution is an execution instance of a graph.
  type Execution struct {
      ID          uuid.UUID
      GraphID     uuid.UUID
      Status      ExecutionStatus
      CurrentNode string
      Variables   json.RawMessage
      Messages    json.RawMessage
      NodeResults json.RawMessage
      Error       string
      StartedAt   *time.Time
      CompletedAt *time.Time
      CreatedAt   time.Time
      UpdatedAt   time.Time
  }
  ```

  **`libs/domain/errors.go`:**
  ```go
  var (
      ErrNotFound            = errors.New("not found")
      ErrConflict            = errors.New("conflict")
      ErrValidation          = errors.New("validation error")
      ErrInvalidGraphStatus  = errors.New("invalid graph status for operation")
  )
  ```

- **Acceptance criteria:** `go build ./libs/domain/...` compiles.
  The package does not import anything from `entgo.io`, `gin`, `go-redis`.
- **Dependencies:** none (can be done in parallel with #1, #2)
- **Commit:** `feat(domain): add Graph and Execution domain types [SPRINT-003 #3]`

### 4. [domain] Implement ports in `libs/ports/storage.go`

- **Agente:** @developer
- **Description:** Define the repository interfaces that the domain
  needs. Methods use exclusively types from `libs/domain/`,
  never Ent types.

  ```go
  // libs/ports/storage.go

  // GraphRepository manages graph persistence.
  type GraphRepository interface {
      Create(ctx context.Context, g *domain.Graph) (*domain.Graph, error)
      FindByID(ctx context.Context, id uuid.UUID) (*domain.Graph, error)
      List(ctx context.Context, opts ListOptions) ([]*domain.Graph, int, error)
      Update(ctx context.Context, g *domain.Graph) (*domain.Graph, error)
      Archive(ctx context.Context, id uuid.UUID) error
  }

  // ExecutionRepository manages execution persistence.
  type ExecutionRepository interface {
      Create(ctx context.Context, e *domain.Execution) (*domain.Execution, error)
      FindByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error)
      CountActiveByGraph(ctx context.Context, graphID uuid.UUID) (int, error)
  }

  // ListOptions configures pagination and filters for List.
  type ListOptions struct {
      Page    int
      PerPage int
      Status  string  // optional
  }
  ```

- **Acceptance criteria:** `go build ./libs/ports/...` compiles.
  The package only imports `libs/domain/` and stdlib.
- **Dependencies:** #3
- **Commit:** `feat(ports): add GraphRepository and ExecutionRepository interfaces [SPRINT-003 #4]`

### 5. [test] Contract tests (Red) — `tests/contract/`

- **Agente:** @qa
- **Description:** Create tests that bring up the real test server
  (with in-memory fake repositories) and verify that responses
  comply with the OpenAPI spec. Use `build tag: contract`.

  Suggested library: `github.com/deepmap/oapi-codegen` or a manual
  solution that validates the response JSON against the schemas from
  `specs/schemas/`.

  Tests to implement:

  ```go
  //go:build contract

  // TestCreateGraphContract verifies that POST /api/v1/graphs returns
  // 201 with a body that complies with GraphResponse.
  func TestCreateGraphContract(t *testing.T)

  // TestCreateGraphValidationContract verifies that an invalid body
  // returns 422 with ErrorResponse.
  func TestCreateGraphValidationContract(t *testing.T)

  // TestListGraphsContract verifies that GET /api/v1/graphs returns
  // 200 with GraphListResponse (items + pagination).
  func TestListGraphsContract(t *testing.T)

  // TestGetGraphContract verifies that GET /api/v1/graphs/:id returns
  // 200 with GraphResponse or 404 with ErrorResponse.
  func TestGetGraphContract(t *testing.T)

  // TestUpdateGraphContract verifies PUT /api/v1/graphs/:id.
  func TestUpdateGraphContract(t *testing.T)

  // TestDeleteGraphContract verifies DELETE /api/v1/graphs/:id → 204.
  func TestDeleteGraphContract(t *testing.T)

  // TestStartExecutionContract verifies POST /api/v1/executions → 201.
  func TestStartExecutionContract(t *testing.T)

  // TestStartExecutionGraphNotFoundContract verifies 404 if the graph
  // does not exist.
  func TestStartExecutionGraphNotFoundContract(t *testing.T)
  ```

- **Acceptance criteria:** `make test-contract` (with fake repositories)
  finishes with all PASS. Tests fail if the response does not
  include a `required` field from the schema.
- **Dependencies:** #2, #3, #4 (to compile the fakes)
- **Commit:** `test(contract): add contract tests for graphs and executions API [SPRINT-003 #5]`

### 6. [test] Use case unit tests (Red) — `tests/unit/usecase/`

- **Agente:** @qa
- **Description:** Tests of the business logic of use cases,
  using in-memory fakes that implement the ports from TODO #4.

  **`graph_usecase_test.go`:**
  ```go
  // TestCreateGraphSuccess verifies that CreateGraph persists and returns
  // the graph with status=draft.
  func TestCreateGraphSuccess(t *testing.T)

  // TestCreateGraphDuplicateVersion verifies that creating a duplicate
  // (name, version) returns ErrConflict.
  func TestCreateGraphDuplicateVersion(t *testing.T)

  // TestUpdateGraphOnlyDraft verifies that updating an active graph
  // returns ErrInvalidGraphStatus.
  func TestUpdateGraphOnlyDraft(t *testing.T)

  // TestArchiveGraphWithActiveExecution verifies that archiving a graph
  // with active executions returns ErrConflict.
  func TestArchiveGraphWithActiveExecution(t *testing.T)
  ```

  **`execution_usecase_test.go`:**
  ```go
  // TestStartExecutionSuccess verifies that StartExecution creates
  // Execution with status=pending and empty variables by default.
  func TestStartExecutionSuccess(t *testing.T)

  // TestStartExecutionGraphNotFound verifies that passing a non-existent
  // graph_id returns ErrNotFound.
  func TestStartExecutionGraphNotFound(t *testing.T)

  // TestStartExecutionWithInitialVariables verifies that the initial
  // variables are persisted in the Execution.
  func TestStartExecutionWithInitialVariables(t *testing.T)
  ```

- **Acceptance criteria:** `go test ./tests/unit/usecase/...` in
  RED (fails because the use cases do not exist yet). When the
  use cases are implemented in TODO #9, they turn GREEN.
- **Dependencies:** #4
- **Commit:** `test(unit): add use case unit tests for Graph and Execution [SPRINT-003 #6]`

### 7. [test] Handler unit tests (Red) — `tests/unit/handler/`

- **Agente:** @qa
- **Description:** Tests of the HTTP layer with `httptest.NewRecorder`.
  Use cases are mocked with interfaces.

  **`graph_handler_test.go`:**
  ```go
  // TestGraphHandlerCreate verifies that the handler correctly binds
  // the JSON, calls the use case and returns 201.
  func TestGraphHandlerCreate(t *testing.T)

  // TestGraphHandlerCreateInvalidBody verifies that invalid JSON
  // returns 400.
  func TestGraphHandlerCreateInvalidBody(t *testing.T)

  // TestGraphHandlerGetNotFound verifies that ErrNotFound from the use
  // case translates to 404 with ErrorResponse.
  func TestGraphHandlerGetNotFound(t *testing.T)

  // TestGraphHandlerDelete verifies that DELETE returns 204.
  func TestGraphHandlerDelete(t *testing.T)
  ```

  **`execution_handler_test.go`:**
  ```go
  // TestExecutionHandlerStart verifies that POST /executions returns 201.
  func TestExecutionHandlerStart(t *testing.T)

  // TestExecutionHandlerStartGraphNotFound verifies 404.
  func TestExecutionHandlerStartGraphNotFound(t *testing.T)
  ```

  `mapDomainError()` is tested indirectly through these tests.

- **Acceptance criteria:** `go test ./tests/unit/handler/...` in
  RED. When handlers are implemented in TODO #10, they turn GREEN.
- **Dependencies:** #3, #4
- **Commit:** `test(unit): add handler unit tests for Graph and Execution [SPRINT-003 #7]`

### 8. [impl] Implement in-memory fakes for tests

- **Agente:** @developer
- **Description:** Create in-memory implementations of the ports
  used by the tests in TODOs #5, #6, #7. They live in
  `tests/testutil/fakes/`.

  ```go
  // tests/testutil/fakes/graph_repo.go
  type InMemoryGraphRepository struct { ... }
  // Implements ports.GraphRepository with a map[uuid.UUID]*domain.Graph.

  // tests/testutil/fakes/execution_repo.go
  type InMemoryExecutionRepository struct { ... }
  // Implements ports.ExecutionRepository with a map[uuid.UUID]*domain.Execution.
  ```

  These fakes have no concurrency or persistence — for tests only.

- **Acceptance criteria:** `go build ./tests/testutil/fakes/...`
  compiles. The fakes fully implement the port interfaces
  (verified by the compiler).
- **Dependencies:** #4
- **Commit:** `test(fakes): add in-memory fakes for GraphRepository and ExecutionRepository [SPRINT-003 #8]`

### 9. [impl] Implement use cases in `services/orchestrator/internal/usecase/`

- **Agente:** @developer
- **Description:** Implement the 6 use cases. Each use case
  receives its dependencies (repositories) via constructor (dependency
  injection). They follow ADR-003 (functions ≤20 lines) and ADR-006
  rule 1 (no business logic outside the domain).

  Structure:
  ```
  services/orchestrator/internal/usecase/
  ├── graph.go          # CreateGraph, GetGraph, ListGraphs, UpdateGraph, ArchiveGraph
  └── execution.go      # StartExecution
  ```

  `CreateGraph` contract:
  1. Assign new UUID and timestamps.
  2. Assign `status = draft`.
  3. Call `repo.Create(ctx, graph)`.
  4. If the repository returns a uniqueness error → `ErrConflict`.
  5. Return the persisted graph.

  `ArchiveGraph` contract:
  1. Load the graph. If not found → `ErrNotFound`.
  2. Count active executions. If > 0 → `ErrConflict`.
  3. Call `repo.Archive(ctx, id)`.

  `UpdateGraph` contract:
  1. Load the graph. If not found → `ErrNotFound`.
  2. If not draft → `ErrInvalidGraphStatus`.
  3. Update fields. Call `repo.Update(ctx, graph)`.

  `StartExecution` contract:
  1. Load the graph. If not found → `ErrNotFound`.
  2. Create `Execution` with `status=pending`, `variables` from input,
     `messages=[]`, `node_results={}`.
  3. Call `executionRepo.Create(ctx, execution)`.

- **Acceptance criteria:** `go test ./tests/unit/usecase/...` turns
  GREEN. `make lint` without errors. No function > 20 lines.
- **Dependencies:** #6, #8
- **Commit:** `feat(orchestrator): implement graph and execution use cases [SPRINT-003 #9]`

### 10. [impl] Implement Gin handlers in `services/orchestrator/internal/handler/`

- **Agente:** @developer
- **Description:** Implement the handlers following ADR-006.

  Structure:
  ```
  services/orchestrator/internal/handler/
  ├── graph.go       # GraphHandler with 5 methods
  ├── execution.go   # ExecutionHandler with 1 method
  └── errors.go      # mapDomainError()
  ```

  Pattern for each handler (ADR-006 rules 1 and 2):
  ```go
  func (h *GraphHandler) Create(c *gin.Context) {
      var input GraphInput
      if err := c.ShouldBindJSON(&input); err != nil {
          c.JSON(http.StatusBadRequest, errorResponse("INVALID_JSON", err.Error()))
          return
      }
      graph, err := h.createGraph.Execute(c.Request.Context(), input.toDomain())
      if err != nil {
          mapDomainError(c, err)
          return
      }
      c.JSON(http.StatusCreated, graphResponseFrom(graph))
  }
  ```

  `mapDomainError()` implements the error table from the design.

  The `toDomain()` and `graphResponseFrom()` functions are translation
  helpers. They count toward the 20-line function limit
  (ADR-003).

- **Acceptance criteria:** `go test ./tests/unit/handler/...` turns
  GREEN. `go test ./tests/contract/...` turns GREEN.
  `make lint` without errors.
- **Dependencies:** #7, #9
- **Commit:** `feat(orchestrator): implement Gin handlers for graphs and executions [SPRINT-003 #10]`

### 11. [impl] Implement storage adapter with Ent

- **Agente:** @developer
- **Description:** Implement `GraphRepository` and `ExecutionRepository`
  using the Ent client generated in SPRINT-002. They live in
  `adapters/storage/`.

  **Rules from ADR-001 and ADR-007:**
  - Ent types only exist inside the adapter.
  - The adapter translates `ent.Graph` → `domain.Graph` and vice versa.
  - Use `client.Tx(ctx)` + defer rollback for write operations.
  - Wrap errors with `fmt.Errorf("storage.CreateGraph: %w", err)`.

  **`adapters/storage/graph_repo.go`:**
  ```go
  type EntGraphRepository struct {
      client *ent.Client
  }

  func (r *EntGraphRepository) Create(ctx context.Context, g *domain.Graph) (*domain.Graph, error) {
      created, err := r.client.Graph.Create().
          SetID(g.ID).
          SetName(g.Name).
          // ...
          Save(ctx)
      if err != nil {
          if ent.IsConstraintError(err) {
              return nil, fmt.Errorf("%w: %w", domain.ErrConflict, err)
          }
          return nil, fmt.Errorf("storage.CreateGraph: %w", err)
      }
      return entGraphToDomain(created), nil
  }
  ```

  The `entGraphToDomain()` function translates `*ent.Graph` → `*domain.Graph`.
  Analogous for `*ent.Execution`.

  `List()` implements pagination with `.Offset()` and `.Limit()`. If
  `opts.Status` is not empty, adds `.Where(graph.StatusEQ(...))`.

- **Acceptance criteria:** `make test-integration` (with PostgreSQL)
  passes. Contract tests with the real adapter (instead of the fake)
  pass. `make lint` without errors.
- **Dependencies:** #4, #9
- **Commit:** `feat(storage): implement EntGraphRepository and EntExecutionRepository [SPRINT-003 #11]`

### 12. [impl] Mount the router and server in `services/orchestrator/`

- **Agente:** @developer
- **Description:** Create the Gin router and the real `main.go`.

  **`services/orchestrator/internal/router/router.go`:**
  ```go
  // NewRouter builds the Gin router with middlewares and routes.
  func NewRouter(graphHandler *handler.GraphHandler, executionHandler *handler.ExecutionHandler) *gin.Engine {
      r := gin.New()
      r.Use(gin.Recovery())
      r.Use(middlewareLogger())
      r.Use(middlewareRequestID())

      v1 := r.Group("/api/v1")
      {
          g := v1.Group("/graphs")
          g.POST("", graphHandler.Create)
          g.GET("", graphHandler.List)
          g.GET("/:id", graphHandler.GetByID)
          g.PUT("/:id", graphHandler.Update)
          g.DELETE("/:id", graphHandler.Archive)

          e := v1.Group("/executions")
          e.POST("", executionHandler.Start)
      }
      return r
  }
  ```

  **`services/orchestrator/cmd/main.go`** — complete initialization:
  1. Connect to PostgreSQL (DSN from env var `DATABASE_URL`).
  2. Build `ent.Client`, run `client.Schema.Create(ctx)`.
  3. Build `EntGraphRepository` and `EntExecutionRepository`.
  4. Build use cases.
  5. Build handlers.
  6. Build router.
  7. Start server on `PORT` (default 8080).

  Graceful shutdown handling (`os.Signal`, `context.WithTimeout`).

- **Acceptance criteria:** `go build ./services/orchestrator/...`
  compiles. The binary starts and responds 200 to
  `GET /api/v1/graphs?page=1&per_page=10` with an empty array.
  `make lint` without errors.
- **Dependencies:** #10, #11
- **Commit:** `feat(orchestrator): wire up router and server in main.go [SPRINT-003 #12]`

### 13. [test] Smoke test of the started API

- **Agente:** @qa
- **Description:** Add a smoke test for the real API with build
  tag `smoke`. Starts the server on a free port with
  `httptest.NewServer`, performs a complete CRUD cycle and verifies
  the expected status codes.

  ```go
  //go:build smoke

  // TestAPICRUDSmoke verifies the complete cycle:
  // POST → GET list → GET by id → PUT → DELETE → GET 404.
  func TestAPICRUDSmoke(t *testing.T)

  // TestAPIStartExecutionSmoke verifies POST /executions with a
  // previously created graph.
  func TestAPIStartExecutionSmoke(t *testing.T)
  ```

- **Acceptance criteria:** `make test-smoke` passes including these
  two new tests.
- **Dependencies:** #12
- **Commit:** `test(smoke): add API CRUD smoke tests for orchestrator [SPRINT-003 #13]`

### 14. [docs] Update docs/index.md and docs/log.md

- **Agente:** @docs
- **Description:** Update the specs section in `docs/index.md`
  indicating that the graph and execution paths are defined.
  Add SPRINT-003 to the sprints table. Update `docs/log.md`.
- **Acceptance criteria:** `docs/index.md` and `docs/log.md`
  reflect the state after the sprint.
- **Dependencies:** #13
- **Commit:** `docs(api): update index with SPRINT-003 results [SPRINT-003 #14]`

## Traceability Matrix

| Spec / ADR | Rule | TODO | Artifact | Verified by |
|------------|------|------|----------|-------------|
| ADR-010 rule 6 | Spec-first, not code-first | #1, #2 | `specs/paths/`, `specs/schemas/` | before any code |
| ADR-010 rule 1 | Mandatory `/api/v1/` prefix | #12 | `router.go` | contract tests |
| ADR-010 rule 8 | Standard `ErrorResponse` format | #10 | `handler/errors.go` | handler unit tests |
| ADR-006 rule 1 | Thin handlers | #10 | each handler ≤20 lines | `make lint` (funlen) |
| ADR-006 rule 2 | `c.Request.Context()` to domain | #10 | handlers | code review |
| ADR-006 rule 3 | Centralized `mapDomainError()` | #10 | `handler/errors.go` | handler tests |
| ADR-006 rule 4 | Routes under `/api/v1/` | #12 | `router.go` | contract tests |
| ADR-001 rule 1 | Domain without infrastructure | #3 | `libs/domain/` without Gin/Ent imports | `go build` + review |
| ADR-001 rule 2 | Ports as Go interfaces | #4 | `libs/ports/storage.go` | compiler |
| ADR-001 rule 4 | Ent types do not leave the adapter | #11 | `adapters/storage/` | code review |
| ADR-001 rule 5 | DI in main.go | #12 | `cmd/main.go` | compilation |
| ADR-016 | 7 node patterns (enum) | #3 | `libs/domain/` constants | domain tests |
| ADR-002 | TDD Red → Green | #5, #6, #7 before #9, #10 | TODO order | CI |
| ADR-003 | Functions ≤20 lines | #9, #10, #11 | all impl files | `make lint` (funlen) |
| `specs/openapi.yaml` | DELETE archives, does not delete | #9 | `ArchiveGraph` use case | `TestArchiveGraph*` |
| `specs/openapi.yaml` | PUT only draft graphs | #9 | `UpdateGraph` use case | `TestUpdateGraphOnlyDraft` |

## Sprint Acceptance Criteria

```bash
# 1. OpenAPI spec is valid
swagger-cli validate specs/openapi.yaml

# 2. All code compiles
go build ./...

# 3. Linter passes
make lint

# 4. Unit tests (without Docker)
go test ./tests/unit/...
go test ./tests/testutil/...

# 5. Contract tests (without Docker, with fakes)
go test -tags=contract ./tests/contract/...

# 6. Integration tests (with Docker)
make test-integration

# 7. Smoke tests (includes API CRUD cycle)
make test-smoke

# 8. Complete CI pipeline
make ci

# 9. Server starts and responds
go run ./services/orchestrator/cmd/main.go &
curl -s http://localhost:8080/api/v1/graphs | jq .
# → {"items":[],"pagination":{"page":1,"per_page":20,"total":0,"total_pages":0}}
```

Additionally:
- `libs/domain/` does not import `entgo.io`, `gin-gonic` or `go-redis`.
- `libs/ports/` only imports `libs/domain/` and stdlib.
- All handlers have ≤20 lines (verified by funlen).
- `mapDomainError()` covers all domain errors (no `switch`
  with default that silences unknown errors).
- DELETE returns 204 without body.
- POST /graphs returns 201 with body.
- POST /executions returns 201 with body.

## Sprint Result

_Completed at the end of the sprint._

### Tests executed

- Total: —
- Passed: —
- Failed: —

### Files created/modified

_List generated at close._

### Decisions made during the sprint

_Any unforeseen decision that requires an ADR or note is documented here._

### Reviewer notes

_Pending review._
