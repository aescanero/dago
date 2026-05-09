# SPRINT-008: LLM Adapter — LLMClient Port, Anthropic and Ollama/Mixtral

## Metadata

- **Start date:** 2026-04-29
- **Estimated end date:** 2026-04-30
- **Status:** completed
- **ADRs applied:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-013, ADR-015, ADR-016
- **Affected specs:** specs/patterns/nodes/llm_call.json (reference; no new spec created in this sprint)
- **Planning agent:** planner
- **Reviewed by:** pending
- **Blocked by:** SPRINT-001 (go.mod)
- **Blocks:** SPRINT-009 (executor uses LLMClient), SPRINT-010 (orchestrator state machine)

## Objective

Implement the `LLMClient` port in `libs/ports/llm.go` and two adapters: Anthropic
(`adapters/llm/anthropic/`) and Ollama with OpenAI-compatible API (`adapters/llm/ollama/`,
default model `mixtral`). Includes a deterministic `FakeLLMClient` for tests. At
completion, the executor can invoke Anthropic claude-sonnet or Mixtral via Ollama through
the same `LLMClient` interface without coupling the domain to any concrete SDK.

## Scope

**Included:**
- Three new domain errors in `libs/domain/errors.go`:
  `ErrUnauthorized`, `ErrRateLimited`, `ErrProviderUnavailable`
- `LLMClient` port with types `Message`, `ToolDefinition`, `LLMRequest`,
  `LLMResponse`, `ToolUse` in `libs/ports/llm.go`
- Anthropic adapter in `adapters/llm/anthropic/`:
  - `client.go` — `AnthropicClient` that implements `LLMClient`
  - `convert.go` — `ports` ↔ Anthropic SDK types conversion
  - `errors.go` — HTTP status → domain errors mapping
  - `client_test.go` — 7 unit tests with `httptest.NewServer`
- Deterministic fake in `adapters/llm/fake/`:
  - `client.go` — `FakeLLMClient` with response queue and call registry
  - `client_test.go` — 1 test `TestFakeLLMClientQueue`
- Dependencies `github.com/anthropics/anthropic-sdk-go` and `github.com/sashabaranov/go-openai` in `go.mod`
- Ollama adapter (OpenAI-compatible API) in `adapters/llm/ollama/`:
  - `client.go` — `OllamaClient` that implements `LLMClient`; `NewOllamaClient` does not return error (BaseURL has default `http://localhost:11434`)
  - `convert.go` — `ports` ↔ `go-openai` types conversion; `convertFinishReason` maps "stop"→"end_turn", "tool_calls"→"tool_use", "length"→"max_tokens"
  - `errors.go` — HTTP 500 → `ErrProviderUnavailable`
  - `client_test.go` — 6 unit tests with `httptest.NewServer`
- Default Ollama model: `mixtral` (Mixtral 8x7B)
- Environment variables documented in `.env.example`
- Target `make test-llm` in `Makefile` (14 tests total)
- Update of `docs/index.md` and `docs/log.md`

**Not included:**
- Other LLM providers (native OpenAI, Google Vertex) — future sprints
- Token streaming (chunked responses) — requires separate ADR
- Integration with the executor — SPRINT-010 or later
- LLM response cache — future sprint
- Integration tests against the real Anthropic API — excluded from `make ci`
- Automatic retry with own backoff — delegated to the SDK (`ANTHROPIC_MAX_RETRIES`)

## Dependencies

- **Blocked by:** SPRINT-001 (valid go.mod with compilable monorepo structure)
- **Parallel with:** SPRINT-002, SPRINT-003, SPRINT-004, SPRINT-007 (no dependency between them)
- **Blocks:** executor (`llm_call`, `react`, `reflection` nodes use `LLMClient`),
  LLM router (uses `LLMClient` for LLM-based routing)

## Behavior Contracts

### C1 — `AnthropicClient.Complete` — text response

```
Given: LLMRequest with messages=[{role:"user",content:"hello"}], model="claude-sonnet-4-6", max_tokens=100
When: AnthropicClient.Complete(ctx, req) — mock HTTP returns 200 with text content
Then: LLMResponse.Content is not empty
      LLMResponse.StopReason == "end_turn"
      LLMResponse.InputTokens > 0 and OutputTokens > 0
      No error is returned
```

### C2 — `AnthropicClient.Complete` — rate limit

```
Given: The mock HTTP server returns HTTP 429
When: AnthropicClient.Complete(ctx, req)
Then: An error is returned such that errors.Is(err, domain.ErrRateLimited) == true
      The caller can retry with exponential backoff
      The returned LLMResponse is empty
```

