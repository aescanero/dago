# SPRINT-009: Executor — llm_call pattern handler

## Metadata

- **Start date:** 2026-04-30
- **Estimated end date:** 2026-05-02
- **Status:** completed
- **Applied ADRs:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-011, ADR-013, ADR-014, ADR-016, ADR-020
- **Affected specs:** specs/asyncapi.yaml (executor operations), specs/patterns/nodes/llm_call.json
- **Planning agent:** planner
- **Reviewed by:** pending
- **Blocked by:** SPRINT-007 (eventbus), SPRINT-008 (LLMClient + domain errors)
- **Blocks:** SPRINT-010 (orchestrator state machine consumes node.executed)

## Objective

Implement the `executor` service as an event worker: consumes `node.execute.requested`
from the Valkey stream, detects the `llm_call` pattern, builds the LLM request applying
`input_mapping`, invokes `LLMClient.Complete`, applies `output_mapping` on the response
and publishes `node.executed` or `node.execute.failed` based on the result. At completion,
the executor can run `llm_call` nodes end-to-end with Anthropic, Ollama
and `FakeLLMClient` in tests.

## Scope

**Included:**
- Executor operations in `specs/asyncapi.yaml`:
  - `executorConsumeNodeExecuteRequested` (receive)
  - `executorPublishNodeExecuted` (send)
  - `executorPublishNodeExecuteFailed` (send)
- Payload schemas: `NodeExecuteRequestedData`, `NodeExecutedData`, `NodeExecuteFailedData`
- Internal structure of `services/executor/`:
  - `main.go` — wiring: config, ports, consumer startup
  - `internal/handler/node_handler.go` — `NodeHandler` interface
  - `internal/handler/llm_call.go` — `LLMCallHandler` + `LLMCallConfig`
  - `internal/handler/dispatcher.go` — `Dispatcher` dispatches by `pattern`
  - `internal/mapping/input.go` — `ApplyInputMapping`
  - `internal/mapping/output.go` — `ApplyOutputMapping`
  - `internal/consumer/node_execute.go` — `NodeExecuteConsumer`
- Simplified path evaluator: `state.variables.<name>`, `state.messages[-1].content`,
  `output.content`, `output.stop_reason`
- LLM error mapping: `ErrRateLimited`/`ErrProviderUnavailable` → retryable; `ErrUnauthorized` → non-retryable
- 10 unit tests without network (FakeLLMClient + inline fakePublisher)
- 1 integration test (build tag `integration`) with real Valkey
- Environment variables in `.env.example`, `make test-executor` target

**Excluded:**
- Other node patterns (`tool_use`, `react`, `reflection`, `router`, `guardrail`, `subgraph`)
- Token streaming — requires a separate ADR
- Arbitrary expression evaluator for mappings — future sprint
- Execution state persistence in Ent — requires SPRINT-002
- Own retry with backoff — uses SPRINT-007 DLQ policy
- Tests against real Anthropic/Ollama API — excluded from `make ci`
- OpenTelemetry telemetry — dedicated future sprint

## Dependencies

- **Blocked by:**
  - SPRINT-007 (`libs/ports/eventbus.go`, `adapters/eventbus/valkey/`, DLQ, ACK/NACK)
  - SPRINT-008 (`libs/ports/llm.go`, `adapters/llm/anthropic/`, `adapters/llm/ollama/`,
    `adapters/llm/fake/`, errors `ErrUnauthorized`, `ErrRateLimited`, `ErrProviderUnavailable`)
- **Parallel with:** SPRINT-005, SPRINT-006 (dashboard — no file conflicts)
- **Blocks:** executor `tool_use` pattern (SPRINT-010+), end-to-end orchestrator→executor tests

## Behavior Contracts

### C1 — `LLMCallHandler.Handle` — success without mappings

```
Given: node.execute.requested event with pattern="llm_call", config={model:"claude-sonnet-4-6",max_tokens:100}
      FakeLLMClient configured with Responses=[{Content:"ok response",StopReason:"end_turn"}]
When: LLMCallHandler.Handle(ctx, data) is executed
Then: Exactly one event is published to the "node.executed" stream
      variables_update["response"] == "ok response"
      duration_ms >= 0
      Handler returns nil (consumer ACKs)
```

### C2 — `LLMCallHandler.Handle` — ErrRateLimited → retryable

