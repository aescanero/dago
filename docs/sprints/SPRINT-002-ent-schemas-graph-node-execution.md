# SPRINT-002: Ent Schemas — Graph, Node and Execution

## Metadata

- **Start date:** 2026-04-28 (after completing SPRINT-001)
- **Estimated end date:** 2026-04-29
- **Status:** completed
- **ADRs applied:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-007, ADR-015, ADR-016
- **Affected specs:**
  - `specs/patterns/graph.json` (read — `Definition` field)
  - `ent/schema/` (data source of truth — ADR-007 rule 1)
- **Planning agent:** planner
- **Reviewed by:** pending
- **Blocks:** SPRINT-003 (orchestrator), SPRINT-015 (memory)

## Sprint Objective

Implement the three base Ent schemas of the orchestration model —
`Graph`, `Node` and `Execution` — with their relationships, fields and
validations. Generate the Ent code with `go generate`, produce the
first migration with Atlas, and verify that it applies cleanly
against the PostgreSQL from docker-compose.

At the end there is a compilable and migrated data model, with unit
tests for the schemas and an integration test that confirms
real persistence in PostgreSQL.

## Scope

### Included

- `ent/schema/graph.go` — Ent schema for the `Graph` entity.
- `ent/schema/node.go` — Ent schema for the `Node` entity.
- `ent/schema/execution.go` — Ent schema for the `Execution` entity.
- `go generate ./ent` — generated code committed (ADR-007 rule 2).
- `atlas migrate diff init_graph_node_execution` — first SQL migration.
- `atlas migrate apply --env local` — migration applied against PostgreSQL.
- Unit tests for schemas (`tests/unit/schema/`) with testify.
- Integration test (`tests/integration/`) that creates and retrieves the
  three entity types against real PostgreSQL.
- Update of `docs/index.md` with the Ent schemas table.
- Entry in `docs/log.md`.

### Excluded

- `ent/schema/user.go`, `org_unit.go` — implemented in SPRINT-auth.
- `ent/schema/episode.go`, `semantic_fact.go` — implemented in
  SPRINT-015 (memory).
- `ent/schema/package.go`, `mcp_server.go`, `agent_card.go` — respective
  support sprints.
- Repository ports (`libs/ports/`) — implemented when the
  first service needs them (SPRINT-003).
- Storage adapter (`adapters/storage/`) — same criterion.
- Graph validation logic (ADR-016) — SPRINT-003.
- pgvector — only needed for `SemanticFact` (SPRINT-015).

## Dependencies

- **SPRINT-001 completed:** needs `go.mod` with `entgo.io/ent`,
  `ent/schema/` directory, `atlas.hcl`, docker-compose with PostgreSQL.
- **Reference specs:** `specs/patterns/graph.json` and ADR-016 for
  `Graph` and `Node` fields. ADR-015 for `Execution`.

## Behavior Contracts

### C1 — Semver version validation in Graph

```
Given: A Graph with field version = "abc" (not semver)
When: ent.Client.Graph.Create().SetVersion("abc").Save(ctx)
Then: The ORM returns a validation error
      The error references the "version" field
      No row is inserted in the graphs table
```

### C2 — Default status of Execution

```
Given: An Execution created without explicitly specifying status
When: ent.Client.Execution.Create().SetGraphID(...).Save(ctx)
Then: The status field of the persisted record is "pending"
      The row is inserted correctly
```

### C3 — Atlas migration produces correct tables

```
Given: PostgreSQL available and migration file generated with `atlas migrate diff`
When: `atlas migrate apply --env local` is executed
Then: Tables graphs, nodes, executions exist in PostgreSQL
      The FK nodes.graph_id → graphs.id is active
      The FK executions.graph_id → graphs.id is active
      `atlas_schema_revisions` contains no errors
```

## Data Model

### Entity `Graph`

Represents the **definition** of an execution graph (the template,
not a concrete execution).

