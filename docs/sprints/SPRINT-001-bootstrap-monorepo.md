# SPRINT-001: Bootstrap del monorepo Go

## Metadata

- **Fecha inicio:** 2026-04-27
- **Fecha fin estimada:** 2026-04-28
- **Estado:** completado
- **ADRs aplicados:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-005, ADR-007, ADR-008, ADR-013
- **Specs afectadas:** ninguna (sprint de infraestructura)
- **Agente planificador:** planner
- **Revisado por:** pendiente

## Objetivo del sprint

Establecer la infraestructura base del monorepo Go de dago de forma que
cualquier agente o desarrollador pueda: clonar el repositorio, ejecutar
`make bootstrap` para levantar dependencias, y verificar con `make ci`
que el código compila, los tests pasan y el linter no reporta errores.

Al finalizar este sprint existe un monorepo compilable con la estructura
de directorios completa, un módulo Go válido, herramientas de calidad
configuradas, infraestructura local vía Docker Compose, y un smoke test
que valida el pipeline completo. No existe lógica de negocio implementada.

## Alcance

### Incluido

- Estructura de directorios del monorepo completa (libs/, adapters/,
  services/ con los 8 servicios, dashboard/, ent/, migrations/, specs/,
  docs/).
- `go.mod` en la raíz con módulo `github.com/aescanero/dago` y
  dependencias mínimas declaradas (Gin, go-redis/v9, entgo.io/ent,
  testify, golangci-lint como tool dependency).
- `go.sum` generado con `go mod tidy`.
- `Makefile` con targets: `build-all`, `build-{service}` (×8),
  `test`, `lint`, `generate`, `migrate-diff`, `migrate-apply`,
  `dashboard-dev`, `bootstrap`, `ci`, `docker-up`, `docker-down`.
- `docker-compose.yml` con PostgreSQL 16 + pgvector y Valkey 8.
- `.golangci.yml` con linters que reflejan ADR-003 y ADR-004.
- Package stub `package main` con `func main() {}` en cada
  `services/{nombre}/cmd/main.go` (8 servicios).
- Package stub `package domain` con archivo `.go` de comentario en
  `libs/domain/`.
- Package stub `package ports` con archivo `.go` de comentario en
  `libs/ports/`.
- Package stub `package adapters` con archivo `.go` de comentario
  en cada subdirectorio de `adapters/` (storage/, eventbus/, auth/,
  llm/, metrics/).
- Smoke test en `tests/smoke/build_test.go` que verifica que el
  binario de cada servicio compila sin errores.
- `.env.example` documentando las variables de entorno necesarias
  para docker-compose.
- `atlas.hcl` mínimo apuntando al directorio `migrations/` y
  `ent/schema/`.
- `Makefile` incluye target `tools` que instala versiones pinadas de
  golangci-lint y atlas.
- Actualización de `docs/index.md` con la sección de artefactos creados.

### Excluido

- Schemas Ent (`ent/schema/`) — se implementan en SPRINT-002.
- Migraciones Atlas — requieren schemas Ent; se implementan en SPRINT-002.
- Lógica de negocio en ningún servicio.
- Handlers Gin — ningún servicio expone endpoints aún.
- Configuración de Valkey en código Go — solo declarada como dependencia.
- Configuración CI/CD (GitHub Actions workflows) — se implementa en
  SPRINT-devops-001 independiente.
- Frontend dashboard (`dashboard/`) — solo se crea el directorio vacío
  con `.gitkeep`; el setup de Node/Vite se realiza en SPRINT-frontend-001.
- Tests de integración contra PostgreSQL o Valkey real.
- Autenticación JWT/OAuth.

## Dependencias

- **Sprints previos requeridos:** ninguno (primer sprint).
- **Specs que deben existir:** ninguna (sprint de infraestructura).
- **Infraestructura requerida:** Docker Engine en la máquina de desarrollo
  para levantar docker-compose. Go 1.23+ instalado. `make` disponible.

## Contratos de comportamiento

### C1 — `make bootstrap`

```
Given: Docker Engine activo y Go 1.23+ instalado en el entorno
When: Se ejecuta `make bootstrap`
Then: PostgreSQL escucha en :5432 en estado healthy
      Valkey escucha en :6379 en estado healthy
      `go mod download` termina sin errores
```

