# SPRINT-011: Working docker-compose — full stack containerization

## Metadata

- **Start date:** 2026-05-09
- **Estimated end date:** 2026-05-12
- **Status:** planned
- **Applied ADRs:** ADR-004, ADR-005, ADR-006, ADR-007, ADR-008, ADR-009, ADR-013, ADR-020
- **Affected specs:** —
- **Planning agent:** planner
- **Reviewed by:** pending
- **Blocked by:** —
- **Blocks:** —

## Objective

Produce a fully working `docker-compose.yml` that starts the complete dago stack
(2 infrastructure services + 8 Go backend services + 1 React dashboard) with a
single command. All services must pass their health checks, communicate over the
shared network, and be configurable via `.env`.

## Audit findings

The current `docker-compose.yml` only declares `postgres` (pgvector/pgvector:pg16)
and `valkey` (valkey/valkey:8). No Dockerfiles exist for any service or the
dashboard. Key gaps:

1. No Dockerfiles for the 8 Go services or the dashboard.
2. Application services absent from `docker-compose.yml`.
3. No `.env.example` documenting all required variables.
4. No database migration step before services start.
5. No `vm.overcommit_memory=1` host-requirement note anywhere.
6. `auth-server` requires a mounted RSA private key (`JWT_PRIVATE_KEY_PATH`).
7. Dashboard `VITE_*` build-time env vars not documented for container build.
8. No Makefile targets for compose operations.

## Scope

- **Dockerfiles** — one shared multi-stage Dockerfile for all 8 Go services
  (build arg `SERVICE`), plus a dedicated Dockerfile for the dashboard.
- **docker-compose.yml** — extended with 4 compose profiles: `infra`,
  `backend`, `frontend`, `all`. Services wire via `dago-network`.
- **Migration init-container** — `migrate` service (atlas) that runs
  `atlas migrate apply` before orchestrator starts.
- **`.env.example`** — all required and optional variables with defaults and comments.
- **Makefile targets** — `compose-up`, `compose-down`, `compose-infra`,
  `compose-logs`, `compose-ps`.
- **Health checks** — HTTP `/health` for Gin services; build-time check for dashboard.
- **Runbook** — `docs/deploy/docker-compose-runbook.md` with prerequisites,
  quick-start, profiles, secrets, troubleshooting.
- **Docs update** — `docs/index.md` SPRINT-011 entry, `docs/log.md` closing entry.

## Dependencies

- **Blocked by:** none (infrastructure-only sprint, no code dependencies on other sprints).
- **Blocks:** any sprint that requires a running local stack for integration tests
  (SPRINT-007 integration tests, SPRINT-010 integration tests, etc.).

## TODOs

### TODO #1 — audit: map all env vars per service [audit]

**Agent:** @devops

**Objective:** Read each `services/*/cmd/main.go` and `dashboard/vite.config.ts`
and produce a complete list of all `os.Getenv` / `VITE_*` variables consumed.
Use the result as the input for TODO #3 and TODO #4.

Services to audit: `orchestrator`, `executor`, `router`, `planner`, `auth-server`,
`catalog`, `mcp-registry`, `agent-registry`, plus the dashboard.

Output: annotated list grouped by service with type (required / optional), default
value if any, and description. Keep as an in-sprint working note; do not create a
separate file.

**Files:** all `services/*/cmd/main.go`, `dashboard/src/`, `dashboard/vite.config.ts`

---

### TODO #2 — impl: shared Go Dockerfile [impl]

**Agent:** @devops

**Objective:** Create a single multi-stage Dockerfile for all 8 Go services.

```dockerfile
# Dockerfile.service
ARG SERVICE
# Stage 1: build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service ./services/${SERVICE}/cmd/

# Stage 2: run
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/service /bin/service
ENTRYPOINT ["/bin/service"]
```

Place at repo root as `Dockerfile.service`. The `SERVICE` build arg selects which
service to compile. Distroless/static keeps the image minimal and secure.

**File:** `Dockerfile.service`

---

### TODO #3 — impl: dashboard Dockerfile [impl]

**Agent:** @devops

**Objective:** Create a multi-stage Dockerfile for the React dashboard.

