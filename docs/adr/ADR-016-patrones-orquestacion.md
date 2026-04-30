# ADR-016: Modelo de orquestación por grafos — patrones de flujo y de nodo

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

Dago orquesta workflows de IA modelados como grafos dirigidos, siguiendo
el modelo de LangGraph. Los vértices del grafo son nodos que ejecutan
comportamientos agénticos. Las aristas definen cómo fluye la ejecución.

Se necesita una taxonomía formal de los patrones disponibles, separando
claramente los patrones de **control de flujo** (aristas — cómo se mueve
la ejecución) de los patrones de **comportamiento de nodo** (vértices —
qué hace un nodo).

Cada patrón se define con un JSON Schema que describe sus parámetros
configurables. Los schemas viven en `specs/patterns/` como la cuarta
spec del sistema.

## Decisión

### Librería de grafos

Se usa **`dominikbraun/graph`** (Go, Apache 2.0, zero dependencies)
para la validación estructural del grafo: detección de ciclos,
conectividad, orden topológico. La librería no ejecuta el grafo —
solo lo valida. La ejecución la coordina el orchestrator vía eventos.

### Taxonomía de patrones

```
┌─────────────────────────────────────────────────────────────┐
│                    GRAFO DE EJECUCIÓN                       │
│                                                             │
│   Patrones de control de flujo (aristas)                    │
│   ──────────────────────────────────────                    │
│   Definen CÓMO se mueve la ejecución entre nodos            │
│                                                             │
│   • sequential    — A → B lineal                            │
│   • conditional   — Branch según estado (regla o LLM)       │
│   • parallel      — Fan-out / fan-in                        │
│   • loop          — Repetir hasta condición de salida       │
│   • interrupt     — Pausar y esperar input externo          │
│                                                             │
│   Patrones de comportamiento de nodo (vértices)             │
│   ─────────────────────────────────────────────             │
│   Definen QUÉ hace un nodo cuando le toca ejecutar          │
│                                                             │
│   • llm_call      — Prompt → LLM → respuesta               │
│   • tool_use      — Invocar herramientas (MCP, APIs)        │
│   • react         — Loop interno think → act → observe      │
│   • reflection    — Generar → criticar → mejorar            │
│   • router        — Decidir siguiente nodo (regla/LLM)      │
│   • guardrail     — Validar input/output contra reglas      │
│   • subgraph      — Ejecutar un grafo completo como nodo    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Patrones de control de flujo (aristas)

### `sequential`

La arista más básica: un nodo sigue a otro sin condición.

```json
{
  "type": "sequential",
  "from": "node_a",
  "to": "node_b"
}
```

### `conditional`

La ejecución toma un camino u otro según una condición evaluada
sobre el estado del grafo.

```json
{
  "type": "conditional",
  "from": "node_classifier",
  "conditions": [
    {
      "expression": "state.variables.sentiment == 'positive'",
      "target": "node_positive_response"
    },
    {
      "expression": "state.variables.sentiment == 'negative'",
      "target": "node_escalate"
    }
  ],
  "default": "node_fallback"
}
```

Modos de evaluación:
- **rule** — Expresión determinística sobre el estado.
- **llm** — Un LLM decide el camino basándose en el estado.
- **hybrid** — Primero intenta regla, si no aplica, usa LLM.

### `parallel`

Fan-out: divide la ejecución en N ramas simultáneas. Fan-in: espera
a que completen según una política (all, any, N-of-M).

```json
{
  "type": "parallel",
  "from": "node_start",
  "branches": ["node_research", "node_analyze", "node_summarize"],
  "join": {
    "target": "node_combine",
    "policy": "all",
    "timeout_seconds": 120
  }
}
```

Políticas de join:
- **all** — Espera a que todas las ramas completen.
- **any** — Continúa cuando la primera rama completa (cancela el resto).
- **n_of_m** — Continúa cuando N de M ramas completan.

### `loop`

La ejecución vuelve a un nodo anterior hasta cumplir una condición
de salida. Criterio de parada **obligatorio** para evitar loops
infinitos.

```json
{
  "type": "loop",
  "from": "node_reviewer",
  "target": "node_writer",
  "exit_condition": {
    "expression": "state.variables.review_score >= 0.8",
    "max_iterations": 5,
    "timeout_seconds": 300
  },
  "exit_target": "node_publish"
}
```

Validación: el orchestrator rechaza grafos con loops sin
`max_iterations` o `timeout_seconds`.

### `interrupt`

La ejecución se pausa y persiste el estado. Se reanuda cuando llega
un input externo (humano vía dashboard, webhook, o sistema externo).

```json
{
  "type": "interrupt",
  "from": "node_proposal",
  "resume_target": "node_execute",
  "reject_target": "node_revise",
  "prompt": "¿Aprobar esta acción?",
  "timeout_seconds": 86400,
  "timeout_target": "node_cancel"
}
```

---

## Patrones de comportamiento de nodo (vértices)

### `llm_call`

El patrón más básico: envía un prompt a un LLM y devuelve la respuesta.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/llm_call.json",
  "pattern": "llm_call",
  "config": {
    "model": "claude-sonnet-4-20250514",
    "system_prompt": "You are a helpful assistant specialized in...",
    "temperature": 0.7,
    "max_tokens": 2048,
    "input_mapping": {
      "user_message": "state.messages[-1].content"
    },
    "output_mapping": {
      "state.variables.response": "output.content"
    }
  }
}
```

