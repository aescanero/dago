# SPRINT-003: API REST del orchestrator — CRUD grafos y ejecuciones

## Metadata

- **Fecha inicio:** 2026-04-29 (tras completar SPRINT-002)
- **Fecha fin estimada:** 2026-05-01
- **Estado:** planificado
- **ADRs aplicados:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-006, ADR-010, ADR-016
- **Specs afectadas:**
  - `specs/openapi.yaml` — añadir `$ref` a los nuevos paths
  - `specs/paths/graphs.yaml` — nuevo (6 endpoints de grafos)
  - `specs/paths/executions.yaml` — nuevo (1 endpoint de ejecuciones)
  - `specs/schemas/graph.yaml` — nuevo
  - `specs/schemas/execution.yaml` — nuevo
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Bloquea:** SPRINT-004 (eventos), SPRINT-dashboard-001 (frontend)
- **Bloqueado por:** SPRINT-001 (go.mod, Gin), SPRINT-002 (Ent schemas)

## Objetivo del sprint

Implementar la API REST del orchestrator para el dashboard: 5 endpoints
CRUD de grafos y 1 endpoint de inicio de ejecución, con el flujo completo
Spec → Tests → Dominio → Casos de uso → Handlers → Adaptador.

Al finalizar existe una API Gin funcional que cumple la spec OpenAPI, con
tests de contrato, unitarios de casos de uso y unitarios de handlers. El
endpoint POST /executions crea registros en PostgreSQL pero aún no publica
eventos (se añade en SPRINT-004).

## Alcance

### Incluido

**Spec OpenAPI (fuente de verdad — ADR-010 regla 6):**
- `specs/paths/graphs.yaml` con los 6 endpoints de grafos.
- `specs/paths/executions.yaml` con el endpoint de ejecución.
- `specs/schemas/graph.yaml`, `specs/schemas/execution.yaml`.
- `specs/openapi.yaml` actualizado con `$ref` a los nuevos paths.

**Dominio (`libs/domain/`):**
- `libs/domain/graph.go` — tipos `Graph`, `GraphStatus`, constantes.
- `libs/domain/execution.go` — tipos `Execution`, `ExecutionStatus`.
- `libs/domain/errors.go` — errores de dominio (`ErrNotFound`,
  `ErrConflict`, `ErrValidation`, `ErrInvalidGraphStatus`).

**Puertos (`libs/ports/`):**
- `libs/ports/storage.go` — interfaces `GraphRepository` y
  `ExecutionRepository`.

**Casos de uso (`services/orchestrator/internal/usecase/`):**
- `CreateGraph`, `GetGraph`, `ListGraphs`, `UpdateGraph`, `ArchiveGraph`.
- `StartExecution`.

**Handlers Gin (`services/orchestrator/internal/handler/`):**
- `GraphHandler` — 5 endpoints.
- `ExecutionHandler` — 1 endpoint.
- `mapDomainError()` — traduce errores de dominio a HTTP (ADR-006).

**Router (`services/orchestrator/internal/router/`):**
- `NewRouter()` — compone middlewares y registra rutas bajo `/api/v1/`.

**Adaptador de storage (`adapters/storage/`):**
- `EntGraphRepository` — implementa `GraphRepository` con Ent.
- `EntExecutionRepository` — implementa `ExecutionRepository` con Ent.

**Tests:**
- `tests/contract/graphs_contract_test.go` — valida respuestas contra
  la spec OpenAPI con `go-openapi`.
- `tests/unit/usecase/graph_usecase_test.go` — casos de uso con fakes.
- `tests/unit/usecase/execution_usecase_test.go` — idem.
- `tests/unit/handler/graph_handler_test.go` — handlers con mock
  `httptest.NewRecorder`.
- `tests/unit/handler/execution_handler_test.go` — idem.

**Entrada de servidor en `services/orchestrator/cmd/main.go`:**
- `main()` real que inicializa Ent, repositorios, casos de uso, router
  y arranca el servidor en el puerto configurado por env var.

### Excluido

- Auth middleware JWT (SPRINT-auth): los endpoints no requieren token
  en este sprint. Se añade middleware después.
- Publicación de eventos Valkey al crear Execution (SPRINT-004).
- Ejecución real del grafo (SPRINT-004 en adelante).
- WebSocket AG-UI (SPRINT-005).
- Endpoint GET /executions y GET /executions/{id} (SPRINT-004).
- Endpoint PATCH /graphs/{id}/status para transiciones de estado.
- Validación semántica de la definición del grafo contra ADR-016
  (patrón de nodos, aristas) — se implementa en SPRINT-004 cuando
  se inicie la ejecución real.
- Generación de cliente TypeScript desde OpenAPI (SPRINT-dashboard-001).

