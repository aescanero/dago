# SPRINT-002: Schemas Ent — Graph, Node y Execution

## Metadata

- **Fecha inicio:** 2026-04-28 (tras completar SPRINT-001)
- **Fecha fin estimada:** 2026-04-29
- **Estado:** planificado
- **ADRs aplicados:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-007, ADR-015, ADR-016
- **Specs afectadas:**
  - `specs/patterns/graph.json` (lectura — campo `Definition`)
  - `ent/schema/` (fuente de verdad de datos — ADR-007 regla 1)
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Bloqueante de:** SPRINT-003 (orchestrator), SPRINT-015 (memoria)

## Objetivo del sprint

Implementar los tres schemas Ent base del modelo de orquestación —
`Graph`, `Node` y `Execution` — con sus relaciones, campos y
validaciones. Generar el código Ent con `go generate`, producir la
primera migración con Atlas y verificar que se aplica limpiamente
contra el PostgreSQL del docker-compose.

Al finalizar existe un modelo de datos compilable y migrado, con tests
unitarios de los schemas y un test de integración que confirma
persistencia real en PostgreSQL.

## Alcance

### Incluido

- `ent/schema/graph.go` — Schema Ent de la entidad `Graph`.
- `ent/schema/node.go` — Schema Ent de la entidad `Node`.
- `ent/schema/execution.go` — Schema Ent de la entidad `Execution`.
- `go generate ./ent` — código generado commiteado (ADR-007 regla 2).
- `atlas migrate diff init_graph_node_execution` — primera migración SQL.
- `atlas migrate apply --env local` — migración aplicada contra PostgreSQL.
- Tests unitarios de schemas (`tests/unit/schema/`) con testify.
- Test de integración (`tests/integration/`) que crea y recupera los
  tres tipos de entidades contra PostgreSQL real.
- Actualización de `docs/index.md` con la tabla de schemas Ent.
- Entrada en `docs/log.md`.

### Excluido

- `ent/schema/user.go`, `org_unit.go` — se implementan en SPRINT-auth.
- `ent/schema/episode.go`, `semantic_fact.go` — se implementan en
  SPRINT-015 (memoria).
- `ent/schema/package.go`, `mcp_server.go`, `agent_card.go` — sprints
  de soporte respectivos.
- Puertos de repositorio (`libs/ports/`) — se implementan cuando el
  primer servicio los necesite (SPRINT-003).
- Adaptador de storage (`adapters/storage/`) — mismo criterio.
- Lógica de validación de grafos (ADR-016) — SPRINT-003.
- pgvector — solo se necesita para `SemanticFact` (SPRINT-015).

## Dependencias

- **SPRINT-001 completado:** necesita `go.mod` con `entgo.io/ent`,
  directorio `ent/schema/`, `atlas.hcl`, docker-compose con PostgreSQL.
- **Specs de referencia:** `specs/patterns/graph.json` y ADR-016 para
  los campos de `Graph` y `Node`. ADR-015 para `Execution`.

## Contratos de comportamiento

### C1 — Validación de version semver en Graph

```
Given: Un Graph con campo version = "abc" (no semver)
When: ent.Client.Graph.Create().SetVersion("abc").Save(ctx)
Then: El ORM retorna error de validación
      El error referencia el campo "version"
      No se inserta ninguna fila en la tabla graphs
```

### C2 — Status por defecto de Execution

```
Given: Una Execution creada sin especificar status explícitamente
When: ent.Client.Execution.Create().SetGraphID(...).Save(ctx)
Then: El campo status del registro persistido es "pending"
      La fila se inserta correctamente
```

### C3 — Migración Atlas produce tablas correctas

```
Given: PostgreSQL disponible y archivo de migración generado con `atlas migrate diff`
When: Se ejecuta `atlas migrate apply --env local`
Then: Las tablas graphs, nodes, executions existen en PostgreSQL
      La FK nodes.graph_id → graphs.id está activa
      La FK executions.graph_id → graphs.id está activa
      `atlas_schema_revisions` no contiene errores
```

## Modelo de datos

### Entidad `Graph`

Representa la **definición** de un grafo de ejecución (la plantilla,
no una ejecución concreta).

