# SPRINT-004: Auth-server básico — login local, JWT y middleware de validación

## Metadata

- **Fecha inicio:** 2026-04-29 (paralelo a SPRINT-002 y SPRINT-003 — solo depende de SPRINT-001)
- **Fecha fin estimada:** 2026-04-30
- **Estado:** planificado
- **ADRs aplicados:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-006, ADR-007, ADR-012
- **Specs afectadas:**
  - `specs/paths/auth.yaml` — nuevo (register, login, JWKS)
  - `specs/schemas/auth.yaml` — nuevo (LoginInput, TokenResponse, RegisterInput)
  - `specs/openapi.yaml` — añadir `$ref` a los nuevos paths de auth
  - `ent/schema/user.go` — nuevo
  - `ent/schema/org_unit.go` — nuevo
- **Agente planificador:** planner
- **Revisado por:** pendiente
- **Depende de:** SPRINT-001 (go.mod, docker-compose PostgreSQL, atlas.hcl)
- **Paralelo a:** SPRINT-002, SPRINT-003 (independiente de Graph/Node/Execution)
- **Bloquea:** SPRINT-005 (rutas protegidas en orchestrator), SPRINT-ABAC (autorización)

## Objetivo del sprint

Implementar el núcleo de autenticación de dago: login local con contraseñas
hasheadas en argon2id, emisión de JWT RS256 con claims ABAC, endpoint JWKS
para validación local, y el middleware de validación para el orchestrator.

Al finalizar: el auth-server acepta credenciales, emite tokens JWT firmados y
sirve su clave pública vía JWKS. El orchestrator tiene un middleware funcional
que valida esos tokens, con modo bypass para desarrollo.

## Alcance

### Incluido

**Nuevos schemas Ent:**
- `ent/schema/user.go` — User (id, email, password_hash, tags, org_unit FK).
- `ent/schema/org_unit.go` — OrgUnit (id, name, path, tags, parent/children self-ref).
- `go generate ./ent` — cliente Ent regenerado con los dos nuevos schemas.
- `atlas migrate diff add_user_org_unit` — migración SQL para las dos tablas.
- `atlas migrate apply --env local` — aplicada contra PostgreSQL de docker-compose.

**Spec OpenAPI:**
- `specs/paths/auth.yaml` — endpoints: POST /auth/register, POST /auth/login,
  GET /.well-known/jwks.json.
- `specs/schemas/auth.yaml` — RegisterInput, LoginInput, TokenResponse.
- `specs/openapi.yaml` actualizado con los `$ref`.

**Dominio (`libs/domain/`):**
- `libs/domain/user.go` — tipos `User`, `Credentials`.
- `libs/domain/token.go` — tipos `Claims`, `TokenPair`, constantes de scopes.

**Puertos (`libs/ports/auth.go`):**
- `PasswordHasher` — `Hash(password string) (string, error)` y `Verify`.
- `TokenIssuer` — `Issue(ctx, *domain.User) (string, error)`.
- `TokenValidator` — `Validate(ctx, token string) (*domain.Claims, error)`.
- `UserRepository` — `Create` y `FindByEmail`.

**Adaptadores (`adapters/auth/`):**
- `adapters/auth/argon2id.go` — implementa `PasswordHasher` con argon2id.
- `adapters/auth/jwt_issuer.go` — implementa `TokenIssuer` con RS256.
- `adapters/auth/jwks_validator.go` — implementa `TokenValidator` con JWKS HTTP.
- `adapters/auth/ent_user_repo.go` — implementa `UserRepository` con Ent.

**auth-server (`services/auth-server/`):**
- `internal/usecase/register.go` — `RegisterUser` use case.
- `internal/usecase/login.go` — `LoginUser` use case.
- `internal/handler/auth.go` — handlers para register, login, JWKS.
- `internal/router/router.go` — `NewRouter()` con middlewares y rutas.
- `cmd/main.go` — wiring completo: RS256 keypair, Ent, repositorio, casos de
  uso, handlers, servidor en puerto 8081.

**Middleware para orchestrator:**
- `services/orchestrator/internal/middleware/auth.go` — valida Bearer JWT,
  inyecta `*domain.Claims` en `gin.Context`, bypass cuando `AUTH_REQUIRED=false`.

**Tests:**
- `tests/unit/auth/argon2id_test.go` — hash + verify, timing-safe.
- `tests/unit/auth/jwt_test.go` — issue + validate, expiración, claims ABAC.
- `tests/contract/auth_contract_test.go` — respuestas contra spec OpenAPI.
- `tests/unit/handler/auth_handler_test.go` — handlers con fakes.
- `tests/unit/middleware/auth_middleware_test.go` — middleware orchestrator.
- `tests/integration/auth_integration_test.go` — login real contra PostgreSQL.

### Excluido

- Authorization Code + PKCE (necesita dashboard — SPRINT-frontend-001).
- Client Credentials M2M (SPRINT-ABAC).
- Refresh tokens / rotación (SPRINT-ABAC).
- `POST /token` endpoint OAuth 2.1 estándar (SPRINT-ABAC).
- `GET /authorize` y `POST /revoke` (SPRINT-ABAC).
- Motor ABAC completo (`Authorizer`, herencia de tags por árbol UO)
  — SPRINT-ABAC. Este sprint solo emite tags del usuario directamente.
