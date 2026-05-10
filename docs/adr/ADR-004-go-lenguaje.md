# ADR-004: Go as the primary development language

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The project needs a language that offers good performance, native concurrency,
self-contained binaries and a mature ecosystem for backend services.
The team values language simplicity, fast compilation times
and ease of deployment in containers.

## Decision

**Go** is adopted as the primary language for all services in the project.

### Mandatory reference guides

The code style follows a hierarchy of authoritative sources. Claude Code
must consult them in this order of precedence:

1. **Effective Go** (official language reference)
   https://go.dev/doc/effective_go
   Foundational principles: formatting with `gofmt`, naming conventions,
   error handling, interfaces, concurrency. It is from 2009 and does not cover
   generics or modules, but the design principles remain valid.

2. **Google Go Style Guide** (most complete and up-to-date normative guide)
   https://google.github.io/styleguide/go/
   Divided into three parts:
   - **Style Guide**: normative fundamentals (clarity, simplicity, conciseness).
   - **Style Decisions**: concrete decisions on style points.
   - **Best Practices**: proven patterns for robust and maintainable code.

3. **Uber Go Style Guide** (practical complement with Good/Bad examples)
   https://github.com/uber-go/guide/blob/master/style.md
   Especially useful for its contrasted examples and focus on
   performance, error handling and concurrency patterns.

4. **Go Code Review Comments** (official community wiki)
   https://go.dev/wiki/CodeReviewComments
   Concise list of common code review points. Resolves frequent
   debates: initialisms (URL, ID not Url, Id), slice declarations,
   error handling, etc.

### Concrete project rules

1. **All code is formatted with `gofmt`/`goimports`.** No exceptions.
   Format is not discussed — the tool decides.

2. **Linting with `golangci-lint`.** Runs in CI with the project
   configuration (`.golangci.yml`). Warnings are treated as errors.

3. **Explicit error handling.** Never `_` to ignore errors except
   documented cases (e.g.: `defer f.Close()` where the error is not
   actionable). Each error is handled or propagated with context:

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

4. **Small interfaces, defined by the consumer.** Interfaces
   are defined where they are consumed, not where they are implemented. Prefer
   interfaces of 1-2 methods (io.Reader, io.Writer as model).

5. **Table-driven tests.** Tests use subtests with a table of cases.
   Descriptive names in test cases, not "case 1", "case 2":

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

6. **Packages by responsibility, not by type.** A `models/`
   or `utils/` package is a signal of poor design. Organize by domain:
   `order/`, `customer/`, `shipping/`.

7. **Concurrency with clear patterns.** Use `context.Context` as the
   first parameter in functions that can be cancelled. Goroutines
   always have a stop mechanism (context, done channel). Never
   fire-and-forget goroutines in production.

8. **Go modules.** Every project uses Go modules. No vendor/ unless
   there is an explicit need for offline builds.

9. **Every long-running service MUST block in `main()` until SIGTERM/SIGINT.**
   A service whose `main()` returns immediately exits with code 0; container
   orchestrators (Docker Compose, Kubernetes) treat this as a crash and restart
   it in an infinite loop, masking the real problem.

   The mandatory pattern for all services:

   ```go
   func main() {
       ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
       defer stop()

       // ... initialisation ...

       log.Println("<service>: started")
       <-ctx.Done()          // blocks until signal received
       log.Println("<service>: stopping")

       // ... graceful shutdown ...
   }
   ```

   **Stub services** that are not yet implemented MUST use the same skeleton
   with `<-ctx.Done()` as the only blocking statement. An empty
   `func main() {}` is not a valid stub — it is a defect.

   ```go
   // ✅ Valid stub
   func main() {
       ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
       defer stop()
       log.Println("router: started (stub)")
       <-ctx.Done()
       log.Println("router: stopped")
   }

   // ❌ Invalid stub — causes infinite restart loop in any container runtime
   func main() {}
   ```

## Alternatives considered

- **Rust:** Superior performance and compile-time memory safety.
  Discarded due to steeper learning curve and significantly longer
  compilation times. The team has more experience with Go.

- **Java/Kotlin (Spring Boot):** Very mature enterprise ecosystem.
  Discarded due to higher runtime resource consumption (JVM) and
  configuration complexity. Go binaries are simpler
  to deploy in containers.

- **TypeScript (Node.js):** Would unify frontend and backend. Discarded
  due to lower performance under concurrent load and a less robust
  concurrency model for intensive backend services.

## Consequences

**Positive:**
- Static and lightweight binaries — Docker images under 20MB.
- Native concurrency with goroutines — scales without additional frameworks.
- Fast compilation — agile development cycle.
- Format unified by tooling — zero style debates.
- Rich standard library (net/http, database/sql, testing).

**Negative:**
- Generics are still relatively recent — some patterns are more verbose.
- Fewer third-party libraries than ecosystems such as Java or Node.js.
- The verbosity of error handling can be repetitive.

## Notes for Claude Code

- All generated Go code must comply with the reference guides listed.
  When in doubt about style, consult the Google Go Style Guide first.
- Always use `fmt.Errorf("context: %w", err)` to propagate errors.
  Never a bare `return err`.
- Define interfaces where they are consumed, not where they are implemented.
- Tests are always table-driven with descriptive subtests.
- Do not create generic packages such as `utils/`, `helpers/`, `common/`.
- When creating a new service, follow the hexagonal structure (ADR-001)
  adapted to Go: packages by domain with interfaces as ports.
- Always format with `goimports`. Imports are grouped into three
  blocks: stdlib, third-party, project-internal.
