# SPRINT-006: Graphs feature — listing, detail and creation in the dashboard

## Metadata

- **Start date:** 2026-05-02 (after completing SPRINT-005)
- **Estimated end date:** 2026-05-04
- **Status:** planned
- **ADRs applied:** ADR-009, ADR-010, ADR-016, ADR-019
- **Affected specs:** none (uses endpoints already defined in SPRINT-003)
- **Planning agent:** planner
- **Reviewed by:** pending
- **Depends on:** SPRINT-005 (base dashboard, PKCE, AppLayout, basic GraphsPage)
- **Blocks:** SPRINT-007 (execution monitor), SPRINT-008 (graph visual editor)

## Sprint Objective

Complete the graphs feature in the dashboard: improve the listing with
filtering and actions, implement the detail page with tabs and the
creation page with a validated form, connecting to the five
CRUD endpoints of the orchestrator (SPRINT-003).

At completion: an authenticated user can list, filter, view the detail,
create, edit and archive graphs from the dashboard, with visual feedback on
each operation.

## Scope

### Included

**Reorganization into feature module:**
- Move `pages/GraphsPage.tsx` to `features/graphs/pages/GraphsPage.tsx`.
- Create the complete `features/graphs/` (components, hooks, pages, types, schemas).

**New shadcn/ui components:**
- `form`, `textarea`, `alert-dialog`, `breadcrumb`, `alert`,
  `tabs`, `switch`, `select`, `tooltip`, `scroll-area`.

**Schemas and utilities:**
- `features/graphs/schemas/graph-form.schema.ts` — Zod schema for the form.
- `features/graphs/lib/definition-templates.ts` — 4 JSON definition templates
  (empty, llm_call, react, router+handlers) derived from ADR-016.

**TanStack Query hooks:**
- `useGraph(id)` — `GET /api/v1/graphs/:id`.
- `useCreateGraph()` — `POST /api/v1/graphs`.
- `useUpdateGraph()` — `PUT /api/v1/graphs/:id`.
- `useArchiveGraph()` — `DELETE /api/v1/graphs/:id`.
- `useGraphs(opts)` refactored — adds `status` parameter for filtering.

**Feature components:**
- `GraphStatusBadge` — badge with color by status (draft/active/archived).
- `GraphTable` — table extracted from the GraphsPage of SPRINT-005, with columns:
  name, version, status, creation date + "View detail" action.
- `GraphDefinitionViewer` — formatted JSON view (nodes + edges) with
  collapsible sections per node.
- `NodePatternBadge` — badge by node pattern (llm_call, react, etc.).
- `EdgeTypeBadge` — badge by edge type (sequential, conditional, etc.).
- `GraphForm` — shared create/edit form with React Hook Form +
  Zod + definition template selector.
- `GraphArchiveDialog` — AlertDialog for archive confirmation.

**Pages (under `features/graphs/pages/`):**
- `GraphsPage` — listing with status filter, pagination, "New graph" button.
- `GraphDetailPage` — detail with 3 tabs: metadata, definition, executions.
- `GraphCreatePage` — page `/graphs/new` with `GraphForm`.
- `GraphEditPage` — page `/graphs/:id/edit` with pre-filled `GraphForm`.

**Routing (`App.tsx`):**
```
/graphs               → GraphsPage
/graphs/new           → GraphCreatePage
/graphs/:id           → GraphDetailPage
/graphs/:id/edit      → GraphEditPage
```

**Breadcrumbs** on all detail/form pages using the
`breadcrumb` component from shadcn/ui.

**Tests:**
- `features/graphs/components/__tests__/GraphForm.test.tsx`
- `features/graphs/pages/__tests__/GraphDetailPage.test.tsx`
- `features/graphs/pages/__tests__/GraphCreatePage.test.tsx`
- `features/graphs/hooks/__tests__/useArchiveGraph.test.ts`

### Excluded

- Visual node/edge editor (drag-and-drop canvas) — SPRINT-008.
- "Executions" tab in the detail with real data (SPRINT-007).
- Graph activation (`status draft → active`) — needs endpoint
  `PATCH /graphs/:id/status` (not implemented in SPRINT-003).
- Duplicate/import/export graph — future sprints.
- Semantic validation of the definition JSON against the ADR-016 JSON Schema
  on the client (only well-formed JSON validation in this sprint).
