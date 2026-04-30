# SPRINT-009: Executor — Handler del patrón llm_call

## Metadata

- **Fecha inicio:** 2026-04-30
- **Fecha fin estimada:** 2026-05-02
- **Estado:** planificado
- **ADRs aplicados:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-011, ADR-013, ADR-014, ADR-016, ADR-020
- **Specs afectadas:** specs/asyncapi.yaml (operaciones executor), specs/patterns/nodes/llm_call.json
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Bloqueado por:** SPRINT-007 (eventbus), SPRINT-008 (LLMClient + errores de dominio)
- **Bloquea:** SPRINT-010 (orchestrator state machine consume node.executed)

## Objetivo

Implementar el servicio `executor` como worker de eventos: consume `node.execute.requested`
del stream Valkey, detecta el patrón `llm_call`, construye la petición LLM aplicando
`input_mapping`, invoca `LLMClient.Complete`, aplica `output_mapping` sobre la respuesta
y publica `node.executed` o `node.execute.failed` según el resultado. Al finalizar,
el executor puede ejecutar nodos `llm_call` de extremo a extremo con Anthropic, Ollama
y `FakeLLMClient` en tests.

## Alcance

**Entra:**
- Operaciones del executor en `specs/asyncapi.yaml`:
  - `executorConsumeNodeExecuteRequested` (receive)
  - `executorPublishNodeExecuted` (send)
  - `executorPublishNodeExecuteFailed` (send)
- Schemas de payload: `NodeExecuteRequestedData`, `NodeExecutedData`, `NodeExecuteFailedData`
- Estructura interna del servicio `services/executor/`:
  - `main.go` — wiring: config, puertos, arranque del consumer
  - `internal/handler/node_handler.go` — interfaz `NodeHandler`
  - `internal/handler/llm_call.go` — `LLMCallHandler` + `LLMCallConfig`
  - `internal/handler/dispatcher.go` — `Dispatcher` despacha por `pattern`
  - `internal/mapping/input.go` — `ApplyInputMapping`
  - `internal/mapping/output.go` — `ApplyOutputMapping`
  - `internal/consumer/node_execute.go` — `NodeExecuteConsumer`
- Evaluador de paths simplificado: `state.variables.<name>`, `state.messages[-1].content`,
  `output.content`, `output.stop_reason`
- Mapeo de errores LLM: `ErrRateLimited`/`ErrProviderUnavailable` → retryable; `ErrUnauthorized` → no retryable
- 10 tests unitarios sin red (FakeLLMClient + fakePublisher inline)
- 1 test de integración (build tag `integration`) con Valkey real
- Variables de entorno en `.env.example`, target `make test-executor`

**No entra:**
- Otros patrones de nodo (`tool_use`, `react`, `reflection`, `router`, `guardrail`, `subgraph`)
- Streaming de tokens — requiere ADR separado
- Evaluador de expresiones arbitrarias para mappings — sprint futuro
- Persistencia del estado de ejecución en Ent — requiere SPRINT-002
- Retry con backoff propio — se usa la política DLQ de SPRINT-007
- Tests contra la API real de Anthropic/Ollama — excluidos de `make ci`
- Telemetría OpenTelemetry — sprint dedicado futuro

## Dependencias

- **Bloqueado por:**
  - SPRINT-007 (`libs/ports/eventbus.go`, `adapters/eventbus/valkey/`, DLQ, ACK/NACK)
  - SPRINT-008 (`libs/ports/llm.go`, `adapters/llm/anthropic/`, `adapters/llm/ollama/`,
    `adapters/llm/fake/`, errores `ErrUnauthorized`, `ErrRateLimited`, `ErrProviderUnavailable`)
- **Paralelo a:** SPRINT-005, SPRINT-006 (dashboard — sin conflicto de ficheros)
- **Bloquea:** executor patrón `tool_use` (SPRINT-010+), tests end-to-end orchestrator→executor

## Contratos de comportamiento

### C1 — `LLMCallHandler.Handle` — éxito sin mappings

