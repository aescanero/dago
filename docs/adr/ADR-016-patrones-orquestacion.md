# ADR-016: Graph orchestration model — flow and node patterns

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

Dago orchestrates AI workflows modelled as directed graphs, following
the LangGraph model. Graph vertices are nodes that execute agentic
behaviours. Edges define how execution flows.

A formal taxonomy of available patterns is needed, clearly separating
**flow control** patterns (edges — how execution moves) from **node
behaviour** patterns (vertices — what a node does).

Each pattern is defined with a JSON Schema describing its configurable
parameters. The schemas live in `specs/patterns/` as the fourth
spec of the system.

## Decision

### Graph library

**`dominikbraun/graph`** (Go, Apache 2.0, zero dependencies) is used
for structural graph validation: cycle detection, connectivity,
topological order. The library does not execute the graph —
it only validates it. Execution is coordinated by the orchestrator via events.

### Pattern taxonomy

```
┌─────────────────────────────────────────────────────────────┐
│                    EXECUTION GRAPH                          │
│                                                             │
│   Flow control patterns (edges)                             │
│   ──────────────────────────────────────                    │
│   Define HOW execution moves between nodes                  │
│                                                             │
│   • sequential    — A → B linear                            │
│   • conditional   — Branch based on state (rule or LLM)     │
│   • parallel      — Fan-out / fan-in                        │
│   • loop          — Repeat until exit condition             │
│   • interrupt     — Pause and wait for external input       │
│                                                             │
│   Node behaviour patterns (vertices)                        │
│   ─────────────────────────────────────────                 │
│   Define WHAT a node does when it is its turn to execute    │
│                                                             │
│   • llm_call      — Prompt → LLM → response                 │
│   • tool_use      — Invoke tools (MCP, APIs)                │
│   • react         — Internal loop think → act → observe     │
│   • reflection    — Generate → critique → improve           │
│   • router        — Decide next node (rule/LLM)             │
│   • guardrail     — Validate input/output against rules     │
│   • subgraph      — Execute a complete graph as a node      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Flow control patterns (edges)

### `sequential`

The most basic edge: one node follows another with no condition.

```json
{
  "type": "sequential",
  "from": "node_a",
  "to": "node_b"
}
```

### `conditional`

Execution takes one path or another based on a condition evaluated
against the graph state.

```json
{
  "type": "conditional",
  "from": "node_classifier",
  "conditions": [
    {
      "expression": "state.variables.sentiment == 'positive'",
      "target": "node_positive_response"
    },
    {
      "expression": "state.variables.sentiment == 'negative'",
      "target": "node_escalate"
    }
  ],
  "default": "node_fallback"
}
```

Evaluation modes:
- **rule** — Deterministic expression over the state.
- **llm** — An LLM decides the path based on the state.
- **hybrid** — Tries a rule first; if none applies, uses LLM.

### `parallel`

Fan-out: splits execution into N simultaneous branches. Fan-in: waits
for them to complete according to a policy (all, any, N-of-M).

```json
{
  "type": "parallel",
  "from": "node_start",
  "branches": ["node_research", "node_analyze", "node_summarize"],
  "join": {
    "target": "node_combine",
    "policy": "all",
    "timeout_seconds": 120
  }
}
```

Join policies:
- **all** — Waits for all branches to complete.
- **any** — Continues when the first branch completes (cancels the rest).
- **n_of_m** — Continues when N of M branches complete.

### `loop`

Execution returns to a previous node until an exit condition is met.
A **mandatory** stop criterion prevents infinite loops.

```json
{
  "type": "loop",
  "from": "node_reviewer",
  "target": "node_writer",
  "exit_condition": {
    "expression": "state.variables.review_score >= 0.8",
    "max_iterations": 5,
    "timeout_seconds": 300
  },
  "exit_target": "node_publish"
}
```

Validation: the orchestrator rejects graphs with loops lacking
`max_iterations` or `timeout_seconds`.

### `interrupt`

Execution pauses and the state is persisted. It resumes when an
external input arrives (a human via the dashboard, a webhook, or an
external system).

```json
{
  "type": "interrupt",
  "from": "node_proposal",
  "resume_target": "node_execute",
  "reject_target": "node_revise",
  "prompt": "Approve this action?",
  "timeout_seconds": 86400,
  "timeout_target": "node_cancel"
}
```

---

## Node behaviour patterns (vertices)

### `llm_call`

The most basic pattern: sends a prompt to an LLM and returns the response.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/llm_call.json",
  "pattern": "llm_call",
  "config": {
    "model": "claude-sonnet-4-20250514",
    "system_prompt": "You are a helpful assistant specialized in...",
    "temperature": 0.7,
    "max_tokens": 2048,
    "input_mapping": {
      "user_message": "state.messages[-1].content"
    },
    "output_mapping": {
      "state.variables.response": "output.content"
    }
  }
}
```