- Infinite pagination or virtual scroll — basic pagination is sufficient.
- Full-text graph search — the filter in this sprint is by status only.

## Dependencies

- **SPRINT-005 completed:** `AppLayout`, `AuthProvider`, `useAuth`,
  base routing, basic `GraphsPage` (list), base shadcn/ui components
  (button, card, table, skeleton, badge, toast).
- **SPRINT-003 completed:** functional graph CRUD endpoints in
  the orchestrator.
- **`openapi-typescript` generated (`make gen-api-types`):** types
  `GraphResponse`, `GraphInput`, `GraphListResponse` available.

## Behavior Contracts

### C1 — `GraphForm` submit with valid data

```
Given: Form with name="mi-grafo", version="1.0.0", entry_node="llm", definition=valid JSON
When: The user clicks the submit button
Then: onSubmit is called with the values transformed by formToGraphInput
      The fields match the GraphInput structure of the API
      The submit button is disabled while isPending=true (no double submit)
```

### C2 — `useArchiveGraph` — successful DELETE

```
Given: Graph with id="<uuid>" in status="draft", without active executions
When: useArchiveGraph().mutate("<uuid>")
Then: DELETE /api/v1/graphs/<uuid> is called
      On 204 response: the ["graphs"] query is invalidated
      Success toast is shown
      The graph disappears from the list
```

### C3 — Editing blocked for non-draft graphs

```
Given: User navigates to GraphEditPage of a graph with status="active"
When: The page loads
Then: An Alert component is shown instead of the edit form
      The form is not accessible (does not exist in the DOM)
      The graph cannot be modified or submitted
```

## Design

### Directory structure

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

### Zod schema for the form

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

// Converts form values to the API's GraphInput.
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

### Definition templates (ADR-016)

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

### GraphDefinitionViewer — visual structure

The viewer displays the definition in two collapsible sections (using `<details>`
with Tailwind or the Radix `Collapsible` component):

```
▶ Nodes (3)
  ┌─ [llm_call] llm
  │   model: claude-sonnet-4-6
  │   system_prompt: You are a helpful...
  ├─ [react] agent
  └─ [router] router

▶ Edges (2)
  router → handler_a  [conditional]
  router → handler_b  [conditional]
```

If the definition cannot be parsed as JSON, it shows the raw text
in a `<pre>` with an alert warning.

### GraphDetailPage — tabs

| Tab | Content |
|-----|---------|
| Metadata | Card with all graph fields + actions |
| Definition | `GraphDefinitionViewer` with the graph JSON |
| Executions | EmptyState: "Executions will appear here" + "Start execution" button (disabled until SPRINT-007) |

Actions on the Metadata tab (buttons in the page header):
- If `status=draft`: "Edit" button → `/graphs/:id/edit` + "Archive" button (dialog).
- If `status=active`: "Archive" button.
- If `status=archived`: read-only, warning badge.

### Creation flow

```
/graphs/new
  │
  ├── Select template (dropdown with 4 options)
  │   → fills the definition textarea
  │
  ├── Fill in: name, version, description, entry node
  │
  ├── Edit definition JSON in monospace textarea
  │   → onBlur validation: is it valid JSON?
  │
  ├── Toggle: Semantic search / Episodic context
  │
  └── Submit
        ├── OK (201)  → toast "Graph created" → navigate to /graphs/:id
        ├── 409       → error toast "Version X.Y.Z of this graph already exists"
        └── 422       → show inline field errors
```

### Archive flow

```
"Archive" button → AlertDialog
  "Archive this graph? No new executions will be able to start."
  [Cancel] [Archive]
        │
        └── DELETE /api/v1/graphs/:id
              ├── 204 → toast "Graph archived" → navigate to /graphs
              └── 409 → error toast "The graph has active executions"
```

## TODOs

### 1. [infra] Add new shadcn/ui components

- **Agente:** @developer
- **Description:** Add the shadcn/ui components needed for
  this sprint that were not included in SPRINT-005.

  Components to add (with `npx shadcn@latest add` or equivalent):
  `form`, `textarea`, `alert-dialog`, `breadcrumb`, `alert`,
  `tabs`, `switch`, `select`, `tooltip`, `scroll-area`, `collapsible`.

  Verify that `ThemeProvider` and dark mode continue working
  correctly after adding the new components.

