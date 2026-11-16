#!/usr/bin/env bash
# 品質ゲート自体のテスト: 違反を意図的に作り、check-rules / check-tests-changed / check-commit-msg が検出できることを確認する。
set -uo pipefail
cd "$(dirname "$0")/.."
tmp=$(mktemp -d); trap 'rm -rf "$tmp"; rm -f internal/zz_violation.go internal/zz_violation_test.go' EXIT
fail=0
expect_fail() { # name, command...
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then echo "NG: $name を検出できていません"; fail=1; else echo "ok: $name"; fi
}

# check-rules: 違反ファイルを置く
cat > internal/zz_violation.go <<'GO'
package internal

import "os"

func bad() { _, _ = os.UserHomeDir(); _ = os.WriteFile("x", nil, 0o600) }
GO
expect_fail "ホーム解決/直接書き込みの検出" bash scripts/check-rules.sh
rm -f internal/zz_violation.go
printf 'package internal\n\nimport "testing"\n\nfunc TestX(t *testing.T) { t.Skip() }\n' > internal/zz_violation_test.go
expect_fail "理由なし t.Skip の検出" bash scripts/check-rules.sh
rm -f internal/zz_violation_test.go
bash scripts/check-rules.sh >/dev/null 2>&1 && echo "ok: 違反なしでは成功" || { echo "NG: クリーンな状態で失敗"; fail=1; }

# check-commit-msg
echo "update stuff" > "$tmp/m1"; echo "feat(ui): 画面を追加" > "$tmp/m2"
expect_fail "Conventional 形式でないメッセージの検出" bash scripts/check-commit-msg.sh "$tmp/m1"
bash scripts/check-commit-msg.sh "$tmp/m2" && echo "ok: 正しいメッセージは通る" || { echo "NG: 正しいメッセージが拒否された"; fail=1; }

# check-pr-issue-link
expect_fail "issue 参照のない PR 本文の検出" env PR_BODY="説明だけ" bash scripts/check-pr-issue-link.sh
PR_BODY="Closes #12" bash scripts/check-pr-issue-link.sh >/dev/null 2>&1 && echo "ok: Closes #N を認識" || { echo "NG: Closes #N を認識できない"; fail=1; }
PR_BODY="Refs #3" bash scripts/check-pr-issue-link.sh >/dev/null 2>&1 && echo "ok: Refs #N を認識" || { echo "NG: Refs #N を認識できない"; fail=1; }
PR_BODY="[no-issue]: 依存更新のみ" bash scripts/check-pr-issue-link.sh >/dev/null 2>&1 && echo "ok: [no-issue] を認識" || { echo "NG: [no-issue] を認識できない"; fail=1; }
expect_fail "理由なしの [no-issue] の検出" env PR_BODY="[no-issue]:" bash scripts/check-pr-issue-link.sh
PR_AUTHOR="dependabot[bot]" PR_BODY="" bash scripts/check-pr-issue-link.sh >/dev/null 2>&1 && echo "ok: dependabot は免除" || { echo "NG: dependabot が免除されない"; fail=1; }

# review-verdict（独立レビューのマーカー解釈）
S1=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa; S2=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
verdict() { printf '%s' "$1" | bash scripts/review-verdict.sh "$2"; }
j_none='{"comments":[{"body":"ただのコメント"}],"reviews":[]}'
j_ok="{\"comments\":[{\"body\":\"x\\n<!-- review-agent: verdict=APPROVE sha=$S1 -->\"}],\"reviews\":[]}"
j_old="{\"comments\":[{\"body\":\"<!-- review-agent: verdict=APPROVE sha=$S1 -->\"}],\"reviews\":[]}"
j_chg="{\"comments\":[{\"body\":\"<!-- review-agent: verdict=APPROVE sha=$S1 -->\"},{\"body\":\"<!-- review-agent: verdict=REQUEST_CHANGES sha=$S1 -->\"}],\"reviews\":[]}"
[ "$(verdict "$j_none" $S1)" = NONE ] && echo "ok: マーカー無しは NONE" || { echo "NG: マーカー無し"; fail=1; }
[ "$(verdict "$j_ok" $S1)" = APPROVE ] && echo "ok: APPROVE を認識" || { echo "NG: APPROVE を認識できない"; fail=1; }
[ "$(verdict "$j_old" $S2)" = NONE ] && echo "ok: head が進んだら過去の承認は無効" || { echo "NG: 古い承認が有効になっている"; fail=1; }
[ "$(verdict "$j_chg" $S1)" = REQUEST_CHANGES ] && echo "ok: 最新の判定が優先される" || { echo "NG: 最新判定が優先されない"; fail=1; }

# check-tests-changed: 一時リポジトリで検証
repo="$tmp/repo"; mkdir -p "$repo/scripts" "$repo/internal/x"; cp scripts/check-tests-changed.sh "$repo/scripts/"
(
  cd "$repo" && git init -q -b main && git config user.email t@example.com && git config user.name t
  echo 'package x' > internal/x/a.go && git add -A && git commit -qm "chore: base"
  git switch -q -c feature && echo 'package x // changed' > internal/x/a.go && git commit -qam "feat: change"
  if bash scripts/check-tests-changed.sh range main >/dev/null 2>&1; then echo "NG: テスト無し変更を検出できていません"; exit 1; else echo "ok: テスト無し変更の検出"; fi
  PR_BODY='[no-test-needed]: 挙動不変のリネーム' bash scripts/check-tests-changed.sh range main >/dev/null 2>&1 \
    && echo "ok: [no-test-needed] で免除" || { echo "NG: 免除が効いていません"; exit 1; }
  echo 'package x' > internal/x/a_test.go && git add -A && git commit -qm "test: add"
  bash scripts/check-tests-changed.sh range main >/dev/null 2>&1 && echo "ok: テストを足すと通る" || { echo "NG: テストを足しても失敗"; exit 1; }
) || fail=1
exit $fail
