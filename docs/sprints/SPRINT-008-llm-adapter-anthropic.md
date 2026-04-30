# SPRINT-008: Adaptador LLM — Puerto LLMClient, Anthropic y Ollama/Mixtral

## Metadata

- **Fecha inicio:** 2026-04-29
- **Fecha fin estimada:** 2026-04-30
- **Estado:** planificado
- **ADRs aplicados:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-013, ADR-015, ADR-016
- **Specs afectadas:** specs/patterns/nodes/llm_call.json (referencia; ninguna spec nueva creada en este sprint)
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Bloqueado por:** SPRINT-001 (go.mod)
- **Bloquea:** SPRINT-009 (executor usa LLMClient), SPRINT-010 (orchestrator state machine)

## Objetivo

Implementar el puerto `LLMClient` en `libs/ports/llm.go` y dos adaptadores: Anthropic
(`adapters/llm/anthropic/`) y Ollama con API OpenAI-compatible (`adapters/llm/ollama/`,
modelo default `mixtral`). Incluye un `FakeLLMClient` determinista para tests. Al
finalizar, el executor puede invocar Anthropic claude-sonnet o Mixtral via Ollama a
través de la misma interfaz `LLMClient` sin acoplar el dominio a ningún SDK concreto.

## Alcance

**Entra:**
- Tres errores de dominio nuevos en `libs/domain/errors.go`:
  `ErrUnauthorized`, `ErrRateLimited`, `ErrProviderUnavailable`
- Puerto `LLMClient` con tipos `Message`, `ToolDefinition`, `LLMRequest`,
  `LLMResponse`, `ToolUse` en `libs/ports/llm.go`
- Adaptador Anthropic en `adapters/llm/anthropic/`:
  - `client.go` — `AnthropicClient` que implementa `LLMClient`
  - `convert.go` — conversión `ports` ↔ Anthropic SDK types
  - `errors.go` — mapeo HTTP status → errores de dominio
  - `client_test.go` — 7 tests unitarios con `httptest.NewServer`
- Fake determinista en `adapters/llm/fake/`:
  - `client.go` — `FakeLLMClient` con cola de respuestas y registro de llamadas
  - `client_test.go` — 1 test `TestFakeLLMClientQueue`
- Dependencias `github.com/anthropics/anthropic-sdk-go` y `github.com/sashabaranov/go-openai` en `go.mod`
- Adaptador Ollama (API OpenAI-compatible) en `adapters/llm/ollama/`:
  - `client.go` — `OllamaClient` que implementa `LLMClient`; `NewOllamaClient` no retorna error (BaseURL tiene default `http://localhost:11434`)
  - `convert.go` — conversión `ports` ↔ `go-openai` types; `convertFinishReason` mapea "stop"→"end_turn", "tool_calls"→"tool_use", "length"→"max_tokens"
  - `errors.go` — HTTP 500 → `ErrProviderUnavailable`
  - `client_test.go` — 6 tests unitarios con `httptest.NewServer`
- Modelo default Ollama: `mixtral` (Mixtral 8x7B)
- Variables de entorno documentadas en `.env.example`
- Target `make test-llm` en `Makefile` (14 tests en total)
- Actualización de `docs/index.md` y `docs/log.md`

**No entra:**
- Otros proveedores LLM (OpenAI nativo, Google Vertex) — sprints futuros
- Streaming de tokens (chunked responses) — requiere ADR separado
- Integración con el executor — SPRINT-010 o posterior
- Caché de respuestas LLM — sprint futuro
- Tests de integración contra la API real de Anthropic — excluidos de `make ci`
- Retry automático con backoff propio — se delega al SDK (`ANTHROPIC_MAX_RETRIES`)

## Dependencias

- **Bloqueado por:** SPRINT-001 (go.mod válido con estructura de monorepo compilable)
- **Paralelo a:** SPRINT-002, SPRINT-003, SPRINT-004, SPRINT-007 (sin dependencia entre ellos)
- **Bloquea:** executor (nodos `llm_call`, `react`, `reflection` usan `LLMClient`),
  router LLM (usa `LLMClient` para routing basado en LLM)

## Contratos de comportamiento

### C1 — `AnthropicClient.Complete` — respuesta de texto

```
Given: LLMRequest con messages=[{role:"user",content:"hello"}], model="claude-sonnet-4-6", max_tokens=100
When: AnthropicClient.Complete(ctx, req) — mock HTTP devuelve 200 con content text
Then: LLMResponse.Content no está vacío
      LLMResponse.StopReason == "end_turn"
      LLMResponse.InputTokens > 0 y OutputTokens > 0
      No se retorna error
```

### C2 — `AnthropicClient.Complete` — rate limit

```
Given: El servidor HTTP mock devuelve HTTP 429
When: AnthropicClient.Complete(ctx, req)
Then: Se retorna error tal que errors.Is(err, domain.ErrRateLimited) == true
      El caller puede reintentar con backoff exponencial
      LLMResponse devuelta es vacía
```

