---
paths:
  - "specs/**"
  - "ent/schema/**"
---

# Spec rules (Spec Driven Development)

- New endpoint → define in `specs/openapi.yaml` BEFORE implementing.
- New event → define in `specs/asyncapi.yaml` BEFORE implementing.
- New pattern → create JSON Schema in `specs/patterns/` BEFORE implementing.
- New entity → create Ent schema in `ent/schema/` + `go generate ./ent` + `atlas migrate diff` + `atlas migrate hash --dir "file://migrations?format=golang-migrate"`.
- If there is a discrepancy between spec and code, the spec is right.
- **NEVER call `client.Schema.Create()` or `client.Schema.WriteTo()` in service code.**
  These methods bypass Atlas, create an unversioned second schema management path,
  and cause type drift between migration files and the live database.
  They are only allowed inside `enttest.Open` test helpers with throwaway databases.
