# CLAUDE.md — Instrucciones para Claude Code

## Proyecto: dago

Sistema de orquestación de grafos de IA con nodos agénticos (LLM call,
ReAct, reflection, tool use, routing, guardrails, subgraphs), comunicación
por eventos, memoria de tres capas, catálogo de paquetes, registros MCP y
A2A, auth OAuth 2.1 con ABAC, y dashboard con microfrontales.

## Principios

**Spec Driven Development.** Cuatro specs como fuentes de verdad:

```
specs/openapi.yaml          → API REST
specs/asyncapi.yaml         → Eventos (Valkey Streams)
specs/patterns/             → Patrones de nodo y flujo (JSON Schema)
ent/schema/                 → Modelo de datos (Ent)
```

Si hay discrepancia entre spec y código, la spec tiene razón.
Todo **open source** (BSD-3, MIT, Apache 2.0, PostgreSQL License).

## Navegación del proyecto

- **`docs/index.md`** — Mapa navegable de todo el proyecto. Leer primero
  para orientarse antes de profundizar en ficheros específicos.
- **`docs/log.md`** — Historial cronológico append-only de operaciones.
  Consultar para saber qué se hizo recientemente.
- **`CHANGELOG.md`** — Historial de releases (generado por git-cliff).

Cuando se compacta el contexto, preservar siempre: la lista de ficheros
modificados, los comandos de test ejecutados, y el sprint activo.

## Stack

| Capa | Tecnología |
|------|-----------|
| Backend | Go · Gin |
| ORM + Migraciones | Ent + Atlas CLI |
| Base de datos | PostgreSQL + pgvector |
| Eventos/Caché/Sesiones | Valkey |
| Frontend | React 19 · TypeScript · Vite |
| Design System | shadcn/ui · Tailwind CSS |
| Microfrontales | Module Federation (@originjs/vite-plugin-federation) |
| Protocolo agente↔usuario | AG-UI (sobre WebSocket) |
| UI generativa | A2UI (declarativa) |
| Auth | OAuth 2.1 (PKCE + Client Credentials) + ABAC |
| Grafos (validación) | dominikbraun/graph |
| Repo | Monorepo · un solo módulo Go |

## Servicios (8 backend + 1 frontend)

```
Orquestación (eventos Valkey Streams):
├── orchestrator       Core: grafos, estado, eventos, API HTTP, WebSocket AG-UI
├── executor           Worker: llm_call, tool_use, react, reflection, guardrail
├── router             Worker: deterministic, llm, hybrid
└── planner            NL → grafo

Soporte (API HTTP):
├── auth-server        OAuth 2.1 + ABAC + Identity Broker
├── catalog            Catálogo de paquetes (workflow + config + UI + versionado)
├── mcp-registry       Registry + broker de MCP servers
└── agent-registry     Agent Cards A2A + discovery de agentes activos

Frontend:
└── dashboard          React 19 + TypeScript + Vite + shadcn/ui + Module Federation
```

## ADRs

Consulta `/docs/adr/` antes de generar código. Son vinculantes.

| ADR | Decisión |
|-----|----------|
| 001 | Arquitectura Hexagonal (puertos en libs/ports/, adaptadores en adapters/) |
| 002 | TDD (Red-Green-Refactor) |
| 003 | Clean Code (funciones ≤20 líneas, parámetros ≤3) |
| 004 | Go + guías de estilo (Effective Go, Google, Uber, Code Review Comments) |
| 005 | GitHub Flow (PRs, squash merge, commits convencionales) |
| 006 | Gin (5 servicios HTTP: orchestrator, auth-server, catalog, mcp-registry, agent-registry) |
| 007 | PostgreSQL + Ent + Atlas (schema as code, migraciones automáticas) |
| 008 | Valkey (eventos Streams/Pub-Sub, caché, sesiones — BSD-3) |
| 009 | React 19 + TypeScript + Vite (dashboard con shadcn/ui, AG-UI, Module Federation) |
| 010 | URL path versioning (/api/v1/) + OpenAPI 3.1 |
| 011 | AsyncAPI 3.0 + Event-Carried State Transfer (Valkey Streams) |
| 012 | OAuth 2.1 propio (auth-server) + ABAC por etiquetas + herencia UO |
| 013 | Monorepo (un solo go.mod, 8 servicios, path-based CI/CD) |
| 014 | Comunicación: eventos (orquestación) + HTTP (soporte) |
| 015 | Memoria de agentes (working + episodic + semantic + consolidación) |
| 016 | Patrones de orquestación (5 flujo + 7 nodo, JSON Schema) |
| 017 | Paquetes como unidad de distribución (workflow + skills + tools + UI) |
| 018 | AG-UI (agente↔usuario sobre WebSocket) + A2UI (UI generativa declarativa) |
| 019 | shadcn/ui + Tailwind CSS (design system) + Module Federation (microfrontales) |
| 020 | Sprints reducidos con trazabilidad completa (spec → test → impl → docs) |

## Protocolos agénticos

```
MCP     → Agente ↔ Herramientas   (executor → mcp-registry → tools)
A2A     → Agente ↔ Agente         (agent-registry → Agent Cards)
AG-UI   → Agente ↔ Usuario        (orchestrator → dashboard, WebSocket)
A2UI    → Agente → UI widgets     (nodo → dashboard, JSON declarativo)
```

## Reglas críticas

**Monorepo:** Un solo `go.mod`. Imports: `github.com/org/dago/libs/...`.
Código interno: `services/{nombre}/internal/`. Compartido: `libs/`, `adapters/`.

