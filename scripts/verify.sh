#!/usr/bin/env bash
# 検証を一括実行。存在するレイヤーのみ。
set -euo pipefail
cd "$(dirname "$0")/.."
if [ -f go.mod ]; then
  echo "== go vet"; go vet ./...
  echo "== go test"; go test ./...
fi
if [ -f frontend/package.json ]; then
  echo "== frontend"; (cd frontend && npm run build && npm test)
fi
echo "verify OK"