```
Given: Evento node.execute.requested con pattern="llm_call", config={model:"claude-sonnet-4-6",max_tokens:100}
      FakeLLMClient configurado con Responses=[{Content:"respuesta ok",StopReason:"end_turn"}]
When: LLMCallHandler.Handle(ctx, data) se ejecuta
Then: Se publica exactamente un evento en el stream "node.executed"
      variables_update["response"] == "respuesta ok"
      duration_ms >= 0
      Handler retorna nil (consumer hace ACK)
```

### C2 — `LLMCallHandler.Handle` — ErrRateLimited → retryable

```
Given: FakeLLMClient que retorna fmt.Errorf("test: %w", domain.ErrRateLimited)
When: LLMCallHandler.Handle(ctx, data)
Then: Se publica evento en "node.execute.failed" con error_code="rate_limited", retryable=true
      Handler retorna error (consumer NO hace ACK → NACK en Valkey)
      No se publica evento en "node.executed"
```

### C3 — `ApplyInputMapping` — path state.variables

```
Given: inputMapping={"user_message":"state.variables.query"}, variables={"query":"¿qué es dago?"}
When: ApplyInputMapping(inputMapping, variables, [])
Then: Retorna []ports.Message{{Role:"user", Content:"¿qué es dago?"}}
      No retorna error
      La función es pura: misma entrada → mismo resultado
```

## TODOs

### TODO #1 — spec: Definir operaciones del executor en specs/asyncapi.yaml

**Agente:** @developer

**Descripción:** Añadir en la sección `operations` las tres operaciones del executor y
en `components/schemas` los tres payloads con campos tipados. Los canales ya existen;
lo que falta son los schemas de datos completos y el enlace operación-canal.

**Archivos afectados:**
- `specs/asyncapi.yaml`

**Operaciones a añadir:**
```yaml
operations:
  executorConsumeNodeExecuteRequested:
    action: receive
    channel:
      $ref: '#/channels/nodeExecuteRequested'
    summary: Executor consume petición de ejecución de nodo
  executorPublishNodeExecuted:
    action: send
    channel:
      $ref: '#/channels/nodeExecuted'
    summary: Executor publica resultado exitoso de ejecución de nodo
  executorPublishNodeExecuteFailed:
    action: send
    channel:
      $ref: '#/channels/nodeExecuteFailed'
    summary: Executor publica fallo en ejecución de nodo
```

**Schemas a añadir:**

`NodeExecuteRequestedData`: `execution_id`, `graph_id`, `node_id`, `node_key`, `pattern`,
`config` (object), `variables` (object), `messages` (array de `{role, content}`), `auth` (string)

`NodeExecutedData`: `execution_id`, `graph_id`, `node_id`, `node_key`,
`output` (object), `variables_update` (object), `duration_ms` (integer)

`NodeExecuteFailedData`: `execution_id`, `graph_id`, `node_id`, `node_key`,
`error` (string), `error_code` (string: "rate_limited"|"provider_unavailable"|"unauthorized"|"execution_error"),
`retryable` (boolean)

**Criterios de aceptación:**
- `asyncapi validate specs/asyncapi.yaml` sin errores
- Cada operación referencia correctamente su canal
- Los tres schemas tienen todos los campos con tipos correctos

**Test asociado:** — (spec pura)

---

### TODO #2 — spec: Verificar specs/patterns/nodes/llm_call.json

**Agente:** @developer

**Descripción:** Revisar que `llm_call.json` documenta defaults (`temperature: 0.7`,
`max_tokens: 2048`) y el alcance de los paths soportados en `input_mapping`/`output_mapping`
para SPRINT-009. Solo enriquecer descripciones — no cambiar la estructura.

**Archivos afectados:**
- `specs/patterns/nodes/llm_call.json`

**Criterios de aceptación:**
- El schema documenta los defaults y los paths soportados
- La validación JSON Schema del schema no reporta errores

**Test asociado:** —

---

### TODO #3 — test: Tests Red de ApplyInputMapping

