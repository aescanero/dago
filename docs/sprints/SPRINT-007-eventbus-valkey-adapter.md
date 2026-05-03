# SPRINT-007: Event Bus Adapter — Valkey Streams + Consumer Groups

## Metadata

- **Start date:** 2026-04-29
- **Estimated end date:** 2026-04-30
- **Status:** planned
- **ADRs applied:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-008, ADR-011, ADR-013, ADR-020
- **Affected specs:** specs/asyncapi.yaml (7 orchestration channels)
- **Planning agent:** planner
- **Reviewed by:** pending
- **Blocked by:** SPRINT-001 (go.mod, docker-compose with Valkey 8)
- **Blocks:** SPRINT-009 (executor consumes node.execute.requested), SPRINT-010 (orchestrator publishes events)

## Objective

Implement the `EventPublisher`/`EventConsumer` port in `libs/ports/` and its
Valkey Streams adapter in `adapters/eventbus/valkey/`, with standard CloudEvents envelope
(including `auth` field), consumer groups with ACK/NACK, pending messages recovery and
dead-letter queue. Integration tests with Testcontainers.

## Scope

**Included:**
- Channel and message definitions in `specs/asyncapi.yaml` (7 orchestration events)
- Domain types: `Event`, `EventAuth` in `libs/domain/events.go`
- Ports: `EventPublisher`, `EventConsumer`, `EventHandler` in `libs/ports/eventbus.go`
- Adapter: publisher, consumer, envelope, consumer group in `adapters/eventbus/valkey/`
- Integration tests with Testcontainers (6 cases, build tag `integration`)
- Environment variables documented in `.env.example`

**Not included:**
- Business event publishing (that is SPRINT-003 onwards for orchestrator)
- Pub-Sub (dashboard notifications — future SPRINT-007-pubsub)
- Valkey cache (future SPRINT-cache)
- OAuth sessions in Valkey (SPRINT-005 uses in-memory for now)
- Consumer group wiring in concrete services (each service does it in its `main.go`)

## Dependencies

- **Blocked by:** SPRINT-001 (go.mod with Go dependencies, docker-compose with Valkey 8)
- **Parallel with:** SPRINT-002, SPRINT-003, SPRINT-004 (no dependency between them)
- **Blocks:** extended SPRINT-003 (orchestrator publishes `execution.requested`),
  SPRINT-008 (executor consumes events), SPRINT-009 (router consumes events)

## Behavior Contracts

### C1 — `EventPublisher.Publish` — basic publish

```
Given: Active Valkey, existing stream "dago.graph.execution.requested"
When: publisher.Publish(ctx, event, PublishOptions{Stream: "dago.graph.execution.requested"})
Then: XLEN of the stream increases by 1
      The entry contains field "envelope" with the complete event JSON
      event.ID is a valid UUID v4
      unmarshal(marshal(event)) == event (exact round-trip)
```

### C2 — `EventConsumer.Subscribe` — automatic ACK on success

```
Given: Message published in the stream, consumer group configured
When: The handler returns nil when processing the message
Then: The message receives XACK
      A second XREADGROUP call does not return the message
      The message does not appear in the PEL (Pending Entry List)
```

### C3 — DLQ after MaxRetries

```
Given: Message published, MaxRetries=3, handler always returns error
When: The consumer processes the message exactly 3 times
Then: The message appears in the "dago.dlq" stream
      The original message receives XACK in the source stream (leaves the PEL)
      The event in dago.dlq preserves the execution_id of the original event
```

## TODOs

### TODO #1 — spec: Define AsyncAPI channels for orchestration events

**Agente:** @developer

**Description:** Update `specs/asyncapi.yaml` with the 7 Valkey Streams channels
for the initial orchestration phase, following the CloudEvents envelope + `auth` field
defined in ADR-011.

**Affected files:**
- `specs/asyncapi.yaml`

