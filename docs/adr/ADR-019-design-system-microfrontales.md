# ADR-019: Design System with shadcn/ui and microfrontends with Module Federation

**Status:** Accepted
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

The dago dashboard needs a professional, clean, and consistent interface.
Packages (ADR-017) include UI components that the dashboard loads
dynamically. A design system is needed that covers both the core dashboard
and package microfrontends, as well as a dynamic loading mechanism that
allows packages to contribute components without recompiling the dashboard.

## Decision

**shadcn/ui** is adopted as the design system and **Module Federation**
(via `@originjs/vite-plugin-federation`) as the microfrontend mechanism.

### shadcn/ui as Design System

shadcn/ui is not a traditional library installed as a dependency — it is
a component generator. Components are copied into the project and fully
owned. They are built on top of Radix UI (accessibility) and Tailwind CSS
(styling).

**Why shadcn/ui?**

- **Code ownership.** Components are project files, not an external
  dependency. No upgrade hell, no customisation limitations.
- **Native accessibility.** Radix UI as the base guarantees WAI-ARIA
  compliance across all components.
- **Tailwind CSS.** Theming via CSS variables — changing the entire
  appearance means editing a few HSL variables.
- **AI-friendly.** Components are plain, readable React code.
  Claude Code can read, edit, and generate new components
  following the same pattern.
- **Mature ecosystem.** 65k+ stars, used by Vercel, Supabase,
  Linear. Production-proven.
- **Lightweight bundle.** Only the components that are used are
  included. No tree-shaking needed because there is no package — only files.

### Design System structure

```
dashboard/
├── src/
│   ├── components/
│   │   ├── ui/                     # shadcn/ui components (owned)
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── form.tsx
│   │   │   ├── input.tsx
│   │   │   ├── table.tsx
│   │   │   ├── tabs.tsx
│   │   │   ├── toast.tsx
│   │   │   └── ...
│   │   │
│   │   ├── composed/               # Composed components (business)
│   │   │   ├── graph-editor.tsx     # Visual graph editor
│   │   │   ├── execution-timeline.tsx
│   │   │   ├── node-config-panel.tsx
│   │   │   ├── agent-chat.tsx       # Chat with AG-UI
│   │   │   └── a2ui-renderer.tsx    # A2UI component renderer
│   │   │
│   │   └── shared/                  # Components shared with microfrontends
│   │       ├── theme-provider.tsx
│   │       ├── error-boundary.tsx
│   │       └── loading-skeleton.tsx
│   │
│   ├── styles/
│   │   ├── globals.css              # Theme CSS variables
│   │   └── tokens.css               # Custom design tokens
│   │
│   └── lib/
│       └── utils.ts                 # cn() helper + utilities
```

### Theming with CSS Variables

```css
/* styles/globals.css */
:root {
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  --primary: 221.2 83.2% 53.3%;
  --primary-foreground: 210 40% 98%;
  --secondary: 210 40% 96.1%;
  --muted: 210 40% 96.1%;
  --accent: 210 40% 96.1%;
  --destructive: 0 84.2% 60.2%;
  --border: 214.3 31.8% 91.4%;
  --radius: 0.5rem;
  /* ... */
}

.dark {
  --background: 222.2 84% 4.9%;
  --foreground: 210 40% 98%;
  /* ... */
}
```

Package microfrontends inherit these CSS variables automatically.
They do not need to define their own theme — they use the host
dashboard's theme.

### Module Federation for microfrontends

Packages (ADR-017) can include UI components that the dashboard
loads dynamically at runtime. Module Federation allows separate
applications to share code without recompiling.

```
┌─────────────────────────────────────────────────┐
│                 Dashboard (Host)                 │
│                                                  │
│  ┌──────────────┐  ┌──────────────────────────┐ │
│  │ Core UI      │  │ Microfrontend Zone       │ │
│  │ (shadcn/ui)  │  │                          │ │
│  │              │  │  ┌────────────────────┐  │ │
│  │ • Navigation │  │  │ Package MFE        │  │ │
│  │ • Layout     │  │  │ (loaded at runtime)│  │ │
│  │ • Auth       │  │  │                    │  │ │
│  │ • Settings   │  │  │ Uses shared:       │  │ │
│  │              │  │  │ • React            │  │ │
│  │              │  │  │ • shadcn/ui theme  │  │ │
│  │              │  │  │ • CSS variables    │  │ │
│  │              │  │  └────────────────────┘  │ │
│  └──────────────┘  └──────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

**Host configuration (dashboard):**

```typescript
// dashboard/vite.config.ts
import federation from '@originjs/vite-plugin-federation'

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'dashboard',
      remotes: {
        // Remotes are registered dynamically from the catalog
      },
      shared: {
        'react': { singleton: true },
        'react-dom': { singleton: true },
        'react-router-dom': { singleton: true }
      }
    })
  ]
})
```

**Remote configuration (package MFE):**

```typescript
// packages/customer-support-ui/vite.config.ts
import federation from '@originjs/vite-plugin-federation'

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'customer_support',
      filename: 'remoteEntry.js',
      exposes: {
        './TicketTimeline': './src/components/TicketTimeline.tsx',
        './SatisfactionDashboard': './src/components/SatisfactionDashboard.tsx'
      },
      shared: {
        'react': { singleton: true },
        'react-dom': { singleton: true }
      }
    })
  ]
})
```

**Dynamic loading in the dashboard:**

```typescript
// The dashboard gets the remoteEntry.js URL from the catalog
const PackageComponent = React.lazy(() =>
  import(/* @vite-ignore */ `${packageRemoteUrl}/TicketTimeline`)
)

