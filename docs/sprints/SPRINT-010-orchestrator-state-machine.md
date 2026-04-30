# SPRINT-010: Orchestrator state machine — Submit, validar, ejecutar, transicionar, completar

## Metadata

- **Fecha inicio:** 2026-05-05
- **Fecha fin estimada:** 2026-05-07
- **Estado:** planificado
- **ADRs aplicados:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-007, ADR-008, ADR-011, ADR-014, ADR-016, ADR-020
- **Specs afectadas:** specs/asyncapi.yaml (operaciones orchestrator), specs/paths/executions.yaml (422)
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Bloqueado por:** SPRINT-003 (ExecutionRepository, StartExecution, Ent), SPRINT-007 (event bus), SPRINT-009 (formatos de evento)
- **Bloquea:** SPRINT-011 (executor tool_use), SPRINT-015 (memoria episódica)

## Objetivo

Conectar el orchestrator con el event bus: validar el grafo al someter una ejecución,
publicar el primer evento `node.execute.requested`, consumir `node.executed` /
`node.execute.failed`, actualizar estado y transicionar hasta completar o fallar.
Solo se soportan grafos con aristas `sequential` en este sprint.

## Alcance

- **AsyncAPI** — nuevas operaciones del orchestrator: publicar `node.execute.requested`,
  consumir `node.executed` y `node.execute.failed`, publicar `graphCompleted` y `graphFailed`.
- **OpenAPI** — respuesta `422 GRAPH_VALIDATION_ERROR` en `POST /api/v1/executions`.
- **Domain** — `ErrGraphValidation` en `libs/domain/errors.go`. `GraphDefinition` struct
  con `EntryNode`, `Nodes map[string]NodeDefinition`, `Edges []EdgeDefinition`.
- **Port** — añadir `UpdateExecution` a `ExecutionRepository`.
- **State machine** en `services/orchestrator/internal/statemachine/`:
  - `graph_validator.go`: `ValidateGraph(g GraphDefinition) error` con `dominikbraun/graph`.
  - `traversal.go`: `NextNode(g GraphDefinition, currentNode string) (string, error)`.
  - `execution_sm.go`: `ExecutionStateMachine` con `HandleNodeExecuted` y `HandleNodeExecuteFailed`.
- **Consumer** `node_result.go` en `services/orchestrator/internal/consumer/` —
  consume `node.executed` y `node.execute.failed`, delega en `ExecutionStateMachine`.
- **StartExecution extendido** — valida grafo, publica `node.execute.requested`, pone
  estado `running` (antes quedaba `pending`).
- **ErrRetryable** — sentinel en `libs/domain/errors.go` para que el consumer propague NACK.
- Tests: 4 unitarios (ValidateGraph, NextNode, HandleNodeExecuted, HandleNodeExecuteFailed),
  2 de integración con Valkey real (build tag `integration`).

## Dependencias

- **Bloqueado por:** SPRINT-003 (ExecutionRepository, StartExecution, Ent), SPRINT-007 (event bus),
  SPRINT-009 (executor llm_call, formatos de evento).
- **Bloquea:** SPRINT-011 (executor tool_use), SPRINT-015 (memoria episódica).

## Contratos de comportamiento

### C1 — `ValidateGraph` — grafo con solo aristas sequential

```
Given: GraphDefinition con entry_node="a", nodes={"a":{},"b":{}}, edges=[{type:"sequential",from:"a",to:"b"}]
When: ValidateGraph(graph)
Then: Retorna nil (sin error)
      Todos los nodos son alcanzables desde entry_node
```

### C2 — `ValidateGraph` — arista no soportada

```
Given: GraphDefinition con edge de tipo "conditional"
When: ValidateGraph(graph)
Then: Retorna error tal que errors.Is(err, domain.ErrGraphValidation) == true
      El mensaje del error contiene "unsupported edge type: conditional"
```

### C3 — `ExecutionStateMachine.HandleNodeExecuted` — nodo terminal

```
Given: Execution en status="running", GraphDefinition donde currentNode no tiene aristas de salida
When: HandleNodeExecuted(ctx, exec, graph, currentNode, output, auth)
Then: Se publica evento graph.completed en el stream correspondiente
      exec.Status se actualiza a "completed"
      UpdateExecution es llamado con el exec actualizado
      Handler retorna nil
```