| Campo | Tipo Ent | Tipo PostgreSQL | Notas |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | Generado con `uuid.New()` |
| `name` | `string` | `VARCHAR(255) NOT NULL` | Nombre descriptivo |
| `version` | `string` | `VARCHAR(20) NOT NULL` | Semver: `^\d+\.\d+\.\d+$` |
| `description` | `string` (optional) | `TEXT` | Descripción libre |
| `entry_node` | `string` | `VARCHAR(255) NOT NULL` | Key del nodo de entrada |
| `definition` | `json.RawMessage` | `JSONB NOT NULL` | JSON completo del grafo (nodos + aristas) |
| `memory_config` | `json.RawMessage` (optional) | `JSONB` | `semantic_search`, `episode_context` |
| `status` | `enum` | `VARCHAR(20) NOT NULL DEFAULT 'draft'` | `draft \| active \| archived` |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Inmutable, auto |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Auto en cada update |

Edges Ent:
- `Nodes []Node` — one Graph tiene many Nodes (`O2M`)
- `Executions []Execution` — one Graph tiene many Executions (`O2M`)

Índices:
- `(name, version)` UNIQUE — una versión de un grafo es única.
- `status` — para filtrar grafos activos.

### Entidad `Node`

Representa un **nodo individual** dentro de un grafo. Se almacena
como entidad separada para permitir consultas y métricas por nodo.

| Campo | Tipo Ent | Tipo PostgreSQL | Notas |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | Generado con `uuid.New()` |
| `graph_id` | `uuid.UUID` | `UUID FK → graphs.id` | FK obligatoria |
| `node_key` | `string` | `VARCHAR(255) NOT NULL` | Key en el JSON del grafo (ej: `"classifier"`) |
| `pattern` | `enum` | `VARCHAR(50) NOT NULL` | `llm_call \| tool_use \| react \| reflection \| router \| guardrail \| subgraph` |
| `config` | `json.RawMessage` | `JSONB NOT NULL` | Configuración específica del patrón |
| `position` | `json.RawMessage` (optional) | `JSONB` | `{"x":0,"y":0}` para editor visual |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Inmutable, auto |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Auto en cada update |

Edges Ent:
- `Graph *Graph` — many Nodes pertenecen a one Graph (`M2O`)

Índices:
- `(graph_id, node_key)` UNIQUE — un key de nodo es único por grafo.
- `(graph_id, pattern)` — para filtrar nodos por patrón.

### Entidad `Execution`

Representa una **instancia de ejecución** de un grafo. Es la capa
Working Memory de ADR-015: estado vivo de la ejecución actual.

| Campo | Tipo Ent | Tipo PostgreSQL | Notas |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | Generado con `uuid.New()` |
| `graph_id` | `uuid.UUID` | `UUID FK → graphs.id` | FK obligatoria |
| `status` | `enum` | `VARCHAR(20) NOT NULL DEFAULT 'pending'` | `pending \| running \| completed \| failed \| cancelled \| interrupted` |
| `current_node` | `string` (optional) | `VARCHAR(255)` | Nodo en ejecución activa |
| `variables` | `json.RawMessage` | `JSONB NOT NULL DEFAULT '{}'` | Variables acumuladas del grafo |
| `messages` | `json.RawMessage` | `JSONB NOT NULL DEFAULT '[]'` | Historial conversacional |
| `node_results` | `json.RawMessage` | `JSONB NOT NULL DEFAULT '{}'` | Resultados por nodo |
| `error` | `string` (optional) | `TEXT` | Mensaje de error si `status=failed` |
| `started_at` | `time.Time` (optional) | `TIMESTAMPTZ` | Cuando pasó a `running` |
| `completed_at` | `time.Time` (optional) | `TIMESTAMPTZ` | Cuando terminó (éxito o fallo) |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Inmutable, auto |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Auto en cada update |

Edges Ent:
- `Graph *Graph` — many Executions pertenecen a one Graph (`M2O`)

Índices:
- `(graph_id, status)` — para consultar ejecuciones activas de un grafo.
- `status` — para el orchestrator al recuperar ejecuciones pendientes.
- `created_at` — para paginación cronológica.

## TODOs

