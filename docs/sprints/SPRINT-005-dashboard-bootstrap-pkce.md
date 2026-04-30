# SPRINT-005: Bootstrap del dashboard — React 19, shadcn/ui, PKCE y tipos OpenAPI

## Metadata

- **Fecha inicio:** 2026-04-30 (tras completar SPRINT-003 y SPRINT-004)
- **Fecha fin estimada:** 2026-05-02
- **Estado:** planificado
- **ADRs aplicados:** ADR-009, ADR-010, ADR-012, ADR-019
- **Specs afectadas:**
  - `specs/paths/auth.yaml` — añadir GET/POST `/authorize`, POST `/token`
  - `specs/schemas/auth.yaml` — añadir `TokenRequest`, `AuthorizeParams`
  - `specs/openapi.yaml` — `$ref` a los nuevos paths
  - `dashboard/src/api/types.gen.ts` — generado por `openapi-typescript`
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Depende de:** SPRINT-003 (API REST + orchestrator), SPRINT-004 (auth-server login, JWT, JWKS)
- **Bloquea:** SPRINT-006 (graphs editor), SPRINT-007 (execution monitor AG-UI)

## Objetivo del sprint

Bootstrapear el dashboard React 19 desde cero con la toolchain completa,
integrar el flujo OAuth 2.1 PKCE contra el auth-server, generar tipos
TypeScript desde la spec OpenAPI, y renderizar una página funcional de
listado de grafos autenticada mediante TanStack Query.

Al finalizar: el dashboard arranca en `npm run dev`, redirige al auth-server
para login PKCE, recibe y almacena el JWT en memoria, y muestra el listado
de grafos consumiendo `/api/v1/graphs` con tipos generados.

## Alcance

### Incluido

**Extensión del auth-server (necesaria para PKCE):**
- `GET /authorize` — renderiza un formulario HTML mínimo de login.
- `POST /authorize` — procesa login, genera código de autorización, redirige.
- `POST /token` — intercambia `code` + `code_verifier` por JWT.
- Store en memoria (`sync.Map` + goroutine de limpieza) para códigos de
  autorización (TTL 60s). Documentado como limitación: no distribuido.

**Spec OpenAPI:**
- `GET /authorize` y `POST /authorize` en `specs/paths/auth.yaml`.
- `POST /token` en `specs/paths/auth.yaml`.
- Schemas `TokenRequest`, `AuthorizeParams` en `specs/schemas/auth.yaml`.

**Dashboard — setup base:**
- `dashboard/package.json` con dependencias completas.
- `dashboard/vite.config.ts` — React plugin, path aliases.
- `dashboard/tsconfig.json` — `strict: true`, `noImplicitAny: true`.
- `dashboard/index.html` — entry HTML.
- `dashboard/tailwind.config.ts` + `dashboard/src/styles/globals.css` —
  CSS variables, dark mode con `class` strategy.
- `dashboard/components.json` — configuración shadcn/ui.
- `dashboard/src/lib/utils.ts` — helper `cn()`.

**shadcn/ui — componentes del sprint:**
- `button`, `input`, `label`, `card`, `table`, `badge`, `skeleton`,
  `toast`, `separator`, `dropdown-menu`, `avatar`.

**Generación de tipos TypeScript desde OpenAPI:**
- `dashboard/src/api/types.gen.ts` — generado por `openapi-typescript`.
- `dashboard/src/api/client.ts` — `openapi-fetch` con inyección del
  Bearer token desde el contexto de auth.
- `Makefile` target `gen-api-types`.

**Módulo de autenticación PKCE (`dashboard/src/auth/`):**
- `pkce.ts` — `generateVerifier()`, `generateChallenge(verifier)` con
  Web Crypto API (sin dependencias externas).
- `AuthContext.tsx` — `AuthContext` + tipos `AuthState`.
- `AuthProvider.tsx` — estado de auth en `useRef` (token en memoria),
  manejo del callback PKCE, persistencia de sesión en `sessionStorage`
  solo para `code_verifier` y `state` (durante el redirect).
- `useAuth.ts` — hook `useAuth()`.
- `ProtectedRoute.tsx` — guarda de rutas: redirige a PKCE si no hay token.
- `AuthCallback.tsx` — componente de callback: intercambia `code` por token.

**API client con TanStack Query:**
- `dashboard/src/hooks/useGraphs.ts` — hook `useGraphs(opts)` con
  TanStack Query v5.
- `dashboard/src/hooks/useGraph.ts` — hook `useGraph(id)`.

**Páginas y layout:**
- `dashboard/src/layouts/AppLayout.tsx` — sidebar + header shell,
  dark mode toggle, avatar de usuario.
- `dashboard/src/pages/GraphsPage.tsx` — tabla de grafos con shadcn
  DataTable, skeleton mientras carga, badge de status.
