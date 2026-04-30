# SPRINT-007: Adaptador Event Bus — Valkey Streams + Consumer Groups

## Metadata

- **Fecha inicio:** 2026-04-29
- **Fecha fin estimada:** 2026-04-30
- **Estado:** planificado
- **ADRs aplicados:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-008, ADR-011, ADR-013, ADR-020
- **Specs afectadas:** specs/asyncapi.yaml (7 canales de orquestación)
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Bloqueado por:** SPRINT-001 (go.mod, docker-compose con Valkey 8)
- **Bloquea:** SPRINT-009 (executor consume node.execute.requested), SPRINT-010 (orchestrator publica eventos)

## Objetivo

Implementar el puerto `EventPublisher`/`EventConsumer` en `libs/ports/` y su adaptador
Valkey Streams en `adapters/eventbus/valkey/`, con envelope estándar CloudEvents (campo
`auth` incluido), consumer groups con ACK/NACK, recuperación de pending messages y
dead-letter queue. Tests de integración con Testcontainers.

## Alcance

**Entra:**
- Definición de canales y mensajes en `specs/asyncapi.yaml` (7 eventos de orquestación)
- Tipos de dominio: `Event`, `EventAuth` en `libs/domain/events.go`
- Puertos: `EventPublisher`, `EventConsumer`, `EventHandler` en `libs/ports/eventbus.go`
- Adaptador: publisher, consumer, envelope, consumer group en `adapters/eventbus/valkey/`
- Tests de integración con Testcontainers (6 casos, build tag `integration`)
- Variables de entorno documentadas en `.env.example`

**No entra:**
- Publicación de eventos de negocio (eso es SPRINT-003 en adelante para orchestrator)
- Pub-Sub (notificaciones de dashboard — SPRINT-007-pubsub futuro)
- Caché Valkey (SPRINT-cache futuro)
- Sesiones OAuth en Valkey (SPRINT-005 usa in-memory por ahora)
- Wiring de consumer groups en servicios concretos (cada servicio lo hace en su `main.go`)

## Dependencias

- **Bloqueado por:** SPRINT-001 (go.mod con dependencias Go, docker-compose con Valkey 8)
- **Paralelo a:** SPRINT-002, SPRINT-003, SPRINT-004 (no hay dependencia entre ellos)
- **Bloquea:** SPRINT-003 extendido (orchestrator publica `execution.requested`),
  SPRINT-008 (executor consume eventos), SPRINT-009 (router consume eventos)

## Contratos de comportamiento

### C1 — `EventPublisher.Publish` — publicación básica

```
Given: Valkey activo, stream "dago.graph.execution.requested" existente
When: publisher.Publish(ctx, event, PublishOptions{Stream: "dago.graph.execution.requested"})
Then: XLEN del stream aumenta en 1
      El entry contiene campo "envelope" con el JSON completo del evento
      event.ID es un UUID v4 válido
      unmarshal(marshal(event)) == event (round-trip exacto)
```

### C2 — `EventConsumer.Subscribe` — ACK automático en éxito

```
Given: Mensaje publicado en el stream, consumer group configurado
When: El handler retorna nil al procesar el mensaje
Then: El mensaje recibe XACK
      Una segunda llamada XREADGROUP no devuelve el mensaje
      El mensaje no aparece en la PEL (Pending Entry List)
```

### C3 — DLQ tras MaxRetries

```
Given: Mensaje publicado, MaxRetries=3, handler siempre retorna error
When: El consumer procesa el mensaje exactamente 3 veces
Then: El mensaje aparece en el stream "dago.dlq"
      El mensaje original recibe XACK en el stream de origen (sale de la PEL)
      El evento en dago.dlq preserva el execution_id del evento original
```

## TODOs

### TODO #1 — spec: Definir canales AsyncAPI para eventos de orquestación

**Agente:** @developer

**Descripción:** Actualizar `specs/asyncapi.yaml` con los 7 canales de Valkey Streams
para la fase inicial de orquestación, siguiendo el envelope CloudEvents + campo `auth`
definido en ADR-011.

**Archivos afectados:**
- `specs/asyncapi.yaml`

