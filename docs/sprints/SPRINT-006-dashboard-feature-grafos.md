# SPRINT-006: Feature grafos — listado, detalle y creación en el dashboard

## Metadata

- **Fecha inicio:** 2026-05-02 (tras completar SPRINT-005)
- **Fecha fin estimada:** 2026-05-04
- **Estado:** planificado
- **ADRs aplicados:** ADR-009, ADR-010, ADR-016, ADR-019
- **Specs afectadas:** ninguna (usa endpoints ya definidos en SPRINT-003)
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Depende de:** SPRINT-005 (dashboard base, PKCE, AppLayout, GraphsPage básica)
- **Bloquea:** SPRINT-007 (execution monitor), SPRINT-008 (graph visual editor)

## Objetivo del sprint

Completar la feature de grafos en el dashboard: mejorar el listado con
filtrado y acciones, implementar la página de detalle con pestañas y la
página de creación con formulario validado, conectando con los cinco
endpoints CRUD del orchestrator (SPRINT-003).

Al finalizar: un usuario autenticado puede listar, filtrar, ver el detalle,
crear, editar y archivar grafos desde el dashboard, con feedback visual en
cada operación.

## Alcance

### Incluido

**Reorganización en feature module:**
- Mover `pages/GraphsPage.tsx` a `features/graphs/pages/GraphsPage.tsx`.
- Crear `features/graphs/` completo (componentes, hooks, páginas, tipos, schemas).

**Nuevos componentes shadcn/ui:**
- `form`, `textarea`, `alert-dialog`, `breadcrumb`, `alert`,
  `tabs`, `switch`, `select`, `tooltip`, `scroll-area`.

**Schemas y utilidades:**
- `features/graphs/schemas/graph-form.schema.ts` — Zod schema del formulario.
- `features/graphs/lib/definition-templates.ts` — 4 plantillas JSON de definición
  (vacía, llm_call, react, router+handlers) derivadas de ADR-016.

**Hooks TanStack Query:**
- `useGraph(id)` — `GET /api/v1/graphs/:id`.
- `useCreateGraph()` — `POST /api/v1/graphs`.
- `useUpdateGraph()` — `PUT /api/v1/graphs/:id`.
- `useArchiveGraph()` — `DELETE /api/v1/graphs/:id`.
- `useGraphs(opts)` refactorizado — añade parámetro `status` para filtrado.

**Componentes de la feature:**
- `GraphStatusBadge` — badge con color por status (draft/active/archived).
- `GraphTable` — tabla extraída de la GraphsPage de SPRINT-005, con columnas:
  nombre, versión, status, fecha creación + acción "Ver detalle".
- `GraphDefinitionViewer` — vista JSON formateada (nodes + edges) con
  secciones plegables por nodo.
- `NodePatternBadge` — badge por patrón de nodo (llm_call, react, etc.).
- `EdgeTypeBadge` — badge por tipo de arista (sequential, conditional, etc.).
- `GraphForm` — formulario compartido creación/edición con React Hook Form +
  Zod + selector de plantilla de definición.
- `GraphArchiveDialog` — AlertDialog de confirmación de archivo.

**Páginas (bajo `features/graphs/pages/`):**
- `GraphsPage` — listado con filtro de status, paginación, botón "Nuevo grafo".
- `GraphDetailPage` — detalle con 3 pestañas: metadatos, definición, ejecuciones.
- `GraphCreatePage` — página `/graphs/new` con `GraphForm`.
- `GraphEditPage` — página `/graphs/:id/edit` con `GraphForm` pre-rellenado.

**Routing (`App.tsx`):**
```
/graphs               → GraphsPage
/graphs/new           → GraphCreatePage
/graphs/:id           → GraphDetailPage
/graphs/:id/edit      → GraphEditPage
```

**Breadcrumbs** en todas las páginas de detalle/formulario usando el
componente `breadcrumb` de shadcn/ui.

**Tests:**
- `features/graphs/components/__tests__/GraphForm.test.tsx`
- `features/graphs/pages/__tests__/GraphDetailPage.test.tsx`
- `features/graphs/pages/__tests__/GraphCreatePage.test.tsx`
- `features/graphs/hooks/__tests__/useArchiveGraph.test.ts`

### Excluido

- Editor visual de nodos/aristas (canvas drag-and-drop) — SPRINT-008.
- Pestaña "Ejecuciones" en el detalle con datos reales (SPRINT-007).
- Activación de grafo (`status draft → active`) — necesita endpoint
  `PATCH /graphs/:id/status` (no implementado en SPRINT-003).
- Duplicar/importar/exportar grafo — sprints futuros.
- Validación semántica del JSON de definición contra JSON Schema de ADR-016
  en el cliente (solo validación de JSON bien formado en este sprint).
