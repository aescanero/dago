---
paths:
  - "specs/**"
  - "ent/schema/**"
---

# Reglas de specs (Spec Driven Development)

- Endpoint nuevo → definir en `specs/openapi.yaml` ANTES de implementar.
- Evento nuevo → definir en `specs/asyncapi.yaml` ANTES de implementar.
- Patrón nuevo → crear JSON Schema en `specs/patterns/` ANTES de implementar.
- Entidad nueva → crear schema Ent en `ent/schema/` + `go generate ./ent` + `atlas migrate diff`.
- Si hay discrepancia entre spec y código, la spec tiene razón.