**Agente:** @qa

**Descripción:** Escribir los tests de `mapping/input.go` ANTES de implementarlo (Red).

**Archivos afectados:**
- `services/executor/internal/mapping/input_test.go` (nuevo)

**Tests a implementar:**
```go
package mapping_test

// TestApplyInputMapping_NoMapping
// Entrada: inputMapping nil, variables {}, messages [{role:"user",content:"hola"}]
// Esperado: []ports.Message{{Role:"user",Content:"hola"}}

// TestApplyInputMapping_EmptyMapping
// Entrada: inputMapping {}, messages [{role:"user",content:"test"}]
// Esperado: messages sin modificar (igual que sin mapping)

// TestApplyInputMapping_StateMessagesLast
// Entrada: inputMapping {"user_message":"state.messages[-1].content"}
//          messages [{role:"user",content:"primero"},{role:"assistant",content:"resp"},{role:"user",content:"segundo"}]
// Esperado: []ports.Message con último mensaje content="segundo"

// TestApplyInputMapping_StateVariables
// Entrada: inputMapping {"user_message":"state.variables.query"}
//          variables {"query":"¿qué es dago?"}
// Esperado: []ports.Message{{Role:"user",Content:"¿qué es dago?"}}

// TestApplyInputMapping_StateVariables_Missing
// Entrada: inputMapping {"user_message":"state.variables.nonexistent"}, variables {}
// Esperado: error descriptivo (variable no existe)

// TestApplyInputMapping_UnknownPath
// Entrada: inputMapping {"x":"state.unknown.path"}
// Esperado: error (path no soportado)
```

**Criterios de aceptación:**
- Los 6 tests fallan en rojo (Red confirmado)
- Solo importan `libs/ports/` — sin infraestructura

**Test asociado:** este TODO ES el test (fase Red)

---

### TODO #4 — test: Tests Red de ApplyOutputMapping

**Agente:** @qa

**Descripción:** Escribir los tests de `mapping/output.go` ANTES de implementarlo (Red).

**Archivos afectados:**
- `services/executor/internal/mapping/output_test.go` (nuevo)

**Tests a implementar:**
```go
package mapping_test

// TestApplyOutputMapping_NoMapping
// Entrada: outputMapping nil, response.Content="respuesta generada"
// Esperado: map[string]any{"response":"respuesta generada"}

// TestApplyOutputMapping_ContentToVariable
// Entrada: outputMapping {"state.variables.summary":"output.content"}, Content="resumen"
// Esperado: map[string]any{"summary":"resumen"}

// TestApplyOutputMapping_StopReasonToVariable
// Entrada: outputMapping {"state.variables.reason":"output.stop_reason"}, StopReason="max_tokens"
// Esperado: map[string]any{"reason":"max_tokens"}

// TestApplyOutputMapping_MultipleTargets
// Entrada: outputMapping {"state.variables.text":"output.content","state.variables.stop":"output.stop_reason"}
// Esperado: map con ambas claves correctas

// TestApplyOutputMapping_UnknownSourcePath
// Entrada: outputMapping {"state.variables.x":"output.unknown_field"}
// Esperado: error (campo de output no soportado)

// TestApplyOutputMapping_InvalidTargetPath
// Entrada: outputMapping {"not.a.state.variable":"output.content"}
// Esperado: error (destino debe ser state.variables.<name>)
```

**Criterios de aceptación:**
- Los 6 tests fallan en rojo
- Solo importan `libs/ports/` (tipo `LLMResponse`)

**Test asociado:** este TODO ES el test (fase Red)

---

### TODO #5 — test: Tests Red de LLMCallHandler

**Agente:** @qa

**Descripción:** Escribir los tests de `handler/llm_call.go` ANTES de implementarlo.
Usan `FakeLLMClient` (SPRINT-008) y un `fakePublisher` inline definido en el test.

**Archivos afectados:**
- `services/executor/internal/handler/llm_call_test.go` (nuevo)

