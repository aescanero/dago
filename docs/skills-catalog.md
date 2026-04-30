# Catálogo de Skills y Agentes para Claude Code

## Propósito

Este documento define los skills (procedimientos) y los agentes (roles
especializados) que Claude Code utilizará para trabajar en el proyecto
dago. Los skills dicen *cómo hacer algo*. Los agentes dicen *quién lo
hace, con qué contexto y qué propósito*.

Los skills se crearán a medida que se implemente la infraestructura.
Este catálogo sirve como índice y plan.

## Estados

| Estado | Significado |
|--------|-------------|
| 📋 Planificado | Definido, pendiente de crear |
| 🚧 En progreso | Parcialmente implementado |
| ✅ Completado | Listo para usar |

---

# Parte 1: Agentes

Los agentes son configuraciones especializadas de Claude Code. Cada
agente tiene un propósito, un conjunto de skills, y un contexto
(qué ADRs y specs conoce en profundidad).

## `planner` — Agente de Planificación

**Propósito:** Descomponer requisitos, historias de usuario o tareas
complejas en TODOs accionables que los demás agentes ejecutan.

**Responsabilidades:**
- Analizar un requisito contra los ADRs y specs del proyecto.
- Identificar qué servicios, componentes y specs se ven afectados.
- Generar una lista de TODOs ordenados con dependencias entre ellos.
- Asignar cada TODO al agente adecuado.
- Identificar riesgos, decisiones pendientes o ambigüedades.
- Estimar complejidad relativa de cada TODO.

**Contexto prioritario:** Todos los ADRs, estructura del monorepo,
catálogo de skills, specs (OpenAPI, AsyncAPI, patterns).

**Skills que usa:** `sdlc`, `adr` (para proponer nuevos ADRs si detecta
decisiones no cubiertas).

**Formato de output:**

```markdown
## Plan: [título del requisito]

### Análisis
- Servicios afectados: orchestrator, executor
- Specs a modificar: openapi.yaml, asyncapi.yaml
- Entidades nuevas: ExecutionCheckpoint
- Riesgos: [descripción]

### TODOs

1. **[spec]** Definir endpoint POST /api/v1/checkpoints en OpenAPI
   - Agente: `developer`
   - Skill: `new-endpoint`
   - Depende de: ninguno

2. **[spec]** Definir evento checkpoint.created en AsyncAPI
   - Agente: `developer`
   - Skill: `new-event`
   - Depende de: ninguno

3. **[data]** Crear schema Ent para ExecutionCheckpoint
   - Agente: `developer`
   - Skill: `new-entity`
   - Depende de: ninguno

4. **[impl]** Implementar handler y caso de uso de checkpoint
   - Agente: `developer`
   - Skill: `new-endpoint`
   - Depende de: #1, #3

5. **[test]** Tests unitarios del caso de uso
   - Agente: `qa`
   - Skill: `unit-test`
   - Depende de: #4

6. **[test]** Tests de integración con Testcontainers
   - Agente: `qa`
   - Skill: `integration-test`
   - Depende de: #4

7. **[docs]** Actualizar diagrama de componentes del orchestrator
   - Agente: `docs`
   - Skill: `logical-view`
   - Depende de: #4
```

**Cómo se activa:** Un desarrollador describe un requisito en lenguaje
natural. El agente planner genera el plan. El desarrollador revisa,
ajusta si es necesario, y luego ejecuta los TODOs (manualmente o
delegando a los otros agentes).

---

## `developer` — Agente de Desarrollo

**Propósito:** Implementar funcionalidades siguiendo los ADRs, specs
y el ciclo TDD.

**Responsabilidades:**
- Implementar endpoints, entidades, eventos, patrones.
- Escribir tests antes del código (TDD).
- Respetar arquitectura hexagonal, clean code, guías de estilo Go.
- Mantener specs sincronizadas con el código (spec-first).

**Contexto prioritario:** ADRs 001-007, 010, 011, 013, 016. Specs
OpenAPI, AsyncAPI, patterns. Guías de estilo Go.