### C4 — `StartExecution` extendido — grafo inválido → 422

```
Given: Graph con aristas "conditional" en su campo definition
When: POST /api/v1/executions con graph_id de ese grafo
Then: HTTP 422, ErrorResponse con code="GRAPH_VALIDATION_ERROR"
      No se persiste ninguna Execution en base de datos
      No se publica ningún evento en Valkey
```

## Nota sobre orden TDD

> Los TODOs #14 y #15 (tests de integración) deben ejecutarse en Red ANTES de los TODOs de implementación (#7–#12). El orden correcto de ejecución es:
> `#1 → #2 → #3 → #4 → #14 → #15 → #5 → #6 → #7 → #8 → #9 → #10 → #11 → #12 → #16 → #13 → #17`

## TODOs

### TODO #1 — spec: AsyncAPI — operaciones orchestrator [spec]

**Agente:** @developer

**Objetivo:** Registrar las operaciones del orchestrator en `specs/asyncapi.yaml`.

Añadir en `operations`:
```yaml
orchestratorPublishNodeExecuteRequested:
  action: send
  channel: $ref: '#/channels/nodeExecuteRequested'
  bindings: {valkey: {group: orchestrator}}

orchestratorConsumeNodeExecuted:
  action: receive
  channel: $ref: '#/channels/nodeExecuted'
  bindings: {valkey: {group: orchestrator}}

orchestratorConsumeNodeExecuteFailed:
  action: receive
  channel: $ref: '#/channels/nodeExecuteFailed'
  bindings: {valkey: {group: orchestrator}}

orchestratorPublishGraphCompleted:
  action: send
  channel: $ref: '#/channels/graphCompleted'

orchestratorPublishGraphFailed:
  action: send
  channel: $ref: '#/channels/graphFailed'
```

Añadir schemas si no existen: `GraphCompletedData` (executionId, graphId, durationMs),
`GraphFailedData` (executionId, graphId, error, errorCode).

**Ficheros:** `specs/asyncapi.yaml`

---

### TODO #2 — spec: OpenAPI — 422 en POST /executions [spec]

**Agente:** @developer

**Objetivo:** Documentar la respuesta `422` con código `GRAPH_VALIDATION_ERROR` en
`POST /api/v1/executions` (en `specs/paths/executions.yaml`).

```yaml
'422':
  description: Graph validation failed (unsupported edge types, unreachable nodes, etc.)
  content:
    application/json:
      schema:
        $ref: '../schemas/error.yaml'
      example:
        code: GRAPH_VALIDATION_ERROR
        message: "graph contains unsupported edge type: conditional"
```

**Ficheros:** `specs/paths/executions.yaml`

---

### TODO #3 — test: ValidateGraph — Red [test]

**Agente:** @qa

**Objetivo:** Tests unitarios para `ValidateGraph` antes de implementar.

Casos:
1. Grafo válido (3 nodos sequential) → `nil`.
2. `entry_node` no existe en `nodes` → `ErrGraphValidation`.
3. Arista `conditional` → `ErrGraphValidation` ("unsupported edge type: conditional").
4. Nodo inalcanzable desde `entry_node` → `ErrGraphValidation`.

**Fichero:** `services/orchestrator/internal/statemachine/graph_validator_test.go`

---

### TODO #4 — test: NextNode + ExecutionStateMachine — Red [test]

**Agente:** @qa

**Objetivo:** Tests unitarios antes de implementar `traversal.go` y `execution_sm.go`.

`NextNode`:
1. Nodo con sucesor sequential → devuelve clave del siguiente.
2. Nodo sin salida (terminal) → `("", nil)`.

`HandleNodeExecuted`:
3. Nodo intermedio → publica `node.execute.requested` y actualiza estado.
4. Nodo terminal → publica `graph.completed` y pone ejecución en `completed`.

`HandleNodeExecuteFailed`:
5. `retryable=false` → publica `graph.failed`, pone ejecución en `failed`, retorna nil.
6. `retryable=true` → retorna `ErrRetryable` (consumer debe NACK).

**Ficheros:** `services/orchestrator/internal/statemachine/traversal_test.go`,
`services/orchestrator/internal/statemachine/execution_sm_test.go`

