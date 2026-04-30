---
paths:
  - "services/**"
  - "libs/**"
  - "adapters/**"
---

# Reglas del monorepo

- Un solo `go.mod` en la raíz. Nunca crear `go.mod` dentro de un servicio.
- Imports: `github.com/org/dago/libs/...`, `github.com/org/dago/adapters/...`.
- Código interno de servicio: `services/{nombre}/internal/`.
- Código compartido: `libs/` (dominio, puertos) o `adapters/` (implementaciones).
- El dominio (`libs/domain/`, `libs/ports/`) no importa infraestructura.
- Tipos Ent no salen de `adapters/storage/`.