**fakePublisher inline (privado al test):**
```go
type fakePublisher struct{ published []ports.Event }
func (f *fakePublisher) Publish(_ context.Context, _ ports.PublishOptions, e ports.Event) error {
    f.published = append(f.published, e)
    return nil
}
func (f *fakePublisher) Close() error { return nil }
```

**Tests a implementar:**
```go
// TestLLMCallHandler_Success
// Config: {"model":"claude-sonnet-4-6","max_tokens":100}; sin mappings
// FakeLLMClient: Content="respuesta ok", StopReason="end_turn"
// Esperado: evento node.executed publicado; variables_update["response"]=="respuesta ok"; duration_ms>=0

// TestLLMCallHandler_WithInputMapping
// Config: input_mapping={"user_message":"state.variables.query"}
// Variables: {"query":"explica dago"}
// Esperado: FakeLLMClient.Calls[0].Messages[0].Content=="explica dago"

// TestLLMCallHandler_WithOutputMapping
// Config: output_mapping={"state.variables.answer":"output.content"}
// FakeLLMClient: Content="respuesta mapeada"
// Esperado: variables_update["answer"]=="respuesta mapeada"

// TestLLMCallHandler_RateLimited
// FakeLLMClient: retorna fmt.Errorf("...%w",domain.ErrRateLimited)
// Esperado: node.execute.failed con error_code=="rate_limited", retryable==true; handler retorna error

// TestLLMCallHandler_ProviderUnavailable
// FakeLLMClient: retorna domain.ErrProviderUnavailable
// Esperado: error_code=="provider_unavailable", retryable==true

// TestLLMCallHandler_Unauthorized
// FakeLLMClient: retorna domain.ErrUnauthorized
// Esperado: error_code=="unauthorized", retryable==false

// TestLLMCallHandler_ExecutionError
// FakeLLMClient: retorna errors.New("internal error")
// Esperado: error_code=="execution_error", retryable==false
```

**Criterios de aceptación:**
- Los 7 tests fallan en rojo
- Ningún test importa adaptadores concretos ni llama a Valkey ni APIs reales
- Cada test verifica el payload exacto del evento publicado

**Test asociado:** este TODO ES el test (fase Red)

---

### TODO #6 — test: Test de integración del consumer (build tag `integration`)

**Agente:** @qa

**Descripción:** Test que ejercita `NodeExecuteConsumer` con Valkey real. Publica
`node.execute.requested` y verifica que el consumer procesa el mensaje y publica
`node.executed` en el stream correcto. Usa `FakeLLMClient` inyectado.

**Archivos afectados:**
- `services/executor/internal/consumer/node_execute_test.go` (nuevo, `//go:build integration`)

**Test a implementar:**
```go
//go:build integration

// TestExecutorConsumer_LLMCallSuccess
// Precondición: Valkey en EXECUTOR_VALKEY_ADDR (default localhost:6379)
// Setup: crear consumer group "executor-group" en "node.execute.requested"
// Acción: publicar node.execute.requested con pattern="llm_call" y config mínimo
// Verificación: stream "node.executed" contiene evento con execution_id y node_id correctos
//               y variables_update.response presente
// FakeLLMClient inyectado, timeout: 5 segundos
```

**Criterios de aceptación:**
- `go test -tags=integration ./services/executor/...` pasa
- No se ejecuta en `make ci`
- El test falla si el consumer no hace ACK del mensaje

**Test asociado:** este TODO ES el test

---

### TODO #7 — impl: mapping/input.go

**Agente:** @developer

**Descripción:** Implementar `ApplyInputMapping` siguiendo el contrato de los tests #3.
Evaluador de paths simplificado para SPRINT-009.

**Archivos afectados:**
- `services/executor/internal/mapping/input.go` (nuevo)

**Firma:**
```go
package mapping

import "github.com/aescanero/dago/libs/ports"

// ApplyInputMapping construye []ports.Message para LLMRequest.
// Si inputMapping es nil o vacío, devuelve messages tal cual.
func ApplyInputMapping(
    inputMapping map[string]string,
    variables    map[string]any,
    messages     []ports.Message,
) ([]ports.Message, error)
```

