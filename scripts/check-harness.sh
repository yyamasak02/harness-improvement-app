#!/usr/bin/env bash
# 安全装置（フック・権限）が実環境で有効かの診断。無効・旧版・壊れていれば失敗する（終了コード 1）。
#   bash scripts/check-harness.sh            … 診断（実環境の settings.json のフックを実際に動かす）
#   bash scripts/check-harness.sh --quiet    … 正常時は何も出さない（SessionStart・post-checkout 用）
# 安全装置を入れる・変えたときは、selftest だけでなくこれを通して「完了」とする（quality.md）。
# 自動実行: SessionStart（status.sh）・ブランチ切り替え後（post-checkout）・pre-push・CI。
set -uo pipefail
cd "$(dirname "$0")/.."
exec python3 -I scripts/check_harness.py "$@"
