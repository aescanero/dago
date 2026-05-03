# ADR-005: GitHub Flow, semantic versioning and changelog

**Status:** Accepted (revised: changelog, releases, traceability)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The team needs a standardized workflow for the development
lifecycle. Additionally, following the adoption of sprints with
traceability (ADR-020), a complete chain is needed from
requirement to release:

```
ADR → Spec → Sprint → TODO → Commit → PR → Changelog → Release → Tag
```

## Decision

**GitHub Flow** is adopted as the branching strategy, **Conventional
Commits** as the strict commit format, **Semantic Versioning**
as the versioning scheme, and **automatic changelog generation**
as the mechanism for communicating changes.

### GitHub Flow

```
main (always deployable)
  │
  ├── sprint/001-bootstrap-monorepo    ← sprint branch
  ├── feature/add-graph-validation     ← feature branch
  ├── fix/shipping-cost-calculation    ← fix branch
  └── hotfix/critical-auth-bypass      ← urgent fix branch
```

### Concrete rules

#### Branches and PRs

1. **`main` is sacred.** Always deployable. Protected: no direct push.
   Everything enters via an approved PR.

2. **One branch per sprint or change.** Branches are ephemeral — they are
   deleted after merge.

3. **Branch naming:**

   ```
   sprint/<NNN>-<description>     → planned sprints (ADR-020)
   feature/<description>          → features outside a sprint
   fix/<description>              → non-urgent fixes
   hotfix/<description>           → critical production fixes
   refactor/<description>         → improvements without functional change
   docs/<description>             → documentation changes
   ```

4. **One PR per sprint.** The complete sprint is submitted as a PR that
   references the sprint document. The PR includes: sprint description,
   link to the document, and the traceability matrix.

5. **`main` protections:**
   - Require pull request reviews (minimum 1).
   - Require status checks to pass (lint, test, build).
   - Require branches to be up to date.
   - No force pushes. No deletions.

#### Conventional Commits (strict)

6. **Mandatory commit format:**

   ```
   <type>(<scope>): <imperative description> [SPRINT-XXX #N]

   [optional body with context]

   [Refs: #issue, ADR-xxx]
   [BREAKING CHANGE: description of the breaking change]
   ```

   **Types:**

   | Type | Use | Appears in changelog |
   |------|-----|---------------------|
   | `feat` | New feature | Yes (Features) |
   | `fix` | Bug fix | Yes (Bug Fixes) |
   | `perf` | Performance improvement | Yes (Performance) |
   | `refactor` | Refactoring without functional change | No |
   | `test` | Adding or modifying tests | No |
   | `docs` | Documentation | No |
   | `ci` | CI/CD changes | No |
   | `chore` | Maintenance tasks | No |
   | `build` | Build system changes | No |

   **Scopes:** name of the affected service or component:
   `orchestrator`, `executor`, `router`, `planner`, `auth-server`,
   `catalog`, `mcp-registry`, `agent-registry`, `dashboard`, `libs`,
   `adapters`, `specs`, `docs`.

   **Examples:**

   ```
   feat(orchestrator): add graph validation endpoint [SPRINT-001 #6]
   fix(executor): handle LLM timeout in react pattern [SPRINT-003 #4]
   test(libs): add unit tests for ABAC tag evaluation [SPRINT-002 #3]
   docs(views): update logical view with auth-server components [SPRINT-002 #7]

   feat(openapi)!: rename /graphs to /workflows [SPRINT-005 #1]

   BREAKING CHANGE: All /api/v1/graphs endpoints are now /api/v1/workflows.
   Clients must update their base URLs.
   ```

7. **The scope references the sprint and TODO.** This closes the
   traceability chain: from the changelog you reach the commit, from the commit
   the sprint, from the sprint the spec, from the spec the ADR.

8. **`BREAKING CHANGE` in the footer** for incompatible changes.
   This triggers a major version bump in semver.

#### Squash merge

9. **Squash and merge as strategy.** PRs are merged with
   squash. The squash message follows Conventional Commits and
   summarizes the sprint:

   ```
   feat(orchestrator): implement graph CRUD and validation [SPRINT-001]

   - Define graph schemas in OpenAPI
   - Add contract tests for graph endpoints
   - Implement CreateGraph, GetGraph, ListGraphs use cases
   - Create Ent schema and migration for Graph entity

   Refs: SPRINT-001, ADR-007, ADR-010, ADR-016
   ```

#### Semantic Versioning

10. **Semantic versioning of the system:**

    ```
    vMAJOR.MINOR.PATCH

    MAJOR → Breaking change in the public API (OpenAPI)
    MINOR → New endpoint, new pattern, new feature
    PATCH → Bug fixes, performance improvements
    ```

11. **Two levels of versioning:**

    - **System (dago):** Global version reflecting the public API.
      Tag: `v1.2.3`.
    - **Catalog packages:** Each package has its own independent semver
      (ADR-017). Package version != system version.