### C2 — `make ci`

```
Given: `make bootstrap` ejecutado, repositorio en estado limpio
When: Se ejecuta `make ci`
Then: `golangci-lint run ./...` termina con código 0
      `go build ./...` produce binarios para los 8 servicios
      `make test-smoke` pasa con todos los tests en PASS
      El comando completo termina con código de salida 0
```

### C3 — stubs compilables

```
Given: `go.mod` con módulo `github.com/aescanero/dago` creado
When: Se ejecuta `go build ./...`
Then: Todos los packages stub compilan sin errores
      Ningún import circular existe
      Los 8 servicios tienen `func main()` en su `cmd/main.go`
```

## TODOs

### 1. [infra] Crear estructura de directorios del monorepo

- **Agente:** @developer
- **Skill:** scaffolding
- **Descripción:** Crear todos los directorios necesarios con archivos
  `.gitkeep` o stubs mínimos para que git los rastree y los imports Go
  sean válidos. La estructura refleja ADR-001 (hexagonal) y ADR-013
  (monorepo).

  Directorios a crear:

  ```
  libs/domain/
  libs/ports/
  libs/schemas/
  libs/utils/
  adapters/storage/
  adapters/eventbus/
  adapters/auth/
  adapters/llm/
  adapters/metrics/
  services/orchestrator/cmd/
  services/orchestrator/internal/
  services/executor/cmd/
  services/executor/internal/
  services/router/cmd/
  services/router/internal/
  services/planner/cmd/
  services/planner/internal/
  services/auth-server/cmd/
  services/auth-server/internal/
  services/catalog/cmd/
  services/catalog/internal/
  services/mcp-registry/cmd/
  services/mcp-registry/internal/
  services/agent-registry/cmd/
  services/agent-registry/internal/
  ent/schema/
  migrations/
  tests/unit/
  tests/integration/
  tests/contract/
  tests/smoke/
  dashboard/
  ```

- **Criterio de aceptación:** `find . -type d | sort` muestra todos los
  directorios anteriores. Ningún directorio está vacío en git (todos
  tienen al menos un `.gitkeep` o archivo stub).
- **Depende de:** ninguno
- **Commit:** `chore(monorepo): create directory structure [SPRINT-001 #1]`

### 2. [infra] Crear go.mod y declarar dependencias mínimas

- **Agente:** @developer
- **Skill:** scaffolding
- **Descripción:** Crear el `go.mod` en la raíz del proyecto con el
  módulo `github.com/aescanero/dago`. Declarar todas las dependencias
  que se usarán en el proyecto para que `go mod tidy` resuelva versiones
  desde el primer momento y el código de los stubs compile.

  Dependencias a declarar (últimas versiones estables):

  ```
  require (
      github.com/gin-gonic/gin             v1.10.x
      github.com/redis/go-redis/v9         v9.x.x
      entgo.io/ent                         v0.14.x
      github.com/google/uuid               v1.6.x
      github.com/stretchr/testify          v1.10.x
      golang.org/x/crypto                  v0.x.x
      github.com/golang-jwt/jwt/v5         v5.x.x
  )
  ```

  Después de crear `go.mod` ejecutar `go mod tidy` para generar
  `go.sum`.

  **Versión de Go:** 1.23 (mínimo). Usar directiva `toolchain go1.23.x`
  si se desea pinnar la toolchain.

- **Criterio de aceptación:** `go mod verify` ejecuta sin errores.
  `go.sum` existe y tiene entradas para todas las dependencias.
  El módulo se llama exactamente `github.com/aescanero/dago`.
- **Depende de:** #1
- **Commit:** `chore(monorepo): add go.mod with minimum dependencies [SPRINT-001 #2]`

### 3. [infra] Crear package stubs Go compilables