### 1. [spec] Revisar graph.json y derivar campos finales

- **Agente:** @developer
- **Descripción:** Leer `specs/patterns/graph.json` y los patrones de
  nodo en `specs/patterns/nodes/` para confirmar que los campos del
  modelo de datos reflejan correctamente la spec. Cualquier discrepancia
  entre spec y plan se resuelve a favor de la spec (ADR-007 regla 1).
  Documentar decisiones en la sección de resultados de este sprint.
- **Criterio de aceptación:** Los campos de las tablas anteriores están
  alineados con `specs/patterns/graph.json`. Sin discrepancias
  no documentadas.
- **Depende de:** ninguno
- **Commit:** `docs(schema): verify graph.json alignment with data model [SPRINT-002 #1]`

### 2. [data] Implementar `ent/schema/graph.go`

- **Agente:** @developer
- **Descripción:** Crear el schema Ent para `Graph` según el modelo
  de datos de este documento. Aplicar las convenciones de ADR-007:
  UUIDs, TIMESTAMPTZ, inmutabilidad de `created_at`, validaciones
  Ent para `version` (regex semver) y `status` (enum).

  Estructura del archivo:

  ```go
  // Package schema contains Ent schema definitions.
  package schema

  import (
      "regexp"
      "time"

      "entgo.io/ent"
      "entgo.io/ent/schema/edge"
      "entgo.io/ent/schema/field"
      "entgo.io/ent/schema/index"
      "github.com/google/uuid"
  )

  // Graph holds the schema definition for the Graph entity.
  type Graph struct {
      ent.Schema
  }

  // Fields of the Graph.
  func (Graph) Fields() []ent.Field { ... }

  // Edges of the Graph.
  func (Graph) Edges() []ent.Edge { ... }

  // Indexes of the Graph.
  func (Graph) Indexes() []ent.Index { ... }
  ```

  Validaciones a incluir:
  - `version`: `field.String("version").Match(regexp.MustCompile(...))`
  - `status`: `field.Enum("status").Values("draft", "active", "archived").Default("draft")`
  - `definition`: `field.JSON("definition", json.RawMessage{}).SchemaType(map[string]string{"postgres": "jsonb"})`
  - `created_at`: `field.Time("created_at").Default(time.Now).Immutable()`
  - `updated_at`: `field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now)`

- **Criterio de aceptación:** `go build ./ent/...` compila sin errores.
  Schema tiene todos los campos, edges e índices del modelo de datos.
- **Depende de:** #1
- **Commit:** `feat(schema): add Graph Ent schema [SPRINT-002 #2]`

### 3. [data] Implementar `ent/schema/node.go`

- **Agente:** @developer
- **Descripción:** Crear el schema Ent para `Node`. El campo `pattern`
  es un enum con los 7 valores de ADR-016. La relación `M2O` con `Graph`
  se define con `edge.From("graph", Graph.Type).Ref("nodes").Unique().Required()`.

  Validaciones a incluir:
  - `pattern`: enum con exactamente 7 valores:
    `llm_call`, `tool_use`, `react`, `reflection`, `router`,
    `guardrail`, `subgraph`.
  - `config`: JSONB not null.
  - `(graph_id, node_key)` UNIQUE via index.

- **Criterio de aceptación:** `go build ./ent/...` compila. El schema
  tiene el índice único `(graph_id, node_key)` declarado.
- **Depende de:** #2
- **Commit:** `feat(schema): add Node Ent schema [SPRINT-002 #3]`

### 4. [data] Implementar `ent/schema/execution.go`

- **Agente:** @developer
- **Descripción:** Crear el schema Ent para `Execution`. El campo
  `status` es un enum con 6 valores (ADR-015). `variables`, `messages`
  y `node_results` son JSONB con valores por defecto vacíos. Los campos
  `started_at` y `completed_at` son opcionales (nillable).

  Validaciones:
  - `status`: enum `pending | running | completed | failed | cancelled | interrupted`. Default `pending`.
  - `variables`: `field.JSON("variables", json.RawMessage{}).Default(json.RawMessage("{}"))`.
  - `messages`: `field.JSON("messages", json.RawMessage{}).Default(json.RawMessage("[]"))`.
  - `node_results`: `field.JSON("node_results", json.RawMessage{}).Default(json.RawMessage("{}"))`.
  - `started_at`, `completed_at`: `.Optional().Nillable()`.

