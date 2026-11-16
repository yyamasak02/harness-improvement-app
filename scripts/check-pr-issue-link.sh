#!/usr/bin/env bash
# PR 本文が issue を参照していること（Closes/Fixes/Resolves/Refs #N）、または `[no-issue]: 理由` を持つことを検査する。
#   環境変数: PR_BODY（必須）、PR_AUTHOR（dependabot[bot] は免除）
set -euo pipefail
body="${PR_BODY:-}"
[ "${PR_AUTHOR:-}" = "dependabot[bot]" ] && { echo "check-pr-issue-link: dependabot は免除"; exit 0; }
if printf '%s' "$body" | grep -qiE '(^|[^[:alnum:]_])(close[sd]?|fix(e[sd])?|resolve[sd]?|refs?|related to)[: ]+#[0-9]+'; then
  echo "check-pr-issue-link OK"; exit 0
fi
if printf '%s' "$body" | grep -qE '\[no-issue\]: .+'; then
  echo "check-pr-issue-link: [no-issue] の宣言あり"; exit 0
fi
echo "PR 本文に関連 issue が必要です: 'Closes #N' / 'Refs #N'、無い場合は '[no-issue]: 理由'。"
echo "探し方: bash scripts/related-issues.sh"
exit 1
