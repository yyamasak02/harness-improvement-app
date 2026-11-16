#!/usr/bin/env bash
# バージョンの整合性を検査する。バージョンの出典は wails.json の info.productVersion。
#   scripts/check-version.sh           … wails.json == frontend/package.json、SemVer 形式、CHANGELOG に [Unreleased] がある
#   scripts/check-version.sh vX.Y.Z    … さらに、タグ == v + wails.json のバージョン、CHANGELOG に [X.Y.Z] の節がある
set -euo pipefail
cd "$(dirname "$0")/.."
tag="${1:-}"
jsonget() { python3 -I -c 'import json,sys;d=json.load(open(sys.argv[1]));print(eval(sys.argv[2],{"d":d}))' "$1" "$2"; }
wv=$(jsonget wails.json 'd["info"]["productVersion"]')
pv=$(jsonget frontend/package.json 'd["version"]')
semver='^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$'
fail=0
bad() { echo "NG: $1"; fail=1; }
[[ "$wv" =~ $semver ]] || bad "wails.json のバージョンが SemVer ではありません: $wv"
[ "$wv" = "$pv" ] || bad "バージョンが一致しません: wails.json=$wv / frontend/package.json=$pv（出典は wails.json）"
[ -f CHANGELOG.md ] || bad "CHANGELOG.md がありません"
grep -qE '^## \[Unreleased\]' CHANGELOG.md 2>/dev/null || bad "CHANGELOG.md に '## [Unreleased]' の節がありません"
if [ -n "$tag" ]; then
  [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || bad "タグが vX.Y.Z 形式ではありません: $tag"
  [ "$tag" = "v$wv" ] || bad "タグ $tag が wails.json のバージョン v$wv と一致しません"
  grep -qE "^## \[${tag#v}\]" CHANGELOG.md || bad "CHANGELOG.md に '## [${tag#v}]' の節がありません（release.sh prepare で確定する）"
fi
[ $fail -eq 0 ] && echo "check-version OK（$wv${tag:+ / $tag}）"
exit $fail