### C3 — `FakeLLMClient.Complete` — cola FIFO y registro de llamadas

```
Given: FakeLLMClient{Responses: [resp1, resp2]}
When: Se llama Complete() tres veces consecutivas
Then: Primera llamada retorna resp1, segunda retorna resp2
      Tercera llamada (cola vacía) retorna {Content:"fake response", StopReason:"end_turn"}
      Calls contiene exactamente 3 LLMRequests en orden de inserción
      Complete nunca retorna error
```

## TODOs

### TODO #1 — data: Añadir errores de dominio LLM a libs/domain/errors.go

**Agente:** @developer

**Descripción:** Añadir tres errores centinela al fichero de errores de dominio.
Tipos puros sin dependencia de infraestructura. El adaptador Anthropic los retornará
envueltos con `fmt.Errorf`. El executor usará `errors.Is` para diferenciar la acción
de recuperación.

**Archivos afectados:**
- `libs/domain/errors.go` (nuevo o existente — añadir si ya existe)

**Tipos a implementar:**
```go
package domain

import "errors"

var (
    // ErrUnauthorized indica credenciales del proveedor LLM inválidas (HTTP 401).
    ErrUnauthorized = errors.New("unauthorized")

    // ErrRateLimited indica que el proveedor LLM rechazó la petición por límite de
    // velocidad (HTTP 429). El caller puede reintentar con backoff exponencial.
    ErrRateLimited = errors.New("rate limited")

    // ErrProviderUnavailable indica que el proveedor LLM no está disponible
    // temporalmente (HTTP 500/529). Retryable.
    ErrProviderUnavailable = errors.New("provider unavailable")
)
```

**Criterios de aceptación:**
- `go build ./libs/domain/...` sin errores
- Sin imports de infraestructura
- Los tres errores son distinguibles con `errors.Is`

**Test asociado:** cubiertos indirectamente por tests del adaptador en TODO #3

---

### TODO #2 — impl: Definir interfaz LLMClient en libs/ports/llm.go

**Agente:** @developer

**Descripción:** Definir el puerto de salida para llamadas a LLMs. El puerto es la
frontera entre el dominio y la infraestructura. `LLMRequest` y `LLMResponse` son tipos
portátiles — no contienen tipos del SDK de Anthropic. `ToolDefinition.InputSchema` es
`json.RawMessage` para transportar un JSON Schema arbitrario sin deserializarlo.

**Archivos afectados:**
- `libs/ports/llm.go` (nuevo)

**Interfaz a implementar:**
```go
package ports

import (
    "context"
    "encoding/json"
)

// Message representa un turno de conversación.
type Message struct {
    Role      string // "user" | "assistant" | "tool_result"
    Content   string
    ToolUseID string // solo para Role == "tool_result"
}

// ToolDefinition describe una herramienta disponible para el LLM.
type ToolDefinition struct {
    Name        string
    Description string
    InputSchema json.RawMessage // JSON Schema del input de la herramienta
}

// LLMRequest encapsula una llamada al LLM.
type LLMRequest struct {
    Model       string
    System      string
    Messages    []Message
    Tools       []ToolDefinition
    MaxTokens   int
    Temperature float64
}

// ToolUse representa una invocación de herramienta solicitada por el LLM.
type ToolUse struct {
    ID    string
    Name  string
    Input json.RawMessage
}

// LLMResponse encapsula la respuesta del LLM.
type LLMResponse struct {
    Content      string    // texto generado; vacío si StopReason == "tool_use"
    StopReason   string    // "end_turn" | "tool_use" | "max_tokens"
    ToolUses     []ToolUse // poblado si StopReason == "tool_use"
    InputTokens  int
    OutputTokens int
}

// LLMClient es el puerto de salida para llamadas a modelos de lenguaje.
type LLMClient interface {
    Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
```

**Criterios de aceptación:**
- `go build ./libs/ports/...` sin errores
- Solo importa `context`, `encoding/json` y nada de infraestructura
- `LLMClient` tiene un único método (ADR-003: responsabilidad única)

**Test asociado:** —

---

### TODO #3 — test: Tests Red del adaptador Anthropic con httptest mock server

**Agente:** @qa

**Descripción:** Escribir los 7 tests del adaptador Anthropic ANTES de implementarlo
(ciclo Red de TDD). Usan `net/http/httptest.NewServer` como mock del endpoint Anthropic.
`ANTHROPIC_BASE_URL` apunta al servidor mock. Al ejecutarse contra stubs vacíos deben
fallar con error de compilación o "not implemented".

**Archivos afectados:**
- `adapters/llm/anthropic/client_test.go` (nuevo)

