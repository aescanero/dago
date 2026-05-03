# ADR-002: TDD as development and testing strategy

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The team needs a testing strategy that guarantees coverage from the
start and serves as living documentation of system behavior.
Historically, tests have been written after the code, resulting in
partial coverage and fragile tests coupled to implementation details.

## Decision

**Test-Driven Development (TDD)** with the Red-Green-Refactor cycle
is adopted as the mandatory strategy for all production code.

### Concrete rules

1. **Red first.** Before writing any production code, write
   a test that fails. The test describes the expected behavior, not the
   implementation.

2. **Minimum green.** Write the minimum code necessary to make the test pass.
   Do not anticipate future functionality or "improve" the code in this step.

3. **Refactor with a safety net.** Only after the test passes should you
   refactor. Existing tests must continue to pass after the refactor.

4. **Test granularity:**

   - **Unit tests** (`tests/unit/`): Cover domain logic and services.
     Isolated from infrastructure. Use fakes of the ports, never mocks of
     external libraries. Must execute in milliseconds.

   - **Integration tests** (`tests/integration/`): Verify that
     adapters work correctly with real infrastructure
     (database, APIs). Run against controlled environments.

   - **Contract tests** (`tests/contract/`): Validate that the implementation
     meets the formal specifications (OpenAPI, schemas). They automate
     spec-first verification.

5. **Test naming.** Each test describes a behavior in business
   language:

   ```
   ✅ "should reject order when stock is insufficient"
   ✅ "should calculate discount for premium customers"
   ❌ "test createOrder method"
   ❌ "test case 1"
   ```

6. **Coverage.** No arbitrary percentage is pursued. The goal is that
   every business behavior has at least one test that documents it.

## Alternatives considered

- **Post-implementation testing:** Faster initially but produces tests
  that verify implementation rather than behavior. Discarded.

- **BDD with Gherkin:** Useful for communication with stakeholders but adds
  a translation layer. Reserved for acceptance tests if needed
  in the future, but not as the primary strategy.

- **Integration tests only:** More realistic but slow and hard to
  diagnose. They do not replace the fast feedback of unit tests.

## Consequences

**Positive:**
- Tests document the expected behavior of the system.
- Code design improves because TDD forces clear interfaces.
- Confidence to refactor is high.
- Bugs are detected in seconds, not at deployment.

**Negative:**
- Initial perceived velocity is slower (compensated in the medium term).
- Requires discipline to not skip the cycle under pressure.
- Risk of trivial tests if not focused on behavior.

## Notes for Claude Code

- If asked to implement a feature, always generate the test file
  first. Present the test, wait for confirmation, and then generate the
  implementation.
- Unit tests for the domain use fakes, not library mocks. Create
  in-memory implementations of the ports (defined in ADR-001).
- Name tests describing business behavior, never methods.
- If asked to "add a test", ask what behavior is to be
  verified, not what method is to be tested.
