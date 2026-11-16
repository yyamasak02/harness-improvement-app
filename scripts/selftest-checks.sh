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

# status.sh（再開用の状況把握）: 一時リポジトリ + gh のスタブで「次にすべきこと」の判定表を検証する
sr="$tmp/sr"; mkdir -p "$sr/scripts" "$tmp/bin" "$tmp/origin.git"
cp scripts/status.sh scripts/review-verdict.sh "$sr/scripts/"
git init -q --bare "$tmp/origin.git"
(cd "$sr" && git init -q -b main && git config user.email t@example.com && git config user.name t \
  && echo a > a.txt && git add -A && git commit -qm "chore: base" && git remote add origin "$tmp/origin.git" \
  && git push -q -u origin main && git switch -q -c feat/x && git push -q -u origin feat/x)
HEAD_SHA=$(git -C "$sr" rev-parse HEAD)
# gh スタブ: シナリオ（GH_SCENARIO）ごとに status.sh が呼ぶ gh の出力を返す
cat > "$tmp/bin/gh" <<'STUB'
#!/usr/bin/env bash
args="$*"
case "$GH_SCENARIO" in
  authfail) echo "HTTP 401: Bad credentials" >&2; exit 1 ;;
esac
case "$args" in
  "repo view --json owner --jq .owner.login") echo owner; exit 0 ;;
  "pr view --json number --jq .number")
    if [ "$GH_SCENARIO" = nopr ]; then echo "no pull requests found for branch \"feat/x\"" >&2; exit 1; fi
    if [ "$GH_SCENARIO" = prerror ]; then echo "error connecting to api.github.com" >&2; exit 1; fi
    echo 7; exit 0 ;;
  "pr view 7 --json state --jq .state") [ "$GH_SCENARIO" = merged ] && echo MERGED || echo OPEN; exit 0 ;;
  "pr view 7 --json title --jq .title") echo "テスト PR"; exit 0 ;;
  "pr view 7 --json headRefOid --jq .headRefOid") echo "$GH_HEAD"; exit 0 ;;
  "pr view 7 --json comments,reviews")
    case "$GH_SCENARIO" in
      approve) m="<!-- review-agent: verdict=APPROVE sha=$GH_HEAD -->" ;;
      reqchg)  m="<!-- review-agent: verdict=REQUEST_CHANGES sha=$GH_HEAD -->" ;;
      *) m="" ;;
    esac
    printf '{"comments":[{"author":{"login":"owner"},"createdAt":"2026-01-01T00:00:00Z","body":"%s"}],"reviews":[]}' "$m"; exit 0 ;;
  "pr checks 7")
    case "$GH_SCENARIO" in
      cifail) printf 'lint\tfail\t1s\turl\ntest\tpass\t1s\turl\n' ;;
      cipend) printf 'lint\tpass\t1s\turl\ntest\tpending\t0\turl\n' ;;
      *) printf 'lint\tpass\t1s\turl\ntest\tpass\t1s\turl\n' ;;
    esac; exit 0 ;;
esac
if [ "$GH_SCENARIO" = issues ] && [ "${args#issue list}" != "$args" ]; then
  echo '[{"number":21,"title":"feat: v0.2","labels":[{"name":"priority/p2"},{"name":"release/v0.2"}]},{"number":51,"title":"refactor: v0.1","labels":[{"name":"priority/p2"},{"name":"release/v0.1"}]},{"number":25,"title":"chore: human","labels":[{"name":"priority/p1"},{"name":"status/needs-human"},{"name":"release/v0.1"}]}]'; exit 0