- `dashboard/src/pages/AuthCallback.tsx` — página de callback OAuth.
- `dashboard/src/pages/NotFoundPage.tsx`.
- `dashboard/src/App.tsx` — routing con React Router v7.
- `dashboard/src/main.tsx` — entry point con `QueryClientProvider` +
  `AuthProvider` + `ThemeProvider`.

**Tests (Vitest + React Testing Library + MSW):**
- `dashboard/vitest.config.ts`.
- `dashboard/src/test/setup.ts` — configuración global.
- `dashboard/src/auth/__tests__/pkce.test.ts` — tests de crypto.
- `dashboard/src/auth/__tests__/AuthProvider.test.tsx` — tests de estado.
- `dashboard/src/pages/__tests__/GraphsPage.test.tsx` — tests con MSW.

### Excluido

- Graph editor visual (SPRINT-006).
- Execution monitor AG-UI / WebSocket (SPRINT-007).
- Catálogo de paquetes (SPRINT-catalog-frontend).
- Module Federation / microfrontales (SPRINT-mfe-001).
- Refresh tokens / token rotation (SPRINT-ABAC).
- Gestión de usuarios / settings UI (SPRINT-ABAC-frontend).
- `POST /api/v1/graphs` desde el dashboard (solo lectura en este sprint).
- Cliente A2A / MCP registry UI.
- Formulario de creación de grafos.

## Dependencias

- **SPRINT-003 completado:** `GET /api/v1/graphs` funcional en el orchestrator.
- **SPRINT-004 completado:** auth-server con `POST /api/v1/auth/login`,
  `POST /api/v1/auth/register`, `GET /.well-known/jwks.json` y JWT RS256.
- **Node.js 22+** y **npm 10+** en el entorno de desarrollo.
- **`specs/openapi.yaml`** válido con los paths de graphs y auth de
  SPRINT-003 y SPRINT-004.

## Contratos de comportamiento

### C1 — Inicio del flujo PKCE — redirección a autorización

```
Given: Dashboard en localhost:5173 sin token en memoria, auth-server en localhost:8081
When: El usuario navega a /graphs (ruta protegida)
Then: El dashboard redirige a /authorize con parámetros:
      response_type=code, client_id, redirect_uri, code_challenge, code_challenge_method=S256, state
      code_challenge = base64url(SHA-256(code_verifier)) conforme RFC 7636
```

### C2 — Callback PKCE — intercambio de código por token

```
Given: code_verifier en sessionStorage, código de autorización válido en URL
When: AuthCallback procesa /auth/callback?code=X&state=Y
Then: POST /token se llama con code, code_verifier, grant_type=authorization_code
      El token se almacena en useRef en memoria (NO en localStorage)
      sessionStorage se limpia (code_verifier eliminado)
      El usuario es redirigido a /graphs
```

### C3 — `generateChallenge(verifier)` — determinismo y formato

```
Given: verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
When: generateChallenge(verifier)
Then: El resultado es base64url(SHA-256(verifier)) sin padding '='
      No contiene '+' ni '/' (solo caracteres URL-safe)
      La función es pura: misma entrada → mismo resultado
```

## Diseño

### Flujo PKCE completo

```
Dashboard (SPA)                    Auth-server
    │                                   │
    │ 1. /graphs → no token             │
    │                                   │
    │ 2. generateVerifier()             │
    │    generateChallenge(verifier)    │
    │    sessionStorage: verifier+state │
    │                                   │
    │ 3. GET /authorize?                │
    │    response_type=code             │
    │    &client_id=dago-dashboard      │
    │    &redirect_uri=.../callback     │
    │    &code_challenge=<b64url>       │
    │    &code_challenge_method=S256    │
    │    &state=<random>                │
    │ ─────────────────────────────────▶│
    │                                   │ renderiza <form>
    │                                   │ (email + password)
    │ ◀─────────────────────────────────│
    │                                   │
    │ 4. POST /authorize (form submit)  │
    │ ─────────────────────────────────▶│
    │                                   │ verifica credenciales
    │                                   │ genera code (random 32B)
    │                                   │ store: hash(code) → {user, challenge, ...} TTL 60s
    │ ◀─────────────────────────────────│
    │  302 → .../callback?code=X&state=Y│
    │                                   │
    │ 5. AuthCallback: extrae code      │
    │    lee verifier de sessionStorage │
    │    POST /token {                  │
    │      grant_type: authorization_code│
    │      code, code_verifier,         │
    │      redirect_uri, client_id      │
    │    }                              │
    │ ─────────────────────────────────▶│
    │                                   │ verifica: SHA-256(verifier)==challenge
    │                                   │ emite JWT (llama TokenIssuer)
    │ ◀─────────────────────────────────│
    │  { access_token, token_type, ... }│
    │                                   │
    │ 6. token → AuthContext (memory)   │
    │    sessionStorage: limpieza       │
    │    → redirige a /graphs           │
```

### Store de códigos de autorización (auth-server)

