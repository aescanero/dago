# ADR-015: Arquitectura de memoria de agentes (tres capas)

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

Los nodos de dago que se comportan como agentes necesitan memoria para
funcionar eficazmente. Sin memoria, cada ejecución parte de cero — el
agente no recuerda decisiones anteriores, no aprende de errores, y no
puede mantener contexto entre ejecuciones de un mismo grafo o entre
ejecuciones diferentes.

El modelo de memoria de OpenClaw (MEMORY.md, notas diarias, dreaming)
demuestra que una arquitectura de tres capas es efectiva. Adaptamos
ese modelo a dago, reemplazando ficheros Markdown por PostgreSQL (Ent),
pgvector y Valkey, adecuados para un sistema multi-usuario y distribuido.

## Decisión

Se adopta una **arquitectura de memoria de tres capas** con un proceso
de consolidación asíncrono:

```
┌─────────────────────────────────────────────────────────────┐
│                   MEMORIA DEL AGENTE                        │
│                                                             │
│  Capa 1: Working Memory (corto plazo)                       │
│  ─────────────────────────────────────                      │
│  Qué: Estado de la ejecución actual del grafo               │
│  Dónde: PostgreSQL (Ent) + Valkey                           │
│  Ciclo de vida: Existe mientras el grafo se ejecuta         │
│  Equivalente OpenClaw: Notas diarias (hoy + ayer)           │
│                                                             │
│  Capa 2: Episodic Memory (medio plazo)                      │
│  ─────────────────────────────────────                      │
│  Qué: Historial de ejecuciones pasadas, resultados,         │
│       decisiones tomadas por cada nodo                      │
│  Dónde: PostgreSQL (Ent)                                    │
│  Ciclo de vida: Persiste indefinidamente, consultable       │
│  Equivalente OpenClaw: Notas diarias archivadas             │
│                                                             │
│  Capa 3: Semantic Memory (largo plazo)                      │
│  ─────────────────────────────────────                      │
│  Qué: Conocimiento destilado — hechos, patrones,            │
│       preferencias, lecciones aprendidas                    │
│  Dónde: pgvector (embeddings) + PostgreSQL (Ent)            │
│  Ciclo de vida: Curada, supersedida pero nunca borrada      │
│  Equivalente OpenClaw: MEMORY.md + búsqueda semántica       │
│                                                             │
│  Consolidación (dreaming)                                   │
│  ─────────────────────────                                  │
│  Qué: Proceso background que promueve datos relevantes      │
│       de working → episodic → semantic                      │
│  Dónde: Servicio background en el orchestrator              │
│  Cuándo: Tras completar una ejecución de grafo              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Capa 1: Working Memory (corto plazo)

Estado vivo de la ejecución actual. Es lo que los nodos leen y escriben
durante la ejecución de un grafo.

```go
// Modelo conceptual del estado de ejecución
type ExecutionState struct {
    ExecutionID  uuid.UUID
    GraphID      uuid.UUID
    Status       ExecutionStatus
    CurrentNode  string
    Variables    map[string]any        // Variables acumuladas del grafo
    Messages     []Message             // Historial conversacional
    NodeResults  map[string]NodeResult // Resultados por nodo
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

**Almacenamiento:**
- Estado completo en PostgreSQL (Ent) para persistencia y recovery.
- Estado caliente en Valkey para acceso rápido durante ejecución.
- Cada transición de nodo actualiza ambos (write-through).

**Acceso:**
- Los nodos reciben el estado relevante vía eventos (ADR-014).
- El orchestrator mantiene la versión canónica.
- Checkpointing tras cada transición para recovery ante crashes.

**Ciclo de vida:**
- Se crea cuando se inicia una ejecución.
- Se actualiza en cada transición de nodo.
- Al completar el grafo, se archiva como episodic memory.

### Capa 2: Episodic Memory (medio plazo)

Historial completo de ejecuciones pasadas. Permite a los agentes
consultar qué pasó en ejecuciones anteriores del mismo grafo o de
grafos similares.

```go
// Modelo conceptual del registro episódico
type EpisodeRecord struct {
    ExecutionID   uuid.UUID
    GraphID       uuid.UUID
    UserID        uuid.UUID
    StartedAt     time.Time
    CompletedAt   time.Time
    Status        ExecutionStatus      // completed, failed, cancelled
    FinalState    map[string]any       // Estado final del grafo
    NodeHistory   []NodeEpisode        // Secuencia de nodos ejecutados
    ErrorDetails  *string              // Si falló, por qué
    Metadata      map[string]string    // Tags, categorías
}

type NodeEpisode struct {
    NodeID       string
    Pattern      string               // react, reflection, tool_use, etc.
    Input        json.RawMessage
    Output       json.RawMessage
    Duration     time.Duration
    TokensUsed   int
    Success      bool
    ErrorMessage *string
}
```

**Almacenamiento:** PostgreSQL (Ent). Tablas de ejecuciones y resultados
de nodos con índices para consulta eficiente.

**Acceso:**
- Consultable por grafo, usuario, rango de fechas, estado.
- Los nodos pueden consultar episodios anteriores como contexto
  (ej: "las últimas 5 veces que ejecutamos este grafo, ¿qué pasó?").
- Acceso vía puerto `EpisodeStore` en `libs/ports/`.

**Ciclo de vida:**
- Se crea al completar una ejecución (archivado de working memory).
- Persiste indefinidamente.
- El proceso de consolidación extrae hechos relevantes hacia
  semantic memory.

### Capa 3: Semantic Memory (largo plazo)

Conocimiento destilado y generalizado. No es el historial crudo sino
hechos, patrones y lecciones aprendidas que el agente puede usar en
futuras ejecuciones.

```go
// Modelo conceptual de memoria semántica
type SemanticFact struct {
    ID           uuid.UUID
    Content      string                // El hecho en lenguaje natural
    Embedding    pgvector.Vector       // Embedding para búsqueda semántica
    FactType     FactType              // preference, lesson, pattern, fact
    Source       string                // De dónde se aprendió
    Confidence   float64               // 0.0 - 1.0
    SupersededBy *uuid.UUID            // Si fue reemplazado por otro hecho
    CreatedAt    time.Time
    LastUsedAt   time.Time
    UseCount     int                   // Cuántas veces se ha consultado
}
```

Tipos de hechos:
- **preference** — "El usuario prefiere respuestas concisas"
- **lesson** — "El nodo de scraping falla los lunes por mantenimiento"
- **pattern** — "Los grafos de análisis financiero funcionan mejor con
  temperatura 0.2"
- **fact** — "La API de pagos tiene un rate limit de 100 req/min"

**Almacenamiento:**
- PostgreSQL con extensión **pgvector** para embeddings.
- Los embeddings se generan con un modelo de embedding (configurable).
- Búsqueda por similitud semántica con `<=>` (cosine distance).

**Acceso:**
- Búsqueda semántica: "¿qué sé sobre el rate limit de la API X?"
  devuelve los hechos más relevantes sin necesidad de keyword exacta.
- Acceso vía puerto `SemanticMemory` en `libs/ports/`.
- Los nodos pueden inyectar hechos relevantes en su contexto antes
  de ejecutar.

**Ciclo de vida:**
- Creada por el proceso de consolidación (dreaming).
- Nunca se borra, solo se supersede: si un hecho cambia, se marca
  el anterior con `superseded_by` y se crea uno nuevo.
- `confidence` y `use_count` se actualizan con el uso.
- Hechos con `confidence` baja y `use_count` cero se depriorizan
  pero no se eliminan.

### Consolidación (dreaming)

Proceso background que promueve conocimiento entre capas:

```
Working Memory ──(al completar grafo)──▶ Episodic Memory
                                              │
                                    (proceso background)
                                              │
                                              ▼
                                      Semantic Memory
```

**Trigger:** Se ejecuta tras cada ejecución completada de grafo.

**Proceso:**
1. Archiva el estado de ejecución como registro episódico.
2. Analiza el episodio buscando hechos nuevos o actualizados:
   - ¿Hubo errores recurrentes? → lesson learned.
   - ¿El usuario expresó una preferencia? → preference.
   - ¿Se descubrió un patrón de rendimiento? → pattern.
3. Genera embeddings para los hechos nuevos.
4. Verifica si el hecho ya existe (búsqueda semántica por similitud).
   - Si existe y es consistente: incrementa confidence.
   - Si existe y contradice: supersede el anterior.
   - Si es nuevo: inserta con confidence inicial.
5. Registra en `DREAMS` log para auditoría.

**El proceso de consolidación puede usar un LLM** para extraer hechos
de los episodios — es un uso del patrón Reflection aplicado a la
memoria del sistema.

### Puertos de memoria

```go
// libs/ports/memory.go

// WorkingMemory gestiona el estado de ejecución activo
type WorkingMemory interface {
    Get(ctx context.Context, executionID uuid.UUID) (*ExecutionState, error)
    Update(ctx context.Context, state *ExecutionState) error
    Checkpoint(ctx context.Context, executionID uuid.UUID) error
}

// EpisodeStore gestiona el historial de ejecuciones
type EpisodeStore interface {
    Archive(ctx context.Context, state *ExecutionState) (*EpisodeRecord, error)
    FindByGraph(ctx context.Context, graphID uuid.UUID, limit int) ([]EpisodeRecord, error)
    FindByUser(ctx context.Context, userID uuid.UUID, limit int) ([]EpisodeRecord, error)
}

// SemanticMemory gestiona el conocimiento a largo plazo
type SemanticMemory interface {
    Store(ctx context.Context, fact *SemanticFact) error
    Search(ctx context.Context, query string, limit int) ([]SemanticFact, error)
    Supersede(ctx context.Context, oldID uuid.UUID, newFact *SemanticFact) error
}

// MemoryConsolidator ejecuta el proceso de dreaming
type MemoryConsolidator interface {
    Consolidate(ctx context.Context, episode *EpisodeRecord) error
}
```

## Alternativas consideradas

- **Ficheros Markdown (estilo OpenClaw):** Simple y transparente pero
  no escala a multi-usuario ni es consultable eficientemente.

- **Vector DB dedicada (Pinecone, Weaviate, Chroma):** Más potente
  para búsqueda vectorial pero añade infraestructura. pgvector cubre
  el caso de uso sin añadir otro sistema.

- **Solo PostgreSQL (sin pgvector):** Posible con búsqueda full-text
  pero pierde la capacidad de búsqueda semántica por similitud.

- **Solo Valkey para todo:** Rápido pero sin persistencia durable ni
  búsqueda semántica. No adecuado para memoria a largo plazo.

- **Mem0 / LangMem:** Soluciones específicas para memoria de agentes.
  Interesantes pero añaden dependencia externa. Preferimos implementar
  sobre nuestro stack existente.

## Consecuencias

**Positivas:**
- Tres capas con responsabilidades claras.
- pgvector reutiliza PostgreSQL existente — sin infraestructura nueva.
- Búsqueda semántica permite recuperar contexto sin keywords exactos.
- "Nunca borrar" preserva historial completo para auditoría y aprendizaje.
- Consolidación automática reduce ruido en la memoria a largo plazo.
- Puertos en `libs/ports/` mantienen la arquitectura hexagonal.

**Negativas:**
- pgvector menos performante que vector DBs dedicadas a gran escala
  (aceptable para nuestro volumen).
- La consolidación con LLM añade coste por ejecución.
- Los embeddings dependen del modelo elegido — cambiar modelo
  requiere re-indexar.
- Complejidad adicional frente a una solución simple de solo DB.

## Notas para Claude Code

- Los puertos de memoria viven en `libs/ports/memory.go`.
- Las implementaciones viven en `adapters/storage/` (Ent + pgvector).
- Working memory: Ent para persistencia, Valkey para acceso rápido.
- Episodic memory: solo Ent (consultas SQL).
- Semantic memory: Ent + pgvector (embeddings + búsqueda semántica).
- Nunca borrar SemanticFacts. Usar `Supersede()` para actualizar.
- El consolidador se ejecuta como background process tras cada
  ejecución completada.