### `tool_use`

The node invokes external tools. Can be direct (specific tool)
or with LLM selection (the LLM decides which tool to use).

```json
{
  "$schema": "https://dago.dev/schemas/patterns/tool_use.json",
  "pattern": "tool_use",
  "config": {
    "mode": "llm_selected",
    "tools": [
      {
        "type": "mcp",
        "server": "github-mcp",
        "allowed_tools": ["search_repos", "read_file"]
      },
      {
        "type": "api",
        "endpoint": "https://api.example.com/search",
        "method": "GET"
      }
    ],
    "model": "claude-sonnet-4-20250514",
    "max_tool_calls": 5,
    "timeout_seconds": 30
  }
}
```

Modes:
- **direct** — Invokes a specific tool without LLM involvement.
- **llm_selected** — The LLM chooses which tool to use based on context.

### `react`

Internal reasoning + acting loop. Think → Act (tool) → Observe →
Repeat. This is the most canonical agentic pattern.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/react.json",
  "pattern": "react",
  "config": {
    "model": "claude-sonnet-4-20250514",
    "system_prompt": "You are an agent that reasons step by step...",
    "tools": [
      {"type": "mcp", "server": "web-search"},
      {"type": "mcp", "server": "calculator"}
    ],
    "max_iterations": 10,
    "stop_condition": "final_answer",
    "timeout_seconds": 120,
    "thought_visibility": "full"
  }
}
```

`thought_visibility`:
- **full** — Thoughts visible in the output (debugging).
- **final_only** — Only the final answer.

### `reflection`

Generates an output, evaluates it critically, and improves it
iteratively.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/reflection.json",
  "pattern": "reflection",
  "config": {
    "generator": {
      "model": "claude-sonnet-4-20250514",
      "system_prompt": "Generate a detailed analysis of..."
    },
    "critic": {
      "model": "claude-sonnet-4-20250514",
      "system_prompt": "Evaluate the following analysis for accuracy...",
      "criteria": ["accuracy", "completeness", "clarity"]
    },
    "max_iterations": 3,
    "acceptance_threshold": 0.85,
    "output_mapping": {
      "state.variables.analysis": "output.final_version"
    }
  }
}
```

The generator and the critic can use different models.

### `router`

Analyses the input and produces a routing decision. It does not generate
content — it decides the path. It differs from the flow `conditional`:
conditional evaluates state, the router uses complex logic (LLM or
rules) to decide.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/router.json",
  "pattern": "router",
  "config": {
    "mode": "hybrid",
    "rules": [
      {
        "condition": "state.variables.language == 'es'",
        "route": "spanish_handler"
      }
    ],
    "llm_fallback": {
      "model": "claude-sonnet-4-20250514",
      "system_prompt": "Based on the input, decide which handler...",
      "routes": ["technical_support", "billing", "general"]
    }
  }
}
```

Modes:
- **deterministic** — Rules only.
- **llm** — LLM decides only.
- **hybrid** — Rules first, LLM if none applies.

### `guardrail`

Validates input or output against rules. Does not generate content.
Accepts or rejects, optionally with a reason.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/guardrail.json",
  "pattern": "guardrail",
  "config": {
    "mode": "input",
    "checks": [
      {
        "type": "json_schema",
        "schema": {"$ref": "#/components/schemas/OrderRequest"}
      },
      {
        "type": "llm_safety",
        "model": "claude-sonnet-4-20250514",
        "policy": "Reject if the input contains PII or harmful content"
      },
      {
        "type": "regex",
        "pattern": "^(?!.*\\b(DROP|DELETE|TRUNCATE)\\b).*$",
        "description": "No SQL injection"
      }
    ],
    "on_fail": "reject",
    "on_fail_target": "node_error_handler"
  }
}
```

Modes: **input** (validates before executing) or **output** (validates
after generating).

Check types: **json_schema**, **llm_safety**, **regex**,
**custom_function**.

### `subgraph`