**Tests a implementar:**
```go
package anthropic_test

// TestCompleteText
// Escenario: LLMRequest sin tools, mock devuelve 200 con content text.
// Verifica: LLMResponse.Content == texto esperado, StopReason == "end_turn",
//           InputTokens > 0, OutputTokens > 0.

// TestCompleteWithTools
// Escenario: LLMRequest con ToolDefinition, mock devuelve 200 con
//            stop_reason "tool_use" y un bloque tool_use.
// Verifica: StopReason == "tool_use", len(ToolUses) == 1,
//           ToolUses[0].Name == nombre esperado, Content == "".

// TestCompleteToolResult
// Escenario: conversación de 3 turnos (user → assistant tool_use →
//            tool_result → assistant end_turn).
// Verifica: StopReason == "end_turn", Content != "".

// TestCompleteRateLimit
// Escenario: mock devuelve HTTP 429.
// Verifica: errors.Is(err, domain.ErrRateLimited).

// TestCompleteServerError
// Escenario: mock devuelve HTTP 500.
// Verifica: errors.Is(err, domain.ErrProviderUnavailable).

// TestCompleteUnauthorized
// Escenario: mock devuelve HTTP 401.
// Verifica: errors.Is(err, domain.ErrUnauthorized).

// TestCompleteContextTimeout
// Escenario: ctx con deadline 10ms, mock duerme 100ms.
// Verifica: error envuelve context.DeadlineExceeded.
```

**Criterios de aceptación:**
- `go test ./adapters/llm/anthropic/...` falla con compilación o "not implemented" (Red)
- Sin frameworks de mock — solo `httptest.NewServer` y fakes manuales
- Cada test crea su propio `httptest.NewServer`, independientes entre sí

**Test asociado:** este TODO ES el test (fase Red)

---

### TODO #4 — test: Test Red del FakeLLMClient

**Agente:** @qa

**Descripción:** Escribir el test del fake ANTES de implementarlo. Verifica la
cola de respuestas FIFO y el registro de llamadas.

**Archivos afectados:**
- `adapters/llm/fake/client_test.go` (nuevo)

**Test a implementar:**
```go
package fake_test

// TestFakeLLMClientQueue
// Escenario: FakeLLMClient con Responses = [resp1, resp2].
// Verifica:
//   - Primera llamada Complete → resp1
//   - Segunda llamada → resp2
//   - Tercera llamada (cola vacía) → LLMResponse{Content: "fake response", StopReason: "end_turn"}
//   - Calls contiene 3 LLMRequests en orden
//   - No retorna error en ningún caso
```

**Criterios de aceptación:**
- `go test ./adapters/llm/fake/...` falla con compilación (Red confirmado)
- Verifica tanto valores retornados como el registro `Calls`

**Test asociado:** este TODO ES el test (fase Red)

---

### TODO #5 — impl: FakeLLMClient en adapters/llm/fake/client.go

**Agente:** @developer

**Descripción:** Implementar el fake determinista para uso en tests del executor.
Sin dependencias externas — solo stdlib. Implementa `ports.LLMClient`.

**Archivos afectados:**
- `adapters/llm/fake/client.go` (nuevo)

**Diseño:**
```go
package fake

import (
    "context"
    "github.com/aescanero/dago/libs/ports"
)

// FakeLLMClient implementa ports.LLMClient para tests.
// Devuelve respuestas de Responses en orden FIFO.
// Cuando la cola se agota, devuelve la respuesta por defecto.
type FakeLLMClient struct {
    Responses []ports.LLMResponse
    Calls     []ports.LLMRequest
}

// Complete retorna la siguiente respuesta o la por defecto.
// Registra siempre la llamada en Calls. Nunca retorna error.
func (f *FakeLLMClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error)
```

**Respuesta por defecto:** `ports.LLMResponse{Content: "fake response", StopReason: "end_turn"}`

**Verificación de interface:** `var _ ports.LLMClient = &FakeLLMClient{}`

**Criterios de aceptación:**
- `TestFakeLLMClientQueue` pasa (Green)
- `go build ./adapters/llm/fake/...` sin errores
- Sin imports de SDKs externos

**Test asociado:** `TestFakeLLMClientQueue`

---

### TODO #6 — impl: Conversión de tipos en adapters/llm/anthropic/convert.go

**Agente:** @developer

**Descripción:** Funciones puras de traducción entre tipos del puerto y tipos del SDK
de Anthropic. Sin estado, sin IO. Separar la conversión del cliente permite testearla
de forma unitaria.

**Archivos afectados:**
- `adapters/llm/anthropic/convert.go` (nuevo)