12. **Versioning is derived from commits.** A `feat:` commit →
    minor bump. A `fix:` commit → patch bump. A commit with
    `BREAKING CHANGE` → major bump. The version is not chosen
    manually.

#### Changelog

13. **Changelog generated automatically** from commits with
    Conventional Commits. **git-cliff** is used (open source, Rust,
    configurable) or equivalent.

14. **Changelog format:**

    ```markdown
    # Changelog

    ## [1.2.0] - 2026-05-15

    ### Features
    - **orchestrator:** Add graph validation endpoint ([SPRINT-001 #6])
    - **catalog:** Implement package publishing API ([SPRINT-002 #5])

    ### Bug Fixes
    - **executor:** Handle LLM timeout in react pattern ([SPRINT-003 #4])

    ### Performance
    - **adapters:** Optimize Valkey connection pooling ([SPRINT-004 #2])

    ### Breaking Changes
    - **openapi:** Rename /graphs to /workflows ([SPRINT-005 #1])

    ## [1.1.0] - 2026-05-01
    ...
    ```

15. **Each entry references the sprint and TODO.** An auditor can
    follow the chain: changelog entry → commit → sprint doc →
    spec → ADR.

16. **The changelog lives in `CHANGELOG.md` at the root** of the repo.
    It is updated automatically at each release.

#### Releases

17. **Release = tag + changelog + GitHub Release.** When a release is decided:

    ```
    1. git-cliff generates the updated changelog.
    2. CHANGELOG.md is committed.
    3. A semver tag is created (v1.2.0).
    4. GitHub Release is generated from the tag with the changelog.
    5. CI/CD deploys the affected services.
    ```

18. **Release frequency.** Changes are not accumulated indefinitely.
    A release is made when a sprint with significant functionality is
    completed or when there are critical fixes. At minimum one release
    per productive sprint.

19. **Individual service releases.** Although there is a global system
    version, each service can have independent releases
    via path-based triggers (ADR-013). The global changelog reflects all
    changes; deploys are selective.

### Complete traceability chain

```
ADR-016 (decision: node patterns)
  → specs/patterns/nodes/react.json (pattern spec)
    → SPRINT-003 (plan: implement react pattern)
      → SPRINT-003 #2 (TODO: react pattern tests)
        → test(executor): add react pattern tests [SPRINT-003 #2] (commit)
      → SPRINT-003 #4 (TODO: implement react handler)
        → feat(executor): implement react pattern handler [SPRINT-003 #4] (commit)
          → PR "SPRINT-003: Implement react pattern" (review)
            → CHANGELOG.md: "**executor:** Implement react pattern" (communication)
              → v1.3.0 (release)
                → Tag + GitHub Release + Deploy executor
```

Any analyst can traverse this chain in both directions:
from the ADR to the release, or from the release to the ADR.

### Integration with sprints (ADR-020)

```
1. planner generates sprint document with TODOs.
2. Branch sprint/NNN-description is created from main.
3. TODOs are executed in order (specs → tests → impl → docs).
4. Each commit follows Conventional Commits with [SPRINT-NNN #TODO].
5. PR is opened with reference to the sprint document.
6. reviewer (CI or human) reviews against plan and ADRs.
7. Squash merge with sprint summary message.
8. git-cliff updates CHANGELOG.md.
9. Release is created if appropriate (tag + GitHub Release).
10. Sprint document is closed with result.
```

## Alternatives considered

- **GitFlow:** Discarded for unnecessary complexity.
- **Trunk-Based Development:** Considered as a future evolution.
- **Manual changelog:** Error-prone and easily forgotten. Discarded.
- **release-please (Google):** Alternative to git-cliff. Automatically generates
  release PRs. More opinionated. Will be evaluated.

## Consequences

**Positive:**
- Complete traceability: ADR → spec → sprint → commit → changelog → release.
- Changelog generated, not written — always up to date.
- Versioning derived from commits — no manual decisions.
- Auditable by humans and machines.
- Conventional Commits as industry standard.

**Negative:**
- Conventional Commits requires discipline in every commit.
- Additional tooling (git-cliff) in the pipeline.
- Squash merge loses sprint granularity in the main history (mitigated:
  the PR and sprint document preserve the detail).

## Notes for Claude Code

- Commit format: `<type>(<scope>): <desc> [SPRINT-XXX #N]`.
- Types that appear in the changelog: `feat`, `fix`, `perf`.
- Scope: name of the service or component.
- `BREAKING CHANGE` in the footer for incompatible changes.
- One branch and one PR per sprint.
- The squash merge message summarizes the complete sprint.
- Never suggest direct commits to `main`.
- If a change is >400 lines, suggest splitting into smaller sprints.
- The changelog is generated with git-cliff — it is not edited manually.