```
Given: FakeLLMClient that returns fmt.Errorf("test: %w", domain.ErrRateLimited)
When: LLMCallHandler.Handle(ctx, data)
Then: Event published to "node.execute.failed" with error_code="rate_limited", retryable=true
      Handler returns error (consumer does NOT ACK → NACK in Valkey)
      No event published to "node.executed"
```

### C3 — `ApplyInputMapping` — state.variables path

```
Given: inputMapping={"user_message":"state.variables.query"}, variables={"query":"what is dago?"}
When: ApplyInputMapping(inputMapping, variables, [])
Then: Returns []ports.Message{{Role:"user", Content:"what is dago?"}}
      Returns no error
      The function is pure: same input → same result
```

## TODOs

### TODO #1 — spec: Define executor operations in specs/asyncapi.yaml

**Agente:** @developer

**Description:** Add the three executor operations to the `operations` section and
the three payloads with typed fields to `components/schemas`. Channels already exist;
what is missing are the complete data schemas and the operation-channel link.

**Affected files:**
- `specs/asyncapi.yaml`

**Operations to add:**
```yaml
operations:
  executorConsumeNodeExecuteRequested:
    action: receive
    channel:
      $ref: '#/channels/nodeExecuteRequested'
    summary: Executor consumes node execution request
  executorPublishNodeExecuted:
    action: send
    channel:
      $ref: '#/channels/nodeExecuted'
    summary: Executor publishes successful node execution result
  executorPublishNodeExecuteFailed:
    action: send
    channel:
      $ref: '#/channels/nodeExecuteFailed'
    summary: Executor publishes node execution failure
```

**Schemas to add:**

`NodeExecuteRequestedData`: `execution_id`, `graph_id`, `node_id`, `node_key`, `pattern`,
`config` (object), `variables` (object), `messages` (array of `{role, content}`), `auth` (string)

`NodeExecutedData`: `execution_id`, `graph_id`, `node_id`, `node_key`,
`output` (object), `variables_update` (object), `duration_ms` (integer)

`NodeExecuteFailedData`: `execution_id`, `graph_id`, `node_id`, `node_key`,
`error` (string), `error_code` (string: "rate_limited"|"provider_unavailable"|"unauthorized"|"execution_error"),
`retryable` (boolean)

**Acceptance criteria:**
- `asyncapi validate specs/asyncapi.yaml` with no errors
- Each operation correctly references its channel
- All three schemas have all fields with correct types

**Associated test:** — (pure spec)

---

### TODO #2 — spec: Verify specs/patterns/nodes/llm_call.json

**Agente:** @developer

**Description:** Review that `llm_call.json` documents defaults (`temperature: 0.7`,
`max_tokens: 2048`) and the scope of supported paths in `input_mapping`/`output_mapping`
for SPRINT-009. Only enrich descriptions — do not change the structure.

**Affected files:**
- `specs/patterns/nodes/llm_call.json`

**Acceptance criteria:**
- Schema documents the defaults and supported paths
- JSON Schema validation reports no errors

**Associated test:** —

---

### TODO #3 — test: Red tests for ApplyInputMapping

**Agente:** @qa

**Description:** Write tests for `mapping/input.go` BEFORE implementing it (Red).

**Affected files:**
- `services/executor/internal/mapping/input_test.go` (new)

**Tests to implement:**
```go
package mapping_test

// TestApplyInputMapping_NoMapping
// Input: inputMapping nil, variables {}, messages [{role:"user",content:"hello"}]
// Expected: []ports.Message{{Role:"user",Content:"hello"}}

// TestApplyInputMapping_EmptyMapping
// Input: inputMapping {}, messages [{role:"user",content:"test"}]
// Expected: messages unchanged (same as without mapping)

// TestApplyInputMapping_StateMessagesLast
// Input: inputMapping {"user_message":"state.messages[-1].content"}
//        messages [{role:"user",content:"first"},{role:"assistant",content:"resp"},{role:"user",content:"second"}]
// Expected: []ports.Message with last message content="second"

// TestApplyInputMapping_StateVariables
// Input: inputMapping {"user_message":"state.variables.query"}
//        variables {"query":"what is dago?"}
// Expected: []ports.Message{{Role:"user",Content:"what is dago?"}}

// TestApplyInputMapping_StateVariables_Missing
// Input: inputMapping {"user_message":"state.variables.nonexistent"}, variables {}
// Expected: descriptive error (variable does not exist)

// TestApplyInputMapping_UnknownPath
// Input: inputMapping {"x":"state.unknown.path"}
// Expected: error (unsupported path)
```

