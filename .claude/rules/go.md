---
description: Go コード規約
paths: ["**/*.go"]
---
- ロジックは `internal/<pkg>/`。`app.go` は薄いバインディングのみ。
- エラーは `fmt.Errorf("...: %w", err)`。panic しない。
- テーブル駆動テスト。外部プロセスはフェイクスクリプトに差し替え。
- SQLite は `modernc.org/sqlite`（cgo 不要）。FTS5 trigram、3 文字未満は LIKE。
- `gofmt` 済みであること。カバレッジ対象（collector/digest/masking/applier/runner）は 70% 以上。