---

### TODO #5 — data: ErrGraphValidation + ErrRetryable + GraphDefinition [data]

**Agente:** @developer

**Objetivo:** Añadir al dominio compartido los nuevos tipos y errores.

En `libs/domain/errors.go`:
```go
var ErrGraphValidation = errors.New("domain: graph validation failed")
var ErrRetryable      = errors.New("domain: retryable — consumer must NACK")
```

En `libs/domain/graph.go` (nuevo archivo si no existe):
```go
type GraphDefinition struct {
    EntryNode string                     `json:"entry_node"`
    Nodes     map[string]NodeDefinition  `json:"nodes"`
    Edges     []EdgeDefinition           `json:"edges"`
}

type NodeDefinition struct {
    Pattern string          `json:"pattern"`
    Config  json.RawMessage `json:"config"`
}

type EdgeDefinition struct {
    Type string `json:"type"`
    From string `json:"from"`
    To   string `json:"to"`
}
```

**Ficheros:** `libs/domain/errors.go`, `libs/domain/graph.go`

---

### TODO #6 — impl: UpdateExecution en ExecutionRepository [impl]

**Agente:** @developer

**Objetivo:** Añadir `UpdateExecution` al puerto `ExecutionRepository`.

```go
type ExecutionRepository interface {
    Create(ctx context.Context, exec *domain.Execution) error
    FindByID(ctx context.Context, id string) (*domain.Execution, error)
    CountActiveByGraph(ctx context.Context, graphID string) (int, error)
    UpdateExecution(ctx context.Context, exec *domain.Execution) error  // nuevo
}
```

El campo `CurrentNode` se añade a `domain.Execution` si no existe.

**Ficheros:** `libs/ports/storage.go`, `libs/domain/execution.go`

---

### TODO #7 — impl: ValidateGraph con dominikbraun/graph [impl]

**Agente:** @developer

**Objetivo:** Implementar `ValidateGraph` en Green.

```go
// services/orchestrator/internal/statemachine/graph_validator.go
func ValidateGraph(g domain.GraphDefinition) error {
    // 1. entry_node existe en nodes
    // 2. todos los edges son "sequential"
    // 3. construir dominikbraun/graph y verificar alcanzabilidad
    //    desde entry_node a todos los nodos
}
```

Usar `github.com/dominikbraun/graph` para construir el DAG y verificar que todos
los vértices son alcanzables desde `entry_node`.

**Fichero:** `services/orchestrator/internal/statemachine/graph_validator.go`

---

### TODO #8 — impl: NextNode [impl]

**Agente:** @developer

**Objetivo:** Implementar `NextNode` en Green.

```go
// traversal.go
func NextNode(g domain.GraphDefinition, currentNode string) (string, error) {
    for _, e := range g.Edges {
        if e.From == currentNode && e.Type == "sequential" {
            return e.To, nil
        }
    }
    return "", nil  // nodo terminal
}
```

**Fichero:** `services/orchestrator/internal/statemachine/traversal.go`

---

### TODO #9 — impl: ExecutionStateMachine [impl]

**Agente:** @developer

**Objetivo:** Implementar la máquina de estados en Green.

```go
type ExecutionStateMachine struct {
    repo      ports.ExecutionRepository
    publisher ports.EventPublisher
}

func (sm *ExecutionStateMachine) HandleNodeExecuted(
    ctx context.Context,
    exec *domain.Execution,
    graph domain.GraphDefinition,
    nodeKey string,
    output json.RawMessage,
    auth string,
) error

func (sm *ExecutionStateMachine) HandleNodeExecuteFailed(
    ctx context.Context,
    exec *domain.Execution,
    retryable bool,
    errMsg, errCode string,
    auth string,
) error
```

- `HandleNodeExecuted`: si `NextNode` devuelve nodo → publica `node.execute.requested`
  (nuevo nodo), actualiza `exec.CurrentNode` y llama `UpdateExecution`.
  Si terminal → publica `graph.completed`, pone `exec.Status = "completed"`,
  llama `UpdateExecution`.
