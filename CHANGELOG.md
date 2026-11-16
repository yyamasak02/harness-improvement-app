# Changelog

このプロジェクトの注目すべき変更はここに記録する。形式は [Keep a Changelog](https://keepachangelog.com/ja/1.1.0/)、バージョンは [SemVer](https://semver.org/lang/ja/) に従う。
`feat` / `fix` の PR は `[Unreleased]` に追記する（CI の `check-changelog` が検査）。リリース時に `scripts/release.sh prepare` が確定する。

## [Unreleased]

### 追加
- Claude Code のセッションログ（`~/.claude`）を読み取り専用で取り込み、概要ダッシュボードとセッションブラウザ（全文検索・訂正とエラーのハイライト）で確認できる
- CLAUDE.md / CLAUDE.local.md / rules の改善案を AI（`claude -p`）で生成し、差分を確認して適用・破棄できる（右側は編集可）。適用前に自動でバックアップし、変更履歴からロールバックできる
- 設定画面（claude のパス・モデル・タイムアウト・除外プロジェクト・訂正パターン・マスキング追加パターン・インデックス再構築）
- macOS（.dmg）と Windows（インストーラ）に対応。いずれも署名・公証はしない

### セキュリティ
- `~/.claude` は読み取り専用。書き込みは許可リスト・ハッシュ照合・バックアップ付きの applier のみ
- AI に渡す digest と対象ファイルは必ずマスキングを通す。`claude -p` は `Read,Write` のみ許可し、ジョブディレクトリ内で実行する
