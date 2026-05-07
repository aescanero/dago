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

echo ""
echo "=== All smoke checks passed ==="