**Acceptance criteria:**
- All 6 tests fail in red (Red confirmed)
- Only import `libs/ports/` — no infrastructure

**Associated test:** this TODO IS the test (Red phase)

---

### TODO #4 — test: Red tests for ApplyOutputMapping

**Agente:** @qa

**Description:** Write tests for `mapping/output.go` BEFORE implementing it (Red).

**Affected files:**
- `services/executor/internal/mapping/output_test.go` (new)

**Tests to implement:**
```go
package mapping_test

// TestApplyOutputMapping_NoMapping
// Input: outputMapping nil, response.Content="generated response"
// Expected: map[string]any{"response":"generated response"}

// TestApplyOutputMapping_ContentToVariable
// Input: outputMapping {"state.variables.summary":"output.content"}, Content="summary"
// Expected: map[string]any{"summary":"summary"}

// TestApplyOutputMapping_StopReasonToVariable
// Input: outputMapping {"state.variables.reason":"output.stop_reason"}, StopReason="max_tokens"
// Expected: map[string]any{"reason":"max_tokens"}

// TestApplyOutputMapping_MultipleTargets
// Input: outputMapping {"state.variables.text":"output.content","state.variables.stop":"output.stop_reason"}
// Expected: map with both keys correct

// TestApplyOutputMapping_UnknownSourcePath
// Input: outputMapping {"state.variables.x":"output.unknown_field"}
// Expected: error (unsupported output field)

// TestApplyOutputMapping_InvalidTargetPath
// Input: outputMapping {"not.a.state.variable":"output.content"}
// Expected: error (target must be state.variables.<name>)
```

**Acceptance criteria:**
- All 6 tests fail in red
- Only import `libs/ports/` (type `LLMResponse`)

**Associated test:** this TODO IS the test (Red phase)

---

### TODO #5 — test: Red tests for LLMCallHandler

**Agente:** @qa

**Description:** Write tests for `handler/llm_call.go` BEFORE implementing it.
Use `FakeLLMClient` (SPRINT-008) and an inline `fakePublisher` defined in the test.

**Affected files:**
- `services/executor/internal/handler/llm_call_test.go` (new)

**Inline fakePublisher (test-private):**
```go
type fakePublisher struct{ published []ports.Event }
func (f *fakePublisher) Publish(_ context.Context, _ ports.PublishOptions, e ports.Event) error {
    f.published = append(f.published, e)
    return nil
}
func (f *fakePublisher) Close() error { return nil }
```

**Tests to implement:**
```go
// TestLLMCallHandler_Success
// Config: {"model":"claude-sonnet-4-6","max_tokens":100}; no mappings
// FakeLLMClient: Content="ok response", StopReason="end_turn"
// Expected: node.executed event published; variables_update["response"]=="ok response"; duration_ms>=0

// TestLLMCallHandler_WithInputMapping
// Config: input_mapping={"user_message":"state.variables.query"}
// Variables: {"query":"explain dago"}
// Expected: FakeLLMClient.Calls[0].Messages[0].Content=="explain dago"

// TestLLMCallHandler_WithOutputMapping
// Config: output_mapping={"state.variables.answer":"output.content"}
// FakeLLMClient: Content="mapped response"
// Expected: variables_update["answer"]=="mapped response"

// TestLLMCallHandler_RateLimited
// FakeLLMClient: returns fmt.Errorf("...%w",domain.ErrRateLimited)
// Expected: node.execute.failed with error_code=="rate_limited", retryable==true; handler returns error

// TestLLMCallHandler_ProviderUnavailable
// FakeLLMClient: returns domain.ErrProviderUnavailable
// Expected: error_code=="provider_unavailable", retryable==true

// TestLLMCallHandler_Unauthorized
// FakeLLMClient: returns domain.ErrUnauthorized
// Expected: error_code=="unauthorized", retryable==false

// TestLLMCallHandler_ExecutionError
// FakeLLMClient: returns errors.New("internal error")
// Expected: error_code=="execution_error", retryable==false
```

**Acceptance criteria:**
- All 7 tests fail in red
- No test imports concrete adapters or calls Valkey or real APIs
- Each test verifies the exact payload of the published event

**Associated test:** this TODO IS the test (Red phase)

---