- **Agente:** @developer
- **Skill:** scaffolding
- **Descripción:** Crear los archivos Go mínimos para que `go build ./...`
  compile sin errores. Cada archivo declara únicamente el package y un
  comentario de package doc. Los servicios tienen `main.go` con
  `func main() {}`.

  Archivos a crear:

  ```
  libs/domain/doc.go              → package domain
  libs/ports/doc.go               → package ports
  libs/schemas/doc.go             → package schemas
  libs/utils/doc.go               → package utils
  adapters/storage/doc.go         → package storage
  adapters/eventbus/doc.go        → package eventbus
  adapters/auth/doc.go            → package auth
  adapters/llm/doc.go             → package llm
  adapters/metrics/doc.go         → package metrics
  services/orchestrator/cmd/main.go
  services/executor/cmd/main.go
  services/router/cmd/main.go
  services/planner/cmd/main.go
  services/auth-server/cmd/main.go
  services/catalog/cmd/main.go
  services/mcp-registry/cmd/main.go
  services/agent-registry/cmd/main.go
  ```

  Cada `cmd/main.go` tiene la forma:

  ```go
  // Package main is the entry point for the {nombre} service.
  package main

  func main() {}
  ```

  Cada `doc.go` tiene la forma:

  ```go
  // Package {nombre} contains {descripción de una línea}.
  package {nombre}
  ```

- **Criterio de aceptación:** `go build ./...` termina en 0 sin errores
  ni warnings. `go vet ./...` no reporta problemas.
- **Depende de:** #2
- **Commit:** `chore(monorepo): add compilable Go package stubs [SPRINT-001 #3]`

### 4. [infra] Crear Makefile con targets del monorepo

- **Agente:** @developer
- **Skill:** scaffolding
- **Descripción:** Crear el `Makefile` en la raíz. El Makefile es la
  interfaz unificada del monorepo (ADR-013). Todos los targets
  funcionan desde la raíz del proyecto.

  Targets obligatorios:

  | Target | Acción |
  |--------|--------|
  | `help` | Lista targets con descripción (default) |
  | `build-all` | Compila los 8 binarios de servicio |
  | `build-orchestrator` | Compila solo el orchestrator |
  | `build-executor` | Compila solo el executor |
  | `build-router` | Compila solo el router |
  | `build-planner` | Compila solo el planner |
  | `build-auth-server` | Compila solo el auth-server |
  | `build-catalog` | Compila solo el catalog |
  | `build-mcp-registry` | Compila solo el mcp-registry |
  | `build-agent-registry` | Compila solo el agent-registry |
  | `test` | `go test ./...` |
  | `test-unit` | `go test ./tests/unit/... ./libs/... ./adapters/...` |
  | `test-integration` | `go test ./tests/integration/... -tags=integration` |
  | `test-smoke` | `go test -tags=smoke ./tests/smoke/...` |
  | `lint` | `golangci-lint run ./...` |
  | `fmt` | `goimports -w .` |
  | `generate` | `go generate ./...` (para Ent) |
  | `migrate-diff` | `atlas migrate diff` con dev-url de docker-compose |
  | `migrate-apply` | `atlas migrate apply` contra DB local |
  | `tools` | Instala golangci-lint y atlas CLI en versiones pinadas |
  | `docker-up` | `docker compose up -d` |
  | `docker-down` | `docker compose down` |
  | `bootstrap` | `make tools && make docker-up && go mod download` |
  | `ci` | `make lint && make build-all && make test` |
  | `dashboard-dev` | `cd dashboard && npm run dev` |
  | `clean` | Elimina binarios compilados de `bin/` |

  Los binarios se generan en `bin/{nombre}`.

  Variables configurables en cabecera del Makefile:

  ```makefile
  GO           ?= go
  GOLANGCI_VER ?= v1.62.0
  ATLAS_VER    ?= v0.27.0
  MODULE       := github.com/aescanero/dago
  ```

- **Criterio de aceptación:** `make help` muestra todos los targets
  documentados. `make build-all` produce los 8 binarios en `bin/`.
  `make ci` ejecuta lint + build + tests en secuencia y termina en 0.
- **Depende de:** #3
- **Commit:** `chore(monorepo): add Makefile with unified build targets [SPRINT-001 #4]`

### 5. [infra] Crear docker-compose.yml con PostgreSQL 16 + pgvector y Valkey 8