**Channels to define:**
- `dago.graph.execution.requested` — orchestrator publishes; executor/router consume
- `dago.node.execution.started` — executor publishes; orchestrator consumes
- `dago.node.execution.completed` — executor publishes; orchestrator consumes
- `dago.node.execution.failed` — executor publishes; orchestrator consumes
- `dago.graph.execution.completed` — orchestrator publishes
- `dago.graph.execution.failed` — orchestrator publishes
- `dago.dlq` — dead-letter queue; any service publishes

**Envelope schema (inline in AsyncAPI):**
```yaml
type: object
required: [id, type, source, specversion, time, datacontenttype, data]
properties:
  id:            { type: string, format: uuid }
  type:          { type: string }
  source:        { type: string }
  specversion:   { type: string, enum: ["1.0"] }
  time:          { type: string, format: date-time }
  datacontenttype: { type: string, enum: ["application/json"] }
  auth:
    type: object
    properties:
      sub:      { type: string, format: uuid }
      scope:    { type: string }
      tags:     { type: array, items: { type: string } }
      org_unit: { type: string, format: uuid }
  data:          { type: object }
```

**Acceptance criteria:**
- `asyncapi validate specs/asyncapi.yaml` reports no errors
- Each channel has: stream name, Valkey binding, subscribed consumer groups, schema

**Associated test:** —

---

### TODO #2 — data: Event and EventAuth types in libs/domain/events.go

**Agente:** @developer

**Description:** Define pure domain types for events. Without infrastructure
dependencies. `Event.Data` is `json.RawMessage` to avoid coupling to the concrete type.

**Affected files:**
- `libs/domain/events.go` (new)

**Types to implement:**
```go
package domain

import (
    "encoding/json"
    "time"
)

type EventAuth struct {
    Sub     string   `json:"sub"`
    Scope   string   `json:"scope"`
    Tags    []string `json:"tags"`
    OrgUnit string   `json:"org_unit"`
}

type Event struct {
    ID              string          `json:"id"`
    Type            string          `json:"type"`
    Source          string          `json:"source"`
    SpecVersion     string          `json:"specversion"`
    Time            time.Time       `json:"time"`
    DataContentType string          `json:"datacontenttype"`
    Auth            *EventAuth      `json:"auth,omitempty"`
    Data            json.RawMessage `json:"data"`
}
```

**Event type constants:**
```go
const (
    EventTypeExecutionRequested  = "dago.graph.execution.requested"
    EventTypeNodeStarted         = "dago.node.execution.started"
    EventTypeNodeCompleted       = "dago.node.execution.completed"
    EventTypeNodeFailed          = "dago.node.execution.failed"
    EventTypeExecutionCompleted  = "dago.graph.execution.completed"
    EventTypeExecutionFailed     = "dago.graph.execution.failed"
    EventTypeDLQ                 = "dago.dlq"
)
```

**Acceptance criteria:**
- `go build ./libs/domain/...` without errors
- No infrastructure imports (`adapters/`, `valkey-go`, etc.)

**Associated test:** —

---

### TODO #3 — impl: EventPublisher and EventConsumer interfaces in libs/ports/eventbus.go

**Agente:** @developer

**Description:** Define the output ports for event publishing and consumption.
Separate `EventPublisher` from `EventConsumer` so each service imports only what it
needs (orchestrator publishes and consumes; executor only consumes certain streams and publishes
results).

**Affected files:**
- `libs/ports/eventbus.go` (new)

**Interfaces to implement:**
```go
package ports

import (
    "context"
    "github.com/aescanero/dago/libs/domain"
)

type PublishOptions struct {
    Stream string
}

type ConsumeOptions struct {
    Stream        string
    Group         string
    ConsumerName  string
    BlockDuration time.Duration
    MaxRetries    int // default 3, after MaxRetries goes to DLQ
}

type EventHandler func(ctx context.Context, event domain.Event) error

type EventPublisher interface {
    Publish(ctx context.Context, event domain.Event, opts PublishOptions) error
    Close() error
}

type EventConsumer interface {
    // Subscribe blocks until ctx is cancelled. Calls handler for each message.
    // Automatic ACK if handler returns nil. No ACK if it returns error.
    Subscribe(ctx context.Context, opts ConsumeOptions, handler EventHandler) error
    // RecoverPending reassigns idle messages > idleThreshold to the current consumer.
    RecoverPending(ctx context.Context, opts ConsumeOptions, idleThreshold time.Duration) error
    Close() error
}
```