### TODO #6 — test: Consumer integration test (build tag `integration`)

**Agente:** @qa

**Description:** Test that exercises `NodeExecuteConsumer` with real Valkey. Publishes
`node.execute.requested` and verifies that the consumer processes the message and publishes
`node.executed` on the correct stream. Uses injected `FakeLLMClient`.

**Affected files:**
- `services/executor/internal/consumer/node_execute_test.go` (new, `//go:build integration`)

**Test to implement:**
```go
//go:build integration

// TestExecutorConsumer_LLMCallSuccess
// Precondition: Valkey at EXECUTOR_VALKEY_ADDR (default localhost:6379)
// Setup: create consumer group "executor-group" on "node.execute.requested"
// Action: publish node.execute.requested with pattern="llm_call" and minimal config
// Verification: stream "node.executed" contains event with correct execution_id and node_id
//               and variables_update.response present
// FakeLLMClient injected, timeout: 5 seconds
```

**Acceptance criteria:**
- `go test -tags=integration ./services/executor/...` passes
- Does not run in `make ci`
- Test fails if the consumer does not ACK the message

**Associated test:** this TODO IS the test

---

### TODO #7 — impl: mapping/input.go

**Agente:** @developer

**Description:** Implement `ApplyInputMapping` following the contract from tests #3.
Simplified path evaluator for SPRINT-009.

**Affected files:**
- `services/executor/internal/mapping/input.go` (new)

**Signature:**
```go
package mapping

import "github.com/aescanero/dago/libs/ports"

// ApplyInputMapping builds []ports.Message for LLMRequest.
// If inputMapping is nil or empty, returns messages as-is.
func ApplyInputMapping(
    inputMapping map[string]string,
    variables    map[string]any,
    messages     []ports.Message,
) ([]ports.Message, error)
```

**Supported paths (map values):**
- `state.messages[-1].content` → last message in the `messages` slice
- `state.variables.<name>` → `variables["<name>"]`; error if not found
- Any other path → `fmt.Errorf("mapping/input: unsupported path %q: %w", path, ErrUnsupportedPath)`

**Acceptance criteria:**
- All 6 tests from TODO #3 pass in green
- Function ≤20 lines (ADR-003)
- No infrastructure imports

**Associated test:** TODO #3

---

### TODO #8 — impl: mapping/output.go

**Agente:** @developer

**Description:** Implement `ApplyOutputMapping` following the contract from tests #4.

**Affected files:**
- `services/executor/internal/mapping/output.go` (new)

**Signature:**
```go
package mapping

import "github.com/aescanero/dago/libs/ports"

// ApplyOutputMapping builds map[string]any of variable updates.
// If outputMapping is nil returns {"response": response.Content}.
func ApplyOutputMapping(
    outputMapping map[string]string,
    response      ports.LLMResponse,
) (map[string]any, error)
```

**Supported source fields:** `output.content`, `output.stop_reason`

**Valid target format:** `state.variables.<name>` (extracts `<name>` as the result key)

**Acceptance criteria:**
- All 6 tests from TODO #4 pass in green
- Function ≤20 lines (ADR-003)

**Associated test:** TODO #4

---

### TODO #9 — impl: handler/node_handler.go, handler/llm_call.go, handler/dispatcher.go

**Agente:** @developer

**Description:** Implement the `NodeHandler` interface, `LLMCallHandler` and `Dispatcher`.
The handler receives `LLMClient` and `EventPublisher` via constructor injection.

**Affected files:**
- `services/executor/internal/handler/node_handler.go` (new)
- `services/executor/internal/handler/llm_call.go` (new)
- `services/executor/internal/handler/dispatcher.go` (new)

**Event data types (in `node_handler.go`):**
```go
type NodeExecuteRequestedData struct {
    ExecutionID string          `json:"execution_id"`
    GraphID     string          `json:"graph_id"`
    NodeID      string          `json:"node_id"`
    NodeKey     string          `json:"node_key"`
    Pattern     string          `json:"pattern"`
    Config      json.RawMessage `json:"config"`
    Variables   map[string]any  `json:"variables"`
    Messages    []ports.Message `json:"messages"`
    Auth        string          `json:"auth"`
}

type NodeExecutedData struct {
    ExecutionID     string          `json:"execution_id"`
    GraphID         string          `json:"graph_id"`
    NodeID          string          `json:"node_id"`
    NodeKey         string          `json:"node_key"`
    Output          json.RawMessage `json:"output"`
    VariablesUpdate json.RawMessage `json:"variables_update"`
    DurationMs      int64           `json:"duration_ms"`
}

type NodeExecuteFailedData struct {
    ExecutionID string `json:"execution_id"`
    GraphID     string `json:"graph_id"`
    NodeID      string `json:"node_id"`
    NodeKey     string `json:"node_key"`
    Error       string `json:"error"`
    ErrorCode   string `json:"error_code"`
    Retryable   bool   `json:"retryable"`
}
```