**Funciones a implementar:**
```go
package anthropic

// toAnthropicMessages convierte []ports.Message → []anthropic.MessageParam.
// Maneja roles: "user", "assistant", "tool_result".
// Para "tool_result": construye ToolResultBlockParam con ToolUseID.
func toAnthropicMessages(messages []ports.Message) []anthropic.MessageParam

// toAnthropicTools convierte []ports.ToolDefinition → []anthropic.ToolParam.
// InputSchema se pasa directamente como json.RawMessage al SDK.
func toAnthropicTools(tools []ports.ToolDefinition) []anthropic.ToolParam

// fromAnthropicResponse convierte anthropic.Message → ports.LLMResponse.
// Extrae Content del primer bloque tipo text.
// Extrae ToolUses de los bloques tipo tool_use.
// Mapea stop_reason: "end_turn" | "tool_use" | "max_tokens".
func fromAnthropicResponse(msg anthropic.Message) ports.LLMResponse
```

**Criterios de aceptación:**
- `go build ./adapters/llm/anthropic/...` sin errores
- Cada función ≤20 líneas (ADR-003)
- `TestCompleteText`, `TestCompleteWithTools` y `TestCompleteToolResult` pasan

**Test asociado:** `TestCompleteText`, `TestCompleteWithTools`, `TestCompleteToolResult`

---

### TODO #7 — impl: Mapeo de errores en adapters/llm/anthropic/errors.go

**Agente:** @developer

**Descripción:** Función que mapea errores del SDK de Anthropic (por código HTTP) a
errores de dominio. El adaptador nunca expone tipos del SDK al dominio.

**Archivos afectados:**
- `adapters/llm/anthropic/errors.go` (nuevo)

**Función a implementar:**
```go
package anthropic

// mapAnthropicError convierte un error del SDK en un error de dominio.
// Si el error es *anthropicsdk.Error, mapea por StatusCode.
// context.DeadlineExceeded y context.Canceled se propagan sin envolver.
func mapAnthropicError(op string, err error) error
```

**Mapeo:**
- 401 → `fmt.Errorf("%s: %w", op, domain.ErrUnauthorized)`
- 429 → `fmt.Errorf("%s: %w", op, domain.ErrRateLimited)`
- 500, 529 → `fmt.Errorf("%s: %w", op, domain.ErrProviderUnavailable)`
- `context.DeadlineExceeded` / `context.Canceled` → propagar sin envolver
- otros → `fmt.Errorf("%s: %w", op, err)`

**Criterios de aceptación:**
- `TestCompleteRateLimit`, `TestCompleteServerError`, `TestCompleteUnauthorized` pasan
- `errors.Is(result, domain.ErrRateLimited)` true para error 429
- `context.DeadlineExceeded` no se envuelve en error de dominio

**Test asociado:** `TestCompleteRateLimit`, `TestCompleteServerError`,
`TestCompleteUnauthorized`, `TestCompleteContextTimeout`

---

### TODO #8 — impl: AnthropicClient en adapters/llm/anthropic/client.go

**Agente:** @developer

**Descripción:** Implementar `AnthropicClient` que implementa `ports.LLMClient`.
Configuración por variables de entorno. Delega serialización al SDK y traducción
de tipos a `convert.go`. `Complete` no supera 20 líneas.

**Archivos afectados:**
- `adapters/llm/anthropic/client.go` (nuevo)

**Diseño:**
```go
package anthropic

// Config contiene la configuración del cliente Anthropic.
type Config struct {
    APIKey     string // ANTHROPIC_API_KEY (requerido)
    BaseURL    string // ANTHROPIC_BASE_URL (opcional)
    MaxRetries int    // ANTHROPIC_MAX_RETRIES (default 2)
}

// AnthropicClient implementa ports.LLMClient usando el SDK oficial de Anthropic.
type AnthropicClient struct {
    client       *anthropicsdk.Client
    defaultModel string
}

// NewAnthropicClient construye un AnthropicClient. Retorna error si APIKey está vacía.
func NewAnthropicClient(cfg Config) (*AnthropicClient, error)

// Complete envía la petición al API de Anthropic y retorna la respuesta del dominio.
func (c *AnthropicClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error)
```

**Comportamiento de Complete:**
1. Construir `MessageNewParams` usando funciones de `convert.go`
2. Llamar `c.client.Messages.New(ctx, params)`
3. Si error: `mapAnthropicError("anthropic complete", err)` y retornar
4. Convertir respuesta con `fromAnthropicResponse(msg)`
5. Retornar `LLMResponse, nil`

**Modelo default:** `claude-sonnet-4-6` (usado cuando `LLMRequest.Model == ""`).

**Verificación de interface:** `var _ ports.LLMClient = &AnthropicClient{}`

**Criterios de aceptación:**
- Todos los 7 tests de `client_test.go` pasan (Green)
- `go build ./adapters/llm/anthropic/...` sin errores
- `golangci-lint run ./adapters/llm/anthropic/...` sin errores
- `Complete` ≤20 líneas (ADR-003)

**Test asociado:** todos los tests de `adapters/llm/anthropic/client_test.go`

---

### TODO #9 — infra: Añadir dependencia anthropic-sdk-go y variables de entorno

**Agente:** @devops