- Paginación infinita o virtual scroll — paginación básica es suficiente.
- Búsqueda full-text de grafos — el filtro de este sprint es solo por status.

## Dependencias

- **SPRINT-005 completado:** `AppLayout`, `AuthProvider`, `useAuth`,
  routing base, `GraphsPage` básica (lista), componentes shadcn/ui
  base (button, card, table, skeleton, badge, toast).
- **SPRINT-003 completado:** endpoints CRUD de grafos funcionales en
  el orchestrator.
- **`openapi-typescript` generado (`make gen-api-types`):** tipos
  `GraphResponse`, `GraphInput`, `GraphListResponse` disponibles.

## Contratos de comportamiento

### C1 — `GraphForm` submit con datos válidos

```
Given: Formulario con name="mi-grafo", version="1.0.0", entry_node="llm", definition=JSON válido
When: El usuario hace clic en el botón de submit
Then: onSubmit es llamado con los valores transformados por formToGraphInput
      Los campos coinciden con la estructura GraphInput de la API
      El botón de submit se deshabilita mientras isPending=true (no doble envío)
```

### C2 — `useArchiveGraph` — DELETE exitoso

```
Given: Grafo con id="<uuid>" en status="draft", sin ejecuciones activas
When: useArchiveGraph().mutate("<uuid>")
Then: Se llama DELETE /api/v1/graphs/<uuid>
      En respuesta 204: la query ["graphs"] es invalidada
      Se muestra toast de éxito
      El grafo desaparece de la lista
```

### C3 — Edición bloqueada para grafos no-draft

```
Given: Usuario navega a GraphEditPage de un grafo con status="active"
When: La página carga
Then: Se muestra un componente Alert en lugar del formulario de edición
      El formulario no es accesible (no existe en el DOM)
      No se puede modificar ni enviar el grafo
```

## Diseño

### Estructura de directorios

```
dashboard/src/features/graphs/
├── components/
│   ├── GraphStatusBadge.tsx
│   ├── GraphTable.tsx
│   ├── GraphDefinitionViewer.tsx
│   ├── NodePatternBadge.tsx
│   ├── EdgeTypeBadge.tsx
│   ├── GraphForm.tsx
│   ├── GraphArchiveDialog.tsx
│   └── __tests__/
│       └── GraphForm.test.tsx
├── hooks/
│   ├── useGraphs.ts
│   ├── useGraph.ts
│   ├── useCreateGraph.ts
│   ├── useUpdateGraph.ts
│   ├── useArchiveGraph.ts
│   └── __tests__/
│       └── useArchiveGraph.test.ts
├── lib/
│   └── definition-templates.ts
├── pages/
│   ├── GraphsPage.tsx
│   ├── GraphDetailPage.tsx
│   ├── GraphCreatePage.tsx
│   ├── GraphEditPage.tsx
│   └── __tests__/
│       ├── GraphDetailPage.test.tsx
│       └── GraphCreatePage.test.tsx
├── schemas/
│   └── graph-form.schema.ts
└── index.ts
```

### Zod schema del formulario

```typescript
// features/graphs/schemas/graph-form.schema.ts
import { z } from "zod";

export const graphFormSchema = z.object({
  name:        z.string().min(1, "Nombre requerido").max(255),
  version:     z.string().regex(/^\d+\.\d+\.\d+$/, "Formato: 1.0.0"),
  description: z.string().max(1000).optional(),
  entry_node:  z.string().min(1, "Nodo de entrada requerido").max(255),
  definition:  z
    .string()
    .min(2, "Definición requerida")
    .refine((v) => { try { JSON.parse(v); return true; } catch { return false; } },
      { message: "JSON inválido" }),
  memory_semantic_search:   z.boolean().default(false),
  memory_episode_context:   z.coerce.number().int().min(0).default(0),
});

export type GraphFormValues = z.infer<typeof graphFormSchema>;

// Convierte los valores del formulario al GraphInput de la API.
export function formToGraphInput(values: GraphFormValues): GraphInput {
  return {
    name:        values.name,
    version:     values.version,
    description: values.description,
    entry_node:  values.entry_node,
    definition:  JSON.parse(values.definition),
    memory_config: {
      semantic_search: values.memory_semantic_search,
      episode_context: values.memory_episode_context,
    },
  };
}
```

### Plantillas de definición (ADR-016)