**Acceptance criteria:**
- `go build ./libs/ports/...` without errors
- Only imports `libs/domain/` and stdlib packages

**Associated test:** —

---

### TODO #4 — test: Integration tests (Red) with Testcontainers

**Agente:** @qa

**Description:** Write the 6 integration tests with build tag `integration` BEFORE
implementing the adapter. They must fail (Red) when run against empty stubs.

**Affected files:**
- `adapters/eventbus/valkey/integration_test.go` (new)

**Tests to implement:**
```go
//go:build integration

package valkey_test

// TestMain: starts valkey/valkey:8 container, gets addr, creates client

func TestPublishAndConsume(t *testing.T)
// Publish an event → Subscribe with consumer group → handler receives same event

func TestConsumerGroupAck(t *testing.T)
// Publish → Subscribe (handler OK → ACK) → second XREADGROUP does not return the message

func TestConsumerGroupNoAck(t *testing.T)
// Publish → Subscribe (handler error → no ACK) → second XREADGROUP returns the same message

func TestPendingRecovery(t *testing.T)
// Publish → consumer1 reads without ACK → RecoverPending with idle>0 → consumer2 receives the message

func TestDLQAfterMaxRetries(t *testing.T)
// Publish → Subscribe with MaxRetries=3, handler always errors → after 3 attempts
// the message appears in stream "dago.dlq"

func TestEnvelopeRoundtrip(t *testing.T)
// Event with all fields including Auth → Publish → Subscribe
// → received event == published event (deep equal)
```

**Acceptance criteria:**
- `go test -tags integration ./adapters/eventbus/valkey/... -run TestPublish` fails with
  "not implemented" or similar (Red confirmed)
- Testcontainers automatically downloads image `valkey/valkey:8`

**Associated test:** this TODO IS the test

---

### TODO #5 — impl: Envelope — CloudEvents serialization/deserialization

**Agente:** @developer

**Description:** Implement the conversion between `domain.Event` and the Valkey Streams
input format (`map[string]any` with `envelope` JSON field).

**Affected files:**
- `adapters/eventbus/valkey/envelope.go` (new)

**Design:**
```go
// marshalEnvelope serializes domain.Event → map[string]any for XADD
// unmarshalEnvelope deserializes XREAD entry → domain.Event
// The stream entry has a single "envelope" field with the complete event JSON
```

**Acceptance criteria:**
- `TestEnvelopeRoundtrip` passes (Green)
- Marshal/Unmarshal are inverses: `unmarshal(marshal(e)) == e`
- `auth` field is omitted if nil (`omitempty`)

**Associated test:** `TestEnvelopeRoundtrip`

---

### TODO #6 — impl: Publisher — XADD with idempotent XGROUP CREATE

**Agente:** @developer

**Description:** Implement `ValkeyPublisher` that implements `ports.EventPublisher`.
Ensure the stream and group exist before publishing (XGROUP CREATE MKSTREAM).

**Affected files:**
- `adapters/eventbus/valkey/publisher.go` (new)

**Behavior:**
- `Publish`: serializes with `marshalEnvelope` → `XADD stream * envelope <json>`
- `EnsureStream`: `XGROUP CREATE stream <group> $ MKSTREAM` (idempotent, ignores BUSYGROUP)
- `Close`: closes Valkey connection

**Error handling:**
- `fmt.Errorf("eventbus publish %s: %w", stream, err)`
- Timeout from `ctx`

**Acceptance criteria:**
- `TestPublishAndConsume` passes (Green) when the consumer is also implemented
- `TestEnvelopeRoundtrip` passes

**Associated test:** `TestPublishAndConsume`, `TestEnvelopeRoundtrip`