- **Acceptance criteria:** All components exist in
  `dashboard/src/components/ui/`. `npm run build` compiles without errors.
- **Dependencies:** none
- **Commit:** `feat(dashboard): add shadcn/ui components for graphs feature [SPRINT-006 #1]`

### 2. [domain] Zod form schema and definition templates

- **Agente:** @developer
- **Description:** Create the schema and templates files according to
  the design in this document.

  **`features/graphs/schemas/graph-form.schema.ts`:** Zod schema with
  the 7 fields, error messages in Spanish, and the
  `formToGraphInput()` function. Import `GraphInput` from the type generated
  by `openapi-typescript`.

  **`features/graphs/lib/definition-templates.ts`:** The 4 templates
  typed as `const` with the ADR-016 patterns. Include a function
  `templateToDefinitionString(id: string): string` that returns the
  serialized JSON with `JSON.stringify(..., null, 2)`.

- **Acceptance criteria:** `npm run type-check` passes. The template types
  match the schemas in
  `specs/patterns/nodes/*.json`.
- **Dependencies:** SPRINT-005 completed (generated types available)
- **Commit:** `feat(graphs): add graph form Zod schema and definition templates [SPRINT-006 #2]`

### 3. [domain] TanStack Query hooks

- **Agente:** @developer
- **Description:** Create the four mutation hooks and the
  detail hook.

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

  Analogous for `useUpdateGraph(id)` and `useArchiveGraph()`.

  `useGraphs()` refactored to accept `status?: string` as an
  additional filter.

- **Acceptance criteria:** `npm run type-check` passes. The parameter
  and return types match the generated types.
- **Dependencies:** #2 (needs the schema types)
- **Commit:** `feat(graphs): add TanStack Query hooks for CRUD operations [SPRINT-006 #3]`

### 4. [test] GraphForm tests (Red)

- **Agente:** @qa
- **Description:** Tests for the form component before
  implementing it.

  ```typescript
  // features/graphs/components/__tests__/GraphForm.test.tsx

  // testRendersAllFields: the form shows fields name, version,
  // description, entry_node, definition textarea and memory toggles.
  test("GraphForm renders all required fields", ...)

  // testVersionValidation: version "abc" shows error "Formato: 1.0.0".
  test("GraphForm shows version format error for non-semver", ...)

  // testDefinitionInvalidJSON: non-JSON text in definition shows
  // the error "JSON inválido" after losing focus.
  test("GraphForm shows JSON error for invalid definition", ...)

  // testTemplateSelector: when selecting a template, the definition
  // textarea is filled with the corresponding JSON.
  test("GraphForm fills definition textarea on template select", ...)

  // testSubmitCallsOnSubmit: with valid data, submit calls the
  // onSubmit function with the values transformed by formToGraphInput.
  test("GraphForm calls onSubmit with transformed values on valid submit", ...)

  // testSubmitDisabledWhilePending: the submit button is disabled
  // while isPending=true.
  test("GraphForm disables submit while pending", ...)
  ```

- **Acceptance criteria:** Tests in RED before TODO #8.
  GREEN after implementing `GraphForm`.
- **Dependencies:** #2, #1 (required shadcn/ui components)
- **Commit:** `test(graphs): add GraphForm component tests [SPRINT-006 #4]`

### 5. [test] GraphDetailPage tests with MSW (Red)

- **Agente:** @qa
- **Description:** Tests for the detail page simulating the API.

  ```typescript
  // features/graphs/pages/__tests__/GraphDetailPage.test.tsx

  // testShowsGraphMetadata: shows name, version and status badge.
  test("GraphDetailPage renders graph metadata", ...)

  // testTabNavigationMetadata: by default the "Metadata" tab
  // is shown as active.
  test("GraphDetailPage shows Metadatos tab by default", ...)

  // testTabDefinition: clicking "Definition" shows
  // the GraphDefinitionViewer with the mock graph's nodes.
  test("GraphDetailPage shows definition viewer on Definición tab", ...)

  // testEditButtonForDraft: for a draft graph the "Edit" button is shown.
  test("GraphDetailPage shows Edit button for draft graph", ...)

  // testNoEditButtonForArchived: for an archived graph the
  // "Edit" button does not appear.
  test("GraphDetailPage hides Edit button for archived graph", ...)

  // testArchiveDialogAppears: clicking "Archive" shows
  // the confirmation AlertDialog.
  test("GraphDetailPage shows archive confirmation dialog", ...)

  // testShowsSkeletonWhileLoading: shows skeleton while loading.
  test("GraphDetailPage shows skeleton while loading", ...)

  // testShowsNotFoundOn404: if the API returns 404, shows an
  // appropriate message.
  test("GraphDetailPage shows not found state on 404", ...)
  ```