| Field | Ent Type | PostgreSQL Type | Notes |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | Generated with `uuid.New()` |
| `name` | `string` | `VARCHAR(255) NOT NULL` | Descriptive name |
| `version` | `string` | `VARCHAR(20) NOT NULL` | Semver: `^\d+\.\d+\.\d+$` |
| `description` | `string` (optional) | `TEXT` | Free description |
| `entry_node` | `string` | `VARCHAR(255) NOT NULL` | Key of the entry node |
| `definition` | `json.RawMessage` | `JSONB NOT NULL` | Complete JSON of the graph (nodes + edges) |
| `memory_config` | `json.RawMessage` (optional) | `JSONB` | `semantic_search`, `episode_context` |
| `status` | `enum` | `VARCHAR(20) NOT NULL DEFAULT 'draft'` | `draft \| active \| archived` |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Immutable, auto |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Auto on each update |

Ent Edges:
- `Nodes []Node` — one Graph has many Nodes (`O2M`)
- `Executions []Execution` — one Graph has many Executions (`O2M`)

Indexes:
- `(name, version)` UNIQUE — a version of a graph is unique.
- `status` — to filter active graphs.

### Entity `Node`

Represents an **individual node** within a graph. Stored
as a separate entity to allow queries and metrics per node.

| Field | Ent Type | PostgreSQL Type | Notes |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | Generated with `uuid.New()` |
| `graph_id` | `uuid.UUID` | `UUID FK → graphs.id` | Required FK |
| `node_key` | `string` | `VARCHAR(255) NOT NULL` | Key in the graph JSON (e.g., `"classifier"`) |
| `pattern` | `enum` | `VARCHAR(50) NOT NULL` | `llm_call \| tool_use \| react \| reflection \| router \| guardrail \| subgraph` |
| `config` | `json.RawMessage` | `JSONB NOT NULL` | Pattern-specific configuration |
| `position` | `json.RawMessage` (optional) | `JSONB` | `{"x":0,"y":0}` for visual editor |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Immutable, auto |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Auto on each update |

Ent Edges:
- `Graph *Graph` — many Nodes belong to one Graph (`M2O`)

Indexes:
- `(graph_id, node_key)` UNIQUE — a node key is unique per graph.
- `(graph_id, pattern)` — to filter nodes by pattern.

### Entity `Execution`

Represents an **execution instance** of a graph. This is the
Working Memory layer from ADR-015: live state of the current execution.

| Field | Ent Type | PostgreSQL Type | Notes |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | Generated with `uuid.New()` |
| `graph_id` | `uuid.UUID` | `UUID FK → graphs.id` | Required FK |
| `status` | `enum` | `VARCHAR(20) NOT NULL DEFAULT 'pending'` | `pending \| running \| completed \| failed \| cancelled \| interrupted` |
| `current_node` | `string` (optional) | `VARCHAR(255)` | Node in active execution |
| `variables` | `json.RawMessage` | `JSONB NOT NULL DEFAULT '{}'` | Accumulated graph variables |
| `messages` | `json.RawMessage` | `JSONB NOT NULL DEFAULT '[]'` | Conversation history |
| `node_results` | `json.RawMessage` | `JSONB NOT NULL DEFAULT '{}'` | Results per node |
| `error` | `string` (optional) | `TEXT` | Error message if `status=failed` |
| `started_at` | `time.Time` (optional) | `TIMESTAMPTZ` | When it moved to `running` |
| `completed_at` | `time.Time` (optional) | `TIMESTAMPTZ` | When it finished (success or failure) |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Immutable, auto |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | Auto on each update |

Ent Edges:
- `Graph *Graph` — many Executions belong to one Graph (`M2O`)

Indexes:
- `(graph_id, status)` — to query active executions of a graph.
- `status` — for the orchestrator when recovering pending executions.
- `created_at` — for chronological pagination.

## TODOs

### 1. [spec] Review graph.json and derive final fields

- **Agente:** @developer
- **Description:** Read `specs/patterns/graph.json` and the node
  patterns in `specs/patterns/nodes/` to confirm that the data model
  fields correctly reflect the spec. Any discrepancy
  between spec and plan is resolved in favor of the spec (ADR-007 rule 1).
  Document decisions in the results section of this sprint.
- **Acceptance criteria:** The fields in the tables above are
  aligned with `specs/patterns/graph.json`. No undocumented discrepancies.
- **Dependencies:** none
- **Commit:** `docs(schema): verify graph.json alignment with data model [SPRINT-002 #1]`

### 2. [data] Implement `ent/schema/graph.go`