```typescript
// features/graphs/lib/definition-templates.ts
export const DEFINITION_TEMPLATES = [
  {
    id: "empty",
    label: "Grafo vacío",
    definition: { nodes: {}, edges: [] },
  },
  {
    id: "llm_call",
    label: "LLM Call simple",
    definition: {
      nodes: {
        llm: {
          pattern: "llm_call",
          config: {
            model: "claude-sonnet-4-6",
            system_prompt: "You are a helpful assistant.",
            temperature: 0.7,
            max_tokens: 2048,
          },
        },
      },
      edges: [],
    },
  },
  {
    id: "react_agent",
    label: "React agent",
    definition: {
      nodes: {
        agent: {
          pattern: "react",
          config: {
            model: "claude-sonnet-4-6",
            system_prompt: "You are an agent that reasons step by step.",
            tools: [],
            max_iterations: 10,
            timeout_seconds: 120,
          },
        },
      },
      edges: [],
    },
  },
  {
    id: "router_handlers",
    label: "Router + handlers",
    definition: {
      nodes: {
        router: {
          pattern: "router",
          config: { mode: "llm", llm_fallback: { model: "claude-sonnet-4-6", routes: ["handler_a", "handler_b"] } },
        },
        handler_a: { pattern: "llm_call", config: { model: "claude-sonnet-4-6" } },
        handler_b: { pattern: "llm_call", config: { model: "claude-sonnet-4-6" } },
      },
      edges: [
        { type: "conditional", from: "router",
          conditions: [
            { expression: "output.route == 'handler_a'", target: "handler_a" },
            { expression: "output.route == 'handler_b'", target: "handler_b" },
          ]},
      ],
    },
  },
] as const;
```

### GraphDefinitionViewer — estructura visual

El visor muestra la definición en dos secciones plegables (usando `<details>`
con Tailwind o el componente `Collapsible` de Radix):

```
▶ Nodos (3)
  ┌─ [llm_call] llm
  │   model: claude-sonnet-4-6
  │   system_prompt: You are a helpful...
  ├─ [react] agent
  └─ [router] router

▶ Aristas (2)
  router → handler_a  [conditional]
  router → handler_b  [conditional]
```

Si la definición no se puede parsear como JSON, muestra el texto raw
en una `<pre>` con aviso de alerta.

### GraphDetailPage — pestañas

| Pestaña | Contenido |
|---------|-----------|
| Metadatos | Card con todos los campos del grafo + acciones |
| Definición | `GraphDefinitionViewer` con el JSON del grafo |
| Ejecuciones | EmptyState: "Las ejecuciones aparecerán aquí" + botón "Iniciar ejecución" (deshabilitado hasta SPRINT-007) |

Acciones en la pestaña Metadatos (botones en la cabecera de la página):
- Si `status=draft`: botón "Editar" → `/graphs/:id/edit` + botón "Archivar" (dialog).
- Si `status=active`: botón "Archivar".
- Si `status=archived`: solo lectura, badge de advertencia.

### Flujo de creación

```
/graphs/new
  │
  ├── Seleccionar plantilla (dropdown con 4 opciones)
  │   → rellena el textarea de definición
  │
  ├── Rellenar: nombre, versión, descripción, nodo entrada
  │
  ├── Editar JSON de definición en textarea monospace
  │   → validación onBlur: ¿es JSON válido?
  │
  ├── Toggle: Búsqueda semántica / Contexto episódico
  │
  └── Submit
        ├── OK (201)  → toast "Grafo creado" → navegar a /graphs/:id
        ├── 409       → toast error "Ya existe la versión X.Y.Z de este grafo"
        └── 422       → mostrar errores de campo inline
```

### Flujo de archivado

```
Botón "Archivar" → AlertDialog
  "¿Archivar este grafo? No se podrán iniciar nuevas ejecuciones."
  [Cancelar] [Archivar]
        │
        └── DELETE /api/v1/graphs/:id
              ├── 204 → toast "Grafo archivado" → navegar a /graphs
              └── 409 → toast error "El grafo tiene ejecuciones activas"
```

## TODOs

### 1. [infra] Añadir nuevos componentes shadcn/ui

- **Agente:** @developer
- **Descripción:** Añadir los componentes shadcn/ui necesarios para
  este sprint que no se incluyeron en SPRINT-005.

  Componentes a añadir (con `npx shadcn@latest add` o equivalente):
  `form`, `textarea`, `alert-dialog`, `breadcrumb`, `alert`,
  `tabs`, `switch`, `select`, `tooltip`, `scroll-area`, `collapsible`.

  Verificar que el `ThemeProvider` y el dark mode siguen funcionando
  correctamente después de añadir los nuevos componentes.

- **Criterio de aceptación:** Todos los componentes existen en
  `dashboard/src/components/ui/`. `npm run build` compila sin errores.
- **Depende de:** ninguno
- **Commit:** `feat(dashboard): add shadcn/ui components for graphs feature [SPRINT-006 #1]`

