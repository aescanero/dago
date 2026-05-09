# SPRINT-010: Orchestrator state machine — Submit, validate, execute, transition, complete

## Metadata

- **Start date:** 2026-05-05
- **Estimated end date:** 2026-05-07
- **Status:** completed
- **Applied ADRs:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-007, ADR-008, ADR-011, ADR-014, ADR-016, ADR-020
- **Affected specs:** specs/asyncapi.yaml (orchestrator operations), specs/paths/executions.yaml (422)
- **Planning agent:** planner
- **Reviewed by:** pending
- **Blocked by:** SPRINT-003 (ExecutionRepository, StartExecution, Ent), SPRINT-007 (event bus), SPRINT-009 (event formats)
- **Blocks:** SPRINT-011 (executor tool_use), SPRINT-015 (episodic memory)

## Objective

Connect the orchestrator with the event bus: validate the graph when submitting an execution,
publish the first `node.execute.requested` event, consume `node.executed` /
`node.execute.failed`, update state and transition until completion or failure.
Only graphs with `sequential` edges are supported in this sprint.

## Scope

- **AsyncAPI** — new orchestrator operations: publish `node.execute.requested`,
  consume `node.executed` and `node.execute.failed`, publish `graphCompleted` and `graphFailed`.
- **OpenAPI** — `422 GRAPH_VALIDATION_ERROR` response on `POST /api/v1/executions`.
- **Domain** — `ErrGraphValidation` in `libs/domain/errors.go`. `GraphDefinition` struct
  with `EntryNode`, `Nodes map[string]NodeDefinition`, `Edges []EdgeDefinition`.
- **Port** — add `UpdateExecution` to `ExecutionRepository`.
- **State machine** in `services/orchestrator/internal/statemachine/`:
  - `graph_validator.go`: `ValidateGraph(g GraphDefinition) error` with `dominikbraun/graph`.
  - `traversal.go`: `NextNode(g GraphDefinition, currentNode string) (string, error)`.
  - `execution_sm.go`: `ExecutionStateMachine` with `HandleNodeExecuted` and `HandleNodeExecuteFailed`.
- **Consumer** `node_result.go` in `services/orchestrator/internal/consumer/` —
  consumes `node.executed` and `node.execute.failed`, delegates to `ExecutionStateMachine`.
- **Extended StartExecution** — validates graph, publishes `node.execute.requested`, sets
  status to `running` (previously remained `pending`).
- **ErrRetryable** — sentinel in `libs/domain/errors.go` for the consumer to propagate NACK.
- Tests: 4 unit (ValidateGraph, NextNode, HandleNodeExecuted, HandleNodeExecuteFailed),
  2 integration with real Valkey (build tag `integration`).

## Dependencies

- **Blocked by:** SPRINT-003 (ExecutionRepository, StartExecution, Ent), SPRINT-007 (event bus),
  SPRINT-009 (executor llm_call, event formats).
- **Blocks:** SPRINT-011 (executor tool_use), SPRINT-015 (episodic memory).

## Behavior Contracts

### C1 — `ValidateGraph` — graph with only sequential edges

```
Given: GraphDefinition with entry_node="a", nodes={"a":{},"b":{}}, edges=[{type:"sequential",from:"a",to:"b"}]
When: ValidateGraph(graph)
Then: Returns nil (no error)
      All nodes are reachable from entry_node
```

### C2 — `ValidateGraph` — unsupported edge

```
Given: GraphDefinition with an edge of type "conditional"
When: ValidateGraph(graph)
Then: Returns error such that errors.Is(err, domain.ErrGraphValidation) == true
      Error message contains "unsupported edge type: conditional"
```

### C3 — `ExecutionStateMachine.HandleNodeExecuted` — terminal node

```
Given: Execution with status="running", GraphDefinition where currentNode has no outgoing edges
When: HandleNodeExecuted(ctx, exec, graph, currentNode, output, auth)
Then: graph.completed event is published to the corresponding stream
      exec.Status is updated to "completed"
      UpdateExecution is called with the updated exec
      Handler returns nil
```

### C4 — Extended `StartExecution` — invalid graph → 422

