---
description: Go コード規約
paths: ["**/*.go"]
---
- ロジックは `internal/<pkg>/`。`app.go` は薄いバインディングのみ。
- エラーは `fmt.Errorf("...: %w", err)`。panic しない。
- テーブル駆動テスト。外部プロセス（claude）は `internal/testutil` の Go 製フェイクに差し替える。
- SQLite は `modernc.org/sqlite`（cgo 不要）。FTS5 trigram、3 文字未満は LIKE。
- `gofmt` 済みであること。カバレッジ対象（collector/digest/masking/applier/runner）は 70% 以上。

## クロスプラットフォーム（macOS / Windows）
- パスは `path/filepath` で組み立て・比較する（`path` や文字列連結で `/` を使わない）。パス比較は `targets.SamePath`（Windows は大文字小文字を無視）。
- Unix 専用 API（`syscall.Setpgid` / `Kill` など）は `*_unix.go`（`//go:build !windows`）に、Windows 版は `*_windows.go` に分ける。`scripts/check-rules.sh` が検査する。
- OS 依存のコマンド起動（`open` / `explorer` / `where` など）は `internal/osutil` か `internal/runner` に集約する。
- 実行ファイルの判定は Windows では拡張子（`.exe` `.cmd`）、他は実行ビット。`claude.exe` を `.cmd` より優先する。
- テストで OS 依存の機能（パーミッションビット・シンボリックリンク・読み取り専用ディレクトリ）を使う場合は、`t.Skip("理由")` を付けて Windows を除外する。外部プロセスは OS 非依存の Go 製フェイクを使う。
- 変更後は `bash scripts/lint.sh vet-windows`（`GOOS=windows go vet ./...`）が通ること。CI の windows-latest でもテストする。
- パッケージ構成・依存の向きは `docs/architecture.md`（レイヤー方針・許容リスト・移行計画）に従う。`internal/archtest` と `frontend/src/arch.test.ts` が検査する（許容リストへの追加は禁止）。
