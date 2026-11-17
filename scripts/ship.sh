#!/usr/bin/env bash
# 使い方: scripts/ship.sh "<コミット/PR タイトル>" "<PR 本文（Closes #N・検証結果込み）>"
# 検証 → commit → push → PR 作成（既存 PR があれば push のみ）。**マージはしない。**
# マージは独立レビューの承認後に scripts/merge-pr.sh で行う（review-pr skill 参照）。main 上では実行を拒否する。
set -euo pipefail
cd "$(dirname "$0")/.."
title="$1"; body="$2"
branch=$(git rev-parse --abbrev-ref HEAD)
[ "$branch" != "main" ] || { echo "main 上では実行できません"; exit 1; }
bash scripts/verify.sh --full
bash scripts/related-issues.sh || true
export PR_BODY="$body"   # pre-push の check-tests-changed が [no-test-needed] を読めるように
PR_BODY="$body" bash scripts/check-pr-issue-link.sh   # issue 参照（Closes/Refs #N か [no-issue]）が無ければここで止まる
# 個人情報の検査（コミットメッセージ・PR タイトル/本文。作者メールは range で検査）
printf '%s\n%s\n' "$title" "$body" | bash scripts/check-privacy.sh text
# コミットの作者・コミッターのメールは noreply にする（実メールを履歴に残さない。check-privacy range も検査する）。
# git config は変更せず、環境変数で上書きする。noreply は main の最新コミット（GitHub の squash マージ）から得る。
cfg_email=$(git config user.email 2>/dev/null || true)
if ! printf '%s' "$cfg_email" | grep -qiE '@users\.noreply\.github\.com$'; then
  nr=$(git log origin/main -1 --format='%ae' 2>/dev/null || true)
  if printf '%s' "$nr" | grep -qiE '@users\.noreply\.github\.com$'; then
    nm=$(git log origin/main -1 --format='%an' 2>/dev/null || echo dev)
    export GIT_AUTHOR_NAME="$nm" GIT_COMMITTER_NAME="$nm" GIT_AUTHOR_EMAIL="$nr" GIT_COMMITTER_EMAIL="$nr"
  else
    echo "noreply のメールを決められません（git config user.email が noreply ではなく、origin/main からも得られない）。GIT_AUTHOR_EMAIL / GIT_COMMITTER_EMAIL に noreply を設定してください"; exit 1
  fi
fi
git add -A
git diff --cached --quiet || git commit -q -m "$title

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
SKIP=harness-verify git push -u origin "$branch"   # verify --full は直前に実行済み
if gh pr view --json number >/dev/null 2>&1; then   # --json 無しは Projects (classic) の廃止警告で失敗する
  echo "既存の PR #$(gh pr view --json number --jq .number) に push しました。再レビューを実行してください（review-pr skill）。"
else
  bash scripts/gh-post.sh pr create --title "$title" --body "$body

🤖 Generated with [Claude Code](https://claude.com/claude-code)"
  echo "PR を作成しました。次: 独立レビュー（review-pr skill）→ bash scripts/merge-pr.sh"
fi
