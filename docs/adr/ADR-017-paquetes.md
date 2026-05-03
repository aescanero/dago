# ADR-017: Packages as the distribution unit

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

Dago needs a distribution unit that groups everything required to
execute a workflow: the graph definition, the configuration of agentic
nodes, the required MCP tools, UI components, and security metadata.
Without this formalisation, each workflow is a collection of loose
artefacts that are hard to version, share, and reuse.

## Decision

The **package** is defined as the atomic distribution unit of the
system. A package is a versioned artefact published to the catalog
service (ADR-013) that contains everything needed to install,
configure, and execute a workflow.

### Package structure

```json
{
  "$schema": "https://dago.dev/schemas/package.json",
  "package": {
    "id": "pkg_customer_support",
    "name": "Customer Support Workflow",
    "version": "2.1.0",
    "author": "team-support",
    "license": "MIT",
    "description": "Automated customer support with routing, escalation and resolution",
    "tags": ["department:support", "tier:production"],
    "dependencies": {
      "packages": [],
      "mcp_servers": ["zendesk-mcp", "jira-mcp"],
      "models": ["claude-sonnet-4-20250514"]
    }
  },

  "workflow": {
    "graph": {
      "entry_node": "classifier",
      "nodes": {
        "classifier": { "pattern": "router", "config": {} },
        "technical": { "pattern": "react", "config": {} },
        "billing": { "pattern": "llm_call", "config": {} },
        "review": { "pattern": "guardrail", "config": {} }
      },
      "edges": [
        {"type": "conditional", "from": "classifier", "conditions": []},
        {"type": "sequential", "from": "technical", "to": "review"},
        {"type": "sequential", "from": "billing", "to": "review"}
      ]
    },
    "memory": {
      "semantic_search": true,
      "episode_context": 3
    }
  },

  "skills": {
    "system_prompts": {
      "classifier": "You are a customer support classifier...",
      "technical": "You are a technical support agent..."
    },
    "few_shot_examples": [],
    "business_rules": [
      "Always escalate billing disputes over $500",
      "Never share internal ticket IDs with customers"
    ]
  },

  "tools": {
    "mcp_servers": [
      {
        "id": "zendesk-mcp",
        "required_tools": ["search_tickets", "update_ticket", "add_comment"]
      },
      {
        "id": "jira-mcp",
        "required_tools": ["create_issue", "assign_issue"]
      }
    ]
  },

  "ui": {
    "components": [
      {
        "id": "ticket_timeline",
        "type": "microfrontend",
        "module": "@dago-pkg/customer-support/TicketTimeline",
        "surface": "execution_detail",
        "props_schema": {
          "type": "object",
          "properties": {
            "executionId": { "type": "string", "format": "uuid" }
          }
        }
      }
    ],
    "a2ui_catalog": [
      {
        "id": "satisfaction_survey",
        "component_type": "form",
        "schema": {}
      }
    ],
    "ag_ui": {
      "streaming": true,
      "human_in_the_loop": true,
      "state_sync": true,
      "tool_visualization": true
    }
  }
}
```

### Package sections

**`package`** — Metadata: id, name, version (semver), author,
license, description, ABAC tags for access control, and
dependencies (other packages, MCP servers, LLM models).

**`workflow`** — Graph definition (ADR-016): nodes with their
patterns, edges with their flows, memory configuration.
This is the execution spec — the orchestrator consumes it to execute.

**`skills`** — Domain knowledge: system prompts per node,
few-shot examples, business rules. This is what differentiates a
generic agent from one specialised in a specific domain.

**`tools`** — Required tools: MCP servers with the specific
tools each node needs. The orchestrator verifies availability
against the mcp-registry before executing.

**`ui`** — Interface components: microfrontends that the dashboard
loads dynamically, A2UI components that agents can request during
execution, and AG-UI configuration for real-time communication.

### Versioning

Packages follow **Semantic Versioning** (semver):

- **Major** — Breaking changes to the workflow or the package API.
- **Minor** — New nodes, tools, or UI components without breaking changes.
- **Patch** — Fixes to prompts, business rules, configuration.

The catalogue retains all versions. In-progress executions use the
version they were started with, not the latest published one.

### Validation

When publishing a package, the catalog validates:

1. **Schema** — The JSON satisfies the package JSON Schema.
2. **Workflow** — The graph is structurally valid (ADR-016).
3. **Dependencies** — Referenced MCP servers exist in
   the mcp-registry. LLM models are available.
4. **Security** — ABAC tags are valid. The author has
   permission to publish with those tags.
5. **UI** — Referenced components exist in the microfrontend
   catalogue or are valid A2UI components.

### Lifecycle

```
Draft → Published → Active → Deprecated → Archived
```

- **Draft** — In development. Not executable.
- **Published** — Available for execution. Immutable.
- **Active** — Has in-progress executions.
- **Deprecated** — Replaced by a new version. Existing executions
  continue but new ones cannot be started.
- **Archived** — Retired. Historical query only.

## Notes for Claude Code

- The package schema lives in `specs/schemas/package.json`.
- Package validation is implemented in `services/catalog/internal/`.
- The orchestrator retrieves the package definition from the catalog via
  HTTP before executing a graph.
- Package tags are evaluated with ABAC (ADR-012) for access control.
- Package UI components are loaded dynamically by the dashboard
  (ADR-019).