**Canales a definir:**
- `dago.graph.execution.requested` — orchestrator publica; executor/router consumen
- `dago.node.execution.started` — executor publica; orchestrator consume
- `dago.node.execution.completed` — executor publica; orchestrator consume
- `dago.node.execution.failed` — executor publica; orchestrator consume
- `dago.graph.execution.completed` — orchestrator publica
- `dago.graph.execution.failed` — orchestrator publica
- `dago.dlq` — dead-letter queue; cualquier servicio publica

**Schema del envelope (inline en AsyncAPI):**
```yaml
type: object
required: [id, type, source, specversion, time, datacontenttype, data]
properties:
  id:            { type: string, format: uuid }
  type:          { type: string }
  source:        { type: string }
  specversion:   { type: string, enum: ["1.0"] }
  time:          { type: string, format: date-time }
  datacontenttype: { type: string, enum: ["application/json"] }
  auth:
    type: object
    properties:
      sub:      { type: string, format: uuid }
      scope:    { type: string }
      tags:     { type: array, items: { type: string } }
      org_unit: { type: string, format: uuid }
  data:          { type: object }
```

**Criterios de aceptación:**
- `asyncapi validate specs/asyncapi.yaml` no reporta errores
- Cada canal tiene: nombre de stream, binding Valkey, consumer groups suscritos, schema

**Test asociado:** —

---

### TODO #2 — data: Tipos Event y EventAuth en libs/domain/events.go

**Agente:** @developer

**Descripción:** Definir los tipos de dominio puros para eventos. Sin dependencias de
infraestructura. `Event.Data` es `json.RawMessage` para no acoplar al tipo concreto.

**Archivos afectados:**
- `libs/domain/events.go` (nuevo)

**Tipos a implementar:**
```go
package domain

import (
    "encoding/json"
    "time"
)

type EventAuth struct {
    Sub     string   `json:"sub"`
    Scope   string   `json:"scope"`
    Tags    []string `json:"tags"`
    OrgUnit string   `json:"org_unit"`
}

type Event struct {
    ID              string          `json:"id"`
    Type            string          `json:"type"`
    Source          string          `json:"source"`
    SpecVersion     string          `json:"specversion"`
    Time            time.Time       `json:"time"`
    DataContentType string          `json:"datacontenttype"`
    Auth            *EventAuth      `json:"auth,omitempty"`
    Data            json.RawMessage `json:"data"`
}
```

**Constantes de tipos de evento:**
```go
const (
    EventTypeExecutionRequested  = "dago.graph.execution.requested"
    EventTypeNodeStarted         = "dago.node.execution.started"
    EventTypeNodeCompleted       = "dago.node.execution.completed"
    EventTypeNodeFailed          = "dago.node.execution.failed"
    EventTypeExecutionCompleted  = "dago.graph.execution.completed"
    EventTypeExecutionFailed     = "dago.graph.execution.failed"
    EventTypeDLQ                 = "dago.dlq"
)
```

**Criterios de aceptación:**
- `go build ./libs/domain/...` sin errores
- Sin imports de infraestructura (`adapters/`, `valkey-go`, etc.)

**Test asociado:** —

---

### TODO #3 — impl: Interfaces EventPublisher y EventConsumer en libs/ports/eventbus.go

**Agente:** @developer

**Descripción:** Definir los puertos de salida para publicación y consumo de eventos.
Separar `EventPublisher` de `EventConsumer` para que cada servicio importe solo lo que
necesita (orchestrator publica y consume; executor solo consume ciertos streams y publica
resultados).

**Archivos afectados:**
- `libs/ports/eventbus.go` (nuevo)

**Interfaces a implementar:**
```go
package ports

import (
    "context"
    "github.com/aescanero/dago/libs/domain"
)

type PublishOptions struct {
    Stream string
}

type ConsumeOptions struct {
    Stream        string
    Group         string
    ConsumerName  string
    BlockDuration time.Duration
    MaxRetries    int // default 3, tras MaxRetries va a DLQ
}

type EventHandler func(ctx context.Context, event domain.Event) error

type EventPublisher interface {
    Publish(ctx context.Context, event domain.Event, opts PublishOptions) error
    Close() error
}

type EventConsumer interface {
    // Subscribe bloquea hasta ctx cancelado. Llama handler por cada mensaje.
    // ACK automático si handler retorna nil. Sin ACK si retorna error.
    Subscribe(ctx context.Context, opts ConsumeOptions, handler EventHandler) error
    // RecoverPending reasigna mensajes idle > idleThreshold al consumer actual.
    RecoverPending(ctx context.Context, opts ConsumeOptions, idleThreshold time.Duration) error
    Close() error
}
```