```
Given: Graph with "conditional" edges in its definition field
When: POST /api/v1/executions with that graph's graph_id
Then: HTTP 422, ErrorResponse with code="GRAPH_VALIDATION_ERROR"
      No Execution is persisted in the database
      No event is published in Valkey
```

## Note on TDD order

> TODOs #14 and #15 (integration tests) must be written in Red BEFORE the implementation TODOs (#7–#12). The correct execution order is:
> `#1 → #2 → #3 → #4 → #14 → #15 → #5 → #6 → #7 → #8 → #9 → #10 → #11 → #12 → #16 → #13 → #17`

## TODOs

### TODO #1 — spec: AsyncAPI — orchestrator operations [spec]

**Agente:** @developer

**Objective:** Register the orchestrator operations in `specs/asyncapi.yaml`.

Add to `operations`:
```yaml
orchestratorPublishNodeExecuteRequested:
  action: send
  channel: $ref: '#/channels/nodeExecuteRequested'
  bindings: {valkey: {group: orchestrator}}

orchestratorConsumeNodeExecuted:
  action: receive
  channel: $ref: '#/channels/nodeExecuted'
  bindings: {valkey: {group: orchestrator}}

orchestratorConsumeNodeExecuteFailed:
  action: receive
  channel: $ref: '#/channels/nodeExecuteFailed'
  bindings: {valkey: {group: orchestrator}}

orchestratorPublishGraphCompleted:
  action: send
  channel: $ref: '#/channels/graphCompleted'

orchestratorPublishGraphFailed:
  action: send
  channel: $ref: '#/channels/graphFailed'
```

Add schemas if they don't exist: `GraphCompletedData` (executionId, graphId, durationMs),
`GraphFailedData` (executionId, graphId, error, errorCode).

**Files:** `specs/asyncapi.yaml`

---

### TODO #2 — spec: OpenAPI — 422 on POST /executions [spec]

**Agente:** @developer

**Objective:** Document the `422` response with code `GRAPH_VALIDATION_ERROR` on
`POST /api/v1/executions` (in `specs/paths/executions.yaml`).

```yaml
'422':
  description: Graph validation failed (unsupported edge types, unreachable nodes, etc.)
  content:
    application/json:
      schema:
        $ref: '../schemas/error.yaml'
      example:
        code: GRAPH_VALIDATION_ERROR
        message: "graph contains unsupported edge type: conditional"
```

**Files:** `specs/paths/executions.yaml`

---

### TODO #3 — test: ValidateGraph — Red [test]

**Agente:** @qa

**Objective:** Unit tests for `ValidateGraph` before implementing.

Cases:
1. Valid graph (3 sequential nodes) → `nil`.
2. `entry_node` does not exist in `nodes` → `ErrGraphValidation`.
3. `conditional` edge → `ErrGraphValidation` ("unsupported edge type: conditional").
4. Node unreachable from `entry_node` → `ErrGraphValidation`.

**File:** `services/orchestrator/internal/statemachine/graph_validator_test.go`

---

### TODO #4 — test: NextNode + ExecutionStateMachine — Red [test]

**Agente:** @qa

**Objective:** Unit tests before implementing `traversal.go` and `execution_sm.go`.

`NextNode`:
1. Node with sequential successor → returns the key of the next node.
2. Node with no outgoing edges (terminal) → `("", nil)`.

`HandleNodeExecuted`:
3. Intermediate node → publishes `node.execute.requested` and updates state.
4. Terminal node → publishes `graph.completed` and sets execution to `completed`.

`HandleNodeExecuteFailed`:
5. `retryable=false` → publishes `graph.failed`, sets execution to `failed`, returns nil.
6. `retryable=true` → returns `ErrRetryable` (consumer must NACK).

**Files:** `services/orchestrator/internal/statemachine/traversal_test.go`,
`services/orchestrator/internal/statemachine/execution_sm_test.go`

---

### TODO #5 — data: ErrGraphValidation + ErrRetryable + GraphDefinition [data]

**Agente:** @developer

**Objective:** Add the new types and errors to the shared domain.

In `libs/domain/errors.go`:
```go
var ErrGraphValidation = errors.New("domain: graph validation failed")
var ErrRetryable      = errors.New("domain: retryable — consumer must NACK")
```