### 2. [domain] Zod schema del formulario y plantillas de definición

- **Agente:** @developer
- **Descripción:** Crear los archivos de schema y plantillas según
  el diseño de este documento.

  **`features/graphs/schemas/graph-form.schema.ts`:** Zod schema con
  los 7 campos, mensajes de error en español, y la función
  `formToGraphInput()`. Importar `GraphInput` del tipo generado
  por `openapi-typescript`.

  **`features/graphs/lib/definition-templates.ts`:** Las 4 plantillas
  tipadas como `const` con los patrones de ADR-016. Incluir una función
  `templateToDefinitionString(id: string): string` que devuelve el
  JSON serializado con `JSON.stringify(..., null, 2)`.

- **Criterio de aceptación:** `npm run type-check` pasa. Los tipos
  de los templates coinciden con los schemas de
  `specs/patterns/nodes/*.json`.
- **Depende de:** SPRINT-005 completado (tipos generados disponibles)
- **Commit:** `feat(graphs): add graph form Zod schema and definition templates [SPRINT-006 #2]`

### 3. [domain] Hooks de TanStack Query

- **Agente:** @developer
- **Descripción:** Crear los cuatro hooks de mutación y el hook de
  detalle.

  **`useGraph(id: string)`:**
  ```typescript
  export function useGraph(id: string) {
    const { apiClient } = useAuth();
    return useQuery({
      queryKey: ["graphs", id],
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/api/v1/graphs/{id}", {
          params: { path: { id } },
        });
        if (error) throw error;
        return data;
      },
      enabled: !!id,
    });
  }
  ```

  **`useCreateGraph()`:**
  ```typescript
  export function useCreateGraph() {
    const { apiClient } = useAuth();
    const queryClient = useQueryClient();
    return useMutation({
      mutationFn: async (input: GraphInput) => {
        const { data, error } = await apiClient.POST("/api/v1/graphs", { body: input });
        if (error) throw error;
        return data;
      },
      onSuccess: () => queryClient.invalidateQueries({ queryKey: ["graphs"] }),
    });
  }
  ```

  Análogos para `useUpdateGraph(id)` y `useArchiveGraph()`.

  `useGraphs()` refactorizado para aceptar `status?: string` como
  filtro adicional.

- **Criterio de aceptación:** `npm run type-check` pasa. Los tipos
  de los parámetros y retornos coinciden con los tipos generados.
- **Depende de:** #2 (necesita los tipos del schema)
- **Commit:** `feat(graphs): add TanStack Query hooks for CRUD operations [SPRINT-006 #3]`

### 4. [test] Tests de GraphForm (Red)

- **Agente:** @qa
- **Descripción:** Tests del componente de formulario antes de
  implementarlo.

  ```typescript
  // features/graphs/components/__tests__/GraphForm.test.tsx

  // testRendersAllFields: el formulario muestra campos name, version,
  // description, entry_node, definition textarea y toggles de memory.
  test("GraphForm renders all required fields", ...)

  // testVersionValidation: version "abc" muestra error "Formato: 1.0.0".
  test("GraphForm shows version format error for non-semver", ...)

  // testDefinitionInvalidJSON: texto no-JSON en definition muestra
  // el error "JSON inválido" tras perder el foco.
  test("GraphForm shows JSON error for invalid definition", ...)

  // testTemplateSelector: al seleccionar una plantilla, el textarea
  // de definición se rellena con el JSON correspondiente.
  test("GraphForm fills definition textarea on template select", ...)

  // testSubmitCallsOnSubmit: con datos válidos, el submit llama la
  // función onSubmit con los valores transformados por formToGraphInput.
  test("GraphForm calls onSubmit with transformed values on valid submit", ...)

  // testSubmitDisabledWhilePending: el botón submit está deshabilitado
  // mientras isPending=true.
  test("GraphForm disables submit while pending", ...)
  ```

- **Criterio de aceptación:** Tests en RED antes del TODO #8.
  GREEN tras implementar `GraphForm`.
- **Depende de:** #2, #1 (componentes shadcn/ui necesarios)
- **Commit:** `test(graphs): add GraphForm component tests [SPRINT-006 #4]`

### 5. [test] Tests de GraphDetailPage con MSW (Red)