- Identity Broker (IdPs externos: Google, Azure AD) — SPRINT-ABAC.
- Aplicar el middleware en las rutas del orchestrator (SPRINT-005).
  Este sprint implementa el middleware y verifica que funciona, pero no
  lo activa en las rutas de SPRINT-003.
- `DELETE /api/v1/auth/users/:id` ni gestión de usuarios (SPRINT-ABAC).

## Dependencias

- **SPRINT-001 completado:** `go.mod` con `golang.org/x/crypto` y
  `github.com/golang-jwt/jwt/v5`, estructura `services/auth-server/`,
  docker-compose PostgreSQL, atlas.hcl.
- **Si SPRINT-002 ya corrió:** la migración de este sprint solo añade
  `users` y `org_units`. Atlas calcula el diff contra el estado actual
  de la BD — no interfiere con las tablas existentes.
- **Si SPRINT-002 no corrió aún:** la migración incluye las tres tablas
  de SPRINT-002 más las dos de este sprint (Atlas es aditivo).

## Contratos de comportamiento

### C1 — `POST /api/v1/auth/login` — credenciales válidas

```
Given: Usuario registrado con email "user@example.com" y password "securepass123"
When: POST /api/v1/auth/login con body {"email":"user@example.com","password":"securepass123"}
Then: HTTP 200, TokenResponse con access_token no vacío, token_type="Bearer", expires_in=3600
      El access_token es un JWT RS256 con claims sub, iss, aud, scope, attrs
```

### C2 — `POST /api/v1/auth/login` — credenciales incorrectas (timing-safe)

```
Given: Usuario registrado con password "securepass123"
When: POST /api/v1/auth/login con password incorrecto "wrongpass"
Then: HTTP 401, ErrorResponse con code = "INVALID_CREDENTIALS"
      El mensaje de error NO revela si el email existe o no (same error)
      El tiempo de respuesta es comparable al de autenticación exitosa (timing-safe argon2id)
```

### C3 — `GET /.well-known/jwks.json`

```
Given: auth-server arrancado con keypair RSA configurado
When: GET /.well-known/jwks.json sin Authorization header
Then: HTTP 200, JSON con campo "keys" como array
      keys[0].kty = "RSA", keys[0].alg = "RS256"
      El endpoint es público (no requiere autenticación)
```

### C4 — Middleware JWT en orchestrator — modo bypass

```
Given: AUTH_REQUIRED=false en variables de entorno del orchestrator
When: Cualquier request al orchestrator sin header Authorization
Then: El middleware llama c.Next() y el handler se ejecuta normalmente
      HTTP 200 (o el status propio del handler)
      Los tests de SPRINT-003 siguen pasando sin cambios
```

## Diseño

### Schemas Ent

**User:**

| Campo | Tipo Ent | Tipo PostgreSQL | Notas |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | `uuid.New()` |
| `email` | `string` | `VARCHAR(320) UNIQUE NOT NULL` | validado con regex RFC 5321 |
| `password_hash` | `string` | `TEXT NOT NULL` | hash argon2id, nunca en respuestas API |
| `tags` | `[]string` | `TEXT[] NOT NULL DEFAULT '{}'` | tags ABAC del usuario |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | inmutable |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | auto |

Edges: `M2O` con `OrgUnit` (opcional).

**OrgUnit:**

| Campo | Tipo Ent | Tipo PostgreSQL | Notas |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | `uuid.New()` |
| `name` | `string` | `VARCHAR(255) NOT NULL` | nombre corto, ej. `"engineering"` |
| `path` | `string` | `TEXT UNIQUE NOT NULL` | path completo, ej. `"/company/engineering"` |
| `tags` | `[]string` | `TEXT[] NOT NULL DEFAULT '{}'` | tags heredados a usuarios |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | inmutable |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | auto |

Edges: `O2M` a `children` (OrgUnit), `M2O` a `parent` (OrgUnit, opcional),
`O2M` a `users`.

### Formato del JWT (RS256)

```json
{
  "iss": "<JWT_ISSUER env>",
  "sub": "<user UUID>",
  "aud": ["dago-api"],
  "exp": <now + 3600>,
  "iat": <now>,
  "scope": "graphs:read graphs:execute graphs:manage",
  "client_type": "user",
  "attrs": {
    "tags": ["<user_tags> ∪ <org_unit_tags>"],
    "org_unit": "<org_unit_id>",
    "org_path": "<org_unit_path>"
  }
}
```

Para este sprint, `attrs.tags` = tags directas del User (sin herencia
de árbol UO — esa lógica se completa en SPRINT-ABAC).

### argon2id — parámetros OWASP 2023

```go
const (
    argon2Memory      = 64 * 1024  // 64 MB
    argon2Iterations  = 3
    argon2Parallelism = 4
    argon2SaltLen     = 16
    argon2KeyLen      = 32
)
```