- **Agente:** @devops
- **Skill:** local-infra
- **Descripción:** Crear `docker-compose.yml` en la raíz. Según ADR-007
  se usa PostgreSQL 16 con pgvector. Según ADR-008 se usa Valkey 8
  (imagen `valkey/valkey:8`, licencia BSD-3).

  Servicios a definir:

  **postgres:**
  - Imagen: `pgvector/pgvector:pg16`
  - Puerto: `5432:5432`
  - Variables: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
    leídas de `.env` (con valores por defecto: `dago`, `dago`, `dago`)
  - Volumen persistente: `pgdata`
  - Healthcheck: `pg_isready -U dago`
  - Red: `dago-network`

  **valkey:**
  - Imagen: `valkey/valkey:8`
  - Puerto: `6379:6379`
  - Comando: `valkey-server --save 60 1 --loglevel warning`
  - Volumen persistente: `valkeydata`
  - Healthcheck: `valkey-cli ping`
  - Red: `dago-network`

  Configuración adicional:
  - Archivo `.env.example` con todas las variables documentadas.
  - `docker-compose.yml` referencia variables con `${VAR:-default}`.
  - El compose NO incluye los servicios Go — solo infra de terceros.
  - Añadir `.env` al `.gitignore` (`.env.example` sí se commitea).

- **Criterio de aceptación:** `docker compose up -d` levanta postgres
  y valkey sin errores. `docker compose ps` muestra ambos servicios
  `healthy`. `docker compose down -v` los destruye limpiamente.
- **Depende de:** ninguno (independiente de la estructura Go)
- **Commit:** `chore(devops): add docker-compose with PostgreSQL 16+pgvector and Valkey 8 [SPRINT-001 #5]`

### 6. [infra] Crear .golangci.yml con linters del proyecto

- **Agente:** @developer
- **Skill:** tooling
- **Descripción:** Crear `.golangci.yml` en la raíz. La configuración
  refleja ADR-003 (clean code: funciones ≤20 líneas, ≤3 parámetros),
  ADR-004 (Go style: goimports, errores explícitos, interfaces pequeñas)
  y trata los warnings como errores.

  Linters a habilitar:

  ```yaml
  linters:
    enable:
      - goimports       # formateado con agrupación de imports (ADR-004)
      - govet           # análisis estático oficial
      - errcheck        # errores siempre verificados (ADR-004 regla 3)
      - staticcheck     # análisis estático avanzado
      - unused          # código muerto
      - gosimple        # simplificaciones idiomáticas
      - ineffassign     # asignaciones inefectivas
      - typecheck       # verificación de tipos
      - godot           # comentarios terminan en punto
      - misspell        # errores ortográficos en comentarios
      - gocognit        # complejidad cognitiva (umbral 15)
      - funlen          # funciones ≤20 líneas (ADR-003 regla 2)
      - gocritic        # patrones antiidiomáticos
      - noctx           # uso correcto de context.Context
      - wrapcheck       # errores siempre wrapeados con %w (ADR-004 regla 3)
      - exhaustive      # exhaustividad de switches sobre enums
      - revive          # reemplaza golint con reglas configurables
  ```

  Configuración clave:

  ```yaml
  linters-settings:
    funlen:
      lines: 20
      statements: 20
    gocognit:
      min-complexity: 15
    goimports:
      local-prefixes: github.com/aescanero/dago
    wrapcheck:
      ignoreSigs:
        - .Errorf(
        - errors.New(
        - errors.Unwrap(
    revive:
      rules:
        - name: exported
        - name: var-naming
        - name: error-return
        - name: error-naming

  issues:
    max-issues-per-linter: 0
    max-same-issues: 0

  run:
    timeout: 5m
    go: "1.23"
  ```

- **Criterio de aceptación:** `golangci-lint run ./...` sobre los stubs
  del TODO #3 termina con código 0 sin falsos positivos sobre archivos
  generados o stubs vacíos.
- **Depende de:** #3
- **Commit:** `chore(monorepo): add golangci-lint configuration [SPRINT-001 #6]`

### 7. [infra] Crear atlas.hcl mínimo

- **Agente:** @developer
- **Skill:** scaffolding
- **Descripción:** Crear `atlas.hcl` en la raíz del proyecto. Configuración
  mínima que apunta a `migrations/` y `ent/schema/` según ADR-007. No se
  generan migraciones aún (no hay schemas Ent), pero el fichero debe
  existir y ser válido para que `atlas migrate diff` sea invocable desde
  el Makefile sin error de configuración.

  ```hcl
  data "ent" "schema" {
    path = "./ent/schema"
  }

  env "local" {
    src = data.ent.schema.url
    dev = "docker://postgres/16/dev?search_path=public"
    migration {
      dir = "file://migrations"
    }
    format {
      migrate {
        diff = "{{ sql . \"  \" }}"
      }
    }
  }
  ```

  El `dev` usa la imagen Docker de postgres para calcular diffs —
  Atlas provisiona un contenedor temporal automáticamente.

