# ADR-001: Arquitectura Hexagonal como patrón estructural

**Estado:** Aceptado (revisado: adaptado a Go y monorepo)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El sistema necesita una estructura que permita evolucionar la lógica de negocio
de forma independiente a las tecnologías de infraestructura (bases de datos,
frameworks web, servicios externos).

## Decisión

Se adopta **Arquitectura Hexagonal** (Ports & Adapters, Alistair Cockburn)
como patrón estructural para todos los servicios del proyecto.

### Mapeo a la estructura Go del monorepo

```
libs/domain/       → Lógica de negocio pura (entidades, value objects)
libs/ports/        → Interfaces (puertos) definidos por el dominio
adapters/          → Implementaciones compartidas de puertos
services/{s}/internal/ → Lógica interna de cada servicio
```

La dirección de dependencia es siempre hacia dentro:

```
adapters/ → libs/ports/ → libs/domain/
services/{s}/internal/ → libs/ports/ → libs/domain/
```

El dominio (`libs/domain/`) no importa NADA fuera de sí mismo.

### Reglas concretas

1. **El dominio es el centro.** Toda lógica de negocio reside en
   `libs/domain/`. No importa Gin, Ent, go-redis ni ninguna
   dependencia de infraestructura.

2. **Los puertos son interfaces Go en `libs/ports/`.** Representan
   lo que el dominio necesita del exterior:

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

3. **Los adaptadores implementan los puertos.** Residen en `adapters/`
   (compartidos) o `services/{s}/internal/` (específicos del servicio):

   ```go
   // adapters/storage/graph_repo.go
   type EntGraphRepository struct {
       client *ent.Client
   }

   func (r *EntGraphRepository) Save(ctx context.Context, g *domain.Graph) error {
       // Traduce domain.Graph → ent types, persiste
   }
   ```

4. **Los tipos Ent no salen del adaptador.** El adaptador traduce
   entre tipos de Ent y tipos del dominio. El dominio trabaja con
   sus propios tipos definidos en `libs/domain/`.

5. **La inyección de dependencias conecta las capas.** En el `main.go`
   de cada servicio o en `internal/config/` se construye el grafo de
   dependencias.

6. **Tests del dominio sin infraestructura.** Se crean fakes in-memory
   de los puertos, nunca mocks de librerías externas.

## Alternativas consideradas

- **Arquitectura en capas clásica (N-tier):** Permite dependencias
  transitivas que acoplan dominio a infraestructura. Descartada.
- **Clean Architecture:** Mismos principios. Se prefiere Hexagonal
  por vocabulario más concreto (puertos/adaptadores).

## Consecuencias

**Positivas:** Dominio testeable sin infra, cambiar DB/broker solo
afecta al adaptador, estructura predecible.

**Negativas:** Más ficheros e indirección, requiere disciplina.

## Notas para Claude Code

- Dominio en `libs/domain/`. Puertos en `libs/ports/`.
- Adaptadores compartidos en `adapters/`. Específicos en `services/{s}/internal/`.
- Si detectas import de Gin, Ent o go-redis desde `libs/`, es una violación.
- Tests del dominio: fakes in-memory, no mocks de librería.