**Paths soportados (valores del mapa):**
- `state.messages[-1].content` → último mensaje del slice `messages`
- `state.variables.<name>` → `variables["<name>"]`; error si no existe
- Cualquier otro path → `fmt.Errorf("mapping/input: unsupported path %q: %w", path, ErrUnsupportedPath)`

**Criterios de aceptación:**
- Los 6 tests del TODO #3 pasan en verde
- Función ≤20 líneas (ADR-003)
- Sin imports de infraestructura

**Test asociado:** TODO #3

---

### TODO #8 — impl: mapping/output.go

**Agente:** @developer

**Descripción:** Implementar `ApplyOutputMapping` siguiendo el contrato de los tests #4.

**Archivos afectados:**
- `services/executor/internal/mapping/output.go` (nuevo)

**Firma:**
```go
package mapping

import "github.com/aescanero/dago/libs/ports"

// ApplyOutputMapping construye map[string]any de actualizaciones de variables.
// Si outputMapping es nil devuelve {"response": response.Content}.
func ApplyOutputMapping(
    outputMapping map[string]string,
    response      ports.LLMResponse,
) (map[string]any, error)
```

**Campos de origen soportados:** `output.content`, `output.stop_reason`

**Formato de destino válido:** `state.variables.<name>` (extrae `<name>` como clave del resultado)

**Criterios de aceptación:**
- Los 6 tests del TODO #4 pasan en verde
- Función ≤20 líneas (ADR-003)

**Test asociado:** TODO #4

---

### TODO #9 — impl: handler/node_handler.go, handler/llm_call.go, handler/dispatcher.go

**Agente:** @developer

**Descripción:** Implementar la interfaz `NodeHandler`, `LLMCallHandler` y `Dispatcher`.
El handler recibe `LLMClient` y `EventPublisher` por inyección en el constructor.

**Archivos afectados:**
- `services/executor/internal/handler/node_handler.go` (nuevo)
- `services/executor/internal/handler/llm_call.go` (nuevo)
- `services/executor/internal/handler/dispatcher.go` (nuevo)

**Tipos de datos del evento (en `node_handler.go`):**
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

**Algoritmo de LLMCallHandler.Handle:**
1. Deserializar `data.Config` → `LLMCallConfig`; defaults: temperature=0.7, max_tokens=2048
2. `ApplyInputMapping(config.InputMapping, data.Variables, data.Messages)` → messages
3. Construir `ports.LLMRequest{Model, System, MaxTokens, Temperature, Messages}`
4. `start := time.Now()`
5. `llmClient.Complete(ctx, req)` → si error: mapear → publicar `node.execute.failed` → retornar error
6. `ApplyOutputMapping(config.OutputMapping, resp)` → variablesUpdate
7. Serializar output y variablesUpdate → publicar `node.executed`
8. Retornar nil

**Mapeo de errores LLM → ErrorCode+Retryable:**
- `errors.Is(err, domain.ErrRateLimited)` → "rate_limited", true
- `errors.Is(err, domain.ErrProviderUnavailable)` → "provider_unavailable", true
- `errors.Is(err, domain.ErrUnauthorized)` → "unauthorized", false
- otros → "execution_error", false

**Dispatcher:**
```go
type Dispatcher struct{ handlers map[string]NodeHandler }
func NewDispatcher(handlers map[string]NodeHandler) *Dispatcher
func (d *Dispatcher) Dispatch(ctx context.Context, data NodeExecuteRequestedData) error
// Si pattern no registrado: publicar node.execute.failed con error_code="execution_error"
```

**Criterios de aceptación:**
- Los 7 tests del TODO #5 pasan en verde
- Handlers solo importan `libs/ports/` y `libs/domain/` — cero acoplamiento a adapters
- Streams de publicación: exactamente `"node.executed"` y `"node.execute.failed"`
- `LLMCallHandler.Handle` ≤20 líneas (extraer funciones privadas si es necesario)

