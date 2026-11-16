#!/usr/bin/env bash
# ユーザーに見える変更（feat / fix）の PR は CHANGELOG.md を更新すること。
#   scripts/check-changelog.sh <base-ref>   環境変数: PR_TITLE, PR_BODY, PR_AUTHOR
# 免除: タイトルが feat/fix 以外、dependabot、本文に `[no-changelog]: 理由`
set -euo pipefail
cd "$(dirname "$0")/.."
base="${1:-origin/main}"
title="${PR_TITLE:-}"
[ "${PR_AUTHOR:-}" = "dependabot[bot]" ] && { echo "check-changelog: dependabot は免除"; exit 0; }
if ! [[ "$title" =~ ^(feat|fix)(\(|:|!) ]]; then echo "check-changelog: feat/fix 以外（$title）は対象外"; exit 0; fi
if git diff --name-only "$base"...HEAD | grep -qx 'CHANGELOG.md'; then echo "check-changelog OK"; exit 0; fi
if printf '%s' "${PR_BODY:-}" | grep -qE '\[no-changelog\]: .+'; then echo "check-changelog: [no-changelog] の宣言あり"; exit 0; fi
echo "feat/fix の PR は CHANGELOG.md の [Unreleased] に変更を書いてください（ユーザーに見えない変更なら本文に '[no-changelog]: 理由'）。"
exit 1
