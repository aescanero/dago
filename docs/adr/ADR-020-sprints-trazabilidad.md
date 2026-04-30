# ADR-020: Desarrollo por sprints reducidos con trazabilidad completa

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El proyecto se desarrolla con asistencia de agentes IA (Claude Code).
Se necesita que el proceso de desarrollo sea completamente trazable y
auditable — que un analista humano o máquina pueda examinar cada sprint,
entender qué se hizo, por qué, y verificar que el resultado cumple las
especificaciones. Esto es especialmente importante cuando el código es
generado o asistido por IA: la transparencia del proceso es tan
relevante como la calidad del resultado.

## Decisión

Se adopta un modelo de **sprints reducidos** donde cada sprint tiene
alcance limitado, documentación explícita, tests derivados de
especificaciones, y trazabilidad completa desde requisito hasta
implementación.

### Estructura de un sprint

Cada sprint se documenta en un fichero Markdown en `docs/sprints/`:

```
docs/sprints/
├── SPRINT-001-bootstrap-monorepo.md
├── SPRINT-002-auth-server-oauth.md
├── SPRINT-003-catalog-packages.md
└── ...
```

### Formato del documento de sprint

```markdown
# SPRINT-XXX: [Título descriptivo]

## Metadata
- **Fecha inicio:** YYYY-MM-DD
- **Fecha fin:** YYYY-MM-DD
- **Estado:** planificado | en progreso | completado | cancelado
- **ADRs aplicados:** ADR-001, ADR-007, ADR-013
- **Specs afectadas:** openapi.yaml (paths/graphs.yaml), asyncapi.yaml
- **Agente planificador:** planner
- **Revisado por:** [humano o agente reviewer]

## Objetivo del sprint
[Descripción clara y concisa de qué se consigue al completar este sprint.
Debe ser verificable — al terminar, se puede comprobar sin ambigüedad si
se cumplió o no.]

## Alcance
### Incluido
- [Lista explícita de lo que se implementa]

### Excluido
- [Lista explícita de lo que NO se implementa y por qué]

## Dependencias
- **Sprints previos requeridos:** SPRINT-XXX, SPRINT-YYY
- **Specs que deben existir:** [specs que se asumen listas]
- **Infraestructura requerida:** [PostgreSQL, Valkey, etc.]

## TODOs

### 1. [spec] Definir schemas en OpenAPI
- **Agente:** developer
- **Skill:** new-endpoint
- **Spec afectada:** specs/paths/graphs.yaml
- **Criterio de aceptación:** El schema es válido y coherente con ADR-010.
- **Depende de:** ninguno

### 2. [test] Tests de contrato para GET /api/v1/graphs
- **Agente:** qa
- **Skill:** contract-test
- **Ubicación del test:** test/contract/graphs_test.go
- **Qué verifica:** Que el endpoint devuelve 200 con el schema definido
  en OpenAPI, 401 sin token, 403 sin tags ABAC adecuadas.
- **Spec de referencia:** specs/paths/graphs.yaml
- **Depende de:** #1

### 3. [test] Tests unitarios del caso de uso CreateGraph
- **Agente:** qa
- **Skill:** unit-test
- **Ubicación del test:** libs/domain/graph/service_test.go
- **Qué verifica:**
  - Grafo válido se crea correctamente.
  - Grafo sin entry_node devuelve error de validación.
  - Grafo con ciclo sin max_iterations es rechazado.
- **Spec de referencia:** specs/patterns/graph.json (validación)
- **Depende de:** ninguno

### 4. [data] Crear schema Ent para Graph
- **Agente:** developer
- **Skill:** new-entity
- **Ubicación:** ent/schema/graph.go
- **Depende de:** ninguno

### 5. [impl] Implementar caso de uso CreateGraph
- **Agente:** developer
- **Skill:** new-endpoint
- **Ubicación:** libs/domain/graph/service.go
- **Cumple test:** #3
- **Depende de:** #3, #4

### 6. [impl] Implementar handler Gin para POST /api/v1/graphs
- **Agente:** developer
- **Skill:** new-endpoint
- **Ubicación:** services/orchestrator/internal/handler/graph_handler.go
- **Cumple test:** #2
- **Depende de:** #2, #5

### 7. [docs] Actualizar diagrama de componentes del orchestrator
- **Agente:** docs
- **Skill:** logical-view
- **Ubicación:** docs/views/logical/components/orchestrator.md
- **Depende de:** #6

## Matriz de trazabilidad

| Spec | Test | Implementación | Ubicación |
|------|------|----------------|-----------|
| openapi.yaml#/paths/graphs | contract/graphs_test.go | handler/graph_handler.go | orchestrator |
| patterns/graph.json | domain/graph/service_test.go | domain/graph/service.go | libs |
| ent/schema/graph.go | (migración Atlas) | adapters/storage/graph_repo.go | adapters |

## Resultado del sprint
[Se completa al finalizar el sprint]

### Tests ejecutados
- Total: X
- Passed: X
- Failed: 0

### Ficheros creados/modificados
[Lista generada automáticamente o manualmente]

### Decisiones tomadas durante el sprint
[Cualquier decisión no prevista que se tomó durante la implementación.
Si es significativa, se propone como ADR.]

### Observaciones del reviewer
[Feedback del agente reviewer o del humano que revisó el sprint]
```

