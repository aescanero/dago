# ADR-002: TDD como estrategia de desarrollo y testing

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El equipo necesita una estrategia de testing que garantice cobertura desde el
inicio y que sirva como documentación viva del comportamiento del sistema.
Históricamente, los tests se han escrito después del código, resultando en
cobertura parcial y tests frágiles acoplados a detalles de implementación.

## Decisión

Se adopta **Test-Driven Development (TDD)** con el ciclo Red-Green-Refactor
como estrategia obligatoria para todo código de producción.

### Reglas concretas

1. **Red primero.** Antes de escribir cualquier código de producción, se escribe
   un test que falla. El test describe el comportamiento esperado, no la
   implementación.

2. **Green mínimo.** Se escribe el código mínimo necesario para que el test pase.
   No se anticipa funcionalidad futura ni se "mejora" el código en este paso.

3. **Refactor con red de seguridad.** Solo después de que el test pasa se
   refactoriza. Los tests existentes deben seguir pasando tras el refactor.

4. **Granularidad de los tests:**

   - **Tests unitarios** (`tests/unit/`): Cubren lógica de dominio y servicios.
     Aislados de infraestructura. Usan fakes de los puertos, nunca mocks de
     librerías externas. Deben ejecutarse en milisegundos.

   - **Tests de integración** (`tests/integration/`): Verifican que los
     adaptadores funcionan correctamente con la infraestructura real
     (base de datos, APIs). Se ejecutan contra entornos controlados.

   - **Tests de contrato** (`tests/contract/`): Validan que la implementación
     cumple las especificaciones formales (OpenAPI, schemas). Automatizan
     la verificación spec-first.

5. **Nomenclatura de tests.** Cada test describe un comportamiento en lenguaje
   de negocio:

   ```
   ✅ "should reject order when stock is insufficient"
   ✅ "should calculate discount for premium customers"
   ❌ "test createOrder method"
   ❌ "test case 1"
   ```

6. **Cobertura.** No se persigue un porcentaje arbitrario. Se persigue que
   cada comportamiento de negocio tenga al menos un test que lo documente.

## Alternativas consideradas

- **Testing post-implementación:** Más rápido inicialmente pero produce tests
  que verifican implementación en vez de comportamiento. Descartada.

- **BDD con Gherkin:** Útil para comunicación con stakeholders pero añade
  una capa de traducción. Se reserva para tests de aceptación si se necesitan
  en el futuro, pero no como estrategia principal.

- **Solo tests de integración:** Más realistas pero lentos y difíciles de
  diagnosticar. No sustituyen la retroalimentación rápida de tests unitarios.

## Consecuencias

**Positivas:**
- Los tests documentan el comportamiento esperado del sistema.
- El diseño del código mejora porque TDD fuerza interfaces claras.
- La confianza para refactorizar es alta.
- Los bugs se detectan en segundos, no en despliegue.

**Negativas:**
- Velocidad inicial percibida como más lenta (se compensa a medio plazo).
- Requiere disciplina para no saltarse el ciclo bajo presión.
- Riesgo de tests triviales si no se enfoca en comportamiento.

## Notas para Claude Code

- Si se te pide implementar una funcionalidad, genera siempre el fichero de
  test primero. Presenta el test, espera confirmación, y luego genera la
  implementación.
- Los tests unitarios del dominio usan fakes, no mocks de librería. Crea
  implementaciones in-memory de los puertos (definidos en ADR-001).
- Nombra los tests describiendo comportamiento de negocio, nunca métodos.
- Si se te pide "añadir un test", pregunta qué comportamiento se quiere
  verificar, no qué método se quiere testear.