- **Agente:** @qa
- **Descripción:** Tests de la página de detalle simulando la API.

  ```typescript
  // features/graphs/pages/__tests__/GraphDetailPage.test.tsx

  // testShowsGraphMetadata: muestra nombre, versión y status badge.
  test("GraphDetailPage renders graph metadata", ...)

  // testTabNavigationMetadata: por defecto se muestra la pestaña
  // "Metadatos" activa.
  test("GraphDetailPage shows Metadatos tab by default", ...)

  // testTabDefinition: al hacer clic en "Definición", se muestra
  // el GraphDefinitionViewer con los nodos del grafo mock.
  test("GraphDetailPage shows definition viewer on Definición tab", ...)

  // testEditButtonForDraft: para un grafo draft se muestra el botón "Editar".
  test("GraphDetailPage shows Edit button for draft graph", ...)

  // testNoEditButtonForArchived: para un grafo archivado no aparece
  // el botón "Editar".
  test("GraphDetailPage hides Edit button for archived graph", ...)

  // testArchiveDialogAppears: al hacer clic en "Archivar" aparece
  // el AlertDialog de confirmación.
  test("GraphDetailPage shows archive confirmation dialog", ...)

  // testShowsSkeletonWhileLoading: muestra skeleton mientras carga.
  test("GraphDetailPage shows skeleton while loading", ...)

  // testShowsNotFoundOn404: si la API devuelve 404, muestra un
  // mensaje apropiado.
  test("GraphDetailPage shows not found state on 404", ...)
  ```

- **Criterio de aceptación:** Tests en RED antes del TODO #11.
  GREEN tras implementar `GraphDetailPage`.
- **Depende de:** #3, #1
- **Commit:** `test(graphs): add GraphDetailPage tests with MSW [SPRINT-006 #5]`

### 6. [test] Tests de GraphCreatePage con MSW (Red)

- **Agente:** @qa
- **Descripción:** Tests del flujo de creación completo.

  ```typescript
  // features/graphs/pages/__tests__/GraphCreatePage.test.tsx

  // testRendersCreateForm: la página muestra el formulario de creación
  // con todos los campos.
  test("GraphCreatePage renders the create form", ...)

  // testSuccessfulCreation: al completar el formulario con datos válidos
  // y el MSW respondiendo 201, se muestra un toast de éxito y se
  // navega a la página de detalle.
  test("GraphCreatePage shows success toast and navigates on 201", ...)

  // testConflictError: si el MSW responde 409, se muestra un toast
  // de error con el mensaje del código GRAPH_DUPLICATE_VERSION.
  test("GraphCreatePage shows conflict toast on 409", ...)

  // testValidationErrors: si el MSW responde 422 con details de campos,
  // los errores se muestran inline bajo cada campo.
  test("GraphCreatePage shows inline errors on 422", ...)

  // testBreadcrumbNavigation: los breadcrumbs muestran
  // "Grafos > Nuevo grafo" con enlace funcional a /graphs.
  test("GraphCreatePage renders correct breadcrumbs", ...)
  ```

- **Criterio de aceptación:** Tests en RED antes del TODO #12.
  GREEN tras implementar `GraphCreatePage`.
- **Depende de:** #3, #4
- **Commit:** `test(graphs): add GraphCreatePage tests with MSW [SPRINT-006 #6]`

### 7. [test] Tests de useArchiveGraph con MSW (Red)

- **Agente:** @qa
- **Descripción:** Tests del hook de mutación de archivado.

  ```typescript
  // features/graphs/hooks/__tests__/useArchiveGraph.test.ts

  // testArchiveSuccessInvalidatesCache: al archivar con éxito (204),
  // la query ["graphs"] se invalida.
  test("useArchiveGraph invalidates graphs cache on success", ...)

  // testArchiveConflictThrows: si el servidor responde 409, la
  // mutación rechaza con el error de la API.
  test("useArchiveGraph throws on 409 conflict", ...)
  ```

- **Criterio de aceptación:** Tests en RED antes del TODO #11.
  GREEN tras implementar el hook + la página de detalle.
- **Depende de:** #3
- **Commit:** `test(graphs): add useArchiveGraph hook tests [SPRINT-006 #7]`

### 8. [impl] GraphStatusBadge, NodePatternBadge, EdgeTypeBadge, GraphTable

- **Agente:** @developer
- **Descripción:** Componentes primitivos de la feature.

  **`GraphStatusBadge`:** Badge de shadcn/ui con variante por status:
  - `draft` → `secondary` (gris)
  - `active` → `default` (azul)
  - `archived` → `outline` (borde, texto atenuado)

  **`NodePatternBadge`:** Badge con variante por patrón. Los 7 patrones
  de ADR-016 con colores semánticamente diferenciados:
  - `llm_call` → azul
  - `react` → verde
  - `reflection` → púrpura
  - `tool_use` → naranja
  - `router` → amarillo
  - `guardrail` → rojo
  - `subgraph` → gris oscuro

  **`EdgeTypeBadge`:** Badge para los 5 tipos de arista de ADR-016.

  **`GraphTable`:** Tabla extraída de la `GraphsPage` de SPRINT-005.
  Columnas: nombre (con link al detalle), versión, status badge, fecha
  creación (formateada). Sin acciones inline (las acciones están en
  el detalle).