Hash almacenado: `$argon2id$v=19$m=65536,t=3,p=4$<salt_b64>$<hash_b64>`
(formato estándar PHC string — parseable por `golang.org/x/crypto/argon2`).

### RS256 keypair

- Cargado desde variables de entorno `JWT_PRIVATE_KEY_PATH` (PEM file)
  y `JWT_PUBLIC_KEY_PATH` (PEM file).
- Si no existen, se generan en memoria al arranque (solo para desarrollo).
  En producción las claves vienen de un secret manager.
- El JWKS endpoint expone únicamente la clave pública en formato JWK
  con `kid` (key ID) derivado del thumbprint SHA-256.

### Endpoints del auth-server

| Método | Path | Body | Response |
|--------|------|------|----------|
| `POST` | `/api/v1/auth/register` | `RegisterInput` | 201 `UserResponse` / 409 |
| `POST` | `/api/v1/auth/login` | `LoginInput` | 200 `TokenResponse` / 401 |
| `GET` | `/.well-known/jwks.json` | — | 200 JWKS JSON |

`RegisterInput`: `{"email": "...", "password": "..."}` (password ≥ 12 chars).
`LoginInput`: `{"email": "...", "password": "..."}`.
`TokenResponse`: `{"access_token": "...", "token_type": "Bearer", "expires_in": 3600}`.
`UserResponse`: `{"id": "...", "email": "...", "tags": [], "created_at": "..."}`.
— `password_hash` **nunca** aparece en la respuesta (ADR-012 regla 10).

### Middleware del orchestrator

```
Authorization: Bearer <jwt>
        │
        ▼
middleware/auth.go
        │
        ├── AUTH_REQUIRED=false → c.Next() (bypass desarrollo)
        │
        └── AUTH_REQUIRED=true
                │
                ├── No header → 401 MISSING_TOKEN
                ├── JWT inválido/expirado → 401 INVALID_TOKEN
                └── JWT válido → inyectar *Claims en ctx → c.Next()
```

Claims accesibles desde handlers: `c.MustGet("claims").(*domain.Claims)`.

Variables de entorno del middleware:
- `AUTH_REQUIRED` — `false` (dev) / `true` (prod).
- `JWKS_URL` — URL del endpoint JWKS del auth-server.
- `JWT_ISSUER` — valor esperado del campo `iss` del JWT.
- `JWT_AUDIENCE` — valor esperado del campo `aud` (default `"dago-api"`).

## TODOs

### 1. [spec] Escribir specs/schemas/auth.yaml y specs/paths/auth.yaml

- **Agente:** @developer
- **Descripción:** Crear los schemas OpenAPI `RegisterInput`,
  `LoginInput`, `TokenResponse`, `UserResponse`. Crear el path item
  para los tres endpoints de auth (register, login, JWKS).
  Actualizar `specs/openapi.yaml` con los `$ref` correspondientes.

  Reglas ADR-010:
  - Register y login bajo `/api/v1/auth/` (prefijo `/api/v1/`).
  - JWKS en `/.well-known/jwks.json` (excepción OAuth estándar —
    ADR-006 regla 4 nota).
  - `UserResponse` no incluye `password_hash` ni ningún campo de
    credencial.
  - Todos los errores usan `ErrorResponse` (code + message).

  Status codes:
  - `POST /auth/register`: 201, 400, 409 (email duplicado), 422.
  - `POST /auth/login`: 200, 400, 401 (credenciales incorrectas), 422.
  - `GET /.well-known/jwks.json`: 200 (siempre — sin auth requerida).

- **Criterio de aceptación:** `swagger-cli validate specs/openapi.yaml`
  pasa. `UserResponse` no tiene campo `password_hash`.
- **Depende de:** ninguno
- **Commit:** `spec(openapi): add auth register, login, and JWKS endpoints [SPRINT-004 #1]`

### 2. [data] Implementar ent/schema/user.go y ent/schema/org_unit.go

- **Agente:** @developer
- **Descripción:** Crear los dos schemas Ent según el modelo de datos
  de este documento.

  Validaciones en `user.go`:
  - `email`: `field.String("email").MaxLen(320).Match(emailRegex).Unique()`.
  - `password_hash`: `field.String("password_hash").Sensitive()` — el tag
    `.Sensitive()` excluye el campo de los logs de Ent.
  - `tags`: `field.Strings("tags").Default([]string{})`.

  Validaciones en `org_unit.go`:
  - `path`: empieza con `/`, no termina con `/` (excepto `/`).
  - `path`: `field.String("path").MaxLen(1024).Unique()`.
  - Self-referential edge correctamente tipado (Ent requiere un
    nombre de ref diferente para padre e hijos).

- **Criterio de aceptación:** `go build ./ent/...` compila. El campo
  `password_hash` tiene `.Sensitive()`.