- **Acceptance criteria:** Tests in RED before TODO #11.
  GREEN after implementing `GraphDetailPage`.
- **Dependencies:** #3, #1
- **Commit:** `test(graphs): add GraphDetailPage tests with MSW [SPRINT-006 #5]`

### 6. [test] GraphCreatePage tests with MSW (Red)

- **Agente:** @qa
- **Description:** Tests for the complete creation flow.

  ```typescript
  // features/graphs/pages/__tests__/GraphCreatePage.test.tsx

  // testRendersCreateForm: the page shows the creation form
  // with all fields.
  test("GraphCreatePage renders the create form", ...)

  // testSuccessfulCreation: completing the form with valid data
  // and MSW responding 201, shows a success toast and
  // navigates to the detail page.
  test("GraphCreatePage shows success toast and navigates on 201", ...)

  // testConflictError: if MSW responds 409, shows an error toast
  // with the message from GRAPH_DUPLICATE_VERSION code.
  test("GraphCreatePage shows conflict toast on 409", ...)

  // testValidationErrors: if MSW responds 422 with field details,
  // errors are shown inline under each field.
  test("GraphCreatePage shows inline errors on 422", ...)

  // testBreadcrumbNavigation: breadcrumbs show
  // "Graphs > New graph" with a working link to /graphs.
  test("GraphCreatePage renders correct breadcrumbs", ...)
  ```

- **Acceptance criteria:** Tests in RED before TODO #12.
  GREEN after implementing `GraphCreatePage`.
- **Dependencies:** #3, #4
- **Commit:** `test(graphs): add GraphCreatePage tests with MSW [SPRINT-006 #6]`

### 7. [test] useArchiveGraph tests with MSW (Red)

- **Agente:** @qa
- **Description:** Tests for the archive mutation hook.

  ```typescript
  // features/graphs/hooks/__tests__/useArchiveGraph.test.ts

  // testArchiveSuccessInvalidatesCache: on successful archive (204),
  // the ["graphs"] query is invalidated.
  test("useArchiveGraph invalidates graphs cache on success", ...)

  // testArchiveConflictThrows: if the server responds 409, the
  // mutation rejects with the API error.
  test("useArchiveGraph throws on 409 conflict", ...)
  ```

- **Acceptance criteria:** Tests in RED before TODO #11.
  GREEN after implementing the hook + detail page.
- **Dependencies:** #3
- **Commit:** `test(graphs): add useArchiveGraph hook tests [SPRINT-006 #7]`

### 8. [impl] GraphStatusBadge, NodePatternBadge, EdgeTypeBadge, GraphTable

- **Agente:** @developer
- **Description:** Primitive components of the feature.

  **`GraphStatusBadge`:** shadcn/ui Badge with variant by status:
  - `draft` → `secondary` (grey)
  - `active` → `default` (blue)
  - `archived` → `outline` (border, muted text)

  **`NodePatternBadge`:** Badge with variant by pattern. The 7 patterns
  from ADR-016 with semantically differentiated colors:
  - `llm_call` → blue
  - `react` → green
  - `reflection` → purple
  - `tool_use` → orange
  - `router` → yellow
  - `guardrail` → red
  - `subgraph` → dark grey

  **`EdgeTypeBadge`:** Badge for the 5 edge types from ADR-016.

  **`GraphTable`:** Table extracted from the `GraphsPage` of SPRINT-005.
  Columns: name (with link to detail), version, status badge, creation
  date (formatted). No inline actions (actions are in the detail).

- **Acceptance criteria:** The three badges render with the
  correct colors. `npm run type-check` without errors.