**Criterios de aceptación:**
- `go build ./libs/ports/...` sin errores
- Solo importa `libs/domain/` y paquetes stdlib

**Test asociado:** —

---

### TODO #4 — test: Tests de integración (Red) con Testcontainers

**Agente:** @qa

**Descripción:** Escribir los 6 tests de integración con build tag `integration` ANTES
de implementar el adaptador. Deben fallar (Red) al ejecutarse contra stubs vacíos.

**Archivos afectados:**
- `adapters/eventbus/valkey/integration_test.go` (nuevo)

**Tests a implementar:**
```go
//go:build integration

package valkey_test

// TestMain: levanta container valkey/valkey:8, obtiene addr, crea cliente

func TestPublishAndConsume(t *testing.T)
// Publish un evento → Subscribe con consumer group → handler recibe evento igual

func TestConsumerGroupAck(t *testing.T)
// Publish → Subscribe (handler OK → ACK) → segundo XREADGROUP no devuelve el mensaje

func TestConsumerGroupNoAck(t *testing.T)
// Publish → Subscribe (handler error → sin ACK) → segundo XREADGROUP devuelve el mismo mensaje

func TestPendingRecovery(t *testing.T)
// Publish → consumer1 lee sin ACK → RecoverPending con idle>0 → consumer2 recibe el mensaje

func TestDLQAfterMaxRetries(t *testing.T)
// Publish → Subscribe con MaxRetries=3, handler siempre error → tras 3 intentos
// el mensaje aparece en stream "dago.dlq"

func TestEnvelopeRoundtrip(t *testing.T)
// Event con todos los campos incluido Auth → Publish → Subscribe
// → evento recibido == evento publicado (deep equal)
```

**Criterios de aceptación:**
- `go test -tags integration ./adapters/eventbus/valkey/... -run TestPublish` falla con
  "not implemented" o similar (Red confirmado)
- Testcontainers descarga imagen `valkey/valkey:8` automáticamente

**Test asociado:** este TODO ES el test

---

### TODO #5 — impl: Envelope — serialización/deserialización CloudEvents

**Agente:** @developer

**Descripción:** Implementar la conversión entre `domain.Event` y el formato de entrada
de Valkey Streams (`map[string]any` con campo `envelope` JSON).

**Archivos afectados:**
- `adapters/eventbus/valkey/envelope.go` (nuevo)

**Diseño:**
```go
// marshalEnvelope serializa domain.Event → map[string]any para XADD
// unmarshalEnvelope deserializa entry de XREAD → domain.Event
// El stream entry tiene un único campo "envelope" con el JSON completo del evento
```

**Criterios de aceptación:**
- `TestEnvelopeRoundtrip` pasa (Green)
- Marshal/Unmarshal son inversas: `unmarshal(marshal(e)) == e`
- Campo `auth` se omite si es nil (`omitempty`)

**Test asociado:** `TestEnvelopeRoundtrip`

---

### TODO #6 — impl: Publisher — XADD con XGROUP CREATE idempotente

**Agente:** @developer

**Descripción:** Implementar `ValkeyPublisher` que implementa `ports.EventPublisher`.
Garantizar que el stream y el grupo existen antes de publicar (XGROUP CREATE MKSTREAM).

**Archivos afectados:**
- `adapters/eventbus/valkey/publisher.go` (nuevo)

**Comportamiento:**
- `Publish`: serializa con `marshalEnvelope` → `XADD stream * envelope <json>`
- `EnsureStream`: `XGROUP CREATE stream <group> $ MKSTREAM` (idempotente, ignora BUSYGROUP)
- `Close`: cierra conexión Valkey

**Error handling:**
- `fmt.Errorf("eventbus publish %s: %w", stream, err)`
- Timeout desde `ctx`

**Criterios de aceptación:**
- `TestPublishAndConsume` pasa (Green) cuando el consumer también está implementado
- `TestEnvelopeRoundtrip` pasa

**Test asociado:** `TestPublishAndConsume`, `TestEnvelopeRoundtrip`

---