### C3 — `FakeLLMClient.Complete` — FIFO queue and call registry

```
Given: FakeLLMClient{Responses: [resp1, resp2]}
When: Complete() is called three consecutive times
Then: First call returns resp1, second returns resp2
      Third call (empty queue) returns {Content:"fake response", StopReason:"end_turn"}
      Calls contains exactly 3 LLMRequests in insertion order
      Complete never returns error
```

## TODOs

### TODO #1 — data: Add LLM domain errors to libs/domain/errors.go

**Agente:** @developer

**Description:** Add three sentinel errors to the domain errors file.
Pure types without infrastructure dependency. The Anthropic adapter will return them
wrapped with `fmt.Errorf`. The executor will use `errors.Is` to differentiate the
recovery action.

**Affected files:**
- `libs/domain/errors.go` (new or existing — add if already exists)

**Types to implement:**
```go
package domain

import "errors"

var (
    // ErrUnauthorized indicates invalid LLM provider credentials (HTTP 401).
    ErrUnauthorized = errors.New("unauthorized")

    // ErrRateLimited indicates that the LLM provider rejected the request due to
    // rate limiting (HTTP 429). The caller can retry with exponential backoff.
    ErrRateLimited = errors.New("rate limited")

    // ErrProviderUnavailable indicates that the LLM provider is temporarily
    // unavailable (HTTP 500/529). Retryable.
    ErrProviderUnavailable = errors.New("provider unavailable")
)
```

**Acceptance criteria:**
- `go build ./libs/domain/...` without errors
- No infrastructure imports
- The three errors are distinguishable with `errors.Is`

**Associated test:** covered indirectly by adapter tests in TODO #3

---

### TODO #2 — impl: Define LLMClient interface in libs/ports/llm.go

**Agente:** @developer

**Description:** Define the output port for LLM calls. The port is the
boundary between domain and infrastructure. `LLMRequest` and `LLMResponse` are
portable types — they do not contain Anthropic SDK types. `ToolDefinition.InputSchema` is
`json.RawMessage` to transport an arbitrary JSON Schema without deserializing it.

**Affected files:**
- `libs/ports/llm.go` (new)

**Interface to implement:**
```go
package ports

import (
    "context"
    "encoding/json"
)

// Message represents a conversation turn.
type Message struct {
    Role      string // "user" | "assistant" | "tool_result"
    Content   string
    ToolUseID string // only for Role == "tool_result"
}

// ToolDefinition describes a tool available to the LLM.
type ToolDefinition struct {
    Name        string
    Description string
    InputSchema json.RawMessage // JSON Schema of the tool input
}

// LLMRequest encapsulates a call to the LLM.
type LLMRequest struct {
    Model       string
    System      string
    Messages    []Message
    Tools       []ToolDefinition
    MaxTokens   int
    Temperature float64
}

// ToolUse represents a tool invocation requested by the LLM.
type ToolUse struct {
    ID    string
    Name  string
    Input json.RawMessage
}

// LLMResponse encapsulates the LLM response.
type LLMResponse struct {
    Content      string    // generated text; empty if StopReason == "tool_use"
    StopReason   string    // "end_turn" | "tool_use" | "max_tokens"
    ToolUses     []ToolUse // populated if StopReason == "tool_use"
    InputTokens  int
    OutputTokens int
}

// LLMClient is the output port for calls to language models.
type LLMClient interface {
    Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
```

**Acceptance criteria:**
- `go build ./libs/ports/...` without errors
- Only imports `context`, `encoding/json` and nothing from infrastructure
- `LLMClient` has a single method (ADR-003: single responsibility)

**Associated test:** —

---

### TODO #3 — test: Red tests for the Anthropic adapter with httptest mock server

**Agente:** @qa

**Description:** Write the 7 Anthropic adapter tests BEFORE implementing it
(TDD Red cycle). They use `net/http/httptest.NewServer` as mock for the Anthropic endpoint.
`ANTHROPIC_BASE_URL` points to the mock server. When run against empty stubs they must
fail with compilation error or "not implemented".

**Affected files:**
- `adapters/llm/anthropic/client_test.go` (new)

