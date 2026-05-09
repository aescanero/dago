#!/usr/bin/env bash
# smoke-test-compose.sh — verifies the full dago stack starts and responds correctly.
# Exits non-zero on any failure.
set -euo pipefail

TIMEOUT=120
POLL=5
PASS=0
FAIL=0

log() { echo "[$(date '+%H:%M:%S')] $*"; }
ok()   { log "PASS: $*"; ((PASS++)); }
fail() { log "FAIL: $*"; ((FAIL++)); }

cleanup() {
  log "cleaning up..."
  docker compose --profile all down --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Step 1: start the stack
# ---------------------------------------------------------------------------
log "starting full stack..."
docker compose --profile all up -d --build

# ---------------------------------------------------------------------------
# Step 2: wait for all services to be healthy / running
# ---------------------------------------------------------------------------
log "waiting up to ${TIMEOUT}s for services to stabilise..."
deadline=$(( $(date +%s) + TIMEOUT ))
while true; do
  unhealthy=$(docker compose ps --format json 2>/dev/null \
    | jq -r 'select(.Health != "" and .Health != "healthy" and .State == "running") | .Service' \
    | tr '\n' ',' || true)
  exited=$(docker compose ps --format json 2>/dev/null \
    | jq -r 'select(.State == "exited" and .ExitCode != 0) | .Service' \
    | tr '\n' ',' || true)

  if [[ -z "$unhealthy" && -z "$exited" ]]; then
    log "all services ready"
    break
  fi

  if (( $(date +%s) >= deadline )); then
    log "timeout waiting for services. unhealthy=[${unhealthy}] exited=[${exited}]"
    docker compose ps
    exit 1
  fi
  sleep "$POLL"
done

# ---------------------------------------------------------------------------
# Step 3: HTTP health checks
# ---------------------------------------------------------------------------
declare -A HTTP_SERVICES=(
  [orchestrator]="http://localhost:8080/health"
  [auth-server]="http://localhost:8081/health"
  [catalog]="http://localhost:8082/health"
  [mcp-registry]="http://localhost:8083/health"
  [agent-registry]="http://localhost:8084/health"
)

for svc in "${!HTTP_SERVICES[@]}"; do
  url="${HTTP_SERVICES[$svc]}"
  status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")
  if [[ "$status" == "200" ]]; then
    ok "GET $url → $status"
  else
    fail "GET $url → $status (expected 200)"
  fi
done

# ---------------------------------------------------------------------------
# Step 4: Valkey ping round-trip
# ---------------------------------------------------------------------------
ping_result=$(docker compose exec -T valkey valkey-cli ping 2>/dev/null || echo "ERROR")
if [[ "$ping_result" == "PONG" ]]; then
  ok "valkey ping → PONG"
else
  fail "valkey ping → $ping_result"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "============================="
echo " PASS: $PASS  FAIL: $FAIL"
echo "============================="

if (( FAIL > 0 )); then
  exit 1
fi