```go
// services/auth-server/internal/oauth/code_store.go
type AuthorizationCodeStore interface {
    Store(ctx context.Context, code AuthorizationCode) error
    Consume(ctx context.Context, codeHash string) (*AuthorizationCode, error)
}

type AuthorizationCode struct {
    CodeHash     string    // SHA-256(code_plaintext) — nunca almacenar el código en claro
    UserID       uuid.UUID
    CodeChallenge string   // recibido del cliente
    RedirectURI  string
    ClientID     string
    ExpiresAt    time.Time
}
```

Implementación del sprint: `InMemoryCodeStore` con `sync.Map` + goroutine
de limpieza cada 30s. Puerto para futura implementación en Valkey.

### Validación de redirect_uri

Para el bootstrap, el auth-server acepta cualquier `redirect_uri` que
comience por `http://localhost`. En producción: registro de clientes en BD.
Esta limitación se documenta en el código con un comentario `// TODO(SPRINT-ABAC): validate against registered clients`.

### Estado de autenticación en el dashboard

```typescript
// dashboard/src/auth/AuthContext.tsx
interface AuthState {
  token: string | null;       // JWT en memoria, nunca en localStorage
  user: UserInfo | null;      // Claims extraídas del JWT (decodificado, no validado en cliente)
  isAuthenticated: boolean;
  isLoading: boolean;
}
```

El token se almacena en `useRef` dentro del provider — persiste entre
renders sin triggear re-renders innecesarios. El estado `AuthState` sí
usa `useState` para que los consumidores se actualicen.

**NOTA:** El token en `useRef` se pierde al recargar la página (comportamiento
correcto per ADR-012). La recarga inicia un nuevo flujo PKCE automáticamente.

### API client con tipos generados

```typescript
// dashboard/src/api/client.ts
import createClient from "openapi-fetch";
import type { paths } from "./types.gen";

export function createApiClient(getToken: () => string | null) {
  const client = createClient<paths>({ baseUrl: import.meta.env.VITE_API_URL });

  client.use({
    onRequest({ request }) {
      const token = getToken();
      if (token) request.headers.set("Authorization", `Bearer ${token}`);
      return request;
    },
  });

  return client;
}
```

El cliente se crea dentro del `AuthProvider` y se expone vía contexto.
Los hooks de TanStack Query lo consumen mediante `useApiClient()`.

### Variables de entorno del dashboard

```bash
# dashboard/.env.development
VITE_API_URL=http://localhost:8080
VITE_AUTH_URL=http://localhost:8081
VITE_AUTH_CLIENT_ID=dago-dashboard
VITE_AUTH_REDIRECT_URI=http://localhost:5173/auth/callback
```

`.env.development` se commitea. `.env.production` y `.env.local` en `.gitignore`.

## TODOs

### 1. [spec] Añadir endpoints OAuth PKCE a specs/paths/auth.yaml

- **Agente:** @developer
- **Descripción:** Ampliar `specs/paths/auth.yaml` con los tres nuevos
  endpoints del flujo PKCE. Añadir schemas `TokenRequest` y
  `AuthorizeParams` en `specs/schemas/auth.yaml`.

  Endpoints a añadir:

  **`GET /authorize`:** Parámetros query: `response_type` (requerido,
  `"code"`), `client_id` (requerido), `redirect_uri` (requerido),
  `code_challenge` (requerido), `code_challenge_method` (requerido,
  `"S256"`), `state` (recomendado). Response: `200 text/html`
  (formulario de login) o `302` redirect en error.

  **`POST /authorize`:** `application/x-www-form-urlencoded` con los
  mismos params + `email` + `password`. Response: `302` redirect con
  `code` y `state` en query params.

  **`POST /token`:** JSON body `TokenRequest` con `grant_type`,
  `code`, `code_verifier`, `redirect_uri`, `client_id`. Response:
  `200 TokenResponse` (ya definido en SPRINT-004) o `400 ErrorResponse`.

  **Nota ADR-010:** `/authorize` y `/token` son endpoints estándar
  OAuth — excepción documentada al prefijo `/api/v1/` (igual que JWKS).

- **Criterio de aceptación:** `swagger-cli validate specs/openapi.yaml`
  pasa. Los tres endpoints están documentados con todos sus parámetros
  y códigos de respuesta.
- **Depende de:** ninguno
- **Commit:** `spec(openapi): add OAuth 2.1 PKCE authorize and token endpoints [SPRINT-005 #1]`

### 2. [infra] Implementar auth-server: store de códigos + GET/POST /authorize + POST /token