---

### TODO #7 — impl: Consumer — XREADGROUP + ACK + DLQ after MaxRetries

**Agente:** @developer

**Description:** Implement `ValkeyConsumer` that implements `ports.EventConsumer`.
`Subscribe` blocks in a loop reading with `XREADGROUP`. Manages ACK/NACK and DLQ.

**Affected files:**
- `adapters/eventbus/valkey/consumer.go` (new)

**Subscribe behavior:**
1. `XGROUP CREATE stream group $ MKSTREAM` (idempotent)
2. Loop until `ctx.Done()`:
   a. `XREADGROUP GROUP group consumer BLOCK blockDuration COUNT 10 STREAMS stream >`
   b. For each message: deserialize → call `handler`
   c. If `handler` returns nil: `XACK stream group id`
   d. If `handler` returns error:
      - Query `XPENDING stream group - + 10` to count message retries
      - If retries >= `MaxRetries`: publish to `dago.dlq` + `XACK` (to remove from pending)
      - If retries < `MaxRetries`: no XACK (message stays in pending)

**Error handling:**
- Connection errors: log + retry with exponential backoff (1s, 2s, 4s, max 30s)
- `fmt.Errorf("eventbus consume %s: %w", stream, err)`

**Acceptance criteria:**
- `TestConsumerGroupAck` passes
- `TestConsumerGroupNoAck` passes
- `TestDLQAfterMaxRetries` passes

**Associated test:** `TestConsumerGroupAck`, `TestConsumerGroupNoAck`, `TestDLQAfterMaxRetries`

---

### TODO #8 — impl: RecoverPending — XAUTOCLAIM for idle messages

**Agente:** @developer

**Description:** Implement `RecoverPending` in `ValkeyConsumer`. When a
service starts, it reclaims messages that have been pending for more than `idleThreshold`
without ACK (they may belong to a previous consumer that died).

**Affected files:**
- `adapters/eventbus/valkey/consumer.go` (add method)

**Behavior:**
```
XAUTOCLAIM stream group consumerName idleMs 0-0 COUNT 100
```
- Reassigns the messages to the current `consumerName`
- Processes each message with the same handler as `Subscribe`
- If handler OK: XACK; if error: same MaxRetries logic as in Subscribe

**Acceptance criteria:**
- `TestPendingRecovery` passes
- Idempotent: calling twice does not duplicate processing

**Associated test:** `TestPendingRecovery`

---

### TODO #9 — infra: Add valkey-go dependency to go.mod and configuration

**Agente:** @devops

**Description:** Add `github.com/valkey-io/valkey-go` to the monorepo go.mod.
Add `github.com/testcontainers/testcontainers-go` for integration tests.
Document environment variables in `.env.example`.

**Affected files:**
- `go.mod` / `go.sum` (via `go get`)
- `.env.example`

**Commands:**
```bash
go get github.com/valkey-io/valkey-go@latest
go get github.com/testcontainers/testcontainers-go@latest
```

**Environment variables to add in `.env.example`:**
```
VALKEY_ADDR=localhost:6379
VALKEY_PASSWORD=           # empty in dev
VALKEY_DLQ_STREAM=dago.dlq
VALKEY_MAX_RETRIES=3
VALKEY_CONSUMER_IDLE_MS=30000
```

**Acceptance criteria:**
- `go build ./adapters/eventbus/...` without errors
- `go test -tags integration ./adapters/eventbus/...` starts container and runs tests

**Associated test:** all integration tests

---

### TODO #10 — infra: Makefile target for eventbus integration tests

**Agente:** @devops

**Description:** Add target `make test-integration-eventbus` in the Makefile to
run only the adapter's integration tests, separated from unit tests.

**Affected files:**
- `Makefile`

**Target to add:**
```makefile
test-integration-eventbus:
	go test -tags integration -count=1 -timeout 120s \
	  ./adapters/eventbus/...

test-integration: test-integration-eventbus
	@echo "All integration tests passed"
```

