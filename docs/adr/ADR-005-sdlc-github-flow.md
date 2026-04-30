# ADR-005: GitHub Flow, versionado semántico y changelog

**Estado:** Aceptado (revisado: changelog, releases, trazabilidad)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El equipo necesita un flujo de trabajo estandarizado para el ciclo de
vida del desarrollo. Adicionalmente, tras la adopción de sprints con
trazabilidad (ADR-020), se necesita una cadena completa desde el
requisito hasta la release:

```
ADR → Spec → Sprint → TODO → Commit → PR → Changelog → Release → Tag
```

## Decisión

Se adopta **GitHub Flow** como estrategia de branching, **Conventional
Commits** como formato estricto de commits, **Semantic Versioning**
como esquema de versionado, y **generación automática de changelog**
como mecanismo de comunicación de cambios.

### GitHub Flow

```
main (siempre desplegable)
  │
  ├── sprint/001-bootstrap-monorepo    ← rama de sprint
  ├── feature/add-graph-validation     ← rama de funcionalidad
  ├── fix/shipping-cost-calculation    ← rama de corrección
  └── hotfix/critical-auth-bypass      ← rama de corrección urgente
```

### Reglas concretas

#### Ramas y PRs

1. **`main` es sagrada.** Siempre desplegable. Protegida: sin push
   directo. Todo entra vía PR aprobado.

2. **Una rama por sprint o cambio.** Las ramas son efímeras — se
   eliminan tras merge.

3. **Nomenclatura de ramas:**

   ```
   sprint/<NNN>-<descripción>     → sprints planificados (ADR-020)
   feature/<descripción>          → funcionalidades fuera de sprint
   fix/<descripción>              → correcciones no urgentes
   hotfix/<descripción>           → correcciones críticas en producción
   refactor/<descripción>         → mejoras sin cambio funcional
   docs/<descripción>             → cambios en documentación
   ```

4. **Un PR por sprint.** El sprint completo se envía como un PR que
   referencia el documento de sprint. El PR incluye: descripción del
   sprint, enlace al documento, y la matriz de trazabilidad.

5. **Protecciones de `main`:**
   - Require pull request reviews (mínimo 1).
   - Require status checks to pass (lint, test, build).
   - Require branches to be up to date.
   - No force pushes. No deletions.

#### Conventional Commits (estricto)

6. **Formato obligatorio de commits:**

   ```
   <tipo>(<scope>): <descripción imperativa> [SPRINT-XXX #N]

   [cuerpo opcional con contexto]

   [Refs: #issue, ADR-xxx]
   [BREAKING CHANGE: descripción del breaking change]
   ```

   **Tipos:**

   | Tipo | Uso | Aparece en changelog |
   |------|-----|---------------------|
   | `feat` | Nueva funcionalidad | ✅ (Features) |
   | `fix` | Corrección de bug | ✅ (Bug Fixes) |
   | `perf` | Mejora de rendimiento | ✅ (Performance) |
   | `refactor` | Refactorización sin cambio funcional | ❌ |
   | `test` | Añadir o modificar tests | ❌ |
   | `docs` | Documentación | ❌ |
   | `ci` | Cambios en CI/CD | ❌ |
   | `chore` | Tareas de mantenimiento | ❌ |
   | `build` | Cambios en build system | ❌ |

   **Scopes:** nombre del servicio o componente afectado:
   `orchestrator`, `executor`, `router`, `planner`, `auth-server`,
   `catalog`, `mcp-registry`, `agent-registry`, `dashboard`, `libs`,
   `adapters`, `specs`, `docs`.

   **Ejemplos:**

   ```
   feat(orchestrator): add graph validation endpoint [SPRINT-001 #6]
   fix(executor): handle LLM timeout in react pattern [SPRINT-003 #4]
   test(libs): add unit tests for ABAC tag evaluation [SPRINT-002 #3]
   docs(views): update logical view with auth-server components [SPRINT-002 #7]

   feat(openapi)!: rename /graphs to /workflows [SPRINT-005 #1]

   BREAKING CHANGE: All /api/v1/graphs endpoints are now /api/v1/workflows.
   Clients must update their base URLs.
   ```

7. **El scope referencia al sprint y TODO.** Esto cierra la cadena
   de trazabilidad: desde el changelog se llega al commit, del commit
   al sprint, del sprint a la spec, de la spec al ADR.

8. **`BREAKING CHANGE` en el footer** para cambios incompatibles.
   Esto dispara un bump de versión major en semver.

#### Squash merge

9. **Squash and merge como estrategia.** Los PRs se mergean con
   squash. El mensaje del squash sigue Conventional Commits y
   resume el sprint:

   ```
   feat(orchestrator): implement graph CRUD and validation [SPRINT-001]

   - Define graph schemas in OpenAPI
   - Add contract tests for graph endpoints
   - Implement CreateGraph, GetGraph, ListGraphs use cases
   - Create Ent schema and migration for Graph entity

   Refs: SPRINT-001, ADR-007, ADR-010, ADR-016
   ```

#### Semantic Versioning

10. **Versionado semántico del sistema:**

    ```
    vMAJOR.MINOR.PATCH

    MAJOR → Breaking change en la API pública (OpenAPI)
    MINOR → Nuevo endpoint, nuevo patrón, nueva funcionalidad
    PATCH → Corrección de bugs, mejoras de rendimiento
    ```

