---
paths:
  - "services/**"
  - "libs/**"
  - "adapters/**"
---

# Monorepo rules

- Single `go.mod` at the root. Never create a `go.mod` inside a service.
- Imports: `github.com/aescanero/dago/libs/...`, `github.com/aescanero/dago/adapters/...`.
- Service-private code: `services/{name}/internal/`.
- Shared code: `libs/` (domain, ports) or `adapters/` (implementations).
- Domain (`libs/domain/`, `libs/ports/`) must not import infrastructure.
- Ent types must not leave `adapters/storage/`.