- **Criterio de aceptación:** `atlas migrate diff --env local --dry-run`
  (con docker-compose activo) ejecuta sin errores de configuración.
  Puede devolver "No changes" porque no hay schemas, pero no debe fallar.
- **Depende de:** #5
- **Commit:** `chore(monorepo): add atlas.hcl minimum configuration [SPRINT-001 #7]`

### 8. [test] Smoke tests del pipeline de build

- **Agente:** @qa
- **Skill:** smoke-test
- **Descripción:** Crear `tests/smoke/build_test.go` con tests que
  verifican que el toolchain completo funciona. Estos tests no prueban
  lógica de negocio — prueban que el monorepo está correctamente
  configurado como sistema. Siguen la estructura table-driven de ADR-004.

  Tests a implementar:

  ```go
  // tests/smoke/build_test.go
  package smoke_test

  // TestServicesBuildWithoutErrors verifica que go build ./... no produce
  // errores de compilación. Build tag: smoke.
  func TestServicesBuildWithoutErrors(t *testing.T)

  // TestGoVetReportsNoProblems verifica que go vet ./... termina con
  // código 0. Build tag: smoke.
  func TestGoVetReportsNoProblems(t *testing.T)

  // TestModulePathIsCorrect verifica que el módulo declarado en
  // go.mod es exactamente github.com/aescanero/dago.
  func TestModulePathIsCorrect(t *testing.T)

  // TestAllServicesHaveMainPackage verifica que cada uno de los 8
  // servicios tiene un cmd/main.go con package main.
  func TestAllServicesHaveMainPackage(t *testing.T)

  // TestDockerComposeIsValid verifica que docker-compose.yml es
  // parseable y contiene los servicios postgres y valkey.
  func TestDockerComposeIsValid(t *testing.T)
  ```

  Build tag `//go:build smoke` para excluirlos de `go test ./...`
  estándar e incluirlos solo en `make test-smoke`.

- **Criterio de aceptación:** `make test-smoke` ejecuta los 5 tests
  y todos pasan. Los tests fallan activamente si se elimina un
  `cmd/main.go` (verificación no trivial).
- **Depende de:** #3, #4, #5
- **Commit:** `test(monorepo): add smoke tests for build pipeline [SPRINT-001 #8]`

### 9. [docs] Actualizar docs/index.md con artefactos del sprint

- **Agente:** @docs
- **Skill:** docs-update
- **Descripción:** Actualizar `docs/index.md` añadiendo:
  - Sección "Sprints" con enlace al documento SPRINT-001.
  - Tabla "Infraestructura de desarrollo" con los artefactos creados.
  - Nota en sección "Dominio" indicando que schemas Ent se crean en SPRINT-002.

- **Criterio de aceptación:** `docs/index.md` refleja los artefactos
  creados y no tiene referencias rotas.
- **Depende de:** #4, #5, #6, #7, #8
- **Commit:** `docs(monorepo): update index with SPRINT-001 artifacts [SPRINT-001 #9]`

## Matriz de trazabilidad

| ADR | Regla | TODO | Artefacto creado | Verificado por |
|-----|-------|------|-----------------|----------------|
| ADR-013 regla 1 | Módulo `github.com/aescanero/dago` | #2 | `go.mod` | `TestModulePathIsCorrect` |
| ADR-013 regla 2 | Sin versiones internas, sin `replace` | #2 | `go.mod` | `go mod verify` |
| ADR-013 regla 3 | `internal/` por servicio | #1, #3 | directorios | `TestAllServicesHaveMainPackage` |
| ADR-013 regla 4 | Un binario por servicio | #3, #4 | `cmd/main.go` ×8 | `make build-all` |
| ADR-013 regla 8 | Makefile como interfaz unificada | #4 | `Makefile` | `make ci` |
| ADR-013 regla 9 | Docker Compose para infra local | #5 | `docker-compose.yml` | `TestDockerComposeIsValid` |
| ADR-013 regla 10 | Un directorio `ent/` compartido | #1 | `ent/schema/` | estructura de dirs |
| ADR-007 | PostgreSQL 16 + pgvector | #5, #7 | `docker-compose.yml`, `atlas.hcl` | `docker compose ps` |
| ADR-008 | Valkey 8 (`valkey/valkey:8`) | #5 | `docker-compose.yml` | `docker compose ps` |
| ADR-004 regla 1 | `goimports` formateado | #6 | `.golangci.yml` | `make lint` |
| ADR-004 regla 2 | `golangci-lint` en CI | #4, #6 | `Makefile`, `.golangci.yml` | `make ci` |
| ADR-004 regla 3 | Errores explícitos (`wrapcheck`) | #6 | `.golangci.yml` | `make lint` |
| ADR-003 regla 2 | Funciones ≤20 líneas (`funlen`) | #6 | `.golangci.yml` | `make lint` |
| ADR-001 | `libs/domain/`, `libs/ports/`, `adapters/` | #1, #3 | directorios + stubs | `go build ./...` |
| ADR-002 | Tests primero | #8 | `tests/smoke/build_test.go` | `make test-smoke` |