In `libs/domain/graph.go` (new file if it doesn't exist):
```go
type GraphDefinition struct {
    EntryNode string                     `json:"entry_node"`
    Nodes     map[string]NodeDefinition  `json:"nodes"`
    Edges     []EdgeDefinition           `json:"edges"`
}

type NodeDefinition struct {
    Pattern string          `json:"pattern"`
    Config  json.RawMessage `json:"config"`
}

type EdgeDefinition struct {
    Type string `json:"type"`
    From string `json:"from"`
    To   string `json:"to"`
}
```

**Files:** `libs/domain/errors.go`, `libs/domain/graph.go`

---

### TODO #6 — impl: UpdateExecution in ExecutionRepository [impl]

**Agente:** @developer

**Objective:** Add `UpdateExecution` to the `ExecutionRepository` port.

```go
type ExecutionRepository interface {
    Create(ctx context.Context, exec *domain.Execution) error
    FindByID(ctx context.Context, id string) (*domain.Execution, error)
    CountActiveByGraph(ctx context.Context, graphID string) (int, error)
    UpdateExecution(ctx context.Context, exec *domain.Execution) error  // new
}
```

The `CurrentNode` field is added to `domain.Execution` if it doesn't exist.

**Files:** `libs/ports/storage.go`, `libs/domain/execution.go`

---

### TODO #7 — impl: ValidateGraph with dominikbraun/graph [impl]

**Agente:** @developer

**Objective:** Implement `ValidateGraph` in Green.

```go
// services/orchestrator/internal/statemachine/graph_validator.go
func ValidateGraph(g domain.GraphDefinition) error {
    // 1. entry_node exists in nodes
    // 2. all edges are "sequential"
    // 3. build dominikbraun/graph and verify reachability
    //    from entry_node to all nodes
}
```

Use `github.com/dominikbraun/graph` to build the DAG and verify that all
vertices are reachable from `entry_node`.

**File:** `services/orchestrator/internal/statemachine/graph_validator.go`

---

### TODO #8 — impl: NextNode [impl]

**Agente:** @developer

**Objective:** Implement `NextNode` in Green.

```go
// traversal.go
func NextNode(g domain.GraphDefinition, currentNode string) (string, error) {
    for _, e := range g.Edges {
        if e.From == currentNode && e.Type == "sequential" {
            return e.To, nil
        }
    }
    return "", nil  // terminal node
}
```

**File:** `services/orchestrator/internal/statemachine/traversal.go`

---

### TODO #9 — impl: ExecutionStateMachine [impl]

**Agente:** @developer

**Objective:** Implement the state machine in Green.

```go
type ExecutionStateMachine struct {
    repo      ports.ExecutionRepository
    publisher ports.EventPublisher
}

func (sm *ExecutionStateMachine) HandleNodeExecuted(
    ctx context.Context,
    exec *domain.Execution,
    graph domain.GraphDefinition,
    nodeKey string,
    output json.RawMessage,
    auth string,
) error

func (sm *ExecutionStateMachine) HandleNodeExecuteFailed(
    ctx context.Context,
    exec *domain.Execution,
    retryable bool,
    errMsg, errCode string,
    auth string,
) error
```

- `HandleNodeExecuted`: if `NextNode` returns a node → publish `node.execute.requested`
  (new node), update `exec.CurrentNode` and call `UpdateExecution`.
  If terminal → publish `graph.completed`, set `exec.Status = "completed"`,
  call `UpdateExecution`.
- `HandleNodeExecuteFailed`: if `!retryable` → publish `graph.failed`, set
  `exec.Status = "failed"`, call `UpdateExecution`, return nil.
  If `retryable` → return `domain.ErrRetryable` (consumer must NACK).
- Idempotency: `CanTransitionTo(current, next Status) bool` prevents double transitions.

**File:** `services/orchestrator/internal/statemachine/execution_sm.go`

---

### TODO #10 — impl: Consumer node_result [impl]

**Agente:** @developer

**Objective:** Consumer that consumes `node.executed` and `node.execute.failed`.

```go
// services/orchestrator/internal/consumer/node_result.go
type NodeResultConsumer struct {
    execRepo  ports.ExecutionRepository
    graphRepo ports.GraphRepository
    sm        *statemachine.ExecutionStateMachine
}

func (c *NodeResultConsumer) HandleNodeExecuted(ctx context.Context, evt domain.Event) error
func (c *NodeResultConsumer) HandleNodeExecuteFailed(ctx context.Context, evt domain.Event) error
```

- Loads `Execution` by `executionID` from the event.
- Loads `GraphDefinition` from the `definition` field of the Graph entity.
- Delegates to `ExecutionStateMachine`.
- If it returns `domain.ErrRetryable` → returns error (the Valkey adapter NACKs).
- If it returns nil → the adapter ACKs.

**File:** `services/orchestrator/internal/consumer/node_result.go`

---

### TODO #11 — impl: Extended StartExecution [impl]

**Agente:** @developer

**Objective:** Extend `StartExecution` (SPRINT-003 use case) to validate + publish + running.

Updated flow:
1. Load Graph from repository.
2. Deserialize `definition` to `domain.GraphDefinition`.
3. `ValidateGraph(graphDef)` — if fails → return `ErrGraphValidation` (handler returns 422).
4. `CountActiveByGraph` — if > 0 → return `ErrConflict`.
5. `Create(execution)` with `status = "running"` (no longer `pending`).
6. Publish `node.execute.requested` for `entry_node`.

Add `ErrGraphValidation` to the HTTP handler as 422.

**Files:** `services/orchestrator/internal/usecase/start_execution.go`,
`services/orchestrator/internal/handler/execution_handler.go`

---

### TODO #12 — impl: Wiring in orchestrator main.go [impl]

**Agente:** @developer

**Objective:** Wire the new consumers and state machine in
`services/orchestrator/main.go`.

- Build `ExecutionStateMachine` with repo and publisher.
- Build `NodeResultConsumer` with repos and state machine.
- Register handlers for `node.executed` and `node.execute.failed` in the EventConsumer.
- Start consumers in goroutines.
- Handle graceful shutdown (context cancel).

**File:** `services/orchestrator/main.go`

---

### TODO #13 — infra: go.mod — dominikbraun/graph [infra]

**Agente:** @devops

**Objective:** Add the dependency to the module.

```
go get github.com/dominikbraun/graph
```

Verify it is Apache 2.0 (compatible with the project's open source license).

**Files:** `go.mod`, `go.sum`

---

### TODO #14 — test: state machine integration with real Valkey [test]

**Agente:** @qa

**TDD Note:** This test must be written in Red BEFORE the implementation TODOs (#7–#12). See "Note on TDD order" at the start of the TODOs.

**Objective:** Integration tests with real Valkey (build tag `integration`).

Case 1: Submit valid execution → status `running`, `node.execute.requested` event
published to the correct stream.

Case 2: Consume `node.executed` (terminal node) → status `completed`, `graph.completed`
event published.

Use Testcontainers for Valkey. Minimal real data in PostgreSQL (can use in-memory SQLite
with Ent to isolate from full infrastructure).

**File:** `services/orchestrator/internal/statemachine/integration_test.go`

---

### TODO #15 — test: node_result consumer integration [test]

**Agente:** @qa

**TDD Note:** This test must be written in Red BEFORE the implementation TODOs (#7–#12). See "Note on TDD order" at the start of the TODOs.

**Objective:** End-to-end integration test of the consumer with real Valkey.

Publish `node.executed` event to the stream → verify the consumer calls
`HandleNodeExecuted` → verify ACK and updated state.

**File:** `services/orchestrator/internal/consumer/node_result_integration_test.go`

---

### TODO #16 — impl: Ent adapter for UpdateExecution [impl]

**Agente:** @developer

**Objective:** Implement `UpdateExecution` in the orchestrator's Ent adapter.

```go
func (r *ExecutionRepo) UpdateExecution(ctx context.Context, exec *domain.Execution) error {
    _, err := r.client.Execution.
        UpdateOneID(exec.ID).
        SetStatus(execution.Status(exec.Status)).
        SetCurrentNode(exec.CurrentNode).
        Save(ctx)
    return err
}
```

**File:** `adapters/storage/ent/execution_repo.go`

---

### TODO #17 — docs: update documentation [docs]

**Agente:** @docs

**Objective:** Update documentation artifacts on sprint close.

- `docs/index.md` — SPRINT-010 status: completed.
- `docs/log.md` — closing entry.
- `docs/views/process/` — Execution state diagram (pending→running→completed/failed).
- Comment in `execution_sm.go` explaining the limitation: only `sequential` edges
  (per-node timeout excluded, documented as future TODO).

**Files:** `docs/index.md`, `docs/log.md`, `docs/views/process/`

---

## Traceability Matrix

| TODO | Spec | Test | Impl | Docs |
|------|------|------|------|------|
| #1 AsyncAPI ops | asyncapi.yaml | — | — | — |
| #2 OpenAPI 422 | executions.yaml | — | — | — |
| #3 ValidateGraph test | — | graph_validator_test.go | — | — |
| #4 NextNode + SM tests | — | traversal_test.go, execution_sm_test.go | — | — |
| #5 ErrGraphValidation + types | — | — | domain/errors.go, domain/graph.go | — |
| #6 UpdateExecution port (impl) | — | — | ports/storage.go | — |
| #7 ValidateGraph impl | ADR-016 | #3 | graph_validator.go | — |
| #8 NextNode impl | ADR-016 | #4 | traversal.go | — |
| #9 ExecutionStateMachine | ADR-014 | #4 | execution_sm.go | — |
| #10 Consumer node_result | ADR-014 | — | consumer/node_result.go | — |
| #11 Extended StartExecution | OpenAPI | — | usecase/, handler/ | — |
| #12 Wiring main.go | — | — | orchestrator/main.go | — |
| #13 go.mod dominikbraun | — | — | go.mod | — |
| #14 SM integration | ADR-007, ADR-008 | integration_test.go | — | — |
| #15 Consumer integration | ADR-008, ADR-014 | node_result_integration_test.go | — | — |
| #16 UpdateExecution Ent | ADR-007 | — | ent/execution_repo.go | — |
| #17 Docs | — | — | — | index.md, log.md, views/ |

## Key decisions

- **Only `sequential` edges** in this sprint. `conditional`, `parallel`, `loop`,
  `interrupt` edges → `ErrGraphValidation`. Documented as a known limitation and future TODO.
- **Per-node timeout explicitly excluded.** Go context is propagated but no per-node
  deadline is added; documented as a future TODO.
- **`ErrRetryable`** as a sentinel in `libs/domain/` allows the Valkey consumer to NACK
  without importing adapter details.
- **Mandatory idempotency** (ADR-014): `CanTransitionTo` prevents double transitions if the
  consumer receives the same event twice.
- **Checkpointing** (ADR-014): `UpdateExecution` is called after each transition before
  publishing the next event, guaranteeing eventual consistency.
- **`StartExecution`** goes directly from validation to `running` (not `pending`), simplifying
  the flow since the first event publication is synchronous in this sprint.

## Result

- **TODOs completed:** 17/17
- **Tests passing:** 13 test suites (all pass); integration tests (build tag: integration) written and verified Red
- **Decisions reviewed:** All key decisions from "Key decisions" section applied and documented
- **Artifacts delivered:**
  - specs: asyncapi.yaml (5 orchestrator ops + 2 schemas), executions.yaml (422 GRAPH_VALIDATION_ERROR)
  - domain: ErrGraphValidation, ErrRetryable, GraphDefinition/NodeDefinition/EdgeDefinition, StreamGraphCompleted/StreamGraphFailed
  - port: UpdateExecution in ExecutionRepository
  - statemachine: ValidateGraph, NextNode, ExecutionStateMachine, CanTransitionTo
  - consumer: NodeResultConsumer (HandleNodeExecuted + HandleNodeExecuteFailed)
  - usecase: StartExecution extended (validate → running → publish first node)
  - handler: 422 GRAPH_VALIDATION_ERROR mapping
  - adapter: UpdateExecution in EntExecutionRepository
  - infra: dominikbraun/graph v0.23.0 (Apache-2.0)
  - main.go: full wiring with consumers + graceful shutdown
  - docs: execution_state.md, index.md, log.md updated

**Status:** completed

**Note on TODO order:** TODOs #5 and #6 (domain types + port) were executed before #3 and #4 (tests) to allow test files to compile against existing domain types. This preserves the TDD spirit (tests before implementation) while keeping CI green with compilable code at each commit.