### `tool_use`

El nodo invoca herramientas externas. Puede ser directo (tool
específica) o con selección LLM (el LLM decide qué tool usar).

```json
{
  "$schema": "https://dago.dev/schemas/patterns/tool_use.json",
  "pattern": "tool_use",
  "config": {
    "mode": "llm_selected",
    "tools": [
      {
        "type": "mcp",
        "server": "github-mcp",
        "allowed_tools": ["search_repos", "read_file"]
      },
      {
        "type": "api",
        "endpoint": "https://api.example.com/search",
        "method": "GET"
      }
    ],
    "model": "claude-sonnet-4-20250514",
    "max_tool_calls": 5,
    "timeout_seconds": 30
  }
}
```

Modos:
- **direct** — Invoca una tool específica sin intervención LLM.
- **llm_selected** — El LLM elige qué tool usar según el contexto.

### `react`

Loop interno de reasoning + acting. Think → Act (tool) → Observe →
Repeat. Es el patrón agéntico más canónico.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/react.json",
  "pattern": "react",
  "config": {
    "model": "claude-sonnet-4-20250514",
    "system_prompt": "You are an agent that reasons step by step...",
    "tools": [
      {"type": "mcp", "server": "web-search"},
      {"type": "mcp", "server": "calculator"}
    ],
    "max_iterations": 10,
    "stop_condition": "final_answer",
    "timeout_seconds": 120,
    "thought_visibility": "full"
  }
}
```

`thought_visibility`:
- **full** — Pensamientos visibles en el output (debugging).
- **final_only** — Solo la respuesta final.

### `reflection`

Genera una salida, la evalúa críticamente, y la mejora iterativamente.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/reflection.json",
  "pattern": "reflection",
  "config": {
    "generator": {
      "model": "claude-sonnet-4-20250514",
      "system_prompt": "Generate a detailed analysis of..."
    },
    "critic": {
      "model": "claude-sonnet-4-20250514",
      "system_prompt": "Evaluate the following analysis for accuracy...",
      "criteria": ["accuracy", "completeness", "clarity"]
    },
    "max_iterations": 3,
    "acceptance_threshold": 0.85,
    "output_mapping": {
      "state.variables.analysis": "output.final_version"
    }
  }
}
```

El generador y el crítico pueden usar modelos diferentes.

### `router`

Analiza el input y produce una decisión de routing. No genera contenido
— decide el camino. Es distinto del `conditional` de flujo: el
conditional evalúa el estado, el router usa lógica compleja (LLM o
reglas) para decidir.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/router.json",
  "pattern": "router",
  "config": {
    "mode": "hybrid",
    "rules": [
      {
        "condition": "state.variables.language == 'es'",
        "route": "spanish_handler"
      }
    ],
    "llm_fallback": {
      "model": "claude-sonnet-4-20250514",
      "system_prompt": "Based on the input, decide which handler...",
      "routes": ["technical_support", "billing", "general"]
    }
  }
}
```

Modos:
- **deterministic** — Solo reglas.
- **llm** — Solo LLM decide.
- **hybrid** — Reglas primero, LLM si ninguna aplica.

### `guardrail`

Valida input o output contra reglas. No genera contenido. Acepta
o rechaza, opcionalmente con razón.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/guardrail.json",
  "pattern": "guardrail",
  "config": {
    "mode": "input",
    "checks": [
      {
        "type": "json_schema",
        "schema": {"$ref": "#/components/schemas/OrderRequest"}
      },
      {
        "type": "llm_safety",
        "model": "claude-sonnet-4-20250514",
        "policy": "Reject if the input contains PII or harmful content"
      },
      {
        "type": "regex",
        "pattern": "^(?!.*\\b(DROP|DELETE|TRUNCATE)\\b).*$",
        "description": "No SQL injection"
      }
    ],
    "on_fail": "reject",
    "on_fail_target": "node_error_handler"
  }
}
```

Modos: **input** (valida antes de ejecutar) o **output** (valida
después de generar).

Tipos de check: **json_schema**, **llm_safety**, **regex**,
**custom_function**.

### `subgraph`

Encapsula un grafo completo como un nodo atómico del grafo padre.
Permite composición jerárquica.

