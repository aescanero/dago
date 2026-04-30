# ADR-007: PostgreSQL con Ent y Atlas

**Estado:** Aceptado (revisado)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El sistema necesita persistencia relacional con transacciones ACID,
integridad referencial y capacidad de consultas complejas. El enfoque
SDD requiere que el schema de datos sea un artefacto formal, versionable
y auditable — igual que OpenAPI y AsyncAPI.

## Decisión

Se adopta **PostgreSQL** como base de datos, **Ent** (entgo.io) como ORM
y **Atlas** (atlasgo.io) como herramienta de migraciones.

### Ent — Schema As Code

Ent genera código tipado estáticamente a partir de schemas Go. No usa
reflexión (a diferencia de GORM). Ejecutar `go generate ./ent` produce
un cliente con queries, mutations y predicados type-safe.

### Atlas — Migraciones automáticas versionadas

Atlas calcula el diff entre el estado deseado (schemas Ent) y el actual
(base de datos), genera ficheros SQL de migración, y hace linting
automático buscando cambios destructivos o bloqueos.

```bash
atlas migrate diff nombre_migracion \
    --dir "file://migrations" \
    --to "ent://ent/schema" \
    --dev-url "docker://postgres/16/dev?search_path=public"
```

### Rol dentro de la arquitectura hexagonal

```
libs/ports/Repository (interfaz)
      ↓
adapters/storage/order_repo.go (implementa con ent.Client)
      ↓
ent/ (código generado)
      ↓
PostgreSQL
```

El dominio (libs/) NO depende de Ent. Los adaptadores traducen entre
tipos del dominio y tipos de Ent.

### Reglas concretas

1. **Schemas Ent = spec de datos.** Se revisan en PR, análogo a OpenAPI.
2. **`go generate ./ent`** tras cada cambio. Código generado se commitea.
3. **Atlas genera migraciones**, nunca a mano. Se revisan en PR.
4. **Linting de migraciones en CI** (destructivos, bloqueos, pérdida de datos).
5. **Query builder de Ent** para CRUD. **SQL raw** para queries complejas.
6. **Transacciones** con `client.Tx(ctx)` y defer rollback.
7. **Nunca exponer tipos Ent** fuera del adaptador.
8. **UUIDs** como identificadores. **TIMESTAMPTZ** para tiempos, en UTC.

## Alternativas consideradas

- **pgx puro:** Máximo control pero sin schema as code ni type safety.
- **GORM:** Reflexión, AutoMigrate inseguro en producción.
- **sqlc:** SQL-first, sin schema as code ni migraciones automáticas.
- **golang-migrate:** Sin linting, estado "dirty" ante fallos.

## Consecuencias

**Positivas:** Schema as code, type safety en compilación, migraciones
con linting, traversal de grafos, hooks, coherencia con SDD.

**Negativas:** Código generado aumenta repo, compilación algo más lenta,
curva de aprendizaje, menor control fino que SQL puro.

## Notas para Claude Code

- Schemas en `ent/schema/`. Tras cambio: `go generate ./ent`.
- Migraciones: `atlas migrate diff`. Nunca a mano.
- Adaptador en `adapters/storage/` usa `ent.Client`.
- Tipos Ent no salen del adaptador. El dominio tiene sus propios tipos.
- Context siempre. Transacciones con `client.Tx(ctx)`.