fi
exit 0   # pr list / issue list などは空
STUB
chmod +x "$tmp/bin/gh"
status_next() { # シナリオ名 -> 「次にすべきこと」の行
  (cd "$sr" && PATH="$tmp/bin:$PATH" GH_SCENARIO="$1" GH_HEAD="$HEAD_SHA" bash scripts/status.sh --brief 2>&1) | sed -n '/次にすべきこと/,$p' | grep '→' | head -1
}
expect_next() { # シナリオ名 期待する部分文字列 説明
  local got; got=$(status_next "$1")
  case "$got" in *"$2"*) echo "ok: 次の行動の判定（$3）" ;; *) echo "NG: 次の行動の判定（$3）: got '$got'"; fail=1 ;; esac
}
expect_not_next() {
  local got; got=$(status_next "$1")
  case "$got" in *"$2"*) echo "NG: 次の行動の判定（$3）で '$2' を指示してはいけない: '$got'"; fail=1 ;; *) echo "ok: 次の行動の判定（$3）で '$2' を指示しない" ;; esac
}
expect_next nopr "PR が無い" "ブランチはあるが PR が無い"
expect_next authfail "判定は不能" "gh が認証失敗"
expect_not_next authfail "PR を作成" "gh が認証失敗"
expect_next prerror "判定不能" "PR の有無を取得できない"
expect_not_next prerror "PR を作成" "PR の有無を取得できない"
expect_next cifail "CI が失敗" "CI 失敗"
expect_next cipend "CI が実行中" "CI 実行中"
expect_next none "独立レビューが無い" "head への独立レビューが無い"
expect_next reqchg "REQUEST_CHANGES" "REQUEST_CHANGES"
expect_next approve "merge-pr.sh" "APPROVE かつ CI 成功"
expect_next merged "マージ済み" "PR がマージ済み"
expect_not_next merged "merge-pr.sh" "PR がマージ済み"
# main がクリーンなときの次の issue: needs-human を除き、優先度 → v0.1 を先 → 番号
(cd "$sr" && git switch -q main)
expect_next issues "#51" "候補は同優先度なら v0.1 を先（#21 より #51）"
expect_not_next issues "#25" "needs-human の issue は候補にしない"
(cd "$sr" && git switch -q feat/x)
# 未コミットの変更・未 push のコミット（gh の状態より先に判定される）
echo b > "$sr/b.txt"
expect_next approve "未コミットの変更" "未コミットの変更がある"
rm -f "$sr/b.txt"
(cd "$sr" && echo c > c.txt && git add -A && git commit -qm "chore: c")
expect_next approve "未 push" "未 push のコミットがある"
# gh が無い・オフライン（push 済みのクリーンな状態で、gh の状態が判定不能になること）
(cd "$sr" && git push -q)
out=$(cd "$sr" && STATUS_NO_GH=1 bash scripts/status.sh 2>&1) && case "$out" in *"== Git"*"判定不能"*) echo "ok: gh が使えなくても動き、判定不能と示す" ;; *) echo "NG: gh 無しの出力が不正"; fail=1 ;; esac || { echo "NG: gh 無しで status.sh が失敗"; fail=1; }
# timeout / gtimeout が無い環境（macOS 相当）でも動く
mkdir -p "$tmp/notimeout"; for c in bash git sed awk grep wc tr head tail cat sort python3 date dirname env cut basename mkdir; do p=$(command -v $c) && ln -sf "$p" "$tmp/notimeout/$c"; done
cp "$tmp/bin/gh" "$tmp/notimeout/gh"
out=$(cd "$sr" && PATH="$tmp/notimeout" GH_SCENARIO=approve GH_HEAD="$HEAD_SHA" bash scripts/status.sh --brief 2>&1)
case "$out" in *"独立レビューの判定（この head に対して）: APPROVE"*) echo "ok: timeout コマンドが無くても gh 情報を取得できる" ;; *) echo "NG: timeout が無いと gh 情報が取れない"; fail=1 ;; esac
# 実リポジトリでも主要セクションが出る
out=$(STATUS_NO_GH=1 bash scripts/status.sh --brief 2>&1) && for sec in "== Git" "== docs/tasks.md" "== 次にすべきこと"; do
  case "$out" in *"$sec"*) echo "ok: status.sh に '$sec' がある" ;; *) echo "NG: status.sh に '$sec' が無い"; fail=1 ;; esac
done