**Tests to implement:**
```go
package anthropic_test

// TestCompleteText
// Scenario: LLMRequest without tools, mock returns 200 with text content.
// Verifies: LLMResponse.Content == expected text, StopReason == "end_turn",
//           InputTokens > 0, OutputTokens > 0.

// TestCompleteWithTools
// Scenario: LLMRequest with ToolDefinition, mock returns 200 with
//            stop_reason "tool_use" and a tool_use block.
// Verifies: StopReason == "tool_use", len(ToolUses) == 1,
//           ToolUses[0].Name == expected name, Content == "".

// TestCompleteToolResult
// Scenario: 3-turn conversation (user → assistant tool_use →
//            tool_result → assistant end_turn).
// Verifies: StopReason == "end_turn", Content != "".

// TestCompleteRateLimit
// Scenario: mock returns HTTP 429.
// Verifies: errors.Is(err, domain.ErrRateLimited).

// TestCompleteServerError
// Scenario: mock returns HTTP 500.
// Verifies: errors.Is(err, domain.ErrProviderUnavailable).

// TestCompleteUnauthorized
// Scenario: mock returns HTTP 401.
// Verifies: errors.Is(err, domain.ErrUnauthorized).

// TestCompleteContextTimeout
// Scenario: ctx with 10ms deadline, mock sleeps 100ms.
// Verifies: error wraps context.DeadlineExceeded.
```

**Acceptance criteria:**
- `go test ./adapters/llm/anthropic/...` fails with compilation or "not implemented" (Red)
- No mock frameworks — only `httptest.NewServer` and manual fakes
- Each test creates its own `httptest.NewServer`, independent from each other

**Associated test:** this TODO IS the test (Red phase)

---

### TODO #4 — test: FakeLLMClient Red test

**Agente:** @qa

**Description:** Write the fake test BEFORE implementing it. Verifies the
FIFO response queue and call registry.

**Affected files:**
- `adapters/llm/fake/client_test.go` (new)

**Test to implement:**
```go
package fake_test

// TestFakeLLMClientQueue
// Scenario: FakeLLMClient with Responses = [resp1, resp2].
// Verifies:
//   - First Complete call → resp1
//   - Second call → resp2
//   - Third call (empty queue) → LLMResponse{Content: "fake response", StopReason: "end_turn"}
//   - Calls contains 3 LLMRequests in order
//   - Returns no error in any case
```

**Acceptance criteria:**
- `go test ./adapters/llm/fake/...` fails with compilation (Red confirmed)
- Verifies both returned values and the `Calls` registry

**Associated test:** this TODO IS the test (Red phase)

---

### TODO #5 — impl: FakeLLMClient in adapters/llm/fake/client.go

**Agente:** @developer

**Description:** Implement the deterministic fake for use in executor tests.
No external dependencies — stdlib only. Implements `ports.LLMClient`.

**Affected files:**
- `adapters/llm/fake/client.go` (new)

**Design:**
```go
package fake

import (
    "context"
    "github.com/aescanero/dago/libs/ports"
)

// FakeLLMClient implements ports.LLMClient for tests.
// Returns responses from Responses in FIFO order.
// When the queue is exhausted, returns the default response.
type FakeLLMClient struct {
    Responses []ports.LLMResponse
    Calls     []ports.LLMRequest
}

// Complete returns the next response or the default one.
// Always records the call in Calls. Never returns error.
func (f *FakeLLMClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error)
```

**Default response:** `ports.LLMResponse{Content: "fake response", StopReason: "end_turn"}`

**Interface verification:** `var _ ports.LLMClient = &FakeLLMClient{}`

**Acceptance criteria:**
- `TestFakeLLMClientQueue` passes (Green)
- `go build ./adapters/llm/fake/...` without errors
- No external SDK imports

**Associated test:** `TestFakeLLMClientQueue`

---

### TODO #6 — impl: Type conversion in adapters/llm/anthropic/convert.go

**Agente:** @developer

**Description:** Pure translation functions between port types and Anthropic SDK
types. No state, no IO. Separating the conversion from the client allows testing it
in isolation.

**Affected files:**
- `adapters/llm/anthropic/convert.go` (new)

**Functions to implement:**
```go
package anthropic

// toAnthropicMessages converts []ports.Message → []anthropic.MessageParam.
// Handles roles: "user", "assistant", "tool_result".
// For "tool_result": builds ToolResultBlockParam with ToolUseID.
func toAnthropicMessages(messages []ports.Message) []anthropic.MessageParam

// toAnthropicTools converts []ports.ToolDefinition → []anthropic.ToolParam.
// InputSchema is passed directly as json.RawMessage to the SDK.
func toAnthropicTools(tools []ports.ToolDefinition) []anthropic.ToolParam

// fromAnthropicResponse converts anthropic.Message → ports.LLMResponse.
// Extracts Content from the first block of type text.
// Extracts ToolUses from blocks of type tool_use.
// Maps stop_reason: "end_turn" | "tool_use" | "max_tokens".
func fromAnthropicResponse(msg anthropic.Message) ports.LLMResponse
```