- `HandleNodeExecuteFailed`: si `!retryable` → publica `graph.failed`, pone
  `exec.Status = "failed"`, llama `UpdateExecution`, retorna nil.
  Si `retryable` → retorna `domain.ErrRetryable` (consumer debe NACK).
- Idempotencia: `CanTransitionTo(current, next Status) bool` evita doble transición.

**Fichero:** `services/orchestrator/internal/statemachine/execution_sm.go`

---

### TODO #10 — impl: Consumer node_result [impl]

**Agente:** @developer

**Objetivo:** Consumer que consume `node.executed` y `node.execute.failed`.

```go
// services/orchestrator/internal/consumer/node_result.go
type NodeResultConsumer struct {
    execRepo  ports.ExecutionRepository
    graphRepo ports.GraphRepository
    sm        *statemachine.ExecutionStateMachine
}

func (c *NodeResultConsumer) HandleNodeExecuted(ctx context.Context, evt domain.Event) error
func (c *NodeResultConsumer) HandleNodeExecuteFailed(ctx context.Context, evt domain.Event) error
```

- Carga `Execution` por `executionID` del evento.
- Carga `GraphDefinition` del campo `definition` de la entidad Graph.
- Delega en `ExecutionStateMachine`.
- Si retorna `domain.ErrRetryable` → retorna error (el adaptador Valkey hace NACK).
- Si retorna nil → el adaptador hace ACK.

**Fichero:** `services/orchestrator/internal/consumer/node_result.go`

---

### TODO #11 — impl: StartExecution extendido [impl]

**Agente:** @developer

**Objetivo:** Extender `StartExecution` (use case SPRINT-003) para validar + publicar + running.

Flujo actualizado:
1. Cargar Graph de repositorio.
2. Deserializar `definition` a `domain.GraphDefinition`.
3. `ValidateGraph(graphDef)` — si falla → retorna `ErrGraphValidation` (handler devuelve 422).
4. `CountActiveByGraph` — si > 0 → retorna `ErrConflict`.
5. `Create(execution)` con `status = "running"` (ya no `pending`).
6. Publicar `node.execute.requested` para `entry_node`.

Añadir `ErrGraphValidation` al handler HTTP como 422.

**Ficheros:** `services/orchestrator/internal/usecase/start_execution.go`,
`services/orchestrator/internal/handler/execution_handler.go`

---

### TODO #12 — impl: Wiring en main.go del orchestrator [impl]

**Agente:** @developer

**Objetivo:** Cablear los nuevos consumers y la state machine en
`services/orchestrator/main.go`.

- Construir `ExecutionStateMachine` con repo y publisher.
- Construir `NodeResultConsumer` con repos y state machine.
- Registrar handlers para `node.executed` y `node.execute.failed` en el EventConsumer.
- Arrancar los consumers en goroutines.
- Manejar shutdown graceful (context cancel).

**Fichero:** `services/orchestrator/main.go`

---

### TODO #13 — infra: go.mod — dominikbraun/graph [infra]

**Agente:** @devops

**Objetivo:** Añadir la dependencia al módulo.

```
go get github.com/dominikbraun/graph
```

Verificar que es Apache 2.0 (compatible con open source del proyecto).

**Fichero:** `go.mod`, `go.sum`

---

### TODO #14 — test: integración state machine con Valkey real [test]

**Agente:** @qa

