#!/usr/bin/env bash
# 検証を一括実行。存在するレイヤーのみ。
#   scripts/verify.sh          … go vet / go test / フロントの build と test
#   scripts/verify.sh --full   … 上記 + 静的検査・ルール検査・カバレッジ基準（push 前・CI 相当）
set -euo pipefail
cd "$(dirname "$0")/.."
full=0; [ "${1:-}" = "--full" ] && full=1
if [ -f go.mod ]; then
  echo "== go vet"; go vet ./...
  echo "== go test"; go test ./...
fi
if [ -f frontend/package.json ]; then
  echo "== frontend"; (cd frontend && npm run build && npm test)
fi
if [ $full -eq 1 ]; then
  echo "== lint"; bash scripts/lint.sh all
  echo "== check-rules"; bash scripts/check-rules.sh
  echo "== go coverage"; bash scripts/coverage.sh
  echo "== frontend coverage"; (cd frontend && npm run test:coverage)
fi
echo "verify OK"