```dockerfile
# dashboard/Dockerfile
# Stage 1: build
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_URL=http://localhost:8080
ARG VITE_AUTH_URL=http://localhost:8081
ARG VITE_CLIENT_ID=dashboard
RUN npm run build

# Stage 2: serve
FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

Also add `dashboard/nginx.conf` with `try_files $uri /index.html` for SPA routing.

**Files:** `dashboard/Dockerfile`, `dashboard/nginx.conf`

---

### TODO #4 — impl: .env.example [impl]

**Agent:** @devops

**Objective:** Create `.env.example` at repo root documenting every variable
needed by all services and the dashboard build. Group by service. Include
defaults, type (required/optional), and one-line description.

Minimum variables (expand based on TODO #1 findings):

```dotenv
# --- Infrastructure ---
POSTGRES_USER=dago
POSTGRES_PASSWORD=dago
POSTGRES_DB=dago
POSTGRES_PORT=5432
VALKEY_PORT=6379

# --- orchestrator ---
DATABASE_URL=postgres://dago:dago@postgres:5432/dago?sslmode=disable
PORT=8080                        # HTTP port

# --- auth-server ---
AUTH_PORT=8081
JWT_PRIVATE_KEY_PATH=/secrets/jwt_private.pem   # RSA-2048, mounted as volume
JWT_PUBLIC_KEY_PATH=/secrets/jwt_public.pem

# --- catalog ---
CATALOG_PORT=8082
CATALOG_DATABASE_URL=postgres://dago:dago@postgres:5432/dago?sslmode=disable

# --- mcp-registry ---
MCP_PORT=8083

# --- agent-registry ---
AGENT_REGISTRY_PORT=8084

# --- executor ---
EXECUTOR_VALKEY_URL=valkey:6379
ANTHROPIC_API_KEY=sk-ant-...     # required for llm_call nodes

# --- router ---
ROUTER_VALKEY_URL=valkey:6379

# --- planner ---
PLANNER_VALKEY_URL=valkey:6379

# --- dashboard build args ---
VITE_API_URL=http://localhost:8080
VITE_AUTH_URL=http://localhost:8081
VITE_CLIENT_ID=dashboard
```

**File:** `.env.example`

---

### TODO #5 — impl: complete docker-compose.yml [impl]

**Agent:** @devops

**Objective:** Extend `docker-compose.yml` to include all 9 application services
plus a `migrate` init-container. Use compose profiles so users can start only
what they need.

Profiles:
- `infra` — `postgres`, `valkey`
- `backend` — all 8 Go services + `migrate` (implies `infra`)
- `frontend` — `dashboard`
- `all` (default profile) — everything

Key requirements:
- All Go services use `build: {context: ., dockerfile: Dockerfile.service, args: {SERVICE: <name>}}`.
- `migrate` service uses `arigaio/atlas:latest`, runs `atlas migrate apply` against
  `DATABASE_URL`, exits 0. Has `depends_on: postgres: condition: service_healthy`.
- `orchestrator` has `depends_on: migrate: condition: service_completed_successfully`.
- All backend services have `depends_on: valkey: condition: service_healthy`.
- Dashboard `build.args` pass `VITE_API_URL`, `VITE_AUTH_URL`, `VITE_CLIENT_ID`
  from `.env`.
- `auth-server` mounts `./secrets:/secrets:ro` for the JWT key files.
- Port mappings use env var defaults (e.g. `${PORT:-8080}:8080`).
- All services on `dago-network`.

**File:** `docker-compose.yml`

---

### TODO #6 — impl: health endpoints for Go services [impl]

**Agent:** @developer

**Objective:** Each Gin service must expose `GET /health` returning `{"status":"ok"}`
with HTTP 200, so Docker can probe it.

Services: `orchestrator`, `auth-server`, `catalog`, `mcp-registry`, `agent-registry`.
(Executor, router, planner are event-driven workers — use a TCP-based health check
`test: ["CMD-SHELL", "wget -qO- http://localhost:${PORT}/health || exit 1"]` or
simply skip HTTP health and rely on process liveness with restart policies.)

The endpoint must be registered before any auth middleware (no token required).

**Files:** each `services/{name}/cmd/main.go` or its router file

---

### TODO #7 — impl: Makefile docker-compose targets [impl]

**Agent:** @devops

**Objective:** Add convenience targets to `Makefile`.

```makefile
.PHONY: compose-infra compose-up compose-down compose-logs compose-ps

