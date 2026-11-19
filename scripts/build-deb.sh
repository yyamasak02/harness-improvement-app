#!/usr/bin/env bash
# build/bin/harness-app（make linux の成果物）から .deb を作る。署名はしない。dpkg-deb のみ使う（追加の依存なし）。
# 使い方: bash scripts/build-deb.sh [出力ディレクトリ(既定 build/bin)]
# バージョンの出典は wails.json。SemVer の pre-release（-rc.1）は Debian の ~ 形式（~rc.1）にする。
set -euo pipefail
cd "$(dirname "$0")/.."
out="${1:-build/bin}"
bin=build/bin/harness-app
[ -x "$bin" ] || { echo "$bin がありません。先に make linux を実行してください"; exit 1; }
ver=$(sed -n 's/.*"productVersion": *"\([^"]*\)".*/\1/p' wails.json)
[ -n "$ver" ] || { echo "wails.json からバージョンを読めません"; exit 1; }
debver="${ver/-/\~}"
root=$(mktemp -d)
trap 'rm -rf "$root"' EXIT
chmod 755 "$root"
install -Dm755 "$bin" "$root/usr/bin/harness-app"
install -Dm644 build/appicon.png "$root/usr/share/icons/hicolor/512x512/apps/harness-app.png"
mkdir -p "$root/usr/share/applications" "$root/DEBIAN"
cat > "$root/usr/share/applications/harness-app.desktop" <<DESK
[Desktop Entry]
Type=Application
Name=harness-app
Comment=Claude Code のハーネスをセッションログから継続改善するローカルアプリ
Exec=/usr/bin/harness-app
Icon=harness-app
Terminal=false
Categories=Development;
DESK
size=$(du -sk "$root/usr" | cut -f1)
cat > "$root/DEBIAN/control" <<CTL
Package: harness-app
Version: $debver
Architecture: amd64
Maintainer: harness-app <noreply@users.noreply.github.com>
Installed-Size: $size
Depends: libgtk-3-0, libwebkit2gtk-4.1-0
Section: devel
Priority: optional
Description: Claude Code のハーネス改善アプリ
 セッションログから CLAUDE.md などの改善案を作り、差分を確認して適用するローカル完結のデスクトップアプリ。
CTL
mkdir -p "$out"
deb="$out/harness-app_${debver}_amd64.deb"
dpkg-deb --root-owner-group --build "$root" "$deb" >/dev/null
echo "$deb"