- **Dependencies:** #1
- **Commit:** `feat(graphs): add GraphStatusBadge, NodePatternBadge, EdgeTypeBadge, GraphTable [SPRINT-006 #8]`

### 9. [impl] GraphDefinitionViewer

- **Agente:** @developer
- **Description:** Component that displays a graph definition
  in a readable way, without editor or drag-and-drop.

  Props: `definition: unknown` (the parsed graph JSON).

  Logic:
  1. Validate that `definition` is an object with `nodes` and `edges`.
  2. If invalid: show `<Alert variant="destructive">` with the raw JSON
     in `<pre>`.
  3. If valid: show two `<Collapsible>` sections:

     **Nodes** (`definition.nodes` is a map `{[key]: NodeDefinition}`):
     - One row per node: `NodePatternBadge` + node key +
       config summary (model if present, max_iterations if present).
     - Expandable to see the full node JSON.

     **Edges** (`definition.edges` is an array):
     - One row per edge: `from` → `to` + `EdgeTypeBadge`.
     - If `type=conditional`: show number of conditions.

  The component uses `<ScrollArea>` if the content is very long.

- **Acceptance criteria:** The component renders correctly
  the 4 definition templates from `definition-templates.ts`.
  `npm run type-check` passes.
- **Dependencies:** #8, #1
- **Commit:** `feat(graphs): add GraphDefinitionViewer component [SPRINT-006 #9]`

### 10. [impl] GraphForm — shared create/edit form

- **Agente:** @developer
- **Description:** Form component with React Hook Form + Zod.
  Used in both `GraphCreatePage` and `GraphEditPage`.

  Props:
  ```typescript
  interface GraphFormProps {
    defaultValues?: Partial<GraphFormValues>;
    onSubmit: (values: GraphFormValues) => Promise<void>;
    isPending: boolean;
    submitLabel: string;
  }
  ```

  Form fields (visual order):
  1. `name` — Input with label "Name".
  2. `version` — Input with label "Version" and placeholder "1.0.0".
  3. `description` — Textarea with label "Description (optional)".
  4. `entry_node` — Input with label "Entry node" and explanatory tooltip:
     "Key of the node where execution begins".
  5. Template selector — `Select` with the 4 options from
     `DEFINITION_TEMPLATES`. On change: fills the textarea.
  6. `definition` — Textarea with monospace font (`font-mono` Tailwind),
     minimum 12 rows, with JSON validation on `onBlur`.
  7. "Memory" section (collapsible with `Collapsible`):
     - `memory_semantic_search` — Switch with label.
     - `memory_episode_context` — Numeric input.
  8. Submit button with `submitLabel` and spinner if `isPending`.

  Errors: below each field using `<FormMessage>` from shadcn/ui.

- **Acceptance criteria:** Tests from TODO #4 pass GREEN.
  `npm run type-check` passes. No component uses `any`.
- **Dependencies:** #4, #1, #2
- **Commit:** `feat(graphs): implement GraphForm with React Hook Form and Zod [SPRINT-006 #10]`

### 11. [impl] GraphDetailPage + GraphArchiveDialog

- **Agente:** @developer
- **Description:** Detail page with the 3 tabs and the archive dialog.

  **`GraphDetailPage`:**
  - `useGraph(id)` with `useParams()`.
  - Skeleton while loading (same dimensions as the content).
  - Breadcrumbs: "Graphs" → graph name.
  - Header: name, version, `GraphStatusBadge`, conditional actions by status.
  - Tabs with shadcn/ui `Tabs`:
    - **Metadata:** `Card` with field grid (description, entry_node,
      dates). `memory_config` shown as read-only checkboxes.
    - **Definition:** `GraphDefinitionViewer` with the graph `definition`.
    - **Executions:** `EmptyState` with icon, text "No executions"
      and "Start execution" button disabled with tooltip
      "Available in the next version".

  **`GraphArchiveDialog`:**
  - shadcn/ui `AlertDialog`.
  - Confirmation message.
  - Calls `useArchiveGraph()`, shows result toast.
  - On 409: toast with "The graph has active executions".

- **Acceptance criteria:** Tests from TODO #5 and #7 pass GREEN.
  `npm run type-check` passes.
- **Dependencies:** #5, #7, #9, #3
- **Commit:** `feat(graphs): implement GraphDetailPage with tabs and archive dialog [SPRINT-006 #11]`

