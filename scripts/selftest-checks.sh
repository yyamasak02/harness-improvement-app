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
printf 'package internal\n\nimport "syscall"\n\nvar _ = syscall.SIGKILL\n' > internal/zz_violation.go
expect_fail "OS 非依存ファイルでの Unix 専用 syscall の検出" bash scripts/check-rules.sh
printf 'package internal\n\nimport "os/exec"\n\nvar _ = exec.Command("xdg-open", "x")\n' > internal/zz_violation.go
expect_fail "osutil 外での OS 依存コマンド起動の検出" bash scripts/check-rules.sh
rm -f internal/zz_violation.go
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
expect_fail "単語の一部（prefix #3）を issue 参照と誤認しない" env PR_BODY="prefix #3 を修正" bash scripts/check-pr-issue-link.sh
expect_fail "理由なしの [no-issue] の検出" env PR_BODY="[no-issue]:" bash scripts/check-pr-issue-link.sh
PR_AUTHOR="dependabot[bot]" PR_BODY="" bash scripts/check-pr-issue-link.sh >/dev/null 2>&1 && echo "ok: dependabot は免除" || { echo "NG: dependabot が免除されない"; fail=1; }

# review-verdict（独立レビューのマーカー解釈）
S1=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa; S2=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
A=owner,github-actions
mk() { # login createdAt body -> コメント JSON 要素
  python3 -I -c 'import json,sys;print(json.dumps({"author":{"login":sys.argv[1]},"createdAt":sys.argv[2],"body":sys.argv[3]}))' "$@"; }
mj() { printf '{"comments":[%s],"reviews":[]}' "$(IFS=,; echo "$*")"; }
verdict() { printf '%s' "$1" | bash scripts/review-verdict.sh "$2" "$A"; }
expect_eq() { [ "$1" = "$2" ] && echo "ok: $3" || { echo "NG: $3（got $1, want $2）"; fail=1; }; }
M_OK="<!-- review-agent: verdict=APPROVE sha=$S1 -->"; M_RC="<!-- review-agent: verdict=REQUEST_CHANGES sha=$S1 -->"
expect_eq "$(verdict "$(mj "$(mk owner 2026-01-01T00:00:00Z 'ただのコメント')")" $S1)" NONE "マーカー無しは NONE"
expect_eq "$(verdict "$(mj "$(mk owner 2026-01-01T00:00:00Z "x"$'\n'"$M_OK")")" $S1)" APPROVE "APPROVE を認識"
expect_eq "$(verdict "$(mj "$(mk owner 2026-01-01T00:00:00Z "$M_OK")")" $S2)" NONE "head が進んだら過去の承認は無効"
expect_eq "$(verdict "$(mj "$(mk owner 2026-01-01T00:00:00Z "$M_OK")" "$(mk owner 2026-01-02T00:00:00Z "$M_RC")")" $S1)" REQUEST_CHANGES "時系列で最新の判定が優先される"
expect_eq "$(verdict "$(mj "$(mk owner 2026-01-02T00:00:00Z "$M_RC")" "$(mk owner 2026-01-01T00:00:00Z "$M_OK")")" $S1)" REQUEST_CHANGES "配列の順ではなく日時で並べる"
expect_eq "$(verdict "$(mj "$(mk mallory 2026-01-01T00:00:00Z "$M_OK")")" $S1)" NONE "許可されていない投稿者のマーカーは無視"
expect_eq "$(verdict "$(mj "$(mk owner 2026-01-01T00:00:00Z "書式の例: $M_OK"$'\n'"続き")")" $S1)" NONE "最終行でないマーカー（引用）は無視"

# status.sh（再開用の状況把握）: gh が使えなくても落ちず、主要セクションを出す
out=$(bash scripts/status.sh --brief 2>&1) && echo "ok: status.sh --brief が正常終了" || { echo "NG: status.sh --brief が失敗"; fail=1; }
for sec in "== Git" "== docs/tasks.md" "== 次にすべきこと"; do
  case "$out" in *"$sec"*) echo "ok: status.sh に '$sec' がある" ;; *) echo "NG: status.sh に '$sec' が無い"; fail=1 ;; esac
done
out=$(STATUS_NO_GH=1 bash scripts/status.sh 2>&1) && case "$out" in *"== Git"*) echo "ok: gh が無くても status.sh は動く" ;; *) echo "NG: gh 無しで出力なし"; fail=1 ;; esac || { echo "NG: gh 無しで status.sh が失敗"; fail=1; }

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