compose-infra: ## Start only postgres + valkey
	docker compose --profile infra up -d

compose-up: ## Start full stack (all profiles)
	docker compose --profile all up -d --build

compose-down: ## Stop and remove all containers
	docker compose --profile all down

compose-logs: ## Tail logs for all running services
	docker compose logs -f

compose-ps: ## Show status of all containers
	docker compose ps
```

**File:** `Makefile`

---

### TODO #8 — test: smoke test — all services healthy [test]

**Agent:** @qa

**Objective:** Write a shell smoke-test script that:
1. Runs `docker compose --profile all up -d --build`.
2. Waits up to 120 s for all services to be healthy (`docker compose ps --format json`).
3. Sends `GET /health` to each HTTP service and asserts HTTP 200.
4. Publishes a ping event to Valkey and reads it back.
5. Prints a PASS/FAIL summary and exits non-zero on failure.
6. Runs `docker compose --profile all down` as cleanup.

**File:** `scripts/smoke-test-compose.sh` (chmod +x)

---

### TODO #9 — docs: docker-compose runbook [docs]

**Agent:** @docs

**Objective:** Create `docs/deploy/docker-compose-runbook.md` covering:

- **Prerequisites:** Docker 24+, Docker Compose v2.24+, host kernel setting
  (`sysctl vm.overcommit_memory=1`), RSA key generation for auth-server.
- **Quick start:** copy `.env.example` → `.env`, fill secrets, `make compose-up`.
- **Profiles:** table of profiles and what each starts.
- **Secrets:** how to generate the JWT RSA key pair.
- **Ports:** table of service → host port mapping.
- **Troubleshooting:** common errors (valkey overcommit, postgres not ready,
  atlas migration failure, dashboard blank page CORS).
- **Stopping:** `make compose-down`, data volumes.

**File:** `docs/deploy/docker-compose-runbook.md`

---

### TODO #10 — docs: update index and log [docs]

**Agent:** @docs

**Objective:** Update project-wide documentation on sprint close.

- `docs/index.md` — add SPRINT-011 entry (status: completed).
- `docs/log.md` — append closing entry with date 2026-05-12 and artifacts delivered.

**Files:** `docs/index.md`, `docs/log.md`

---

## Traceability Matrix

| TODO | Spec | Test | Impl | Docs |
|------|------|------|------|------|
| #1 Env-var audit | — | — | (working note) | — |
| #2 Dockerfile.service | ADR-013 | — | Dockerfile.service | — |
| #3 dashboard/Dockerfile | ADR-009 | — | dashboard/Dockerfile, nginx.conf | — |
| #4 .env.example | ADR-013 | — | .env.example | — |
| #5 docker-compose.yml | ADR-006, ADR-007, ADR-008, ADR-009 | — | docker-compose.yml | — |
| #6 /health endpoints | ADR-006 | — | services/*/cmd/main.go | — |
| #7 Makefile targets | ADR-005 | — | Makefile | — |
| #8 Smoke test | — | scripts/smoke-test-compose.sh | — | — |
| #9 Runbook | — | — | — | docs/deploy/docker-compose-runbook.md |
| #10 Docs update | — | — | — | docs/index.md, docs/log.md |

## Key decisions

- **Single `Dockerfile.service`** with `SERVICE` build arg avoids duplicating 8
  nearly-identical Dockerfiles and keeps the build context at repo root (monorepo, ADR-013).
- **Distroless/static** for Go services: minimal attack surface, no shell.
- **Atlas init-container** (`migrate` service) ensures schema migrations run before
  the orchestrator boots, preventing startup races.
- **Compose profiles** (`infra` / `backend` / `frontend` / `all`) let developers
  start only the layers they need without editing the file.
- **`./secrets` volume mount** for JWT keys: keeps private key out of the image
  and out of `.env`, consistent with ADR-012.
- **`vm.overcommit_memory=1`** documented as a host prerequisite in the runbook;
  not enforced in compose (requires host-level sysctl and privilege escalation).
- **Health endpoints unprotected** (`GET /health` bypasses auth middleware) to allow
  Docker and load balancers to probe without tokens.

## Result

> _Complete on sprint close._

- TODOs completed: —/10
- Tests passing: —
- Decisions reviewed: —
- Artifacts delivered: —