### Reglas concretas

#### Planificación

1. **Sprints reducidos.** Cada sprint tiene un alcance que se puede
   completar en 1-3 días de trabajo. Si el alcance es mayor, se
   divide en sprints más pequeños.

2. **Tests primero en el plan.** Los TODOs de test se definen antes
   que los TODOs de implementación. Cada test referencia la spec de
   la que se deriva y describe exactamente qué verifica.

3. **Alcance explícito.** Se documenta tanto lo incluido como lo
   excluido. "No se implementa autenticación en este sprint porque
   depende de SPRINT-005" es información valiosa.

4. **Dependencias entre TODOs.** Cada TODO indica de qué otros
   depende. Esto define el orden de ejecución y permite
   paralelización cuando no hay dependencias.

5. **Matriz de trazabilidad obligatoria.** Cada spec → test →
   implementación se mapea explícitamente. Si una spec no tiene
   test, es una brecha. Si un test no referencia una spec, no
   debería existir.

#### Ejecución

6. **Test-first dentro del sprint.** Los tests del sprint se
   escriben (Red) antes de la implementación (Green). Los TODOs
   se ejecutan en el orden definido por las dependencias.

7. **Cada TODO se commitea por separado** con un mensaje que
   referencia el sprint y el número de TODO:

   ```
   feat: define graph schemas in OpenAPI [SPRINT-001 #1]
   test: add contract tests for GET /api/v1/graphs [SPRINT-001 #2]
   feat: implement CreateGraph use case [SPRINT-001 #5]
   ```

8. **Un PR por sprint.** El sprint completo se envía como un único
   PR que incluye todos los cambios. El PR referencia el documento
   de sprint.

#### Cierre

9. **Resultado documentado.** Al cerrar el sprint se completa la
   sección de resultado: tests ejecutados, ficheros modificados,
   decisiones no previstas.

10. **Review obligatorio.** El agente `reviewer` (en CI) o un
    humano revisa el sprint completo contra el plan. Las
    discrepancias se documentan.

11. **Sprint cerrado es inmutable.** Una vez completado, el documento
    de sprint no se modifica. Si hay correcciones, se crean en un
    sprint nuevo que referencia al anterior.

### Cómo el agente planner genera sprints

El agente `planner` recibe un requisito y genera el documento de
sprint siguiendo esta secuencia:

```
1. Analizar el requisito contra ADRs y specs.
2. Identificar qué specs se necesitan crear o modificar.
3. Derivar tests de las specs (contract tests, unit tests).
4. Definir la implementación que hará pasar los tests.
5. Identificar documentación afectada.
6. Construir la matriz de trazabilidad.
7. Estimar si cabe en un sprint o hay que dividir.
8. Generar el documento de sprint.
```

El orden de generación de TODOs dentro del sprint es siempre:

```
specs → tests → datos (Ent) → implementación → documentación
```

Esto garantiza que las specs gobiernan los tests, los tests gobiernan
la implementación, y la documentación refleja el resultado — no al revés.

## Alternativas consideradas

- **Sin sprints formales (desarrollo ad-hoc):** Máxima velocidad
  inicial pero cero trazabilidad. Descartado porque la auditabilidad
  del proceso es un requisito.

- **Sprints Scrum clásicos (2 semanas):** Demasiado largos para un
  proyecto asistido por IA donde el throughput es mayor. Los sprints
  reducidos (1-3 días) se adaptan mejor al ritmo de trabajo con
  Claude Code.

- **Kanban sin sprints:** Flujo continuo. Descartado porque pierde
  la noción de "unidad de trabajo completada y revisada" que es
  esencial para la trazabilidad.

- **Solo issues de GitHub:** Posible pero no captura el razonamiento
  del plan ni la matriz de trazabilidad. Los issues se pueden usar
  como complemento para tracking.

## Consecuencias

**Positivas:**
- Trazabilidad completa: spec → test → código → documentación.
- Cualquier analista puede reconstruir el proceso de construcción.
- Tests derivados de specs, no de la implementación.
- Decisiones no previstas se documentan explícitamente.
- Sprints reducidos permiten revisión frecuente y corrección temprana.
- El agente planner genera planes auditables y reproducibles.

**Negativas:**
- Overhead de documentación por sprint (mitigado porque el planner
  lo genera automáticamente).
- Rigidez: cambios durante el sprint requieren actualizar el plan.
- Acumulación de documentos de sprint (mitigable con archivado).

## Notas para Claude Code

- Los documentos de sprint viven en `docs/sprints/`.
- El agente `planner` genera el documento de sprint completo.
- Orden de TODOs: specs → tests → datos → implementación → docs.
- Cada commit referencia sprint y número de TODO.
- Un PR por sprint. Review obligatorio antes de merge.
- La matriz de trazabilidad es obligatoria en cada sprint.
- Al cerrar el sprint, completar la sección de resultado.
