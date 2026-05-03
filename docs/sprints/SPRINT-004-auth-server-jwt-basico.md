# SPRINT-004: Auth-server — local login, JWT RS256, JWKS, middleware

## Metadata

- **Start date:** 2026-04-29 (parallel to SPRINT-002 and SPRINT-003 — only depends on SPRINT-001)
- **Estimated end date:** 2026-04-30
- **Status:** planned
- **Applied ADRs:** ADR-001, ADR-002, ADR-003, ADR-004, ADR-006, ADR-007, ADR-012
- **Affected specs:**
  - `specs/paths/auth.yaml` — new (register, login, JWKS)
  - `specs/schemas/auth.yaml` — new (LoginInput, TokenResponse, RegisterInput)
  - `specs/openapi.yaml` — add `$ref` to the new auth paths
  - `ent/schema/user.go` — new
  - `ent/schema/org_unit.go` — new
- **Planning agent:** planner
- **Reviewed by:** pending
- **Depends on:** SPRINT-001 (go.mod, docker-compose PostgreSQL, atlas.hcl)
- **Parallel with:** SPRINT-002, SPRINT-003 (independent of Graph/Node/Execution)
- **Blocks:** SPRINT-005 (protected routes in orchestrator), SPRINT-ABAC (authorization)

## Objective

Implement the core authentication for dago: local login with passwords
hashed in argon2id, JWT RS256 issuance with ABAC claims, JWKS endpoint
for local validation, and the validation middleware for the orchestrator.

At completion: the auth-server accepts credentials, issues signed JWT tokens and
serves its public key via JWKS. The orchestrator has a functional middleware
that validates those tokens, with bypass mode for development.

## Scope

### Included

**New Ent schemas:**
- `ent/schema/user.go` — User (id, email, password_hash, tags, org_unit FK).
- `ent/schema/org_unit.go` — OrgUnit (id, name, path, tags, parent/children self-ref).
- `go generate ./ent` — Ent client regenerated with the two new schemas.
- `atlas migrate diff add_user_org_unit` — SQL migration for the two tables.
- `atlas migrate apply --env local` — applied against docker-compose PostgreSQL.

**OpenAPI spec:**
- `specs/paths/auth.yaml` — endpoints: POST /auth/register, POST /auth/login,
  GET /.well-known/jwks.json.
- `specs/schemas/auth.yaml` — RegisterInput, LoginInput, TokenResponse.
- `specs/openapi.yaml` updated with `$ref` entries.

**Domain (`libs/domain/`):**
- `libs/domain/user.go` — types `User`, `Credentials`.
- `libs/domain/token.go` — types `Claims`, `TokenPair`, scope constants.

**Ports (`libs/ports/auth.go`):**
- `PasswordHasher` — `Hash(password string) (string, error)` and `Verify`.
- `TokenIssuer` — `Issue(ctx, *domain.User) (string, error)`.
- `TokenValidator` — `Validate(ctx, token string) (*domain.Claims, error)`.
- `UserRepository` — `Create` and `FindByEmail`.

**Adapters (`adapters/auth/`):**
- `adapters/auth/argon2id.go` — implements `PasswordHasher` with argon2id.
- `adapters/auth/jwt_issuer.go` — implements `TokenIssuer` with RS256.
- `adapters/auth/jwks_validator.go` — implements `TokenValidator` with JWKS HTTP.
- `adapters/auth/ent_user_repo.go` — implements `UserRepository` with Ent.

**auth-server (`services/auth-server/`):**
- `internal/usecase/register.go` — `RegisterUser` use case.
- `internal/usecase/login.go` — `LoginUser` use case.
- `internal/handler/auth.go` — handlers for register, login, JWKS.
- `internal/router/router.go` — `NewRouter()` with middlewares and routes.
- `cmd/main.go` — full wiring: RS256 keypair, Ent, repository, use
  cases, handlers, server on port 8081.

**Orchestrator middleware:**
- `services/orchestrator/internal/middleware/auth.go` — validates Bearer JWT,
  injects `*domain.Claims` into `gin.Context`, bypass when `AUTH_REQUIRED=false`.

**Tests:**
- `tests/unit/auth/argon2id_test.go` — hash + verify, timing-safe.
- `tests/unit/auth/jwt_test.go` — issue + validate, expiry, ABAC claims.
- `tests/contract/auth_contract_test.go` — responses against OpenAPI spec.
- `tests/unit/handler/auth_handler_test.go` — handlers with fakes.
- `tests/unit/middleware/auth_middleware_test.go` — orchestrator middleware.
- `tests/integration/auth_integration_test.go` — real login against PostgreSQL.

