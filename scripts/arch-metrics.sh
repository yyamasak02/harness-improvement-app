#!/usr/bin/env bash
# アーキテクチャの健全性の計測（refactor-campaign 用）。読み取り専用。
# 層の違反の許容リストの件数・パッケージ数・巨大ファイル・time.Now の直接呼び出しを出す。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

echo "== 許容リスト（依存の向きの違反。目標は 0。増やすのは禁止）"
go_cap=$(grep -hE '^\s*(const\s+)?allowlistCap\s*=' internal/archtest/*_test.go 2>/dev/null | head -1 | grep -oE '[0-9]+' | head -1 || true)
fe_cap=$(grep -hE 'ALLOWLIST_CAP\s*=' frontend/src/arch.test.ts 2>/dev/null | head -1 | grep -oE '[0-9]+' | head -1 || true)
printf '  Go（internal/archtest allowlistCap）      %s\n' "${go_cap:-不明}"
printf '  フロント（arch.test.ts ALLOWLIST_CAP）    %s\n' "${fe_cap:-不明}"

echo "== Go のパッケージ数（テスト補助を除く）"
n=$(find internal -name '*.go' ! -name '*_test.go' -exec dirname {} \; 2>/dev/null | sort -u | grep -vE 'testutil|archtest' | wc -l | tr -d ' ')
printf '  internal 配下 %s（ほかに main）\n' "$n"

echo "== 巨大ファイル（非テストの Go と TS/TSX。行数の上位 8。分割の候補）"
git ls-files '*.go' '*.ts' '*.tsx' | grep -vE '_test\.go$|\.test\.tsx?$|wailsjs/|\.e2e\.ts$|\.ux\.ts$' \
  | xargs wc -l 2>/dev/null | grep -vE ' (total|合計)$' | sort -rn | head -8 | sed 's/^/  /'

echo "== time.Now() の直接呼び出し（非テストの Go。注入に置き換える候補。目標は main のみ）"
git grep -nE 'time\.Now\(\)' -- '*.go' ':!*_test.go' | sed 's/^/  /' || true

echo "== os.UserHomeDir の直接呼び出し（非テストの Go。main のみが望ましい）"
git grep -nE 'os\.UserHomeDir\(' -- '*.go' ':!*_test.go' | sed 's/^/  /' || true