**LLMCallConfig:**
```go
type LLMCallConfig struct {
    Model         string            `json:"model"`
    SystemPrompt  string            `json:"system_prompt"`
    Temperature   float64           `json:"temperature"`
    MaxTokens     int               `json:"max_tokens"`
    InputMapping  map[string]string `json:"input_mapping"`
    OutputMapping map[string]string `json:"output_mapping"`
}
```

**LLMCallHandler.Handle algorithm:**
1. Deserialize `data.Config` → `LLMCallConfig`; defaults: temperature=0.7, max_tokens=2048
2. `ApplyInputMapping(config.InputMapping, data.Variables, data.Messages)` → messages
3. Build `ports.LLMRequest{Model, System, MaxTokens, Temperature, Messages}`
4. `start := time.Now()`
5. `llmClient.Complete(ctx, req)` → on error: map → publish `node.execute.failed` → return error
6. `ApplyOutputMapping(config.OutputMapping, resp)` → variablesUpdate
7. Serialize output and variablesUpdate → publish `node.executed`
8. Return nil

**LLM error → ErrorCode+Retryable mapping:**
- `errors.Is(err, domain.ErrRateLimited)` → "rate_limited", true
- `errors.Is(err, domain.ErrProviderUnavailable)` → "provider_unavailable", true
- `errors.Is(err, domain.ErrUnauthorized)` → "unauthorized", false
- others → "execution_error", false

**Dispatcher:**
```go
type Dispatcher struct{ handlers map[string]NodeHandler }
func NewDispatcher(handlers map[string]NodeHandler) *Dispatcher
func (d *Dispatcher) Dispatch(ctx context.Context, data NodeExecuteRequestedData) error
// If pattern not registered: publish node.execute.failed with error_code="execution_error"
```

**Acceptance criteria:**
- All 7 tests from TODO #5 pass in green
- Handlers only import `libs/ports/` and `libs/domain/` — zero coupling to adapters
- Publish streams: exactly `"node.executed"` and `"node.execute.failed"`
- `LLMCallHandler.Handle` ≤20 lines (extract private functions if needed)

**Associated test:** TODO #5

---

### TODO #10 — impl: consumer/node_execute.go

**Agente:** @developer

**Description:** `NodeExecuteConsumer` subscribes to `node.execute.requested`, deserializes
the event envelope and delegates to `Dispatcher`. ACK on success or non-retryable error;
NACK on retryable error.

**Affected files:**
- `services/executor/internal/consumer/node_execute.go` (new)

**Design:**
```go
type NodeExecuteConsumer struct {
    consumer   ports.EventConsumer
    dispatcher *handler.Dispatcher
}

func NewNodeExecuteConsumer(consumer ports.EventConsumer, d *handler.Dispatcher) *NodeExecuteConsumer

// Run blocks until ctx is cancelled. For each node.execute.requested event:
//   1. Deserialize data as NodeExecuteRequestedData
//   2. If pattern not supported: silent ACK + log
//   3. dispatcher.Dispatch(ctx, data)
//   4. If nil → ACK
//   5. If errors.Is(err, domain.ErrRateLimited||ErrProviderUnavailable) → NACK
//   6. If other error → ACK (failure already published as node.execute.failed)
func (c *NodeExecuteConsumer) Run(ctx context.Context) error
```

**Acceptance criteria:**
- Integration test from TODO #6 passes in green
- Correct ACK for non-retryable errors
- NACK for rate_limited and provider_unavailable

**Associated test:** TODO #6

---

### TODO #11 — impl: services/executor/main.go

**Agente:** @developer

**Description:** Service entry point. Reads config from env, instantiates adapters,
builds Dispatcher with `LLMCallHandler` registered for `"llm_call"`, starts consumer
with signal handling (SIGTERM, SIGINT).

**Affected files:**
- `services/executor/main.go` (new)

