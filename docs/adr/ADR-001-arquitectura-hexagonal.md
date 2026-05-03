# ADR-001: Hexagonal Architecture as structural pattern

**Status:** Accepted (revised: adapted to Go and monorepo)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The system needs a structure that allows the business logic to evolve
independently of infrastructure technologies (databases,
web frameworks, external services).

## Decision

**Hexagonal Architecture** (Ports & Adapters, Alistair Cockburn)
is adopted as the structural pattern for all services in the project.

### Mapping to the Go monorepo structure

```
libs/domain/       → Pure business logic (entities, value objects)
libs/ports/        → Interfaces (ports) defined by the domain
adapters/          → Shared port implementations
services/{s}/internal/ → Internal logic of each service
```

The dependency direction always points inward:

```
adapters/ → libs/ports/ → libs/domain/
services/{s}/internal/ → libs/ports/ → libs/domain/
```

The domain (`libs/domain/`) imports NOTHING from outside itself.

### Concrete rules

1. **The domain is the center.** All business logic resides in
   `libs/domain/`. It does not import Gin, Ent, go-redis or any
   infrastructure dependency.

2. **Ports are Go interfaces in `libs/ports/`.** They represent
   what the domain needs from the outside:

   ```go
   // libs/ports/storage.go
   type GraphRepository interface {
       Save(ctx context.Context, graph *domain.Graph) error
       FindByID(ctx context.Context, id uuid.UUID) (*domain.Graph, error)
   }

   // libs/ports/eventbus.go
   type EventPublisher interface {
       Publish(ctx context.Context, event *domain.Event) error
   }
   ```

3. **Adapters implement the ports.** They reside in `adapters/`
   (shared) or `services/{s}/internal/` (service-specific):

   ```go
   // adapters/storage/graph_repo.go
   type EntGraphRepository struct {
       client *ent.Client
   }

   func (r *EntGraphRepository) Save(ctx context.Context, g *domain.Graph) error {
       // Traduce domain.Graph → ent types, persiste
   }
   ```

4. **Ent types do not leave the adapter.** The adapter translates
   between Ent types and domain types. The domain works with
   its own types defined in `libs/domain/`.

5. **Dependency injection connects the layers.** The dependency graph
   is built in the `main.go` of each service or in `internal/config/`.

6. **Domain tests without infrastructure.** In-memory fakes of the
   ports are created; never mocks of external libraries.

## Alternatives considered

- **Classic layered architecture (N-tier):** Allows transitive dependencies
  that couple domain to infrastructure. Discarded.
- **Clean Architecture:** Same principles. Hexagonal is preferred
  for more concrete vocabulary (ports/adapters).

## Consequences

**Positive:** Domain is testable without infrastructure, changing the DB/broker
only affects the adapter, predictable structure.

**Negative:** More files and indirection, requires discipline.

## Notes for Claude Code

- Domain in `libs/domain/`. Ports in `libs/ports/`.
- Shared adapters in `adapters/`. Service-specific in `services/{s}/internal/`.
- If you detect imports of Gin, Ent or go-redis from `libs/`, it is a violation.
- Domain tests: in-memory fakes, not library mocks.