// Rendered with error boundary and skeleton
<ErrorBoundary fallback={<ErrorCard />}>
  <Suspense fallback={<LoadingSkeleton />}>
    <PackageComponent executionId={executionId} />
  </Suspense>
</ErrorBoundary>
```

### Concrete rules

#### Design System

1. **shadcn/ui as the base.** All primitive components (button,
   input, card, dialog, table, etc.) come from shadcn/ui. They are
   not mixed with components from other libraries (no MUI + shadcn,
   no Ant + shadcn).

2. **Tailwind CSS for all styling.** No CSS-in-JS, no
   styled-components, no CSS modules. Tailwind with CSS variables
   for theming.

3. **Composed components in `components/composed/`.** Dago business
   components (graph editor, execution timeline, agent chat) are
   built by composing shadcn/ui primitives.

4. **Dark mode mandatory.** All components must work in light and dark
   mode. shadcn/ui + Tailwind handle this automatically with CSS variables.

5. **Accessibility.** Components inherit accessibility from Radix UI.
   ARIA properties must not be overridden without justification.

6. **Responsive.** Layouts use Tailwind responsive utilities
   (`sm:`, `md:`, `lg:`). The dashboard works on desktop and tablet.

#### Microfrontends

7. **Module Federation with `@originjs/vite-plugin-federation`.**
   This is the Vite plugin for Module Federation. It allows loading
   remote modules at runtime.

8. **React as a shared singleton.** The host and all remotes share
   the same React and React DOM instance. This ensures hooks and
   context work correctly.

9. **Microfrontends inherit the host theme.** They do not define their
   own CSS variables. They use the dashboard's variables (defined in
   `globals.css`). This guarantees visual consistency.

10. **Mandatory error boundaries.** Each microfrontend zone is wrapped
    in an Error Boundary. If an MFE fails, the rest of the dashboard
    keeps working.

11. **Lazy loading.** Microfrontends are loaded on demand with
    `React.lazy()` and `Suspense`. The user sees a skeleton while
    the remote module loads.

12. **Dynamic remote registration.** remoteEntry.js URLs are not
    hardcoded — they are fetched from the catalog service at runtime.
    When a new package with UI is installed, the dashboard can
    load it without recompiling.

13. **Security sandbox.** Microfrontends do not access localStorage,
    cookies, or the dashboard's global configuration directly. They
    communicate via props and callbacks.

## Considered Alternatives

### Design System

- **Material UI (MUI):** More components but larger bundle,
  runtime CSS-in-JS, opinionated with Material Design. Discarded
  due to size and difficulty of deep customisation.

- **Ant Design:** Complete ecosystem but aesthetic that is hard to
  customise beyond basic theming. Discarded.

- **Mantine:** Good balance but lower adoption and ecosystem
  than shadcn/ui. Considered as a viable alternative.

- **No library (pure Tailwind):** Maximum flexibility but
  requires building every component from scratch, including
  accessibility. Discarded due to development cost.

### Microfrontends

- **Web Components:** Framework-agnostic but difficult to
  integrate with React state and context. Discarded for complex
  interactive components.

- **iframe:** Total isolation but poor UX (no shared theme,
  complex communication, layout issues). Discarded.

- **Import maps / dynamic ES modules:** Simpler than Module
  Federation but without shared dependencies or singleton management.
  Discarded due to complexity of managing shared dependencies.

- **Single-SPA:** Full microfrontend framework but opinionated
  and heavy for our use case. Module Federation is lighter and
  integrates natively with Vite.

## Consequences

**Positive:**
- Professional and consistent interface with shadcn/ui.
- Native dark mode and accessibility.
- Packages can contribute UI without recompiling the dashboard.
- Shared theme — microfrontends look like part of the dashboard.
- Claude Code can generate and edit shadcn/ui components easily.
- Lightweight bundle — only what is used.

**Negative:**
- shadcn/ui requires manual maintenance of components
  (trade-off of ownership vs auto-updates).
- Module Federation adds complexity to the frontend build and deploy.
- Microfrontends need their own build pipeline.
- Cross-module debugging can be more complex.
- `@originjs/vite-plugin-federation` is a community plugin,
  not official from Vite.

## Notes for Claude Code

- Primitive components: use shadcn/ui via `npx shadcn-ui@latest add`.
  They live in `dashboard/src/components/ui/`.
- Business components: `dashboard/src/components/composed/`.
  Compose over shadcn/ui primitives.
- Styling: Tailwind CSS utilities + CSS variables only. Never
  inline styles, styled-components, or CSS modules.
- When creating a microfrontend for a package, configure Module
  Federation to expose the components and share React as a singleton.
- The microfrontend does NOT define its own theme — it inherits CSS
  variables from the host.
- Wrap each microfrontend zone in Error Boundary + Suspense.
- Forms: use shadcn/ui Form + React Hook Form + Zod for validation.
- Data tables: use shadcn/ui DataTable + TanStack Table.