**Environment variables:**
- `EXECUTOR_VALKEY_ADDR` (default: `localhost:6379`)
- `EXECUTOR_GROUP` (default: `executor-group`)
- `EXECUTOR_CONSUMER_NAME` (default: `executor-1`)
- `EXECUTOR_LLM_PROVIDER` (default: `anthropic`; values: `anthropic`, `ollama`)
- `EXECUTOR_BLOCK_DURATION_MS` (default: `5000`)
- If `anthropic`: reads `ANTHROPIC_API_KEY`
- If `ollama`: reads `OLLAMA_BASE_URL` (default `http://localhost:11434`)

**Acceptance criteria:**
- `go build ./services/executor/` compiles without errors
- Service shuts down cleanly on SIGTERM

**Associated test:** smoke — `go build ./services/executor/`

---

### TODO #12 — infra: Makefile + .env.example

**Agente:** @devops

**Description:** `make test-executor` target for unit tests. Executor environment variables
in `.env.example`.

**Affected files:**
- `Makefile`
- `.env.example`

**Target:**
```makefile
## test-executor: executor service unit tests
test-executor:
	go test -count=1 -timeout 30s ./services/executor/...
```

**Variables in `.env.example`:**
```
# Executor
EXECUTOR_VALKEY_ADDR=localhost:6379
EXECUTOR_GROUP=executor-group
EXECUTOR_CONSUMER_NAME=executor-1
EXECUTOR_LLM_PROVIDER=anthropic
EXECUTOR_BLOCK_DURATION_MS=5000
```

**Acceptance criteria:**
- `make test-executor` passes 10 unit tests without network or real credentials

**Associated test:** all executor unit tests

---

### TODO #13 — docs: Update docs/index.md and docs/log.md

**Agente:** @docs

**Description:** Register SPRINT-009 in the index and log. Annotate in the Services table
that `executor` has partial implementation (pattern `llm_call`).

**Affected files:**
- `docs/index.md`
- `docs/log.md`

**Acceptance criteria:**
- `grep "SPRINT-009" docs/index.md` returns the row
- `grep "SPRINT-009" docs/log.md` returns the entry

**Associated test:** —

---

## Traceability Matrix

| TODO | Type  | ADR              | Spec                             | Test                                   | Impl                                    |
|------|-------|------------------|----------------------------------|----------------------------------------|-----------------------------------------|
| #1   | spec  | 011, 014         | asyncapi.yaml                    | —                                      | specs/asyncapi.yaml                     |
| #2   | spec  | 016              | llm_call.json                    | —                                      | specs/patterns/nodes/llm_call.json      |
| #3   | test  | 002, 016         | llm_call.json (input_mapping)    | input_test.go (6 tests, Red)           | —                                       |
| #4   | test  | 002, 016         | llm_call.json (output_mapping)   | output_test.go (6 tests, Red)          | —                                       |
| #5   | test  | 002, 003         | asyncapi.yaml, llm_call.json     | llm_call_test.go (7 tests, Red)        | —                                       |
| #6   | test  | 002, 011         | asyncapi.yaml                    | node_execute_test.go (1 test, Red)     | —                                       |
| #7   | impl  | 001, 003, 004    | llm_call.json (input_mapping)    | TestApplyInputMapping_* (Green)        | mapping/input.go                        |
| #8   | impl  | 001, 003, 004    | llm_call.json (output_mapping)   | TestApplyOutputMapping_* (Green)       | mapping/output.go                       |
| #9   | impl  | 001, 003, 004    | asyncapi.yaml, llm_call.json     | TestLLMCallHandler_* (Green)           | handler/{node_handler,llm_call,dispatcher}.go |
| #10  | impl  | 001, 011, 014    | asyncapi.yaml                    | TestExecutorConsumer_LLMCallSuccess    | consumer/node_execute.go                |
| #11  | impl  | 001, 013, 014    | —                                | go build smoke                         | services/executor/main.go               |
| #12  | infra | 002, 013         | —                                | make test-executor                     | Makefile, .env.example                  |
| #13  | docs  | 020              | —                                | —                                      | docs/index.md, docs/log.md              |

## Implementation notes

