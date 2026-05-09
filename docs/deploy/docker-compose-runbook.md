# Docker Compose Runbook — dago full stack

## Prerequisites

| Requirement | Minimum version |
|-------------|----------------|
| Docker Engine | 24.0 |
| Docker Compose plugin | v2.24 |
| Available RAM | 4 GB (8 GB recommended) |
| `jq` | any (for smoke-test script) |

### Host kernel setting (Linux)

Valkey requires the overcommit memory setting to avoid data loss warnings:

```bash
# Apply immediately (lost on reboot)
sudo sysctl -w vm.overcommit_memory=1

# Persist across reboots
echo "vm.overcommit_memory = 1" | sudo tee /etc/sysctl.d/99-valkey.conf
sudo sysctl -p /etc/sysctl.d/99-valkey.conf
```

### Generate JWT RSA key pair (auth-server)

The `auth-server` reads its private key from `./secrets/jwt_private.pem`.
If `JWT_PRIVATE_KEY_PATH` is unset or the file is missing, the service falls
back to an in-memory key (dev only — keys regenerate on restart, invalidating
all tokens).

```bash
mkdir -p secrets
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 \
  -out secrets/jwt_private.pem
chmod 600 secrets/jwt_private.pem
```

> `secrets/` is listed in `.gitignore` — never commit private keys.

---

## Quick start

```bash
# 1. Copy and fill in the environment file
cp .env.example .env
# Edit .env: set ANTHROPIC_API_KEY and POSTGRES_PASSWORD at minimum.

# 2. Generate JWT key pair
mkdir -p secrets
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 \
  -out secrets/jwt_private.pem && chmod 600 secrets/jwt_private.pem

# 3. Start the full stack (builds images on first run)
make compose-up

# 4. Verify
make compose-ps
curl http://localhost:8080/health   # orchestrator
curl http://localhost:8081/health   # auth-server
```

---

## Profiles

| Profile | Command | Services started |
|---------|---------|-----------------|
| `infra` | `make compose-infra` | postgres, valkey |
| `backend` | `docker compose --profile backend up -d` | infra + migrate + 8 Go services |
| `frontend` | `docker compose --profile frontend up -d` | dashboard |
| `all` | `make compose-up` | everything |

You can combine profiles: `docker compose --profile backend --profile frontend up -d`.

---

## Ports

| Service | Host port | Container port |
|---------|-----------|---------------|
| postgres | 5432 | 5432 |
| valkey | 6379 | 6379 |
| orchestrator | 8080 | 8080 |
| auth-server | 8081 | 8081 |
| catalog | 8082 | 8082 |
| mcp-registry | 8083 | 8083 |
| agent-registry | 8084 | 8084 |
| dashboard | 3000 | 80 |

Ports can be overridden in `.env` (e.g. `PORT=9080`).

---

## Secrets

| Variable | File | Notes |
|----------|------|-------|
| `JWT_PRIVATE_KEY_PATH` | `./secrets/jwt_private.pem` | RSA-2048 PEM; mounted read-only |
| `ANTHROPIC_API_KEY` | `.env` | Required for llm_call nodes with anthropic provider |
| `POSTGRES_PASSWORD` | `.env` | Change from default `dago` in production |

---

## Stopping

```bash
make compose-down          # stops and removes containers, keeps volumes
docker compose down -v     # also removes named volumes (DELETES DATA)
```

---

## Troubleshooting

### Valkey overcommit warning

```
WARNING Memory overcommit must be enabled!
```

Apply the kernel setting from the Prerequisites section.

### postgres: not ready / migrate fails

The `migrate` service depends on postgres being healthy. If postgres fails to
start, check:

```bash
docker compose logs postgres
```

Common causes: port 5432 already in use, insufficient disk space.

### Atlas migration failure

```bash
docker compose logs migrate
```

If the migration directory is empty or out of sync with the schema, run:

```bash
make migrate-diff    # generate new migration file
```

Then rebuild: `make compose-up`.

### Dashboard blank page / CORS errors

The dashboard's `VITE_API_URL` and `VITE_AUTH_URL` are baked in at **build
time**. If you change these values in `.env` after the first build, you must
rebuild the dashboard image:

```bash
docker compose --profile frontend build dashboard
docker compose --profile frontend up -d dashboard
```

For CORS issues, ensure the orchestrator's `AUTH_REQUIRED` and CORS settings
match the dashboard's origin (`http://localhost:3000`).

### auth-server: JWT key not found

Ensure `./secrets/jwt_private.pem` exists and is readable:

```bash
ls -la secrets/
docker compose logs auth-server
```

### Smoke test

Run the automated smoke test to verify the full stack in one command:

```bash
bash scripts/smoke-test-compose.sh
```

The script starts the stack, waits for all health checks, probes each HTTP
service, pings Valkey, then tears down. Exits non-zero on any failure.
