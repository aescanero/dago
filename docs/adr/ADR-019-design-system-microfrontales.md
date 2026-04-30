# ADR-019: Design System con shadcn/ui y microfrontales con Module Federation

**Estado:** Aceptado
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

El dashboard de dago necesita una interfaz profesional, limpia y
consistente. Los paquetes (ADR-017) incluyen componentes de UI que el
dashboard carga dinámicamente. Se necesita un design system que cubra
tanto el dashboard core como los microfrontales de los paquetes, y un
mecanismo de carga dinámica que permita a los paquetes contribuir
componentes sin recompilar el dashboard.

## Decisión

Se adopta **shadcn/ui** como design system y **Module Federation**
(vía `@originjs/vite-plugin-federation`) como mecanismo de
microfrontales.

### shadcn/ui como Design System

shadcn/ui no es una librería tradicional que se instala como
dependencia — es un generador de componentes. Se copian los
componentes al proyecto y se poseen completamente. Están construidos
sobre Radix UI (accesibilidad) y Tailwind CSS (styling).

**¿Por qué shadcn/ui?**

- **Propiedad del código.** Los componentes son ficheros del
  proyecto, no una dependencia externa. Sin upgrade hell, sin
  limitaciones de personalización.
- **Accesibilidad nativa.** Radix UI como base garantiza WAI-ARIA
  compliance en todos los componentes.
- **Tailwind CSS.** Theming via CSS variables — cambiar el aspecto
  completo es editar unas variables HSL.
- **AI-friendly.** Los componentes son código React plano y legible.
  Claude Code puede leerlos, editarlos y generar nuevos componentes
  siguiendo el mismo patrón.
- **Ecosistema maduro.** 65k+ estrellas, usado por Vercel, Supabase,
  Linear. Producción probada.
- **Bundle ligero.** Solo se incluyen los componentes que se usan.
  No hay tree-shaking porque no hay paquete — solo ficheros.

### Estructura del Design System

```
dashboard/
├── src/
│   ├── components/
│   │   ├── ui/                     # Componentes shadcn/ui (propiedad)
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
│   │   ├── composed/               # Componentes compuestos (negocio)
│   │   │   ├── graph-editor.tsx     # Editor visual de grafos
│   │   │   ├── execution-timeline.tsx
│   │   │   ├── node-config-panel.tsx
│   │   │   ├── agent-chat.tsx       # Chat con AG-UI
│   │   │   └── a2ui-renderer.tsx    # Renderer de componentes A2UI
│   │   │
│   │   └── shared/                  # Componentes compartidos con microfrontales
│   │       ├── theme-provider.tsx
│   │       ├── error-boundary.tsx
│   │       └── loading-skeleton.tsx
│   │
│   ├── styles/
│   │   ├── globals.css              # CSS variables del tema
│   │   └── tokens.css               # Design tokens personalizados
│   │
│   └── lib/
│       └── utils.ts                 # cn() helper + utilidades
```

### Theming con CSS Variables

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

Los microfrontales de los paquetes heredan estas CSS variables
automáticamente. No necesitan definir su propio tema — usan el
del dashboard host.

### Module Federation para microfrontales

Los paquetes (ADR-017) pueden incluir componentes de UI que el
dashboard carga dinámicamente en runtime. Module Federation permite
que aplicaciones separadas compartan código sin recompilar.

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

**Configuración del host (dashboard):**

```typescript
// dashboard/vite.config.ts
import federation from '@originjs/vite-plugin-federation'

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'dashboard',
      remotes: {
        // Los remotes se registran dinámicamente desde el catalog
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

**Configuración del remote (paquete MFE):**

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

**Carga dinámica en el dashboard:**

```typescript
// El dashboard obtiene la URL del remoteEntry.js del catalog
const PackageComponent = React.lazy(() =>
  import(/* @vite-ignore */ `${packageRemoteUrl}/TicketTimeline`)
)

// Renderizado con error boundary y skeleton
<ErrorBoundary fallback={<ErrorCard />}>
  <Suspense fallback={<LoadingSkeleton />}>
    <PackageComponent executionId={executionId} />
  </Suspense>