### TODO #7 — impl: Consumer — XREADGROUP + ACK + DLQ tras MaxRetries

**Agente:** @developer

**Descripción:** Implementar `ValkeyConsumer` que implementa `ports.EventConsumer`.
`Subscribe` bloquea en un loop leyendo con `XREADGROUP`. Gestiona ACK/NACK y DLQ.

**Archivos afectados:**
- `adapters/eventbus/valkey/consumer.go` (nuevo)

**Comportamiento de Subscribe:**
1. `XGROUP CREATE stream group $ MKSTREAM` (idempotente)
2. Loop hasta `ctx.Done()`:
   a. `XREADGROUP GROUP group consumer BLOCK blockDuration COUNT 10 STREAMS stream >`
   b. Para cada mensaje: deserializar → llamar `handler`
   c. Si `handler` retorna nil: `XACK stream group id`
   d. Si `handler` retorna error:
      - Consultar `XPENDING stream group - + 10` para contar reintentos del mensaje
      - Si reintentos >= `MaxRetries`: publicar en `dago.dlq` + `XACK` (para sacarlo del pending)
      - Si reintentos < `MaxRetries`: no XACK (el mensaje permanece en pending)

**Error handling:**
- Errores de conexión: log + retry con backoff exponencial (1s, 2s, 4s, máx 30s)
- `fmt.Errorf("eventbus consume %s: %w", stream, err)`

**Criterios de aceptación:**
- `TestConsumerGroupAck` pasa
- `TestConsumerGroupNoAck` pasa
- `TestDLQAfterMaxRetries` pasa

**Test asociado:** `TestConsumerGroupAck`, `TestConsumerGroupNoAck`, `TestDLQAfterMaxRetries`

---

### TODO #8 — impl: RecoverPending — XAUTOCLAIM para mensajes idle

**Agente:** @developer

**Descripción:** Implementar `RecoverPending` en `ValkeyConsumer`. Al arrancar un
servicio, reclama los mensajes que llevan más de `idleThreshold` sin ACK (pueden
pertenecer a un consumer anterior que murió).

**Archivos afectados:**
- `adapters/eventbus/valkey/consumer.go` (añadir método)

**Comportamiento:**
```
XAUTOCLAIM stream group consumerName idleMs 0-0 COUNT 100
```
- Reasigna los mensajes al `consumerName` actual
- Procesa cada mensaje con el mismo handler de `Subscribe`
- Si handler OK: XACK; si error: lógica de MaxRetries igual que en Subscribe

**Criterios de aceptación:**
- `TestPendingRecovery` pasa
- Idempotente: llamar dos veces no duplica el procesamiento

**Test asociado:** `TestPendingRecovery`

---

### TODO #9 — infra: Añadir dependencia valkey-go al go.mod y configuración

**Agente:** @devops

**Descripción:** Añadir `github.com/valkey-io/valkey-go` al go.mod del monorepo.
Añadir `github.com/testcontainers/testcontainers-go` para tests de integración.
Documentar variables de entorno en `.env.example`.

**Archivos afectados:**
- `go.mod` / `go.sum` (via `go get`)
- `.env.example`

**Comandos:**
```bash
go get github.com/valkey-io/valkey-go@latest
go get github.com/testcontainers/testcontainers-go@latest
```

**Variables de entorno a añadir en `.env.example`:**
```
VALKEY_ADDR=localhost:6379
VALKEY_PASSWORD=           # vacío en dev
VALKEY_DLQ_STREAM=dago.dlq
VALKEY_MAX_RETRIES=3
VALKEY_CONSUMER_IDLE_MS=30000
```

**Criterios de aceptación:**
- `go build ./adapters/eventbus/...` sin errores
- `go test -tags integration ./adapters/eventbus/...` levanta container y ejecuta tests

**Test asociado:** todos los tests de integración

---

### TODO #10 — infra: Target Makefile para tests de integración del eventbus

**Agente:** @devops

**Descripción:** Añadir target `make test-integration-eventbus` en el Makefile para
ejecutar solo los tests de integración del adaptador, separados de los unitarios.

**Archivos afectados:**
- `Makefile`

**Target a añadir:**
```makefile
test-integration-eventbus:
	go test -tags integration -count=1 -timeout 120s \
	  ./adapters/eventbus/...

test-integration: test-integration-eventbus
	@echo "All integration tests passed"
```

