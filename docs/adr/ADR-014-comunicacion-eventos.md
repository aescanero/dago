# ADR-014: Inter-service communication (events + HTTP)

**Status:** Accepted (revised: two modes after decomposition)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

Dago distributes graph execution following the LangGraph model:
centralised state with event-driven transitions. After decomposing
into 8 services, two communication modes are needed.

## Decision

### Two communication modes

**Events (Valkey Streams)** — Graph orchestration. Execution services
communicate exclusively via events:

```
orchestrator ↔ executor (node.execute.requested / node.executed)
orchestrator ↔ router   (node.route.requested / node.routed)
orchestrator ↔ planner  (graph.plan.requested / graph.planned)
executor     ↔ mcp-registry (mcp.tool.invoked / mcp.tool.result)
```

**HTTP (Gin)** — Support services. Low-latency synchronous queries:

```
orchestrator → catalog      (get package definition)
orchestrator → auth-server   (JWKS)
executor     → mcp-registry  (MCP discovery)
dashboard    → orchestrator  (REST API + WebSocket AG-UI)
dashboard    → catalog       (package management)
dashboard    → agent-registry (Agent Cards)
any          → auth-server   (JWKS validation)
```

### Orchestration event catalogue

| Event | Producer | Consumer |
|-------|----------|----------|
| `graph.submitted` | orchestrator (API) | orchestrator |
| `graph.planned` | planner | orchestrator |
| `node.execute.requested` | orchestrator | executor |
| `node.executed` | executor | orchestrator |
| `node.execute.failed` | executor | orchestrator |
| `node.route.requested` | orchestrator | router |
| `node.routed` | router | orchestrator |
| `graph.completed` | orchestrator | dashboard (AG-UI/WS) |
| `graph.failed` | orchestrator | dashboard (AG-UI/WS) |
| `graph.paused` | orchestrator | dashboard (AG-UI/WS) |
| `graph.resumed` | orchestrator (API) | orchestrator |
| `mcp.tool.invoked` | executor | mcp-registry |
| `mcp.tool.result` | mcp-registry | executor |

All defined in `specs/asyncapi.yaml` (ADR-011).

### Centralised state (LangGraph model)

The orchestrator maintains the canonical state of each execution in
PostgreSQL (Ent). Each event carries the relevant state (Event-Carried
State Transfer). Other services never query the orchestrator's DB
directly.

### Concrete rules

1. **Orchestration: events only.** The executor never calls the
   orchestrator via HTTP. No exceptions.

2. **Support: synchronous HTTP.** Fast queries before acting.

3. **All events carry `execution_id`** and `auth` (token).

4. **Consumer groups per service.** Horizontal scaling.

5. **Mandatory idempotency** in consumers.

6. **State checkpointing** in PostgreSQL after each transition.

7. **Timeout per node.** If executor/router does not respond,
   `node.execute.failed` with reason `timeout`.

8. **Dead letter:** `{stream}.dlq` for repeated failures.

9. **Retry with backoff** for transient failures (LLM rate limited).

10. **Human-in-the-loop:** `graph.paused` persists state,
    `graph.resumed` when the user responds.

## Notes for Claude Code

- Orchestration events: Valkey Streams.
- Support services: HTTP with Gin.
- Never HTTP between orchestrator ↔ executor/router/planner.
- Each event: envelope with id, type, source, timestamp, data, auth.
- Consumer handlers must be idempotent.
- New node type → define events in AsyncAPI first.
