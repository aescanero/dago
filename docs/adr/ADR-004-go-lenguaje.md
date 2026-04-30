# ADR-004: Go como lenguaje principal de desarrollo

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El proyecto necesita un lenguaje que ofrezca buen rendimiento, concurrencia
nativa, binarios autocontenidos y un ecosistema maduro para servicios backend.
El equipo valora la simplicidad del lenguaje, los tiempos de compilación rápidos
y la facilidad de despliegue en contenedores.

## Decisión

Se adopta **Go** como lenguaje principal para todos los servicios del proyecto.

### Guías de referencia obligatorias

El estilo de código sigue una jerarquía de fuentes de autoridad. Claude Code
debe consultarlas en este orden de precedencia:

1. **Effective Go** (referencia oficial del lenguaje)
   https://go.dev/doc/effective_go
   Principios fundacionales: formato con `gofmt`, convenciones de naming,
   manejo de errores, interfaces, concurrencia. Es de 2009 y no cubre
   generics ni modules, pero los principios de diseño siguen vigentes.

2. **Google Go Style Guide** (guía normativa más completa y actualizada)
   https://google.github.io/styleguide/go/
   Dividida en tres partes:
   - **Style Guide**: fundamentos normativos (claridad, simplicidad, concisión).
   - **Style Decisions**: decisiones concretas sobre puntos de estilo.
   - **Best Practices**: patrones probados para código robusto y mantenible.

3. **Uber Go Style Guide** (complemento práctico con ejemplos Good/Bad)
   https://github.com/uber-go/guide/blob/master/style.md
   Especialmente útil por sus ejemplos contrastados y su enfoque en
   rendimiento, manejo de errores y patrones de concurrencia.

4. **Go Code Review Comments** (wiki oficial de la comunidad)
   https://go.dev/wiki/CodeReviewComments
   Lista concisa de puntos comunes en code reviews. Resuelve debates
   frecuentes: initialisms (URL, ID no Url, Id), declaración de slices,
   manejo de errores, etc.

### Reglas concretas del proyecto

1. **Todo código se formatea con `gofmt`/`goimports`.** Sin excepciones.
   No se discute formato — la herramienta decide.

2. **Linting con `golangci-lint`.** Se ejecuta en CI con la configuración
   del proyecto (`.golangci.yml`). Los warnings se tratan como errores.

3. **Manejo de errores explícito.** Nunca `_` para ignorar errores salvo
   casos documentados (ej: `defer f.Close()` donde el error no es
   accionable). Cada error se maneja o se propaga con contexto:

   ```go
   // ✅ Correcto: contexto añadido
   if err != nil {
       return fmt.Errorf("creating order for customer %s: %w", customerID, err)
   }

   // ❌ Incorrecto: error propagado sin contexto
   if err != nil {
       return err
   }

   // ❌ Incorrecto: error ignorado
   result, _ := doSomething()
   ```

4. **Interfaces pequeñas, definidas por el consumidor.** Las interfaces
   se definen donde se consumen, no donde se implementan. Preferir
   interfaces de 1-2 métodos (io.Reader, io.Writer como modelo).

5. **Table-driven tests.** Los tests usan subtests con tabla de casos.
   Nombres descriptivos en los casos de test, no "case 1", "case 2":

   ```go
   tests := []struct {
       name    string
       input   OrderRequest
       wantErr bool
   }{
       {name: "valid order creates successfully", ...},
       {name: "empty customer ID returns error", ...},
   }
   ```

6. **Packages por responsabilidad, no por tipo.** Un package `models/`
   o `utils/` es una señal de mal diseño. Organizar por dominio:
   `order/`, `customer/`, `shipping/`.

7. **Concurrencia con patrones claros.** Usar `context.Context` como
   primer parámetro en funciones que puedan cancelarse. Goroutines
   siempre con mecanismo de parada (context, done channel). Nunca
   goroutines "fire and forget" en producción.

8. **Go modules.** Todo proyecto usa Go modules. Sin vendor/ salvo
   necesidad explícita de builds offline.

## Alternativas consideradas

- **Rust:** Rendimiento superior y seguridad de memoria en compilación.
  Descartado por curva de aprendizaje más pronunciada y tiempos de
  compilación significativamente mayores. El equipo tiene más experiencia
  con Go.

- **Java/Kotlin (Spring Boot):** Ecosistema enterprise muy maduro.
  Descartado por mayor consumo de recursos en runtime (JVM) y
  complejidad de configuración. Los binarios de Go son más simples
  de desplegar en contenedores.

- **TypeScript (Node.js):** Unificaría frontend y backend. Descartado
  por menor rendimiento en carga concurrente y modelo de concurrencia
  menos robusto para servicios backend intensivos.

## Consecuencias

**Positivas:**
- Binarios estáticos y ligeros — imágenes Docker de <20MB.
- Concurrencia nativa con goroutines — escala sin frameworks adicionales.
- Compilación rápida — ciclo de desarrollo ágil.
- Formato unificado por herramientas — cero debates de estilo.
- Ecosistema estándar rico (net/http, database/sql, testing).

**Negativas:**
- Generics aún relativamente recientes — algunos patrones son más verbosos.
- Menos librerías de terceros que ecosistemas como Java o Node.js.
- La verbosidad del manejo de errores puede ser repetitiva.

## Notas para Claude Code

- Todo código Go generado debe cumplir las guías de referencia listadas.
  Ante duda sobre estilo, consulta primero Google Go Style Guide.
- Usa siempre `fmt.Errorf("contexto: %w", err)` para propagar errores.
  Nunca `return err` desnudo.
- Define interfaces donde se consumen, no donde se implementan.
- Los tests siempre son table-driven con subtests descriptivos.
- No crees packages genéricos como `utils/`, `helpers/`, `common/`.
- Al crear un nuevo servicio, sigue la estructura hexagonal (ADR-001)
  adaptada a Go: packages por dominio con interfaces como puertos.
- Formatea siempre con `goimports`. Los imports se agrupan en tres
  bloques: stdlib, terceros, internos del proyecto.