## Dependencias

- **SPRINT-001 completado:** `go.mod` con Gin, estructura de dirs,
  `services/orchestrator/cmd/main.go` stub.
- **SPRINT-002 completado:** schemas Ent de Graph, Node, Execution
  generados y migrados en PostgreSQL.

## Contratos de comportamiento

### C1 — `POST /api/v1/graphs` — creación exitosa

```
Given: Body JSON válido con name, version semver ("1.0.0"), entry_node y definition
When: POST /api/v1/graphs con Content-Type application/json
Then: HTTP 201, body con schema GraphResponse
      id es UUID v4, status = "draft"
      created_at y updated_at presentes en RFC3339
```

### C2 — `POST /api/v1/graphs` — versión inválida

```
Given: Body JSON con version = "v1" (no es semver)
When: POST /api/v1/graphs
Then: HTTP 422, body ErrorResponse con code = "VALIDATION_ERROR"
      No se persiste ninguna fila
```

### C3 — `DELETE /api/v1/graphs/:id` — archivado (no borrado físico)

```
Given: Grafo con status="draft" y sin ejecuciones activas
When: DELETE /api/v1/graphs/:id
Then: HTTP 204 sin body
      El status del grafo en BD cambia a "archived"
      La fila no se elimina físicamente (soft delete)
```

```
Given: Grafo con ejecuciones en status="running"
When: DELETE /api/v1/graphs/:id
Then: HTTP 409, ErrorResponse con code = "GRAPH_HAS_ACTIVE_EXECUTIONS"
```

## Diseño de la API

### Endpoints

| Método | Path | Handler | Status codes |
|--------|------|---------|--------------|
| `POST` | `/api/v1/graphs` | `CreateGraph` | 201, 400, 409, 422, 500 |
| `GET` | `/api/v1/graphs` | `ListGraphs` | 200, 500 |
| `GET` | `/api/v1/graphs/:id` | `GetGraph` | 200, 404, 500 |
| `PUT` | `/api/v1/graphs/:id` | `UpdateGraph` | 200, 400, 404, 409, 422, 500 |
| `DELETE` | `/api/v1/graphs/:id` | `ArchiveGraph` | 204, 404, 409, 500 |
| `POST` | `/api/v1/executions` | `StartExecution` | 201, 400, 404, 422, 500 |

### Decisiones de diseño

**DELETE archiva en lugar de borrar físicamente.** Establece
`status=archived`. Devuelve 409 si el grafo tiene ejecuciones activas
(`status=running`). Razón: preservar historial de ejecuciones
asociadas al grafo (ADR-015).

**PUT solo actualiza grafos en status `draft`.** Devuelve 409 si el
grafo está `active` o `archived`. Para activar un grafo existe un
endpoint dedicado (fuera del alcance de este sprint).

**POST /executions crea la Execution en `pending`.** No inicia
ejecución real ni publica evento (SPRINT-004). La respuesta incluye
el ID de ejecución para que el cliente pueda suscribirse vía
WebSocket (SPRINT-005).

### Schemas de la API

**GraphInput** (request de creación y actualización):
```yaml
type: object
required: [name, version, entry_node, definition]
properties:
  name:
    type: string
    maxLength: 255
  version:
    type: string
    pattern: '^\d+\.\d+\.\d+$'
  description:
    type: string
  entry_node:
    type: string
    maxLength: 255
  definition:
    type: object          # JSON libre del grafo (nodos + aristas)
  memory_config:
    type: object
    properties:
      semantic_search: {type: boolean}
      episode_context: {type: integer, minimum: 0}
```

**GraphResponse** (respuesta individual):
```yaml
type: object
required: [id, name, version, entry_node, definition, status, created_at, updated_at]
properties:
  id:       {type: string, format: uuid}
  name:     {type: string}
  version:  {type: string}
  description: {type: string}
  entry_node: {type: string}
  definition: {type: object}
  memory_config: {type: object}
  status:   {type: string, enum: [draft, active, archived]}
  created_at: {type: string, format: date-time}
  updated_at: {type: string, format: date-time}
```

**GraphListResponse** (respuesta paginada):
```yaml
type: object
required: [items, pagination]
properties:
  items:
    type: array
    items: {$ref: '#/components/schemas/GraphResponse'}
  pagination: {$ref: '#/components/schemas/Pagination'}
```

**ExecutionInput** (request de inicio):
```yaml
type: object
required: [graph_id]
properties:
  graph_id:
    type: string
    format: uuid
  variables:
    type: object     # variables iniciales del grafo
    default: {}
```

