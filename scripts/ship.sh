#!/usr/bin/env bash
# 使い方: scripts/ship.sh "<コミット/PR タイトル>" "<PR 本文（検証結果込み）>"
# 全変更を commit → push → PR 作成 → squash マージ → main 更新。main 上では実行を拒否する。
set -euo pipefail
cd "$(dirname "$0")/.."
title="$1"; body="$2"
branch=$(git rev-parse --abbrev-ref HEAD)
[ "$branch" != "main" ] || { echo "main 上では実行できません"; exit 1; }
bash scripts/verify.sh --full
export PR_BODY="$body"   # pre-push の check-tests-changed が [no-test-needed] を読めるように
git add -A
git diff --cached --quiet || git commit -q -m "$title

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
SKIP=harness-verify git push -u origin "$branch"   # verify --full は直前に実行済み
gh pr create --title "$title" --body "$body

🤖 Generated with [Claude Code](https://claude.com/claude-code)"
gh pr merge --squash --delete-branch
git switch main
git pull -q