**Acceptance criteria:**
- `go build ./adapters/llm/anthropic/...` without errors
- Each function ≤20 lines (ADR-003)
- `TestCompleteText`, `TestCompleteWithTools` and `TestCompleteToolResult` pass

**Associated test:** `TestCompleteText`, `TestCompleteWithTools`, `TestCompleteToolResult`

---

### TODO #7 — impl: Error mapping in adapters/llm/anthropic/errors.go

**Agente:** @developer

**Description:** Function that maps Anthropic SDK errors (by HTTP code) to
domain errors. The adapter never exposes SDK types to the domain.

**Affected files:**
- `adapters/llm/anthropic/errors.go` (new)

**Function to implement:**
```go
package anthropic

// mapAnthropicError converts an SDK error into a domain error.
// If the error is *anthropicsdk.Error, maps by StatusCode.
// context.DeadlineExceeded and context.Canceled are propagated without wrapping.
func mapAnthropicError(op string, err error) error
```

**Mapping:**
- 401 → `fmt.Errorf("%s: %w", op, domain.ErrUnauthorized)`
- 429 → `fmt.Errorf("%s: %w", op, domain.ErrRateLimited)`
- 500, 529 → `fmt.Errorf("%s: %w", op, domain.ErrProviderUnavailable)`
- `context.DeadlineExceeded` / `context.Canceled` → propagate without wrapping
- others → `fmt.Errorf("%s: %w", op, err)`

**Acceptance criteria:**
- `TestCompleteRateLimit`, `TestCompleteServerError`, `TestCompleteUnauthorized` pass
- `errors.Is(result, domain.ErrRateLimited)` true for 429 error
- `context.DeadlineExceeded` is not wrapped in domain error

**Associated test:** `TestCompleteRateLimit`, `TestCompleteServerError`,
`TestCompleteUnauthorized`, `TestCompleteContextTimeout`

---

### TODO #8 — impl: AnthropicClient in adapters/llm/anthropic/client.go

**Agente:** @developer

**Description:** Implement `AnthropicClient` that implements `ports.LLMClient`.
Configuration via environment variables. Delegates serialization to the SDK and type
translation to `convert.go`. `Complete` does not exceed 20 lines.

**Affected files:**
- `adapters/llm/anthropic/client.go` (new)

**Design:**
```go
package anthropic

// Config contains the Anthropic client configuration.
type Config struct {
    APIKey     string // ANTHROPIC_API_KEY (required)
    BaseURL    string // ANTHROPIC_BASE_URL (optional)
    MaxRetries int    // ANTHROPIC_MAX_RETRIES (default 2)
}

// AnthropicClient implements ports.LLMClient using the official Anthropic SDK.
type AnthropicClient struct {
    client       *anthropicsdk.Client
    defaultModel string
}

// NewAnthropicClient builds an AnthropicClient. Returns error if APIKey is empty.
func NewAnthropicClient(cfg Config) (*AnthropicClient, error)

// Complete sends the request to the Anthropic API and returns the domain response.
func (c *AnthropicClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error)
```

**Complete behavior:**
1. Build `MessageNewParams` using functions from `convert.go`
2. Call `c.client.Messages.New(ctx, params)`
3. If error: `mapAnthropicError("anthropic complete", err)` and return
4. Convert response with `fromAnthropicResponse(msg)`
5. Return `LLMResponse, nil`

**Default model:** `claude-sonnet-4-6` (used when `LLMRequest.Model == ""`).

**Interface verification:** `var _ ports.LLMClient = &AnthropicClient{}`

**Acceptance criteria:**
- All 7 tests from `client_test.go` pass (Green)
- `go build ./adapters/llm/anthropic/...` without errors
- `golangci-lint run ./adapters/llm/anthropic/...` without errors
- `Complete` ≤20 lines (ADR-003)

**Associated test:** all tests from `adapters/llm/anthropic/client_test.go`

---

### TODO #9 — infra: Add anthropic-sdk-go dependency and environment variables

**Agente:** @devops

**Description:** Add the official Anthropic SDK to go.mod. Document environment
variables in `.env.example`. Tests use `httptest.NewServer` — they do not need
real credentials.

**Affected files:**
- `go.mod` / `go.sum` (via `go get`)
- `.env.example`

