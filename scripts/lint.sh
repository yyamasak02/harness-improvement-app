#!/usr/bin/env bash
# 静的検査。使い方: scripts/lint.sh gofmt|vet|staticcheck|tsc|all
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$PATH:$(go env GOPATH)/bin"
gofmt_check() { out=$(gofmt -l app.go main.go internal); [ -z "$out" ] || { echo "gofmt が必要です:"; echo "$out"; exit 1; }; }
staticcheck_run() {
  command -v staticcheck >/dev/null || { echo "staticcheck が無い: go install honnef.co/go/tools/cmd/staticcheck@latest"; exit 1; }
  staticcheck ./...
}
case "${1:-all}" in
  gofmt) gofmt_check ;;
  vet) go vet ./... ;;
  staticcheck) staticcheck_run ;;
  tsc) (cd frontend && npx tsc --noEmit) ;;
  all) gofmt_check; go vet ./...; staticcheck_run; (cd frontend && npx tsc --noEmit) ;;
  *) echo "usage: $0 gofmt|vet|staticcheck|tsc|all"; exit 2 ;;
esac