- **Criterio de aceptación:** `go build ./ent/...` compila. Todos los
  campos opcionales son nillable. Índices `(graph_id, status)` y
  `status` declarados.
- **Depende de:** #2
- **Commit:** `feat(schema): add Execution Ent schema [SPRINT-002 #4]`

### 5. [data] Ejecutar `go generate ./ent` y commitear código generado

- **Agente:** @developer
- **Descripción:** Ejecutar `go generate ./ent` desde la raíz del
  proyecto. Este comando genera el cliente Ent tipado a partir de los
  tres schemas. El código generado se commitea (ADR-007 regla 2).

  Verificar que se generan los archivos:
  ```
  ent/client.go
  ent/ent.go
  ent/generate.go
  ent/graph.go, ent/graph_create.go, ent/graph_update.go, ent/graph_query.go, ent/graph_delete.go
  ent/node.go, ent/node_create.go, ent/node_update.go, ent/node_query.go, ent/node_delete.go
  ent/execution.go, ent/execution_create.go, ent/execution_update.go, ent/execution_query.go, ent/execution_delete.go
  ent/schema/ (los tres schemas manuales)
  ent/predicate/
  ent/internal/
  ent/enttest/
  ent/hook/
  ent/migrate/
  ```

  **Nota:** `ent/generate.go` debe existir en el directorio `ent/`
  con el contenido `//go:generate go run entgo.io/ent/cmd/ent generate ./schema`.

- **Criterio de aceptación:** `go build ./...` compila después de
  `go generate`. `go vet ./...` sin errores. El directorio `ent/`
  tiene todos los archivos generados.
- **Depende de:** #3, #4
- **Commit:** `feat(schema): generate Ent client for Graph, Node, Execution [SPRINT-002 #5]`

### 6. [test] Tests unitarios de schemas (Red → Green)

- **Agente:** @qa
- **Descripción:** Crear `tests/unit/schema/graph_test.go`,
  `node_test.go` y `execution_test.go` siguiendo TDD (ADR-002).
  Usar `ent/enttest` con SQLite en memoria para tests sin PostgreSQL.

  Tests a implementar:

  **graph_test.go:**
  ```go
  // TestGraphCreate verifica que se puede crear un Graph válido.
  func TestGraphCreate(t *testing.T)

  // TestGraphVersionValidation verifica que una versión no semver
  // es rechazada por el schema.
  func TestGraphVersionValidation(t *testing.T)

  // TestGraphStatusDefault verifica que el status por defecto es "draft".
  func TestGraphStatusDefault(t *testing.T)

  // TestGraphUniqueNameVersion verifica que (name, version) es único.
  func TestGraphUniqueNameVersion(t *testing.T)
  ```

  **node_test.go:**
  ```go
  // TestNodeCreate verifica que se puede crear un Node válido.
  func TestNodeCreate(t *testing.T)

  // TestNodePatternValidation verifica que un patrón inválido
  // es rechazado.
  func TestNodePatternValidation(t *testing.T)

  // TestNodeUniqueKeyPerGraph verifica que (graph_id, node_key) es único.
  func TestNodeUniqueKeyPerGraph(t *testing.T)

  // TestNodeBelongsToGraph verifica la relación M2O con Graph.
  func TestNodeBelongsToGraph(t *testing.T)
  ```

  **execution_test.go:**
  ```go
  // TestExecutionCreate verifica que se puede crear una Execution válida.
  func TestExecutionCreate(t *testing.T)

  // TestExecutionStatusDefault verifica que el status por defecto
  // es "pending".
  func TestExecutionStatusDefault(t *testing.T)

  // TestExecutionJSONDefaults verifica que variables, messages y
  // node_results tienen valores por defecto válidos.
  func TestExecutionJSONDefaults(t *testing.T)

  // TestExecutionOptionalFields verifica que started_at y
  // completed_at son opcionales.
  func TestExecutionOptionalFields(t *testing.T)
  ```

  Usar `enttest.Open(t, "sqlite3", "file:ent?mode=memory&...")`.
  Añadir `github.com/mattn/go-sqlite3` al `go.mod` solo para tests
  (tag `sqlite3`).

