# ADR-003: Clean Code como estándar de calidad de código

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El proyecto será mantenido por múltiples desarrolladores a lo largo del tiempo.
Se necesita un estándar compartido de calidad de código que reduzca la carga
cognitiva al leer, revisar y modificar código ajeno. Sin un estándar explícito,
cada desarrollador aplica sus propios criterios y el código diverge en estilo
y calidad.

## Decisión

Se adoptan principios de **Clean Code** como estándar de calidad, adaptados
a las necesidades concretas del proyecto. No se adopta como dogma — cada regla
tiene un umbral de aplicación pragmático.

### Reglas concretas

1. **Nombres descriptivos y sin abreviaturas.**

   ```
   ✅ calculateShippingCost(), customerRepository, isOrderExpired
   ❌ calcShpCst(), repo, check()
   ```

   Excepción: variables de iteración (`i`, `j`) y convenciones del lenguaje
   ampliamente reconocidas (`ctx`, `err`, `req`, `res`).

2. **Funciones pequeñas con una sola responsabilidad.**
   - Máximo orientativo: 20 líneas por función.
   - Si una función supera 20 líneas, debe poder justificarse (por ejemplo,
     un switch exhaustivo sobre un enum, o un builder con muchos parámetros).
   - Cada función hace una cosa. Si necesitas usar "y" para describir lo que
     hace, probablemente son dos funciones.

3. **Máximo 3 parámetros por función.**
   - Si necesitas más, agrupa en un objeto de configuración o un DTO.
   - Los booleanos como parámetros son una señal de que la función hace
     dos cosas — considera dividirla.

4. **Sin comentarios redundantes.** El código debe ser autoexplicativo.
   Los comentarios se reservan para:
   - **Por qué** se hace algo no obvio (decisiones de negocio, workarounds).
   - **Advertencias** sobre consecuencias no evidentes.
   - **TODOs** con contexto suficiente para que cualquiera los entienda.

   ```
   ❌ // Incrementa el contador
      counter++;

   ✅ // Retry necesario porque el proveedor X tiene rate limiting
      // agresivo los primeros 5 segundos tras autenticar.
      await retryWithBackoff(callProviderX, { maxAttempts: 3 });
   ```

5. **Sin código muerto.** No se comenta código "por si acaso". El control
   de versiones ya lo preserva. Si no se usa, se elimina.

6. **Manejo explícito de errores.** No se silencian excepciones. Cada catch
   o bien maneja el error con una acción concreta, o bien lo propaga con
   contexto adicional. Nunca `catch (e) {}` vacío.

7. **Principio de mínima sorpresa.** El código debe hacer lo que su nombre
   sugiere, ni más ni menos. Un método `getUser()` no debe tener efectos
   secundarios como enviar un email o modificar estado.

8. **DRY pragmático.** La duplicación se elimina cuando representa el mismo
   concepto de negocio. Dos fragmentos de código que hoy son iguales pero
   podrían evolucionar por razones distintas no deben forzarse en una
   abstracción compartida.

## Alternativas consideradas

- **Linter estricto sin guía de principios:** Captura formato pero no diseño.
  Se usa como complemento, no como sustituto.

- **No definir estándar explícito:** Confianza en la experiencia individual.
  Descartada porque produce fricción en code reviews y divergencia estilística.

- **Adopción literal del libro Clean Code:** Algunas reglas del libro son
  opinadas o anticuadas (por ejemplo, la insistencia en evitar todo
  comentario). Se prefiere una adaptación pragmática.

## Consecuencias

**Positivas:**
- El código es legible sin necesidad del autor original.
- Las code reviews se centran en lógica, no en estilo.
- Onboarding más rápido para nuevos desarrolladores.

**Negativas:**
- Requiere criterio para aplicar las reglas (no son mecánicas).
- Posible sobre-ingeniería al dividir funciones demasiado pequeñas.
- Debates ocasionales sobre qué es "descriptivo" o "pequeño".

## Notas para Claude Code

- Al generar código, usa nombres completos y descriptivos. Nunca abrevies
  nombres de variables, funciones o clases.
- Mantén las funciones por debajo de 20 líneas. Si la lógica lo requiere,
  extrae subfunciones con nombres que describan su propósito.
- No generes comentarios que repitan lo que el código ya dice. Añade
  comentarios solo para decisiones no obvias.
- Si se te pide refactorizar, aplica estas reglas en orden: primero nombres,
  luego tamaño de funciones, luego eliminación de código muerto.
- Nunca generes bloques catch vacíos. Si no sabes cómo manejar un error,
  propaga con contexto.
