# harness-app

Claude Code のハーネス（CLAUDE.md・rules・メモリ・skills）を、`~/.claude` のセッションログから継続改善する **ローカル完結デスクトップアプリ**（Wails v2 / Go / React+TS）。
外部通信は `claude -p` の実行時のみ。外部共有機能は持たない。

- 仕様: @docs/spec.md
- タスク: @docs/tasks.md
- 詳細ルール: `.claude/rules/`（safety / go / frontend / git-workflow / issues-labels / quality / session / release）、レビュー基準は `docs/review-guidelines.md`
- 手順の再現: `.claude/skills/`（`implement-task` / `quality-gate` / `fix-bug` / `verify` / `open-pr` / `create-issue` / `review-pr` / `resume` / `release`）。タスク実装は必ず `implement-task` に従う

## セッション開始・再開時
- **最初に `bash scripts/status.sh`**（SessionStart フックで要約が自動表示される）を確認し、`resume` skill でフェーズを判定して再開する。`docs/handoff.md` は参考で、現実（git・PR・issue）が優先。マージ・フェーズ変更・待ち状態の前に handoff を更新する（`.claude/rules/session.md`）。

## やってはいけないこと（絶対）

### 安全
- `~/.claude` や実ホームを **書き込まない**。テストでも禁止。書き込みは `internal/applier` 経由のみ。
- 実 `~/.claude` / ホームを読むテストを書かない。パスは必ず注入し、`testdata/` のみ使う。
- ファイルを直接 `os.WriteFile` で上書きしない（バックアップ → 同ディレクトリ temp → fsync → `os.Rename`）。
- AI 出力を自動保存・自動適用しない。diff プレビューと人間の「適用」を必ず経由する。
- digest を `internal/masking` を通さず AI に渡さない。
- `claude -p` に `Read,Write` 以外のツールを許可しない。cwd をジョブディレクトリ以外にしない。
- 実 `claude` をテストから呼ばない（`internal/testutil` の `FakeClaude` を使う。Go 製なので Windows でも動く）。
- 秘密情報・トークン・実セッションログの内容を `testdata/`・コミット・PR 本文に入れない（匿名化必須）。

### 開発プロセス
- テスト・`go vet`・フロントビルドが失敗したままコミット／PR／マージしない。
- `git commit/push --no-verify` で pre-commit を迂回しない。テストの `skip` / `.only`、期待値の緩和、カバレッジ閾値の引き下げで通さない。
- 実装を変えてテストを変えない（直さない・足さない）まま PR にしない。意味を失ったテストを放置しない。
- リファクタ・テスト不足を見つけて黙って見送らない。今回やらないなら `type/tech-debt` / `type/test` の issue にする。
- `main` に直接コミット・push しない（初期コミットを除く）。必ずブランチ → PR → squash マージ。
- `feat` / `fix` の変更で `CHANGELOG.md` の `[Unreleased]` を更新しない。バージョンを `wails.json` 以外で勝手に変えない（出典は `wails.json`）。リリースタグの push・付け替え・削除をしない（`scripts/release.sh tag` でローカルに作り、push は人に依頼する）。
- force push、`git reset --hard`、`git clean`、`rm -rf`、履歴改変、タグ push、`gh release` をしない（タグ・リリースはユーザーが行う）。
- 未着手・保留の課題を口頭 TODO や握りつぶしにしない。必ず issue にする（`create-issue` skill、ラベルは `.claude/rules/issues-labels.md`）。GitHub 既定ラベルは使わない。
- **ユーザーの要求（機能追加・修正・方針変更・調査の依頼）を、issue にしないまま着手しない。** 受け取ったら最初に issue を作る（`create-issue` skill）。口頭のまま実装して後追いで起票しない。`[no-issue]` は依存更新・誤字修正・生成物の更新・リリース準備だけ。
- 修正・実装の前に関連 issue を探さない（`bash scripts/related-issues.sh`）、PR 本文で issue を参照しない。
- 独立レビュー（`review-pr` skill・`code-reviewer` エージェント）を経ずにマージしない。自分で自分をレビューしたことにしない。`gh pr merge` を直接実行しない（`scripts/merge-pr.sh` を使う）。レビュー後に push したら再レビューを受ける。
- 1PR に複数機能を混ぜない。タスク順（`docs/tasks.md`）を飛ばさない。
- 仕様にない機能・依存を勝手に足さない。外部共有・クラウド同期・テレメトリを入れない。
- CDN 依存（Monaco 等）を入れない。署名・公証をしない。
- 同じエラーを 3 回直せなければ粘らず `docs/blockers.md` に記録して次へ。
- `.claude/settings.json` の deny を緩めない。

## 自律実行ルール
- 確認で止まらない。迷う細部は安全側を選び `docs/decisions.md` に 1 行追記。
- 止まってよいのは: 仕様と矛盾する変更が必要／`~/.claude` への書き込みが必要になった時のみ。
- 機能（PR）単位で `scripts/ship.sh`（PR 作成）→ 独立レビュー（`review-pr`）→ `scripts/merge-pr.sh`（マージ）まで自分で進める。
- 全タスク後は `docs/spec.md` §9 の完了の定義を 1 項目ずつ検証して終了する。環境上できない項目（macOS 専用）は未検証と明記する。

## コマンド
- 開発: `wails dev` / Go: `go test ./...`・`go vet ./...` / フロント: `cd frontend && npm run build && npm test` / 配布: `make dmg`
- コミット前: `verify` skill（`scripts/verify.sh`）。push・PR 前: `bash scripts/verify.sh --full`（lint・ルール検査・カバレッジ基準）と `quality-gate` skill。
- 初回は `make setup`（pre-commit のフックを 3 ステージ導入）。

## 規約
- UI 文言は日本語。コミットメッセージは日本語＋プレフィックス（feat/fix/chore/test/docs）。
