# ADR-010: API versioning by URL path and OpenAPI contract

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The system exposes a REST API consumed by the React frontend (ADR-009)
and potentially by external clients in the future. A versioning strategy
is needed that allows the API to evolve without breaking existing clients,
and a formal contract format that is the single source of truth for
both backend and frontend.

## Decision

### Versioning: URL path

**URL path versioning** is adopted as the API versioning strategy.

```
/api/v1/orders
/api/v1/customers/:id
/api/v2/orders          ← future version with breaking changes
```

### Contract: OpenAPI 3.1

**OpenAPI 3.1** is adopted as the REST API specification format.
The spec file lives in `specs/openapi.yaml` and is the central artifact
of the Spec Driven Development approach.

### Concrete rules

#### Versioning

1. **The version prefix is mandatory.** Every API route
   starts with `/api/v{N}/`. No unversioned endpoints exist.

2. **Semantics of the version number.** Only the major version
   is incremented (v1 → v2) when there are **breaking changes**:
   - Removal of an endpoint or field.
   - Type change of an existing field.
   - Semantic change of an existing endpoint.
   - Change in the error format.

   Non-breaking changes (adding optional fields, adding new endpoints)
   are made **within the same version**.

3. **Version coexistence.** When v2 is created, v1 remains active
   with a documented deprecation period. Both versions can
   coexist served by the same Go process.

4. **Grouping in Gin:**

   ```go
   v1 := router.Group("/api/v1")
   {
       v1.POST("/orders", orderHandlerV1.Create)
       v1.GET("/orders/:id", orderHandlerV1.GetByID)
   }

   v2 := router.Group("/api/v2")
   {
       v2.POST("/orders", orderHandlerV2.Create)
   }
   ```

5. **Internal code versioning.** If v1 and v2 share domain logic
   (which should be the norm), only the handlers differ.
   The domain does not know about versions — the version is a detail of the
   HTTP adapter.

#### OpenAPI

6. **Spec-first, not code-first.** The OpenAPI spec is written first.
   The code is implemented to comply with it. The spec is not generated
   from the code.

7. **Location and format:**

   ```
   specs/
   ├── openapi.yaml              # Main spec (references $ref)
   ├── paths/
   │   ├── orders.yaml           # Orders endpoints
   │   └── customers.yaml
   └── schemas/
       ├── order.yaml             # Reusable schemas
       ├── customer.yaml
       └── error.yaml             # Standard error format
   ```

8. **Standard error format across the entire API:**

   ```yaml
   ErrorResponse:
     type: object
     required: [code, message]
     properties:
       code:
         type: string
         description: Código de error de negocio (ej. ORDER_NOT_FOUND)
       message:
         type: string
         description: Mensaje legible para humanos
       details:
         type: array
         items:
           type: object
           properties:
             field:
               type: string
             reason:
               type: string
   ```

9. **TypeScript type generation for the frontend.** The TypeScript types of the
   API client in React (ADR-009) are generated automatically from
   the OpenAPI spec using `openapi-typescript`. This guarantees that the
   contract is enforced on both ends.

10. **Validation in CI.** The CI pipeline validates that the OpenAPI spec
    is valid (with tools such as `spectral` or `swagger-cli validate`)
    and optionally that the implementation meets the spec (contract tests).

11. **Auto-generated documentation.** The OpenAPI spec is served as
    interactive documentation (Swagger UI or Redoc) at `/api/docs`
    in development and staging environments. Not in production.

## Alternatives considered

### Versioning

- **Header versioning (`Accept: application/vnd.api.v1+json`):**
  More "pure" according to REST but less visible, harder to test
  (cannot be opened in a browser), and more complex to implement
  in Gin. Discarded for complexity without clear benefit.

- **Query parameter (`?version=1`):** Non-standard, easy to forget
  or lose when copying URLs. Discarded.

- **No versioning (always avoid breaking changes):** Idealistic
  but impractical in the long term. Discarded.

### Contract

- **gRPC + Protobuf:** Excellent for service-to-service communication
  but not suitable as a public API consumed by an SPA. Reserved
  for inter-service communication if needed in the future.

- **Standalone JSON Schema:** Covers only types, not endpoints
  or operations. OpenAPI is a superset that includes JSON Schema.

- **GraphQL:** Eliminates the versioning problem but adds significant
  complexity (resolvers, N+1, field-level authorization). Discarded
  for this project.

## Consequences

**Positive:**
- Versioning visible and explicit in the URL — no ambiguity.
- OpenAPI as the single contract — backend and frontend aligned.
- Automatic TypeScript type generation — zero contract drift.
- Interactive documentation for free from the spec.
- Automated validation in CI.

**Negative:**
- Maintaining the spec by hand requires discipline (mitigated with
  CI validation).
- Version coexistence adds code in handlers (acceptable
  because the domain is not duplicated).
- URL path versioning mixes contract into the URL (trade-off accepted
  for simplicity).

## Notes for Claude Code

- When creating a new endpoint, add it to `specs/openapi.yaml` first.
  Then implement the handler in Gin.
- Every route starts with `/api/v1/`. Never create endpoints without a
  version prefix.
- Errors always follow the `ErrorResponse` format defined in
  the spec. Never invent ad-hoc error formats.
- If asked to generate the API client for React, use the types
  generated from OpenAPI. Do not create manual types that duplicate
  the spec.
- The OpenAPI spec is the source of truth. If there is a discrepancy between
  the spec and the code, the spec is correct and the code must be fixed.
