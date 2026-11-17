#!/usr/bin/env bash
# Release ノートの先頭に付けるダウンロード案内。GitHub が自動で付ける「Source code (zip / tar.gz)」は無効にできないため、
# どのファイルをダウンロードすればよいかをノートの先頭で明示する。
cat <<'EOF'
## ダウンロード

使うのは次の**インストーラ**だけです。下の「Source code (zip / tar.gz)」は GitHub が自動で付けるソースコードで、アプリではありません。

| OS | ダウンロードするファイル |
|---|---|
| macOS | `harness-app.dmg` |
| Windows | `harness-app-amd64-installer.exe` |

インストール手順と、未署名アプリの開き方は README の「インストール」を参照してください。

EOF