**Skills que usa:** `sdlc`, `new-service`, `new-endpoint`, `new-entity`,
`new-event`, `new-pattern`, `new-graph`, `unit-test`.

---

## `qa` — Agente de QA

**Propósito:** Verificar calidad, buscar problemas, generar tests
exhaustivos y validar cumplimiento de specs.

**Responsabilidades:**
- Crear tests unitarios para código existente sin cobertura.
- Crear tests de integración con Testcontainers.
- Validar que endpoints cumplen specs OpenAPI (contract testing).
- Validar que eventos cumplen specs AsyncAPI.
- Ejecutar análisis de seguridad (gosec).
- Identificar tests faltantes, edge cases, race conditions.
- Revisar fuzzing para parsers y validadores.

**Contexto prioritario:** ADRs 002, 004, 010, 011. Specs completas.

**Skills que usa:** `unit-test`, `integration-test`, `contract-test`,
`lint-security`.

---

## `reviewer` — Agente de Code Review

**Propósito:** Revisar PRs automáticamente en CI, verificando
cumplimiento de ADRs, specs, y calidad del código.

**Responsabilidades:**
- Review automático en cada PR vía Claude Code Action.
- Verificar que handlers Gin son delgados (ADR-006).
- Verificar que el dominio no importa infraestructura (ADR-001).
- Verificar que tipos Ent no salen del adaptador (ADR-007).
- Verificar que los tests siguen TDD y son descriptivos (ADR-002).
- Verificar que las funciones ≤20 líneas (ADR-003).
- Sugerir mejoras de rendimiento y legibilidad.

**Contexto prioritario:** Todos los ADRs. Guías de estilo Go.

**Skills que usa:** `ci-claude`.

**Ejecución:** Automático en CI vía `anthropics/claude-code-action`.
No se invoca manualmente.

---

## `docs` — Agente de Documentación

**Propósito:** Crear y mantener la documentación del proyecto
sincronizada con el código y las decisiones.

**Responsabilidades:**
- Mantener las cuatro vistas de documentación (4+1 de Kruchten).
- Generar y actualizar diagramas C4 en Mermaid.
- Mantener la documentación de API (Swagger/Redoc, AsyncAPI Studio).
- Crear y actualizar runbooks de despliegue.
- Escribir guías de usuario.
- Detectar documentación desactualizada.

**Contexto prioritario:** Todos los ADRs. Estructura del monorepo.
Skills catalog. Specs.

**Skills que usa:** `scenarios-view`, `logical-view`, `process-view`,
`development-view`, `physical-view`, `api-docs`, `deploy-docs`,
`user-docs`, `adr`.

---

## `devops` — Agente de DevOps/SRE

**Propósito:** Configurar y mantener infraestructura, contenedores,
CI/CD y entornos de desarrollo.

**Responsabilidades:**
- Crear y mantener Dockerfiles de cada servicio.
- Configurar docker-compose para desarrollo local.
- Crear y mantener workflows de GitHub Actions.
- Configurar variables de entorno y secrets.
- Optimizar pipelines de CI (caché, paralelización).
- Configurar monitorización y alertas.

**Contexto prioritario:** ADRs 005, 007, 008, 013, 014.

**Skills que usa:** `dockerfile`, `docker-compose`, `ci-workflow`,
`ci-claude`, `deploy-docs`.

---

## Flujo de trabajo con agentes

```
Requisito / Historia de usuario
        │
        ▼
   ┌─────────┐
   │ planner │ → Genera lista de TODOs con dependencias y asignaciones
   └────┬────┘
        │
        ├──▶ developer (specs, entities, endpoints, events, impl)
        ├──▶ qa (tests unitarios, integración, contrato, seguridad)
        ├──▶ docs (diagramas, API docs, guías)
        └──▶ devops (Dockerfile, CI, deploy)
                │
                ▼
           ┌──────────┐
           │ reviewer  │ → Review automático del PR en CI
           └──────────┘
                │
                ▼
           Merge a main
```

---

# Parte 2: Skills

## Fundamentos

