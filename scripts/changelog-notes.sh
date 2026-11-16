#!/usr/bin/env bash
# CHANGELOG.md の指定バージョンの節（見出し行を除く）を標準出力に出す。Release ノートに使う。
#   scripts/changelog-notes.sh 0.1.0
set -euo pipefail
cd "$(dirname "$0")/.."
v="${1:?バージョン（例: 0.1.0）が必要です}"
awk -v v="$v" '
  $0 ~ "^## \\[" v "\\]" { on=1; next }
  on && /^## \[/ { exit }
  on { print }
' CHANGELOG.md | sed -e :a -e '/^\n*$/{$d;N;ba' -e '}'