</ErrorBoundary>
```

### Reglas concretas

#### Design System

1. **shadcn/ui como base.** Todos los componentes primitivos (button,
   input, card, dialog, table, etc.) son de shadcn/ui. No se mezclan
   con componentes de otras librerías (no MUI + shadcn, no Ant + shadcn).

2. **Tailwind CSS para todo el styling.** No se usa CSS-in-JS, ni
   styled-components, ni CSS modules. Tailwind con CSS variables
   para theming.

3. **Componentes compuestos en `components/composed/`.** Los
   componentes de negocio de dago (graph editor, execution timeline,
   agent chat) se construyen componiendo primitivos de shadcn/ui.

4. **Dark mode obligatorio.** Todos los componentes deben funcionar
   en light y dark mode. shadcn/ui + Tailwind lo manejan
   automáticamente con las CSS variables.

5. **Accesibilidad.** Los componentes heredan la accesibilidad de
   Radix UI. No se deben sobreescribir propiedades ARIA sin
   justificación.

6. **Responsive.** Los layouts usan las utilidades responsive de
   Tailwind (`sm:`, `md:`, `lg:`). El dashboard funciona en desktop
   y tablet.

#### Microfrontales

7. **Module Federation con `@originjs/vite-plugin-federation`.**
   Es el plugin de Vite para Module Federation. Permite carga de
   módulos remotos en runtime.

8. **React como singleton compartido.** El host y todos los remotes
   comparten la misma instancia de React y React DOM. Esto garantiza
   que hooks y context funcionen correctamente.

9. **Los microfrontales heredan el tema del host.** No definen sus
   propias CSS variables. Usan las del dashboard (definidas en
   `globals.css`). Esto garantiza consistencia visual.

10. **Error boundaries obligatorios.** Cada zona de microfrontal se
    envuelve en un Error Boundary. Si un MFE falla, el resto del
    dashboard sigue funcionando.

11. **Lazy loading.** Los microfrontales se cargan bajo demanda con
    `React.lazy()` y `Suspense`. El usuario ve un skeleton mientras
    se carga el módulo remoto.

12. **Registro dinámico de remotes.** Las URLs de los remoteEntry.js
    no se hardcodean — se obtienen del servicio catalog en runtime.
    Cuando se instala un nuevo paquete con UI, el dashboard puede
    cargarlo sin recompilarse.

13. **Sandbox de seguridad.** Los microfrontales no acceden a
    localStorage, cookies, ni a la configuración global del
    dashboard directamente. Se comunican vía props y callbacks.

## Alternativas consideradas

### Design System

- **Material UI (MUI):** Más componentes pero mayor bundle,
  CSS-in-JS en runtime, opinionado con Material Design. Descartado
  por peso y dificultad de personalización profunda.

- **Ant Design:** Ecosistema completo pero estética difícil de
  personalizar más allá del theming básico. Descartado.

- **Mantine:** Buen equilibrio pero menor adopción y ecosistema
  que shadcn/ui. Considerado como alternativa viable.

- **Sin librería (Tailwind puro):** Máxima flexibilidad pero
  requiere construir cada componente desde cero, incluyendo
  accesibilidad. Descartado por coste de desarrollo.

### Microfrontales

- **Web Components:** Agnósticos al framework pero difíciles de
  integrar con React state y context. Descartado para componentes
  complejos con interactividad.

- **iframe:** Aislamiento total pero pobre UX (no comparte tema,
  comunicación compleja, problemas de layout). Descartado.

- **Import maps / ES modules dinámicos:** Más simple que Module
  Federation pero sin shared dependencies ni singleton management.
  Descartado por complejidad de gestionar dependencias compartidas.

- **Single-SPA:** Framework de microfrontales completo pero
  opinionado y pesado para nuestro caso. Module Federation es
  más ligero y se integra nativamente con Vite.

## Consecuencias

**Positivas:**
- Interfaz profesional y consistente con shadcn/ui.
- Dark mode y accesibilidad nativos.
- Los paquetes pueden contribuir UI sin recompilar el dashboard.
- Tema compartido — los microfrontales se ven parte del dashboard.
- Claude Code puede generar y editar componentes shadcn/ui fácilmente.
- Bundle ligero — solo lo que se usa.

**Negativas:**
- shadcn/ui requiere mantenimiento manual de los componentes
  (trade-off de ownership vs auto-updates).
- Module Federation añade complejidad al build y deploy del frontend.
- Los microfrontales necesitan su propio build pipeline.
- Debugging cross-module puede ser más complejo.
- `@originjs/vite-plugin-federation` es un plugin comunitario,
  no oficial de Vite.

## Notas para Claude Code

- Componentes primitivos: usar shadcn/ui vía `npx shadcn-ui@latest add`.
  Viven en `dashboard/src/components/ui/`.
- Componentes de negocio: `dashboard/src/components/composed/`.
  Componer sobre primitivos de shadcn/ui.
- Styling: solo Tailwind CSS utilities + CSS variables. Nunca
  inline styles, styled-components ni CSS modules.
- Al crear un microfrontal para un paquete, configurar Module
  Federation para exponer los componentes y compartir React como
  singleton.
- El microfrontal NO define su propio tema — hereda CSS variables
  del host.
- Envolver cada zona de microfrontal en Error Boundary + Suspense.
- Formularios: usar shadcn/ui Form + React Hook Form + Zod para
  validación.
- Tablas de datos: usar shadcn/ui DataTable + TanStack Table.