### Excluded

- Authorization Code + PKCE (requires dashboard — SPRINT-frontend-001).
- Client Credentials M2M (SPRINT-ABAC).
- Refresh tokens / rotation (SPRINT-ABAC).
- `POST /token` standard OAuth 2.1 endpoint (SPRINT-ABAC).
- `GET /authorize` and `POST /revoke` (SPRINT-ABAC).
- Full ABAC engine (`Authorizer`, tag inheritance via OU tree)
  — SPRINT-ABAC. This sprint only emits the user's tags directly.
- Identity Broker (external IdPs: Google, Azure AD) — SPRINT-ABAC.
- Applying the middleware to the orchestrator routes (SPRINT-005).
  This sprint implements the middleware and verifies it works, but does not
  activate it on SPRINT-003 routes.
- `DELETE /api/v1/auth/users/:id` or user management (SPRINT-ABAC).

## Dependencies

- **SPRINT-001 completed:** `go.mod` with `golang.org/x/crypto` and
  `github.com/golang-jwt/jwt/v5`, `services/auth-server/` structure,
  docker-compose PostgreSQL, atlas.hcl.
- **If SPRINT-002 already ran:** this sprint's migration only adds
  `users` and `org_units`. Atlas calculates the diff against the current
  DB state — does not interfere with existing tables.
- **If SPRINT-002 has not run yet:** the migration includes the three tables
  from SPRINT-002 plus the two from this sprint (Atlas is additive).

## Behavior Contracts

### C1 — `POST /api/v1/auth/login` — valid credentials

```
Given: User registered with email "user@example.com" and password "securepass123"
When: POST /api/v1/auth/login with body {"email":"user@example.com","password":"securepass123"}
Then: HTTP 200, TokenResponse with non-empty access_token, token_type="Bearer", expires_in=3600
      The access_token is a JWT RS256 with claims sub, iss, aud, scope, attrs
```

### C2 — `POST /api/v1/auth/login` — wrong credentials (timing-safe)

```
Given: User registered with password "securepass123"
When: POST /api/v1/auth/login with incorrect password "wrongpass"
Then: HTTP 401, ErrorResponse with code = "INVALID_CREDENTIALS"
      The error message does NOT reveal whether the email exists (same error)
      Response time is comparable to a successful authentication (timing-safe argon2id)
```

### C3 — `GET /.well-known/jwks.json`

```
Given: auth-server started with configured RSA keypair
When: GET /.well-known/jwks.json without Authorization header
Then: HTTP 200, JSON with field "keys" as array
      keys[0].kty = "RSA", keys[0].alg = "RS256"
      The endpoint is public (does not require authentication)
```

### C4 — JWT middleware in orchestrator — bypass mode

```
Given: AUTH_REQUIRED=false in orchestrator environment variables
When: Any request to the orchestrator without Authorization header
Then: The middleware calls c.Next() and the handler executes normally
      HTTP 200 (or the handler's own status)
      SPRINT-003 tests continue to pass without changes
```

## Design

### Ent Schemas

**User:**

| Field | Ent Type | PostgreSQL Type | Notes |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | `uuid.New()` |
| `email` | `string` | `VARCHAR(320) UNIQUE NOT NULL` | validated with RFC 5321 regex |
| `password_hash` | `string` | `TEXT NOT NULL` | argon2id hash, never in API responses |
| `tags` | `[]string` | `TEXT[] NOT NULL DEFAULT '{}'` | ABAC tags of the user |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | immutable |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | auto |

Edges: `M2O` with `OrgUnit` (optional).

**OrgUnit:**