**Descripción:** Añadir el SDK oficial de Anthropic al go.mod. Documentar variables
de entorno en `.env.example`. Los tests usan `httptest.NewServer` — no necesitan
credenciales reales.

**Archivos afectados:**
- `go.mod` / `go.sum` (via `go get`)
- `.env.example`

**Comandos:**
```bash
go get github.com/anthropics/anthropic-sdk-go@latest
go mod tidy
```

**Variables a añadir en `.env.example`:**
```
# Anthropic LLM
ANTHROPIC_API_KEY=sk-ant-...          # requerido en producción
ANTHROPIC_BASE_URL=                   # vacío → api.anthropic.com; tests: http://localhost:PORT
ANTHROPIC_MAX_RETRIES=2
```

**Criterios de aceptación:**
- `go build ./adapters/llm/...` sin errores
- `go test ./adapters/llm/...` ejecuta 8 tests sin red ni credenciales reales
- `.env.example` tiene las tres variables con comentarios

**Test asociado:** habilita compilación y ejecución de todos los tests

---

### TODO #10 — infra: Target make test-llm en Makefile

**Agente:** @devops

**Descripción:** Añadir target `make test-llm` para ejecutar tests unitarios del
adaptador LLM. Tests unitarios (no de integración) → incluidos en `make ci`.

**Archivos afectados:**
- `Makefile`

**Targets a añadir:**
```makefile
## test-llm: tests unitarios del adaptador LLM (anthropic + fake)
test-llm:
	go test -count=1 -timeout 30s ./adapters/llm/...
```

Añadir `test-llm` como dependencia del target `test` existente.

**Criterios de aceptación:**
- `make test-llm` ejecuta 8 tests, todos pasan
- `make ci` incluye `make test-llm` (sin credenciales reales)

**Test asociado:** todos los tests de `adapters/llm/`

---

### TODO #11 — docs: Actualizar docs/index.md y docs/log.md

**Agente:** @docs

**Descripción:** Registrar SPRINT-008 en el índice y log. Añadir sección
`## Adaptadores` en `docs/index.md` con Event Bus Valkey (SPRINT-007) y LLM
Anthropic+Fake+Ollama (SPRINT-008).

**Archivos afectados:**
- `docs/index.md`
- `docs/log.md`

**Criterios de aceptación:**
- `grep "SPRINT-008" docs/index.md` retorna al menos dos filas
- `grep "SPRINT-008" docs/log.md` retorna la entrada
- Sección `## Adaptadores` lista los cuatro adaptadores planificados

**Test asociado:** —

---

### TODO #12 — infra: Añadir dependencia go-openai y variables de entorno para Ollama

**Agente:** @devops

**Descripción:** Añadir el cliente OpenAI-compatible `sashabaranov/go-openai` al go.mod.
Este SDK es reutilizable para el adaptador OpenAI nativo en sprints futuros. Documentar
las dos variables de Ollama en `.env.example`. Los tests usan `httptest.NewServer` —
no requieren instancia Ollama real ni ninguna API key.

**Archivos afectados:**
- `go.mod` / `go.sum` (via `go get`)
- `.env.example`

**Comandos:**
```bash
go get github.com/sashabaranov/go-openai@latest
go mod tidy
```

**Variables a añadir en `.env.example`:**
```
# Ollama LLM
OLLAMA_BASE_URL=http://localhost:11434   # default local; tests: http://127.0.0.1:PORT
OLLAMA_DEFAULT_MODEL=mixtral             # modelo Mixtral 8x7B (alternativa: mixtral:8x7b)
```

**Criterios de aceptación:**
- `go build ./adapters/llm/ollama/...` sin errores tras añadir la dependencia
- `go test ./adapters/llm/ollama/...` no requiere red ni credenciales
- `.env.example` tiene las dos variables con comentarios

**Test asociado:** habilita compilación y ejecución de los tests del TODO #13

---

### TODO #13 — test: Tests Red del adaptador Ollama con httptest mock server

**Agente:** @qa

**Descripción:** Escribir los 6 tests del adaptador Ollama ANTES de implementarlo
(ciclo Red de TDD). El servidor mock simula la API OpenAI-compatible de Ollama en
`/v1/chat/completions`. Cada test crea su propio `httptest.NewServer` independiente.

**Archivos afectados:**
- `adapters/llm/ollama/client_test.go` (nuevo)