**Nota TDD:** Este test debe escribirse en Red ANTES de los TODOs de implementación (#7–#12). Ver "Nota sobre orden TDD" al inicio de los TODOs.

**Objetivo:** Tests de integración con Valkey real (build tag `integration`).

Caso 1: Submit execution válido → estado `running`, evento `node.execute.requested`
publicado en stream correcto.

Caso 2: Consume `node.executed` (nodo terminal) → estado `completed`, evento
`graph.completed` publicado.

Usar Testcontainers para Valkey. Mínimo datos reales en PostgreSQL (puede ser SQLite
en memoria con Ent para aislar de infra completa).

**Fichero:** `services/orchestrator/internal/statemachine/integration_test.go`

---

### TODO #15 — test: integración consumer node_result [test]

**Agente:** @qa

**Nota TDD:** Este test debe escribirse en Red ANTES de los TODOs de implementación (#7–#12). Ver "Nota sobre orden TDD" al inicio de los TODOs.

**Objetivo:** Test de integración end-to-end del consumer con Valkey real.

Publicar evento `node.executed` en el stream → verificar que el consumer llama
`HandleNodeExecuted` → verificar ACK y estado actualizado.

**Fichero:** `services/orchestrator/internal/consumer/node_result_integration_test.go`

---

### TODO #16 — impl: Adaptador Ent para UpdateExecution [impl]

**Agente:** @developer

**Objetivo:** Implementar `UpdateExecution` en el adaptador Ent de orchestrator.

```go
func (r *ExecutionRepo) UpdateExecution(ctx context.Context, exec *domain.Execution) error {
    _, err := r.client.Execution.
        UpdateOneID(exec.ID).
        SetStatus(execution.Status(exec.Status)).
        SetCurrentNode(exec.CurrentNode).
        Save(ctx)
    return err
}
```

**Fichero:** `adapters/storage/ent/execution_repo.go`

---

### TODO #17 — docs: actualizar documentación [docs]

**Agente:** @docs

**Objetivo:** Actualizar artefactos de documentación al cerrar el sprint.

- `docs/index.md` — estado SPRINT-010: completado.
- `docs/log.md` — entrada de cierre.
- `docs/views/process/` — diagrama de estado de Execution (pending→running→completed/failed).
- Comentario en `execution_sm.go` explicando la limitación: solo aristas `sequential`
  (timeout por nodo excluido, documentado como TODO futuro).

**Ficheros:** `docs/index.md`, `docs/log.md`, `docs/views/process/`

---

## Matriz de trazabilidad

| TODO | Spec | Test | Impl | Docs |
|------|------|------|------|------|
| #1 AsyncAPI ops | asyncapi.yaml | — | — | — |
| #2 OpenAPI 422 | executions.yaml | — | — | — |
| #3 ValidateGraph test | — | graph_validator_test.go | — | — |
| #4 NextNode + SM tests | — | traversal_test.go, execution_sm_test.go | — | — |
| #5 ErrGraphValidation + types | — | — | domain/errors.go, domain/graph.go | — |
| #6 UpdateExecution port (impl) | — | — | ports/storage.go | — |
| #7 ValidateGraph impl | ADR-016 | #3 | graph_validator.go | — |
| #8 NextNode impl | ADR-016 | #4 | traversal.go | — |
| #9 ExecutionStateMachine | ADR-014 | #4 | execution_sm.go | — |
| #10 Consumer node_result | ADR-014 | — | consumer/node_result.go | — |
| #11 StartExecution extendido | OpenAPI | — | usecase/, handler/ | — |
| #12 Wiring main.go | — | — | orchestrator/main.go | — |
| #13 go.mod dominikbraun | — | — | go.mod | — |
| #14 Integración SM | ADR-007, ADR-008 | integration_test.go | — | — |
| #15 Integración consumer | ADR-008, ADR-014 | node_result_integration_test.go | — | — |
| #16 UpdateExecution Ent | ADR-007 | — | ent/execution_repo.go | — |
| #17 Docs | — | — | — | index.md, log.md, views/ |

## Decisiones clave

- **Solo aristas `sequential`** en este sprint. Aristas `conditional`, `parallel`, `loop`,
  `interrupt` → `ErrGraphValidation`. Se documentan como limitación conocida y TODO futuro.
- **Timeout por nodo excluido** explícitamente. El context de Go se propaga pero no se
  añade deadline por nodo; se documenta como TODO futuro.
- **`ErrRetryable`** como sentinel en `libs/domain/` permite que el consumer Valkey haga
  NACK sin importar detalles del adaptador.
- **Idempotencia obligatoria** (ADR-014): `CanTransitionTo` evita doble transición si el
  consumer recibe el mismo evento dos veces.
- **Checkpointing** (ADR-014): `UpdateExecution` se llama tras cada transición antes de
  publicar el siguiente evento, garantizando consistencia eventual.
- **`StartExecution`** pasa directamente de validar a `running` (no `pending`), simplificando
  el flujo ya que la publicación del primer evento es síncrona en este sprint.

## Resultado

> _Completar al cerrar el sprint._

- TODOs completados: —/17
- Tests pasando: —
- Decisiones revisadas: —
- Artefactos entregados: —