**Arquitectura:** El dominio (`libs/`) no importa infraestructura.
Tipos Ent no salen de `adapters/storage/`. Handlers Gin delgados.

**Comunicación:**
- Orquestación (orchestrator ↔ executor/router/planner): solo eventos Valkey.
- Soporte (→ catalog, auth-server, mcp-registry, agent-registry): HTTP.
- Dashboard → orchestrator: API REST + WebSocket (AG-UI).
- Nunca HTTP entre orchestrator ↔ executor/router/planner.

**Patrones de flujo (aristas):** sequential, conditional, parallel, loop, interrupt.
**Patrones de nodo (vértices):** llm_call, tool_use, react, reflection,
router, guardrail, subgraph. JSON Schema en `specs/patterns/`.

**Memoria:**
- Working (ejecución activa): Ent + Valkey.
- Episodic (historial): Ent.
- Semantic (conocimiento destilado): Ent + pgvector.
- Consolidación: background post-ejecución. Nunca borrar, solo superseder.

**Auth:** OAuth 2.1 PKCE (usuarios) + Client Credentials (M2M).
ABAC: tags recurso ⊆ tags efectivas sujeto. Herencia por árbol de UO.
Tokens JWT validados localmente (JWKS). Propagados en eventos (campo `auth`).

**Paquetes:** Unidad de distribución: workflow + skills + tools + UI.
Semver. Publicados en catalog. Validados antes de ejecutar.

**Datos:** Schemas Ent en `ent/schema/`. `go generate ./ent` + `atlas migrate diff`.

**Frontend:** TypeScript strict. shadcn/ui + Tailwind. TanStack Query.
Module Federation para microfrontales de paquetes. Tokens en memoria.

**Código:** TDD. `fmt.Errorf("ctx: %w", err)`. `goimports` + `golangci-lint`.

**Proceso:** PRs. Squash merge a `main`. Commits: `<tipo>: <descripción>`.

## Guías de estilo Go

1. Effective Go: https://go.dev/doc/effective_go
2. Google Go Style Guide: https://google.github.io/styleguide/go/
3. Uber Go Style Guide: https://github.com/uber-go/guide/blob/master/style.md
4. Go Code Review Comments: https://go.dev/wiki/CodeReviewComments

## Estructura inicial del proyecto

```
dago/
├── CLAUDE.md
│
├── docs/
│   ├── skills-catalog.md
│   ├── adr/
│   │   ├── ADR-001 a ADR-020
│   │   └── ...
│   ├── sprints/                    # Documentos de sprint (trazabilidad)
│   └── views/                      # Documentación 4+1 (Kruchten)
│       ├── scenarios/              # +1: Casos de uso
│       ├── logical/                # Vista 1: Componentes, dominio
│       ├── process/                # Vista 2: Flujos, concurrencia
│       ├── development/            # Vista 3: Código, build
│       └── physical/              # Vista 4: Infra, despliegue
│
└── specs/
    ├── openapi.yaml                # API REST (estructura inicial)
    ├── asyncapi.yaml               # Eventos (estructura inicial)
    ├── paths/
    ├── schemas/
    └── patterns/
        ├── graph.json
        ├── edges/
        │   ├── sequential.json
        │   ├── conditional.json
        │   ├── parallel.json
        │   ├── loop.json
        │   └── interrupt.json
        └── nodes/
            ├── llm_call.json
            ├── tool_use.json
            ├── react.json
            ├── reflection.json
            ├── router.json
            ├── guardrail.json
            └── subgraph.json
```

## Estructura objetivo (tras implementación)

```
dago/
├── go.mod · CLAUDE.md · Makefile · docker-compose.yml · atlas.hcl · .golangci.yml
├── docs/ (adr/ · views/ · deploy/ · user-guide/ · api/ · skills-catalog.md)
├── specs/ (openapi.yaml · asyncapi.yaml · paths/ · schemas/ · patterns/)
├── ent/schema/ · migrations/
├── libs/ (domain/ · ports/ · schemas/ · utils/)
├── adapters/ (llm/ · eventbus/ · storage/ · metrics/ · auth/)
├── services/
│   ├── orchestrator/ · executor/ · router/ · planner/
│   ├── auth-server/ · catalog/ · mcp-registry/ · agent-registry/
├── dashboard/ (src/ con api/ auth/ features/ components/ hooks/ pages/)
└── .github/workflows/ (ci.yaml + deploy-{service}.yaml × 8 + deploy-dashboard.yaml)
```

## Flujo de trabajo

1. El agente `planner` genera el documento de sprint en `docs/sprints/`.
2. El sprint define: objetivo, alcance, TODOs con dependencias, matriz de trazabilidad.
3. Orden de TODOs: specs → tests (Red) → datos (Ent) → implementación (Green) → refactor → docs.
4. Cada commit referencia sprint y TODO: `feat: descripción [SPRINT-XXX #N]`.
5. Un PR por sprint. Review obligatorio. Squash merge a `main`.
6. Al cerrar, se completa la sección de resultado del documento de sprint.

## Agentes de Claude Code

| Agente | Propósito |
|--------|-----------|
| planner | Descompone requisitos en TODOs con dependencias y asignaciones |
| developer | Implementa specs, código, tests (TDD) |
| qa | Tests unitarios, integración, contrato, seguridad |
| reviewer | Review automático de PRs en CI (Claude Code Action) |
| docs | Documentación 4+1, API docs, runbooks, guías |
| devops | Dockerfiles, docker-compose, CI/CD, despliegue |
