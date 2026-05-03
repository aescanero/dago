# ADR-020: Reduced sprints with full traceability

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The project is developed with AI agent assistance (Claude Code).
The development process must be fully traceable and auditable — a human
or machine analyst must be able to examine each sprint, understand what
was done, why, and verify that the result meets the specifications. This
is especially important when code is generated or assisted by AI:
the transparency of the process is as relevant as the quality of the
result.

## Decision

A **reduced sprints** model is adopted where each sprint has a limited
scope, explicit documentation, tests derived from specifications, and
full traceability from requirement to implementation.

### Sprint structure

Each sprint is documented in a Markdown file in `docs/sprints/`:

```
docs/sprints/
├── SPRINT-001-bootstrap-monorepo.md
├── SPRINT-002-auth-server-oauth.md
├── SPRINT-003-catalog-packages.md
└── ...
```

### Sprint document format

```markdown
# SPRINT-XXX: [Descriptive title]

## Metadata
- **Start date:** YYYY-MM-DD
- **End date:** YYYY-MM-DD
- **Status:** planned | in progress | completed | cancelled
- **Applied ADRs:** ADR-001, ADR-007, ADR-013
- **Affected specs:** openapi.yaml (paths/graphs.yaml), asyncapi.yaml
- **Planning agent:** planner
- **Reviewed by:** [human or reviewer agent]

## Sprint objective
[Clear and concise description of what is achieved when this sprint is
completed. Must be verifiable — when finished, it can be checked
unambiguously whether it was met or not.]

## Scope
### Included
- [Explicit list of what is implemented]

### Excluded
- [Explicit list of what is NOT implemented and why]

## Dependencies
- **Required previous sprints:** SPRINT-XXX, SPRINT-YYY
- **Specs that must exist:** [specs assumed to be ready]
- **Required infrastructure:** [PostgreSQL, Valkey, etc.]

## TODOs

### 1. [spec] Define schemas in OpenAPI
- **Agent:** developer
- **Skill:** new-endpoint
- **Affected spec:** specs/paths/graphs.yaml
- **Acceptance criterion:** The schema is valid and consistent with ADR-010.
- **Depends on:** none

### 2. [test] Contract tests for GET /api/v1/graphs
- **Agent:** qa
- **Skill:** contract-test
- **Test location:** test/contract/graphs_test.go
- **What it verifies:** That the endpoint returns 200 with the schema defined
  in OpenAPI, 401 without a token, 403 without appropriate ABAC tags.
- **Reference spec:** specs/paths/graphs.yaml
- **Depends on:** #1

### 3. [test] Unit tests for the CreateGraph use case
- **Agent:** qa
- **Skill:** unit-test
- **Test location:** libs/domain/graph/service_test.go
- **What it verifies:**
  - A valid graph is created correctly.
  - A graph without entry_node returns a validation error.
  - A graph with a cycle without max_iterations is rejected.
- **Reference spec:** specs/patterns/graph.json (validation)
- **Depends on:** none

### 4. [data] Create Ent schema for Graph
- **Agent:** developer
- **Skill:** new-entity
- **Location:** ent/schema/graph.go
- **Depends on:** none

### 5. [impl] Implement CreateGraph use case
- **Agent:** developer
- **Skill:** new-endpoint
- **Location:** libs/domain/graph/service.go
- **Satisfies test:** #3
- **Depends on:** #3, #4

### 6. [impl] Implement Gin handler for POST /api/v1/graphs
- **Agent:** developer
- **Skill:** new-endpoint
- **Location:** services/orchestrator/internal/handler/graph_handler.go
- **Satisfies test:** #2
- **Depends on:** #2, #5

### 7. [docs] Update orchestrator component diagram
- **Agent:** docs
- **Skill:** logical-view
- **Location:** docs/views/logical/components/orchestrator.md
- **Depends on:** #6

## Traceability matrix

| Spec | Test | Implementation | Location |
|------|------|----------------|----------|
| openapi.yaml#/paths/graphs | contract/graphs_test.go | handler/graph_handler.go | orchestrator |
| patterns/graph.json | domain/graph/service_test.go | domain/graph/service.go | libs |
| ent/schema/graph.go | (Atlas migration) | adapters/storage/graph_repo.go | adapters |

## Sprint result
[Completed when the sprint finishes]

### Tests executed
- Total: X
- Passed: X
- Failed: 0

### Files created/modified
[List generated automatically or manually]

### Decisions made during the sprint
[Any unplanned decision made during implementation.
If significant, it is proposed as an ADR.]

### Reviewer observations
[Feedback from the reviewer agent or the human who reviewed the sprint]
```