**TODO order by dependencies:**
```
#1 (spec asyncapi) ─┐
#2 (spec llm_call)  ┘ parallelizable
         ↓
#3 (test input)  + #4 (test output)  ← parallelizable
         ↓
#5 (test LLMCallHandler)
         ↓
#7 (impl input) + #8 (impl output)  ← parallelizable
         ↓
#9 (impl handler)
         ↓
#6 (test consumer) + #10 (impl consumer)  ← in parallel if consumer stub available
         ↓
#11 (impl main.go)
         ↓
#12 (infra) + #13 (docs)
```

**Critical architecture rules:**
- `services/executor/internal/` is service-private code — no other service imports it
- Handlers only import `libs/ports/` and `libs/domain/` (not `adapters/`)
- Concrete adapters are only instantiated in `main.go`
- The consumer imports `libs/ports/` + `internal/handler/` + `internal/mapping/`

**FakeEventPublisher in unit tests:**
The `fakePublisher` is defined as a private type in each `_test.go` file that needs it.
It is not exported until another handler needs it — at that point it is extracted to
`adapters/eventbus/fake/`.

**LLMClient selection:**
Provider selection happens in `main.go` based on `EXECUTOR_LLM_PROVIDER`, not inside
the handler. The handler receives a single already-configured `ports.LLMClient`.

**ACK/NACK in the consumer:**
The consumer uses `errors.Is` on domain sentinels to determine retryability,
not the `Retryable` field of the published event. Non-retryable errors receive ACK so
they are not retried (the `node.execute.failed` was already published for the orchestrator to act).

**Sprint commits:**
```
spec: add executor operations and data schemas to asyncapi.yaml [SPRINT-009 #1]
test: add unit tests for ApplyInputMapping [SPRINT-009 #3]
test: add unit tests for ApplyOutputMapping [SPRINT-009 #4]
test: add unit tests for LLMCallHandler [SPRINT-009 #5]
feat: implement ApplyInputMapping [SPRINT-009 #7]
feat: implement ApplyOutputMapping [SPRINT-009 #8]
feat: implement LLMCallHandler and Dispatcher [SPRINT-009 #9]
feat: implement NodeExecuteConsumer [SPRINT-009 #10]
feat: wire executor main.go [SPRINT-009 #11]
chore: add test-executor Makefile target and env vars [SPRINT-009 #12]
docs: update index.md and log.md for SPRINT-009 [SPRINT-009 #13]
```

## Result

**Status:** completed — 2026-05-09
**Reviewed by:** @reviewer (Claude Code)

- [x] `TestApplyInputMapping_NoMapping` passes
- [x] `TestApplyInputMapping_StateMessagesLast` passes
- [x] `TestApplyInputMapping_StateVariables` passes
- [x] `TestApplyInputMapping_StateVariables_Missing` passes — descriptive error
- [x] `TestApplyOutputMapping_NoMapping` passes — default {"response": content}
- [x] `TestApplyOutputMapping_ContentToVariable` passes
- [x] `TestApplyOutputMapping_MultipleTargets` passes
- [x] `TestLLMCallHandler_Success` passes — node.executed published with variables_update
- [x] `TestLLMCallHandler_WithInputMapping` passes — LLMRequest.Messages[0].Content correct
- [x] `TestLLMCallHandler_WithOutputMapping` passes — variables_update mapped correctly
- [x] `TestLLMCallHandler_RateLimited` passes — error_code=="rate_limited", retryable==true
- [x] `TestLLMCallHandler_ProviderUnavailable` passes — retryable==true
- [x] `TestLLMCallHandler_Unauthorized` passes — retryable==false
- [x] `TestLLMCallHandler_ExecutionError` passes — error_code=="execution_error", retryable==false
- [x] `TestExecutorConsumer_LLMCallSuccess` implemented (build tag integration, requires Valkey)
- [x] `go build ./services/executor/cmd/` no errors
- [x] `golangci-lint run ./services/executor/...` — 0 issues
- [x] `make test-executor` runs 19 unit tests (12 mapping + 7 handler), all pass, no network
- [x] `specs/asyncapi.yaml` — 3 channels, 3 operations, 3 data schemas added
- [x] `docs/index.md` and `docs/log.md` updated

**Deviations from plan:**
- `make test-executor` runs 19 tests (not 10 as estimated); 12 mapping + 7 handler.
- Build path is `./services/executor/cmd/` not `./services/executor/` (consistent with all other services).
- FakeLLMClient was extended with `Errors []error` field (not in original SPRINT-008 scope) to enable handler error-injection tests.
- Integration test (TODO #6) is implemented but not run in CI (build tag: integration).
