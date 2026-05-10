# ADR-012: Custom OAuth 2.1 + tag-based ABAC authorization

**Status:** Accepted (revised: independent auth-server, full ABAC)
**Date:** 2026-04-20
**Authors:** [Architecture team]

## Context

Dago needs authentication (users + M2M) and fine-grained attribute-based
authorization. The auth-server is an independent service (ADR-013).

## Decision

### Authentication: auth-server with OAuth 2.1

The **auth-server** service acts as the OAuth 2.1 Authorization Server
and Identity Broker for the dago ecosystem.

```
auth-server
├── OAuth 2.1 endpoints (/authorize, /token, /revoke, /introspect, JWKS)
├── Identity Broker (external IdP: Google, Azure AD, OIDC, LDAP; or local)
├── Local user and credential management (argon2id)
├── Client management (services, MCP servers, agents)
├── Organizational units (tree with tag inheritance)
└── ABAC engine (tag-based access evaluation)
```

### Flow 1: User (Authorization Code + PKCE)

```
Dashboard → auth-server → external IdP (or local login)
         ← authorization_code
         → code + code_verifier
         ← access_token (JWT) + refresh_token
         → Bearer token on each API request
```

### Flow 2: Machine-to-machine (Client Credentials)

```
Service → POST /token (client_id + client_secret)
        ← access_token (JWT)
```

### JWT tokens

```json
{
  "iss": "https://auth.dago.example.com",
  "sub": "user_550e8400-...",
  "aud": "dago-api",
  "scope": "graphs:read graphs:execute",
  "attrs": {
    "tags": ["department:engineering", "clearance:internal"],
    "org_unit": "ou_engineering_backend",
    "org_path": "/company/engineering/backend"
  },
  "client_type": "user"
}
```

Local validation with JWKS (`/.well-known/jwks.json`). No
introspection on each request.

### Token propagation

The token travels in Valkey events (the `auth` field of the envelope,
ADR-011) all the way to the MCP servers:

```
Dashboard → orchestrator → events → executor → mcp-registry → MCP server
                                                                   │
                                                           auth-server (JWKS)
```

### Authorization: tag-based ABAC

**Tag model:**

```
Subjects (who):
├── Users       → own tags + inherited from OU
└── Services/MCP → assigned tags

Resources (what):
├── Graphs/Packages → required tags
├── Executions      → tags inherited from the graph
└── MCP Servers     → assigned tags

Operations (how):
└── Scopes: read, execute, manage, admin
```

**Organizational units (tree with inheritance):**

```
/company                           tags: [env:production]
├── /company/engineering           tags: [department:engineering]
│   ├── /company/engineering/backend    tags: [team:backend]
│   └── /company/engineering/frontend   tags: [team:frontend]
└── /company/finance               tags: [department:finance, data:sensitive]
```

Effective tags = own tags ∪ tags inherited from the entire OU chain.

**Access rule:**

```
ALLOW if:
  token scopes cover the operation
  AND
  resource tags ⊆ subject's effective tags
```

**Sharing:** Add tags to the resource or to the recipient.

### Ent schemas (data model)

```go
// ent/schema/org_unit.go
func (OrgUnit) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID{}).Default(uuid.New),
        field.String("name"),
        field.String("path").Unique(),
        field.Strings("tags"),
    }
}

func (OrgUnit) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("children", OrgUnit.Type),
        edge.From("parent", OrgUnit.Type).Ref("children").Unique(),
        edge.To("users", User.Type),
    }
}
```

### Ports

```go
// libs/ports/auth.go
type TokenValidator interface {
    Validate(ctx context.Context, token string) (*Claims, error)
}

type Authorizer interface {
    CanAccess(ctx context.Context, subject *Claims, resource Resource, op string) (bool, error)
    EffectiveTags(ctx context.Context, subjectID uuid.UUID) ([]string, error)
}

type Claims struct {
    Subject    string
    ClientType string       // "user" or "service"
    Scopes     []string
    Attrs      AttributeSet
}

type AttributeSet struct {
    Tags    []string
    OrgUnit string
    OrgPath string
}
```

### Concrete rules

1. Two flows: Authorization Code + PKCE (users), Client Credentials (M2M).
2. Identity Broker: external IdPs or local domain (argon2id).
3. JWT signed (RS256 or EdDSA) with ABAC attrs.
4. Local validation via JWKS. No introspection per request.
5. Refresh tokens: mandatory rotation (one-time use).
6. Dashboard: tokens in memory, never localStorage.
7. ABAC: resource tags ⊆ subject's effective tags.
8. Tag inheritance through the OU tree.
9. Token propagation in events (`auth` field).
10. No passwords stored in the backend API. Only the auth-server
    manages credentials.

### JWT signing key management

The auth-server requires an RSA private key (`jwt_private.pem`) for RS256 signing.
Key management differs by environment:

| Environment | Approach |
|-------------|----------|
| **Local / docker-compose** | `secrets-init` init container generates the key pair in `./secrets/` if absent. `auth-server` declares `depends_on: secrets-init: condition: service_completed_successfully`. |
| **Production** | Keys come from an external secret manager (Vault, K8s Secrets, AWS SM). The `secrets-init` container is disabled via `AUTO_GENERATE_SECRETS=false`. Keys are NEVER auto-generated. |

Rules:
- `secrets/` is listed in `.gitignore`. Private keys are NEVER committed.
- The init container only generates the key if `jwt_private.pem` is absent — it never overwrites an existing key.
- Key rotation in production follows the secret manager's rotation policy.

## Notes for Claude Code

- auth-server in `services/auth-server/internal/`.
  - `oauth/` — OAuth 2.1 endpoints.
  - `identity/` — IdP broker, local domain.
  - `abac/` — ABAC engine, OUs, tag evaluation.
- `TokenValidator` and `Authorizer` in `libs/ports/auth.go`.
- JWT/JWKS implementation in `adapters/auth/`.
- Ent schemas: OrgUnit, User with tags fields.
- Tokens propagated in the `auth` field of the event envelope.
