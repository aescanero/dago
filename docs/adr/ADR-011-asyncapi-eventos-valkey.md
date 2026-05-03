# ADR-011: AsyncAPI and event pattern with Valkey

**Status:** Accepted (revised: Redis → Valkey)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

ADR-008 establishes Valkey as the event infrastructure. We need to
define which event pattern to adopt, how messages are structured,
and how the asynchronous API is formally documented.

## Decision

### Contract: AsyncAPI 3.0

**AsyncAPI 3.0** is adopted as the asynchronous API specification.
File: `specs/asyncapi.yaml`.

### Pattern: Event Notification + Event-Carried State Transfer

```
Event Notification (Pub/Sub)       Event-Carried State (Streams)
─────────────────────────          ──────────────────────────────
Guarantee: none                    Guarantee: at-least-once
Payload: minimal (ID)              Payload: complete
Use: cache, real-time UI           Use: business events
```

### Concrete rules

1. **Event Notification (Valkey Pub/Sub)** for ephemeral signals:
   cache invalidation, UI notifications.

2. **Event-Carried State Transfer (Valkey Streams)** for business
   events with guaranteed processing.

3. **No Event Sourcing.** State in PostgreSQL (Ent). If event
   sourcing is needed, evaluate Kafka/EventStoreDB.

4. **Standard envelope:**

   ```go
   type Event struct {
       ID        string          `json:"id"`
       Type      string          `json:"type"`
       Source    string          `json:"source"`
       Timestamp time.Time       `json:"timestamp"`
       Data      json.RawMessage `json:"data"`
       Auth      string          `json:"auth,omitempty"` // Propagated token
   }
   ```

5. **Naming:** `{domain}.{action}` in past tense. `node.executed`, not
   `node.execute`.

6. **Consumer groups** for distributed processing (Valkey Streams).

7. **Dead letter:** `{stream}.dlq` for messages with repeated failures.

8. **Retention:** `MAXLEN ~1000` per stream.

9. **Spec-first:** Define in `specs/asyncapi.yaml` before
   implementing. Contract tests validate compliance.

10. **`auth` field in the envelope** for propagating OAuth 2.1 tokens
    (ADR-012) through events.

### AsyncAPI example

```yaml
asyncapi: 3.0.0
info:
  title: Dago Event API
  version: 1.0.0

channels:
  nodeExecuted:
    address: node.executed
    messages:
      nodeExecutedMessage:
        $ref: '#/components/messages/NodeExecuted'

components:
  messages:
    NodeExecuted:
      payload:
        $ref: '#/components/schemas/NodeExecutedPayload'

  schemas:
    NodeExecutedPayload:
      type: object
      required: [id, type, source, timestamp, data]
      properties:
        id:
          type: string
        type:
          type: string
          const: node.executed
        source:
          type: string
        timestamp:
          type: string
          format: date-time
        auth:
          type: string
        data:
          type: object
```

## Notes for Claude Code

- Events in `specs/asyncapi.yaml` — spec-first.
- Valkey Streams for business events, Pub/Sub for ephemeral ones.
- Envelope with id, type, source, timestamp, data, auth.
- Naming: simple past tense (`node.executed`).
- ACK after successful processing, never before.
- Publisher in `adapters/eventbus/`. Consumer in each service.
- The `auth` field propagates the OAuth 2.1 token.
