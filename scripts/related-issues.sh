#!/usr/bin/env bash
# 着手前・PR 前に「関連する open issue」を探す。
#   scripts/related-issues.sh [キーワード...]
# 1) 変更中のファイル（origin/main との差分＋作業ツリー）から area ラベルを導出（.github/labeler.yml が定義）
# 2) その area の open issue を列挙  3) ブランチ名・キーワードで全文検索
set -uo pipefail
cd "$(dirname "$0")/.."
command -v gh >/dev/null || { echo "gh が必要です"; exit 1; }

files=$( { git diff --name-only origin/main...HEAD 2>/dev/null; git diff --name-only HEAD 2>/dev/null; git ls-files --others --exclude-standard 2>/dev/null; } | sort -u)
areas=$(FILES="$files" python3 - <<'PY'
import os, re, fnmatch
try:
    import yaml
except ImportError:
    raise SystemExit
conf = yaml.safe_load(open('.github/labeler.yml'))
files = [f for f in os.environ.get('FILES', '').split('\n') if f]
hit = []
for label, rules in conf.items():
    globs = []
    for r in rules:
        for k, v in (r.get('changed-files') or [{}])[0].items():
            globs += v if isinstance(v, list) else [v]
    if any(fnmatch.fnmatch(f, g.replace('**', '*')) for f in files for g in globs):
        hit.append(label)
print('\n'.join(hit))
PY
)

echo "== 変更ファイル: $(echo "$files" | grep -c .) 件 / 該当 area: $(echo $areas | tr '\n' ' ')"
seen=""
show() { # 標準入力: "番号<TAB>タイトル [ラベル]" の行。重複を除いて表示する
  while IFS=$'\t' read -r num title; do
    [ -n "$num" ] || continue
    case " $seen " in *" $num "*) continue ;; esac
    seen="$seen $num"; printf '  #%s %s\n' "$num" "$title"
  done
}
JQ='.[] | "\(.number)\t\(.title)  [\(.labels | map(.name) | join(","))]"'
for a in $areas; do
  echo "-- $a"; gh issue list --state open --label "$a" --limit 30 --json number,title,labels --jq "$JQ" | show
done
kw="$*"
[ -n "$kw" ] || kw=$(git rev-parse --abbrev-ref HEAD | sed -E 's#^[a-z]+/##; s#[-_/]# #g')
if [ -n "$kw" ] && [ "$kw" != "main" ]; then
  echo "-- キーワード検索: $kw"; gh issue list --state open --search "$kw" --limit 20 --json number,title,labels --jq "$JQ" | show
fi
echo "（関連があれば PR 本文に 'Closes #N' か 'Refs #N' を書く。無ければ '[no-issue]: 理由'。無い課題は create-issue で起票）"
