#!/usr/bin/env bash
# CLAUDE.md の「やってはいけないこと」のうち機械的に検出できるものを静的に検査する。
set -uo pipefail
cd "$(dirname "$0")/.."
fail=0
report() { echo "NG: $1"; printf '%s\n' "$2" | sed 's/^/  /'; fail=1; }

# 1. internal/ は実ホームを解決しない（パスは注入する。app.go だけが解決してよい）
hits=$(grep -rnE 'os\.UserHomeDir|os\.Getenv\("HOME"\)|user\.Current\(' --include='*.go' internal || true)
[ -z "$hits" ] || report "internal/ でホームディレクトリを解決しています（パスは注入すること）" "$hits"

# 2. テストは実 ~/.claude を参照しない
hits=$(grep -rnE '~/\.claude|\.claude/projects' --include='*_test.go' internal | grep -v 'testdata' || true)
[ -z "$hits" ] || report "テストが ~/.claude を参照しています（testdata のみ使うこと）" "$hits"

# 3. 実 claude をテストから呼ばない / claude を PATH 名で直接実行しない
hits=$(grep -rnE 'exec\.Command\("claude"|exec\.CommandContext\([^,]+, *"claude"' --include='*.go' . | grep -v node_modules || true)
[ -z "$hits" ] || report "claude を名前で直接実行しています（runner.ResolveClaude + 注入を使うこと）" "$hits"

# 4. claude -p の許可ツールは Read,Write 固定
hits=$(grep -rn 'allowedTools' --include='*.go' internal | grep -v '_test.go' | grep -v 'AllowedTools' || true)
[ -z "$hits" ] || report "allowedTools をハードコードしています（runner.AllowedTools を使うこと）" "$hits"

# 5. 書き込み系 API は許可されたパッケージ以外で使わない（書き込みは applier / fsutil 経由）
hits=$(grep -rnE 'os\.(WriteFile|Create|Rename|Remove|RemoveAll)\(' --include='*.go' internal \
  | grep -vE '_test\.go|internal/(applier|fsutil|runner|skills|testutil)/' || true)
[ -z "$hits" ] || report "applier / fsutil / runner / skills 以外でファイル書き込み API を使っています" "$hits"

# 6. testdata・ソースに本物らしい秘密情報を置かない（DUMMY を含むものは可）
hits=$(grep -rnIE 'sk-ant-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|xox[baprs]-[A-Za-z0-9-]{20,}' \
  --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=dist . | grep -vi 'dummy' || true)
[ -z "$hits" ] || report "秘密情報らしき文字列があります（ダミーは DUMMY を含めること）" "$hits"

# 7. 無効化されたテスト（.only / skip / 理由なしの t.Skip）を残さない
hits=$(grep -rnE '\b(it|test|describe)\.(only|skip)\(|\b(fit|xit|xdescribe|fdescribe)\(' frontend/src || true)
[ -z "$hits" ] || report "フロントのテストに only / skip が残っています" "$hits"
hits=$(grep -rnE 't\.Skip(Now)?\(\)' --include='*_test.go' internal || true)
[ -z "$hits" ] || report "理由なしの t.Skip（理由の文字列か issue 番号を書くこと）" "$hits"

# 8. 混入させてはいけないファイル
hits=$(git ls-files | grep -E 'Zone\.Identifier$|settings\.local\.json$|\.env$' || true)
[ -z "$hits" ] || report "コミットしてはいけないファイルが追跡されています" "$hits"

# 9. Unix 専用 API は `_unix.go`（build タグ付き）に閉じ込める（Windows でビルドできなくなるため）
hits=$(grep -rnE 'syscall\.(Setpgid|Kill|Getpgid|Getuid|Umask|Mkfifo|SIGKILL|SIGTERM|SIGHUP)' --include='*.go' . \
  | grep -vE '_unix\.go|_windows\.go|node_modules' || true)
[ -z "$hits" ] || report "Unix 専用の syscall を OS 非依存のファイルで使っています（*_unix.go / *_windows.go に分けること）" "$hits"

# 10. OS 依存のコマンド起動は internal/osutil か internal/runner に集約する
hits=$(grep -rnE 'exec\.Command(Context)?\([^)]*"(open|xdg-open|explorer|cmd|powershell|sh|bash)"' --include='*.go' internal app.go \
  | grep -vE '_test\.go|internal/(osutil|runner|testutil)/' || true)
[ -z "$hits" ] || report "OS 依存のコマンドを osutil / runner の外で起動しています" "$hits"

# 12. 実行時の生成物（Python のバイトコード等）を追跡しない。ローカルの絶対パス（ホーム・ユーザー名）が埋め込まれ、個人情報の検査（テキスト差分）をすり抜ける
hits=$(git ls-files | grep -E '(^|/)__pycache__/|\.pyc$' || true)
[ -z "$hits" ] || report "コミットしてはいけない生成物が追跡されています（.gitignore で除外すること）" "$hits"

# 11. go:embed のプレースホルダーが追跡されていること（無いとクリーンな clone で go vet が失敗する）
[ -n "$(git ls-files frontend/dist/.gitkeep)" ] || report "frontend/dist/.gitkeep が追跡されていません（main.go の go:embed が失敗します）" "git add -f frontend/dist/.gitkeep"

# 13. 計測（UX 指標・#157）はテレメトリ・外部送信を配布物に持ち込まない。
#     アプリ本体（frontend/src の非テスト）は外部送信 API・計測ライブラリを使わない。計測の依存（axe-core / playwright）は devDependencies のみ。
#     E2E（frontend/e2e）の計測コードは localhost 以外の URL を持たない。
hits=$(grep -rnE 'sendBeacon|XMLHttpRequest|new (WebSocket|EventSource)\(|\bfetch *\(|PerformanceObserver|axe-core|@playwright|ux-metrics' \
  --include='*.ts' --include='*.tsx' frontend/src | grep -vE '\.test\.tsx?:' || true)
[ -z "$hits" ] || report "アプリ本体に外部送信・計測のコードがあります（計測は frontend/e2e の開発用だけに置くこと）" "$hits"
pkg="${HARNESS_PKG_JSON:-frontend/package.json}"
if [ -f "$pkg" ]; then
  hits=$(python3 - "$pkg" <<'PY'
import json, sys
deps = json.load(open(sys.argv[1])).get("dependencies", {})
for name in deps:
    if "axe-core" in name or "playwright" in name or "telemetry" in name or "analytics" in name or "sentry" in name:
        print(name)
PY
)
  [ -z "$hits" ] || report "計測・テスト用の依存が dependencies にあります（devDependencies に置くこと）" "$hits"
fi
hits=$(grep -rnE 'https?://' frontend/e2e 2>/dev/null | grep -vE 'https?://(127\.0\.0\.1|localhost)' || true)
[ -z "$hits" ] || report "E2E の計測コードに localhost 以外の URL があります（外部送信禁止）" "$hits"

[ $fail -eq 0 ] && echo "check-rules OK"
exit $fail