### Concrete rules

#### Planning

1. **Reduced sprints.** Each sprint has a scope that can be completed
   in 1-3 days of work. If the scope is larger, it is split into
   smaller sprints.

2. **Tests first in the plan.** Test TODOs are defined before
   implementation TODOs. Each test references the spec it is derived
   from and describes exactly what it verifies.

3. **Explicit scope.** Both what is included and what is excluded are
   documented. "Authentication is not implemented in this sprint because
   it depends on SPRINT-005" is valuable information.

4. **Dependencies between TODOs.** Each TODO indicates what others it
   depends on. This defines the execution order and enables
   parallelisation when there are no dependencies.

5. **Mandatory traceability matrix.** Each spec → test →
   implementation is mapped explicitly. If a spec has no test,
   it is a gap. If a test does not reference a spec, it
   should not exist.

#### Execution

6. **Test-first within the sprint.** Sprint tests are written (Red)
   before the implementation (Green). TODOs are executed in the order
   defined by their dependencies.

7. **Each TODO is committed separately** with a message that
   references the sprint and the TODO number:

   ```
   feat: define graph schemas in OpenAPI [SPRINT-001 #1]
   test: add contract tests for GET /api/v1/graphs [SPRINT-001 #2]
   feat: implement CreateGraph use case [SPRINT-001 #5]
   ```

8. **One PR per sprint.** The complete sprint is submitted as a single
   PR that includes all changes. The PR references the sprint document.

#### Closure

9. **Documented result.** When closing the sprint, the result section
   is completed: tests executed, modified files, unplanned decisions.

10. **Mandatory review.** The `reviewer` agent (in CI) or a human
    reviews the complete sprint against the plan. Discrepancies
    are documented.

11. **Closed sprint is immutable.** Once completed, the sprint document
    is not modified. If there are corrections, they are created in a
    new sprint that references the previous one.

### How the planner agent generates sprints

The `planner` agent receives a requirement and generates the sprint
document following this sequence:

```
1. Analyse the requirement against ADRs and specs.
2. Identify which specs need to be created or modified.
3. Derive tests from the specs (contract tests, unit tests).
4. Define the implementation that will make the tests pass.
5. Identify affected documentation.
6. Build the traceability matrix.
7. Estimate whether it fits in one sprint or needs to be split.
8. Generate the sprint document.
```

The order of TODO generation within the sprint is always:

```
specs → tests → data (Ent) → implementation → documentation
```

This ensures that specs govern tests, tests govern implementation,
and documentation reflects the result — not the other way around.

## Considered Alternatives

- **No formal sprints (ad-hoc development):** Maximum initial velocity
  but zero traceability. Discarded because process auditability is a
  requirement.

- **Classic Scrum sprints (2 weeks):** Too long for an AI-assisted
  project where throughput is higher. Reduced sprints (1-3 days) adapt
  better to the working pace with Claude Code.

- **Kanban without sprints:** Continuous flow. Discarded because it
  loses the notion of "completed and reviewed unit of work" that is
  essential for traceability.

- **GitHub issues only:** Possible but does not capture the reasoning
  behind the plan or the traceability matrix. Issues can be used as
  a complement for tracking.

## Consequences

**Positive:**
- Full traceability: spec → test → code → documentation.
- Any analyst can reconstruct the build process.
- Tests derived from specs, not from the implementation.
- Unplanned decisions are documented explicitly.
- Reduced sprints allow frequent review and early correction.
- The planner agent generates auditable and reproducible plans.

**Negative:**
- Documentation overhead per sprint (mitigated because the planner
  generates it automatically).
- Rigidity: changes during a sprint require updating the plan.
- Accumulation of sprint documents (mitigable with archiving).

## Notes for Claude Code

- Sprint documents live in `docs/sprints/`.
- The `planner` agent generates the complete sprint document.
- TODO order: specs → tests → data → implementation → docs.
- Each commit references the sprint and the TODO number.
- One PR per sprint. Mandatory review before merge.
- The traceability matrix is mandatory in every sprint.
- When closing the sprint, complete the result section.
