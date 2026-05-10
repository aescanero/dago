# SPRINT-012: LLM adapters — OpenAI, Azure OpenAI, Mistral

## Metadata

- **Start date:** 2026-05-10
- **Estimated end date:** 2026-05-13
- **Status:** planned
- **Applied ADRs:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-005, ADR-013, ADR-020
- **Affected specs:** — (port interface unchanged)
- **Planning agent:** planner
- **Reviewed by:** pending
- **Blocked by:** SPRINT-008 (LLMClient port + go-openai in go.mod), SPRINT-009 (executor wiring pattern)
- **Blocks:** any sprint needing OpenAI/Azure/Mistral as LLM backend (tool_use, react, reflection)

## Objective

Implement three new `LLMClient` adapters — native OpenAI, Azure OpenAI, and Mistral — each
following the four-file structure established in SPRINT-008
(`client.go`, `convert.go`, `errors.go`, `client_test.go`).
All three reuse `github.com/sashabaranov/go-openai` already in `go.mod`.
At completion the executor selects any of five providers
(`anthropic`, `ollama`, `openai`, `azureopenai`, `mistral`)
via `EXECUTOR_LLM_PROVIDER` without touching the domain or handler layer.

## Scope

**Included:**
- `adapters/llm/openai/` — `OpenAIClient` (APIKey, BaseURL, Model)
- `adapters/llm/azureopenai/` — `AzureOpenAIClient` (APIKey, Endpoint, Deployment, APIVersion)
- `adapters/llm/mistral/` — `MistralClient` (APIKey, BaseURL, Model; OpenAI-compatible API)
- `services/executor/cmd/main.go` — extend `buildLLMClient` with 3 new cases
- `.env.example` — 9 new variables (3 per provider)
- `Makefile` — verify `make test-llm` covers `./adapters/llm/...`
- `docs/index.md` + `docs/log.md` — updated at close

**Not included:** token streaming, Google Vertex / Gemini, Bedrock, integration tests against real APIs, per-provider retry with own backoff, LLM response cache.

## Dependencies

- **Blocked by:** SPRINT-008, SPRINT-009
- **Parallel with:** any sprint not touching `adapters/llm/` or `services/executor/cmd/main.go`
- **Blocks:** executor patterns `tool_use`, `react`, `reflection` when those need OpenAI/Azure/Mistral

## TODOs

### TODO #1 — test: Red tests — OpenAI adapter [test]

**Agent:** @qa

**Objective:** Write 7 unit tests for `adapters/llm/openai/` BEFORE implementation (TDD Red).
All use `httptest.NewServer` to mock `/v1/chat/completions`. Tests fail at compilation before
TODO #4–#5 exist.

Tests: `TestOpenAICompleteText`, `TestOpenAICompleteWithTools`, `TestOpenAICompleteToolResult`,
`TestOpenAICompleteRateLimit` (429 → `ErrRateLimited`), `TestOpenAICompleteServerError`
(500 → `ErrProviderUnavailable`), `TestOpenAICompleteUnauthorized` (401 → `ErrUnauthorized`),
`TestOpenAICompleteContextTimeout` (10ms deadline, mock sleeps 200ms → `context.DeadlineExceeded`).

**Files:** `adapters/llm/openai/client_test.go` (new)

---

### TODO #2 — test: Red tests — Azure OpenAI adapter [test]

**Agent:** @qa

**Objective:** Write 6 unit tests for `adapters/llm/azureopenai/` BEFORE implementation.
Key difference from OpenAI: client sends `api-key` header; deployment name appears as `model`
in request body. `TestAzureOpenAICompleteText` verifies the `model` field equals the configured
deployment.

Tests: `TestAzureOpenAICompleteText`, `TestAzureOpenAICompleteWithTools`,
`TestAzureOpenAICompleteRateLimit`, `TestAzureOpenAICompleteServerError`,
`TestAzureOpenAICompleteUnauthorized`, `TestAzureOpenAICompleteContextTimeout`.

**Files:** `adapters/llm/azureopenai/client_test.go` (new)

---

### TODO #3 — test: Red tests — Mistral adapter [test]

**Agent:** @qa

**Objective:** Write 7 unit tests for `adapters/llm/mistral/` BEFORE implementation.
Mistral is OpenAI-compatible so mock JSON format is identical.

Tests: `TestMistralCompleteText`, `TestMistralCompleteWithTools`, `TestMistralCompleteToolResult`,
`TestMistralCompleteRateLimit`, `TestMistralCompleteServerError`, `TestMistralCompleteUnauthorized`,
`TestMistralCompleteContextTimeout`.

