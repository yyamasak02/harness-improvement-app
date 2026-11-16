# harness-app

Claude Code のハーネス（CLAUDE.md・rules・メモリ・skills）を、`~/.claude` のセッションログから継続改善する **ローカル完結デスクトップアプリ**（Wails v2 / Go / React+TS）。
外部通信は `claude -p` の実行時のみ。外部共有機能は持たない。

- 仕様: @docs/spec.md
- タスク: @docs/tasks.md
- 詳細ルール: `.claude/rules/`（safety / go / frontend / git-workflow）
- 手順の再現: `.claude/skills/`（`implement-task` / `verify` / `open-pr`）。タスク実装は必ず `implement-task` に従う

## やってはいけないこと（絶対）

### 安全
- `~/.claude` や実ホームを **書き込まない**。テストでも禁止。書き込みは `internal/applier` 経由のみ。
- 実 `~/.claude` / ホームを読むテストを書かない。パスは必ず注入し、`testdata/` のみ使う。
- ファイルを直接 `os.WriteFile` で上書きしない（バックアップ → 同ディレクトリ temp → fsync → `os.Rename`）。
- AI 出力を自動保存・自動適用しない。diff プレビューと人間の「適用」を必ず経由する。
- digest を `internal/masking` を通さず AI に渡さない。
- `claude -p` に `Read,Write` 以外のツールを許可しない。cwd をジョブディレクトリ以外にしない。
- 実 `claude` をテストから呼ばない（`scripts/fake-claude.sh` を使う）。
- 秘密情報・トークン・実セッションログの内容を `testdata/`・コミット・PR 本文に入れない（匿名化必須）。

### 開発プロセス
- テスト・`go vet`・フロントビルドが失敗したままコミット／PR／マージしない。
- `main` に直接コミット・push しない（初期コミットを除く）。必ずブランチ → PR → squash マージ。
- force push、`git reset --hard`、`git clean`、`rm -rf`、履歴改変、タグ push、`gh release` をしない（タグ・リリースはユーザーが行う）。
- 1PR に複数機能を混ぜない。タスク順（`docs/tasks.md`）を飛ばさない。
- 仕様にない機能・依存を勝手に足さない。外部共有・クラウド同期・テレメトリを入れない。
- CDN 依存（Monaco 等）を入れない。署名・公証をしない。
- 同じエラーを 3 回直せなければ粘らず `docs/blockers.md` に記録して次へ。
- `.claude/settings.json` の deny を緩めない。

## 自律実行ルール
- 確認で止まらない。迷う細部は安全側を選び `docs/decisions.md` に 1 行追記。
- 止まってよいのは: 仕様と矛盾する変更が必要／`~/.claude` への書き込みが必要になった時のみ。
- 機能（PR）単位で `gh pr create` → CI/ローカル検証通過 → `gh pr merge --squash --delete-branch` まで自分で進める。
- 全タスク後は `docs/spec.md` §9 の完了の定義を 1 項目ずつ検証して終了する。環境上できない項目（macOS 専用）は未検証と明記する。

## コマンド
- 開発: `wails dev` / Go: `go test ./...`・`go vet ./...` / フロント: `cd frontend && npm run build && npm test` / 配布: `make dmg`
- コミット前の検証は `verify` skill（`scripts/verify.sh`）で一括実行。

## 規約
- UI 文言は日本語。コミットメッセージは日本語＋プレフィックス（feat/fix/chore/test/docs）。