| Field | Ent Type | PostgreSQL Type | Notes |
|-------|----------|-----------------|-------|
| `id` | `uuid.UUID` | `UUID PK` | `uuid.New()` |
| `name` | `string` | `VARCHAR(255) NOT NULL` | short name, e.g. `"engineering"` |
| `path` | `string` | `TEXT UNIQUE NOT NULL` | full path, e.g. `"/company/engineering"` |
| `tags` | `[]string` | `TEXT[] NOT NULL DEFAULT '{}'` | tags inherited by users |
| `created_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | immutable |
| `updated_at` | `time.Time` | `TIMESTAMPTZ NOT NULL` | auto |

Edges: `O2M` to `children` (OrgUnit), `M2O` to `parent` (OrgUnit, optional),
`O2M` to `users`.

### JWT format (RS256)

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

For this sprint, `attrs.tags` = direct User tags (without OU tree inheritance —
that logic is completed in SPRINT-ABAC).

### argon2id — OWASP 2023 parameters

```go
const (
    argon2Memory      = 64 * 1024  // 64 MB
    argon2Iterations  = 3
    argon2Parallelism = 4
    argon2SaltLen     = 16
    argon2KeyLen      = 32
)
```

Stored hash: `$argon2id$v=19$m=65536,t=3,p=4$<salt_b64>$<hash_b64>`
(standard PHC string format — parseable by `golang.org/x/crypto/argon2`).

### RS256 keypair

- Loaded from environment variables `JWT_PRIVATE_KEY_PATH` (PEM file)
  and `JWT_PUBLIC_KEY_PATH` (PEM file).
- If they don't exist, generated in memory at startup (development only).
  In production the keys come from a secret manager.
- The JWKS endpoint exposes only the public key in JWK format
  with `kid` (key ID) derived from the SHA-256 thumbprint.

### auth-server endpoints

| Method | Path | Body | Response |
|--------|------|------|----------|
| `POST` | `/api/v1/auth/register` | `RegisterInput` | 201 `UserResponse` / 409 |
| `POST` | `/api/v1/auth/login` | `LoginInput` | 200 `TokenResponse` / 401 |
| `GET` | `/.well-known/jwks.json` | — | 200 JWKS JSON |

`RegisterInput`: `{"email": "...", "password": "..."}` (password ≥ 12 chars).
`LoginInput`: `{"email": "...", "password": "..."}`.
`TokenResponse`: `{"access_token": "...", "token_type": "Bearer", "expires_in": 3600}`.
`UserResponse`: `{"id": "...", "email": "...", "tags": [], "created_at": "..."}`.
— `password_hash` **never** appears in the response (ADR-012 rule 10).

### Orchestrator middleware

```
Authorization: Bearer <jwt>
        │
        ▼
middleware/auth.go
        │
        ├── AUTH_REQUIRED=false → c.Next() (dev bypass)
        │
        └── AUTH_REQUIRED=true
                │
                ├── No header → 401 MISSING_TOKEN
                ├── Invalid/expired JWT → 401 INVALID_TOKEN
                └── Valid JWT → inject *Claims in ctx → c.Next()
