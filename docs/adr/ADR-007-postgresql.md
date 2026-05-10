# ADR-007: PostgreSQL with Ent and Atlas

**Status:** Accepted (revised)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The system needs relational persistence with ACID transactions,
referential integrity and the ability to run complex queries. The SDD
approach requires the data schema to be a formal, versionable
and auditable artifact — just like OpenAPI and AsyncAPI.

## Decision

**PostgreSQL** is adopted as the database, **Ent** (entgo.io) as the ORM
and **Atlas** (atlasgo.io) as the migrations tool.

### Ent — Schema As Code

Ent generates statically typed code from Go schemas. It does not use
reflection (unlike GORM). Running `go generate ./ent` produces
a client with type-safe queries, mutations and predicates.

### Atlas — Versioned automatic migrations

Atlas calculates the diff between the desired state (Ent schemas) and the current one
(database), generates SQL migration files, and performs automatic linting
searching for destructive changes or locks.

```bash
atlas migrate diff nombre_migracion \
    --dir "file://migrations" \
    --to "ent://ent/schema" \
    --dev-url "docker://postgres/16/dev?search_path=public"
```

### Role within the hexagonal architecture

```
libs/ports/Repository (interface)
      ↓
adapters/storage/order_repo.go (implements with ent.Client)
      ↓
ent/ (generated code)
      ↓
PostgreSQL
```

The domain (libs/) does NOT depend on Ent. Adapters translate between
domain types and Ent types.

### Concrete rules

1. **Ent schemas = data spec.** Reviewed in PRs, analogous to OpenAPI.
2. **`go generate ./ent`** after each change. Generated code is committed.
3. **Atlas generates migrations**, never by hand. Reviewed in PRs.
   After `atlas migrate diff` (or any manual addition of files to `migrations/`),
   always run `atlas migrate hash --dir "file://migrations?format=golang-migrate"` to keep `atlas.sum` in sync.
   Without this step `atlas migrate apply` will fail with a checksum mismatch error.
4. **Migration linting in CI** (destructive changes, locks, data loss).
5. **Ent query builder** for CRUD. **Raw SQL** for complex queries.
6. **Transactions** with `client.Tx(ctx)` and defer rollback.
7. **Never expose Ent types** outside the adapter.
8. **UUIDs** as identifiers. **TIMESTAMPTZ** for timestamps, in UTC.
9. **`client.Schema.Create()` and `client.Schema.WriteTo()` are FORBIDDEN in service
   code.** They bypass Atlas, generate a parallel unversioned schema management path,
   and produce type divergence between the migration files and the live database
   (e.g. `text[]` vs `jsonb`). These methods are only permitted inside test helpers
   that use `enttest.Open` with an in-memory or throwaway database.

10. **Type-conversion migrations must follow the DROP DEFAULT → ALTER TYPE → SET DEFAULT
    pattern.** When changing a column type (e.g. `text[]` → `jsonb`), PostgreSQL applies
    the `USING` clause to existing row data but validates the column's `DEFAULT` expression
    separately — and cannot cast it automatically. Failing to drop the default first causes
    error `42804` even though the data conversion is valid.

    Mandatory pattern for any `ALTER COLUMN ... TYPE` migration:

    ```sql
    -- 1. Remove the old default (type-incompatible with the new type)
    ALTER TABLE "t" ALTER COLUMN "col" DROP DEFAULT;
    -- 2. Change the type; USING converts existing row data
    ALTER TABLE "t" ALTER COLUMN "col" TYPE new_type USING <expr>;
    -- 3. Restore the default with the correct new-type literal
    ALTER TABLE "t" ALTER COLUMN "col" SET DEFAULT <new_default>;
    ```

11. **Every migration must be verified by executing it against a real PostgreSQL instance
    before the PR is merged.** `atlas migrate lint` performs static analysis (destructive
    changes, locks) but does not execute SQL — it cannot catch runtime errors such as
    invalid casts, missing extensions, or constraint violations. Run `make migrate-test`
    after writing any migration file and before opening the PR.

## Alternatives considered

- **pgx pure:** Maximum control but without schema as code or type safety.
- **GORM:** Reflection, AutoMigrate is unsafe in production.
- **sqlc:** SQL-first, without schema as code or automatic migrations.
- **golang-migrate:** Without linting, "dirty" state on failures.

## Consequences

**Positive:** Schema as code, compile-time type safety, migrations
with linting, graph traversal, hooks, consistency with SDD.

**Negative:** Generated code increases the repo, slightly slower compilation,
learning curve, less fine-grained control than pure SQL.

## Notes for Claude Code

- Schemas in `ent/schema/`. After a change: `go generate ./ent`.
- Migrations: `atlas migrate diff` → `atlas migrate hash --dir "file://migrations?format=golang-migrate"`. Never by hand.
- If any file is added to `migrations/` manually, always rehash: `atlas migrate hash --dir "file://migrations?format=golang-migrate"`.
- The migrations directory uses `golang-migrate` format (`.up.sql` / `.down.sql` pairs). Always pass `?format=golang-migrate` to Atlas commands.
- Adapter in `adapters/storage/` uses `ent.Client`.
- Ent types do not leave the adapter. The domain has its own types.
- Always use context. Transactions with `client.Tx(ctx)`.
- **NEVER call `client.Schema.Create()` in service code.** Search for it in PRs — its presence is a blocking review comment. Only allowed in `enttest.Open` test helpers.
- **Any migration with `ALTER COLUMN ... TYPE` must use the three-step pattern** (rule 10). Always drop the default first.
- **Run `make migrate-test` after writing any migration.** Do not open a PR with a migration that has not been executed against a real PostgreSQL instance.
