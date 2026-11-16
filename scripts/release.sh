#!/usr/bin/env bash
# リリース手順の補助。タグの push は人が行う（エージェントには権限が無い）。
#   scripts/release.sh prepare X.Y.Z   … リリース用ブランチを作り、バージョン更新と CHANGELOG の確定を行う
#   scripts/release.sh tag X.Y.Z       … 事前チェックの上で、ローカルに注釈付きタグ vX.Y.Z を作り push コマンドを案内する
set -euo pipefail
cd "$(dirname "$0")/.."
cmd="${1:-}"; ver="${2:-}"
[[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || { echo "使い方: $0 prepare|tag X.Y.Z（例: 0.1.0 / 0.2.0-rc.1）"; exit 2; }
tag="v$ver"
require_clean_main() {
  [ "$(git rev-parse --abbrev-ref HEAD)" = "main" ] || { echo "main ブランチで実行してください"; exit 1; }
  [ -z "$(git status --porcelain)" ] || { echo "作業ツリーがクリーンではありません"; exit 1; }
  git fetch -q origin main
  [ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || { echo "main が origin/main と一致していません（git pull してください）"; exit 1; }
}
case "$cmd" in
  prepare)
    require_clean_main
    git rev-parse -q --verify "refs/tags/$tag" >/dev/null && { echo "タグ $tag は既に存在します"; exit 1; }
    grep -qE '^## \[Unreleased\]' CHANGELOG.md || { echo "CHANGELOG.md に [Unreleased] がありません"; exit 1; }
    [ -n "$(awk '/^## \[Unreleased\]/{on=1;next} on&&/^## \[/{exit} on&&NF{print}' CHANGELOG.md)" ] || { echo "CHANGELOG の [Unreleased] が空です。リリースする変更を書いてください"; exit 1; }
    git switch -q -c "release/$tag"
    today=$(date +%Y-%m-%d)
    # [Unreleased] を新しい節に確定し、空の [Unreleased] を先頭に作る
    python3 -I - "$ver" "$today" <<'PY'
import re, sys
ver, today = sys.argv[1:3]
s = open("CHANGELOG.md", encoding="utf-8").read()
s = s.replace("## [Unreleased]", "## [Unreleased]\n\n## [%s] - %s" % (ver, today), 1)
open("CHANGELOG.md", "w", encoding="utf-8").write(s)
PY
    python3 -I - "$ver" <<'PY'
import json, sys
ver = sys.argv[1]
p = "wails.json"
d = json.load(open(p, encoding="utf-8")); d["info"]["productVersion"] = ver
open(p, "w", encoding="utf-8").write(json.dumps(d, indent=2, ensure_ascii=False) + "\n")
PY
    (cd frontend && npm version "$ver" --no-git-tag-version --allow-same-version >/dev/null)
    bash scripts/check-version.sh "$tag"
    git add -A && git commit -q -m "chore: $tag のリリース準備（バージョン更新・CHANGELOG 確定）

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
    echo "リリース準備のコミットを release/$tag に作りました。次: bash scripts/ship.sh \"chore: $tag のリリース準備\" \"Refs #<リリース issue>\\n[no-test-needed]: バージョンと CHANGELOG のみ\" → 独立レビュー → merge-pr.sh → bash scripts/release.sh tag $ver"
    ;;
  tag)
    require_clean_main
    git rev-parse -q --verify "refs/tags/$tag" >/dev/null && { echo "タグ $tag は既にローカルに存在します"; exit 1; }
    git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null 2>&1 && { echo "タグ $tag は既にリモートに存在します"; exit 1; }
    bash scripts/check-version.sh "$tag"
    head=$(git rev-parse HEAD)
    # HEAD の CI が全て成功していること
    run=$(gh run list --workflow CI --branch main --limit 20 --json databaseId,headSha,status,conclusion --jq "[.[]|select(.headSha==\"$head\")][0]")
    [ -n "$run" ] && [ "$run" != "null" ] || { echo "main の HEAD（${head:0:7}）の CI 実行が見つかりません"; exit 1; }
    st=$(echo "$run" | python3 -I -c 'import json,sys;d=json.load(sys.stdin);print(d["status"],d["conclusion"])')
    [ "$st" = "completed success" ] || { echo "main の HEAD の CI が成功していません: $st"; exit 1; }
    echo "== 未解決の優先度 p0/p1 の issue（リリース前に確認）"
    gh issue list --state open --limit 100 --json number,title,labels --jq '.[]|select(.labels|map(.name)|any(.=="priority/p0" or .=="priority/p1"))|"  #\(.number) \(.title)  [\(.labels|map(.name)|join(","))]"' || true
    notes=$(bash scripts/changelog-notes.sh "$ver")
    git tag -a "$tag" -m "$tag

$notes"
    echo "ローカルに注釈付きタグ $tag を作成しました（${head:0:7}）。"
    echo "push はあなたが実行してください（エージェントには権限がありません）:"
    echo "  git push origin $tag"
    echo "push すると .github/workflows/release.yml が macOS（.dmg）と Windows（.zip・インストーラ）をビルドして Release に添付します。"
    ;;
  *) echo "使い方: $0 prepare|tag X.Y.Z"; exit 2 ;;
esac