- **Depende de:** ninguno (paralelo a #1)
- **Commit:** `feat(schema): add User and OrgUnit Ent schemas [SPRINT-004 #2]`

### 3. [data] Ejecutar go generate y migración Atlas

- **Agente:** @developer
- **Descripción:** Con `make docker-up` activo:

  ```bash
  # Regenerar cliente Ent (incluye User y OrgUnit además de schemas previos)
  go generate ./ent

  # Generar migración para las nuevas tablas
  atlas migrate diff add_user_org_unit --env local

  # Aplicar migración
  atlas migrate apply --env local

  # Verificar las tablas creadas
  docker compose exec postgres psql -U dago -d dago \
      -c "\d users" -c "\d org_units"
  ```

  Verificar que el SQL generado incluye:
  - `CREATE TABLE users` con `email UNIQUE`, `password_hash TEXT`,
    `tags TEXT[]`.
  - `CREATE TABLE org_units` con `path TEXT UNIQUE`, `tags TEXT[]`,
    FK self-referencial opcional.
  - FK `users.org_unit_id → org_units.id` (nullable).

- **Criterio de aceptación:** `atlas migrate lint --env local` sin
  errores. Las tablas `users` y `org_units` existen en PostgreSQL.
  `go build ./...` compila con los nuevos tipos generados.
- **Depende de:** #2
- **Commit:** `feat(schema): generate Ent client with User and OrgUnit; add migration [SPRINT-004 #3]`

### 4. [domain] Tipos de dominio y puertos de auth

- **Agente:** @developer
- **Descripción:** Crear los tipos puros de dominio y los puertos
  de auth. Sin imports de Ent, Gin ni crypto.

  **`libs/domain/user.go`:**
  ```go
  type User struct {
      ID           uuid.UUID
      Email        string
      PasswordHash string    // Solo para escritura. Nunca serializar a JSON.
      Tags         []string
      OrgUnitID    *uuid.UUID
      OrgPath      string
      CreatedAt    time.Time
      UpdatedAt    time.Time
  }

  type Credentials struct {
      Email    string
      Password string  // plaintext, solo en memoria durante autenticación
  }
  ```

  **`libs/domain/token.go`:**
  ```go
  const (
      ScopeGraphsRead    = "graphs:read"
      ScopeGraphsExecute = "graphs:execute"
      ScopeGraphsManage  = "graphs:manage"
      DefaultUserScopes  = ScopeGraphsRead + " " + ScopeGraphsExecute + " " + ScopeGraphsManage
  )

  type Claims struct {
      Subject    string
      Issuer     string
      Audience   []string
      Scope      string
      ClientType string
      Attrs      ClaimsAttrs
      ExpiresAt  time.Time
      IssuedAt   time.Time
  }

  type ClaimsAttrs struct {
      Tags    []string
      OrgUnit string
      OrgPath string
  }

  type TokenPair struct {
      AccessToken string
      TokenType   string
      ExpiresIn   int
  }
  ```

  **`libs/ports/auth.go`:**
  ```go
  type PasswordHasher interface {
      Hash(password string) (string, error)
      Verify(password, hash string) (bool, error)
  }

  type TokenIssuer interface {
      Issue(ctx context.Context, user *domain.User) (string, error)
  }

  type TokenValidator interface {
      Validate(ctx context.Context, token string) (*domain.Claims, error)
  }

  type UserRepository interface {
      Create(ctx context.Context, u *domain.User) (*domain.User, error)
      FindByEmail(ctx context.Context, email string) (*domain.User, error)
  }
  ```

- **Criterio de aceptación:** `go build ./libs/...` compila. El paquete
  `libs/domain/` no importa `golang.org/x/crypto` ni ninguna librería
  de crypto o JWT.
- **Depende de:** ninguno (paralelo a #1, #2)
- **Commit:** `feat(domain): add User, Credentials, Claims domain types and auth ports [SPRINT-004 #4]`

### 5. [test] Tests unitarios del hasher argon2id (Red)

- **Agente:** @qa
- **Descripción:** Tests del adaptador argon2id antes de implementarlo.

  ```go
  // tests/unit/auth/argon2id_test.go

  // TestHashProducesValidPHCString verifica que Hash() devuelve un
  // string con formato $argon2id$v=19$m=65536,...
  func TestHashProducesValidPHCString(t *testing.T)

  // TestVerifyCorrectPassword verifica que Verify() devuelve true
  // para la contraseña original.
  func TestVerifyCorrectPassword(t *testing.T)

  // TestVerifyWrongPassword verifica que Verify() devuelve false
  // para una contraseña incorrecta.
  func TestVerifyWrongPassword(t *testing.T)

  // TestHashesAreDifferentForSamePassword verifica que dos llamadas
  // a Hash() con la misma contraseña producen hashes distintos (salt aleatorio).
  func TestHashesAreDifferentForSamePassword(t *testing.T)

  // TestVerifyTimingSafe verifica que Verify() no termina antes con
  // contraseñas incorrectas (timing attack). Mide que el tiempo de
  // respuesta no revela información (umbral: desviación < 10%).
  func TestVerifyTimingSafe(t *testing.T)
  ```

- **Criterio de aceptación:** tests en RED antes del TODO #9. GREEN
  tras implementar `adapters/auth/argon2id.go`.
- **Depende de:** #4
- **Commit:** `test(unit): add argon2id hasher unit tests [SPRINT-004 #5]`

### 6. [test] Tests unitarios del emisor y validador JWT (Red)

- **Agente:** @qa
- **Descripción:** Tests del adaptador JWT.

  ```go
  // tests/unit/auth/jwt_test.go

  // TestIssueProducesValidJWT verifica que Issue() devuelve un JWT
  // parseable con los claims correctos (sub, iss, aud, scope, attrs).
  func TestIssueProducesValidJWT(t *testing.T)

  // TestIssueIncludesUserTags verifica que los tags del usuario
  // aparecen en attrs.tags del JWT.
  func TestIssueIncludesUserTags(t *testing.T)

  // TestValidateAcceptsValidJWT verifica que Validate() devuelve
  // los Claims correctos para un JWT bien formado y no expirado.
  func TestValidateAcceptsValidJWT(t *testing.T)

  // TestValidateRejectsExpiredJWT verifica que un JWT con exp en
  // el pasado devuelve error.
  func TestValidateRejectsExpiredJWT(t *testing.T)

  // TestValidateRejectsWrongSignature verifica que un JWT firmado
  // con una clave diferente devuelve error.
  func TestValidateRejectsWrongSignature(t *testing.T)

  // TestValidateRejectsWrongAudience verifica que un JWT con aud
  // diferente al esperado devuelve error.
  func TestValidateRejectsWrongAudience(t *testing.T)
  ```

  En los tests, el `TokenValidator` usa directamente la clave pública
  del mismo par RSA del `TokenIssuer` (sin HTTP, sin JWKS endpoint).
  El test de JWKS HTTP se hace en el test de contrato (#7).

- **Criterio de aceptación:** tests en RED antes del TODO #10.
  GREEN tras implementar `adapters/auth/jwt_issuer.go` y
  `adapters/auth/jwks_validator.go`.
- **Depende de:** #4
- **Commit:** `test(unit): add JWT issuer and validator unit tests [SPRINT-004 #6]`

### 7. [test] Tests de contrato del auth-server (Red)

- **Agente:** @qa
- **Descripción:** Tests con servidor de prueba real y fakes en memoria.
  Build tag `contract`.

  ```go
  //go:build contract

  // TestRegisterContract verifica POST /api/v1/auth/register → 201
  // con body que cumple UserResponse (sin password_hash).
  func TestRegisterContract(t *testing.T)

  // TestRegisterDuplicateEmailContract verifica 409 si el email
  // ya existe.
  func TestRegisterDuplicateEmailContract(t *testing.T)

  // TestLoginSuccessContract verifica POST /api/v1/auth/login → 200
  // con body que cumple TokenResponse (access_token, token_type, expires_in).
  func TestLoginSuccessContract(t *testing.T)

  // TestLoginWrongPasswordContract verifica 401 con ErrorResponse
  // code=INVALID_CREDENTIALS.
  func TestLoginWrongPasswordContract(t *testing.T)

  // TestJWKSEndpointContract verifica GET /.well-known/jwks.json → 200
  // con body que contiene campo "keys" array con al menos un JWK.
  func TestJWKSEndpointContract(t *testing.T)

  // TestLoginReturnsJWTWithCorrectClaims verifica que el token
  // devuelto por login es un JWT RS256 con sub, iss, aud, scope, attrs.
  func TestLoginReturnsJWTWithCorrectClaims(t *testing.T)
  ```

- **Criterio de aceptación:** tests en RED antes del TODO #12. GREEN
  tras implementar handlers y router.
- **Depende de:** #1, #4
- **Commit:** `test(contract): add auth-server contract tests [SPRINT-004 #7]`

### 8. [test] Tests unitarios de handlers y middleware (Red)

- **Agente:** @qa
- **Descripción:** Tests de la capa HTTP con `httptest.NewRecorder`.

  **`tests/unit/handler/auth_handler_test.go`:**
  ```go
  // TestRegisterHandlerSuccess verifica 201 con UserResponse.
  func TestRegisterHandlerSuccess(t *testing.T)

  // TestRegisterHandlerInvalidEmail verifica 422 con ValidationError.
  func TestRegisterHandlerInvalidEmail(t *testing.T)

  // TestLoginHandlerSuccess verifica 200 con TokenResponse.
  func TestLoginHandlerSuccess(t *testing.T)

  // TestLoginHandlerWrongCredentials verifica que ErrInvalidCredentials
  // se traduce a 401 (no 500).
  func TestLoginHandlerWrongCredentials(t *testing.T)
  ```

  **`tests/unit/middleware/auth_middleware_test.go`:**
  ```go
  // TestAuthMiddlewareBypassMode verifica que con AUTH_REQUIRED=false
  // el handler siguiente se ejecuta sin token.
  func TestAuthMiddlewareBypassMode(t *testing.T)

  // TestAuthMiddlewareValidToken verifica que un token válido inyecta
  // Claims en el contexto.
  func TestAuthMiddlewareValidToken(t *testing.T)

  // TestAuthMiddlewareMissingToken verifica que AUTH_REQUIRED=true
  // sin token devuelve 401 MISSING_TOKEN.
  func TestAuthMiddlewareMissingToken(t *testing.T)

  // TestAuthMiddlewareExpiredToken verifica que un token expirado
  // devuelve 401 INVALID_TOKEN.
  func TestAuthMiddlewareExpiredToken(t *testing.T)
  ```

- **Criterio de aceptación:** RED antes de #12 y #13. GREEN tras
  implementar handlers y middleware.
- **Depende de:** #4
- **Commit:** `test(unit): add auth handler and middleware unit tests [SPRINT-004 #8]`

### 9. [impl] Implementar adapters/auth/argon2id.go (Green para #5)

- **Agente:** @developer
- **Descripción:** Implementar `PasswordHasher` con argon2id.

  Parámetros (OWASP 2023): `memory=65536, time=3, threads=4,
  saltLen=16, keyLen=32`.

  `Hash()`:
  1. Generar salt aleatorio con `crypto/rand`.
  2. Computar hash con `golang.org/x/crypto/argon2.IDKey`.
  3. Encodificar como PHC string: `$argon2id$v=19$m=65536,t=3,p=4$<b64salt>$<b64hash>`.

  `Verify()`:
  1. Parsear el PHC string para extraer parámetros + salt.
  2. Recomputar hash con los mismos parámetros.
  3. Comparar con `subtle.ConstantTimeCompare` (timing-safe).

  La función de parseo del PHC string puede reutilizarse entre
  `Hash()` y `Verify()`.

- **Criterio de aceptación:** `go test ./tests/unit/auth/...`
  (solo `TestHash*` y `TestVerify*`) pasa a GREEN.
  `make lint` sin errores. Ninguna función > 20 líneas.
- **Depende de:** #5
- **Commit:** `feat(auth): implement argon2id password hasher [SPRINT-004 #9]`

### 10. [impl] Implementar adapters/auth/jwt_issuer.go y jwks_validator.go

- **Agente:** @developer
- **Descripción:** Implementar `TokenIssuer` y `TokenValidator`.

  **`jwt_issuer.go`** — `JWTIssuer` struct con `*rsa.PrivateKey` y
  config (`issuer`, `audience`, `ttl`):
  1. Construir `jwt.MapClaims` con todos los campos de ADR-012.
  2. Firmar con `jwt.SigningMethodRS256`.
  3. Serializar con `token.SignedString(privateKey)`.

  **`jwks_validator.go`** — `JWKSValidator` struct:
  - En modo test/dev: acepta directamente una `*rsa.PublicKey`.
  - En modo producción: fetch HTTP lazy + caché con TTL configurable
    (default 5 min) del endpoint `JWKS_URL`.
  - `Validate()`: parsea JWT con `jwt.ParseWithClaims`, verifica
    `exp`, `iss`, `aud`. Devuelve `*domain.Claims`.

  **`jwks_endpoint.go`** — función que genera el JSON del JWKS:
  - Convierte `*rsa.PublicKey` a JWK (kid, kty, alg, n, e).
  - `kid` = primeros 8 bytes del SHA-256 del DER encoding de la clave.

- **Criterio de aceptación:** `go test ./tests/unit/auth/...`
  (todos los tests de `jwt_test.go`) pasa a GREEN. `make lint` limpio.
- **Depende de:** #6
- **Commit:** `feat(auth): implement JWT RS256 issuer and JWKS validator [SPRINT-004 #10]`

### 11. [impl] Implementar use cases RegisterUser y LoginUser

- **Agente:** @developer
- **Descripción:** Casos de uso en `services/auth-server/internal/usecase/`.

  **`RegisterUser.Execute(ctx, Credentials) (*domain.User, error)`:**
  1. Validar email (formato RFC) y password (≥ 12 chars).
     Si inválido → `domain.ErrValidation`.
  2. Verificar que el email no existe en `UserRepository`.
     Si existe → `domain.ErrConflict`.
  3. Hashear password con `PasswordHasher.Hash()`.
  4. Crear `domain.User` con UUID nuevo, email, hash, tags vacíos.
  5. Persistir con `UserRepository.Create()`.
  6. Devolver el usuario creado (sin `PasswordHash` en la respuesta —
     el handler lo filtra).

  **`LoginUser.Execute(ctx, Credentials) (*domain.TokenPair, error)`:**
  1. Buscar usuario por email. Si no existe → `domain.ErrNotFound`
     (pero el handler devuelve 401 genérico — sin revelar si el email existe).
  2. Verificar password con `PasswordHasher.Verify()`. Si falla → error
     centinela `ErrInvalidCredentials`.
  3. Emitir JWT con `TokenIssuer.Issue()`.
  4. Devolver `TokenPair{AccessToken, "Bearer", 3600}`.

  `ErrInvalidCredentials` nuevo en `libs/domain/errors.go`.

- **Criterio de aceptación:** `go test ./tests/unit/usecase/...`
  (si se añaden tests de usecase) pasa. Lógica de `LoginUser` nunca
  revela si el email existe o no (ambos retornan el mismo error genérico
  al handler).
- **Depende de:** #4, #9, #10
- **Commit:** `feat(auth): implement RegisterUser and LoginUser use cases [SPRINT-004 #11]`

### 12. [impl] Implementar handlers Gin y router del auth-server

- **Agente:** @developer
- **Descripción:** Handlers en `services/auth-server/internal/handler/auth.go`
  siguiendo ADR-006.

  **`Register`** handler:
  - `ShouldBindJSON` → `RegisterInput`.
  - Llama `RegisterUser.Execute()`.
  - Mapea `ErrConflict` → 409, `ErrValidation` → 422.
  - Devuelve 201 con `UserResponse` (filtra `PasswordHash`).

  **`Login`** handler:
  - Llama `LoginUser.Execute()`.
  - Mapea `ErrInvalidCredentials` → 401 con `code=INVALID_CREDENTIALS`.
  - Devuelve 200 con `TokenResponse`.

  **`JWKS`** handler:
  - Sirve el JSON del JWKS endpoint pre-computado al arranque.
  - No requiere autenticación. Content-Type: `application/json`.

  **`services/auth-server/internal/router/router.go`:**
  ```go
  func NewRouter(authHandler *handler.AuthHandler, jwksJSON []byte) *gin.Engine {
      r := gin.New()
      r.Use(gin.Recovery(), middlewareLogger(), middlewareRequestID())

      // Ruta OAuth estándar — fuera de /api/v1/
      r.GET("/.well-known/jwks.json", authHandler.JWKS)

      v1 := r.Group("/api/v1")
      auth := v1.Group("/auth")
      {
          auth.POST("/register", authHandler.Register)
          auth.POST("/login", authHandler.Login)
      }
      return r
  }
  ```

- **Criterio de aceptación:** `go test ./tests/unit/handler/...` y
  `go test -tags=contract ./tests/contract/...` pasan a GREEN.
  `make lint` limpio.
- **Depende de:** #7, #8, #11
- **Commit:** `feat(auth): implement auth-server Gin handlers and router [SPRINT-004 #12]`

### 13. [impl] Implementar middleware JWT en el orchestrator

- **Agente:** @developer
- **Descripción:** Crear `services/orchestrator/internal/middleware/auth.go`.

  Configuración desde env vars (`AUTH_REQUIRED`, `JWKS_URL`,
  `JWT_ISSUER`, `JWT_AUDIENCE`).

  Lógica:
  1. Si `AUTH_REQUIRED=false` → `c.Next()` directo.
  2. Extraer header `Authorization: Bearer <token>`. Si falta → 401.
  3. Llamar `TokenValidator.Validate(ctx, token)`. Si error → 401.
  4. `c.Set("claims", claims)`.
  5. `c.Next()`.

  Helper para handlers: `ClaimsFromContext(c *gin.Context) (*domain.Claims, bool)`.

  El middleware **no se aplica** a las rutas de SPRINT-003 todavía
  (eso es SPRINT-005). Se registra en el router pero protege
  únicamente un grupo `/api/v1/protected` vacío por ahora (para
  verificar que compila y los tests pasan).

- **Criterio de aceptación:** `go test ./tests/unit/middleware/...`
  pasa a GREEN. El middleware compila y se registra en el router del
  orchestrator sin afectar las rutas existentes.
- **Depende de:** #8, #10
- **Commit:** `feat(orchestrator): add JWT validation middleware [SPRINT-004 #13]`

### 14. [impl] auth-server main.go y adaptador Ent de usuarios

- **Agente:** @developer
- **Descripción:** Dos artefactos para completar el wiring.

  **`adapters/auth/ent_user_repo.go`** — `EntUserRepository` que
  implementa `ports.UserRepository` usando el cliente Ent. Traduce
  `*ent.User` ↔ `*domain.User`. Nunca expone `*ent.User` fuera del
  adaptador.

  **`services/auth-server/cmd/main.go`** — wiring completo:
  1. Cargar RSA keypair desde `JWT_PRIVATE_KEY_PATH` /
     `JWT_PUBLIC_KEY_PATH`. Si no existen, generar un par en memoria
     y loguear advertencia.
  2. Conectar a PostgreSQL (`DATABASE_URL`), crear `ent.Client`.
  3. Construir `EntUserRepository`, `Argon2idHasher`, `JWTIssuer`
     (con `JWT_ISSUER`, `JWT_AUDIENCE=dago-api`, TTL=3600s).
  4. Construir casos de uso.
  5. Construir handlers con `jwksJSON` pre-computado.
  6. Construir y arrancar router en `AUTH_PORT` (default 8081).
  7. Manejo de señales de cierre.

- **Criterio de aceptación:** `go build ./services/auth-server/...`
  compila. El binario arranca y responde:
  ```bash
  curl -s http://localhost:8081/.well-known/jwks.json | jq .keys[0].kty
  # → "RSA"
  curl -s -X POST http://localhost:8081/api/v1/auth/register \
       -H 'Content-Type: application/json' \
       -d '{"email":"test@example.com","password":"securepass123"}' | jq .id
  # → "<uuid>"
  curl -s -X POST http://localhost:8081/api/v1/auth/login \
       -H 'Content-Type: application/json' \
       -d '{"email":"test@example.com","password":"securepass123"}' | jq .token_type
  # → "Bearer"
  ```
- **Depende de:** #11, #12, #13, #3
- **Commit:** `feat(auth): wire up auth-server with Ent user repo and RSA keypair [SPRINT-004 #14]`

### 15. [test] Test de integración end-to-end auth

- **Agente:** @qa
- **Descripción:** Test con servidor real y PostgreSQL.
  Build tag `integration`.

  ```go
  //go:build integration

  // TestAuthFlowIntegration verifica el flujo completo:
  // 1. Registrar usuario.
  // 2. Login → obtener JWT.
  // 3. Parsear JWT y verificar claims (sub=user ID, scope, attrs.tags vacíos).
  // 4. Validar JWT contra el JWKS del mismo servidor.
  // 5. Verificar que login con contraseña incorrecta devuelve 401.
  func TestAuthFlowIntegration(t *testing.T)
  ```

- **Criterio de aceptación:** `make test-integration` pasa con este
  test. Falla si PostgreSQL no está levantado.
- **Depende de:** #14
- **Commit:** `test(integration): add end-to-end auth flow integration test [SPRINT-004 #15]`

### 16. [docs] Actualizar docs/index.md y docs/log.md

- **Agente:** @docs
- **Descripción:** Añadir SPRINT-004 a la tabla de sprints. Actualizar
  la sección "Dominio — Schemas Ent" marcando User y OrgUnit como
  planificados en SPRINT-004. Añadir sección "Servicios implementados"
  que mencione el auth-server básico y el middleware del orchestrator.
  Actualizar `docs/log.md`.
- **Criterio de aceptación:** `docs/index.md` refleja el nuevo estado.
- **Depende de:** #15
- **Commit:** `docs(auth): update index with SPRINT-004 results [SPRINT-004 #16]`

## Matriz de trazabilidad

| Spec / ADR | Regla | TODO | Artefacto | Verificado por |
|------------|-------|------|-----------|----------------|
| ADR-012 regla 2 | argon2id para contraseñas locales | #9 | `adapters/auth/argon2id.go` | `TestHash*`, `TestVerify*` |
| ADR-012 regla 3 | JWT RS256 con attrs ABAC | #10 | `adapters/auth/jwt_issuer.go` | `TestIssueProducesValidJWT` |
| ADR-012 regla 4 | Validación local vía JWKS | #10, #13 | `jwks_validator.go`, middleware | `TestValidateAcceptsValidJWT` |
| ADR-012 regla 6 | Tokens en memoria, nunca localStorage | #13 | middleware inyecta en ctx | code review |
| ADR-012 regla 10 | No almacenar contraseñas en backend API | #2, #11 | `Sensitive()`, `UserResponse` sin hash | `TestRegisterContract` (sin password_hash) |
| ADR-010 regla 1 | Prefijo `/api/v1/` para rutas de negocio | #12 | router auth-server | tests de contrato |
| ADR-010 regla 8 | Formato `ErrorResponse` estándar | #12 | `mapDomainError` auth | `TestLogin*` |
| ADR-006 regla 1 | Handlers delgados | #12 | handler/auth.go | `make lint` (funlen) |
| ADR-007 regla 7 | Tipos Ent no salen del adaptador | #14 | `ent_user_repo.go` | code review |
| ADR-007 regla 8 | UUIDs + TIMESTAMPTZ | #2 | `user.go`, `org_unit.go` | `\d users` PostgreSQL |
| ADR-001 regla 1 | Dominio sin infraestructura | #4 | `libs/domain/` sin crypto | `go build` |
| ADR-001 regla 2 | Puertos como interfaces | #4 | `libs/ports/auth.go` | compilador |
| ADR-002 | TDD Red → Green | #5–#8 antes de #9–#13 | orden de TODOs | CI |
| ADR-003 regla 2 | Funciones ≤ 20 líneas | todos los #impl | todos los ficheros | `make lint` |
| OWASP argon2id | m=64MB, t=3, p=4 | #9 | `argon2id.go` constantes | `TestHashProducesValidPHCString` |

## Criterios de aceptación del sprint

```bash
# 1. Spec válida
swagger-cli validate specs/openapi.yaml

# 2. Compilación completa
go build ./...

# 3. Linter limpio
make lint

# 4. Tests unitarios (sin Docker)
go test ./tests/unit/auth/...
go test ./tests/unit/handler/...
go test ./tests/unit/middleware/...

# 5. Tests de contrato (con fakes, sin Docker)
go test -tags=contract ./tests/contract/...

# 6. Tests de integración (con Docker)
make test-integration

# 7. Flujo completo manual
make docker-up
go run ./services/auth-server/cmd/main.go &
# registrar, login, inspeccionar JWKS
```

Adicionalmente:
- `libs/domain/` no importa `golang.org/x/crypto` ni `golang-jwt/jwt`.
- `UserResponse` (spec + handler) no incluye `password_hash`.
- `LoginUser` devuelve el mismo error para email inexistente y contraseña
  incorrecta (sin revelar existencia de cuenta).
- El middleware del orchestrator con `AUTH_REQUIRED=false` no rompe
  ningún test de SPRINT-003.
- `argon2id.go` usa `subtle.ConstantTimeCompare` en `Verify()`.
- JWKS contiene exactamente una clave con `kty=RSA`, `alg=RS256`.

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