- **Criterio de aceptación:** Los tres badges se renderizan con los
  colores correctos. `npm run type-check` sin errores.
- **Depende de:** #1
- **Commit:** `feat(graphs): add GraphStatusBadge, NodePatternBadge, EdgeTypeBadge, GraphTable [SPRINT-006 #8]`

### 9. [impl] GraphDefinitionViewer

- **Agente:** @developer
- **Descripción:** Componente que muestra la definición de un grafo
  de forma legible, sin editor ni drag-and-drop.

  Props: `definition: unknown` (el JSON del grafo parseado).

  Lógica:
  1. Validar que `definition` es un objeto con `nodes` y `edges`.
  2. Si inválido: mostrar `<Alert variant="destructive">` con el JSON
     raw en `<pre>`.
  3. Si válido: mostrar dos secciones `<Collapsible>`:

     **Nodos** (`definition.nodes` es un mapa `{[key]: NodeDefinition}`):
     - Una fila por nodo: `NodePatternBadge` + key del nodo +
       resumen de config (model si existe, max_iterations si existe).
     - Expandible para ver el JSON completo del nodo.

     **Aristas** (`definition.edges` es un array):
     - Una fila por arista: `from` → `to` + `EdgeTypeBadge`.
     - Si `type=conditional`: mostrar número de condiciones.

  El componente usa `<ScrollArea>` si el contenido es muy largo.

- **Criterio de aceptación:** El componente renderiza correctamente
  los 4 templates de definición de `definition-templates.ts`.
  `npm run type-check` pasa.
- **Depende de:** #8, #1
- **Commit:** `feat(graphs): add GraphDefinitionViewer component [SPRINT-006 #9]`

### 10. [impl] GraphForm — formulario compartido creación/edición

- **Agente:** @developer
- **Descripción:** Componente de formulario con React Hook Form + Zod.
  Usado tanto en `GraphCreatePage` como en `GraphEditPage`.

  Props:
  ```typescript
  interface GraphFormProps {
    defaultValues?: Partial<GraphFormValues>;
    onSubmit: (values: GraphFormValues) => Promise<void>;
    isPending: boolean;
    submitLabel: string;
  }
  ```

  Campos del formulario (orden visual):
  1. `name` — Input con label "Nombre".
  2. `version` — Input con label "Versión" y placeholder "1.0.0".
  3. `description` — Textarea con label "Descripción (opcional)".
  4. `entry_node` — Input con label "Nodo de entrada" y tooltip
     explicativo: "Clave del nodo por el que comienza la ejecución".
  5. Selector de plantilla — `Select` con las 4 opciones de
     `DEFINITION_TEMPLATES`. Al cambiar: rellena el textarea.
  6. `definition` — Textarea con fuente monospace (`font-mono` Tailwind),
     mínimo 12 filas, con validación JSON en `onBlur`.
  7. Sección "Memoria" (colapsable con `Collapsible`):
     - `memory_semantic_search` — Switch con label.
     - `memory_episode_context` — Input numérico.
  8. Botón submit con `submitLabel` y spinner si `isPending`.

  Errores: bajo cada campo usando `<FormMessage>` de shadcn/ui.

- **Criterio de aceptación:** Tests del TODO #4 pasan a GREEN.
  `npm run type-check` pasa. Ningún componente usa `any`.
- **Depende de:** #4, #1, #2
- **Commit:** `feat(graphs): implement GraphForm with React Hook Form and Zod [SPRINT-006 #10]`

### 11. [impl] GraphDetailPage + GraphArchiveDialog

- **Agente:** @developer
- **Descripción:** Página de detalle con las 3 pestañas y el dialog
  de archivado.

  **`GraphDetailPage`:**
  - `useGraph(id)` con `useParams()`.
  - Skeleton mientras carga (mismas dimensiones que el contenido).
  - Breadcrumbs: "Grafos" → nombre del grafo.
  - Cabecera: nombre, versión, `GraphStatusBadge`, acciones
    condicionales por status.
  - Pestañas con `Tabs` de shadcn/ui:
    - **Metadatos:** `Card` con grid de campos (descripción, entry_node,
      fechas). `memory_config` mostrado como checkboxes read-only.
    - **Definición:** `GraphDefinitionViewer` con `definition` del grafo.
    - **Ejecuciones:** `EmptyState` con icono, texto "Sin ejecuciones"
      y botón "Iniciar ejecución" deshabilitado con tooltip
      "Disponible en la próxima versión".

  **`GraphArchiveDialog`:**
  - `AlertDialog` de shadcn/ui.
  - Mensaje de confirmación.
  - Llama `useArchiveGraph()`, muestra toast de resultado.
  - En 409: toast con "El grafo tiene ejecuciones activas".