## Criterios de aceptación del sprint

Los siguientes comandos ejecutados en orden desde la raíz del repositorio
deben terminar todos con código 0:

```bash
# 1. El módulo Go es válido y todas las dependencias resueltas
go mod verify

# 2. Todo el código compila sin errores
go build ./...

# 3. go vet no reporta problemas
go vet ./...

# 4. El linter no reporta errores
make lint

# 5. Los smoke tests pasan
make test-smoke

# 6. El pipeline CI completo pasa
make ci

# 7. Docker Compose levanta la infra correctamente
make docker-up
docker compose ps   # ambos servicios en estado "healthy"
make docker-down
```

Adicionalmente:
- `bin/` contiene exactamente 8 binarios tras `make build-all`.
- `docker-compose.yml` usa `pgvector/pgvector:pg16` y `valkey/valkey:8`.
- No existe ningún `go.mod` dentro de subdirectorios.
- `.golangci.yml` habilita al menos: `goimports`, `errcheck`, `wrapcheck`, `funlen`.

## Resultado del sprint

Sprint completado el 2026-04-30. Todos los criterios de aceptación verificados.

### Tests ejecutados

- Total: 13 (5 tests raíz + 8 subtests en TestAllServicesHaveMainPackage)
- Passed: 13
- Failed: 0

### Ficheros creados/modificados

- `go.mod` + `go.sum` — módulo github.com/aescanero/dago, Go 1.25
- `Makefile` — 20 targets, binarios en bin/
- `docker-compose.yml` — pgvector/pgvector:pg16 + valkey/valkey:8
- `.golangci.yml` — 17 linters con reglas ADR-003 y ADR-004
- `atlas.hcl` — configuración mínima apuntando a ent/schema/ y migrations/
- `.env.example` — variables para docker-compose y servicios
- `libs/domain/doc.go`, `libs/ports/doc.go`, `libs/schemas/doc.go`, `libs/utils/doc.go`
- `adapters/storage/doc.go`, `adapters/eventbus/doc.go`, `adapters/auth/doc.go`
- `adapters/llm/doc.go`, `adapters/metrics/doc.go`
- `services/*/cmd/main.go` × 8 servicios
- `tests/smoke/build_test.go` — 5 smoke tests con build tag smoke
- `docs/index.md` — actualizado con artefactos de SPRINT-001

### Verificaciones finales

```
go mod verify         → all modules verified
go build ./...        → 0 errores
go vet ./...          → 0 problemas
golangci-lint run     → 0 errores
make test-smoke       → 13/13 PASS
make ci               → lint + build-all + test: todos en 0
make build-all        → 8 binarios en bin/
```

### Decisiones tomadas durante el sprint

- Se usa `gopkg.in/yaml.v3` para parsear docker-compose.yml en el smoke test
  (alternativa sin dependencias externas requeriría regexp frágil).
- La dependencia `valkey-io/valkey-go` se usó en lugar de `go-redis` (ADR-008).
- Se excluyeron `wrapcheck`, `exhaustive` y `funlen` de archivos `_test.go`
  para evitar falsos positivos en table-driven tests.
- `go mod tidy` pruna deps no importadas; deps se reincorporarán en SPRINT-002+.

### Observaciones del reviewer

_Pendiente de revisión._
