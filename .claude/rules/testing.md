---
paths:
  - "**/*_test.go"
  - "test/**"
---

# Testing rules (TDD)

- Test first (Red), minimal implementation (Green), refactor.
- Unit tests: table-driven with descriptive subtests.
- In-memory port fakes, never external library mocks.
- Integration tests: `//go:build integration` tag, Testcontainers.
- Contract tests: validate against OpenAPI/AsyncAPI specs.
- Naming: `TestServiceName_Behavior_ExpectedResult`.