**Tests a implementar:**
```go
package ollama_test

// TestOllamaCompleteText
// Escenario: LLMRequest sin tools, mock devuelve 200 con choices[0].message.content
//            y finish_reason "stop".
// Verifica: Content == texto esperado, StopReason == "end_turn",
//           InputTokens > 0, OutputTokens > 0.

// TestOllamaCompleteWithTools
// Escenario: LLMRequest con ToolDefinition, mock devuelve 200 con
//            finish_reason "tool_calls" y choices[0].message.tool_calls.
// Verifica: StopReason == "tool_use", len(ToolUses) == 1,
//           ToolUses[0].Name == nombre esperado, Content == "".

// TestOllamaCompleteServerError
// Escenario: mock devuelve HTTP 500.
// Verifica: errors.Is(err, domain.ErrProviderUnavailable) es true.

// TestOllamaCompleteContextTimeout
// Escenario: ctx con deadline 10ms, mock duerme 100ms antes de responder.
// Verifica: error retornado envuelve context.DeadlineExceeded.

// TestOllamaCompleteModelDefault
// Escenario: LLMRequest con Model == "", mock verifica el body recibido.
// Verifica: el campo "model" en el JSON enviado al mock es "mixtral".

// TestOllamaConvertFinishReason
// Escenario: tabla de casos para convertFinishReason.
// Verifica: "stop"→"end_turn", "tool_calls"→"tool_use",
//           "length"→"max_tokens", ""→"end_turn" (fallback).
```

**Criterios de aceptación:**
- `go test ./adapters/llm/ollama/...` falla con compilación o "not implemented" (Red)
- Sin frameworks de mock — solo `httptest.NewServer` y tabla de casos
- Cada test es independiente, no comparte estado

**Test asociado:** este TODO ES el test (fase Red)

---

### TODO #14 — impl: Conversión de tipos en adapters/llm/ollama/convert.go

**Agente:** @developer

**Descripción:** Funciones puras de traducción entre tipos del puerto y tipos del SDK
`go-openai`. Sin estado, sin IO. `convertFinishReason` mapea `finish_reason` de la
API OpenAI-compatible a los valores estándar del dominio.

**Archivos afectados:**
- `adapters/llm/ollama/convert.go` (nuevo)

**Funciones a implementar:**
```go
package ollama

// toOpenAIMessages convierte []ports.Message en []openai.ChatCompletionMessage.
// Roles: "user"→ChatMessageRoleUser, "assistant"→ChatMessageRoleAssistant,
//        "tool_result"→ChatMessageRoleTool (con ToolCallID).
func toOpenAIMessages(messages []ports.Message) []openai.ChatCompletionMessage

// toOpenAITools convierte []ports.ToolDefinition en []openai.Tool.
// Cada herramienta: openai.Tool{Type:"function", Function:{Name, Description, Parameters}}.
// InputSchema se pasa directamente como Parameters (json.RawMessage).
func toOpenAITools(tools []ports.ToolDefinition) []openai.Tool

// fromOpenAIResponse convierte openai.ChatCompletionResponse en ports.LLMResponse.
// Content: choices[0].Message.Content.
// ToolUses: choices[0].Message.ToolCalls → ports.ToolUse{ID, Name, Input}.
// StopReason: via convertFinishReason(choices[0].FinishReason).
// Tokens: Usage.PromptTokens, Usage.CompletionTokens.
func fromOpenAIResponse(resp openai.ChatCompletionResponse) ports.LLMResponse

// convertFinishReason mapea finish_reason → StopReason del dominio.
// "stop"→"end_turn", "tool_calls"→"tool_use", "length"→"max_tokens", otros→"end_turn".
func convertFinishReason(reason string) string
```

**Criterios de aceptación:**
- `go build ./adapters/llm/ollama/...` sin errores
- Cada función ≤20 líneas (ADR-003)
- `TestOllamaCompleteText`, `TestOllamaCompleteWithTools`, `TestOllamaConvertFinishReason` pasan

**Test asociado:** `TestOllamaCompleteText`, `TestOllamaCompleteWithTools`, `TestOllamaConvertFinishReason`

---

### TODO #15 — impl: OllamaClient + mapeo de errores en adapters/llm/ollama/

**Agente:** @developer

**Descripción:** Implementar `OllamaClient` que satisface `ports.LLMClient` usando la
API OpenAI-compatible de Ollama. Como Ollama no requiere API key, se pasa cadena vacía
al SDK. `NewOllamaClient` no retorna error porque BaseURL siempre tiene un valor por
defecto válido.

**Archivos afectados:**
- `adapters/llm/ollama/client.go` (nuevo)
- `adapters/llm/ollama/errors.go` (nuevo)

**Diseño de client.go:**
```go
package ollama

// Config contiene la configuración del cliente Ollama.
type Config struct {
    BaseURL      string // OLLAMA_BASE_URL (default: http://localhost:11434)
    DefaultModel string // OLLAMA_DEFAULT_MODEL (default: mixtral)
}

// OllamaClient implementa ports.LLMClient usando la API OpenAI-compatible de Ollama.
type OllamaClient struct {
    client       *openai.Client
    defaultModel string
}

// NewOllamaClient construye un OllamaClient.
// Si BaseURL está vacío usa "http://localhost:11434".
// Si DefaultModel está vacío usa "mixtral".
func NewOllamaClient(cfg Config) *OllamaClient

// Complete envía la petición a Ollama y retorna la respuesta del dominio.
func (c *OllamaClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error)

var _ ports.LLMClient = &OllamaClient{}
```