**Test asociado:** TODO #5

---

### TODO #10 — impl: consumer/node_execute.go

**Agente:** @developer

**Descripción:** `NodeExecuteConsumer` suscribe a `node.execute.requested`, deserializa
el envelope del evento y delega al `Dispatcher`. ACK si éxito o error no-retryable;
NACK si error retryable.

**Archivos afectados:**
- `services/executor/internal/consumer/node_execute.go` (nuevo)

**Diseño:**
```go
type NodeExecuteConsumer struct {
    consumer   ports.EventConsumer
    dispatcher *handler.Dispatcher
}

func NewNodeExecuteConsumer(consumer ports.EventConsumer, d *handler.Dispatcher) *NodeExecuteConsumer

// Run bloquea hasta ctx cancelado. Por cada evento de node.execute.requested:
//   1. Deserializar data como NodeExecuteRequestedData
//   2. Si pattern != soportado: ACK silencioso + log
//   3. dispatcher.Dispatch(ctx, data)
//   4. Si nil → ACK
//   5. Si errors.Is(err, domain.ErrRateLimited||ErrProviderUnavailable) → NACK
//   6. Si otro error → ACK (failure ya publicado como node.execute.failed)
func (c *NodeExecuteConsumer) Run(ctx context.Context) error
```

**Criterios de aceptación:**
- Test de integración del TODO #6 pasa en verde
- ACK correcto para errores no-retryable
- NACK para rate_limited y provider_unavailable

**Test asociado:** TODO #6

---

### TODO #11 — impl: services/executor/main.go

**Agente:** @developer

**Descripción:** Punto de entrada del servicio. Lee config desde env, instancia adaptadores,
construye Dispatcher con `LLMCallHandler` registrado para `"llm_call"`, arranca consumer
con signal handling (SIGTERM, SIGINT).

**Archivos afectados:**
- `services/executor/main.go` (nuevo)

**Variables de entorno:**
- `EXECUTOR_VALKEY_ADDR` (default: `localhost:6379`)
- `EXECUTOR_GROUP` (default: `executor-group`)
- `EXECUTOR_CONSUMER_NAME` (default: `executor-1`)
- `EXECUTOR_LLM_PROVIDER` (default: `anthropic`; valores: `anthropic`, `ollama`)
- `EXECUTOR_BLOCK_DURATION_MS` (default: `5000`)
- Si `anthropic`: lee `ANTHROPIC_API_KEY`
- Si `ollama`: lee `OLLAMA_BASE_URL` (default `http://localhost:11434`)

**Criterios de aceptación:**
- `go build ./services/executor/` compila sin errores
- El servicio termina limpiamente con SIGTERM

**Test asociado:** smoke — `go build ./services/executor/`

---

### TODO #12 — infra: Makefile + .env.example

**Agente:** @devops

**Descripción:** Target `make test-executor` para tests unitarios. Variables de entorno
del executor en `.env.example`.

**Archivos afectados:**
- `Makefile`
- `.env.example`

**Target:**
```makefile
## test-executor: tests unitarios del servicio executor
test-executor:
	go test -count=1 -timeout 30s ./services/executor/...
```

**Variables en `.env.example`:**
```
# Executor
EXECUTOR_VALKEY_ADDR=localhost:6379
EXECUTOR_GROUP=executor-group
EXECUTOR_CONSUMER_NAME=executor-1
EXECUTOR_LLM_PROVIDER=anthropic
EXECUTOR_BLOCK_DURATION_MS=5000
```

**Criterios de aceptación:**
- `make test-executor` pasa los 10 tests unitarios sin red ni credenciales reales

**Test asociado:** todos los tests unitarios del executor

---

### TODO #13 — docs: Actualizar docs/index.md y docs/log.md

**Agente:** @docs

**Descripción:** Registrar SPRINT-009 en el índice y log. Anotar en la tabla de Servicios
que `executor` tiene implementación parcial (patrón `llm_call`).