11. **Dos niveles de versionado:**

    - **Sistema (dago):** Versión global que refleja la API pública.
      Tag: `v1.2.3`.
    - **Paquetes del catálogo:** Cada paquete tiene su propio semver
      independiente (ADR-017). Versión del paquete ≠ versión del sistema.

12. **El versionado se deriva de los commits.** Un commit `feat:` →
    bump minor. Un commit `fix:` → bump patch. Un commit con
    `BREAKING CHANGE` → bump major. No se elige la versión
    manualmente.

#### Changelog

13. **Changelog generado automáticamente** desde los commits con
    Conventional Commits. Se usa **git-cliff** (open source, Rust,
    configurable) o equivalente.

14. **Formato del changelog:**

    ```markdown
    # Changelog

    ## [1.2.0] - 2026-05-15

    ### Features
    - **orchestrator:** Add graph validation endpoint ([SPRINT-001 #6])
    - **catalog:** Implement package publishing API ([SPRINT-002 #5])

    ### Bug Fixes
    - **executor:** Handle LLM timeout in react pattern ([SPRINT-003 #4])

    ### Performance
    - **adapters:** Optimize Valkey connection pooling ([SPRINT-004 #2])

    ### Breaking Changes
    - **openapi:** Rename /graphs to /workflows ([SPRINT-005 #1])

    ## [1.1.0] - 2026-05-01
    ...
    ```

15. **Cada entrada referencia el sprint y TODO.** Un auditor puede
    seguir la cadena: changelog entry → commit → sprint doc →
    spec → ADR.

16. **El changelog vive en `CHANGELOG.md` en la raíz** del repo.
    Se actualiza automáticamente en cada release.

#### Releases

17. **Release = tag + changelog + GitHub Release.** Cuando se decide
    hacer una release:

    ```
    1. git-cliff genera el changelog actualizado.
    2. Se commitea CHANGELOG.md.
    3. Se crea tag semver (v1.2.0).
    4. GitHub Release se genera desde el tag con el changelog.
    5. CI/CD despliega los servicios afectados.
    ```

18. **Frecuencia de releases.** No se acumulan cambios indefinidamente.
    Se hace release cuando se completa un sprint con funcionalidad
    significativa o cuando hay fixes críticos. Mínimo una release
    por sprint productivo.

19. **Releases de servicios individuales.** Aunque hay una versión
    global del sistema, cada servicio puede tener releases
    independientes vía path-based triggers (ADR-013). El changelog
    global refleja todos los cambios; los deploy son selectivos.

### Cadena completa de trazabilidad

```
ADR-016 (decisión: patrones de nodo)
  → specs/patterns/nodes/react.json (spec del patrón)
    → SPRINT-003 (plan: implementar patrón react)
      → SPRINT-003 #2 (TODO: tests del patrón react)
        → test(executor): add react pattern tests [SPRINT-003 #2] (commit)
      → SPRINT-003 #4 (TODO: implementar handler react)
        → feat(executor): implement react pattern handler [SPRINT-003 #4] (commit)
          → PR "SPRINT-003: Implement react pattern" (review)
            → CHANGELOG.md: "**executor:** Implement react pattern" (comunicación)
              → v1.3.0 (release)
                → Tag + GitHub Release + Deploy executor
```

Cualquier analista puede recorrer esta cadena en ambas direcciones:
desde el ADR hasta la release, o desde la release hasta el ADR.

### Integración con sprints (ADR-020)

```
1. planner genera documento de sprint con TODOs.
2. Se crea rama sprint/NNN-descripción desde main.
3. Se ejecutan los TODOs en orden (specs → tests → impl → docs).
4. Cada commit sigue Conventional Commits con [SPRINT-NNN #TODO].
5. Se abre PR con referencia al documento de sprint.
6. reviewer (CI o humano) revisa contra plan y ADRs.
7. Squash merge con mensaje resumen del sprint.
8. git-cliff actualiza CHANGELOG.md.
9. Se crea release si corresponde (tag + GitHub Release).
10. Se cierra el documento de sprint con resultado.
```

## Alternativas consideradas

- **GitFlow:** Descartado por complejidad innecesaria.
- **Trunk-Based Development:** Considerado como evolución futura.
- **Changelog manual:** Propenso a errores y olvidos. Descartado.
- **release-please (Google):** Alternativa a git-cliff. Genera PRs
  de release automáticamente. Más opinionado. Se evaluará.

## Consecuencias

**Positivas:**
- Trazabilidad completa: ADR → spec → sprint → commit → changelog → release.
- Changelog generado, no escrito — siempre actualizado.
- Versionado derivado de commits — sin decisiones manuales.
- Auditable por humanos y máquinas.
- Conventional Commits como estándar de la industria.

**Negativas:**
- Conventional Commits requiere disciplina en cada commit.
- Tooling adicional (git-cliff) en el pipeline.
- El squash merge pierde granularidad del sprint en el historial de
  main (mitigado: el PR y el documento de sprint preservan el detalle).

## Notas para Claude Code

- Formato de commits: `<tipo>(<scope>): <desc> [SPRINT-XXX #N]`.
- Tipos que van al changelog: `feat`, `fix`, `perf`.
- Scope: nombre del servicio o componente.
- `BREAKING CHANGE` en el footer para cambios incompatibles.
- Una rama y un PR por sprint.
- El mensaje del squash merge resume el sprint completo.
- Nunca sugieras commits directos a `main`.
- Si un cambio es >400 líneas, sugiere dividir en sprints más pequeños.
- El changelog se genera con git-cliff — no se edita manualmente.
