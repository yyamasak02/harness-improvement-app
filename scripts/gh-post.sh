#!/usr/bin/env bash
# GitHub への投稿（issue / PR / コメント / レビュー）の唯一の経路。本文とタイトルを個人情報検査してから gh を実行する。
# 使い方: bash scripts/gh-post.sh <issue|pr> <create|comment|edit|review> [gh のオプション...]
#   例: bash scripts/gh-post.sh issue create --title "feat: …" --body-file body.md --label type/feature,…
# gh を直接実行して投稿することは settings.json の deny で禁止している（無検査の投稿を防ぐ）。
# 環境変数 GH_POST_DRY_RUN=1 なら検査だけして gh を呼ばない（テスト用）。
set -uo pipefail
cd "$(dirname "$0")/.."
[ $# -ge 2 ] || { echo "使い方: gh-post.sh <issue|pr> <create|comment|edit|review> [options]" >&2; exit 2; }
case "$1 $2" in
  "issue create"|"issue comment"|"issue edit"|"pr create"|"pr comment"|"pr edit"|"pr review") ;;
  *) echo "gh-post.sh が扱えるのは issue create|comment|edit と pr create|comment|edit|review のみです（$1 $2）" >&2; exit 2 ;;
esac
tmp=$(mktemp); trap 'rm -f "$tmp"' EXIT
args=("$@"); i=2
read_file() { # 本文ファイルを検査対象に足す
  local f="$1"
  if [ "$f" = "-" ]; then echo "--body-file - （標準入力）は検査できないので使えません。ファイルに書いてください" >&2; exit 2; fi
  [ -f "$f" ] || { echo "本文ファイルがありません: $f" >&2; exit 2; }
  cat "$f" >> "$tmp"; printf '\n' >> "$tmp"
}
# すべての引数の文字列を検査対象にする（--body / --title / -b=… / -bTEXT などの書式の違いで抜けないように）。
# --body-file / -F は、値（別引数・=付き・連結）のファイル内容も検査する。
while [ $i -lt ${#args[@]} ]; do
  a="${args[$i]}"
  case "$a" in
    --body-file|-F) read_file "${args[$((i+1))]:-}"; i=$((i+2)); continue ;;
    --body-file=*)  read_file "${a#--body-file=}" ;;
    -F?*)           v="${a#-F}"; read_file "${v#=}" ;;
  esac
  printf '%s\n' "$a" >> "$tmp"
  i=$((i+1))
done
if ! bash scripts/check-privacy.sh text < "$tmp"; then
  echo "投稿を中止しました（本文に個人情報・秘密らしきものがあります）。取り除くか、意図したダミー値なら 'privacy-allow: 理由' を行に書いてください。" >&2
  exit 1
fi
[ -n "${GH_POST_DRY_RUN:-}" ] && { echo "dry-run: gh $*"; exit 0; }
exec gh "$@"
