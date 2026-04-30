---
paths:
  - "**/*_test.go"
  - "test/**"
---

# Reglas de testing (TDD)

- Test primero (Red), implementación mínima (Green), refactor.
- Tests unitarios: table-driven con subtests descriptivos.
- Fakes in-memory de puertos, nunca mocks de librería externa.
- Tests de integración: tag `//go:build integration`, Testcontainers.
- Tests de contrato: validar contra specs OpenAPI/AsyncAPI.
- Naming: `TestServiceName_Behavior_ExpectedResult`.
