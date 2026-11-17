#!/usr/bin/env bash
# 第一承認者（別エージェント change-approver）の評価を記録する。
#   bash scripts/record-approval.sh <変更対象のパス> <APPROVE|REJECT> <loosen|tighten|neutral|unknown> <要約のファイル> <評価した変更後の全文のファイル>
# 記録は <git-dir>/harness-approvals/ に保存される（コミットされない。有効 30 分・評価した内容にのみ有効）。
# PreToolUse フック（scripts/hooks/pre_tool_guard.py）が、境界ファイルの変更前にこの記録を確認する。
# 要約は引数ではなくファイルで渡す（フックが記録コマンドを「安全な文字だけの引数」に限って通すため）。
# 信頼モデル: これを実行してよいのは change-approver だけ。実装者が自分で記録することは禁止
# （記録は目印であり、最終承認者は人。確認画面にこの要約が表示される）。
set -euo pipefail
cd "$(dirname "$0")/.."
[ $# -eq 5 ] || { echo "使い方: record-approval.sh <path> <APPROVE|REJECT> <loosen|tighten|neutral|unknown> <要約のファイル> <評価した変更後の全文のファイル>" >&2; exit 2; }
path="$1"; verdict="$2"; risk="$3"; summary_file="$4"; proposed="$5"
[ -f "$summary_file" ] || { echo "要約のファイルがありません: $summary_file" >&2; exit 2; }
summary=$(cat "$summary_file")
case "$path" in *[!A-Za-z0-9_./@:+-]*) echo "path に使えない文字があります" >&2; exit 2 ;; esac
[ -f "$proposed" ] || { echo "評価した変更後の全文のファイルがありません: $proposed" >&2; exit 2; }
case "$verdict" in APPROVE|REJECT) ;; *) echo "判定は APPROVE か REJECT" >&2; exit 2 ;; esac
case "$risk" in loosen|tighten|neutral|unknown) ;; *) echo "影響は loosen|tighten|neutral|unknown" >&2; exit 2 ;; esac
# 要約にも個人情報検査をかける（確認画面・ログに残るため）
printf '%s\n' "$summary" | bash scripts/check-privacy.sh text >/dev/null || { echo "要約に個人情報らしきものがあります" >&2; exit 1; }
python3 -I - "$path" "$verdict" "$risk" "$summary" "$proposed" <<'PY'
import sys
sys.path.insert(0, "scripts/hooks")
import approval_lib as A
with open(sys.argv[5], encoding="utf-8", errors="replace") as fh:
    content_sha = A.sha(fh.read())
rec = A.write_record(sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4], content_sha=content_sha)
print("記録しました: %s %s（影響: %s）有効 %d 分" % (rec["path"], rec["verdict"], rec["risk"], A.TTL_SECONDS // 60))
PY
# 追跡用の台帳（コミットされ、PR の差分に載る。CI の check-approvals.sh が境界ファイルの変更ごとに確認する）
log=docs/approval-log.md
if [ ! -f "$log" ]; then
  printf '# 境界ファイルの変更の承認記録\n\n第一承認者（別エージェント change-approver）の評価の台帳。`scripts/record-approval.sh` が追記する（手で編集しない）。\n人の最終承認は、PR のレビューとマージで行う。\n\n| 日時（UTC） | 対象 | 判定 | 影響 | 要約 |\n|---|---|---|---|---|\n' > "$log"
fi
oneline=$(printf '%s' "$summary" | tr '\n\r' '  ' | sed 's/|/／/g')
printf '| %s | %s | %s | %s | %s |\n' "$(date -u +%Y-%m-%dT%H:%M)" "$(python3 -I -c 'import sys; sys.path.insert(0, "scripts/hooks"); import approval_lib as A; print(A.norm(sys.argv[1]))' "$path")" "$verdict" "$risk" "$oneline" >> "$log"
echo "台帳に追記しました: $log（PR に含めてコミットしてください）"
