# ADR-018: AG-UI como protocolo agente↔usuario y A2UI para UI generativa

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El dashboard de dago necesita comunicarse en tiempo real con los agentes
durante la ejecución de grafos: streaming de tokens, visualización de
tool calls, human-in-the-loop, sincronización de estado. Además, los
paquetes (ADR-017) pueden incluir componentes de UI que los agentes
solicitan renderizar durante la ejecución.

Existen dos protocolos complementarios que cubren estas necesidades,
completando el "tridente" de protocolos agénticos junto con MCP
(agente↔herramienta) y A2A (agente↔agente).

## Decisión

Se adoptan dos protocolos complementarios:

- **AG-UI** como protocolo de comunicación agente↔usuario.
- **A2UI** como formato de UI generativa declarativa.

### AG-UI — Agent-User Interaction Protocol

AG-UI es un protocolo abierto, basado en eventos, que estandariza
cómo los agentes se conectan a aplicaciones de usuario. Licencia MIT,
creado por CopilotKit, con soporte de Amazon Bedrock, Microsoft
Agent Framework y más de 13k estrellas en GitHub.

AG-UI define ~16 tipos de eventos estándar:

```
Lifecycle:     RUN_STARTED, RUN_FINISHED, RUN_ERROR
Text:          TEXT_MESSAGE_START, TEXT_MESSAGE_CONTENT, TEXT_MESSAGE_END
Tool calls:    TOOL_CALL_START, TOOL_CALL_ARGS, TOOL_CALL_RESULT, TOOL_CALL_END
State:         STATE_SNAPSHOT, STATE_DELTA
Custom:        CUSTOM
```

**Transporte:** AG-UI funciona sobre WebSocket (que ya decidimos usar)
o SSE. No reemplaza WebSocket — es una capa de protocolo encima del
transporte que estandariza el formato de los mensajes.

**Rol en dago:** El orchestrator emite eventos AG-UI hacia el
dashboard durante la ejecución de grafos:

```
Orchestrator                          Dashboard
    │                                      │
    ├── RUN_STARTED ──────────────────────▶│ Muestra "ejecutando..."
    ├── TEXT_MESSAGE_CONTENT (token) ─────▶│ Streaming de texto
    ├── TOOL_CALL_START ──────────────────▶│ Muestra tool activity
    ├── TOOL_CALL_RESULT ─────────────────▶│ Muestra resultado
    ├── STATE_DELTA ──────────────────────▶│ Actualiza estado del grafo
    ├── (user approval request) ◀─────────│ Human-in-the-loop
    ├── RUN_FINISHED ─────────────────────▶│ Muestra resultado final
    │                                      │
```

### A2UI — Declarative UI for Agents

A2UI resuelve cómo los agentes pueden solicitar UIs ricas de forma
segura. En vez de generar HTML o ejecutar código, el agente envía
una descripción declarativa de lo que quiere mostrar y el frontend
lo renderiza con componentes pre-aprobados del catálogo.

```json
{
  "type": "form",
  "id": "satisfaction_survey",
  "fields": [
    {"type": "rating", "label": "Satisfaction", "min": 1, "max": 5},
    {"type": "text", "label": "Comments", "multiline": true}
  ],
  "actions": [
    {"type": "submit", "label": "Send feedback"}
  ]
}
```

**Seguridad:** Los agentes solo pueden usar componentes del catálogo
registrado. No hay inyección de UI, no hay ejecución de código
arbitrario. El dashboard valida el descriptor A2UI contra un catálogo
de componentes permitidos antes de renderizar.

**Rol en dago:** Los nodos del grafo pueden emitir descriptores A2UI
como parte de su output. El orchestrator los envía al dashboard vía
AG-UI (usando eventos CUSTOM). El dashboard los renderiza con
componentes de shadcn/ui (ADR-019).

### El tridente de protocolos agénticos en dago

```
MCP     → Agente ↔ Herramientas   (executor → mcp-registry → tools)
A2A     → Agente ↔ Agente         (agent-registry → Agent Cards)
AG-UI   → Agente ↔ Usuario        (orchestrator → dashboard, runtime)
A2UI    → Agente → UI widgets     (nodo → dashboard, declarativo)
```

### Reglas concretas

1. **AG-UI sobre WebSocket.** El orchestrator expone un endpoint
   WebSocket que emite eventos AG-UI. El dashboard consume estos
   eventos con un cliente AG-UI compatible.

2. **Eventos AG-UI para toda comunicación en tiempo real.** Streaming
   de tokens, tool calls, state sync, human-in-the-loop — todo usa
   el formato de eventos AG-UI, no un formato propietario.

3. **A2UI para UI dinámica.** Cuando un nodo necesita mostrar un
   formulario, una tabla, un gráfico o cualquier widget interactivo,
   emite un descriptor A2UI. El dashboard lo renderiza con
   componentes del catálogo.

4. **Catálogo de componentes A2UI registrado.** Solo los tipos de
   componentes registrados en el catálogo se renderizan. Componentes
   desconocidos se ignoran con un fallback informativo.

5. **Los descriptores A2UI se incluyen en los paquetes.** Cada
   paquete (ADR-017) puede declarar qué componentes A2UI utiliza
   en su sección `ui.a2ui_catalog`.

## Alternativas consideradas

- **Protocolo WebSocket propietario:** Máxima flexibilidad pero cero
  interoperabilidad. Cada cambio requiere actualizar cliente y
  servidor. Descartado.

- **Solo A2UI sin AG-UI:** A2UI solo cubre la generación de widgets,
  no el streaming de texto, state sync ni human-in-the-loop. Insuficiente.

- **Solo AG-UI sin A2UI:** AG-UI cubre la comunicación pero no tiene
  un formato estándar para describir widgets de UI. Se necesitaría
  un formato propietario para eso.

- **HTML/JS generado por el agente:** Riesgo de seguridad alto
  (XSS, inyección). Descartado.

## Consecuencias

**Positivas:**
- Interoperabilidad con cualquier frontend AG-UI compatible.
- Seguridad: A2UI es declarativo, no ejecutable.
- Protocolo estándar con adopción creciente (AWS, Microsoft, CopilotKit).
- Los paquetes pueden incluir definiciones de UI.
- Human-in-the-loop estandarizado.

**Negativas:**
- AG-UI es relativamente joven — posibles cambios en el protocolo.
- A2UI limita la expresividad de la UI a los componentes del catálogo.
- Dos protocolos de UI añaden complejidad conceptual.

## Notas para Claude Code

- El endpoint WebSocket del orchestrator emite eventos AG-UI.
- Los tipos de evento AG-UI se definen en `libs/domain/events/agui.go`.
- Los descriptores A2UI se validan contra el catálogo en el dashboard.
- Al crear un patrón de nodo que necesite UI, incluir la definición
  A2UI en el paquete y el handler AG-UI en el orchestrator.
- El cliente AG-UI en el dashboard vive en `dashboard/src/api/agui/`.
