# ADR-009: React 19 + TypeScript + Vite (dashboard)

**Status:** Accepted (revised: shadcn/ui, AG-UI, microfrontends)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The system needs a web dashboard for monitoring, graph management,
package catalog and real-time communication with agents
during execution.

## Decision

**React 19** + **TypeScript** + **Vite** are adopted as the frontend stack.
The dashboard is complemented by decisions made in subsequent ADRs:

- **shadcn/ui + Tailwind CSS** as design system (ADR-019).
- **Module Federation** for package microfrontends (ADR-019).
- **AG-UI** as the agent-to-user protocol over WebSocket (ADR-018).
- **A2UI** as the declarative generative UI format (ADR-018).

### Concrete rules

1. **Vite as build tool.** Instant HMR, Rollup builds.
   Module Federation via `@originjs/vite-plugin-federation`.

2. **Strict TypeScript.** `strict: true`. No `any`.

3. **Feature-based organization**, not by technical type:

   ```
   dashboard/src/
   ├── api/                  # Types generated from OpenAPI
   ├── auth/                 # OAuth 2.1 / PKCE flow
   ├── features/
   │   ├── graphs/           # Visual editor, listing
   │   ├── executions/       # Real-time monitoring (AG-UI)
   │   ├── catalog/          # Package management
   │   ├── agents/           # A2A Agent Cards
   │   ├── mcp/              # MCP servers
   │   └── settings/         # Config, OUs, users
   ├── components/
   │   ├── ui/               # shadcn/ui (owned)
   │   ├── composed/         # dago business components
   │   └── shared/           # Theme, error boundary, skeletons
   ├── hooks/
   ├── layouts/
   ├── pages/
   └── lib/                  # utils, cn()
   ```

4. **Data fetching with TanStack Query.** Never `useEffect` for HTTP.

5. **API client generated from OpenAPI** (`openapi-typescript`).

6. **Real-time communication with AG-UI** over WebSocket. AG-UI
   client in `dashboard/src/api/agui/`.

7. **A2UI renderer** for declarative components from agents.
   Validates against the catalog of approved components.

8. **shadcn/ui + Tailwind CSS** for all styling. No CSS-in-JS,
   no styled-components, no CSS modules. Dark mode mandatory.

9. **Tests with Vitest + React Testing Library + MSW.**

10. **OAuth tokens in memory**, never in localStorage (ADR-012).

## Notes for Claude Code

- The dashboard lives in `dashboard/` with its own `package.json`.
- shadcn/ui components: `npx shadcn-ui@latest add`. They live in
  `src/components/ui/`.
- Styling: Tailwind utilities + CSS variables only. No inline styles.
- Forms: shadcn Form + React Hook Form + Zod.
- Data tables: shadcn DataTable + TanStack Table.
- Package microfrontends: Module Federation + lazy loading +
  Error Boundary + Suspense.