**Files:** `adapters/llm/mistral/client_test.go` (new)

---

### TODO #4 — impl: OpenAI adapter — convert.go + errors.go [impl]

**Agent:** @developer

**Objective:** Implement conversion and error-mapping for `adapters/llm/openai/`.
`convert.go` is structurally identical to `adapters/llm/ollama/convert.go` (same go-openai types).
`errors.go` maps: 401 → `ErrUnauthorized`, 429 → `ErrRateLimited`, 500/503 → `ErrProviderUnavailable`,
context errors propagated unchanged, others wrapped with `fmt.Errorf`.

Functions: `toOpenAIMessages`, `toOpenAITools`, `fromOpenAIResponse`, `convertFinishReason`
("stop"→"end_turn", "tool_calls"→"tool_use", "length"→"max_tokens").

**Files:** `adapters/llm/openai/convert.go`, `adapters/llm/openai/errors.go` (new)

**Depends on:** #1

---

### TODO #5 — impl: OpenAI adapter — client.go [impl]

**Agent:** @developer

**Objective:** Implement `OpenAIClient` satisfying `ports.LLMClient`. Uses `goai.DefaultConfig(apiKey)`
with optional `BaseURL` override. Default model `gpt-4o` when `LLMRequest.Model == ""`.
Returns error from constructor if `APIKey` is empty. `var _ ports.LLMClient = &OpenAIClient{}`
must compile.

```go
type Config struct {
    APIKey  string // OPENAI_API_KEY (required)
    BaseURL string // OPENAI_BASE_URL (optional; default https://api.openai.com/v1)
    Model   string // OPENAI_MODEL   (optional; default gpt-4o)
}
```

**Files:** `adapters/llm/openai/client.go` (new)

**Depends on:** #1, #4

---

### TODO #6 — impl: Azure OpenAI adapter — convert.go + errors.go [impl]

**Agent:** @developer

**Objective:** Same as TODO #4 but in `package azureopenai`. Do NOT import from
`adapters/llm/openai/` — each adapter is independently replaceable (ADR-001).

**Files:** `adapters/llm/azureopenai/convert.go`, `adapters/llm/azureopenai/errors.go` (new)

**Depends on:** #2

---

### TODO #7 — impl: Azure OpenAI adapter — client.go [impl]

**Agent:** @developer

**Objective:** Implement `AzureOpenAIClient`. Uses `goai.DefaultAzureConfig(apiKey, endpoint)`.
`Deployment` always overrides `req.Model` (Azure routes by deployment name, not model name).
All four config fields required; return error from constructor if any is empty.

```go
type Config struct {
    APIKey     string // AZURE_OPENAI_API_KEY        (required)
    Endpoint   string // AZURE_OPENAI_ENDPOINT        (required)
    Deployment string // AZURE_OPENAI_DEPLOYMENT      (required)
    APIVersion string // AZURE_OPENAI_API_VERSION     (required)
}
```

**Files:** `adapters/llm/azureopenai/client.go` (new)

**Depends on:** #2, #6

---

### TODO #8 — impl: Mistral adapter — convert.go + errors.go [impl]

**Agent:** @developer

**Objective:** Same as TODO #4 but in `package mistral`. Mistral is OpenAI-compatible so
conversion is identical. Do NOT import from other llm packages.

**Files:** `adapters/llm/mistral/convert.go`, `adapters/llm/mistral/errors.go` (new)

**Depends on:** #3

---

### TODO #9 — impl: Mistral adapter — client.go [impl]

**Agent:** @developer

**Objective:** Implement `MistralClient`. Uses `goai.DefaultConfig(apiKey)` with `BaseURL`
override pointing to Mistral's API. Default model `mistral-large-latest`.

```go
type Config struct {
    APIKey  string // MISTRAL_API_KEY  (required)
    BaseURL string // MISTRAL_BASE_URL (optional; default https://api.mistral.ai/v1)
    Model   string // MISTRAL_MODEL    (optional; default mistral-large-latest)
}
```

**Files:** `adapters/llm/mistral/client.go` (new)

**Depends on:** #3, #8

---

### TODO #10 — impl: wire new providers in executor [impl]

**Agent:** @developer

**Objective:** Extend `buildLLMClient` in `services/executor/cmd/main.go` with three new cases.
Each reads its env vars and calls the adapter constructor; returns fatal error if required vars
are missing.

