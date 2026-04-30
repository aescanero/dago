# ADR-010: Versionado de API por URL path y contrato OpenAPI

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El sistema expone una API REST consumida por el frontend React (ADR-009)
y potencialmente por clientes externos en el futuro. Se necesita una
estrategia de versionado que permita evolucionar la API sin romper clientes
existentes, y un formato de contrato formal que sea la fuente única de
verdad tanto para el backend como para el frontend.

## Decisión

### Versionado: URL path

Se adopta **versionado por URL path** como estrategia de versionado de la API.

```
/api/v1/orders
/api/v1/customers/:id
/api/v2/orders          ← versión futura con breaking changes
```

### Contrato: OpenAPI 3.1

Se adopta **OpenAPI 3.1** como formato de especificación de la API REST.
El fichero spec vive en `specs/openapi.yaml` y es el artefacto central
del enfoque Spec Driven Development.

### Reglas concretas

#### Versionado

1. **El prefijo de versión es obligatorio.** Toda ruta de la API
   comienza con `/api/v{N}/`. No existen endpoints sin versión.

2. **Semántica del número de versión.** Solo se incrementa la versión
   major (v1 → v2) cuando hay **breaking changes**:
   - Eliminación de un endpoint o campo.
   - Cambio de tipo de un campo existente.
   - Cambio de semántica de un endpoint existente.
   - Cambio en el formato de errores.

   Los cambios no-breaking (añadir campos opcionales, añadir endpoints
   nuevos) se hacen **dentro de la misma versión**.

3. **Coexistencia de versiones.** Cuando se crea v2, v1 sigue activa
   con un periodo de deprecación documentado. Ambas versiones pueden
   coexistir servidas por el mismo proceso Go.

4. **Agrupación en Gin:**

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

5. **Versionado interno del código.** Si v1 y v2 comparten lógica
   de dominio (que debería ser la norma), solo los handlers difieren.
   El dominio no conoce versiones — la versión es un detalle del
   adaptador HTTP.

#### OpenAPI

6. **Spec-first, no code-first.** La spec OpenAPI se escribe primero.
   El código se implementa para cumplirla. No se genera la spec desde
   el código.

7. **Ubicación y formato:**

   ```
   specs/
   ├── openapi.yaml              # Spec principal (referencia $ref)
   ├── paths/
   │   ├── orders.yaml           # Endpoints de orders
   │   └── customers.yaml
   └── schemas/
       ├── order.yaml             # Esquemas reutilizables
       ├── customer.yaml
       └── error.yaml             # Formato de error estándar
   ```

8. **Formato de error estándar en toda la API:**

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

9. **Generación de tipos para el frontend.** Los tipos TypeScript del
   cliente API en React (ADR-009) se generan automáticamente desde
   la spec OpenAPI con `openapi-typescript`. Esto garantiza que el
   contrato se cumple en ambos extremos.

10. **Validación en CI.** El pipeline de CI valida que la spec OpenAPI
    es válida (con herramientas como `spectral` o `swagger-cli validate`)
    y opcionalmente que la implementación cumple la spec (tests de contrato).

11. **Documentación auto-generada.** La spec OpenAPI se sirve como
    documentación interactiva (Swagger UI o Redoc) en `/api/docs`
    en entornos de desarrollo y staging. No en producción.

## Alternativas consideradas

### Versionado

- **Header versioning (`Accept: application/vnd.api.v1+json`):**
  Más "puro" según REST pero menos visible, más difícil de probar
  (no se puede abrir en navegador), y más complejo de implementar
  en Gin. Descartado por complejidad sin beneficio claro.

- **Query parameter (`?version=1`):** Poco estándar, fácil de olvidar
  o perder en copias de URLs. Descartado.

- **Sin versionado (evitar breaking changes siempre):** Idealista
  pero impracticable a largo plazo. Descartado.

### Contrato

- **gRPC + Protobuf:** Excelente para comunicación entre servicios
  pero no adecuado como API pública consumida por una SPA. Se
  reserva para comunicación inter-servicios si se necesita en el futuro.

- **JSON Schema standalone:** Cubre solo los tipos, no los endpoints
  ni las operaciones. OpenAPI es un superset que incluye JSON Schema.

- **GraphQL:** Elimina el problema de versionado pero añade complejidad
  significativa (resolvers, N+1, autorización por campo). Descartado
  para este proyecto.

## Consecuencias

**Positivas:**
- Versionado visible y explícito en la URL — sin ambigüedad.
- OpenAPI como contrato único — backend y frontend alineados.
- Generación automática de tipos TypeScript — cero drift de contrato.
- Documentación interactiva gratis desde la spec.
- Validación automatizada en CI.

**Negativas:**
- Mantener la spec a mano requiere disciplina (mitigado con
  validación en CI).
- Coexistencia de versiones añade código en handlers (aceptable
  porque el dominio no se duplica).
- URL path versioning mezcla contrato en la URL (trade-off aceptado
  por simplicidad).

## Notas para Claude Code

- Al crear un nuevo endpoint, añádelo primero a `specs/openapi.yaml`.
  Luego implementa el handler en Gin.
- Toda ruta comienza con `/api/v1/`. Nunca crees endpoints sin prefijo
  de versión.
- Los errores siempre siguen el formato `ErrorResponse` definido en
  la spec. Nunca inventes formatos de error ad-hoc.
- Si se te pide generar el cliente API para React, usa los tipos
  generados desde OpenAPI. No crees tipos manuales que dupliquen
  la spec.
- La spec OpenAPI es la fuente de verdad. Si hay discrepancia entre
  la spec y el código, la spec tiene razón y el código debe corregirse.