```

Claims accessible from handlers: `c.MustGet("claims").(*domain.Claims)`.

Middleware environment variables:
- `AUTH_REQUIRED` — `false` (dev) / `true` (prod).
- `JWKS_URL` — URL of the auth-server JWKS endpoint.
- `JWT_ISSUER` — expected value of the JWT `iss` field.
- `JWT_AUDIENCE` — expected value of the `aud` field (default `"dago-api"`).

## TODOs

### 1. [spec] Write specs/schemas/auth.yaml and specs/paths/auth.yaml

- **Agente:** @developer
- **Description:** Create OpenAPI schemas `RegisterInput`,
  `LoginInput`, `TokenResponse`, `UserResponse`. Create the path item
  for the three auth endpoints (register, login, JWKS).
  Update `specs/openapi.yaml` with the corresponding `$ref` entries.

  ADR-010 rules:
  - Register and login under `/api/v1/auth/` (prefix `/api/v1/`).
  - JWKS at `/.well-known/jwks.json` (standard OAuth exception —
    ADR-006 rule 4 note).
  - `UserResponse` does not include `password_hash` or any credential field.
  - All errors use `ErrorResponse` (code + message).

  Status codes:
  - `POST /auth/register`: 201, 400, 409 (duplicate email), 422.
  - `POST /auth/login`: 200, 400, 401 (wrong credentials), 422.
  - `GET /.well-known/jwks.json`: 200 (always — no auth required).

- **Acceptance criteria:** `swagger-cli validate specs/openapi.yaml`
  passes. `UserResponse` has no `password_hash` field.
- **Depends on:** none
- **Commit:** `spec(openapi): add auth register, login, and JWKS endpoints [SPRINT-004 #1]`

### 2. [data] Implement ent/schema/user.go and ent/schema/org_unit.go

- **Agente:** @developer
- **Description:** Create the two Ent schemas according to the data model
  in this document.

  Validations in `user.go`:
  - `email`: `field.String("email").MaxLen(320).Match(emailRegex).Unique()`.
  - `password_hash`: `field.String("password_hash").Sensitive()` — the
    `.Sensitive()` tag excludes the field from Ent logs.
  - `tags`: `field.Strings("tags").Default([]string{})`.

  Validations in `org_unit.go`:
  - `path`: starts with `/`, does not end with `/` (except `/`).
  - `path`: `field.String("path").MaxLen(1024).Unique()`.
  - Self-referential edge correctly typed (Ent requires a different
    ref name for parent and children).

- **Acceptance criteria:** `go build ./ent/...` compiles. The
  `password_hash` field has `.Sensitive()`.
- **Depends on:** none (parallel to #1)
- **Commit:** `feat(schema): add User and OrgUnit Ent schemas [SPRINT-004 #2]`

### 3. [data] Run go generate and Atlas migration

- **Agente:** @developer
- **Description:** With `make docker-up` active:

  ```bash
  # Regenerate Ent client (includes User and OrgUnit alongside prior schemas)
  go generate ./ent

  # Generate migration for the new tables
  atlas migrate diff add_user_org_unit --env local

  # Apply migration
  atlas migrate apply --env local

  # Verify created tables
  docker compose exec postgres psql -U dago -d dago \
      -c "\d users" -c "\d org_units"
  ```

  Verify that the generated SQL includes:
  - `CREATE TABLE users` with `email UNIQUE`, `password_hash TEXT`,
    `tags TEXT[]`.
  - `CREATE TABLE org_units` with `path TEXT UNIQUE`, `tags TEXT[]`,
    optional self-referential FK.
  - FK `users.org_unit_id → org_units.id` (nullable).

- **Acceptance criteria:** `atlas migrate lint --env local` no
  errors. Tables `users` and `org_units` exist in PostgreSQL.
  `go build ./...` compiles with the new generated types.
- **Depends on:** #2
- **Commit:** `feat(schema): generate Ent client with User and OrgUnit; add migration [SPRINT-004 #3]`

### 4. [domain] Domain types and auth ports

- **Agente:** @developer
- **Description:** Create the pure domain types and the auth ports.
  No imports of Ent, Gin or crypto.

  **`libs/domain/user.go`:**
  ```go
  type User struct {
      ID           uuid.UUID
      Email        string
      PasswordHash string    // Write-only. Never serialize to JSON.
      Tags         []string
      OrgUnitID    *uuid.UUID
      OrgPath      string
      CreatedAt    time.Time
      UpdatedAt    time.Time
  }

  type Credentials struct {
      Email    string
      Password string  // plaintext, in memory only during authentication
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

- **Acceptance criteria:** `go build ./libs/...` compiles. The
  `libs/domain/` package does not import `golang.org/x/crypto` or any
  crypto or JWT library.
- **Depends on:** none (parallel to #1, #2)
- **Commit:** `feat(domain): add User, Credentials, Claims domain types and auth ports [SPRINT-004 #4]`

### 5. [test] argon2id hasher unit tests (Red)

- **Agente:** @qa
- **Description:** Tests for the argon2id adapter before implementing it.

  ```go
  // tests/unit/auth/argon2id_test.go

  // TestHashProducesValidPHCString verifies that Hash() returns a
  // string with format $argon2id$v=19$m=65536,...
  func TestHashProducesValidPHCString(t *testing.T)

  // TestVerifyCorrectPassword verifies that Verify() returns true
  // for the original password.
  func TestVerifyCorrectPassword(t *testing.T)

  // TestVerifyWrongPassword verifies that Verify() returns false
  // for an incorrect password.
  func TestVerifyWrongPassword(t *testing.T)

  // TestHashesAreDifferentForSamePassword verifies that two calls
  // to Hash() with the same password produce different hashes (random salt).
  func TestHashesAreDifferentForSamePassword(t *testing.T)

  // TestVerifyTimingSafe verifies that Verify() does not terminate earlier with
  // incorrect passwords (timing attack). Measures that response time
  // does not reveal information (threshold: deviation < 10%).
  func TestVerifyTimingSafe(t *testing.T)
  ```

- **Acceptance criteria:** tests in RED before TODO #9. GREEN
  after implementing `adapters/auth/argon2id.go`.
- **Depends on:** #4
- **Commit:** `test(unit): add argon2id hasher unit tests [SPRINT-004 #5]`

### 6. [test] JWT issuer and validator unit tests (Red)

- **Agente:** @qa
- **Description:** Tests for the JWT adapter.

  ```go
  // tests/unit/auth/jwt_test.go

  // TestIssueProducesValidJWT verifies that Issue() returns a JWT
  // parseable with the correct claims (sub, iss, aud, scope, attrs).
  func TestIssueProducesValidJWT(t *testing.T)

  // TestIssueIncludesUserTags verifies that the user's tags
  // appear in attrs.tags of the JWT.
  func TestIssueIncludesUserTags(t *testing.T)

  // TestValidateAcceptsValidJWT verifies that Validate() returns
  // the correct Claims for a well-formed and non-expired JWT.
  func TestValidateAcceptsValidJWT(t *testing.T)

  // TestValidateRejectsExpiredJWT verifies that a JWT with exp in
  // the past returns an error.
  func TestValidateRejectsExpiredJWT(t *testing.T)

  // TestValidateRejectsWrongSignature verifies that a JWT signed
  // with a different key returns an error.
  func TestValidateRejectsWrongSignature(t *testing.T)

  // TestValidateRejectsWrongAudience verifies that a JWT with an aud
  // different from the expected one returns an error.
  func TestValidateRejectsWrongAudience(t *testing.T)
  ```

  In the tests, the `TokenValidator` uses the public key from the same RSA
  pair as the `TokenIssuer` (no HTTP, no JWKS endpoint).
  The JWKS HTTP test is done in the contract test (#7).

- **Acceptance criteria:** tests in RED before TODO #10.
  GREEN after implementing `adapters/auth/jwt_issuer.go` and
  `adapters/auth/jwks_validator.go`.
- **Depends on:** #4
- **Commit:** `test(unit): add JWT issuer and validator unit tests [SPRINT-004 #6]`

### 7. [test] auth-server contract tests (Red)

- **Agente:** @qa
- **Description:** Tests with a real test server and in-memory fakes.
  Build tag `contract`.

  ```go
  //go:build contract

  // TestRegisterContract verifies POST /api/v1/auth/register → 201
  // with body that fulfills UserResponse (without password_hash).
  func TestRegisterContract(t *testing.T)

  // TestRegisterDuplicateEmailContract verifies 409 if the email
  // already exists.
  func TestRegisterDuplicateEmailContract(t *testing.T)

  // TestLoginSuccessContract verifies POST /api/v1/auth/login → 200
  // with body that fulfills TokenResponse (access_token, token_type, expires_in).
  func TestLoginSuccessContract(t *testing.T)

  // TestLoginWrongPasswordContract verifies 401 with ErrorResponse
  // code=INVALID_CREDENTIALS.
  func TestLoginWrongPasswordContract(t *testing.T)

  // TestJWKSEndpointContract verifies GET /.well-known/jwks.json → 200
  // with body that contains field "keys" array with at least one JWK.
  func TestJWKSEndpointContract(t *testing.T)

  // TestLoginReturnsJWTWithCorrectClaims verifies that the token
  // returned by login is a JWT RS256 with sub, iss, aud, scope, attrs.
  func TestLoginReturnsJWTWithCorrectClaims(t *testing.T)
  ```

- **Acceptance criteria:** tests in RED before TODO #12. GREEN
  after implementing handlers and router.
- **Depends on:** #1, #4
- **Commit:** `test(contract): add auth-server contract tests [SPRINT-004 #7]`

### 8. [test] Handler and middleware unit tests (Red)

- **Agente:** @qa
- **Description:** Tests for the HTTP layer with `httptest.NewRecorder`.

  **`tests/unit/handler/auth_handler_test.go`:**
  ```go
  // TestRegisterHandlerSuccess verifies 201 with UserResponse.
  func TestRegisterHandlerSuccess(t *testing.T)

  // TestRegisterHandlerInvalidEmail verifies 422 with ValidationError.
  func TestRegisterHandlerInvalidEmail(t *testing.T)

  // TestLoginHandlerSuccess verifies 200 with TokenResponse.
  func TestLoginHandlerSuccess(t *testing.T)

  // TestLoginHandlerWrongCredentials verifies that ErrInvalidCredentials
  // translates to 401 (not 500).
  func TestLoginHandlerWrongCredentials(t *testing.T)
  ```

  **`tests/unit/middleware/auth_middleware_test.go`:**
  ```go
  // TestAuthMiddlewareBypassMode verifies that with AUTH_REQUIRED=false
  // the next handler executes without a token.
  func TestAuthMiddlewareBypassMode(t *testing.T)

  // TestAuthMiddlewareValidToken verifies that a valid token injects
  // Claims into the context.
  func TestAuthMiddlewareValidToken(t *testing.T)

  // TestAuthMiddlewareMissingToken verifies that AUTH_REQUIRED=true
  // without a token returns 401 MISSING_TOKEN.
  func TestAuthMiddlewareMissingToken(t *testing.T)

  // TestAuthMiddlewareExpiredToken verifies that an expired token
  // returns 401 INVALID_TOKEN.
  func TestAuthMiddlewareExpiredToken(t *testing.T)
  ```

- **Acceptance criteria:** RED before #12 and #13. GREEN after
  implementing handlers and middleware.
- **Depends on:** #4
- **Commit:** `test(unit): add auth handler and middleware unit tests [SPRINT-004 #8]`

### 9. [impl] Implement adapters/auth/argon2id.go (Green for #5)

- **Agente:** @developer
- **Description:** Implement `PasswordHasher` with argon2id.

  Parameters (OWASP 2023): `memory=65536, time=3, threads=4,
  saltLen=16, keyLen=32`.

  `Hash()`:
  1. Generate random salt with `crypto/rand`.
  2. Compute hash with `golang.org/x/crypto/argon2.IDKey`.
  3. Encode as PHC string: `$argon2id$v=19$m=65536,t=3,p=4$<b64salt>$<b64hash>`.

  `Verify()`:
  1. Parse the PHC string to extract parameters + salt.
  2. Recompute hash with the same parameters.
  3. Compare with `subtle.ConstantTimeCompare` (timing-safe).

  The PHC string parsing function can be reused between
  `Hash()` and `Verify()`.

- **Acceptance criteria:** `go test ./tests/unit/auth/...`
  (only `TestHash*` and `TestVerify*`) turns GREEN.
  `make lint` no errors. No function > 20 lines.
- **Depends on:** #5
- **Commit:** `feat(auth): implement argon2id password hasher [SPRINT-004 #9]`

### 10. [impl] Implement adapters/auth/jwt_issuer.go and jwks_validator.go

- **Agente:** @developer
- **Description:** Implement `TokenIssuer` and `TokenValidator`.

  **`jwt_issuer.go`** — `JWTIssuer` struct with `*rsa.PrivateKey` and
  config (`issuer`, `audience`, `ttl`):
  1. Build `jwt.MapClaims` with all fields from ADR-012.
  2. Sign with `jwt.SigningMethodRS256`.
  3. Serialize with `token.SignedString(privateKey)`.

  **`jwks_validator.go`** — `JWKSValidator` struct:
  - In test/dev mode: accepts a `*rsa.PublicKey` directly.
  - In production mode: lazy HTTP fetch + cache with configurable TTL
    (default 5 min) from the `JWKS_URL` endpoint.
  - `Validate()`: parses JWT with `jwt.ParseWithClaims`, verifies
    `exp`, `iss`, `aud`. Returns `*domain.Claims`.

  **`jwks_endpoint.go`** — function that generates the JWKS JSON:
  - Converts `*rsa.PublicKey` to JWK (kid, kty, alg, n, e).
  - `kid` = first 8 bytes of SHA-256 of the DER encoding of the key.

- **Acceptance criteria:** `go test ./tests/unit/auth/...`
  (all tests in `jwt_test.go`) turns GREEN. `make lint` clean.
- **Depends on:** #6
- **Commit:** `feat(auth): implement JWT RS256 issuer and JWKS validator [SPRINT-004 #10]`

### 11. [impl] Implement RegisterUser and LoginUser use cases

- **Agente:** @developer
- **Description:** Use cases in `services/auth-server/internal/usecase/`.

  **`RegisterUser.Execute(ctx, Credentials) (*domain.User, error)`:**
  1. Validate email (RFC format) and password (≥ 12 chars).
     If invalid → `domain.ErrValidation`.
  2. Verify the email does not exist in `UserRepository`.
     If exists → `domain.ErrConflict`.
  3. Hash password with `PasswordHasher.Hash()`.
  4. Create `domain.User` with new UUID, email, hash, empty tags.
  5. Persist with `UserRepository.Create()`.
  6. Return the created user (without `PasswordHash` in the response —
     the handler filters it).

  **`LoginUser.Execute(ctx, Credentials) (*domain.TokenPair, error)`:**
  1. Find user by email. If not found → `domain.ErrNotFound`
     (but the handler returns a generic 401 — without revealing whether the email exists).
  2. Verify password with `PasswordHasher.Verify()`. If fails → sentinel
     error `ErrInvalidCredentials`.
  3. Issue JWT with `TokenIssuer.Issue()`.
  4. Return `TokenPair{AccessToken, "Bearer", 3600}`.

  New `ErrInvalidCredentials` in `libs/domain/errors.go`.

- **Acceptance criteria:** `go test ./tests/unit/usecase/...`
  (if use case tests are added) passes. `LoginUser` logic never
  reveals whether the email exists or not (both return the same generic error
  to the handler).
- **Depends on:** #4, #9, #10
- **Commit:** `feat(auth): implement RegisterUser and LoginUser use cases [SPRINT-004 #11]`

### 12. [impl] Implement Gin handlers and auth-server router

- **Agente:** @developer
- **Description:** Handlers in `services/auth-server/internal/handler/auth.go`
  following ADR-006.

  **`Register`** handler:
  - `ShouldBindJSON` → `RegisterInput`.
  - Calls `RegisterUser.Execute()`.
  - Maps `ErrConflict` → 409, `ErrValidation` → 422.
  - Returns 201 with `UserResponse` (filters `PasswordHash`).

  **`Login`** handler:
  - Calls `LoginUser.Execute()`.
  - Maps `ErrInvalidCredentials` → 401 with `code=INVALID_CREDENTIALS`.
  - Returns 200 with `TokenResponse`.

  **`JWKS`** handler:
  - Serves the pre-computed JWKS endpoint JSON at startup.
  - Does not require authentication. Content-Type: `application/json`.

  **`services/auth-server/internal/router/router.go`:**
  ```go
  func NewRouter(authHandler *handler.AuthHandler, jwksJSON []byte) *gin.Engine {
      r := gin.New()
      r.Use(gin.Recovery(), middlewareLogger(), middlewareRequestID())

      // Standard OAuth route — outside /api/v1/
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

- **Acceptance criteria:** `go test ./tests/unit/handler/...` and
  `go test -tags=contract ./tests/contract/...` turn GREEN.
  `make lint` clean.
- **Depends on:** #7, #8, #11
- **Commit:** `feat(auth): implement auth-server Gin handlers and router [SPRINT-004 #12]`

### 13. [impl] Implement JWT middleware in the orchestrator

- **Agente:** @developer
- **Description:** Create `services/orchestrator/internal/middleware/auth.go`.

  Configuration from env vars (`AUTH_REQUIRED`, `JWKS_URL`,
  `JWT_ISSUER`, `JWT_AUDIENCE`).

  Logic:
  1. If `AUTH_REQUIRED=false` → `c.Next()` directly.
  2. Extract `Authorization: Bearer <token>` header. If missing → 401.
  3. Call `TokenValidator.Validate(ctx, token)`. If error → 401.
  4. `c.Set("claims", claims)`.
  5. `c.Next()`.

  Helper for handlers: `ClaimsFromContext(c *gin.Context) (*domain.Claims, bool)`.

  The middleware is **not applied** to SPRINT-003 routes yet
  (that is SPRINT-005). It is registered in the router but protects
  only an empty `/api/v1/protected` group (to verify it compiles and tests pass).

- **Acceptance criteria:** `go test ./tests/unit/middleware/...`
  turns GREEN. The middleware compiles and is registered in the orchestrator
  router without affecting existing routes.
- **Depends on:** #8, #10
- **Commit:** `feat(orchestrator): add JWT validation middleware [SPRINT-004 #13]`

### 14. [impl] auth-server main.go and Ent user adapter

- **Agente:** @developer
- **Description:** Two artifacts to complete the wiring.

  **`adapters/auth/ent_user_repo.go`** — `EntUserRepository` that
  implements `ports.UserRepository` using the Ent client. Translates
  `*ent.User` ↔ `*domain.User`. Never exposes `*ent.User` outside the
  adapter.

  **`services/auth-server/cmd/main.go`** — full wiring:
  1. Load RSA keypair from `JWT_PRIVATE_KEY_PATH` /
     `JWT_PUBLIC_KEY_PATH`. If they don't exist, generate a pair in memory
     and log a warning.
  2. Connect to PostgreSQL (`DATABASE_URL`), create `ent.Client`.
  3. Build `EntUserRepository`, `Argon2idHasher`, `JWTIssuer`
     (with `JWT_ISSUER`, `JWT_AUDIENCE=dago-api`, TTL=3600s).
  4. Build use cases.
  5. Build handlers with pre-computed `jwksJSON`.
  6. Build and start router on `AUTH_PORT` (default 8081).
  7. Graceful shutdown signal handling.

- **Acceptance criteria:** `go build ./services/auth-server/...`
  compiles. The binary starts and responds:
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
- **Depends on:** #11, #12, #13, #3
- **Commit:** `feat(auth): wire up auth-server with Ent user repo and RSA keypair [SPRINT-004 #14]`

### 15. [test] End-to-end auth integration test

- **Agente:** @qa
- **Description:** Test with real server and PostgreSQL.
  Build tag `integration`.

  ```go
  //go:build integration

  // TestAuthFlowIntegration verifies the complete flow:
  // 1. Register user.
  // 2. Login → obtain JWT.
  // 3. Parse JWT and verify claims (sub=user ID, scope, attrs.tags empty).
  // 4. Validate JWT against the JWKS of the same server.
  // 5. Verify that login with wrong password returns 401.
  func TestAuthFlowIntegration(t *testing.T)
  ```

- **Acceptance criteria:** `make test-integration` passes with this
  test. Fails if PostgreSQL is not running.
- **Depends on:** #14
- **Commit:** `test(integration): add end-to-end auth flow integration test [SPRINT-004 #15]`

### 16. [docs] Update docs/index.md and docs/log.md

- **Agente:** @docs
- **Description:** Add SPRINT-004 to the sprints table. Update
  the "Domain — Ent Schemas" section marking User and OrgUnit as
  planned in SPRINT-004. Add "Implemented services" section
  mentioning the basic auth-server and the orchestrator middleware.
  Update `docs/log.md`.
- **Acceptance criteria:** `docs/index.md` reflects the new state.
- **Depends on:** #15
- **Commit:** `docs(auth): update index with SPRINT-004 results [SPRINT-004 #16]`

## Traceability Matrix

| Spec / ADR | Rule | TODO | Artifact | Verified by |
|------------|------|------|----------|-------------|
| ADR-012 rule 2 | argon2id for local passwords | #9 | `adapters/auth/argon2id.go` | `TestHash*`, `TestVerify*` |
| ADR-012 rule 3 | JWT RS256 with ABAC attrs | #10 | `adapters/auth/jwt_issuer.go` | `TestIssueProducesValidJWT` |
| ADR-012 rule 4 | Local validation via JWKS | #10, #13 | `jwks_validator.go`, middleware | `TestValidateAcceptsValidJWT` |
| ADR-012 rule 6 | Tokens in memory, never localStorage | #13 | middleware injects in ctx | code review |
| ADR-012 rule 10 | Do not store passwords in backend API | #2, #11 | `Sensitive()`, `UserResponse` without hash | `TestRegisterContract` (no password_hash) |
| ADR-010 rule 1 | `/api/v1/` prefix for business routes | #12 | auth-server router | contract tests |
| ADR-010 rule 8 | Standard `ErrorResponse` format | #12 | `mapDomainError` auth | `TestLogin*` |
| ADR-006 rule 1 | Thin handlers | #12 | handler/auth.go | `make lint` (funlen) |
| ADR-007 rule 7 | Ent types do not leave the adapter | #14 | `ent_user_repo.go` | code review |
| ADR-007 rule 8 | UUIDs + TIMESTAMPTZ | #2 | `user.go`, `org_unit.go` | `\d users` PostgreSQL |
| ADR-001 rule 1 | Domain without infrastructure | #4 | `libs/domain/` without crypto | `go build` |
| ADR-001 rule 2 | Ports as interfaces | #4 | `libs/ports/auth.go` | compiler |
| ADR-002 | TDD Red → Green | #5–#8 before #9–#13 | TODO order | CI |
| ADR-003 rule 2 | Functions ≤ 20 lines | all #impl | all files | `make lint` |
| OWASP argon2id | m=64MB, t=3, p=4 | #9 | `argon2id.go` constants | `TestHashProducesValidPHCString` |

## Sprint acceptance criteria

```bash
# 1. Valid spec
swagger-cli validate specs/openapi.yaml

# 2. Full compilation
go build ./...

# 3. Clean linter
make lint

# 4. Unit tests (no Docker)
go test ./tests/unit/auth/...
go test ./tests/unit/handler/...
go test ./tests/unit/middleware/...

# 5. Contract tests (with fakes, no Docker)
go test -tags=contract ./tests/contract/...

# 6. Integration tests (with Docker)
make test-integration

# 7. Full manual flow
make docker-up
go run ./services/auth-server/cmd/main.go &
# register, login, inspect JWKS
```

Additionally:
- `libs/domain/` does not import `golang.org/x/crypto` or `golang-jwt/jwt`.
- `UserResponse` (spec + handler) does not include `password_hash`.
- `LoginUser` returns the same error for non-existent email and wrong password
  (without revealing account existence).
- The orchestrator middleware with `AUTH_REQUIRED=false` does not break
  any SPRINT-003 test.
- `argon2id.go` uses `subtle.ConstantTimeCompare` in `Verify()`.
- JWKS contains exactly one key with `kty=RSA`, `alg=RS256`.

## Sprint result

_Completed on sprint close._

### Tests run

- Total: —
- Passed: —
- Failed: —

### Files created/modified

_Generated on close._

### Decisions made during the sprint

_Any unforeseen decision requiring an ADR or note is documented here._

### Reviewer notes

_Pending review._
