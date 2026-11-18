#!/usr/bin/env bash
# 境界ファイルを変更する PR に、第一承認者（別エージェント）の評価の記録があることを検査する（変更の追跡可能性）。
#   bash scripts/check-approvals.sh range <base>     （CI の pr-policy・pre-push から呼ぶ）
# 評価は scripts/record-approval.sh が docs/approval-log.md に追記する（コミットされ、PR の差分に載る）。
# 台帳の行は「評価した内容のハッシュ（sha:…）」を持ち、PR の最終的な内容と一致する行だけを有効とする。
# 境界ファイルの定義は scripts/hooks/pre_tool_guard.py の PROTECTED と同じ。差分を取得できないときは失敗にする（素通りさせない）。
set -uo pipefail
cd "$(dirname "$0")/.."
[ "${1:-}" = range ] || { echo "使い方: check-approvals.sh range <base>" >&2; exit 2; }
base="${2:-origin/main}"
git rev-parse --verify -q "$base" >/dev/null || { echo "NG: 比較元 $base が見つかりません（差分を取得できないため、境界ファイルの承認を確認できません）"; exit 1; }
changed=$(git diff --name-only --diff-filter=AMR "$base"...HEAD) || { echo "NG: git diff に失敗しました（境界ファイルの承認を確認できません）"; exit 1; }
[ -n "$changed" ] || { echo "check-approvals OK（変更なし）"; exit 0; }
prot=$(printf '%s\n' "$changed" | python3 -I -c '
import sys
sys.path.insert(0, "scripts/hooks")
import pre_tool_guard as G
for l in sys.stdin.read().split():
    if G.is_protected(l) and l != "docs/approval-log.md":
        print(l)
') || { echo "NG: 境界ファイルの判定に失敗しました"; exit 1; }
[ -n "$prot" ] || { echo "check-approvals OK（境界ファイルの変更なし）"; exit 0; }
added=$(git diff -U0 "$base"...HEAD -- docs/approval-log.md | grep '^+|' || true)
fail=0
while IFS= read -r f; do
  want=$(git show "HEAD:$f" 2>/dev/null | python3 -I -c 'import sys; sys.path.insert(0, "scripts/hooks"); import approval_lib as A; print(A.sha(sys.stdin.buffer.read().decode("utf-8", "replace"))[:12])') || want=""
  if [ -n "$want" ] && printf '%s\n' "$added" | grep -F "| $f |" | grep "| APPROVE |" | grep -qF "sha:$want"; then
    echo "ok: $f（第一承認者の評価あり・内容のハッシュ一致 $want）"
  elif printf '%s\n' "$added" | grep -F "| $f |" | grep -q "| APPROVE |"; then
    echo "NG: $f は、評価の後に内容が変わっています（台帳のハッシュが現在の内容 $want と一致しません）。評価をやり直して記録してください"; fail=1
  else
    echo "NG: $f は境界ファイルですが、この PR に第一承認者（change-approver）の APPROVE の記録がありません（docs/approval-log.md）"; fail=1
  fi
done <<< "$prot"
[ $fail -eq 0 ] && echo "check-approvals OK"
exit $fail