**Comportamiento de Complete:**
1. Si `req.Model == ""` usar `c.defaultModel`
2. Construir `openai.ChatCompletionRequest` usando `convert.go`
3. Llamar `c.client.CreateChatCompletion(ctx, chatReq)`
4. Si error: `mapOllamaError("ollama complete", err)` y retornar
5. `fromOpenAIResponse(resp)` y retornar

**Diseño de errors.go — mapeo:**
- `*openai.APIError` con StatusCode 500 → `fmt.Errorf("%s: %w", op, domain.ErrProviderUnavailable)`
- `context.DeadlineExceeded` / `context.Canceled` → propagar sin envolver
- otros → `fmt.Errorf("%s: %w", op, err)`

**Nota:** Ollama local no genera 401/429; si se expone detrás de un proxy autenticado,
`mapOllamaError` se puede extender sin cambiar la interfaz.

**Criterios de aceptación:**
- Los 6 tests de `ollama/client_test.go` pasan (Green)
- `go build ./adapters/llm/ollama/...` sin errores
- `golangci-lint run ./adapters/llm/ollama/...` sin errores
- `Complete` ≤20 líneas (ADR-003)

**Test asociado:** todos los tests de `adapters/llm/ollama/client_test.go`

---

### TODO #16 — infra: Actualizar make test-llm e docs/index.md con Ollama

**Agente:** @devops

