# harness-app

Claude Code のハーネス（CLAUDE.md・rules・メモリ・skills）を、`~/.claude` に溜まったセッションログから継続改善するための **ローカル完結デスクトップアプリ**。
外部共有機能は持たない。外部通信は `claude -p` の実行時のみ。

- 仕様: @docs/spec.md
- タスク: @docs/tasks.md

## 進め方（自律実行ルール）

- `docs/tasks.md` の未完了タスクを **上から順に** 実装する。1タスク完了ごとにチェックを付け、コミットして次へ進む。
- 確認のために手を止めない。判断に迷う細部は、安全側の選択をして `docs/decisions.md` に1行記録して進める。
- 止まってよいのは次の場合だけ：
  - 仕様と矛盾する変更が必要なとき
  - `~/.claude` への書き込みが必要になったとき（テストでも禁止）
  - 同じエラーを3回直せなかったとき → `docs/blockers.md` に状況を書いて次のタスクへ進む
- 全タスク完了後、「完了の定義」を満たしているか確認して終了する。

## コマンド

- 開発起動: `wails dev`
- Go テスト: `go test ./...`
- 静的解析: `go vet ./...`
- フロント: `cd frontend && npm run build && npm test`
- 配布ビルド: `make dmg`（`build/bin/harness-app.dmg` を出力）

## コミット前に必ず通すもの

1. `go vet ./... && go test ./...`
2. `cd frontend && npm run build && npm test`
3. 失敗したまま次のタスクに進まない

## 絶対ルール（安全設計）

- **`~/.claude` は読み取り専用。** 書き込むのはユーザーが UI で「適用」を押したときの `internal/applier` 経由だけ。
- テストは `testdata/` のダミーデータのみを使う。実際の `~/.claude` やホームディレクトリを読むテストは書かない（パスは必ず注入できるようにする）。
- ファイルの書き込みは必ず「バックアップ → 同じディレクトリの temp ファイル → `os.Rename`」の順で行う。
- AI の出力は自動で保存しない。必ず diff プレビューを経由する。
- AI に渡す digest は必ず `internal/masking` を通す。
- `claude -p` を呼ぶときは `--allowedTools Read,Write`、cwd はアプリの作業ディレクトリに固定する。

## コード規約

- Go: 標準のレイアウト。ロジックは `internal/` に置き、`app.go` は薄いバインディング層に留める。エラーは `fmt.Errorf("...: %w", err)` でラップする。
- テーブル駆動テストを使う。外部プロセス（`claude`）はフェイクのスクリプトに差し替えてテストする。
- フロント: TypeScript strict。Wails バインディングは `frontend/src/api.ts` に集約し、ブラウザ単体（`npm run dev`）では mock 実装に切り替わるようにする。
- UI 文言は日本語。
