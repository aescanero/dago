# ADR-011: AsyncAPI y patrón de eventos con Valkey

**Estado:** Aceptado (revisado: Redis → Valkey)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El ADR-008 establece Valkey como infraestructura de eventos. Se necesita
definir qué patrón de eventos se adopta, cómo se estructuran los mensajes,
y cómo se documenta formalmente la API asíncrona.

## Decisión

### Contrato: AsyncAPI 3.0

Se adopta **AsyncAPI 3.0** como especificación de la API asíncrona.
Fichero: `specs/asyncapi.yaml`.

### Patrón: Event Notification + Event-Carried State Transfer

```
Event Notification (Pub/Sub)       Event-Carried State (Streams)
─────────────────────────          ──────────────────────────────
Garantía: ninguna                  Garantía: at-least-once
Payload: mínimo (ID)               Payload: completo
Uso: caché, UI real-time           Uso: eventos de negocio
```

### Reglas concretas

1. **Event Notification (Valkey Pub/Sub)** para señales efímeras:
   invalidación de caché, notificaciones UI.

2. **Event-Carried State Transfer (Valkey Streams)** para eventos
   de negocio con procesamiento garantizado.

3. **No Event Sourcing.** Estado en PostgreSQL (Ent). Si se necesita
   event sourcing, evaluar Kafka/EventStoreDB.

4. **Envelope estándar:**

   ```go
   type Event struct {
       ID        string          `json:"id"`
       Type      string          `json:"type"`
       Source    string          `json:"source"`
       Timestamp time.Time       `json:"timestamp"`
       Data      json.RawMessage `json:"data"`
       Auth      string          `json:"auth,omitempty"` // Token propagado
   }
   ```

5. **Naming:** `{dominio}.{acción}` en pasado. `node.executed`, no
   `node.execute`.

6. **Consumer groups** para procesamiento distribuido (Valkey Streams).

7. **Dead letter:** `{stream}.dlq` para mensajes con fallos repetidos.

8. **Retención:** `MAXLEN ~1000` por stream.

9. **Spec-first:** Definir en `specs/asyncapi.yaml` antes de
   implementar. Tests de contrato validan cumplimiento.

10. **Campo `auth` en el envelope** para propagación de tokens
    OAuth 2.1 (ADR-012) a través de los eventos.

### Ejemplo AsyncAPI

```yaml
asyncapi: 3.0.0
info:
  title: Dago Event API
  version: 1.0.0

channels:
  nodeExecuted:
    address: node.executed
    messages:
      nodeExecutedMessage:
        $ref: '#/components/messages/NodeExecuted'

components:
  messages:
    NodeExecuted:
      payload:
        $ref: '#/components/schemas/NodeExecutedPayload'

  schemas:
    NodeExecutedPayload:
      type: object
      required: [id, type, source, timestamp, data]
      properties:
        id:
          type: string
        type:
          type: string
          const: node.executed
        source:
          type: string
        timestamp:
          type: string
          format: date-time
        auth:
          type: string
        data:
          type: object
```

## Notas para Claude Code

- Eventos en `specs/asyncapi.yaml` — spec-first.
- Valkey Streams para negocio, Pub/Sub para efímeros.
- Envelope con id, type, source, timestamp, data, auth.
- Naming: pasado simple (`node.executed`).
- ACK tras procesamiento exitoso, nunca antes.
- Publisher en `adapters/eventbus/`. Consumer en cada servicio.
- El campo `auth` propaga el token OAuth 2.1.
