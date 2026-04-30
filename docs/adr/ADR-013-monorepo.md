# ADR-013: Monorepo con un solo módulo Go

**Estado:** Aceptado (revisado: 8 servicios tras descomposición)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El sistema dago consta de 8 servicios backend en Go y 1 frontend React.
Originalmente eran repos separados con cadena de dependencias libs ← adapters
← servicios. Cualquier cambio en libs generaba 6+ PRs y drift de versiones.

## Decisión

Se adopta **monorepo con un solo módulo Go** (`go.mod` en raíz).

### Servicios

| Servicio | Tipo | Responsabilidad |
|----------|------|-----------------|
| orchestrator | Eventos + HTTP | Core: grafos, estado, coordinación, API, WebSocket |
| executor | Eventos | Worker: llm_call, tool_use, react, reflection, guardrail |
| router | Eventos | Worker: deterministic, llm, hybrid |
| planner | Eventos | NL → grafo |
| auth-server | HTTP | OAuth 2.1, Identity Broker, ABAC |
| catalog | HTTP | Catálogo de paquetes, versionado |
| mcp-registry | HTTP | Registry + broker MCP |
| agent-registry | HTTP | Agent Cards A2A, discovery |

### Reglas concretas

1. **Un solo `go.mod` en la raíz.** Módulo: `github.com/org/dago`.
   Imports: `github.com/org/dago/libs/...`, `github.com/org/dago/adapters/...`.

2. **No hay versiones internas.** Servicios consumen libs y adapters
   del mismo commit. Sin `replace` directives.

3. **`internal/` por servicio.** Garantiza encapsulación:
   `services/executor/internal/` no es importable por otros servicios.

4. **Un binario y un Dockerfile por servicio.**

5. **Path-based CI/CD triggers:**

   ```yaml
   on:
     push:
       branches: [main]
       paths:
         - 'services/executor/**'
         - 'libs/**'
         - 'adapters/**'
         - 'go.mod'
   ```

6. **CI unificado** compila, lintea y testea todo el monorepo.

7. **Dashboard independiente** con su propio `package.json` y pipeline.

8. **Makefile como interfaz unificada:** build-all, build-{service},
   test, lint, generate, migrate-diff, migrate-apply, dashboard-dev.

9. **Docker Compose** para PostgreSQL + Valkey en desarrollo local.

10. **Un solo directorio `ent/` compartido.** Modelo de datos
    centralizado. Si un servicio necesita DB propia, crea `ent/`
    dentro de su `internal/`.

11. **Un solo directorio `migrations/`.** Atlas contra schema Ent.

## Notas para Claude Code

- Nunca crees `go.mod` dentro de un servicio.
- Código interno: `services/{nombre}/internal/`.
- Código compartido: `libs/` o `adapters/`.
- Nuevo servicio: `services/{nombre}/cmd/main.go`, `internal/`, `Dockerfile`.
- Frontend: `dashboard/` con `package.json` propio.