**ExecutionResponse** (respuesta):
```yaml
type: object
required: [id, graph_id, status, variables, messages, node_results, created_at, updated_at]
properties:
  id:            {type: string, format: uuid}
  graph_id:      {type: string, format: uuid}
  status:        {type: string, enum: [pending, running, completed, failed, cancelled, interrupted]}
  current_node:  {type: string}
  variables:     {type: object}
  messages:      {type: array, items: {type: object}}
  node_results:  {type: object}
  error:         {type: string}
  started_at:    {type: string, format: date-time}
  completed_at:  {type: string, format: date-time}
  created_at:    {type: string, format: date-time}
  updated_at:    {type: string, format: date-time}
```

**ErrorResponse** (ya en openapi.yaml):
```yaml
type: object
required: [code, message]
properties:
  code:    {type: string}   # GRAPH_NOT_FOUND, GRAPH_NOT_DRAFT, etc.
  message: {type: string}
  details: {type: array, items: {type: object}}
```

### Mapa de errores de dominio → HTTP

| Error de dominio | HTTP | `code` en ErrorResponse |
|-----------------|------|-------------------------|
| `ErrNotFound` | 404 | `GRAPH_NOT_FOUND`, `EXECUTION_NOT_FOUND` |
| `ErrConflict` | 409 | `GRAPH_DUPLICATE_VERSION`, `GRAPH_NOT_DRAFT`, `GRAPH_HAS_ACTIVE_EXECUTIONS` |
| `ErrValidation` | 422 | `VALIDATION_ERROR` |
| `ErrInvalidGraphStatus` | 409 | `GRAPH_NOT_FOUND_OR_ACTIVE` |
| Otro error | 500 | `INTERNAL_ERROR` |

## TODOs

### 1. [spec] Escribir `specs/schemas/graph.yaml` y `specs/schemas/execution.yaml`

- **Agente:** @developer
- **Descripción:** Crear los schemas OpenAPI reutilizables para
  `GraphInput`, `GraphResponse`, `GraphListResponse`,
  `ExecutionInput`, `ExecutionResponse`. Seguir el diseño de la
  sección anterior y el formato de error ya definido en `openapi.yaml`.

  Cada schema va en su propio fichero en `specs/schemas/` referenciado
  desde `specs/openapi.yaml` con `$ref`.

- **Criterio de aceptación:** `swagger-cli validate specs/openapi.yaml`
  (o equivalente) pasa sin errores. Cada schema tiene `required`,
  `properties` y `description` completos.
- **Depende de:** ninguno
- **Commit:** `spec(openapi): add Graph and Execution schemas [SPRINT-003 #1]`

### 2. [spec] Escribir `specs/paths/graphs.yaml` y `specs/paths/executions.yaml`

- **Agente:** @developer
- **Descripción:** Crear los path items OpenAPI para los 6 endpoints.
  Cada endpoint documenta: parámetros, request body, respuestas (todos
  los status codes), referencias a los schemas del TODO #1 y al
  `ErrorResponse` ya existente.

  Añadir en `specs/openapi.yaml` bajo `paths:`:
  ```yaml
  paths:
    /graphs:
      $ref: './paths/graphs.yaml#/graphs'
    /graphs/{id}:
      $ref: './paths/graphs.yaml#/graphsId'
    /executions:
      $ref: './paths/executions.yaml#/executions'
  ```

  Parámetros de paginación para `GET /graphs`:
  - `page` (integer, default 1)
  - `per_page` (integer, default 20, max 100)
  - `status` (string, opcional, filtro por status)

- **Criterio de aceptación:** `swagger-cli validate specs/openapi.yaml`
  pasa. Todos los endpoints tienen documentados todos sus status codes
  con ejemplos de request/response.
- **Depende de:** #1
- **Commit:** `spec(openapi): add graphs and executions path items [SPRINT-003 #2]`

### 3. [domain] Implementar tipos de dominio en `libs/domain/`