- **Criterio de aceptación:** Tests del TODO #5 y #7 pasan a GREEN.
  `npm run type-check` pasa.
- **Depende de:** #5, #7, #9, #3
- **Commit:** `feat(graphs): implement GraphDetailPage with tabs and archive dialog [SPRINT-006 #11]`

### 12. [impl] GraphCreatePage, GraphEditPage y GraphsPage mejorada

- **Agente:** @developer
- **Descripción:** Las tres páginas restantes.

  **`GraphCreatePage`** (`/graphs/new`):
  - Breadcrumbs: "Grafos" → "Nuevo grafo".
  - Título "Crear grafo".
  - `GraphForm` con `submitLabel="Crear"` y `useCreateGraph()`.
  - En éxito (201): `toast.success("Grafo creado")` + `navigate(/graphs/:id)`.
  - En 409: `toast.error("Ya existe versión X.Y.Z de ...")`.
  - En 422: los errores inline llegan via la respuesta y se mapean
    a los campos del formulario.

  **`GraphEditPage`** (`/graphs/:id/edit`):
  - Breadcrumbs: "Grafos" → nombre del grafo → "Editar".
  - Carga el grafo con `useGraph(id)`.
  - Si `status !== "draft"`: muestra `Alert` "Solo se pueden editar
    grafos en borrador" y botón "Volver".
  - `GraphForm` pre-rellenado con los datos actuales.
  - Usa `useUpdateGraph()`.
  - En éxito: `toast.success("Grafo actualizado")` + `navigate(/graphs/:id)`.

  **`GraphsPage` mejorada:**
  - Filtro de status: `Select` con opciones "Todos", "Borrador",
    "Activo", "Archivado". Guarda la selección en `searchParams`.
  - Botón "Nuevo grafo" en cabecera → navega a `/graphs/new`.
  - `GraphTable` con los datos del hook `useGraphs({ status })`.
  - Paginación con botones "Anterior" / "Siguiente" funcionales.
  - Estado vacío específico por filtro (ej. "No hay grafos activos").

- **Criterio de aceptación:** Tests del TODO #6 pasan a GREEN.
  `npm run type-check` pasa. Flujo completo (crear → ver detalle →
  editar → archivar) funciona manualmente.
- **Depende de:** #6, #10, #11
- **Commit:** `feat(graphs): implement GraphCreatePage, GraphEditPage and enhanced GraphsPage [SPRINT-006 #12]`

### 13. [impl] Actualizar routing en App.tsx + mover páginas a features/

- **Agente:** @developer
- **Descripción:** Actualizar el routing para registrar las nuevas rutas
  y mover las páginas de `pages/` a `features/graphs/pages/`.

  **`App.tsx`:**
  ```tsx
  import { GraphsPage, GraphDetailPage, GraphCreatePage, GraphEditPage }
    from "@/features/graphs";

  // En el Route protegido:
  <Route path="/graphs" element={<GraphsPage />} />
  <Route path="/graphs/new" element={<GraphCreatePage />} />
  <Route path="/graphs/:id" element={<GraphDetailPage />} />
  <Route path="/graphs/:id/edit" element={<GraphEditPage />} />
  ```

  Eliminar `pages/GraphsPage.tsx` (reemplazada por la de la feature).

  Actualizar `AppLayout.tsx` para que el link "Grafos" en el sidebar
  vaya a `/graphs`.

  **`features/graphs/index.ts`** — barrel export de todas las páginas
  y componentes públicos de la feature.

- **Criterio de aceptación:** `npm run build` compila. Ningún import
  apunta a la ruta `pages/GraphsPage.tsx` obsoleta.
- **Depende de:** #12
- **Commit:** `feat(graphs): update routing and move pages to feature module [SPRINT-006 #13]`

### 14. [test] Smoke test del flujo completo de grafos