- **Agente:** @developer
- **Description:** Create the Ent schema for `Graph` according to the data
  model in this document. Apply ADR-007 conventions:
  UUIDs, TIMESTAMPTZ, immutability of `created_at`, Ent validations
  for `version` (semver regex) and `status` (enum).

  File structure:

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

  Validations to include:
  - `version`: `field.String("version").Match(regexp.MustCompile(...))`
  - `status`: `field.Enum("status").Values("draft", "active", "archived").Default("draft")`
  - `definition`: `field.JSON("definition", json.RawMessage{}).SchemaType(map[string]string{"postgres": "jsonb"})`
  - `created_at`: `field.Time("created_at").Default(time.Now).Immutable()`
  - `updated_at`: `field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now)`

- **Acceptance criteria:** `go build ./ent/...` compiles without errors.
  Schema has all the fields, edges and indexes from the data model.
- **Dependencies:** #1
- **Commit:** `feat(schema): add Graph Ent schema [SPRINT-002 #2]`

### 3. [data] Implement `ent/schema/node.go`

- **Agente:** @developer
- **Description:** Create the Ent schema for `Node`. The `pattern` field
  is an enum with the 7 values from ADR-016. The `M2O` relationship with `Graph`
  is defined with `edge.From("graph", Graph.Type).Ref("nodes").Unique().Required()`.

  Validations to include:
  - `pattern`: enum with exactly 7 values:
    `llm_call`, `tool_use`, `react`, `reflection`, `router`,
    `guardrail`, `subgraph`.
  - `config`: JSONB not null.
  - `(graph_id, node_key)` UNIQUE via index.

- **Acceptance criteria:** `go build ./ent/...` compiles. The schema
  has the unique index `(graph_id, node_key)` declared.
- **Dependencies:** #2
- **Commit:** `feat(schema): add Node Ent schema [SPRINT-002 #3]`

### 4. [data] Implement `ent/schema/execution.go`

- **Agente:** @developer
- **Description:** Create the Ent schema for `Execution`. The
  `status` field is an enum with 6 values (ADR-015). `variables`, `messages`
  and `node_results` are JSONB with empty default values. The fields
  `started_at` and `completed_at` are optional (nillable).

  Validations:
  - `status`: enum `pending | running | completed | failed | cancelled | interrupted`. Default `pending`.
  - `variables`: `field.JSON("variables", json.RawMessage{}).Default(json.RawMessage("{}"))`.
  - `messages`: `field.JSON("messages", json.RawMessage{}).Default(json.RawMessage("[]"))`.
  - `node_results`: `field.JSON("node_results", json.RawMessage{}).Default(json.RawMessage("{}"))`.
  - `started_at`, `completed_at`: `.Optional().Nillable()`.

- **Acceptance criteria:** `go build ./ent/...` compiles. All
  optional fields are nillable. Indexes `(graph_id, status)` and
  `status` declared.
- **Dependencies:** #2
- **Commit:** `feat(schema): add Execution Ent schema [SPRINT-002 #4]`

### 5. [data] Run `go generate ./ent` and commit generated code

- **Agente:** @developer
- **Description:** Run `go generate ./ent` from the project root.
  This command generates the typed Ent client from the
  three schemas. The generated code is committed (ADR-007 rule 2).

  Verify that the following files are generated:
  ```
  ent/client.go
  ent/ent.go
  ent/generate.go
  ent/graph.go, ent/graph_create.go, ent/graph_update.go, ent/graph_query.go, ent/graph_delete.go
  ent/node.go, ent/node_create.go, ent/node_update.go, ent/node_query.go, ent/node_delete.go
  ent/execution.go, ent/execution_create.go, ent/execution_update.go, ent/execution_query.go, ent/execution_delete.go
  ent/schema/ (the three manual schemas)
  ent/predicate/
  ent/internal/
  ent/enttest/
  ent/hook/
  ent/migrate/
  ```

  **Note:** `ent/generate.go` must exist in the `ent/` directory
  with the content `//go:generate go run entgo.io/ent/cmd/ent generate ./schema`.

- **Acceptance criteria:** `go build ./...` compiles after
  `go generate`. `go vet ./...` without errors. The `ent/` directory
  has all generated files.