**Acceptance criteria:**
- `make test-integration-eventbus` runs the 6 tests and all pass
- `make ci` does not run integration tests (only `//go:build integration` activates them)

**Associated test:** all integration tests

---

### TODO #11 — docs: Update docs/index.md and docs/log.md

**Agente:** @docs

**Description:** Register SPRINT-007 in the project index and log.

**Affected files:**
- `docs/index.md` — add row in Sprints table
- `docs/log.md` — add append-only entry

**Acceptance criteria:**
- `grep "SPRINT-007" docs/index.md` returns the row
- `grep "SPRINT-007" docs/log.md` returns the entry
- `docs/index.md` adds section "## Adapters" if it does not exist, with eventbus row

**Associated test:** —

---

## Traceability Matrix

| TODO | Type   | ADR       | Spec              | Test                        | Impl                                  |
|------|--------|-----------|-------------------|-----------------------------|---------------------------------------|
| #1   | spec   | 011       | asyncapi.yaml     | —                           | specs/asyncapi.yaml                   |
| #2   | data   | 001, 011  | asyncapi.yaml     | —                           | libs/domain/events.go                 |
| #3   | impl   | 001, 008  | —                 | —                           | libs/ports/eventbus.go                |
| #4   | test   | 002, 008  | asyncapi.yaml     | integration_test.go (Red)   | —                                     |
| #5   | impl   | 008, 011  | asyncapi.yaml     | TestEnvelopeRoundtrip        | adapters/eventbus/valkey/envelope.go  |
| #6   | impl   | 008       | asyncapi.yaml     | TestPublishAndConsume        | adapters/eventbus/valkey/publisher.go |
| #7   | impl   | 008, 011  | asyncapi.yaml     | TestAck, TestNoAck, TestDLQ | adapters/eventbus/valkey/consumer.go  |
| #8   | impl   | 008       | —                 | TestPendingRecovery          | adapters/eventbus/valkey/consumer.go  |
| #9   | infra  | 008, 013  | —                 | (enables all tests)         | go.mod, .env.example                  |
| #10  | infra  | 002       | —                 | (enables CI execution)      | Makefile                              |
| #11  | docs   | 020       | —                 | —                           | docs/index.md, docs/log.md            |

## Implementation Notes

**Valkey client:** use `github.com/valkey-io/valkey-go` (not `go-redis`). API:
```go
client, err := valkey.NewClient(valkey.ClientOption{
    InitAddress: []string{addr},
})
```

**XPENDING to count retries:** the `delivery-count` field of each PEL entry
(Pending Entry List) returns how many times it was delivered. Use:
```
XPENDING stream group - + 1 consumerName
```
Or use `XRANGE` on the PEL. Alternatively, include a counter in the DLQ envelope
(`retry_count` in the `data` field of the `dago.dlq` event).

**Minimal Testcontainers setup:**
```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    req := testcontainers.ContainerRequest{
        Image:        "valkey/valkey:8",
        ExposedPorts: []string{"6379/tcp"},
        WaitingFor:   wait.ForLog("Ready to accept connections"),
    }
    container, _ := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req, Started: true,
    })
    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "6379")
    valkeyAddr = fmt.Sprintf("%s:%s", host, port.Port())
    os.Exit(m.Run())
}
```

## Result (complete on close)

- [ ] `TestPublishAndConsume` passes
- [ ] `TestConsumerGroupAck` passes
- [ ] `TestConsumerGroupNoAck` passes
- [ ] `TestPendingRecovery` passes
- [ ] `TestDLQAfterMaxRetries` passes
- [ ] `TestEnvelopeRoundtrip` passes
- [ ] `go build ./libs/... ./adapters/eventbus/...` without errors
- [ ] `golangci-lint run ./libs/... ./adapters/eventbus/...` without errors
- [ ] `specs/asyncapi.yaml` validated with 7 channels defined
- [ ] `.env.example` updated with Valkey variables
- [ ] `docs/index.md` and `docs/log.md` updated
