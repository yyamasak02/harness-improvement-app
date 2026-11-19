#!/usr/bin/env bash
# 改善提案（scout skill）用の計測。読み取り専用。常時ロードされる文脈のサイズなどを出す。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

size() { if [ -f "$1" ]; then wc -c <"$1" | tr -d ' '; else echo 0; fi; }

echo "== 常時ロードされる文脈（バイト。日本語は 1 文字 3 バイトなので、トークン換算は概ね 1/2〜1/3）"
claude_md=$(size CLAUDE.md)
imports=0
while read -r f; do
  [ -n "$f" ] || continue
  s=$(size "$f")
  imports=$((imports + s))
  printf '  CLAUDE.md の @参照 %-28s %6d\n' "$f" "$s"
done < <(grep -oE '@docs/[A-Za-z0-9_./-]+' CLAUDE.md | sed 's/^@//' | sort -u)
rules=0
for f in .claude/rules/*.md; do rules=$((rules + $(size "$f"))); done
status=$(bash scripts/status.sh 2>/dev/null | wc -c | tr -d ' ')
printf '  %-40s %6d\n' "CLAUDE.md 本体" "$claude_md"
printf '  %-40s %6d\n' ".claude/rules/*.md 合計" "$rules"
printf '  %-40s %6d\n' "status.sh 出力（SessionStart）" "$status"
printf '  %-40s %6d\n' "合計（概算）" $((claude_md + imports + rules + status))

echo "== 大きいファイル（行数の上位 5。分割・可読性の候補）"
git ls-files 'scripts/*' '*.go' '*.ts' '*.tsx' | xargs wc -l 2>/dev/null | grep -vE ' (total|合計)$' | sort -rn | head -5

echo "== TODO / FIXME（コード内。issue 化されていない課題の目印）"
git grep -nE 'TODO|FIXME' -- '*.go' '*.ts' '*.tsx' | head -10 || true
