#!/usr/bin/env bash
# 独立レビューの承認（現在の head に対する APPROVE マーカー）と CI 成功を確認してから squash マージする。
#   使い方: scripts/merge-pr.sh [PR番号]   （省略時は現在のブランチの PR）
set -euo pipefail
cd "$(dirname "$0")/.."
n="${1:-$(gh pr view --json number --jq .number)}"
sha=$(gh pr view "$n" --json headRefOid --jq .headRefOid)
verdict=$(gh pr view "$n" --json comments,reviews | bash scripts/review-verdict.sh "$sha")
if [ "$verdict" != "APPROVE" ]; then
  echo "マージできません: head ${sha:0:7} に対する独立レビューの判定が '$verdict' です（APPROVE が必要）。review-pr skill に従ってください。"
  exit 1
fi
echo "レビュー: APPROVE（head ${sha:0:7}）。CI を待ちます…"
gh pr checks "$n" --watch --interval 20 --fail-fast || { echo "CI が成功していません。マージしません。"; exit 1; }
[ "$(gh pr view "$n" --json headRefOid --jq .headRefOid)" = "$sha" ] || { echo "待機中に head が更新されました。再レビューが必要です。"; exit 1; }
gh pr merge "$n" --squash --delete-branch
git switch main
git pull -q