- **Dependencies:** #3, #4
- **Commit:** `feat(schema): generate Ent client for Graph, Node, Execution [SPRINT-002 #5]`

### 6. [test] Schema unit tests (Red → Green)

- **Agente:** @qa
- **Description:** Create `tests/unit/schema/graph_test.go`,
  `node_test.go` and `execution_test.go` following TDD (ADR-002).
  Use `ent/enttest` with in-memory SQLite for tests without PostgreSQL.

  Tests to implement:

  **graph_test.go:**
  ```go
  // TestGraphCreate verifies that a valid Graph can be created.
  func TestGraphCreate(t *testing.T)

  // TestGraphVersionValidation verifies that a non-semver version
  // is rejected by the schema.
  func TestGraphVersionValidation(t *testing.T)

  // TestGraphStatusDefault verifies that the default status is "draft".
  func TestGraphStatusDefault(t *testing.T)

  // TestGraphUniqueNameVersion verifies that (name, version) is unique.
  func TestGraphUniqueNameVersion(t *testing.T)
  ```

  **node_test.go:**
  ```go
  // TestNodeCreate verifies that a valid Node can be created.
  func TestNodeCreate(t *testing.T)

  // TestNodePatternValidation verifies that an invalid pattern
  // is rejected.
  func TestNodePatternValidation(t *testing.T)

  // TestNodeUniqueKeyPerGraph verifies that (graph_id, node_key) is unique.
  func TestNodeUniqueKeyPerGraph(t *testing.T)

  // TestNodeBelongsToGraph verifies the M2O relationship with Graph.
  func TestNodeBelongsToGraph(t *testing.T)
  ```

  **execution_test.go:**
  ```go
  // TestExecutionCreate verifies that a valid Execution can be created.
  func TestExecutionCreate(t *testing.T)

  // TestExecutionStatusDefault verifies that the default status
  // is "pending".
  func TestExecutionStatusDefault(t *testing.T)

  // TestExecutionJSONDefaults verifies that variables, messages and
  // node_results have valid default values.
  func TestExecutionJSONDefaults(t *testing.T)

  // TestExecutionOptionalFields verifies that started_at and
  // completed_at are optional.
  func TestExecutionOptionalFields(t *testing.T)
  ```

  Use `enttest.Open(t, "sqlite3", "file:ent?mode=memory&...")`.
  Add `github.com/mattn/go-sqlite3` to `go.mod` only for tests
  (tag `sqlite3`).

- **Acceptance criteria:** `go test ./tests/unit/schema/...` passes
  with all tests in PASS. Validation tests fail if the schema
  validation is removed (active verification).
- **Dependencies:** #5
- **Commit:** `test(schema): add unit tests for Graph, Node, Execution schemas [SPRINT-002 #6]`

### 7. [infra] Generate migration with Atlas

- **Agente:** @developer
- **Description:** With docker-compose running (`make docker-up`),
  run Atlas to generate the first SQL migration:

  ```bash
  atlas migrate diff init_graph_node_execution \
      --dir "file://migrations" \
      --to "ent://ent/schema" \
      --dev-url "docker://postgres/16/dev?search_path=public"
  ```

  Or, using `atlas.hcl`:

  ```bash
  atlas migrate diff init_graph_node_execution --env local
  ```

  Review the SQL generated in `migrations/` before committing.
  The SQL must include:
  - `CREATE TABLE graphs (...)` with all fields and constraints.
  - `CREATE TABLE nodes (...)` with FK to `graphs`.
  - `CREATE TABLE executions (...)` with FK to `graphs`.
  - Indexes declared in the schemas.
  - Enum constraints for `status` and `pattern`.

  Apply Atlas linting:
  ```bash
  atlas migrate lint --env local
  ```
  It must not report destructive changes or locks.

- **Acceptance criteria:** `migrations/YYYYMMDDHHMMSS_init_graph_node_execution.sql`
  exists with the correct SQL. `atlas migrate lint` finishes without errors.
  The SQL is readable and exactly reflects the schemas.
- **Dependencies:** #5, docker-compose running
- **Commit:** `chore(migration): add initial migration for Graph, Node, Execution [SPRINT-002 #7]`

### 8. [infra] Apply migration and verify against PostgreSQL

