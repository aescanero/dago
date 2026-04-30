# ADR-012: OAuth 2.1 propio + autorización ABAC por etiquetas

**Estado:** Aceptado (revisado: auth-server independiente, ABAC completo)
**Fecha:** 2026-04-20
**Autores:** [Equipo de arquitectura]

## Contexto

Dago necesita autenticación (usuarios + M2M) y autorización granular
basada en atributos. El auth-server es un servicio independiente (ADR-013).

## Decisión

### Autenticación: auth-server con OAuth 2.1

El servicio **auth-server** actúa como Authorization Server OAuth 2.1
e Identity Broker del ecosistema dago.

```
auth-server
├── OAuth 2.1 endpoints (/authorize, /token, /revoke, /introspect, JWKS)
├── Identity Broker (IdP externo: Google, Azure AD, OIDC, LDAP; o local)
├── Gestión de usuarios y credenciales locales (argon2id)
├── Gestión de clientes (servicios, MCP servers, agentes)
├── Unidades organizativas (árbol con herencia de tags)
└── Motor ABAC (evaluación de acceso por etiquetas)
```

### Flujo 1: Usuario (Authorization Code + PKCE)

```
Dashboard → auth-server → IdP externo (o login local)
         ← authorization_code
         → code + code_verifier
         ← access_token (JWT) + refresh_token
         → Bearer token en cada request a la API
```

### Flujo 2: Machine-to-machine (Client Credentials)

```
Servicio → POST /token (client_id + client_secret)
         ← access_token (JWT)
```

### Tokens JWT

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

Validación local con JWKS (`/.well-known/jwks.json`). Sin
introspection en cada request.

### Propagación de tokens

El token viaja en los eventos Valkey (campo `auth` del envelope,
ADR-011) hasta los MCP servers:

```
Dashboard → orchestrator → eventos → executor → mcp-registry → MCP server
                                                                    │
                                                            auth-server (JWKS)
```

### Autorización: ABAC por etiquetas

**Modelo de etiquetas (tags):**

```
Sujetos (quién):
├── Usuarios       → tags propias + heredadas de UO
└── Servicios/MCP  → tags asignadas

Recursos (qué):
├── Grafos/Paquetes → tags requeridas
├── Ejecuciones     → tags heredadas del grafo
└── MCP Servers     → tags asignadas

Operaciones (cómo):
└── Scopes: read, execute, manage, admin
```

**Unidades organizativas (árbol con herencia):**

```
/company                           tags: [env:production]
├── /company/engineering           tags: [department:engineering]
│   ├── /company/engineering/backend    tags: [team:backend]
│   └── /company/engineering/frontend   tags: [team:frontend]
└── /company/finance               tags: [department:finance, data:sensitive]
```

Tags efectivas = tags propias ∪ tags heredadas de toda la cadena de UO.

**Regla de acceso:**

```
PERMITIR si:
  scopes del token cubren la operación
  Y
  tags del recurso ⊆ tags efectivas del sujeto
```

**Compartir:** Añadir tags al recurso o al destinatario.

### Schemas Ent (modelo de datos)

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

### Puertos

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
    ClientType string       // "user" o "service"
    Scopes     []string
    Attrs      AttributeSet
}

type AttributeSet struct {
    Tags    []string
    OrgUnit string
    OrgPath string
}
```

### Reglas concretas

1. Dos flujos: Authorization Code + PKCE (usuarios), Client Credentials (M2M).
2. Identity Broker: IdPs externos o dominio local (argon2id).
3. JWT firmados (RS256 o EdDSA) con attrs ABAC.
4. Validación local vía JWKS. Sin introspection por request.
5. Refresh tokens: rotación obligatoria (one-time use).
6. Dashboard: tokens en memoria, nunca localStorage.
7. ABAC: tags recurso ⊆ tags efectivas sujeto.
8. Herencia de tags por árbol de UOs.
9. Propagación de tokens en eventos (campo `auth`).
10. No se almacenan contraseñas en el backend API. Solo el auth-server
    gestiona credenciales.

## Notas para Claude Code

- auth-server en `services/auth-server/internal/`.
  - `oauth/` — endpoints OAuth 2.1.
  - `identity/` — IdP broker, dominio local.
  - `abac/` — motor ABAC, UOs, evaluación de tags.
- `TokenValidator` y `Authorizer` en `libs/ports/auth.go`.
- Implementación JWT/JWKS en `adapters/auth/`.
- Schemas Ent: OrgUnit, User con campos tags.
- Tokens propagados en campo `auth` del envelope de eventos.
