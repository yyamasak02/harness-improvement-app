# 判断記録

- 2026-11-17 .claude/settings.json の git push deny を外し、強制 push・タグ push 等のみ deny に変更（ユーザー承認済み）
- 2026-11-17 Wails は v2.10.2 に固定（v2.17 は Go 1.26 必須。go.mod は toolchain 自動切替で 1.26.0）
- 2026-11-17 assistant の usage は message.id で重複排除（最後の値を採用）。user_msgs は tool_result・isMeta・システムタグを除く
- 2026-11-17 applier はシンボリックリンクへの書き込みを拒否。失敗時は作成済みバックアップを削除。ロールバックは apply 履歴のみ対象
- 2026-11-17 runner は Setpgid でプロセスグループごと kill（macOS/Linux 前提）。stdout/stderr は job.log に統合
- 2026-11-17 ロジックは internal/service に集約（app.go は薄い層）。E2E は testdata を TempDir に複製し cwd を書き換えて実行（/home/demo 等に触れない）。Wails バインディングは wails generate module で生成し frontend/wailsjs をコミット
- 2026-11-17 wails.json の author（テンプレートが個人メールを埋めていた）を削除。過去コミットには残っているためユーザーに通知。dmg ビルド/起動確認は macOS 実機が必要で未検証（CI/release で検証する）
- 2026-11-17 build/darwin/Info*.plist は v2.17 CLI が生成した safeBundleID を使っており、固定した Wails v2.10.2 では解決できないため {{.Name}} に置換（CI の dmg-smoke で検出）
- 2026-11-17 独立レビューの承認は、リポジトリオーナー・github-actions・claude の投稿で、本文の最終行がマーカー、sha が head に一致するもののみ有効（公開リポジトリで第三者が承認を偽装できないように）。ブランチ保護（#47）が入るまでは merge-pr.sh を通さない手動マージ自体は防げない
- 2026-11-17 Windows 対応を必須とした（ユーザー判断、#46）。テスト用 claude を bash から Go 製（internal/testutil/fakeclaude）に置換して全 OS で共通化。パス比較は Windows のみ大文字小文字を無視。Unix 専用 API は *_unix.go に分離し check-rules で強制
- 2026-11-17 Windows の `claude.cmd`（npm shim）は Go の exec が引数の特殊文字を拒否するため `claude.exe` を優先。プロンプトに特殊文字（% や " など）を入れない
- 2026-11-17 go-webview2 は Wails v2.10.2 が想定する v1.0.19 に固定（v2.17 時代の go.mod 由来の v1.0.22 だと Windows ビルドが失敗した）。ローカルの Wails CLI も v2.10.2 に揃える（別版は生成物を書き換える）
- 2026-11-17 WSL 対応は「Linux 版を WSL 内（WSLg）で動かす」方式（ユーザー判断）。WebKitGTK 4.1（`-tags webkit2_41`）を使い、Release には添付せず `make linux` でビルド。CI は linux-build-smoke でビルド確認のみ（GUI 起動は未検証）