Encapsulates a complete graph as an atomic node of the parent graph.
Enables hierarchical composition.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/subgraph.json",
  "pattern": "subgraph",
  "config": {
    "graph_id": "uuid-of-child-graph",
    "input_mapping": {
      "child_state.query": "parent_state.variables.sub_task"
    },
    "output_mapping": {
      "parent_state.variables.sub_result": "child_state.variables.result"
    },
    "timeout_seconds": 600,
    "on_failure": "propagate"
  }
}
```

`on_failure`:
- **propagate** — The parent fails if the subgraph fails.
- **isolate** — The parent continues with an empty/error result.

---

## Complete graph definition

A graph combines nodes (with their behaviour patterns) and edges
(with their flow patterns):

```json
{
  "$schema": "https://dago.dev/schemas/graph.json",
  "id": "graph_customer_support",
  "name": "Customer Support Workflow",
  "version": "1.0.0",
  "entry_node": "classifier",
  "nodes": {
    "classifier": {
      "pattern": "router",
      "config": { "..." : "..." }
    },
    "technical": {
      "pattern": "react",
      "config": { "..." : "..." }
    },
    "billing": {
      "pattern": "llm_call",
      "config": { "..." : "..." }
    },
    "review": {
      "pattern": "guardrail",
      "config": { "..." : "..." }
    }
  },
  "edges": [
    {"type": "conditional", "from": "classifier", "conditions": [
      {"expression": "output.route == 'technical'", "target": "technical"},
      {"expression": "output.route == 'billing'", "target": "billing"}
    ]},
    {"type": "sequential", "from": "technical", "to": "review"},
    {"type": "sequential", "from": "billing", "to": "review"}
  ],
  "memory": {
    "semantic_search": true,
    "episode_context": 3
  }
}
```

### Schema locations

```
specs/
├── openapi.yaml               # REST API
├── asyncapi.yaml              # Events
└── patterns/                  # Pattern schemas (JSON Schema)
    ├── graph.json             # Full graph schema
    ├── edges/
    │   ├── sequential.json
    │   ├── conditional.json
    │   ├── parallel.json
    │   ├── loop.json
    │   └── interrupt.json
    └── nodes/
        ├── llm_call.json
        ├── tool_use.json
        ├── react.json
        ├── reflection.json
        ├── router.json
        ├── guardrail.json
        └── subgraph.json
```

### Graph validation

Before executing, the orchestrator validates:

1. **Structure** (using `dominikbraun/graph`):
   - It is a valid directed graph.
   - It has a defined and reachable `entry_node`.
   - It has no disconnected nodes.
   - Loops have a stop criterion (`max_iterations` or `timeout`).
   - Parallel edges have a defined join policy.

2. **Schemas** (using JSON Schema validation):
   - Each node satisfies the JSON Schema of its pattern.
   - Each edge satisfies the JSON Schema of its flow type.
   - `input_mapping` and `output_mapping` reference valid paths.

3. **Resources**:
   - Referenced LLM models are available.
   - Referenced MCP servers are registered.
   - Referenced subgraphs exist.

## Considered Alternatives

- **Patterns as Go code (hardcoded):** More performant but not
  user-configurable without recompiling. Graphs must be creatable
  and modifiable from the dashboard.

- **YAML instead of JSON Schema:** More readable but without a
  validation ecosystem. JSON Schema has native support in Go, TypeScript,
  and the dashboard UI.

- **BPMN / Workflow standards:** More complete but designed for
  enterprise workflows, not AI agent orchestration. Excessive
  complexity for our use case.

- **Protobuf for pattern definitions:** Good type safety but less
  accessible to non-technical users editing graphs from the UI.

## Consequences

**Positive:**
- Clear separation between flow (edges) and behaviour (nodes).
- JSON Schema as the fourth spec — consistent with SDD.
- Graphs definable and editable from the dashboard.
- Automatic validation before execution.
- Extensible patterns — adding a new one means creating a JSON Schema
  and a handler in the executor.
- Hierarchical composition with subgraphs.

**Negative:**
- JSON can be verbose for complex graphs (mitigated by the visual
  editor UI in the dashboard).
- The expressiveness of conditions is limited by the chosen expression
  evaluator.
- Each new pattern requires implementation in the executor as well
  as the schema.

## Notes for Claude Code

- Pattern schemas live in `specs/patterns/`.
- When creating a new node pattern, create the JSON Schema in
  `specs/patterns/nodes/` and the handler in `services/executor/internal/`.
- When creating a new flow pattern, create the JSON Schema in
  `specs/patterns/edges/` and the logic in `services/orchestrator/internal/`.
- Graph validation uses `dominikbraun/graph` for structure
  and JSON Schema validation for configuration.
- Graphs are defined as JSON. The dashboard provides a visual editor
  but the underlying format is JSON.
- Every node with a loop (`react`, `reflection`) must have
  `max_iterations` and/or `timeout_seconds`. Reject graphs without them.