- **Agente:** @qa
- **Descripción:** Ampliar `dashboard/scripts/smoke.sh` con la
  verificación del flujo de grafos. Añadir en el smoke check:

  ```bash
  # Verifica que las 4 rutas de grafos están en el bundle
  grep -q "GraphsPage" dist/assets/*.js
  grep -q "GraphDetailPage" dist/assets/*.js
  grep -q "GraphCreatePage" dist/assets/*.js
  ```

  Y documentar en `docs/sprints/SPRINT-006-dashboard-feature-grafos.md`
  el procedimiento de prueba manual E2E:
  1. Abrir `/graphs` → ver tabla vacía o con grafos.
  2. Hacer clic "Nuevo grafo" → ver formulario.
  3. Seleccionar plantilla "LLM Call simple" → ver JSON.
  4. Completar todos los campos → submit.
  5. Verificar redirección a detalle del grafo creado.
  6. Navegar a pestaña "Definición" → ver nodos y aristas.
  7. Hacer clic "Archivar" → confirmar → ver toast.
  8. Verificar badge cambia a "archived".

- **Criterio de aceptación:** `make dashboard-check` pasa con los
  nuevos checks. El procedimiento manual E2E documentado.
- **Depende de:** #13
- **Commit:** `test(graphs): add smoke checks and E2E procedure for graphs feature [SPRINT-006 #14]`

### 15. [docs] Actualizar docs/index.md y docs/log.md

- **Agente:** @docs
- **Descripción:** Añadir SPRINT-006 a la tabla de sprints. Actualizar
  la sección de servicios marcando el dashboard con "feature grafos
  implementada". Actualizar `docs/log.md`.
- **Criterio de aceptación:** Índice y log actualizados.
- **Depende de:** #14
- **Commit:** `docs(graphs): update index with SPRINT-006 results [SPRINT-006 #15]`

## Matriz de trazabilidad

| Spec / ADR | Regla | TODO | Artefacto | Verificado por |
|------------|-------|------|-----------|----------------|
| ADR-016 | 7 patrones de nodo | #2, #8 | `DEFINITION_TEMPLATES`, `NodePatternBadge` | tests + type-check |
| ADR-016 | 5 patrones de arista | #9, #8 | `EdgeTypeBadge`, `GraphDefinitionViewer` | tests + type-check |
| ADR-016 | Grafo: `entry_node` requerido | #2 | `graphFormSchema.entry_node` | `TestVersionValidation` |
| ADR-009 regla 2 | TypeScript strict: sin any | todos | todos los ficheros tsx/ts | `npm run type-check` |
| ADR-009 regla 4 | TanStack Query, no useEffect | #3 | todos los hooks | code review |
| ADR-009 regla 8 | shadcn/ui + Tailwind, sin CSS-in-JS | #8–#13 | todos los componentes | build + code review |
| ADR-019 regla 4 | Dark mode en todos los componentes | #8–#13 | CSS variables en todos | inspección visual |
| ADR-019 regla 5 | Accesibilidad Radix UI | #1, #8–#13 | componentes shadcn/ui | Radix base |
| ADR-010 regla 9 | Tipos TS generados desde OpenAPI | #3 | hooks usan tipos generados | `npm run type-check` |
| ADR-009 (forms) | React Hook Form + Zod | #2, #10 | `GraphForm` | tests TODO #4 |
| SPRINT-003 | DELETE archiva, no borra | #11 | `GraphArchiveDialog` y toast | tests TODO #7 |
| SPRINT-003 | PUT solo grafos draft | #12 | `GraphEditPage` guarda + alerta | tests TODO #5 |

## Criterios de aceptación del sprint

```bash
# 1. TypeScript sin errores
cd dashboard && npm run type-check

# 2. Tests unitarios pasan
cd dashboard && npm test

# 3. Build sin errores
cd dashboard && npm run build

# 4. Linter sin errores
cd dashboard && npm run lint

# 5. Smoke checks
make dashboard-check

# 6. Flujo E2E manual (con servicios levantados)
make docker-up
go run ./services/auth-server/cmd/main.go &   # 8081
go run ./services/orchestrator/cmd/main.go &  # 8080
cd dashboard && npm run dev                   # 5173
# → crear grafo → ver detalle → editar → archivar
```

Adicionalmente:
- Ningún componente nuevo usa `any` en TypeScript.
- `GraphForm` no renderiza más de 200 líneas (ADR-003 adaptado a TSX).
- `GraphDefinitionViewer` muestra correctamente los 4 templates.
- `GraphStatusBadge` diferencia visualmente los 3 estados.
- El filtro de status en `GraphsPage` persiste en los query params de la URL.
- La pestaña "Ejecuciones" en el detalle muestra un estado vacío claro.
- Todos los toasts de error contienen el `code` del `ErrorResponse` de la API.

## Resultado del sprint

_Se completa al finalizar el sprint._

### Tests ejecutados

- Total: —
- Passed: —
- Failed: —

### Ficheros creados/modificados

_Lista generada al cierre._

### Decisiones tomadas durante el sprint

_Cualquier decisión no prevista que requiera un ADR o nota se documenta aquí._

### Observaciones del reviewer

_Pendiente de revisión._
