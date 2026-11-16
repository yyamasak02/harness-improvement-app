# 判断記録

- 2026-11-17 .claude/settings.json の git push deny を外し、強制 push・タグ push 等のみ deny に変更（ユーザー承認済み）
- 2026-11-17 Wails は v2.10.2 に固定（v2.17 は Go 1.26 必須。go.mod は toolchain 自動切替で 1.26.0）
- 2026-11-17 assistant の usage は message.id で重複排除（最後の値を採用）。user_msgs は tool_result・isMeta・システムタグを除く
- 2026-11-17 applier はシンボリックリンクへの書き込みを拒否。失敗時は作成済みバックアップを削除。ロールバックは apply 履歴のみ対象
- 2026-11-17 runner は Setpgid でプロセスグループごと kill（macOS/Linux 前提）。stdout/stderr は job.log に統合
- 2026-11-17 ロジックは internal/service に集約（app.go は薄い層）。E2E は testdata を TempDir に複製し cwd を書き換えて実行（/home/demo 等に触れない）。Wails バインディングは wails generate module で生成し frontend/wailsjs をコミット
