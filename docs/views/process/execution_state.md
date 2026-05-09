# Execution State Machine

Execution lifecycle implemented in SPRINT-010.

## States

```
pending ──► running ──► completed
                 └──► failed
```

| Status | Description |
|--------|-------------|
| `pending` | Initial state (reserved; StartExecution now goes directly to `running`) |
| `running` | Graph is actively executing; orchestrator waiting for node events |
| `completed` | All nodes finished successfully; `graph.completed` published |
| `failed` | A non-retryable node error occurred; `graph.failed` published |
| `cancelled` | Cancelled by user (future sprint) |
| `interrupted` | Interrupted mid-execution (future sprint) |

## Valid Transitions

| From | To | Trigger |
|------|----|---------|
| `pending` / `running` | `running` | `StartExecution` or `HandleNodeExecuted` (intermediate node) |
| `running` | `completed` | `HandleNodeExecuted` (terminal node) |
| `running` | `failed` | `HandleNodeExecuteFailed` (retryable=false) |

Idempotency: `CanTransitionTo(current, next)` returns false for already-completed
transitions, preventing double state changes under at-least-once delivery.

## Event Flow (sequential graph, 2 nodes)

```
[Client] POST /api/v1/executions
  │
  ▼
[Orchestrator usecase]
  1. FindGraph + unmarshal GraphDefinition
  2. ValidateGraph (sequential-only check + reachability)
  3. Create Execution (status=running, current_node=entry_node)
  4. Publish node.execute.requested → node.execute.requested stream
  │
  ▼ (async)
[Executor] consumes node.execute.requested
  1. Dispatch to LLMCallHandler
  2. Publish node.executed → node.executed stream
  │
  ▼ (async)
[Orchestrator consumer] NodeResultConsumer.HandleNodeExecuted
  1. Load Execution + GraphDefinition
  2. NextNode(graph, nodeKey) → "node_b" (intermediate)
  3. UpdateExecution(current_node="node_b")
  4. Publish node.execute.requested for node_b
  │
  ▼ (async — node_b is terminal)
[Executor] consumes node.execute.requested for node_b
  → Publish node.executed
  │
  ▼ (async)
[Orchestrator consumer] HandleNodeExecuted
  1. NextNode(graph, "node_b") → "" (terminal)
  2. exec.Status = completed
  3. UpdateExecution
  4. Publish graph.completed → dago.graph.execution.completed stream
```

## Known Limitations (documented as future TODOs)

- **Only `sequential` edges** are supported. `conditional`, `parallel`, `loop`,
  `interrupt` → `ErrGraphValidation`. These will be implemented in future sprints
  (SPRINT-011+).
- **Per-node timeout** is not implemented. The Go context is propagated through
  the call chain but no per-node deadline is added. This is a future TODO.