```json
{
  "$schema": "https://dago.dev/schemas/patterns/subgraph.json",
  "pattern": "subgraph",
  "config": {
    "graph_id": "uuid-of-child-graph",
    "input_mapping": {
      "child_state.query": "parent_state.variables.sub_task"
    },
    "output_mapping": {
      "parent_state.variables.sub_result": "child_state.variables.result"
    },
    "timeout_seconds": 600,
    "on_failure": "propagate"
  }
}
```

`on_failure`:
- **propagate** — El padre falla si el subgrafo falla.
- **isolate** — El padre continúa con un resultado vacío/error.

---

## Definición de un grafo completo

Un grafo combina nodos (con sus patrones de comportamiento) y aristas
(con sus patrones de flujo):

```json
{
  "$schema": "https://dago.dev/schemas/graph.json",
  "id": "graph_customer_support",
  "name": "Customer Support Workflow",
  "version": "1.0.0",
  "entry_node": "classifier",
  "nodes": {
    "classifier": {
      "pattern": "router",
      "config": { "..." : "..." }
    },
    "technical": {
      "pattern": "react",
      "config": { "..." : "..." }
    },
    "billing": {
      "pattern": "llm_call",
      "config": { "..." : "..." }
    },
    "review": {
      "pattern": "guardrail",
      "config": { "..." : "..." }
    }
  },
  "edges": [
    {"type": "conditional", "from": "classifier", "conditions": [
      {"expression": "output.route == 'technical'", "target": "technical"},
      {"expression": "output.route == 'billing'", "target": "billing"}
    ]},
    {"type": "sequential", "from": "technical", "to": "review"},
    {"type": "sequential", "from": "billing", "to": "review"}
  ],
  "memory": {
    "semantic_search": true,
    "episode_context": 3
  }
}
```

### Ubicación de los schemas

```
specs/
├── openapi.yaml               # API REST
├── asyncapi.yaml              # Eventos
└── patterns/                  # Schemas de patrones (JSON Schema)
    ├── graph.json             # Schema del grafo completo
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

### Validación del grafo

Antes de ejecutar, el orchestrator valida:

1. **Estructura** (usando `dominikbraun/graph`):
   - Es un grafo dirigido válido.
   - Tiene un `entry_node` definido y alcanzable.
   - No tiene nodos desconectados.
   - Los loops tienen criterio de parada (`max_iterations` o `timeout`).
   - Los parallel tienen política de join definida.

2. **Schemas** (usando JSON Schema validation):
   - Cada nodo cumple el JSON Schema de su patrón.
   - Cada arista cumple el JSON Schema de su tipo de flujo.
   - Los `input_mapping` y `output_mapping` referencian paths válidos.

3. **Recursos**:
   - Los modelos LLM referenciados están disponibles.
   - Los MCP servers referenciados están registrados.
   - Los subgraphs referenciados existen.

## Alternativas consideradas

- **Patrones como código Go (hardcoded):** Más performante pero no
  configurable por el usuario sin recompilar. Los grafos deben poder
  crearse y modificarse desde el dashboard.

- **YAML en vez de JSON Schema:** Más legible pero sin ecosistema de
  validación. JSON Schema tiene soporte nativo en Go, TypeScript, y
  en la UI del dashboard.

- **BPMN / Workflow standards:** Más completos pero diseñados para
  workflows empresariales, no para orquestación de agentes IA. Excesiva
  complejidad para nuestro caso.

- **Protobuf para definición de patrones:** Buen type safety pero menos
  accesible para usuarios no-técnicos que editen grafos desde el UI.

## Consecuencias

**Positivas:**
- Separación clara entre flujo (aristas) y comportamiento (nodos).
- JSON Schema como cuarta spec — coherente con SDD.
- Grafos definibles y editables desde el dashboard.
- Validación automática antes de ejecución.
- Patrones extensibles — añadir uno nuevo es crear un JSON Schema
  y un handler en el executor.
- Composición jerárquica con subgraphs.

**Negativas:**
- JSON puede ser verboso para grafos complejos (mitigado con UI
  de editor visual en el dashboard).
- La expresividad de las condiciones está limitada por el evaluador
  de expresiones elegido.
- Cada patrón nuevo requiere implementación en el executor además
  del schema.

## Notas para Claude Code

- Los schemas de patrones viven en `specs/patterns/`.
- Al crear un nuevo patrón de nodo, crea el JSON Schema en
  `specs/patterns/nodes/` y el handler en `services/executor/internal/`.
- Al crear un nuevo patrón de flujo, crea el JSON Schema en
  `specs/patterns/edges/` y la lógica en `services/orchestrator/internal/`.
- La validación del grafo usa `dominikbraun/graph` para estructura
  y JSON Schema validation para configuración.
- Los grafos se definen como JSON. El dashboard proporciona un editor
  visual pero el formato subyacente es JSON.
- Todo nodo con loop (`react`, `reflection`) debe tener
  `max_iterations` y/o `timeout_seconds`. Rechaza grafos sin ellos.