**Commands:**
```bash
go get github.com/anthropics/anthropic-sdk-go@latest
go mod tidy
```

**Variables to add in `.env.example`:**
```
# Anthropic LLM
ANTHROPIC_API_KEY=sk-ant-...          # required in production
ANTHROPIC_BASE_URL=                   # empty → api.anthropic.com; tests: http://localhost:PORT
ANTHROPIC_MAX_RETRIES=2
```

**Acceptance criteria:**
- `go build ./adapters/llm/...` without errors
- `go test ./adapters/llm/...` runs 8 tests without network or real credentials
- `.env.example` has the three variables with comments

**Associated test:** enables compilation and execution of all tests

---

### TODO #10 — infra: make test-llm target in Makefile

**Agente:** @devops

**Description:** Add target `make test-llm` to run LLM adapter unit tests.
Unit tests (not integration) → included in `make ci`.

**Affected files:**
- `Makefile`

**Targets to add:**
```makefile
## test-llm: unit tests for the LLM adapter (anthropic + fake)
test-llm:
	go test -count=1 -timeout 30s ./adapters/llm/...
```

Add `test-llm` as a dependency of the existing `test` target.

**Acceptance criteria:**
- `make test-llm` runs 8 tests, all pass
- `make ci` includes `make test-llm` (without real credentials)

**Associated test:** all tests in `adapters/llm/`

---

### TODO #11 — docs: Update docs/index.md and docs/log.md

**Agente:** @docs

**Description:** Register SPRINT-008 in the index and log. Add section
`## Adapters` in `docs/index.md` with Event Bus Valkey (SPRINT-007) and LLM
Anthropic+Fake+Ollama (SPRINT-008).

**Affected files:**
- `docs/index.md`
- `docs/log.md`

**Acceptance criteria:**
- `grep "SPRINT-008" docs/index.md` returns at least two rows
- `grep "SPRINT-008" docs/log.md` returns the entry
- Section `## Adapters` lists the four planned adapters

**Associated test:** —

---

### TODO #12 — infra: Add go-openai dependency and Ollama environment variables

**Agente:** @devops

**Description:** Add the OpenAI-compatible client `sashabaranov/go-openai` to go.mod.
This SDK is reusable for the native OpenAI adapter in future sprints. Document
the two Ollama variables in `.env.example`. Tests use `httptest.NewServer` —
they do not require a real Ollama instance or any API key.

**Affected files:**
- `go.mod` / `go.sum` (via `go get`)
- `.env.example`

**Commands:**
```bash
go get github.com/sashabaranov/go-openai@latest
go mod tidy
```

**Variables to add in `.env.example`:**
```
# Ollama LLM
OLLAMA_BASE_URL=http://localhost:11434   # default local; tests: http://127.0.0.1:PORT
OLLAMA_DEFAULT_MODEL=mixtral             # Mixtral 8x7B model (alternative: mixtral:8x7b)
```

**Acceptance criteria:**
- `go build ./adapters/llm/ollama/...` without errors after adding the dependency
- `go test ./adapters/llm/ollama/...` does not require network or credentials
- `.env.example` has the two variables with comments

**Associated test:** enables compilation and execution of TODO #13 tests

---

### TODO #13 — test: Red tests for the Ollama adapter with httptest mock server

**Agente:** @qa

**Description:** Write the 6 Ollama adapter tests BEFORE implementing it
(TDD Red cycle). The mock server simulates Ollama's OpenAI-compatible API at
`/v1/chat/completions`. Each test creates its own independent `httptest.NewServer`.

**Affected files:**
- `adapters/llm/ollama/client_test.go` (new)

**Tests to implement:**
```go
package ollama_test

// TestOllamaCompleteText
// Scenario: LLMRequest without tools, mock returns 200 with choices[0].message.content
//            and finish_reason "stop".
// Verifies: Content == expected text, StopReason == "end_turn",
//           InputTokens > 0, OutputTokens > 0.

// TestOllamaCompleteWithTools
// Scenario: LLMRequest with ToolDefinition, mock returns 200 with
//            finish_reason "tool_calls" and choices[0].message.tool_calls.
// Verifies: StopReason == "tool_use", len(ToolUses) == 1,
//           ToolUses[0].Name == expected name, Content == "".

// TestOllamaCompleteServerError
// Scenario: mock returns HTTP 500.
// Verifies: errors.Is(err, domain.ErrProviderUnavailable) is true.

// TestOllamaCompleteContextTimeout
// Scenario: ctx with 10ms deadline, mock sleeps 100ms before responding.
// Verifies: returned error wraps context.DeadlineExceeded.

// TestOllamaCompleteModelDefault
// Scenario: LLMRequest with Model == "", mock verifies the received body.
// Verifies: the "model" field in the JSON sent to mock is "mixtral".

// TestOllamaConvertFinishReason
// Scenario: table of cases for convertFinishReason.
// Verifies: "stop"→"end_turn", "tool_calls"→"tool_use",
//           "length"→"max_tokens", ""→"end_turn" (fallback).
```