**Criterios de aceptación:**
- `make test-integration-eventbus` ejecuta los 6 tests y todos pasan
- `make ci` no ejecuta tests de integración (solo `//go:build integration` los activa)

**Test asociado:** todos los tests de integración

---

### TODO #11 — docs: Actualizar docs/index.md y docs/log.md

**Agente:** @docs

**Descripción:** Registrar SPRINT-007 en el índice y el log del proyecto.

**Archivos afectados:**
- `docs/index.md` — añadir fila en tabla Sprints
- `docs/log.md` — añadir entrada append-only

**Criterios de aceptación:**
- `grep "SPRINT-007" docs/index.md` retorna la fila
- `grep "SPRINT-007" docs/log.md` retorna la entrada
- `docs/index.md` añade sección "## Adaptadores" si no existe, con fila del eventbus

**Test asociado:** —

---

## Matriz de trazabilidad

| TODO | Tipo   | ADR       | Spec              | Test                        | Impl                                  |
|------|--------|-----------|-------------------|-----------------------------|---------------------------------------|
| #1   | spec   | 011       | asyncapi.yaml     | —                           | specs/asyncapi.yaml                   |
| #2   | data   | 001, 011  | asyncapi.yaml     | —                           | libs/domain/events.go                 |
| #3   | impl   | 001, 008  | —                 | —                           | libs/ports/eventbus.go                |
| #4   | test   | 002, 008  | asyncapi.yaml     | integration_test.go (Red)   | —                                     |
| #5   | impl   | 008, 011  | asyncapi.yaml     | TestEnvelopeRoundtrip        | adapters/eventbus/valkey/envelope.go  |
| #6   | impl   | 008       | asyncapi.yaml     | TestPublishAndConsume        | adapters/eventbus/valkey/publisher.go |
| #7   | impl   | 008, 011  | asyncapi.yaml     | TestAck, TestNoAck, TestDLQ | adapters/eventbus/valkey/consumer.go  |
| #8   | impl   | 008       | —                 | TestPendingRecovery          | adapters/eventbus/valkey/consumer.go  |
| #9   | infra  | 008, 013  | —                 | (habilita todos los tests)  | go.mod, .env.example                  |
| #10  | infra  | 002       | —                 | (habilita ejecución CI)     | Makefile                              |
| #11  | docs   | 020       | —                 | —                           | docs/index.md, docs/log.md            |

## Notas de implementación

**Cliente Valkey:** usar `github.com/valkey-io/valkey-go` (no `go-redis`). API:
```go
client, err := valkey.NewClient(valkey.ClientOption{
    InitAddress: []string{addr},
})
```

**XPENDING para contar reintentos:** el campo `delivery-count` de cada entrada PEL
(Pending Entry List) devuelve cuántas veces se entregó. Usar:
```
XPENDING stream group - + 1 consumerName
```
O bien `XRANGE` sobre la PEL. Alternativamente, incluir un contador en el envelope de
DLQ (`retry_count` en el campo `data` del evento `dago.dlq`).

**Testcontainers setup mínimo:**
```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    req := testcontainers.ContainerRequest{
        Image:        "valkey/valkey:8",
        ExposedPorts: []string{"6379/tcp"},
        WaitingFor:   wait.ForLog("Ready to accept connections"),
    }
    container, _ := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req, Started: true,
    })
    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "6379")
    valkeyAddr = fmt.Sprintf("%s:%s", host, port.Port())
    os.Exit(m.Run())
}
```

## Resultado (completar al cerrar)

- [ ] `TestPublishAndConsume` pasa
- [ ] `TestConsumerGroupAck` pasa
- [ ] `TestConsumerGroupNoAck` pasa
- [ ] `TestPendingRecovery` pasa
- [ ] `TestDLQAfterMaxRetries` pasa
- [ ] `TestEnvelopeRoundtrip` pasa
- [ ] `go build ./libs/... ./adapters/eventbus/...` sin errores
- [ ] `golangci-lint run ./libs/... ./adapters/eventbus/...` sin errores
- [ ] `specs/asyncapi.yaml` validado con 7 canales definidos
- [ ] `.env.example` actualizado con variables Valkey
- [ ] `docs/index.md` y `docs/log.md` actualizados