### 12. [impl] GraphCreatePage, GraphEditPage and enhanced GraphsPage

- **Agente:** @developer
- **Description:** The three remaining pages.

  **`GraphCreatePage`** (`/graphs/new`):
  - Breadcrumbs: "Graphs" → "New graph".
  - Title "Create graph".
  - `GraphForm` with `submitLabel="Create"` and `useCreateGraph()`.
  - On success (201): `toast.success("Graph created")` + `navigate(/graphs/:id)`.
  - On 409: `toast.error("Version X.Y.Z of ... already exists")`.
  - On 422: inline errors come via the response and are mapped
    to the form fields.

  **`GraphEditPage`** (`/graphs/:id/edit`):
  - Breadcrumbs: "Graphs" → graph name → "Edit".
  - Loads the graph with `useGraph(id)`.
  - If `status !== "draft"`: shows `Alert` "Only draft graphs can be edited"
    and "Back" button.
  - `GraphForm` pre-filled with current data.
  - Uses `useUpdateGraph()`.
  - On success: `toast.success("Graph updated")` + `navigate(/graphs/:id)`.

  **Enhanced `GraphsPage`:**
  - Status filter: `Select` with options "All", "Draft",
    "Active", "Archived". Saves selection in `searchParams`.
  - "New graph" button in header → navigates to `/graphs/new`.
  - `GraphTable` with data from the `useGraphs({ status })` hook.
  - Pagination with functional "Previous" / "Next" buttons.
  - Empty state specific to each filter (e.g. "No active graphs").

- **Acceptance criteria:** Tests from TODO #6 pass GREEN.
  `npm run type-check` passes. Complete flow (create → view detail →
  edit → archive) works manually.
- **Dependencies:** #6, #10, #11
- **Commit:** `feat(graphs): implement GraphCreatePage, GraphEditPage and enhanced GraphsPage [SPRINT-006 #12]`

### 13. [impl] Update routing in App.tsx + move pages to features/

- **Agente:** @developer
- **Description:** Update the routing to register the new routes
  and move pages from `pages/` to `features/graphs/pages/`.

  **`App.tsx`:**
  ```tsx
  import { GraphsPage, GraphDetailPage, GraphCreatePage, GraphEditPage }
    from "@/features/graphs";

  // In the protected Route:
  <Route path="/graphs" element={<GraphsPage />} />
  <Route path="/graphs/new" element={<GraphCreatePage />} />
  <Route path="/graphs/:id" element={<GraphDetailPage />} />
  <Route path="/graphs/:id/edit" element={<GraphEditPage />} />
  ```

  Delete `pages/GraphsPage.tsx` (replaced by the feature one).

  Update `AppLayout.tsx` so the "Graphs" link in the sidebar
  points to `/graphs`.

  **`features/graphs/index.ts`** — barrel export of all public pages
  and components of the feature.

- **Acceptance criteria:** `npm run build` compiles. No import
  points to the obsolete `pages/GraphsPage.tsx` path.
- **Dependencies:** #12
- **Commit:** `feat(graphs): update routing and move pages to feature module [SPRINT-006 #13]`

### 14. [test] Smoke test for the complete graphs flow

- **Agente:** @qa
- **Description:** Extend `dashboard/scripts/smoke.sh` with the
  graphs flow verification. Add to the smoke check:

  ```bash
  # Verify that the 4 graph routes are in the bundle
  grep -q "GraphsPage" dist/assets/*.js
  grep -q "GraphDetailPage" dist/assets/*.js
  grep -q "GraphCreatePage" dist/assets/*.js
  ```

  And document in `docs/sprints/SPRINT-006-dashboard-feature-grafos.md`
  the manual E2E test procedure:
  1. Open `/graphs` → see empty table or with graphs.
  2. Click "New graph" → see form.
  3. Select template "LLM Call simple" → see JSON.
  4. Fill in all fields → submit.
  5. Verify redirect to the created graph detail.
  6. Navigate to "Definition" tab → see nodes and edges.
  7. Click "Archive" → confirm → see toast.
  8. Verify badge changes to "archived".

- **Acceptance criteria:** `make dashboard-check` passes with the
  new checks. Manual E2E procedure documented.