**Acceptance criteria:**
- `go test ./adapters/llm/ollama/...` fails with compilation or "not implemented" (Red)
- No mock frameworks — only `httptest.NewServer` and case tables
- Each test is independent, does not share state

**Associated test:** this TODO IS the test (Red phase)

---

### TODO #14 — impl: Type conversion in adapters/llm/ollama/convert.go

**Agente:** @developer

**Description:** Pure translation functions between port types and `go-openai` SDK
types. No state, no IO. `convertFinishReason` maps `finish_reason` from the
OpenAI-compatible API to standard domain values.

**Affected files:**
- `adapters/llm/ollama/convert.go` (new)

**Functions to implement:**
```go
package ollama

// toOpenAIMessages converts []ports.Message to []openai.ChatCompletionMessage.
// Roles: "user"→ChatMessageRoleUser, "assistant"→ChatMessageRoleAssistant,
//        "tool_result"→ChatMessageRoleTool (with ToolCallID).
func toOpenAIMessages(messages []ports.Message) []openai.ChatCompletionMessage

// toOpenAITools converts []ports.ToolDefinition to []openai.Tool.
// Each tool: openai.Tool{Type:"function", Function:{Name, Description, Parameters}}.
// InputSchema is passed directly as Parameters (json.RawMessage).
func toOpenAITools(tools []ports.ToolDefinition) []openai.Tool

// fromOpenAIResponse converts openai.ChatCompletionResponse to ports.LLMResponse.
// Content: choices[0].Message.Content.
// ToolUses: choices[0].Message.ToolCalls → ports.ToolUse{ID, Name, Input}.
// StopReason: via convertFinishReason(choices[0].FinishReason).
// Tokens: Usage.PromptTokens, Usage.CompletionTokens.
func fromOpenAIResponse(resp openai.ChatCompletionResponse) ports.LLMResponse

// convertFinishReason maps finish_reason → domain StopReason.
// "stop"→"end_turn", "tool_calls"→"tool_use", "length"→"max_tokens", others→"end_turn".
func convertFinishReason(reason string) string
```

**Acceptance criteria:**
- `go build ./adapters/llm/ollama/...` without errors
- Each function ≤20 lines (ADR-003)
- `TestOllamaCompleteText`, `TestOllamaCompleteWithTools`, `TestOllamaConvertFinishReason` pass

**Associated test:** `TestOllamaCompleteText`, `TestOllamaCompleteWithTools`, `TestOllamaConvertFinishReason`

---

### TODO #15 — impl: OllamaClient + error mapping in adapters/llm/ollama/

**Agente:** @developer

**Description:** Implement `OllamaClient` that satisfies `ports.LLMClient` using
Ollama's OpenAI-compatible API. Since Ollama does not require an API key, an empty string
is passed to the SDK. `NewOllamaClient` does not return error because BaseURL always has a
valid default value.

**Affected files:**
- `adapters/llm/ollama/client.go` (new)
- `adapters/llm/ollama/errors.go` (new)

**client.go design:**
```go
package ollama

// Config contains the Ollama client configuration.
type Config struct {
    BaseURL      string // OLLAMA_BASE_URL (default: http://localhost:11434)
    DefaultModel string // OLLAMA_DEFAULT_MODEL (default: mixtral)
}

// OllamaClient implements ports.LLMClient using Ollama's OpenAI-compatible API.
type OllamaClient struct {
    client       *openai.Client
    defaultModel string
}

// NewOllamaClient builds an OllamaClient.
// If BaseURL is empty, uses "http://localhost:11434".
// If DefaultModel is empty, uses "mixtral".
func NewOllamaClient(cfg Config) *OllamaClient

// Complete sends the request to Ollama and returns the domain response.
func (c *OllamaClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error)

var _ ports.LLMClient = &OllamaClient{}
```

**Complete behavior:**
1. If `req.Model == ""` use `c.defaultModel`
2. Build `openai.ChatCompletionRequest` using `convert.go`
3. Call `c.client.CreateChatCompletion(ctx, chatReq)`
4. If error: `mapOllamaError("ollama complete", err)` and return
5. `fromOpenAIResponse(resp)` and return

