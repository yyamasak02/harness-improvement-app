#!/usr/bin/env bash
# 開発プロセスでの個人情報・秘密の外部流出を防ぐ検査（決定論的）。
#   check-privacy.sh staged          … ステージした差分の追加行（pre-commit）
#   check-privacy.sh range <base>    … <base>..HEAD の追加行・コミットメッセージ・作者/コミッターのメール（pre-push / CI）
#   check-privacy.sh text            … 標準入力の本文（PR/issue/コメント。gh-post.sh が使う）
# 検出値は出力しない（CI ログは公開されうる）。ダミー値は行に `privacy-allow: 理由` を書く。
set -uo pipefail
cd "$(dirname "$0")/.."
mode="${1:-}"
scan() { python3 -I scripts/privacy_scan.py "$1"; }
case "$mode" in
  staged) git diff --cached --no-color -U0 --diff-filter=AM -- . ':!go.sum' ':!frontend/package-lock.json' | scan diff ;;
  range)
    base="${2:-origin/main}"; fail=0
    git diff --no-color -U0 --diff-filter=AM "$base"...HEAD -- . ':!go.sum' ':!frontend/package-lock.json' | scan diff || fail=1
    git log --format='%B' "$base"..HEAD | scan text || fail=1
    # 作者・コミッターのメールは noreply のみ許可（GitHub のコミッターは noreply@github.com）
    bad=$(git log --format='%ae%n%ce' "$base"..HEAD | sort -u | grep -viE '(@users\.noreply\.github\.com|^noreply@github\.com|^noreply@anthropic\.com|^$)' || true)
    if [ -n "$bad" ]; then
      echo "NG: コミットの作者/コミッターのメールが noreply ではありません（$(echo "$bad" | wc -l | tr -d ' ') 件）。GIT_AUTHOR_EMAIL / GIT_COMMITTER_EMAIL に noreply を設定して作り直す"; fail=1
    fi
    exit $fail ;;
  text) scan text ;;
  *) echo "使い方: check-privacy.sh staged | range <base> | text" >&2; exit 2 ;;
esac
