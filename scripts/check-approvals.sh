#!/usr/bin/env bash
# 境界ファイルを変更する PR に、第一承認者（別エージェント）の評価の記録があることを検査する（変更の追跡可能性）。
#   bash scripts/check-approvals.sh range <base>     （CI の pr-policy・pre-push から呼ぶ）
# 評価は scripts/record-approval.sh が docs/approval-log.md に追記する（コミットされ、PR の差分に載る）。
# 境界ファイルの定義は scripts/hooks/pre_tool_guard.py の PROTECTED と同じ。
set -uo pipefail
cd "$(dirname "$0")/.."
[ "${1:-}" = range ] || { echo "使い方: check-approvals.sh range <base>" >&2; exit 2; }
base="${2:-origin/main}"
changed=$(git diff --name-only --diff-filter=AMR "$base"...HEAD 2>/dev/null || true)
[ -n "$changed" ] || { echo "check-approvals OK（変更なし）"; exit 0; }
prot=$(printf '%s\n' "$changed" | python3 -I -c '
import sys
sys.path.insert(0, "scripts/hooks")
import pre_tool_guard as G
for l in sys.stdin.read().split():
    if G.is_protected(l) and l != "docs/approval-log.md":
        print(l)
')
[ -n "$prot" ] || { echo "check-approvals OK（境界ファイルの変更なし）"; exit 0; }
added=$(git diff -U0 "$base"...HEAD -- docs/approval-log.md 2>/dev/null | grep '^+|' || true)
fail=0
while IFS= read -r f; do
  if printf '%s\n' "$added" | grep -F "| $f |" | grep -q "| APPROVE |"; then
    echo "ok: $f（第一承認者の評価あり）"
  else
    echo "NG: $f は境界ファイルですが、この PR に第一承認者（change-approver）の APPROVE の記録がありません（docs/approval-log.md）"; fail=1
  fi
done <<< "$prot"
[ $fail -eq 0 ] && echo "check-approvals OK"
exit $fail