- **Criterio de aceptación:** `go test ./tests/unit/schema/...` pasa
  con todos los tests en PASS. Los tests de validación fallan si se
  elimina la validación del schema (verificación activa).
- **Depende de:** #5
- **Commit:** `test(schema): add unit tests for Graph, Node, Execution schemas [SPRINT-002 #6]`

### 7. [infra] Generar migración con Atlas

- **Agente:** @developer
- **Descripción:** Con docker-compose levantado (`make docker-up`),
  ejecutar Atlas para generar la primera migración SQL:

  ```bash
  atlas migrate diff init_graph_node_execution \
      --dir "file://migrations" \
      --to "ent://ent/schema" \
      --dev-url "docker://postgres/16/dev?search_path=public"
  ```

  O bien, usando `atlas.hcl`:

  ```bash
  atlas migrate diff init_graph_node_execution --env local
  ```

  Revisar el SQL generado en `migrations/` antes de commitear.
  El SQL debe incluir:
  - `CREATE TABLE graphs (...)` con todos los campos y constraints.
  - `CREATE TABLE nodes (...)` con FK a `graphs`.
  - `CREATE TABLE executions (...)` con FK a `graphs`.
  - Índices declarados en los schemas.
  - Constraints de enum para `status` y `pattern`.

  Aplicar linting de Atlas:
  ```bash
  atlas migrate lint --env local
  ```
  No debe reportar cambios destructivos ni bloqueos.

- **Criterio de aceptación:** Existe `migrations/YYYYMMDDHHMMSS_init_graph_node_execution.sql`
  con el SQL correcto. `atlas migrate lint` termina sin errores.
  El SQL es legible y refleja exactamente los schemas.
- **Depende de:** #5, docker-compose levantado
- **Commit:** `chore(migration): add initial migration for Graph, Node, Execution [SPRINT-002 #7]`

### 8. [infra] Aplicar migración y verificar contra PostgreSQL

- **Agente:** @developer
- **Descripción:** Aplicar la migración al PostgreSQL del docker-compose
  y verificar el resultado:

  ```bash
  atlas migrate apply --env local
  ```

  Después verificar que las tablas existen con la estructura correcta:

  ```bash
  docker compose exec postgres psql -U dago -d dago \
      -c "\dt" \
      -c "\d graphs" \
      -c "\d nodes" \
      -c "\d executions"
  ```

  Verificar específicamente:
  - Las tres tablas existen.
  - Los campos UUID son de tipo `uuid`.
  - Los campos JSON/JSONB son de tipo `jsonb`.
  - Los campos de tiempo son `timestamptz`.
  - Las FKs de `nodes.graph_id` y `executions.graph_id` apuntan a `graphs.id`.
  - Los índices únicos existen.

- **Criterio de aceptación:** `atlas migrate apply` termina con
  `Applied N migration(s)`. Las tres tablas existen en PostgreSQL con
  la estructura correcta verificada por los comandos `\d`.