- **Agente:** @developer
- **Descripción:** Crear los tipos de dominio puros, sin dependencias
  de Ent ni Gin. Siguen ADR-001 (el dominio es el centro) y ADR-007
  regla 7 (tipos Ent no salen del adaptador).

  **`libs/domain/graph.go`:**
  ```go
  // GraphStatus representa el estado de un grafo.
  type GraphStatus string

  const (
      GraphStatusDraft    GraphStatus = "draft"
      GraphStatusActive   GraphStatus = "active"
      GraphStatusArchived GraphStatus = "archived"
  )

  // Graph es la definición de un grafo de ejecución (no una instancia).
  type Graph struct {
      ID           uuid.UUID
      Name         string
      Version      string
      Description  string
      EntryNode    string
      Definition   json.RawMessage
      MemoryConfig json.RawMessage
      Status       GraphStatus
      CreatedAt    time.Time
      UpdatedAt    time.Time
  }

  // IsDraft devuelve true si el grafo puede ser modificado.
  func (g *Graph) IsDraft() bool { return g.Status == GraphStatusDraft }

  // IsActive devuelve true si el grafo puede iniciar ejecuciones.
  func (g *Graph) IsActive() bool { return g.Status == GraphStatusActive }
  ```

  **`libs/domain/execution.go`:**
  ```go
  // ExecutionStatus representa el estado de una ejecución.
  type ExecutionStatus string

  const (
      ExecutionStatusPending     ExecutionStatus = "pending"
      ExecutionStatusRunning     ExecutionStatus = "running"
      ExecutionStatusCompleted   ExecutionStatus = "completed"
      ExecutionStatusFailed      ExecutionStatus = "failed"
      ExecutionStatusCancelled   ExecutionStatus = "cancelled"
      ExecutionStatusInterrupted ExecutionStatus = "interrupted"
  )

  // Execution es una instancia de ejecución de un grafo.
  type Execution struct {
      ID          uuid.UUID
      GraphID     uuid.UUID
      Status      ExecutionStatus
      CurrentNode string
      Variables   json.RawMessage
      Messages    json.RawMessage
      NodeResults json.RawMessage
      Error       string
      StartedAt   *time.Time
      CompletedAt *time.Time
      CreatedAt   time.Time
      UpdatedAt   time.Time
  }
  ```

  **`libs/domain/errors.go`:**
  ```go
  var (
      ErrNotFound            = errors.New("not found")
      ErrConflict            = errors.New("conflict")
      ErrValidation          = errors.New("validation error")
      ErrInvalidGraphStatus  = errors.New("invalid graph status for operation")
  )
  ```

- **Criterio de aceptación:** `go build ./libs/domain/...` compila.
  El paquete no importa nada de `entgo.io`, `gin`, `go-redis`.