### `sdlc`
**Estado:** 📋 Planificado
**ADRs:** 005 (GitHub Flow), 003 (Clean Code)
**Agentes:** planner, developer
**Alcance:**
- Crear rama con nomenclatura (`feature/`, `fix/`, `hotfix/`).
- Commits: `<tipo>: <descripción imperativa>`.
- PR: descripción, refs a issues/ADRs, checklist.
- Code review. Squash merge. Limpieza de rama.

### `new-service`
**Estado:** 📋 Planificado
**ADRs:** 013 (Monorepo), 001 (Hexagonal)
**Agentes:** developer, devops
**Alcance:**
- Scaffoldear `services/{nombre}/cmd/main.go` + `internal/`.
- Dockerfile multi-stage. Workflow de deploy.
- Actualizar Makefile y docker-compose.

### `new-endpoint`
**Estado:** 📋 Planificado
**ADRs:** 006 (Gin), 010 (OpenAPI), 002 (TDD)
**Agentes:** developer
**Alcance:**
- Spec-first: definir en `specs/openapi.yaml`.
- Handler Gin delgado + caso de uso en dominio.
- Mapeo errores dominio → HTTP. Tests.

### `new-entity`
**Estado:** 📋 Planificado
**ADRs:** 007 (Ent + Atlas)
**Agentes:** developer
**Alcance:**
- Schema Ent en `ent/schema/`. `go generate ./ent`.
- Migración: `atlas migrate diff`. Revisar SQL.
- Puerto en `libs/ports/`. Adaptador en `adapters/storage/`.

### `new-event`
**Estado:** 📋 Planificado
**ADRs:** 011 (AsyncAPI), 008 (Valkey), 014 (Comunicación)
**Agentes:** developer
**Alcance:**
- Spec-first: definir en `specs/asyncapi.yaml`.
- Streams (negocio) o Pub/Sub (efímero).
- Publisher + consumer idempotente. Consumer group.

---

## Containers

### `dockerfile`
**Estado:** 📋 Planificado
**ADRs:** 013 (Monorepo)
**Agentes:** devops
**Alcance:**
- Multi-stage: build (Go) → runtime (distroless/alpine).
- CGO_ENABLED=0. Caché de go mod download.
- Labels, health check. Un Dockerfile por servicio.

### `docker-compose`
**Estado:** 📋 Planificado
**ADRs:** 007, 008, 013
**Agentes:** devops
**Alcance:**
- PostgreSQL 16 + pgvector, Valkey 8.
- Volúmenes, red interna, `.env.example`.
- Opcionalmente: servicios de dago para testing.

---

## QA / Testing

### `unit-test`
**Estado:** 📋 Planificado
**ADRs:** 002 (TDD), 004 (Go)
**Agentes:** developer, qa
**Alcance:**
- Table-driven tests con subtests descriptivos.
- Fakes in-memory de puertos.
- `go test ./... -short`.

### `integration-test`
**Estado:** 📋 Planificado
**ADRs:** 002, 007, 008
**Herramientas:** Testcontainers for Go
**Agentes:** qa
**Alcance:**
- PostgreSQL y Valkey reales via Testcontainers.
- Build tag `//go:build integration`.
- Flujos end-to-end.

### `contract-test`
**Estado:** 📋 Planificado
**ADRs:** 010 (OpenAPI), 011 (AsyncAPI)
**Agentes:** qa
**Alcance:**
- Validar endpoints contra spec OpenAPI.
- Validar eventos contra schemas AsyncAPI.
- Automatización en CI.

### `lint-security`
**Estado:** 📋 Planificado
**ADRs:** 004, 003
**Herramientas:** golangci-lint, gosec, go test -fuzz
**Agentes:** qa
**Alcance:**
- `.golangci.yml` con linters: errcheck, govet, staticcheck,
  gosimple, bodyclose, contextcheck, gosec.
- Fuzzing para parsers y validadores.
- CI: lint falla → PR no mergeable.

---

## CI/CD

### `ci-workflow`
**Estado:** 📋 Planificado
**ADRs:** 005, 013
**Agentes:** devops
**Alcance:**
- `ci.yaml`: lint → test → build.
- Deploy por servicio con path-based triggers.
- Caché, paralelización, publicación de imágenes Docker.

