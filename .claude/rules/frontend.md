---
paths:
  - "dashboard/**"
---

# Frontend rules (dashboard)

- TypeScript strict. No `any`.
- shadcn/ui + Tailwind CSS. No CSS-in-JS or styled-components.
- Data fetching: TanStack Query. Never `useEffect` for HTTP.
- Feature-based organisation (`features/{name}/`).
- Forms: shadcn Form + React Hook Form + Zod.
- Tests: Vitest + React Testing Library + MSW.
- OAuth tokens in memory, never in localStorage.
