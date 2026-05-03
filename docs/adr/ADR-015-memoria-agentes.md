# ADR-015: Agent memory architecture (three layers)

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

Dago nodes that behave as agents need memory to function effectively.
Without memory, each execution starts from scratch — the agent does not
remember previous decisions, does not learn from errors, and cannot
maintain context across executions of the same graph or across different
executions.

The OpenClaw memory model (MEMORY.md, daily notes, dreaming) demonstrates
that a three-layer architecture is effective. We adapt that model to dago,
replacing Markdown files with PostgreSQL (Ent), pgvector, and Valkey,
which are appropriate for a multi-user distributed system.

## Decision

A **three-layer memory architecture** is adopted with an asynchronous
consolidation process:

```
┌─────────────────────────────────────────────────────────────┐
│                     AGENT MEMORY                            │
│                                                             │
│  Layer 1: Working Memory (short-term)                       │
│  ─────────────────────────────────────                      │
│  What: State of the current graph execution                 │
│  Where: PostgreSQL (Ent) + Valkey                           │
│  Lifecycle: Exists while the graph is executing             │
│  OpenClaw equivalent: Daily notes (today + yesterday)       │
│                                                             │
│  Layer 2: Episodic Memory (medium-term)                     │
│  ─────────────────────────────────────                      │
│  What: History of past executions, results,                 │
│        decisions made by each node                          │
│  Where: PostgreSQL (Ent)                                    │
│  Lifecycle: Persists indefinitely, queryable                │
│  OpenClaw equivalent: Archived daily notes                  │
│                                                             │
│  Layer 3: Semantic Memory (long-term)                       │
│  ─────────────────────────────────────                      │
│  What: Distilled knowledge — facts, patterns,               │
│        preferences, lessons learned                         │
│  Where: pgvector (embeddings) + PostgreSQL (Ent)            │
│  Lifecycle: Curated, superseded but never deleted           │
│  OpenClaw equivalent: MEMORY.md + semantic search           │
│                                                             │
│  Consolidation (dreaming)                                   │
│  ─────────────────────────                                  │
│  What: Background process that promotes relevant data       │
│        from working → episodic → semantic                   │
│  Where: Background service in the orchestrator              │
│  When: After completing a graph execution                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Layer 1: Working Memory (short-term)

Live state of the current execution. This is what nodes read and write
during a graph execution.

```go
// Conceptual model of execution state
type ExecutionState struct {
    ExecutionID  uuid.UUID
    GraphID      uuid.UUID
    Status       ExecutionStatus
    CurrentNode  string
    Variables    map[string]any        // Accumulated graph variables
    Messages     []Message             // Conversational history
    NodeResults  map[string]NodeResult // Results per node
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

**Storage:**
- Full state in PostgreSQL (Ent) for persistence and recovery.
- Hot state in Valkey for fast access during execution.
- Each node transition updates both (write-through).

**Access:**
- Nodes receive relevant state via events (ADR-014).
- The orchestrator maintains the canonical version.
- Checkpointing after each transition for crash recovery.

**Lifecycle:**
- Created when an execution starts.
- Updated on each node transition.
- When the graph completes, archived as episodic memory.

### Layer 2: Episodic Memory (medium-term)

Complete history of past executions. Allows agents to query what
happened in previous executions of the same graph or similar graphs.

```go
// Conceptual model of the episodic record
type EpisodeRecord struct {
    ExecutionID   uuid.UUID
    GraphID       uuid.UUID
    UserID        uuid.UUID
    StartedAt     time.Time
    CompletedAt   time.Time
    Status        ExecutionStatus      // completed, failed, cancelled
    FinalState    map[string]any       // Final graph state
    NodeHistory   []NodeEpisode        // Sequence of executed nodes
    ErrorDetails  *string              // If it failed, why
    Metadata      map[string]string    // Tags, categories
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

**Storage:** PostgreSQL (Ent). Execution and node result tables
with indexes for efficient querying.

**Access:**
- Queryable by graph, user, date range, status.
- Nodes can query previous episodes as context
  (e.g., "the last 5 times we ran this graph, what happened?").
- Access via `EpisodeStore` port in `libs/ports/`.

**Lifecycle:**
- Created when an execution completes (archiving working memory).
- Persists indefinitely.
- The consolidation process extracts relevant facts toward
  semantic memory.

### Layer 3: Semantic Memory (long-term)

Distilled and generalised knowledge. Not the raw history but
facts, patterns, and lessons learned that the agent can use in
future executions.

```go
// Conceptual model of semantic memory
type SemanticFact struct {
    ID           uuid.UUID
    Content      string                // The fact in natural language
    Embedding    pgvector.Vector       // Embedding for semantic search
    FactType     FactType              // preference, lesson, pattern, fact
    Source       string                // Where it was learned from
    Confidence   float64               // 0.0 - 1.0
    SupersededBy *uuid.UUID            // If replaced by another fact
    CreatedAt    time.Time
    LastUsedAt   time.Time
    UseCount     int                   // How many times it has been queried
}
```

Fact types:
- **preference** — "The user prefers concise answers"
- **lesson** — "The scraping node fails on Mondays due to maintenance"
- **pattern** — "Financial analysis graphs work better with
  temperature 0.2"
- **fact** — "The payments API has a rate limit of 100 req/min"

**Storage:**
- PostgreSQL with the **pgvector** extension for embeddings.
- Embeddings are generated with a configurable embedding model.
- Semantic similarity search with `<=>` (cosine distance).

**Access:**
- Semantic search: "what do I know about the rate limit of API X?"
  returns the most relevant facts without needing an exact keyword.
- Access via `SemanticMemory` port in `libs/ports/`.
- Nodes can inject relevant facts into their context before
  executing.

**Lifecycle:**
- Created by the consolidation process (dreaming).
- Never deleted, only superseded: if a fact changes, the previous
  one is marked with `superseded_by` and a new one is created.
- `confidence` and `use_count` are updated with usage.
- Facts with low `confidence` and zero `use_count` are de-prioritised
  but not removed.

### Consolidation (dreaming)

Background process that promotes knowledge between layers:

```
Working Memory ──(on graph completion)──▶ Episodic Memory
                                               │
                                     (background process)
                                               │
                                               ▼
                                       Semantic Memory
```

**Trigger:** Runs after each completed graph execution.

**Process:**
1. Archives the execution state as an episodic record.
2. Analyses the episode looking for new or updated facts:
   - Were there recurring errors? → lesson learned.
   - Did the user express a preference? → preference.
   - Was a performance pattern discovered? → pattern.
3. Generates embeddings for new facts.
4. Checks if the fact already exists (semantic similarity search).
   - If it exists and is consistent: increment confidence.
   - If it exists and contradicts: supersede the previous one.
   - If it is new: insert with initial confidence.
5. Records in the `DREAMS` log for auditing.

**The consolidation process can use an LLM** to extract facts
from episodes — it is an application of the Reflection pattern
applied to system memory.

### Memory ports

```go
// libs/ports/memory.go

// WorkingMemory manages active execution state
type WorkingMemory interface {
    Get(ctx context.Context, executionID uuid.UUID) (*ExecutionState, error)
    Update(ctx context.Context, state *ExecutionState) error
    Checkpoint(ctx context.Context, executionID uuid.UUID) error
}

// EpisodeStore manages execution history
type EpisodeStore interface {
    Archive(ctx context.Context, state *ExecutionState) (*EpisodeRecord, error)
    FindByGraph(ctx context.Context, graphID uuid.UUID, limit int) ([]EpisodeRecord, error)
    FindByUser(ctx context.Context, userID uuid.UUID, limit int) ([]EpisodeRecord, error)
}

// SemanticMemory manages long-term knowledge
type SemanticMemory interface {
    Store(ctx context.Context, fact *SemanticFact) error
    Search(ctx context.Context, query string, limit int) ([]SemanticFact, error)
    Supersede(ctx context.Context, oldID uuid.UUID, newFact *SemanticFact) error
}

// MemoryConsolidator runs the dreaming process
type MemoryConsolidator interface {
    Consolidate(ctx context.Context, episode *EpisodeRecord) error
}
```

## Considered Alternatives

- **Markdown files (OpenClaw style):** Simple and transparent but
  does not scale to multi-user and is not efficiently queryable.

- **Dedicated vector DB (Pinecone, Weaviate, Chroma):** More powerful
  for vector search but adds infrastructure. pgvector covers the
  use case without adding another system.

- **PostgreSQL only (without pgvector):** Possible with full-text
  search but loses the semantic similarity search capability.

- **Valkey only for everything:** Fast but without durable persistence
  or semantic search. Not suitable for long-term memory.

- **Mem0 / LangMem:** Agent-specific memory solutions. Interesting
  but add an external dependency. We prefer to implement on top of
  our existing stack.

## Consequences

**Positive:**
- Three layers with clear responsibilities.
- pgvector reuses the existing PostgreSQL — no new infrastructure.
- Semantic search allows retrieving context without exact keywords.
- "Never delete" preserves the full history for auditing and learning.
- Automatic consolidation reduces noise in long-term memory.
- Ports in `libs/ports/` maintain hexagonal architecture.

**Negative:**
- pgvector less performant than dedicated vector DBs at large scale
  (acceptable for our volume).
- Consolidation with LLM adds cost per execution.
- Embeddings depend on the chosen model — changing models
  requires re-indexing.
- Additional complexity compared to a simple DB-only solution.

## Notes for Claude Code

- Memory ports live in `libs/ports/memory.go`.
- Implementations live in `adapters/storage/` (Ent + pgvector).
- Working memory: Ent for persistence, Valkey for fast access.
- Episodic memory: Ent only (SQL queries).
- Semantic memory: Ent + pgvector (embeddings + semantic search).
- Never delete SemanticFacts. Use `Supersede()` to update.
- The consolidator runs as a background process after each
  completed execution.