**Archivos afectados:**
- `docs/index.md`
- `docs/log.md`

**Criterios de aceptación:**
- `grep "SPRINT-009" docs/index.md` retorna la fila
- `grep "SPRINT-009" docs/log.md` retorna la entrada

**Test asociado:** —

---

## Matriz de trazabilidad

| TODO | Tipo  | ADR              | Spec                             | Test                                   | Impl                                    |
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

## Notas de implementación

**Orden de TODOs por dependencias:**
```
#1 (spec asyncapi) ─┐
#2 (spec llm_call)  ┘ paralelizables
         ↓
#3 (test input)  + #4 (test output)  ← paralelizables
         ↓
#5 (test LLMCallHandler)
         ↓
#7 (impl input) + #8 (impl output)  ← paralelizables
         ↓
#9 (impl handler)
         ↓
#6 (test consumer) + #10 (impl consumer)  ← en paralelo si se tiene stub del consumer
         ↓
#11 (impl main.go)
         ↓
#12 (infra) + #13 (docs)
```

**Reglas de arquitectura críticas:**
- `services/executor/internal/` es código privado del servicio — ningún otro servicio lo importa
- Los handlers solo importan `libs/ports/` y `libs/domain/` (no `adapters/`)
- Los adaptadores concretos solo se instancian en `main.go`
- El consumer importa `libs/ports/` + `internal/handler/` + `internal/mapping/`

**FakeEventPublisher en tests unitarios:**
El `fakePublisher` se define como tipo privado en cada fichero `_test.go` que lo necesite.
No se exporta hasta que otro handler lo necesite — en ese momento se extrae a
`adapters/eventbus/fake/`.

**Selección de LLMClient:**
La selección de proveedor se hace en `main.go` según `EXECUTOR_LLM_PROVIDER`, no dentro
del handler. El handler recibe un único `ports.LLMClient` ya configurado.

**ACK/NACK en el consumer:**
El consumer usa `errors.Is` sobre los centinelas de dominio para determinar retryable,
no el campo `Retryable` del evento publicado. Errores no-retryable reciben ACK para que
no se reintentan (el `node.execute.failed` ya fue publicado para que el orchestrator actúe).

**Commits del sprint:**
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

## Resultado (completar al cerrar)

- [ ] `TestApplyInputMapping_NoMapping` pasa
- [ ] `TestApplyInputMapping_StateMessagesLast` pasa
- [ ] `TestApplyInputMapping_StateVariables` pasa
- [ ] `TestApplyInputMapping_StateVariables_Missing` pasa — error descriptivo
- [ ] `TestApplyOutputMapping_NoMapping` pasa — default {"response": content}
- [ ] `TestApplyOutputMapping_ContentToVariable` pasa
- [ ] `TestApplyOutputMapping_MultipleTargets` pasa
- [ ] `TestLLMCallHandler_Success` pasa — node.executed publicado con variables_update
- [ ] `TestLLMCallHandler_WithInputMapping` pasa — LLMRequest.Messages[0].Content correcto
- [ ] `TestLLMCallHandler_WithOutputMapping` pasa — variables_update mapeado correctamente
- [ ] `TestLLMCallHandler_RateLimited` pasa — error_code=="rate_limited", retryable==true
- [ ] `TestLLMCallHandler_ProviderUnavailable` pasa — retryable==true
- [ ] `TestLLMCallHandler_Unauthorized` pasa — retryable==false
- [ ] `TestLLMCallHandler_ExecutionError` pasa — error_code=="execution_error", retryable==false
- [ ] `TestExecutorConsumer_LLMCallSuccess` pasa (build tag integration)
- [ ] `go build ./services/executor/` sin errores
- [ ] `golangci-lint run ./services/executor/...` sin errores
- [ ] `make test-executor` ejecuta 10 tests unitarios, todos pasan, sin red
- [ ] `specs/asyncapi.yaml` validado con operaciones del executor
- [ ] `docs/index.md` y `docs/log.md` actualizados