- **Agente:** @developer
- **Descripción:** Extender el auth-server con el flujo PKCE completo.

  **`services/auth-server/internal/oauth/code_store.go`:**
  - Interfaz `AuthorizationCodeStore` y tipo `AuthorizationCode`.
  - Implementación `InMemoryCodeStore` con `sync.RWMutex` (no `sync.Map`
    para poder iterar con lock en el cleanup) y goroutine de limpieza.
  - `Store()`: calcula `SHA-256(code_plaintext)`, almacena el hash.
  - `Consume()`: busca por hash, verifica `ExpiresAt`, borra la entrada
    (one-time use), devuelve la struct.

  **`services/auth-server/internal/handler/oauth.go`:**

  `GetAuthorize` handler:
  1. Validar parámetros (response_type=code, client_id, redirect_uri,
     code_challenge, code_challenge_method=S256).
  2. Si inválidos: devolver 400 con ErrorResponse JSON.
  3. Renderizar HTML con `html/template` — formulario con campos
     `email`, `password` y campos hidden: `state`, `code_challenge`,
     `redirect_uri`, `client_id`.

  `PostAuthorize` handler:
  1. Parsear formulario (`c.Request.ParseForm()`).
  2. Llamar `LoginUser.Execute()` con email + password.
  3. Si falla → renderizar formulario de nuevo con mensaje de error.
  4. Generar código: `crypto/rand` 32 bytes → base64url.
  5. Calcular `code_hash = SHA-256(code_plaintext)`.
  6. Almacenar en `CodeStore` con TTL 60s.
  7. Redirigir a `redirect_uri?code=<code>&state=<state>`.

  `PostToken` handler:
  1. Parsear body JSON `TokenRequest`.
  2. Validar `grant_type=authorization_code`.
  3. Calcular hash del `code` recibido.
  4. `CodeStore.Consume(hash)` — si no existe → 400 INVALID_CODE.
  5. Verificar: `base64url(SHA-256(code_verifier)) == stored.CodeChallenge`.
     Si falla → 400 INVALID_CODE_VERIFIER.
  6. Verificar: `redirect_uri` y `client_id` coinciden con los almacenados.
  7. Cargar usuario por ID, llamar `TokenIssuer.Issue()`.
  8. Devolver `TokenResponse`.

  Añadir las rutas al router del auth-server:
  ```go
  r.GET("/authorize", oauthHandler.GetAuthorize)
  r.POST("/authorize", oauthHandler.PostAuthorize)
  r.POST("/token", oauthHandler.PostToken)
  ```

- **Criterio de aceptación:** El flujo PKCE completo funciona manualmente:
  navegar a `/authorize?...` → ver formulario → submit → recibir redirect
  con code → `POST /token` → recibir JWT válido.
  `go test ./services/auth-server/...` pasa.
- **Depende de:** #1 (spec), SPRINT-004 completado
- **Commit:** `feat(auth): implement OAuth 2.1 PKCE authorize and token endpoints [SPRINT-005 #2]`

### 3. [infra] Dashboard: package.json, Vite, TypeScript, estructura de directorios

- **Agente:** @developer
- **Descripción:** Crear la configuración base del dashboard en el
  directorio `dashboard/`.

  **`dashboard/package.json`** — dependencias:
  ```json
  {
    "dependencies": {
      "react": "^19.0.0",
      "react-dom": "^19.0.0",
      "react-router-dom": "^7.0.0",
      "@tanstack/react-query": "^5.0.0",
      "openapi-fetch": "^0.13.0",
      "class-variance-authority": "^0.7.0",
      "clsx": "^2.0.0",
      "tailwind-merge": "^2.0.0",
      "lucide-react": "^0.500.0",
      "next-themes": "^0.4.0",
      "sonner": "^1.0.0"
    },
    "devDependencies": {
      "@types/react": "^19.0.0",
      "@types/react-dom": "^19.0.0",
      "@vitejs/plugin-react": "^4.0.0",
      "vite": "^6.0.0",
      "typescript": "^5.7.0",
      "tailwindcss": "^4.0.0",
      "@tailwindcss/vite": "^4.0.0",
      "openapi-typescript": "^7.0.0",
      "vitest": "^2.0.0",
      "@vitest/coverage-v8": "^2.0.0",
      "@testing-library/react": "^16.0.0",
      "@testing-library/user-event": "^14.0.0",
      "msw": "^2.0.0",
      "jsdom": "^25.0.0"
    }
  }
  ```

  **`dashboard/vite.config.ts`:**
  - `@vitejs/plugin-react`
  - `@tailwindcss/vite`
  - Path alias: `@` → `src/`
  - Variables de entorno con `envPrefix: 'VITE_'`

  **`dashboard/tsconfig.json`:**
  - `strict: true`, `noImplicitAny: true`, `noUnusedLocals: true`
  - `moduleResolution: "bundler"`, `jsx: "react-jsx"`
  - Path alias `@/*` → `src/*`

  **`dashboard/.env.development`** con las cuatro variables de entorno.

  **Estructura de directorios** con `index.ts` vacíos:
  ```
  dashboard/src/
  ├── api/          (types.gen.ts vendrá del TODO #5)
  ├── auth/
  ├── components/ui/
  ├── components/composed/
  ├── components/shared/
  ├── features/graphs/
  ├── hooks/
  ├── layouts/
  ├── lib/
  ├── pages/
  ├── styles/
  └── test/
  ```