- **Agente:** @developer
- **Description:** Apply the migration to the docker-compose PostgreSQL
  and verify the result:

  ```bash
  atlas migrate apply --env local
  ```

  Then verify that the tables exist with the correct structure:

  ```bash
  docker compose exec postgres psql -U dago -d dago \
      -c "\dt" \
      -c "\d graphs" \
      -c "\d nodes" \
      -c "\d executions"
  ```

  Verify specifically:
  - The three tables exist.
  - UUID fields are of type `uuid`.
  - JSON/JSONB fields are of type `jsonb`.
  - Time fields are `timestamptz`.
  - FKs for `nodes.graph_id` and `executions.graph_id` point to `graphs.id`.
  - Unique indexes exist.

- **Acceptance criteria:** `atlas migrate apply` finishes with
  `Applied N migration(s)`. The three tables exist in PostgreSQL with
  the correct structure verified by the `\d` commands.
- **Dependencies:** #7
- **Commit:** not applicable (the migration was already committed in #7)

### 9. [test] Integration test against real PostgreSQL

- **Agente:** @qa
- **Description:** Create `tests/integration/schema_integration_test.go`
  with build tag `integration`. The test connects to the docker-compose
  PostgreSQL and verifies real persistence.

  ```go
  //go:build integration

  package integration_test

  // TestGraphNodeExecutionPersistence verifies that you can:
  // 1. Create a Graph with Ent entity.
  // 2. Create 2 Nodes associated with the Graph.
  // 3. Create an Execution associated with the Graph.
  // 4. Retrieve the Graph with its Nodes and Executions.
  // 5. Update the Execution status to "running".
  // 6. Verify that JSON fields are persisted and retrieved correctly.
  func TestGraphNodeExecutionPersistence(t *testing.T)

  // TestExecutionStatusTransitions verifies the valid state transitions
  // of an Execution.
  func TestExecutionStatusTransitions(t *testing.T)
  ```

  The connection string is read from the environment variable
  `DATABASE_URL` (with fallback to `postgres://dago:dago@localhost:5432/dago?sslmode=disable`).

  **Prerequisite:** `make docker-up` and migration applied.

- **Acceptance criteria:** `make test-integration` (with docker-compose
  active) passes with all tests in PASS. Fails if PostgreSQL is not
  available (does not silence the error — the test must be active).
- **Dependencies:** #6, #8
- **Commit:** `test(integration): add PostgreSQL integration tests for schemas [SPRINT-002 #9]`

### 10. [docs] Update docs/index.md and docs/log.md

- **Agente:** @docs
- **Description:** Update the "Domain" section of `docs/index.md`
  marking the three schemas as implemented. Add a link to the
  SPRINT-002 sprint document. Update `docs/log.md` with the
  sprint closing entry.
- **Acceptance criteria:** `docs/index.md` reflects the created schemas.
  SPRINT-002 appears in the sprints table. `docs/log.md`
  has the entry with the sprint result.
- **Dependencies:** #9
- **Commit:** `docs(schema): update index with SPRINT-002 results [SPRINT-002 #10]`

## Traceability Matrix

| Spec / ADR | Rule | TODO | Artifact | Verified by |
|------------|------|------|----------|-------------|
| ADR-007 rule 1 | Ent schema = data spec | #1 | alignment with `graph.json` | TODO #1 review |
| ADR-007 rule 2 | `go generate` → commit | #5 | generated `ent/` | `go build ./...` |
| ADR-007 rule 3 | Atlas generates migrations | #7 | `migrations/` | `atlas migrate lint` |
| ADR-007 rule 4 | Migration linting in CI | #7 | `atlas migrate lint` | no errors |
| ADR-007 rule 8 | UUIDs + TIMESTAMPTZ | #2, #3, #4 | schema fields | `\d` PostgreSQL |
| ADR-016 | 7 node patterns | #3 | `Node.pattern` enum | `TestNodePatternValidation` |
| ADR-016 | Graph: id, name, version, entry_node | #2 | `Graph` schema | `TestGraphCreate` |
| ADR-015 | Working Memory: status, variables, messages, node_results | #4 | `Execution` schema | `TestExecutionCreate` |
| ADR-015 | Never delete — only supersede | #4 | no `DeleteExecution` in use | integration tests |
| `specs/patterns/graph.json` | semver version `^\d+\.\d+\.\d+$` | #2 | `Graph.version` validated | `TestGraphVersionValidation` |
| ADR-002 | TDD (Red → Green) | #6, #9 | tests before TODO #5 | `make test` |
| ADR-001 | Ent types do not leave the adapter | #2–#4 | schemas in `ent/schema/` | code review |
| ADR-003 | Functions ≤20 lines | #2–#4 | Go schemas | `make lint` |

## Sprint Acceptance Criteria

```bash
# 1. Ent code compiles
go build ./ent/...

# 2. Unit tests pass (without Docker)
go test ./tests/unit/schema/...

# 3. Linter without errors
make lint

# 4. Migration generated and without lint errors
atlas migrate lint --env local

# 5. Migration applied
atlas migrate apply --env local

# 6. Tables verified in PostgreSQL
docker compose exec postgres psql -U dago -d dago -c "\dt"

# 7. Integration tests pass (with Docker)
make test-integration

# 8. Complete CI pipeline
make ci
```

Additionally:
- `migrations/` has exactly one file `*_init_graph_node_execution.sql`.
- The SQL does not contain `DROP`, `DELETE` or destructive changes.
- The three schemas have `created_at` immutable and `updated_at` auto.
- JSON fields use type `jsonb` in PostgreSQL (not `json`).
- `Node.pattern` only accepts the 7 values from ADR-016.
- `Execution.status` only accepts the 6 values from ADR-015.

## Sprint Result

_Completed 2026-05-03._

### Tests executed

- Total: 14 (12 unit + 2 integration)
- Passed: 14
- Failed: 0

Unit tests (`go test ./tests/unit/schema/...`): 12 PASS
Integration tests (`go test -tags=integration ./tests/integration/...`): 2 PASS

### Files created/modified

- `ent/schema/graph.go` — Graph Ent schema (UUID PK, semver validation, JSONB, enum status)
- `ent/schema/node.go` — Node Ent schema (7-value pattern enum, unique (graph_id, node_key))
- `ent/schema/execution.go` — Execution Ent schema (6-value status enum, JSONB defaults)
- `ent/generate.go` — `//go:generate` directive
- `ent/entc.go` — entc generation program (build:ignore)
- `ent/` (full generated client) — client.go, ent.go, mutation.go, runtime.go, tx.go,
  graph.go, node.go, execution.go (+_create/_update/_query/_delete for each),
  enttest/, hook/, migrate/, predicate/, internal/
- `cmd/migrate/main.go` — migration generation tool using `schema.Diff` API
- `migrations/20260503191126_init_graph_node_execution.up.sql` — creates graphs, nodes, executions
- `migrations/20260503191126_init_graph_node_execution.down.sql` — drops all three tables
- `migrations/atlas.sum` — Atlas migration integrity hash
- `tests/unit/schema/graph_test.go` — 4 tests (create, version validation, status default, unique)
- `tests/unit/schema/node_test.go` — 4 tests (create, pattern validation, unique key, belongs-to)
- `tests/unit/schema/execution_test.go` — 4 tests (create, status default, JSON defaults, optional fields)
- `tests/integration/schema_integration_test.go` — 2 integration tests (persistence, status transitions)
- `go.mod` / `go.sum` — added `github.com/lib/pq v1.12.3`

### Decisions made during the sprint

1. **Atlas lint skipped** — `atlas migrate lint` requires Atlas Pro since v0.38.
   SQL verified manually with `psql \d` commands. Tables, FKs, indexes, and column types confirmed correct.

2. **Migration format** — Used golang-migrate up/down format (via `schema.Diff` API) instead of
   Atlas versioned format, because the standalone `atlas` CLI does not support the `ent://` data
   source without the Pro extension. The `cmd/migrate/main.go` tool uses `schema.Diff` directly.

3. **`atlas.hcl` `env local` not usable** — The `ent://` URL requires the Atlas Ent provider
   plugin, which is not available in the Community Edition CLI.

4. **SQLite driver** — Used `github.com/mattn/go-sqlite3` (already in go.mod as indirect)
   for unit tests. Pure CGO driver; no issues on the build host.

5. **PostgreSQL driver** — Added `github.com/lib/pq` for integration tests and the migration tool.

### Reviewer notes

_Pending review._
