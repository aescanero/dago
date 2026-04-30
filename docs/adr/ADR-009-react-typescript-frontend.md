# ADR-009: React 19 + TypeScript + Vite (dashboard)

**Estado:** Aceptado (revisado: shadcn/ui, AG-UI, microfrontales)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El sistema necesita un dashboard web para monitorización, gestión de
grafos, catálogo de paquetes y comunicación en tiempo real con agentes
durante la ejecución.

## Decisión

Se adopta **React 19** + **TypeScript** + **Vite** como stack frontend.
El dashboard se complementa con decisiones tomadas en ADRs posteriores:

- **shadcn/ui + Tailwind CSS** como design system (ADR-019).
- **Module Federation** para microfrontales de paquetes (ADR-019).
- **AG-UI** como protocolo agente↔usuario sobre WebSocket (ADR-018).
- **A2UI** como formato de UI generativa declarativa (ADR-018).

### Reglas concretas

1. **Vite como build tool.** HMR instantáneo, builds con Rollup.
   Module Federation vía `@originjs/vite-plugin-federation`.

2. **TypeScript estricto.** `strict: true`. Sin `any`.

3. **Organización por feature**, no por tipo técnico:

   ```
   dashboard/src/
   ├── api/                  # Tipos generados desde OpenAPI
   ├── auth/                 # Flujo OAuth 2.1 / PKCE
   ├── features/
   │   ├── graphs/           # Editor visual, listado
   │   ├── executions/       # Monitorización en tiempo real (AG-UI)
   │   ├── catalog/          # Gestión de paquetes
   │   ├── agents/           # Agent Cards A2A
   │   ├── mcp/              # MCP servers
   │   └── settings/         # Config, UOs, usuarios
   ├── components/
   │   ├── ui/               # shadcn/ui (propiedad)
   │   ├── composed/         # Componentes de negocio dago
   │   └── shared/           # Theme, error boundary, skeletons
   ├── hooks/
   ├── layouts/
   ├── pages/
   └── lib/                  # utils, cn()
   ```

4. **Data fetching con TanStack Query.** Nunca `useEffect` para HTTP.

5. **API client generado desde OpenAPI** (`openapi-typescript`).

6. **Comunicación real-time con AG-UI** sobre WebSocket. Cliente
   AG-UI en `dashboard/src/api/agui/`.

7. **Renderer A2UI** para componentes declarativos de los agentes.
   Valida contra catálogo de componentes aprobados.

8. **shadcn/ui + Tailwind CSS** para todo el styling. Sin CSS-in-JS,
   sin styled-components, sin CSS modules. Dark mode obligatorio.

9. **Tests con Vitest + React Testing Library + MSW.**

10. **Tokens OAuth en memoria**, nunca en localStorage (ADR-012).

## Notas para Claude Code

- El dashboard vive en `dashboard/` con su propio `package.json`.
- Componentes shadcn/ui: `npx shadcn-ui@latest add`. Viven en
  `src/components/ui/`.
- Styling: solo Tailwind utilities + CSS variables. No inline styles.
- Formularios: shadcn Form + React Hook Form + Zod.
- Data tables: shadcn DataTable + TanStack Table.
- Microfrontales de paquetes: Module Federation + lazy loading +
  Error Boundary + Suspense.
