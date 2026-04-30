---
paths:
  - "dashboard/**"
---

# Reglas del frontend (dashboard)

- TypeScript strict. Sin `any`.
- shadcn/ui + Tailwind CSS. Sin CSS-in-JS ni styled-components.
- Data fetching: TanStack Query. Nunca `useEffect` para HTTP.
- Organización por feature (`features/{nombre}/`).
- Formularios: shadcn Form + React Hook Form + Zod.
- Tests: Vitest + React Testing Library + MSW.
- Tokens OAuth en memoria, nunca en localStorage.