### `ci-claude`
**Estado:** 📋 Planificado
**ADRs:** 002, 005
**Herramientas:** anthropics/claude-code-action
**Agentes:** reviewer (automático), devops (configuración)
**Alcance:**
- Review automático de PRs contra ADRs y specs.
- Auto-fix de CI failures.
- Issue-to-PR con `@claude`.
- Modelos: Sonnet para reviews, Haiku para triage.

---

## Documentación (modelo 4+1 de Kruchten)

```
docs/
├── views/
│   ├── scenarios/              # +1: Escenarios / casos de uso
│   │   ├── overview.md         # Lista de escenarios principales
│   │   ├── graph-execution.md  # Escenario: ejecutar un grafo
│   │   ├── user-login.md       # Escenario: autenticación
│   │   ├── package-publish.md  # Escenario: publicar paquete
│   │   └── mcp-tool-invoke.md  # Escenario: invocar tool MCP
│   │
│   ├── logical/                # Vista 1: Lógica
│   │   ├── overview.md         # Descomposición en servicios
│   │   ├── domain-model.md     # Entidades, value objects, relaciones
│   │   ├── patterns.md         # Patrones de nodo y flujo
│   │   ├── memory.md           # Arquitectura de memoria (3 capas)
│   │   ├── auth-model.md       # OAuth 2.1 + ABAC
│   │   └── components/         # C4 Level 3 por servicio
│   │       ├── orchestrator.md
│   │       ├── executor.md
│   │       └── ...
│   │
│   ├── process/                # Vista 2: Procesos
│   │   ├── overview.md         # Comunicación entre servicios
│   │   ├── event-flows.md      # Flujos de eventos (Valkey Streams)
│   │   ├── concurrency.md      # Goroutines, semáforos, consumer groups
│   │   ├── graph-lifecycle.md  # State machine de ejecución de grafo
│   │   └── consolidation.md    # Proceso de dreaming (memoria)
│   │
│   ├── development/            # Vista 3: Desarrollo
│   │   ├── overview.md         # Estructura del monorepo
│   │   ├── dependencies.md     # Dependencias entre packages
│   │   ├── build-system.md     # Makefile, go generate, atlas
│   │   ├── coding-standards.md # Resumen de ADRs 003, 004
│   │   └── testing-strategy.md # Pirámide de tests, herramientas
│   │
│   └── physical/               # Vista 4: Física
│       ├── overview.md         # Topología de despliegue
│       ├── infrastructure.md   # PostgreSQL, Valkey, networking
│       ├── containers.md       # Imágenes Docker, registries
│       └── environments.md     # Dev, staging, production
│
├── deploy/                     # Runbooks de despliegue
│   ├── overview.md
│   ├── services/
│   │   ├── orchestrator.md
│   │   └── ...
│   ├── migrations.md
│   └── rollback.md
│
├── user-guide/                 # Guías de uso
│   ├── quickstart.md
│   ├── creating-graphs.md
│   ├── publishing-packages.md
│   ├── registering-mcp.md
│   └── dashboard-guide.md
│
├── api/                        # Documentación de API (generada)
│   ├── rest.md                 # Link a Swagger UI
│   └── events.md               # Link a AsyncAPI Studio
│
├── adr/                        # Architecture Decision Records
│   ├── ADR-001-*.md
│   └── ...
│
└── skills-catalog.md           # Este documento
```

### `scenarios-view` (+1)
**Estado:** 📋 Planificado
**Agentes:** docs, planner
**Alcance:**
- Documentar escenarios principales como flujos end-to-end.
- Cada escenario traza su recorrido por las cuatro vistas.
- Diagramas de secuencia en Mermaid.
- Sirve como test de coherencia: si un escenario no se puede
  trazar limpiamente, algo falta en las otras vistas.

### `logical-view` (Vista 1)
**Estado:** 📋 Planificado
**Agentes:** docs
**Herramientas:** Mermaid
**Alcance:**
- C4 Level 2 (Container): 8 servicios + dashboard + infra.
- C4 Level 3 (Component): internos de cada servicio.
- Modelo de dominio: entidades Ent y relaciones.
- Patrones de nodo y flujo con ejemplos.
- Modelo de autorización ABAC.
- Modelo de memoria (3 capas).