- **Depende de:** #7
- **Commit:** no aplica (la migración ya fue commiteada en #7)

### 9. [test] Test de integración contra PostgreSQL real

- **Agente:** @qa
- **Descripción:** Crear `tests/integration/schema_integration_test.go`
  con build tag `integration`. El test se conecta al PostgreSQL del
  docker-compose y verifica persistencia real.

  ```go
  //go:build integration

  package integration_test

  // TestGraphNodeExecutionPersistence verifica que se puede:
  // 1. Crear un Graph con entidad Ent.
  // 2. Crear 2 Nodes asociados al Graph.
  // 3. Crear una Execution asociada al Graph.
  // 4. Recuperar el Graph con sus Nodes y Executions.
  // 5. Actualizar el status de la Execution a "running".
  // 6. Verificar que los JSON fields se persisten y recuperan correctamente.
  func TestGraphNodeExecutionPersistence(t *testing.T)

  // TestExecutionStatusTransitions verifica las transiciones de estado
  // válidas de una Execution.
  func TestExecutionStatusTransitions(t *testing.T)
  ```

  La cadena de conexión se lee de la variable de entorno
  `DATABASE_URL` (con fallback a `postgres://dago:dago@localhost:5432/dago?sslmode=disable`).

  **Prerequisito:** `make docker-up` y migración aplicada.

- **Criterio de aceptación:** `make test-integration` (con docker-compose
  activo) pasa con todos los tests en PASS. Falla si PostgreSQL no está
  disponible (no silencia el error — el test debe ser activo).
- **Depende de:** #6, #8
- **Commit:** `test(integration): add PostgreSQL integration tests for schemas [SPRINT-002 #9]`

### 10. [docs] Actualizar docs/index.md y docs/log.md

- **Agente:** @docs
- **Descripción:** Actualizar la sección "Dominio" de `docs/index.md`
  marcando los tres schemas como implementados. Añadir enlace al
  documento de sprint SPRINT-002. Actualizar `docs/log.md` con la
  entrada de cierre del sprint.
- **Criterio de aceptación:** `docs/index.md` refleja los schemas
  creados. SPRINT-002 aparece en la tabla de sprints. `docs/log.md`
  tiene la entrada con el resultado del sprint.
- **Depende de:** #9
- **Commit:** `docs(schema): update index with SPRINT-002 results [SPRINT-002 #10]`

## Matriz de trazabilidad

| Spec / ADR | Regla | TODO | Artefacto | Verificado por |
|------------|-------|------|-----------|----------------|
| ADR-007 regla 1 | Schema Ent = spec de datos | #1 | alineación con `graph.json` | revisión TODO #1 |
| ADR-007 regla 2 | `go generate` → commitear | #5 | `ent/` generado | `go build ./...` |
| ADR-007 regla 3 | Atlas genera migraciones | #7 | `migrations/` | `atlas migrate lint` |
| ADR-007 regla 4 | Linting de migraciones en CI | #7 | `atlas migrate lint` | sin errores |
| ADR-007 regla 8 | UUIDs + TIMESTAMPTZ | #2, #3, #4 | campos de schemas | `\d` PostgreSQL |
| ADR-016 | 7 patrones de nodo | #3 | `Node.pattern` enum | `TestNodePatternValidation` |
| ADR-016 | Grafo: id, name, version, entry_node | #2 | `Graph` schema | `TestGraphCreate` |
| ADR-015 | Working Memory: status, variables, messages, node_results | #4 | `Execution` schema | `TestExecutionCreate` |
| ADR-015 | Nunca borrar — solo superseder | #4 | sin `DeleteExecution` en uso | tests de integración |
| `specs/patterns/graph.json` | version semver `^\d+\.\d+\.\d+$` | #2 | `Graph.version` validado | `TestGraphVersionValidation` |
| ADR-002 | TDD (Red → Green) | #6, #9 | tests antes de TODO #5 | `make test` |
| ADR-001 | Tipos Ent no salen del adaptador | #2–#4 | schemas en `ent/schema/` | code review |
| ADR-003 | Funciones ≤20 líneas | #2–#4 | schemas Go | `make lint` |

## Criterios de aceptación del sprint

```bash
# 1. El código Ent compila
go build ./ent/...

# 2. Tests unitarios pasan (sin Docker)
go test ./tests/unit/schema/...

# 3. Linter sin errores
make lint

# 4. Migración generada y sin errores de lint
atlas migrate lint --env local

# 5. Migración aplicada
atlas migrate apply --env local

# 6. Tablas verificadas en PostgreSQL
docker compose exec postgres psql -U dago -d dago -c "\dt"

# 7. Tests de integración pasan (con Docker)
make test-integration

# 8. Pipeline CI completa
make ci
```

Adicionalmente:
- `migrations/` tiene exactamente un fichero `*_init_graph_node_execution.sql`.
- El SQL no contiene `DROP`, `DELETE` ni cambios destructivos.
- Los tres schemas tienen `created_at` inmutable y `updated_at` auto.
- Los campos JSON usan tipo `jsonb` en PostgreSQL (no `json`).
- `Node.pattern` solo acepta los 7 valores de ADR-016.
- `Execution.status` solo acepta los 6 valores de ADR-015.

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