- **Criterio de aceptación:** `npm install` en `dashboard/` termina sin
  errores. `npm run build` (aunque falle por falta de fuentes) no falla
  por configuración de Vite/TypeScript/Tailwind.
- **Depende de:** ninguno (paralelo a #1, #2)
- **Commit:** `chore(dashboard): scaffold Vite + React 19 + TypeScript + Tailwind base [SPRINT-005 #3]`

### 4. [infra] Tailwind CSS + shadcn/ui + dark mode + globals.css

- **Agente:** @developer
- **Descripción:** Configurar el design system completo.

  **`dashboard/tailwind.config.ts`:** Dark mode con `darkMode: ["class"]`,
  content paths cubriendo `src/**/*.{tsx,ts}`.

  **`dashboard/src/styles/globals.css`:** CSS variables HSL para light
  y dark mode exactamente como la paleta por defecto de shadcn/ui.
  Usar `@layer base` para aplicar las variables.

  **`dashboard/components.json`** (shadcn/ui config):
  ```json
  {
    "style": "default",
    "rsc": false,
    "tsx": true,
    "tailwind": { "config": "tailwind.config.ts", "css": "src/styles/globals.css", "baseColor": "slate", "cssVariables": true },
    "aliases": { "components": "@/components", "utils": "@/lib/utils" }
  }
  ```

  **`dashboard/src/lib/utils.ts`:** función `cn(...inputs)` con
  `clsx` + `tailwind-merge`.

  **Añadir componentes shadcn/ui** (con `npx shadcn-ui@latest add`
  o manualmente): `button`, `input`, `label`, `card`, `table`,
  `badge`, `skeleton`, `separator`, `dropdown-menu`, `avatar`,
  `toast` (vía sonner).

  **`dashboard/src/components/shared/theme-provider.tsx`:** wrapper de
  `next-themes` que aplica `attribute="class"` y `defaultTheme="system"`.

- **Criterio de aceptación:** `npm run build` compila sin errores de
  Tailwind. Los componentes shadcn/ui existen en `src/components/ui/`.
  Dark mode funciona en development (toggle manual).
- **Depende de:** #3
- **Commit:** `feat(dashboard): add shadcn/ui design system and dark mode [SPRINT-005 #4]`

### 5. [infra] Generación de tipos TypeScript desde OpenAPI

- **Agente:** @developer
- **Descripción:** Configurar la generación automática de tipos TypeScript
  desde `specs/openapi.yaml` usando `openapi-typescript`.

  **`dashboard/package.json`** script:
  ```json
  "scripts": {
    "gen:api": "openapi-typescript ../specs/openapi.yaml -o src/api/types.gen.ts"
  }
  ```

  **`Makefile`** target (en la raíz del monorepo):
  ```makefile
  gen-api-types: ## Genera tipos TypeScript desde specs/openapi.yaml
      cd dashboard && npm run gen:api
  ```

  **`dashboard/src/api/client.ts`:**
  - `createApiClient(getToken: () => string | null)` usando
    `openapi-fetch` y los tipos generados de `types.gen.ts`.
  - Middleware de inyección del Bearer token.
  - Exportar también `ApiClient` type para usar en los hooks.

  **Verificar** que los tipos generados incluyen `GraphResponse`,
  `GraphListResponse`, `ExecutionResponse`, `TokenResponse`, etc.

  El archivo `types.gen.ts` se incluye en el repositorio (no es
  `.gitignore`) porque es el contrato entre frontend y backend. Se
  regenera con `make gen-api-types` tras cada cambio a la spec.

- **Criterio de aceptación:** `make gen-api-types` genera
  `dashboard/src/api/types.gen.ts` sin errores. El archivo contiene
  los tipos de graphs, executions y auth. `npm run build` compila
  correctamente con los tipos generados.
- **Depende de:** #3, specs de SPRINT-003 y SPRINT-004 completadas
- **Commit:** `feat(dashboard): add OpenAPI TypeScript type generation [SPRINT-005 #5]`

### 6. [test] Configurar Vitest + React Testing Library + MSW

- **Agente:** @qa
- **Descripción:** Configurar el entorno de tests del dashboard.

  **`dashboard/vitest.config.ts`:**
  - Entorno `jsdom`, setup file `src/test/setup.ts`.
  - Coverage con `@vitest/coverage-v8`.
  - Alias `@` igual que en Vite.

  **`dashboard/src/test/setup.ts`:**
  - `import '@testing-library/jest-dom'` (matchers custom).
  - Mock de `window.crypto.subtle` si no disponible en jsdom.
  - MSW server setup con `beforeAll/afterAll/afterEach`.

  **`dashboard/src/test/server.ts`:**
  - MSW `setupServer` con handlers vacíos (se añaden en cada test).

  **`dashboard/src/test/handlers.ts`:**
  - Handler MSW para `GET /api/v1/graphs` → respuesta paginada vacía.
  - Handler MSW para `POST /token` → respuesta JWT mock.

  `npm run test` ejecuta todos los tests. `npm run test:coverage`
  genera reporte de cobertura.

- **Criterio de aceptación:** `npm test` en `dashboard/` pasa con 0
  tests fallando (aún no hay tests sustanciales). El entorno carga
  sin errores.
- **Depende de:** #3, #4
- **Commit:** `test(dashboard): configure Vitest + RTL + MSW test environment [SPRINT-005 #6]`

### 7. [test] Tests del módulo PKCE (Red)

- **Agente:** @qa
- **Descripción:** Tests para las funciones criptográficas PKCE antes
  de implementarlas.

  ```typescript
  // dashboard/src/auth/__tests__/pkce.test.ts

  // testGenerateVerifierLength: el verifier tiene entre 43 y 128 caracteres
  // (base64url de 32 bytes = 43 chars).
  test("generateVerifier produces valid base64url string", ...)

  // testVerifierIsUnique: dos llamadas producen valores distintos.
  test("generateVerifier produces unique values", ...)

  // testChallengeIsDeterministic: el mismo verifier siempre produce el mismo challenge.
  test("generateChallenge is deterministic for a given verifier", ...)

  // testChallengeIsBase64urlOfSHA256: el challenge es el SHA-256 del verifier
  // encodificado en base64url (RFC 7636 §4.2).
  test("generateChallenge produces correct S256 challenge", ...)

  // testBase64urlNoPadding: el challenge no contiene '=', '+', '/'.
  test("generateChallenge uses base64url alphabet without padding", ...)
  ```

- **Criterio de aceptación:** Tests en RED antes del TODO #9. GREEN
  tras implementar `pkce.ts`.
- **Depende de:** #6
- **Commit:** `test(dashboard): add PKCE crypto unit tests [SPRINT-005 #7]`

### 8. [test] Tests del AuthProvider (Red)

- **Agente:** @qa
- **Descripción:** Tests del estado de autenticación y del flujo PKCE
  en el frontend.

  ```typescript
  // dashboard/src/auth/__tests__/AuthProvider.test.tsx

  // testInitialStateUnauthenticated: estado inicial es
  // { isAuthenticated: false, token: null, isLoading: false }.
  test("AuthProvider initial state is unauthenticated", ...)

  // testLoginSetsToken: tras llamar a una función setToken interna,
  // isAuthenticated es true.
  test("setToken makes isAuthenticated true", ...)

  // testLogoutClearsToken: tras logout, token es null.
  test("logout clears token and user", ...)

  // testProtectedRouteRedirects: ProtectedRoute redirige al flujo PKCE
  // si no hay token (simular con MSW + React Router MemoryRouter).
  test("ProtectedRoute redirects to PKCE authorize if no token", ...)

  // testAuthCallbackExchangesCode: AuthCallback llama POST /token con
  // el code de la URL y almacena el token resultante.
  test("AuthCallback exchanges code for token via POST /token", ...)
  ```

- **Criterio de aceptación:** Tests en RED antes del TODO #10. GREEN
  tras implementar el módulo de auth.
- **Depende de:** #6, #7
- **Commit:** `test(dashboard): add AuthProvider and PKCE flow unit tests [SPRINT-005 #8]`

### 9. [impl] Implementar dashboard/src/auth/ (pkce.ts, AuthProvider, useAuth, ProtectedRoute, AuthCallback)

- **Agente:** @developer
- **Descripción:** Implementar el módulo de autenticación completo.

  **`pkce.ts`** — funciones puras con Web Crypto API:
  ```typescript
  export async function generateVerifier(): Promise<string> {
    const bytes = crypto.getRandomValues(new Uint8Array(32));
    return base64urlEncode(bytes);
  }

  export async function generateChallenge(verifier: string): Promise<string> {
    const bytes = new TextEncoder().encode(verifier);
    const hash = await crypto.subtle.digest("SHA-256", bytes);
    return base64urlEncode(new Uint8Array(hash));
  }

  function base64urlEncode(bytes: Uint8Array): string {
    return btoa(String.fromCharCode(...bytes))
      .replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
  }
  ```

  **`AuthProvider.tsx`** — estado en `useState` + token raw en `useRef`:
  - `startPKCE()`: genera verifier+challenge, guarda en `sessionStorage`,
    redirige a `VITE_AUTH_URL/authorize?...`.
  - `handleCallback(code)`: llama `POST VITE_AUTH_URL/token`, guarda
    token en `useRef`, actualiza estado `isAuthenticated=true`, borra
    `sessionStorage`.
  - `logout()`: limpia `useRef` + estado.

  **`ProtectedRoute.tsx`:**
  ```tsx
  export function ProtectedRoute({ children }: { children: ReactNode }) {
    const { isAuthenticated, isLoading } = useAuth();
    if (isLoading) return <AppSkeleton />;
    if (!isAuthenticated) {
      startPKCE(); // asíncrono, redirige
      return null;
    }
    return <>{children}</>;
  }
  ```

  **`AuthCallback.tsx`** — página en `/auth/callback`:
  1. Extraer `code` y `state` de `useSearchParams`.
  2. Verificar `state` contra `sessionStorage`.
  3. Llamar `handleCallback(code)`.
  4. Redirigir a la ruta original (guardada antes del PKCE).

- **Criterio de aceptación:** Tests del TODO #7 y #8 pasan a GREEN.
  `npm run build` compila sin errores de TypeScript.
- **Depende de:** #7, #8
- **Commit:** `feat(dashboard): implement PKCE auth module and AuthProvider [SPRINT-005 #9]`

### 10. [test] Tests de GraphsPage con MSW (Red)

- **Agente:** @qa
- **Descripción:** Tests de la página de listado de grafos, simulando
  la API con MSW.

  ```typescript
  // dashboard/src/pages/__tests__/GraphsPage.test.tsx

  // testShowsSkeletonWhileLoading: mientras TanStack Query está en estado
  // loading, se muestran skeletons de la tabla.
  test("GraphsPage shows skeletons while loading", ...)

  // testRendersGraphList: con MSW devolviendo 2 grafos, la tabla muestra
  // 2 filas con nombre, versión y badge de status.
  test("GraphsPage renders graph list from API", ...)

  // testShowsEmptyState: con MSW devolviendo lista vacía, muestra mensaje
  // "No hay grafos" (o similar).
  test("GraphsPage shows empty state when no graphs", ...)

  // testShowsErrorState: con MSW devolviendo 500, muestra mensaje de error.
  test("GraphsPage shows error state on API failure", ...)

  // testPaginationControls: con total_pages > 1, aparecen controles de
  // paginación y el cambio de página dispara nueva query.
  test("GraphsPage pagination changes current page", ...)
  ```

- **Criterio de aceptación:** Tests en RED antes del TODO #11. GREEN
  tras implementar GraphsPage.
- **Depende de:** #5, #6
- **Commit:** `test(dashboard): add GraphsPage unit tests with MSW [SPRINT-005 #10]`

### 11. [impl] Implementar GraphsPage, AppLayout, App.tsx, main.tsx

- **Agente:** @developer
- **Descripción:** Implementar las páginas y el routing completo.

  **`dashboard/src/hooks/useGraphs.ts`:**
  ```typescript
  export function useGraphs(opts: { page: number; perPage: number; status?: string }) {
    const { apiClient } = useAuth();
    return useQuery({
      queryKey: ["graphs", opts],
      queryFn: async () => {
        const { data, error } = await apiClient.GET("/api/v1/graphs", {
          params: { query: { page: opts.page, per_page: opts.perPage, status: opts.status } },
        });
        if (error) throw error;
        return data;
      },
    });
  }
  ```

  **`dashboard/src/pages/GraphsPage.tsx`:**
  - Tabla con columnas: nombre, versión, status (Badge), fecha creación.
  - Skeletons con `Skeleton` de shadcn/ui mientras `isLoading`.
  - Estado vacío y estado de error con mensajes adecuados.
  - Paginación básica con botones "Anterior" / "Siguiente".
  - Dark mode funcional.

  **`dashboard/src/layouts/AppLayout.tsx`:**
  - Sidebar fijo con navegación: Grafos, Ejecuciones (deshabilitado),
    Catálogo (deshabilitado).
  - Header: título, toggle dark/light mode, avatar de usuario.
  - Responsive: sidebar colapsable en móvil.

  **`dashboard/src/App.tsx`:**
  ```tsx
  export function App() {
    return (
      <Routes>
        <Route path="/auth/callback" element={<AuthCallback />} />
        <Route element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
          <Route path="/" element={<Navigate to="/graphs" replace />} />
          <Route path="/graphs" element={<GraphsPage />} />
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    );
  }
  ```

  **`dashboard/src/main.tsx`:**
  - `ThemeProvider` (dark mode) → `AuthProvider` → `QueryClientProvider`
    → `BrowserRouter` → `App`.

- **Criterio de aceptación:** Tests del TODO #10 pasan a GREEN.
  `npm run dev` arranca el dashboard en `localhost:5173`. Con auth-server
  y orchestrator levantados, el flujo completo funciona: navegar →
  login PKCE → ver tabla de grafos.
- **Depende de:** #9, #10
- **Commit:** `feat(dashboard): implement GraphsPage, AppLayout, routing and providers [SPRINT-005 #11]`

### 12. [test] Test de smoke E2E del dashboard

- **Agente:** @qa
- **Descripción:** Test E2E manual documentado + script de smoke
  automatizable. No se usa Playwright/Cypress en este sprint (se
  añaden en SPRINT-dashboard-e2e). En su lugar, un script shell
  `dashboard/scripts/smoke.sh` que:

  1. Verifica que `npm run build` compila sin errores.
  2. Verifica que `npm run type-check` pasa sin errores TypeScript.
  3. Verifica que `npm run lint` (ESLint) no reporta errores.
  4. Verifica que `npm test` pasa todos los tests unitarios.

  Añadir scripts en `package.json`:
  ```json
  "type-check": "tsc --noEmit",
  "lint": "eslint src --ext .ts,.tsx",
  "build": "vite build",
  "test": "vitest run",
  "test:watch": "vitest"
  ```

  Añadir `make dashboard-check` en el Makefile raíz que llama al
  script de smoke.

- **Criterio de aceptación:** `make dashboard-check` pasa limpio.
  El pipeline `make ci` puede incluir `make dashboard-check`.
- **Depende de:** #11
- **Commit:** `test(dashboard): add build and type-check smoke scripts [SPRINT-005 #12]`

### 13. [docs] Actualizar docs/index.md y docs/log.md

- **Agente:** @docs
- **Descripción:** Añadir SPRINT-005 a la tabla de sprints. Actualizar
  la sección "Servicios" marcando el dashboard como en progreso.
  Documentar las variables de entorno nuevas en una tabla.
  Actualizar `docs/log.md` con entrada del sprint.
- **Criterio de aceptación:** `docs/index.md` y `docs/log.md`
  actualizados.
- **Depende de:** #12
- **Commit:** `docs(dashboard): update index with SPRINT-005 results [SPRINT-005 #13]`

## Matriz de trazabilidad

| Spec / ADR | Regla | TODO | Artefacto | Verificado por |
|------------|-------|------|-----------|----------------|
| ADR-009 regla 2 | TypeScript strict: true, sin any | #3 | `tsconfig.json` | `npm run type-check` |
| ADR-009 regla 4 | TanStack Query, no useEffect para HTTP | #11 | `useGraphs.ts` | code review |
| ADR-009 regla 5 | API client generado desde OpenAPI | #5 | `types.gen.ts` + `client.ts` | `make gen-api-types` |
| ADR-009 regla 10 | Tokens en memoria, nunca localStorage | #9 | `AuthProvider` con `useRef` | test `testLoginSetsToken` |
| ADR-012 regla 1 | PKCE para usuarios SPA | #1, #2, #9 | PKCE endpoints + módulo | `testAuthCallbackExchangesCode` |
| ADR-012 regla 5 | Refresh tokens one-time (excluido del sprint) | — | nota en doc | revisión en SPRINT-ABAC |
| ADR-019 regla 1 | shadcn/ui como base | #4 | `components/ui/` | build |
| ADR-019 regla 2 | Tailwind CSS, no CSS-in-JS | #4 | `globals.css` sin styled-components | code review |
| ADR-019 regla 4 | Dark mode obligatorio | #4 | CSS variables `.dark {}` | test visual |
| ADR-019 regla 5 | Accesibilidad (Radix UI) | #4 | componentes shadcn/ui | Radix base |
| ADR-010 regla 9 | Tipos TypeScript generados desde OpenAPI | #5 | `types.gen.ts` | `npm run type-check` |
| ADR-012 regla 3 | JWT RS256 con attrs ABAC | #2 | `PostToken` devuelve JWT SPRINT-004 | `testAuthCallbackExchangesCode` |
| PKCE RFC 7636 | S256: SHA-256(verifier) == challenge | #7, #9 | `pkce.ts` | `testChallengeIsBase64urlOfSHA256` |

## Criterios de aceptación del sprint

```bash
# 1. Spec válida
swagger-cli validate specs/openapi.yaml

# 2. Backend compila
go build ./...

# 3. Backend tests pasan
make ci

# 4. PKCE endpoints funcionales (manual)
curl -v "http://localhost:8081/authorize?response_type=code&client_id=dago-dashboard&redirect_uri=http://localhost:5173/auth/callback&code_challenge=<challenge>&code_challenge_method=S256&state=xyz"
# → 200 HTML con formulario

# 5. Dashboard compila sin errores TypeScript
cd dashboard && npm run type-check

# 6. Dashboard tests pasan
cd dashboard && npm test

# 7. Tipos generados desde OpenAPI
make gen-api-types
# → dashboard/src/api/types.gen.ts actualizado

# 8. Smoke scripts pasan
make dashboard-check

# 9. Flujo E2E (manual, con todos los servicios levantados)
make docker-up
go run ./services/auth-server/cmd/main.go &   # puerto 8081
go run ./services/orchestrator/cmd/main.go &  # puerto 8080
cd dashboard && npm run dev                   # puerto 5173
# → navegar a localhost:5173 → redirige a login → tras login → tabla de grafos
```

Adicionalmente:
- `AuthProvider` almacena token en `useRef`, nunca en `localStorage`.
- `sessionStorage` solo se usa temporalmente para `code_verifier` y
  `state` durante el redirect (se limpia en `AuthCallback`).
- `types.gen.ts` en `.gitignore` NO — se commitea como contrato.
- El componente `ProtectedRoute` no renderiza nada mientras redirige.
- Los tests de MSW no hacen llamadas HTTP reales.
- `npm run build` produce un bundle sin errores en menos de 60 segundos.

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
