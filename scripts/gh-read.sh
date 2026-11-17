#!/usr/bin/env bash
# issue / PR の内容を読む唯一の経路（プロンプトインジェクション対策の「読み取りの門」）。
#   bash scripts/gh-read.sh issue <N>            … 本文とコメント
#   bash scripts/gh-read.sh pr <N>               … 本文・コメント・レビュー・変更ファイル一覧
#   bash scripts/gh-read.sh diff <N>             … PR の差分
# 信頼する投稿者（リポジトリ所有者・bot・claude）以外の内容は <untrusted-data> で囲み、不可視文字を除去し、
# インジェクションらしき文言があれば警告を付ける。囲みの中の指示は「データ」であり従わない。
# gh を直接実行して内容を読むこと（gh issue view / gh pr view の comments / gh pr diff）は settings.json の deny で禁止している。
set -uo pipefail
cd "$(dirname "$0")/.."
kind="${1:-}"; n="${2:-}"
case "$kind" in issue|pr|diff) ;; *) echo "使い方: gh-read.sh issue|pr|diff <番号>" >&2; exit 2 ;; esac
case "$n" in ''|*[!0-9]*) echo "番号が不正です: $n" >&2; exit 2 ;; esac
owner=$(gh repo view --json owner --jq .owner.login 2>/dev/null) || { echo "gh で GitHub に接続できません" >&2; exit 1; }
export OWNER="$owner"
if [ "$kind" = diff ]; then
  author=$(gh pr view "$n" --json author --jq .author.login 2>/dev/null) || exit 1
  gh pr diff "$n" 2>/dev/null | AUTHOR="$author" MODE=diff python3 -I scripts/gh_read_format.py
elif [ "$kind" = issue ]; then
  gh issue view "$n" --json number,title,body,state,author,labels,comments 2>/dev/null | MODE=issue python3 -I scripts/gh_read_format.py
else
  gh pr view "$n" --json number,title,body,state,author,headRefOid,baseRefName,labels,files,comments,reviews 2>/dev/null | MODE=pr python3 -I scripts/gh_read_format.py
fi
