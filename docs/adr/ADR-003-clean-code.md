# ADR-003: Clean Code as code quality standard

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The project will be maintained by multiple developers over time.
A shared code quality standard is needed to reduce cognitive load
when reading, reviewing and modifying other people's code. Without an explicit standard,
each developer applies their own criteria and the code diverges in style
and quality.

## Decision

**Clean Code** principles are adopted as the quality standard, adapted
to the specific needs of the project. It is not adopted as dogma — each rule
has a pragmatic application threshold.

### Concrete rules

1. **Descriptive names without abbreviations.**

   ```
   ✅ calculateShippingCost(), customerRepository, isOrderExpired
   ❌ calcShpCst(), repo, check()
   ```

   Exception: iteration variables (`i`, `j`) and widely recognized
   language conventions (`ctx`, `err`, `req`, `res`).

2. **Small functions with a single responsibility.**
   - Indicative maximum: 20 lines per function.
   - If a function exceeds 20 lines, it must be justifiable (for example,
     an exhaustive switch over an enum, or a builder with many parameters).
   - Each function does one thing. If you need to use "and" to describe what
     it does, it is probably two functions.

3. **Maximum 3 parameters per function.**
   - If more are needed, group them into a configuration object or a DTO.
   - Booleans as parameters are a signal that the function does
     two things — consider splitting it.

4. **No redundant comments.** Code must be self-explanatory.
   Comments are reserved for:
   - **Why** something non-obvious is done (business decisions, workarounds).
   - **Warnings** about non-evident consequences.
   - **TODOs** with enough context for anyone to understand them.

   ```
   ❌ // Incrementa el contador
      counter++;

   ✅ // Retry necesario porque el proveedor X tiene rate limiting
      // agresivo los primeros 5 segundos tras autenticar.
      await retryWithBackoff(callProviderX, { maxAttempts: 3 });
   ```

5. **No dead code.** Code is not commented out "just in case". Version
   control already preserves it. If it is not used, it is removed.

6. **Explicit error handling.** Exceptions are not silenced. Every catch
   either handles the error with a concrete action, or propagates it with
   additional context. Never an empty `catch (e) {}`.

7. **Principle of least surprise.** Code must do what its name
   suggests, no more and no less. A `getUser()` method must not have
   side effects such as sending an email or modifying state.

8. **Pragmatic DRY.** Duplication is eliminated when it represents the same
   business concept. Two code fragments that are identical today but
   could evolve for different reasons should not be forced into a
   shared abstraction.

## Alternatives considered

- **Strict linter without a principles guide:** Captures format but not design.
  Used as a complement, not a substitute.

- **No explicit standard:** Relying on individual experience.
  Discarded because it produces friction in code reviews and stylistic divergence.

- **Literal adoption of the Clean Code book:** Some rules in the book are
  opinionated or outdated (for example, the insistence on avoiding all
  comments). A pragmatic adaptation is preferred.

## Consequences

**Positive:**
- Code is readable without the original author.
- Code reviews focus on logic, not style.
- Faster onboarding for new developers.

**Negative:**
- Requires judgment to apply the rules (they are not mechanical).
- Possible over-engineering when splitting functions too small.
- Occasional debates about what is "descriptive" or "small".

## Notes for Claude Code

- When generating code, use complete and descriptive names. Never abbreviate
  variable, function or class names.
- Keep functions below 20 lines. If the logic requires it,
  extract sub-functions with names that describe their purpose.
- Do not generate comments that repeat what the code already says. Add
  comments only for non-obvious decisions.
- If asked to refactor, apply these rules in order: first names,
  then function size, then removal of dead code.
- Never generate empty catch blocks. If you do not know how to handle an error,
  propagate with context.