### `process-view` (Vista 2)
**Estado:** 📋 Planificado
**Agentes:** docs
**Herramientas:** Mermaid
**Alcance:**
- Diagramas de secuencia de flujos de eventos.
- State machine del ciclo de vida de un grafo.
- Modelo de concurrencia (goroutines, semáforos).
- Consumer groups y escalado horizontal.
- Flujo de autenticación OAuth 2.1.
- Proceso de consolidación de memoria.

### `development-view` (Vista 3)
**Estado:** 📋 Planificado
**Agentes:** docs
**Alcance:**
- Diagrama de packages y dependencias del monorepo.
- Flujo de build: go generate → atlas migrate → go build.
- Estándares de código (resumen de ADRs).
- Estrategia de testing (pirámide, herramientas).
- Flujo de CI/CD con GitHub Actions.

### `physical-view` (Vista 4)
**Estado:** 📋 Planificado
**Agentes:** docs, devops
**Herramientas:** Mermaid
**Alcance:**
- Diagrama de despliegue: contenedores, infra, redes.
- Requisitos por entorno (dev, staging, production).
- Configuración de PostgreSQL, Valkey.
- Estrategia de alta disponibilidad.

### `api-docs`
**Estado:** 📋 Planificado
**ADRs:** 010, 011
**Agentes:** docs
**Alcance:**
- Swagger UI / Redoc desde OpenAPI.
- AsyncAPI Studio desde AsyncAPI.
- Disponible en dev y staging.

### `deploy-docs`
**Estado:** 📋 Planificado
**ADRs:** 013
**Agentes:** docs, devops
**Alcance:**
- Runbook por servicio: env vars, puertos, dependencias, health.
- Procedimiento de migración Atlas.
- Rollback. Checklist de despliegue.

### `user-docs`
**Estado:** 📋 Planificado
**Agentes:** docs
**Alcance:**
- Quickstart: primer grafo, primera ejecución.
- Crear y publicar paquetes.
- Registrar MCP servers. Usar dashboard.
- Referencia de patrones.

### `adr`
**Estado:** 📋 Planificado
**Agentes:** docs, planner
**Alcance:**
- Plantilla ADR. Cómo proponer, revisar, deprecar.
- Numeración secuencial. Ubicación: `docs/adr/`.

---

## Patrones

### `new-pattern`
**Estado:** 📋 Planificado
**ADRs:** 016
**Agentes:** developer
**Alcance:**
- JSON Schema en `specs/patterns/nodes/` o `edges/`.
- Handler en executor (nodo) u orchestrator (flujo).
- Validación. Documentación. Tests.

### `new-graph`
**Estado:** 📋 Planificado
**ADRs:** 016
**Agentes:** developer
**Alcance:**
- Definir grafo JSON. Validar estructura y schemas.
- Verificar recursos. Publicar en catálogo.

---

## Frontend

### `frontend-feature`
**Estado:** 📋 Planificado
**ADRs:** 009
**Agentes:** developer
**Alcance:**
- Feature module en `dashboard/src/features/{nombre}/`.
- TanStack Query. Vitest + RTL. API client generado.

### `api-client-gen`
**Estado:** 📋 Planificado
**ADRs:** 009, 010
**Herramientas:** openapi-typescript
**Agentes:** developer, devops
**Alcance:**
- Generar tipos TypeScript desde OpenAPI.
- Output en `dashboard/src/api/`. CI verifica sincronización.

---

## Costes

### `model-routing`
**Estado:** 📋 Planificado
**ADRs:** 016
**Agentes:** planner, developer
**Alcance:**
- Opus/GPT-4o: planificación, reflection, code review.
- Sonnet: ejecución estándar, ReAct, tool use.
- Haiku/Flash: clasificación, guardrails, embeddings.
- Config por defecto en cada patrón. Override por grafo/nodo.
- Monitorización de costes por ejecución.
