#!/usr/bin/env bash
# .github/labels.tsv の内容でラベルを作成・更新する（冪等）。
set -euo pipefail
cd "$(dirname "$0")/.."
grep -v '^#' .github/labels.tsv | while IFS=$'\t' read -r name color desc; do
  [ -n "$name" ] || continue
  gh label create "$name" --color "$color" --description "$desc" --force >/dev/null
  echo "label: $name"
done
