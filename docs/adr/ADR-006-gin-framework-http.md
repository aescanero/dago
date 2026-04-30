# ADR-006: Gin como framework HTTP

**Estado:** Aceptado (revisado: múltiples servicios HTTP)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El proyecto necesita un framework HTTP para exponer APIs REST.
Inicialmente solo el orchestrator exponía HTTP; tras la descomposición
de servicios (ADR-013, ADR-014), también exponen HTTP: auth-server,
catalog, mcp-registry y agent-registry.

## Decisión

Se adopta **Gin** (github.com/gin-gonic/gin) como framework HTTP
para todos los servicios que exponen API REST.

### Servicios con API HTTP

| Servicio | Endpoints principales |
|----------|----------------------|
| orchestrator | API REST grafos/ejecuciones + WebSocket (AG-UI) |
| auth-server | OAuth 2.1 (/authorize, /token, /revoke, JWKS) |
| catalog | CRUD paquetes, versionado |
| mcp-registry | Registro MCP, broker de invocaciones |
| agent-registry | Agent Cards A2A, discovery |

Los servicios de orquestación (executor, router, planner) NO exponen
HTTP — solo consumen/producen eventos Valkey (ADR-014).

### Reglas concretas

1. **Handlers delgados.** Bind → servicio de dominio → respuesta HTTP.
   Sin lógica de negocio en handlers.

2. **Context propagation.** `c.Request.Context()` al dominio, nunca
   `*gin.Context`.

3. **Errores centralizados.** Función `mapDomainError()` traduce
   errores de dominio → HTTP status codes.

4. **Rutas versionadas.** Todo bajo `/api/v1/` (ADR-010). Excepción:
   endpoints OAuth estándar del auth-server (`/authorize`, `/token`,
   `/.well-known/*`).

5. **Middlewares comunes.** Recovery, logging (slog), CORS, RequestID,
   auth (JWT validation via JWKS). Cada servicio compone los que
   necesita.

6. **`ShouldBindJSON`**, no `BindJSON`. Validación de negocio en el
   dominio, no en tags del binding.

## Notas para Claude Code

- Handlers en `services/{nombre}/internal/handler/`.
- Middlewares en `services/{nombre}/internal/middleware/` o compartidos
  en `adapters/auth/` (JWT validation).
- Todo servicio HTTP tiene `c.Request.Context()` → dominio.
- Los 5 servicios HTTP usan Gin. Los 3 workers (executor, router,
  planner) no tienen HTTP.
