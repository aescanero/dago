# ADR-006: Gin as HTTP framework

**Status:** Accepted (revised: multiple HTTP services)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The project needs an HTTP framework to expose REST APIs.
Initially only the orchestrator exposed HTTP; after the service
decomposition (ADR-013, ADR-014), the following also expose HTTP:
auth-server, catalog, mcp-registry and agent-registry.

## Decision

**Gin** (github.com/gin-gonic/gin) is adopted as the HTTP framework
for all services that expose a REST API.

### Services with HTTP API

| Service | Main endpoints |
|---------|----------------|
| orchestrator | REST API graphs/executions + WebSocket (AG-UI) |
| auth-server | OAuth 2.1 (/authorize, /token, /revoke, JWKS) |
| catalog | Package CRUD, versioning |
| mcp-registry | MCP registry, invocation broker |
| agent-registry | A2A Agent Cards, discovery |

Orchestration services (executor, router, planner) do NOT expose
HTTP — they only consume/produce Valkey events (ADR-014).

### Concrete rules

1. **Thin handlers.** Bind → domain service → HTTP response.
   No business logic in handlers.

2. **Context propagation.** `c.Request.Context()` to the domain, never
   `*gin.Context`.

3. **Centralized errors.** A `mapDomainError()` function translates
   domain errors → HTTP status codes.

4. **Versioned routes.** Everything under `/api/v1/` (ADR-010). Exception:
   standard OAuth endpoints of auth-server (`/authorize`, `/token`,
   `/.well-known/*`).

5. **Common middlewares.** Recovery, logging (slog), CORS, RequestID,
   auth (JWT validation via JWKS). Each service composes the ones
   it needs.

6. **`ShouldBindJSON`**, not `BindJSON`. Business validation in the
   domain, not in binding tags.

## Notes for Claude Code

- Handlers in `services/{name}/internal/handler/`.
- Middlewares in `services/{name}/internal/middleware/` or shared
  in `adapters/auth/` (JWT validation).
- Every HTTP service uses `c.Request.Context()` → domain.
- The 5 HTTP services use Gin. The 3 workers (executor, router,
  planner) have no HTTP.
