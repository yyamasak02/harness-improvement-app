# harness-app

Claude Code のハーネス（CLAUDE.md・rules・メモリ・skills）を、`~/.claude` に溜まったセッションログから**継続的に改善する**ローカル完結のデスクトップアプリです。

```
Claude Code を使う → ログ・メモリが溜まる → AI が改善案を出す → 人間が diff を見て適用 → …
```

- 外部共有・クラウド同期・テレメトリはありません。外部通信は改善案の生成時に実行する `claude -p` のみです。
- `~/.claude` は**読み取り専用**です。書き込みは、あなたが「適用」を押したときだけ、バックアップ付きで行われます。

## 機能

| 画面 | 内容 |
|---|---|
| 概要 | 期間（7/30/90 日）ごとのセッション数・メッセージ数・トークン・訂正数・ツールエラー数、日別グラフ、モデル別使用量、プロジェクト別分布、メモリファイル数 |
| セッション | プロジェクト・期間で絞り込み、全文検索、時系列表示（訂正・エラーをハイライト） |
| CLAUDE.md 改善 | 対象選択 → digest プレビュー（マスク済み）→ 改善案の生成（キャンセル可）→ 左右 diff（右側は編集可）→ 適用 / 破棄 |
| 変更履歴 | アプリが書き込んだ全ファイルのバックアップ、適用前後の diff、ロールバック |
| 設定 | claude のパス（自動検出＋手動）、モデル、除外プロジェクト、訂正パターン、マスキング追加パターン、インデックス再構築 |

<!-- スクリーンショット: docs/screenshots/ に追加してください -->

## インストール（macOS）

1. [Releases](../../releases) から `harness-app.dmg` をダウンロードして開き、`harness-app.app` を「アプリケーション」へドラッグします。
2. **本アプリは署名・公証をしていません。** 初回は次のどちらかで開きます。
   - Finder で `harness-app.app` を **右クリック →「開く」** → 確認ダイアログで「開く」
   - またはターミナルで隔離属性を外します。
     ```sh
     xattr -dr com.apple.quarantine /Applications/harness-app.app
     ```
3. [Claude Code](https://claude.com/claude-code) がインストールされている必要があります。見つからない場合は「設定」でフルパスを指定してください（`which claude` で分かります）。

## 使い方

1. 起動すると `~/.claude/projects/` を自動で取り込みます（差分のみ。以降はファイル変更を監視して追従します）。
2. 「概要」「セッション」で状況を眺めます。
3. 「CLAUDE.md 改善」で対象（グローバル / プロジェクトの CLAUDE.md / CLAUDE.local.md / rules）と期間、必要なら追加指示を選び、digest を確認してから「改善案を生成」します。
4. 差分を確認し、必要なら右側を直接編集して「適用」します。気に入らなければ「破棄」します。
5. 元に戻したいときは「変更履歴」からロールバックします。

## データの保存場所

| 内容 | 場所（macOS） |
|---|---|
| 設定 | `~/Library/Application Support/harness-app/config.json` |
| 索引 DB（消しても再構築できます） | `~/Library/Application Support/harness-app/index.db` |
| バックアップ・変更履歴 | `~/Library/Application Support/harness-app/backups/<時刻>-<ハッシュ>/` |
| AI 実行のジョブ（入出力・ログ） | `~/Library/Application Support/harness-app/work/<job-id>/` |

## 安全設計

- **`~/.claude` は読み取り専用。** 書き込みは `internal/applier` だけが行い、許可リスト（発見済みの CLAUDE.md / CLAUDE.local.md / rules と `~/.claude/skills/` 配下）以外は拒否します。シンボリックリンクにも書き込みません。
- 適用前に対象ファイルの SHA-256 を、改善案を生成した時点の値と比較します。変わっていたら中止して再生成を促します。
- 手順は「バックアップ → 同ディレクトリの temp ファイル → fsync → rename」。ロールバック自体もバックアップされます。
- **AI の出力は自動保存されません。** 必ず diff プレビューを経由します。
- AI に渡す digest と対象ファイル本文は、必ずマスキングを通します（`sk-ant-…`、AWS キー、GitHub/Slack トークン、Bearer、PEM 秘密鍵、メールアドレス、`password|secret|token|api_key = …`、ユーザー追加パターン）。
- `claude -p` は `--allowedTools Read,Write` に固定し、cwd はアプリの作業ディレクトリ内のジョブディレクトリに固定します。タイムアウト（既定 5 分）とキャンセルに対応します。

## 開発

初回は `make setup` で pre-commit のフック（commit / commit-msg / pre-push）を導入します。
コミット時に gofmt・go vet・staticcheck・tsc・ルール検査・コミットメッセージ形式を、push 時に `scripts/verify.sh --full`（テスト・カバレッジ基準まで）を自動実行します。CI でも同じ検査と、`go test -race`・PR タイトル形式・「実装を変えたらテストも変える」を検査します。品質の方針は [.claude/rules/quality.md](.claude/rules/quality.md) を参照してください。

必要なもの: Go、Node.js 20+、[Wails v2](https://wails.io)（`go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`）。

```sh
make dev            # wails dev
make setup          # pre-commit フックの導入（初回のみ）
make test           # go vet / go test / フロントの build と test
make verify-full    # lint・ルール検査・カバレッジ基準まで（push 前・CI 相当）
bash scripts/coverage.sh   # collector・digest・masking・applier・runner のカバレッジ 70% 以上を確認
make bindings       # Go のメソッドを変えたら Wails バインディングを再生成
make dmg            # macOS 専用: build/bin/harness-app.dmg を出力
```

- ブラウザ単体の UI 確認: `cd frontend && npm run dev`（Wails が無いときは mock API に切り替わります）。
- テストは `testdata/` のダミーデータだけを使い、実際の `~/.claude` には触れません。`claude` はフェイク（`internal/testutil/fakeclaude`、Go 製でテスト時にビルド）に差し替えます。
- 仕様は [docs/spec.md](docs/spec.md)、タスクは [docs/tasks.md](docs/tasks.md)、開発ルールは [CLAUDE.md](CLAUDE.md) と `.claude/` を参照してください。

## リリース

`v*` タグを push すると GitHub Actions（macos-latest）が `.dmg` をビルドして Release に添付します。タグの push は手動で行います。