- **Depende de:** ninguno (puede hacerse en paralelo con #1, #2)
- **Commit:** `feat(domain): add Graph and Execution domain types [SPRINT-003 #3]`

### 4. [domain] Implementar puertos en `libs/ports/storage.go`

- **Agente:** @developer
- **Descripción:** Definir las interfaces de repositorio que el dominio
  necesita. Los métodos usan exclusivamente tipos de `libs/domain/`,
  nunca tipos de Ent.

  ```go
  // libs/ports/storage.go

  // GraphRepository gestiona la persistencia de grafos.
  type GraphRepository interface {
      Create(ctx context.Context, g *domain.Graph) (*domain.Graph, error)
      FindByID(ctx context.Context, id uuid.UUID) (*domain.Graph, error)
      List(ctx context.Context, opts ListOptions) ([]*domain.Graph, int, error)
      Update(ctx context.Context, g *domain.Graph) (*domain.Graph, error)
      Archive(ctx context.Context, id uuid.UUID) error
  }

  // ExecutionRepository gestiona la persistencia de ejecuciones.
  type ExecutionRepository interface {
      Create(ctx context.Context, e *domain.Execution) (*domain.Execution, error)
      FindByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error)
      CountActiveByGraph(ctx context.Context, graphID uuid.UUID) (int, error)
  }

  // ListOptions configura la paginación y filtros de List.
  type ListOptions struct {
      Page    int
      PerPage int
      Status  string  // opcional
  }
  ```

- **Criterio de aceptación:** `go build ./libs/ports/...` compila.
  El paquete solo importa `libs/domain/` y stdlib.
- **Depende de:** #3
- **Commit:** `feat(ports): add GraphRepository and ExecutionRepository interfaces [SPRINT-003 #4]`

### 5. [test] Tests de contrato (Red) — `tests/contract/`

- **Agente:** @qa
- **Descripción:** Crear tests que levantan el servidor de prueba real
  (con repositorios fake en memoria) y verifican que las respuestas
  cumplen la spec OpenAPI. Usar `build tag: contract`.

  Librería sugerida: `github.com/deepmap/oapi-codegen` o una solución
  manual que valide el JSON de respuesta contra los schemas de
  `specs/schemas/`.

  Tests a implementar:

  ```go
  //go:build contract

  // TestCreateGraphContract verifica que POST /api/v1/graphs devuelve
  // 201 con un body que cumple GraphResponse.
  func TestCreateGraphContract(t *testing.T)

  // TestCreateGraphValidationContract verifica que un body inválido
  // devuelve 422 con ErrorResponse.
  func TestCreateGraphValidationContract(t *testing.T)

  // TestListGraphsContract verifica que GET /api/v1/graphs devuelve
  // 200 con GraphListResponse (items + pagination).
  func TestListGraphsContract(t *testing.T)

  // TestGetGraphContract verifica que GET /api/v1/graphs/:id devuelve
  // 200 con GraphResponse o 404 con ErrorResponse.
  func TestGetGraphContract(t *testing.T)

  // TestUpdateGraphContract verifica PUT /api/v1/graphs/:id.
  func TestUpdateGraphContract(t *testing.T)

  // TestDeleteGraphContract verifica DELETE /api/v1/graphs/:id → 204.
  func TestDeleteGraphContract(t *testing.T)

  // TestStartExecutionContract verifica POST /api/v1/executions → 201.
  func TestStartExecutionContract(t *testing.T)

  // TestStartExecutionGraphNotFoundContract verifica 404 si el grafo
  // no existe.
  func TestStartExecutionGraphNotFoundContract(t *testing.T)
  ```

- **Criterio de aceptación:** `make test-contract` (con repositorios
  fake) termina con todos PASS. Los tests fallan si la respuesta no
  incluye un campo `required` del schema.
- **Depende de:** #2, #3, #4 (para compilar los fakes)
- **Commit:** `test(contract): add contract tests for graphs and executions API [SPRINT-003 #5]`

### 6. [test] Tests unitarios de casos de uso (Red) — `tests/unit/usecase/`

- **Agente:** @qa
- **Descripción:** Tests de la lógica de negocio de los casos de uso,
  usando fakes en memoria que implementan los puertos del TODO #4.

  **`graph_usecase_test.go`:**
  ```go
  // TestCreateGraphSuccess verifica que CreateGraph persiste y devuelve
  // el grafo con status=draft.
  func TestCreateGraphSuccess(t *testing.T)

  // TestCreateGraphDuplicateVersion verifica que crear (name, version)
  // duplicado devuelve ErrConflict.
  func TestCreateGraphDuplicateVersion(t *testing.T)

  // TestUpdateGraphOnlyDraft verifica que actualizar un grafo active
  // devuelve ErrInvalidGraphStatus.
  func TestUpdateGraphOnlyDraft(t *testing.T)

  // TestArchiveGraphWithActiveExecution verifica que archivar un grafo
  // con ejecuciones activas devuelve ErrConflict.
  func TestArchiveGraphWithActiveExecution(t *testing.T)
  ```

  **`execution_usecase_test.go`:**
  ```go
  // TestStartExecutionSuccess verifica que StartExecution crea
  // Execution con status=pending y variables vacías por defecto.
  func TestStartExecutionSuccess(t *testing.T)

  // TestStartExecutionGraphNotFound verifica que pasar un graph_id
  // inexistente devuelve ErrNotFound.
  func TestStartExecutionGraphNotFound(t *testing.T)

  // TestStartExecutionWithInitialVariables verifica que las variables
  // iniciales se persisten en la Execution.
  func TestStartExecutionWithInitialVariables(t *testing.T)
  ```

- **Criterio de aceptación:** `go test ./tests/unit/usecase/...` en
  RED (falla porque los casos de uso no existen aún). Al implementar
  los casos de uso en TODO #9, pasan a GREEN.
- **Depende de:** #4
- **Commit:** `test(unit): add use case unit tests for Graph and Execution [SPRINT-003 #6]`

### 7. [test] Tests unitarios de handlers (Red) — `tests/unit/handler/`

- **Agente:** @qa
- **Descripción:** Tests de la capa HTTP con `httptest.NewRecorder`.
  Los casos de uso se mockean con interfaces.

  **`graph_handler_test.go`:**
  ```go
  // TestGraphHandlerCreate verifica que el handler bindea correctamente
  // el JSON, llama al caso de uso y devuelve 201.
  func TestGraphHandlerCreate(t *testing.T)

  // TestGraphHandlerCreateInvalidBody verifica que JSON inválido
  // devuelve 400.
  func TestGraphHandlerCreateInvalidBody(t *testing.T)

  // TestGraphHandlerGetNotFound verifica que ErrNotFound del caso de
  // uso se traduce a 404 con ErrorResponse.
  func TestGraphHandlerGetNotFound(t *testing.T)

  // TestGraphHandlerDelete verifica que DELETE devuelve 204.
  func TestGraphHandlerDelete(t *testing.T)
  ```

  **`execution_handler_test.go`:**
  ```go
  // TestExecutionHandlerStart verifica que POST /executions devuelve 201.
  func TestExecutionHandlerStart(t *testing.T)

  // TestExecutionHandlerStartGraphNotFound verifica 404.
  func TestExecutionHandlerStartGraphNotFound(t *testing.T)
  ```

  `mapDomainError()` se testea indirectamente a través de estos tests.

- **Criterio de aceptación:** `go test ./tests/unit/handler/...` en
  RED. Al implementar los handlers en TODO #10, pasan a GREEN.
- **Depende de:** #3, #4
- **Commit:** `test(unit): add handler unit tests for Graph and Execution [SPRINT-003 #7]`

### 8. [impl] Implementar fakes en memoria para tests

- **Agente:** @developer
- **Descripción:** Crear implementaciones in-memory de los puertos
  que usan los tests de los TODOs #5, #6, #7. Viven en
  `tests/testutil/fakes/`.

  ```go
  // tests/testutil/fakes/graph_repo.go
  type InMemoryGraphRepository struct { ... }
  // Implementa ports.GraphRepository con un map[uuid.UUID]*domain.Graph.

  // tests/testutil/fakes/execution_repo.go
  type InMemoryExecutionRepository struct { ... }
  // Implementa ports.ExecutionRepository con un map[uuid.UUID]*domain.Execution.
  ```

  Estos fakes no tienen concurrencia ni persistencia — solo para tests.

- **Criterio de aceptación:** `go build ./tests/testutil/fakes/...`
  compila. Los fakes implementan completamente las interfaces de los
  puertos (verificado por el compilador).
- **Depende de:** #4
- **Commit:** `test(fakes): add in-memory fakes for GraphRepository and ExecutionRepository [SPRINT-003 #8]`

### 9. [impl] Implementar casos de uso en `services/orchestrator/internal/usecase/`

- **Agente:** @developer
- **Descripción:** Implementar los 6 casos de uso. Cada caso de uso
  recibe sus dependencias (repositorios) vía constructor (inyección
  de dependencias). Siguen ADR-003 (funciones ≤20 líneas) y ADR-006
  regla 1 (sin lógica de negocio fuera del dominio).

  Estructura:
  ```
  services/orchestrator/internal/usecase/
  ├── graph.go          # CreateGraph, GetGraph, ListGraphs, UpdateGraph, ArchiveGraph
  └── execution.go      # StartExecution
  ```

  Contrato de `CreateGraph`:
  1. Asignar UUID nuevo y timestamps.
  2. Asignar `status = draft`.
  3. Llamar `repo.Create(ctx, graph)`.
  4. Si el repositorio devuelve error de unicidad → `ErrConflict`.
  5. Devolver el grafo persistido.

  Contrato de `ArchiveGraph`:
  1. Cargar el grafo. Si no existe → `ErrNotFound`.
  2. Contar ejecuciones activas. Si > 0 → `ErrConflict`.
  3. Llamar `repo.Archive(ctx, id)`.

  Contrato de `UpdateGraph`:
  1. Cargar el grafo. Si no existe → `ErrNotFound`.
  2. Si no es draft → `ErrInvalidGraphStatus`.
  3. Actualizar campos. Llamar `repo.Update(ctx, graph)`.

  Contrato de `StartExecution`:
  1. Cargar el grafo. Si no existe → `ErrNotFound`.
  2. Crear `Execution` con `status=pending`, `variables` del input,
     `messages=[]`, `node_results={}`.
  3. Llamar `executionRepo.Create(ctx, execution)`.

- **Criterio de aceptación:** `go test ./tests/unit/usecase/...` pasa
  a GREEN. `make lint` sin errores. Ninguna función > 20 líneas.
- **Depende de:** #6, #8
- **Commit:** `feat(orchestrator): implement graph and execution use cases [SPRINT-003 #9]`

### 10. [impl] Implementar handlers Gin en `services/orchestrator/internal/handler/`

- **Agente:** @developer
- **Descripción:** Implementar los handlers siguiendo ADR-006.

  Estructura:
  ```
  services/orchestrator/internal/handler/
  ├── graph.go       # GraphHandler con 5 métodos
  ├── execution.go   # ExecutionHandler con 1 método
  └── errors.go      # mapDomainError()
  ```

  Patrón de cada handler (ADR-006 regla 1 y 2):
  ```go
  func (h *GraphHandler) Create(c *gin.Context) {
      var input GraphInput
      if err := c.ShouldBindJSON(&input); err != nil {
          c.JSON(http.StatusBadRequest, errorResponse("INVALID_JSON", err.Error()))
          return
      }
      graph, err := h.createGraph.Execute(c.Request.Context(), input.toDomain())
      if err != nil {
          mapDomainError(c, err)
          return
      }
      c.JSON(http.StatusCreated, graphResponseFrom(graph))
  }
  ```

  `mapDomainError()` implementa la tabla de errores del diseño.

  Las funciones `toDomain()` y `graphResponseFrom()` son helpers de
  traducción. Cuentan para el límite de 20 líneas de las funciones
  (ADR-003).

- **Criterio de aceptación:** `go test ./tests/unit/handler/...` pasa
  a GREEN. `go test ./tests/contract/...` pasa a GREEN.
  `make lint` sin errores.
- **Depende de:** #7, #9
- **Commit:** `feat(orchestrator): implement Gin handlers for graphs and executions [SPRINT-003 #10]`

### 11. [impl] Implementar adaptador de storage con Ent

- **Agente:** @developer
- **Descripción:** Implementar `GraphRepository` y `ExecutionRepository`
  usando el cliente Ent generado en SPRINT-002. Viven en
  `adapters/storage/`.

  **Reglas de ADR-001 y ADR-007:**
  - Los tipos Ent solo existen dentro del adaptador.
  - El adaptador traduce `ent.Graph` → `domain.Graph` y viceversa.
  - Usar `client.Tx(ctx)` + defer rollback para operaciones de escritura.
  - Wrappear errores con `fmt.Errorf("storage.CreateGraph: %w", err)`.

  **`adapters/storage/graph_repo.go`:**
  ```go
  type EntGraphRepository struct {
      client *ent.Client
  }

  func (r *EntGraphRepository) Create(ctx context.Context, g *domain.Graph) (*domain.Graph, error) {
      created, err := r.client.Graph.Create().
          SetID(g.ID).
          SetName(g.Name).
          // ...
          Save(ctx)
      if err != nil {
          if ent.IsConstraintError(err) {
              return nil, fmt.Errorf("%w: %w", domain.ErrConflict, err)
          }
          return nil, fmt.Errorf("storage.CreateGraph: %w", err)
      }
      return entGraphToDomain(created), nil
  }
  ```

  La función `entGraphToDomain()` traduce `*ent.Graph` → `*domain.Graph`.
  Análoga para `*ent.Execution`.

  `List()` implementa paginación con `.Offset()` y `.Limit()`. Si
  `opts.Status` no está vacío, añade `.Where(graph.StatusEQ(...))`.

- **Criterio de aceptación:** `make test-integration` (con PostgreSQL)
  pasa. Los tests de contrato con el adaptador real (en lugar del fake)
  pasan. `make lint` sin errores.
- **Depende de:** #4, #9
- **Commit:** `feat(storage): implement EntGraphRepository and EntExecutionRepository [SPRINT-003 #11]`

### 12. [impl] Montar el router y el servidor en `services/orchestrator/`

- **Agente:** @developer
- **Descripción:** Crear el router Gin y el `main.go` real.

  **`services/orchestrator/internal/router/router.go`:**
  ```go
  // NewRouter construye el router Gin con middlewares y rutas.
  func NewRouter(graphHandler *handler.GraphHandler, executionHandler *handler.ExecutionHandler) *gin.Engine {
      r := gin.New()
      r.Use(gin.Recovery())
      r.Use(middlewareLogger())
      r.Use(middlewareRequestID())

      v1 := r.Group("/api/v1")
      {
          g := v1.Group("/graphs")
          g.POST("", graphHandler.Create)
          g.GET("", graphHandler.List)
          g.GET("/:id", graphHandler.GetByID)
          g.PUT("/:id", graphHandler.Update)
          g.DELETE("/:id", graphHandler.Archive)

          e := v1.Group("/executions")
          e.POST("", executionHandler.Start)
      }
      return r
  }
  ```

  **`services/orchestrator/cmd/main.go`** — inicialización completa:
  1. Conectar a PostgreSQL (DSN desde env var `DATABASE_URL`).
  2. Construir `ent.Client`, ejecutar `client.Schema.Create(ctx)`.
  3. Construir `EntGraphRepository` y `EntExecutionRepository`.
  4. Construir casos de uso.
  5. Construir handlers.
  6. Construir router.
  7. Iniciar servidor en `PORT` (default 8080).

  Manejo de señales de cierre (`os.Signal`, `context.WithTimeout`).

- **Criterio de aceptación:** `go build ./services/orchestrator/...`
  compila. El binario arranca y responde 200 a
  `GET /api/v1/graphs?page=1&per_page=10` con un array vacío.
  `make lint` sin errores.
- **Depende de:** #10, #11
- **Commit:** `feat(orchestrator): wire up router and server in main.go [SPRINT-003 #12]`

### 13. [test] Smoke test de la API arrancada

- **Agente:** @qa
- **Descripción:** Añadir un test de smoke para la API real con build
  tag `smoke`. Levanta el servidor en un puerto libre con
  `httptest.NewServer`, hace un ciclo CRUD completo y verifica
  los status codes esperados.

  ```go
  //go:build smoke

  // TestAPICRUDSmoke verifica el ciclo completo:
  // POST → GET list → GET by id → PUT → DELETE → GET 404.
  func TestAPICRUDSmoke(t *testing.T)

  // TestAPIStartExecutionSmoke verifica POST /executions con un
  // grafo creado previamente.
  func TestAPIStartExecutionSmoke(t *testing.T)
  ```

- **Criterio de aceptación:** `make test-smoke` pasa incluyendo estos
  dos nuevos tests.
- **Depende de:** #12
- **Commit:** `test(smoke): add API CRUD smoke tests for orchestrator [SPRINT-003 #13]`

### 14. [docs] Actualizar docs/index.md y docs/log.md

- **Agente:** @docs
- **Descripción:** Actualizar la sección de specs en `docs/index.md`
  indicando que los paths de grafos y ejecuciones están definidos.
  Añadir SPRINT-003 a la tabla de sprints. Actualizar `docs/log.md`.
- **Criterio de aceptación:** `docs/index.md` y `docs/log.md`
  reflejan el estado tras el sprint.
- **Depende de:** #13
- **Commit:** `docs(api): update index with SPRINT-003 results [SPRINT-003 #14]`

## Matriz de trazabilidad

| Spec / ADR | Regla | TODO | Artefacto | Verificado por |
|------------|-------|------|-----------|----------------|
| ADR-010 regla 6 | Spec-first, no code-first | #1, #2 | `specs/paths/`, `specs/schemas/` | antes de cualquier código |
| ADR-010 regla 1 | Prefijo `/api/v1/` obligatorio | #12 | `router.go` | tests de contrato |
| ADR-010 regla 8 | Formato `ErrorResponse` estándar | #10 | `handler/errors.go` | tests unitarios handler |
| ADR-006 regla 1 | Handlers delgados | #10 | cada handler ≤20 líneas | `make lint` (funlen) |
| ADR-006 regla 2 | `c.Request.Context()` al dominio | #10 | handlers | code review |
| ADR-006 regla 3 | `mapDomainError()` centralizado | #10 | `handler/errors.go` | tests handler |
| ADR-006 regla 4 | Rutas bajo `/api/v1/` | #12 | `router.go` | tests de contrato |
| ADR-001 regla 1 | Dominio sin infraestructura | #3 | `libs/domain/` sin imports Gin/Ent | `go build` + revisión |
| ADR-001 regla 2 | Puertos como interfaces Go | #4 | `libs/ports/storage.go` | compilador |
| ADR-001 regla 4 | Tipos Ent no salen del adaptador | #11 | `adapters/storage/` | code review |
| ADR-001 regla 5 | DI en main.go | #12 | `cmd/main.go` | compilación |
| ADR-016 | 7 patrones de nodo (enum) | #3 | `libs/domain/` constantes | tests de dominio |
| ADR-002 | TDD Red → Green | #5, #6, #7 antes de #9, #10 | orden de TODOs | CI |
| ADR-003 | Funciones ≤20 líneas | #9, #10, #11 | todos los ficheros impl | `make lint` (funlen) |
| `specs/openapi.yaml` | DELETE archiva, no borra | #9 | `ArchiveGraph` use case | `TestArchiveGraph*` |
| `specs/openapi.yaml` | PUT solo grafos draft | #9 | `UpdateGraph` use case | `TestUpdateGraphOnlyDraft` |

## Criterios de aceptación del sprint

```bash
# 1. La spec OpenAPI es válida
swagger-cli validate specs/openapi.yaml

# 2. Todo el código compila
go build ./...

# 3. El linter pasa
make lint

# 4. Tests unitarios (sin Docker)
go test ./tests/unit/...
go test ./tests/testutil/...

# 5. Tests de contrato (sin Docker, con fakes)
go test -tags=contract ./tests/contract/...

# 6. Tests de integración (con Docker)
make test-integration

# 7. Tests de smoke (incluye ciclo CRUD API)
make test-smoke

# 8. Pipeline CI completo
make ci

# 9. El servidor arranca y responde
go run ./services/orchestrator/cmd/main.go &
curl -s http://localhost:8080/api/v1/graphs | jq .
# → {"items":[],"pagination":{"page":1,"per_page":20,"total":0,"total_pages":0}}
```

Adicionalmente:
- `libs/domain/` no importa `entgo.io`, `gin-gonic` ni `go-redis`.
- `libs/ports/` solo importa `libs/domain/` y stdlib.
- Todos los handlers tienen ≤20 líneas (verificado por funlen).
- `mapDomainError()` cubre todos los errores de dominio (sin `switch`
  con default que silencia errores desconocidos).
- DELETE devuelve 204 sin body.
- POST /graphs devuelve 201 con body.
- POST /executions devuelve 201 con body.

## Resultado del sprint

_Se completa al finalizar el sprint._

### Tests ejecutados

- Total: —
- Passed: —
- Failed: —

### Ficheros creados/modificados

_Lista generada al cierre._

### Decisiones tomadas durante el sprint

_Cualquier decisión no prevista que requiera un ADR o nota se documenta aquí._

### Observaciones del reviewer

_Pendiente de revisión._