```go
case "openai":
    apiKey := os.Getenv("OPENAI_API_KEY")
    // ...
    return openaillm.NewOpenAIClient(openaillm.Config{...})
case "azureopenai":
    // AZURE_OPENAI_API_KEY, AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_DEPLOYMENT, AZURE_OPENAI_API_VERSION
case "mistral":
    // MISTRAL_API_KEY, MISTRAL_BASE_URL, MISTRAL_MODEL
```

Import aliases: `openaillm`, `azureopenaillm`, `mistrallm` to avoid package name collisions.

**Files:** `services/executor/cmd/main.go`

**Depends on:** #5, #7, #9

---

### TODO #11 — impl: .env.example + Makefile [impl]

**Agent:** @devops

**Objective:** Append 9 new variables to `.env.example` grouped by provider.
Verify `make test-llm` target already covers `./adapters/llm/...` (it does via glob) —
no Makefile change needed unless the glob is narrower.

New variables:
```dotenv
# --- OpenAI ---
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o

# --- Azure OpenAI ---
AZURE_OPENAI_API_KEY=...
AZURE_OPENAI_ENDPOINT=https://my-resource.openai.azure.com
AZURE_OPENAI_DEPLOYMENT=gpt-4o-deployment
AZURE_OPENAI_API_VERSION=2024-08-01-preview

# --- Mistral ---
MISTRAL_API_KEY=...
MISTRAL_BASE_URL=https://api.mistral.ai/v1
MISTRAL_MODEL=mistral-large-latest
```

**Files:** `.env.example`

**Depends on:** #10

---

### TODO #12 — docs: update index.md and log.md [docs]

**Agent:** @docs

**Objective:** Update project-wide documentation on sprint close.

- `docs/index.md` — add SPRINT-012 row (completed) + 3 new adapter rows in the Adapters table
- `docs/log.md` — append closing entry with date 2026-05-13 and artifacts delivered

**Files:** `docs/index.md`, `docs/log.md`

---

## Traceability Matrix

| TODO | Spec | Test | Impl | Docs |
|------|------|------|------|------|
| #1 OpenAI Red tests | — | `adapters/llm/openai/client_test.go` | — | — |
| #2 Azure OpenAI Red tests | — | `adapters/llm/azureopenai/client_test.go` | — | — |
| #3 Mistral Red tests | — | `adapters/llm/mistral/client_test.go` | — | — |
| #4 OpenAI convert + errors | ADR-001 | ← #1 | `openai/convert.go`, `openai/errors.go` | — |
| #5 OpenAI client | ADR-001, ADR-002 | ← #1 | `openai/client.go` | — |
| #6 Azure convert + errors | ADR-001 | ← #2 | `azureopenai/convert.go`, `azureopenai/errors.go` | — |
| #7 Azure client | ADR-001, ADR-002 | ← #2 | `azureopenai/client.go` | — |
| #8 Mistral convert + errors | ADR-001 | ← #3 | `mistral/convert.go`, `mistral/errors.go` | — |
| #9 Mistral client | ADR-001, ADR-002 | ← #3 | `mistral/client.go` | — |
| #10 Executor wiring | ADR-009 | — | `services/executor/cmd/main.go` | — |
| #11 .env.example | — | — | `.env.example` | — |
| #12 Docs | — | — | — | `docs/index.md`, `docs/log.md` |

## Key decisions

- **Reuse go-openai SDK for all three** — `go-openai` already supports OpenAI, Azure OpenAI
  (via `DefaultAzureConfig`), and any OpenAI-compatible endpoint (Mistral, Ollama). Adding
  three adapters has zero new dependencies.
- **No shared convert.go across adapters** — ADR-001 requires each adapter to be independently
  replaceable. Duplicating ~40 lines of conversion is preferable to cross-adapter imports.
- **Azure deployment overrides req.Model** — Azure routes by deployment name, not model name.
  The adapter always substitutes `cfg.Deployment` regardless of `LLMRequest.Model`.
- **Mistral default model `mistral-large-latest`** — most capable model available on Mistral's
  API at sprint time; configurable via `MISTRAL_MODEL`.
- **No integration tests against real APIs** — unit tests with `httptest.NewServer` are
  sufficient and run without credentials in CI. Real-API smoke tests are an optional future sprint.
- **Import aliases to avoid collision** — `openai` conflicts with the go-openai package alias
  used inside the adapter. Use `openaillm`, `azureopenaillm`, `mistrallm` as import aliases
  in `executor/cmd/main.go`.

## Result

> _Complete on sprint close._

- TODOs completed: —/12
- Tests passing: —
- Decisions reviewed: —
- Artifacts delivered: —
