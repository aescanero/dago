# ADR-013: Monorepo with a single Go module

**Status:** Accepted (revised: 8 services after decomposition)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The dago system consists of 8 Go backend services and 1 React frontend.
Originally they were separate repositories with a dependency chain libs ← adapters
← services. Any change in libs triggered 6+ PRs and version drift.

## Decision

A **monorepo with a single Go module** is adopted (`go.mod` at the root).

### Services

| Service | Type | Responsibility |
|---------|------|----------------|
| orchestrator | Events + HTTP | Core: graphs, state, coordination, API, WebSocket |
| executor | Events | Worker: llm_call, tool_use, react, reflection, guardrail |
| router | Events | Worker: deterministic, llm, hybrid |
| planner | Events | NL → graph |
| auth-server | HTTP | OAuth 2.1, Identity Broker, ABAC |
| catalog | HTTP | Package catalogue, versioning |
| mcp-registry | HTTP | Registry + MCP broker |
| agent-registry | HTTP | A2A Agent Cards, discovery |

### Concrete rules

1. **A single `go.mod` at the root.** Module: `github.com/org/dago`.
   Imports: `github.com/org/dago/libs/...`, `github.com/org/dago/adapters/...`.

2. **No internal versions.** Services consume libs and adapters
   from the same commit. No `replace` directives.

3. **`internal/` per service.** Guarantees encapsulation:
   `services/executor/internal/` is not importable by other services.

4. **One binary and one Dockerfile per service.**

5. **Path-based CI/CD triggers:**

   ```yaml
   on:
     push:
       branches: [main]
       paths:
         - 'services/executor/**'
         - 'libs/**'
         - 'adapters/**'
         - 'go.mod'
   ```

6. **Unified CI** compiles, lints, and tests the entire monorepo.

7. **Independent dashboard** with its own `package.json` and pipeline.

8. **Makefile as unified interface:** build-all, build-{service},
   test, lint, generate, migrate-diff, migrate-apply, dashboard-dev.

9. **Docker Compose** for PostgreSQL + Valkey in local development.

10. **A single shared `ent/` directory.** Centralised data model.
    If a service needs its own DB, it creates an `ent/`
    inside its `internal/`.

11. **A single `migrations/` directory.** Atlas against the Ent schema.

## Notes for Claude Code

- Never create `go.mod` inside a service.
- Internal code: `services/{name}/internal/`.
- Shared code: `libs/` or `adapters/`.
- New service: `services/{name}/cmd/main.go`, `internal/`, `Dockerfile`.
- Frontend: `dashboard/` with its own `package.json`.
