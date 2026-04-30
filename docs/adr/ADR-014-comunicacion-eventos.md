# ADR-014: Comunicación inter-servicio (eventos + HTTP)

**Estado:** Aceptado (revisado: dos modos tras descomposición)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

Dago distribuye la ejecución de grafos siguiendo el modelo LangGraph:
estado centralizado con transiciones por eventos. Tras la descomposición
en 8 servicios, se necesitan dos modos de comunicación.

## Decisión

### Dos modos de comunicación

**Eventos (Valkey Streams)** — Orquestación de grafos. Los servicios
de ejecución se comunican exclusivamente por eventos:

```
orchestrator ↔ executor (node.execute.requested / node.executed)
orchestrator ↔ router   (node.route.requested / node.routed)
orchestrator ↔ planner  (graph.plan.requested / graph.planned)
executor     ↔ mcp-registry (mcp.tool.invoked / mcp.tool.result)
```

**HTTP (Gin)** — Servicios de soporte. Consultas síncronas de baja
latencia:

```
orchestrator → catalog      (obtener definición de paquete)
orchestrator → auth-server   (JWKS)
executor     → mcp-registry  (discovery MCP)
dashboard    → orchestrator  (API REST + WebSocket AG-UI)
dashboard    → catalog       (gestión de paquetes)
dashboard    → agent-registry (Agent Cards)
cualquiera   → auth-server   (validación JWKS)
```

### Catálogo de eventos de orquestación

| Evento | Productor | Consumidor |
|--------|-----------|------------|
| `graph.submitted` | orchestrator (API) | orchestrator |
| `graph.planned` | planner | orchestrator |
| `node.execute.requested` | orchestrator | executor |
| `node.executed` | executor | orchestrator |
| `node.execute.failed` | executor | orchestrator |
| `node.route.requested` | orchestrator | router |
| `node.routed` | router | orchestrator |
| `graph.completed` | orchestrator | dashboard (AG-UI/WS) |
| `graph.failed` | orchestrator | dashboard (AG-UI/WS) |
| `graph.paused` | orchestrator | dashboard (AG-UI/WS) |
| `graph.resumed` | orchestrator (API) | orchestrator |
| `mcp.tool.invoked` | executor | mcp-registry |
| `mcp.tool.result` | mcp-registry | executor |

Todos definidos en `specs/asyncapi.yaml` (ADR-011).

### Estado centralizado (modelo LangGraph)

El orchestrator mantiene el estado canónico de cada ejecución en
PostgreSQL (Ent). Cada evento lleva el estado relevante (Event-Carried
State Transfer). Los demás servicios nunca consultan la DB del
orchestrator directamente.

### Reglas concretas

1. **Orquestación: solo eventos.** Executor nunca llama al
   orchestrator por HTTP. Sin excepciones.

2. **Soporte: HTTP síncrono.** Consultas rápidas antes de actuar.

3. **Todos los eventos llevan `execution_id`** y `auth` (token).

4. **Consumer groups por servicio.** Escalado horizontal.

5. **Idempotencia obligatoria** en consumidores.

6. **Checkpointing** del estado en PostgreSQL tras cada transición.

7. **Timeout por nodo.** Si executor/router no responde,
   `node.execute.failed` con razón `timeout`.

8. **Dead letter:** `{stream}.dlq` para fallos repetidos.

9. **Retry con backoff** para fallos transitorios (LLM rate limited).

10. **Human-in-the-loop:** `graph.paused` persiste estado,
    `graph.resumed` cuando el usuario responde.

## Notas para Claude Code

- Eventos de orquestación: Valkey Streams.
- Servicios de soporte: HTTP con Gin.
- Nunca HTTP entre orchestrator ↔ executor/router/planner.
- Cada evento: envelope con id, type, source, timestamp, data, auth.
- Consumer handlers idempotentes.
- Nuevo tipo de nodo → definir eventos en AsyncAPI primero.