- **Dependencies:** #13
- **Commit:** `test(graphs): add smoke checks and E2E procedure for graphs feature [SPRINT-006 #14]`

### Manual E2E Test Procedure (TODO #14)

With all services running (`make docker-up`, auth-server :8081, orchestrator :8080, dashboard :5173):

1. Open `/graphs` → see empty table or graphs list with status filter.
2. Click "New graph" button → form page loads.
3. Select template "LLM Call simple" from the dropdown → definition textarea fills with JSON.
4. Fill in name, version (e.g. `1.0.0`), entry node → click "Create".
5. Verify redirect to `/graphs/{id}` detail page.
6. Navigate to "Definition" tab → see nodes and edges with pattern badges.
7. Click "Archive" button → confirm in dialog → success toast appears.
8. Verify status badge changes to "archived" and edit button disappears.

### 15. [docs] Update docs/index.md and docs/log.md

- **Agente:** @docs
- **Description:** Add SPRINT-006 to the sprints table. Update
  the services section marking the dashboard with "graphs feature
  implemented". Update `docs/log.md`.
- **Acceptance criteria:** Index and log updated.
- **Dependencies:** #14
- **Commit:** `docs(graphs): update index with SPRINT-006 results [SPRINT-006 #15]`

## Traceability Matrix

| Spec / ADR | Rule | TODO | Artifact | Verified by |
|------------|------|------|----------|-------------|
| ADR-016 | 7 node patterns | #2, #8 | `DEFINITION_TEMPLATES`, `NodePatternBadge` | tests + type-check |
| ADR-016 | 5 edge patterns | #9, #8 | `EdgeTypeBadge`, `GraphDefinitionViewer` | tests + type-check |
| ADR-016 | Graph: `entry_node` required | #2 | `graphFormSchema.entry_node` | `TestVersionValidation` |
| ADR-009 rule 2 | TypeScript strict: no any | all | all tsx/ts files | `npm run type-check` |
| ADR-009 rule 4 | TanStack Query, no useEffect | #3 | all hooks | code review |
| ADR-009 rule 8 | shadcn/ui + Tailwind, no CSS-in-JS | #8–#13 | all components | build + code review |
| ADR-019 rule 4 | Dark mode on all components | #8–#13 | CSS variables in all | visual inspection |
| ADR-019 rule 5 | Radix UI accessibility | #1, #8–#13 | shadcn/ui components | Radix base |
| ADR-010 rule 9 | TS types generated from OpenAPI | #3 | hooks use generated types | `npm run type-check` |
| ADR-009 (forms) | React Hook Form + Zod | #2, #10 | `GraphForm` | tests TODO #4 |
| SPRINT-003 | DELETE archives, does not delete | #11 | `GraphArchiveDialog` and toast | tests TODO #7 |
| SPRINT-003 | PUT only draft graphs | #12 | `GraphEditPage` saves + alert | tests TODO #5 |

## Sprint acceptance criteria

```bash
# 1. TypeScript without errors
cd dashboard && npm run type-check

# 2. Unit tests pass
cd dashboard && npm test

# 3. Build without errors
cd dashboard && npm run build

# 4. Linter without errors
cd dashboard && npm run lint

# 5. Smoke checks
make dashboard-check

# 6. Manual E2E flow (with services running)
make docker-up
go run ./services/auth-server/cmd/main.go &   # 8081
go run ./services/orchestrator/cmd/main.go &  # 8080
cd dashboard && npm run dev                   # 5173
# → create graph → view detail → edit → archive
```

Additionally:
- No new component uses `any` in TypeScript.
- `GraphForm` renders no more than 200 lines (ADR-003 adapted to TSX).
- `GraphDefinitionViewer` correctly shows the 4 templates.
- `GraphStatusBadge` visually differentiates the 3 states.
- The status filter in `GraphsPage` persists in the URL query params.
- The "Executions" tab in the detail shows a clear empty state.
- All error toasts contain the `code` from the API's `ErrorResponse`.

## Sprint Result

_Completed at the end of the sprint._

### Tests executed

- Total: —
- Passed: —
- Failed: —

### Files created/modified

_List generated at close._

### Decisions made during the sprint

_Any unforeseen decision requiring an ADR or note is documented here._

### Reviewer observations

_Pending review._
