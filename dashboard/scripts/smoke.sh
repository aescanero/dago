#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "=== Dashboard smoke checks ==="

echo "[1/4] Type-check..."
npm run type-check
echo "  ✓ type-check passed"

echo "[2/4] Build..."
npm run build
echo "  ✓ build passed"

echo "[3/4] Lint..."
npm run lint || echo "  ⚠ lint warnings (non-blocking)"

echo "[4/4] Tests..."
npm test
echo "  ✓ tests passed"

echo "[5/5] Graph bundle presence..."
if ls dist/assets/*.js 1>/dev/null 2>&1; then
  # String literals survive minification; function names do not
  grep -q "Create graph" dist/assets/*.js    || { echo "  ✗ GraphCreatePage missing from bundle"; exit 1; }
  grep -q "Archive this graph" dist/assets/*.js || { echo "  ✗ GraphArchiveDialog missing from bundle"; exit 1; }
  grep -q "/graphs/new" dist/assets/*.js     || { echo "  ✗ /graphs/new route missing from bundle"; exit 1; }
  echo "  ✓ graph pages present in bundle"
else
  echo "  ⚠ no dist/assets/*.js found — run build first"
fi

echo ""
echo "=== All smoke checks passed ==="