**Descripción:** Verificar que `make test-llm` (creado en TODO #10) ya cubre
`./adapters/llm/...` por globbing e incluye los tests de Ollama. Actualizar
`docs/index.md` añadiendo la fila del adaptador Ollama en la tabla de adaptadores.

**Archivos afectados:**
- `Makefile` (verificar/ajustar si el patrón no cubre `ollama/`)
- `docs/index.md`

**Fila a añadir en tabla de adaptadores de docs/index.md:**
```
| LLM Ollama (Mixtral) | libs/ports/llm.go | adapters/llm/ollama/ | planificado | SPRINT-008 |
```

**Criterios de aceptación:**
- `make test-llm` ejecuta 14 tests en total (8 Anthropic+Fake + 6 Ollama), todos pasan
- `grep "ollama" docs/index.md` retorna la fila del adaptador
- `make ci` incluye los tests de Ollama (sin red ni instancia Ollama real)

**Test asociado:** todos los tests de `adapters/llm/`

---

## Matriz de trazabilidad

| TODO | Tipo   | ADR           | Spec                               | Test                                                                      | Impl                                    |
|------|--------|---------------|------------------------------------|---------------------------------------------------------------------------|-----------------------------------------|
| #1   | data   | 001, 004      | —                                  | cubierto por tests del adaptador (#3)                                     | libs/domain/errors.go                   |
| #2   | impl   | 001, 003, 016 | specs/patterns/nodes/llm_call.json | —                                                                         | libs/ports/llm.go                       |
| #3   | test   | 002, 003      | libs/ports/llm.go                  | client_test.go (7 tests, Red)                                             | —                                       |
| #4   | test   | 002, 003      | libs/ports/llm.go                  | fake/client_test.go (1 test, Red)                                         | —                                       |
| #5   | impl   | 001, 003      | libs/ports/llm.go                  | TestFakeLLMClientQueue                                                    | adapters/llm/fake/client.go             |
| #6   | impl   | 001, 003, 004 | libs/ports/llm.go                  | TestCompleteText, TestCompleteWithTools, TestCompleteToolResult            | adapters/llm/anthropic/convert.go       |
| #7   | impl   | 001, 003, 004 | libs/domain/errors.go              | TestCompleteRateLimit, TestCompleteServerError, TestCompleteUnauthorized, TestCompleteContextTimeout | adapters/llm/anthropic/errors.go |
| #8   | impl   | 001, 003, 004 | libs/ports/llm.go                  | todos los tests de client_test.go (Green)                                 | adapters/llm/anthropic/client.go        |
| #9   | infra  | 004, 013      | —                                  | habilita compilación y tests                                              | go.mod, go.sum, .env.example            |
| #10  | infra  | 002           | —                                  | habilita ejecución en CI                                                  | Makefile                                |
| #11  | docs   | 020           | —                                  | —                                                                         | docs/index.md, docs/log.md              |
| #12  | infra  | 004, 013      | —                                  | habilita compilación y tests Ollama                                       | go.mod, go.sum, .env.example            |
| #13  | test   | 002, 003      | libs/ports/llm.go                  | ollama/client_test.go (6 tests, Red)                                      | —                                       |
| #14  | impl   | 001, 003, 004 | libs/ports/llm.go                  | TestOllamaCompleteText, TestOllamaCompleteWithTools, TestOllamaConvertFinishReason | adapters/llm/ollama/convert.go  |
| #15  | impl   | 001, 003, 004 | libs/domain/errors.go, libs/ports/llm.go | todos los tests de ollama/client_test.go (Green)                    | adapters/llm/ollama/client.go, ollama/errors.go |
| #16  | infra+docs | 002, 020  | —                                  | make test-llm ejecuta 14 tests                                            | Makefile, docs/index.md                 |

## Notas de implementación

**SDK de Anthropic Go:** `github.com/anthropics/anthropic-sdk-go`. Cliente con
`BaseURL` personalizada via `option.WithBaseURL(cfg.BaseURL)` — imprescindible
para que `httptest.NewServer` funcione en tests.

**Mock server en tests:** Cada test de `client_test.go` crea su propio
`httptest.NewServer` con un handler ad-hoc que devuelve JSON de la API de Anthropic
(`/v1/messages`). El cliente se construye con `Config{APIKey: "test-key", BaseURL: ts.URL}`.

**Estructura mínima del response JSON mock:**
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

**SDK Ollama:** `github.com/sashabaranov/go-openai` con `openai.ClientConfig{BaseURL: cfg.BaseURL + "/v1"}`.
Ollama no requiere API key — se pasa `""`. El SDK `go-openai` es reutilizable para el
adaptador OpenAI nativo en sprints futuros (solo cambiar BaseURL y añadir la key).

**Mock server para Ollama:** cada test crea un `httptest.NewServer` que sirve
`/v1/chat/completions`. Estructura mínima de la respuesta mock:
```json
{
  "id": "chatcmpl-01",
  "object": "chat.completion",
  "choices": [{"message": {"role": "assistant", "content": "hello"}, "finish_reason": "stop"}],
  "usage": {"prompt_tokens": 10, "completion_tokens": 5}
}
```

**Verificación de interface en tiempo de compilación** (incluir en cada fichero):
```go
var _ ports.LLMClient = &AnthropicClient{}
var _ ports.LLMClient = &FakeLLMClient{}
var _ ports.LLMClient = &OllamaClient{}
```

**Orden de TODOs por dependencias:**
```
#1 (domain errors) ─┐
#2 (port)           ├─ paralelizables
#9 (infra Anthropic)┤
#12 (infra Ollama)  ┘
        ↓
#3 (tests Red Anthropic) + #4 (test Red Fake) + #13 (tests Red Ollama)  ← bloque paralelo
        ↓
#5 (impl Fake) + #6 (convert Anthropic) + #7 (errors Anthropic) + #14 (convert Ollama)
        ↓
#8 (client Anthropic) + #15 (client+errors Ollama)
        ↓
#10+#16 (Makefile) + #11 (docs)
```

## Resultado (completar al cerrar)

- [ ] `TestCompleteText` pasa
- [ ] `TestCompleteWithTools` pasa
- [ ] `TestCompleteToolResult` pasa
- [ ] `TestCompleteRateLimit` pasa — `errors.Is(err, domain.ErrRateLimited)` true
- [ ] `TestCompleteServerError` pasa — `errors.Is(err, domain.ErrProviderUnavailable)` true
- [ ] `TestCompleteUnauthorized` pasa — `errors.Is(err, domain.ErrUnauthorized)` true
- [ ] `TestCompleteContextTimeout` pasa — `context.DeadlineExceeded` propagado
- [ ] `TestFakeLLMClientQueue` pasa
- [ ] `var _ ports.LLMClient = &AnthropicClient{}` compila
- [ ] `var _ ports.LLMClient = &FakeLLMClient{}` compila
- [ ] `go build ./libs/... ./adapters/llm/...` sin errores
- [ ] `golangci-lint run ./libs/... ./adapters/llm/...` sin errores
- [ ] `make test-llm` ejecuta 14 tests (8 Anthropic+Fake + 6 Ollama), todos pasan, sin red real
- [ ] `.env.example` actualizado con variables Anthropic y Ollama
- [ ] `docs/index.md` y `docs/log.md` actualizados
- [ ] `TestOllamaCompleteText` pasa
- [ ] `TestOllamaCompleteWithTools` pasa
- [ ] `TestOllamaCompleteServerError` pasa — `errors.Is(err, domain.ErrProviderUnavailable)` true
- [ ] `TestOllamaCompleteContextTimeout` pasa — `context.DeadlineExceeded` propagado
- [ ] `TestOllamaCompleteModelDefault` pasa — body contiene `"model":"mixtral"`
- [ ] `TestOllamaConvertFinishReason` pasa — "stop"→"end_turn", "tool_calls"→"tool_use", "length"→"max_tokens"
- [ ] `var _ ports.LLMClient = &OllamaClient{}` compila
- [ ] `go build ./adapters/llm/ollama/...` sin errores
- [ ] `golangci-lint run ./adapters/llm/ollama/...` sin errores