# check-version / changelog-notes / check-changelog（リリース整備）
cv="$tmp/cv"; mkdir -p "$cv/scripts" "$cv/frontend"; cp scripts/check-version.sh scripts/changelog-notes.sh "$cv/scripts/"
printf '{"info":{"productVersion":"1.2.3"}}\n' > "$cv/wails.json"; printf '{"version":"1.2.3"}\n' > "$cv/frontend/package.json"
printf '# Changelog\n\n## [Unreleased]\n\n## [1.2.3] - 2026-01-01\n\n### 追加\n- 機能 A\n\n## [1.2.2] - 2025-12-01\n- 旧\n' > "$cv/CHANGELOG.md"
(cd "$cv" && bash scripts/check-version.sh v1.2.3 >/dev/null 2>&1) && echo "ok: 整合したバージョンとタグは通る" || { echo "NG: 正しいバージョンが拒否された"; fail=1; }
(cd "$cv" && ! bash scripts/check-version.sh v1.2.4 >/dev/null 2>&1) && echo "ok: タグとバージョンの不一致を検出" || { echo "NG: タグ不一致を検出できない"; fail=1; }
(cd "$cv" && ! bash scripts/check-version.sh 1.2.3 >/dev/null 2>&1) && echo "ok: v 無しのタグ形式を検出" || { echo "NG: タグ形式の誤りを検出できない"; fail=1; }
printf '{"version":"1.2.0"}\n' > "$cv/frontend/package.json"
(cd "$cv" && ! bash scripts/check-version.sh >/dev/null 2>&1) && echo "ok: wails.json と package.json の不一致を検出" || { echo "NG: バージョン不一致を検出できない"; fail=1; }
printf '{"version":"1.2.3"}\n' > "$cv/frontend/package.json"; printf '{"info":{"productVersion":"1.2"}}\n' > "$cv/wails.json"
(cd "$cv" && ! bash scripts/check-version.sh >/dev/null 2>&1) && echo "ok: SemVer でないバージョンを検出" || { echo "NG: SemVer 違反を検出できない"; fail=1; }
printf '{"info":{"productVersion":"1.2.3"}}\n' > "$cv/wails.json"; sed -i.bak 's/^## \[1.2.3\].*$/## [9.9.9] - x/' "$cv/CHANGELOG.md"
(cd "$cv" && ! bash scripts/check-version.sh v1.2.3 >/dev/null 2>&1) && echo "ok: CHANGELOG に節が無いタグを検出" || { echo "NG: CHANGELOG の節欠落を検出できない"; fail=1; }
sed -i.bak 's/^## \[9.9.9\].*$/## [1.2.3] - 2026-01-01/' "$cv/CHANGELOG.md"
notes=$(cd "$cv" && bash scripts/changelog-notes.sh 1.2.3)
case "$notes" in *"機能 A"*) case "$notes" in *"旧"*) echo "NG: 次の節まで含めている"; fail=1 ;; *) echo "ok: changelog-notes は該当節だけを出す" ;; esac ;; *) echo "NG: changelog-notes が節を取れない"; fail=1 ;; esac
cc="$tmp/cc"; mkdir -p "$cc/scripts"; cp scripts/check-changelog.sh "$cc/scripts/"
(cd "$cc" && git init -q -b main && git config user.email t@example.com && git config user.name t && echo '# c' > CHANGELOG.md && echo a > a.txt && git add -A && git commit -qm "chore: base" \
  && git switch -q -c f && echo b >> a.txt && git commit -qam "feat: x")
(cd "$cc" && ! PR_TITLE="feat: x" bash scripts/check-changelog.sh main >/dev/null 2>&1) && echo "ok: feat で CHANGELOG 未更新を検出" || { echo "NG: CHANGELOG 未更新を検出できない"; fail=1; }
(cd "$cc" && PR_TITLE="feat: x" PR_BODY="[no-changelog]: 内部のみ" bash scripts/check-changelog.sh main >/dev/null 2>&1) && echo "ok: [no-changelog] で免除" || { echo "NG: 免除が効かない"; fail=1; }
(cd "$cc" && PR_TITLE="docs: x" bash scripts/check-changelog.sh main >/dev/null 2>&1) && echo "ok: docs は対象外" || { echo "NG: docs が対象になっている"; fail=1; }
(cd "$cc" && echo '- 追加' >> CHANGELOG.md && git commit -qam "feat: y" && PR_TITLE="feat: x" bash scripts/check-changelog.sh main >/dev/null 2>&1) && echo "ok: CHANGELOG を更新すれば通る" || { echo "NG: 更新しても通らない"; fail=1; }

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