**errors.go design — mapping:**
- `*openai.APIError` with StatusCode 500 → `fmt.Errorf("%s: %w", op, domain.ErrProviderUnavailable)`
- `context.DeadlineExceeded` / `context.Canceled` → propagate without wrapping
- others → `fmt.Errorf("%s: %w", op, err)`

**Note:** Local Ollama does not generate 401/429; if exposed behind an authenticated proxy,
`mapOllamaError` can be extended without changing the interface.

**Acceptance criteria:**
- All 6 tests from `ollama/client_test.go` pass (Green)
- `go build ./adapters/llm/ollama/...` without errors
- `golangci-lint run ./adapters/llm/ollama/...` without errors
- `Complete` ≤20 lines (ADR-003)

**Associated test:** all tests from `adapters/llm/ollama/client_test.go`

---

### TODO #16 — infra: Update make test-llm and docs/index.md with Ollama

**Agente:** @devops

**Description:** Verify that `make test-llm` (created in TODO #10) already covers
`./adapters/llm/...` by globbing and includes the Ollama tests. Update
`docs/index.md` by adding the Ollama adapter row in the adapters table.

**Affected files:**
- `Makefile` (verify/adjust if the pattern does not cover `ollama/`)
- `docs/index.md`

**Row to add in docs/index.md adapters table:**
```
| LLM Ollama (Mixtral) | libs/ports/llm.go | adapters/llm/ollama/ | planned | SPRINT-008 |
```

**Acceptance criteria:**
- `make test-llm` runs 14 tests in total (8 Anthropic+Fake + 6 Ollama), all pass
- `grep "ollama" docs/index.md` returns the adapter row
- `make ci` includes the Ollama tests (without network or real Ollama instance)

**Associated test:** all tests in `adapters/llm/`

---

## Traceability Matrix

| TODO | Type   | ADR           | Spec                               | Test                                                                      | Impl                                    |
|------|--------|---------------|------------------------------------|---------------------------------------------------------------------------|-----------------------------------------|
| #1   | data   | 001, 004      | —                                  | covered by adapter tests (#3)                                             | libs/domain/errors.go                   |
| #2   | impl   | 001, 003, 016 | specs/patterns/nodes/llm_call.json | —                                                                         | libs/ports/llm.go                       |
| #3   | test   | 002, 003      | libs/ports/llm.go                  | client_test.go (7 tests, Red)                                             | —                                       |
| #4   | test   | 002, 003      | libs/ports/llm.go                  | fake/client_test.go (1 test, Red)                                         | —                                       |
| #5   | impl   | 001, 003      | libs/ports/llm.go                  | TestFakeLLMClientQueue                                                    | adapters/llm/fake/client.go             |
| #6   | impl   | 001, 003, 004 | libs/ports/llm.go                  | TestCompleteText, TestCompleteWithTools, TestCompleteToolResult            | adapters/llm/anthropic/convert.go       |
| #7   | impl   | 001, 003, 004 | libs/domain/errors.go              | TestCompleteRateLimit, TestCompleteServerError, TestCompleteUnauthorized, TestCompleteContextTimeout | adapters/llm/anthropic/errors.go |
| #8   | impl   | 001, 003, 004 | libs/ports/llm.go                  | all tests from client_test.go (Green)                                     | adapters/llm/anthropic/client.go        |
| #9   | infra  | 004, 013      | —                                  | enables compilation and tests                                             | go.mod, go.sum, .env.example            |
| #10  | infra  | 002           | —                                  | enables CI execution                                                      | Makefile                                |
| #11  | docs   | 020           | —                                  | —                                                                         | docs/index.md, docs/log.md              |
| #12  | infra  | 004, 013      | —                                  | enables compilation and Ollama tests                                      | go.mod, go.sum, .env.example            |
| #13  | test   | 002, 003      | libs/ports/llm.go                  | ollama/client_test.go (6 tests, Red)                                      | —                                       |
| #14  | impl   | 001, 003, 004 | libs/ports/llm.go                  | TestOllamaCompleteText, TestOllamaCompleteWithTools, TestOllamaConvertFinishReason | adapters/llm/ollama/convert.go  |
| #15  | impl   | 001, 003, 004 | libs/domain/errors.go, libs/ports/llm.go | all tests from ollama/client_test.go (Green)                        | adapters/llm/ollama/client.go, ollama/errors.go |
| #16  | infra+docs | 002, 020  | —                                  | make test-llm runs 14 tests                                               | Makefile, docs/index.md                 |

## Implementation Notes

**Anthropic Go SDK:** `github.com/anthropics/anthropic-sdk-go`. Client with
custom `BaseURL` via `option.WithBaseURL(cfg.BaseURL)` — essential
so that `httptest.NewServer` works in tests.

**Mock server in tests:** Each test in `client_test.go` creates its own
`httptest.NewServer` with an ad-hoc handler that returns Anthropic API JSON
(`/v1/messages`). The client is built with `Config{APIKey: "test-key", BaseURL: ts.URL}`.

**Minimal mock response JSON structure:**
```json
{
  "id": "msg_01",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "hello"}],
  "model": "claude-sonnet-4-6-20251001",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}
```

**Ollama SDK:** `github.com/sashabaranov/go-openai` with `openai.ClientConfig{BaseURL: cfg.BaseURL + "/v1"}`.
Ollama does not require an API key — `""` is passed. The `go-openai` SDK is reusable for the
native OpenAI adapter in future sprints (just change BaseURL and add the key).

**Mock server for Ollama:** each test creates an `httptest.NewServer` that serves
`/v1/chat/completions`. Minimal mock response structure:
```json
{
  "id": "chatcmpl-01",
  "object": "chat.completion",
  "choices": [{"message": {"role": "assistant", "content": "hello"}, "finish_reason": "stop"}],
  "usage": {"prompt_tokens": 10, "completion_tokens": 5}
}
```

**Compile-time interface verification** (include in each file):
```go
var _ ports.LLMClient = &AnthropicClient{}
var _ ports.LLMClient = &FakeLLMClient{}
var _ ports.LLMClient = &OllamaClient{}
```

**TODO order by dependencies:**
```
#1 (domain errors) ─┐
#2 (port)           ├─ parallelizable
#9 (infra Anthropic)┤
#12 (infra Ollama)  ┘
        ↓
#3 (tests Red Anthropic) + #4 (test Red Fake) + #13 (tests Red Ollama)  ← parallel block
        ↓
#5 (impl Fake) + #6 (convert Anthropic) + #7 (errors Anthropic) + #14 (convert Ollama)
        ↓
#8 (client Anthropic) + #15 (client+errors Ollama)
        ↓
#10+#16 (Makefile) + #11 (docs)
```

## Result (complete on close)

- [x] `TestCompleteText` passes
- [x] `TestCompleteWithTools` passes
- [x] `TestCompleteToolResult` passes
- [x] `TestCompleteRateLimit` passes — `errors.Is(err, domain.ErrRateLimited)` true
- [x] `TestCompleteServerError` passes — `errors.Is(err, domain.ErrProviderUnavailable)` true
- [x] `TestCompleteUnauthorized` passes — `errors.Is(err, domain.ErrUnauthorized)` true
- [x] `TestCompleteContextTimeout` passes — `context.DeadlineExceeded` propagated
- [x] `TestFakeLLMClientQueue` passes
- [x] `var _ ports.LLMClient = &AnthropicClient{}` compiles
- [x] `var _ ports.LLMClient = &FakeLLMClient{}` compiles
- [x] `go build ./libs/... ./adapters/llm/...` without errors
- [x] `golangci-lint run ./libs/... ./adapters/llm/...` without errors
- [x] `make test-llm` runs 14 tests (7 Anthropic + 1 Fake + 6 Ollama), all pass, without real network
- [x] `.env.example` updated with Anthropic and Ollama variables
- [x] `docs/index.md` and `docs/log.md` updated
- [x] `TestOllamaCompleteText` passes
- [x] `TestOllamaCompleteWithTools` passes
- [x] `TestOllamaCompleteServerError` passes — `errors.Is(err, domain.ErrProviderUnavailable)` true
- [x] `TestOllamaCompleteContextTimeout` passes — `context.DeadlineExceeded` propagated
- [x] `TestOllamaCompleteModelDefault` passes — body contains `"model":"mixtral"`
- [x] `TestOllamaConvertFinishReason` passes — "stop"→"end_turn", "tool_calls"→"tool_use", "length"→"max_tokens"
- [x] `var _ ports.LLMClient = &OllamaClient{}` compiles
- [x] `go build ./adapters/llm/ollama/...` without errors
- [x] `golangci-lint run ./adapters/llm/ollama/...` without errors

**PR:** [#9](https://github.com/aescanero/dago/pull/9) — merged 2026-05-09
