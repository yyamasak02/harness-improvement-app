# harness-app 仕様書 v0.1

## 1. 目的

Claude Code を使う → セッションログ・メモリが溜まる → それを材料にハーネス（CLAUDE.md / rules / skills）の改善案を AI に出させる → 人間が diff を見て適用する、というループをローカルで回す。

対象外：他者・他社との共有、Slack 等への外部投稿、クラウド同期。

## 2. 技術スタック

| レイヤー | 採用 |
|---|---|
| デスクトップ | Wails v2（Go + WebView）。**macOS と Windows を対象**（Windows は WebView2 が必要。Windows 10 以降）。加えて **Linux（WSL2 の WSLg を含む）はソースからのビルドと `.deb`（amd64）で対応**（GTK3 + WebKitGTK 4.1。`.deb` は Release に添付する） |
| バックエンド | Go 1.24+ |
| フロントエンド | React + TypeScript + Vite |
| UI | Tailwind CSS（コンポーネントは自作 or shadcn/ui） |
| グラフ | Recharts |
| diff 表示 | `diff`（jsdiff）で行 diff を計算し、自前で描画する。Monaco は CDN 依存を避けるため使わない |
| Markdown | react-markdown |
| DB | SQLite（`modernc.org/sqlite`、cgo 不要）。全文検索は FTS5（`tokenize='trigram'`、3文字未満は LIKE にフォールバック） |
| 監視 | fsnotify（差分取り込み） |
| AI 実行 | `claude -p` を `os/exec` でヘッドレス実行 |
| テスト | Go: 標準 testing / フロント: Vitest |
| 配布 | macOS: `wails build -platform darwin/universal` → `hdiutil` で .dmg。Windows: `wails build -platform windows/amd64` → NSIS インストーラ（Release に添付するのはインストーラのみ。zip は配布しない）。いずれも署名・公証はしない。Linux: `make linux`（`wails build -platform linux/amd64 -tags webkit2_41`）→ `scripts/build-deb.sh`（`dpkg-deb`。依存は `libgtk-3-0` `libwebkit2gtk-4.1-0`。署名なし。Release に添付） |

## 3. ディレクトリ構成

```
harness-app/
├── main.go / app.go          # Wails 起動とバインディング（薄く）
├── internal/
│   ├── config/               # 設定の読み書き（アプリデータディレクトリ）
│   ├── collector/            # jsonl パース → SQLite 取り込み（読み取り専用）
│   ├── store/                # SQLite スキーマ・クエリ
│   ├── digest/               # 期間・プロジェクト指定で digest を生成
│   ├── masking/              # 秘密情報のマスキング
│   ├── targets/              # CLAUDE.md / rules の発見
│   ├── runner/               # claude -p の実行、claude バイナリの解決
│   ├── applier/              # バックアップ → temp → rename、ロールバック
│   └── skills/               # 同梱 skill（embed）
├── frontend/
├── testdata/                 # ダミーの ~/.claude 構造
├── internal/testutil/        # テスト用ヘルパー。fakeclaude/ は Go 製のフェイク claude（Windows でも動く）
├── build/                    # Wails のビルド資材、アイコン
├── Makefile
└── .github/workflows/release.yml
```

## 4. データソース

- Windows では `~/.claude` は `%USERPROFILE%\.claude`、アプリデータは `%APPDATA%\harness-app`、jsonl の `cwd` は `C:\...` 形式。パスの比較は Windows では大文字小文字を区別しない。
- `~/.claude/projects/<エンコード済みパス>/<session-id>.jsonl`：セッションログ。1行1 JSON。
- `~/.claude/projects/<project>/memory/*.md`：プロジェクトメモリ（存在すれば）。
- `~/.claude/CLAUDE.md`、`<cwd>/CLAUDE.md`、`<cwd>/CLAUDE.local.md`、`<cwd>/.claude/rules/*.md`：改善対象。`<cwd>` は jsonl の `cwd` フィールドから得る。
- `~/.claude/settings.json`、`~/.claude.json`、`~/.claude/skills/`、`~/.claude/plugins/`：拡張機能の状態表示用（読み取りのみ）。

### jsonl パースの方針

- フォーマットは Claude Code のバージョンで変わるため、**寛容にパースする**。未知のフィールドは無視し、壊れた行はスキップして件数だけ数える。
- 主に使うフィールド：`type`（user/assistant/summary/system）、`sessionId`、`timestamp`、`cwd`、`gitBranch`、`message.role`、`message.content`（文字列 or ブロック配列：`text` / `tool_use` / `tool_result`）、`message.model`、`message.usage`、`tool_result.is_error`、`isMeta`。
- 実装前に、自分の環境の実ファイルを数件読んで構造を確認し、匿名化したサンプルを `testdata/` に置く。

### 抽出するシグナル

- **訂正発言**：tool_result ではない user のテキストのうち、訂正パターンに一致するもの（例：「違う」「そうじゃない」「じゃなくて」「やめて」「間違」「no,」「don't」「instead」「actually」）。パターンは設定で編集できるようにする。
- **ツールエラー**：`is_error: true` の tool_result。ツール名と先頭行を記録する。
- **頻出コマンド**：Bash の tool_use の `command` の先頭トークン。

## 5. SQLite スキーマ（初版）

```sql
files(path TEXT PRIMARY KEY, size INTEGER, mtime_ns INTEGER, broken_lines INTEGER);
sessions(id TEXT PRIMARY KEY, project TEXT, file_path TEXT, cwd TEXT, git_branch TEXT,
         started_at TEXT, ended_at TEXT, user_msgs INT, assistant_msgs INT,
         tool_errors INT, corrections INT,
         input_tokens INT, output_tokens INT, cache_read INT, cache_create INT);
model_usage(session_id TEXT, model TEXT, messages INT, input_tokens INT, output_tokens INT);
messages(id INTEGER PRIMARY KEY, session_id TEXT, project TEXT, ts TEXT, role TEXT,
         kind TEXT,   -- text | tool_use | tool_result
         tool TEXT, text TEXT, is_error INT, is_correction INT);
CREATE VIRTUAL TABLE messages_fts USING fts5(text, content='messages', content_rowid='id', tokenize='trigram');
```

- 取り込みは差分のみ：`files` の size/mtime が変わったファイルだけ、そのセッションを消してから入れ直す。
- DB の場所はアプリデータディレクトリ（macOS: `~/Library/Application Support/harness-app/index.db`）。DB を消しても再構築できること。

## 6. 機能

### MVP

| # | 画面 | 内容 |
|---|---|---|
| F1 | 概要ダッシュボード | 期間（7/30/90日）を選び、セッション数・メッセージ数・トークン量・訂正数・ツールエラー数の KPI、日別の活動グラフ、モデル別の使用量、プロジェクト別の分布、メモリファイル数を表示 |
| F2 | セッションブラウザ | プロジェクト・期間で絞り込んだ一覧、全文検索、セッション詳細（時系列表示、訂正とエラーをハイライト） |
| F3 | CLAUDE.md 改善 | ①対象を選ぶ（グローバル / プロジェクトの CLAUDE.md / CLAUDE.local.md / rules） ②期間・追加指示を入力 ③digest を生成してプレビュー（マスク済み） ④AI で改善案を生成 ⑤diff 表示、右側で手直しも可能 ⑥適用 or 破棄 |
| F8 | 変更履歴 | アプリが書き込んだ全ファイルのバックアップ一覧、適用前後の diff、ロールバック |
| F9 | 設定 | claude バイナリのパス（自動検出＋手動指定）、使用モデル、除外するプロジェクト、訂正パターン、マスキングの追加パターン、データディレクトリを開く、インデックスの再構築 |

### MVP 後

| # | 画面 | 内容 |
|---|---|---|
| F4 | メモリビューア | プロジェクトメモリの閲覧と編集（適用は applier 経由）、重複・古いメモリの検出 |
| F5 | 自己ナレッジ生成 | メモリと期間を選び、skill セット（SKILL.md + knowledge）を生成 → プレビュー → `~/.claude/skills/<name>/` に適用 |
| F6 | 日次振り返り | 指定時刻に当日分の日報と TODO を生成し、`<データディレクトリ>/reports/YYYY-MM-DD.md` に保存。アプリ起動中は内部スケジューラで実行し、オプションで launchd の plist を生成 |
| F7 | 拡張機能 | MCP / plugins / skills / hooks の一覧と状態（読み取り専用。実装済み）。MCP の env・headers は値を読まずキー名のみ表示し、コマンド・引数・URL は masking を通す |

## 7. AI 実行（runner）

- 入出力はファイルで受け渡す：`<work>/<job-id>/input.json` → skill → `<work>/<job-id>/output.md`
- `input.json`：`{ "digest": string, "currentClaudeMd": string, "instruction": string, "targetKind": "global|project|local|rule" }`
- 同梱 skill（`internal/skills` に embed し、実行時に `<work>/<job-id>/.claude/skills/` へ展開する）：
  - `claudemd-improve`：改善後の全文を `output.md` に書く。加えて変更理由を `rationale.md` に箇条書きで書く。既存の記述は理由なく消さない。重複は統合する。行数を増やしすぎない。
  - `skill-generate`（F5）：入力 `input.json` は `{ "skillName": string, "memories": [{ "label": string, "content": string }], "digest": string, "instruction": string }`（全てマスク済み）。出力は `output/SKILL.md`（必須）、`output/knowledge/*.md`（任意）、`rationale.md`。生成物は `.md` のみ・最大 20 ファイル・各 200KB まで、`output/` の外や不正なパスは拒否する。skill 名は小文字・数字・ハイフン（64 文字まで）。アプリは生成結果を diff で表示し、人が「適用」したときだけ `~/.claude/skills/<名前>/` に applier 経由で書き込む（ファイル単位でバックアップ・ロールバック可）。実行コマンドは `claudemd-improve` と同じ（`--allowedTools Read,Write`、cwd はジョブディレクトリ）。
  - `daily-report`（F6）: 当日 digest（マスク済み）から日報と TODO を `output.md` に書く
- 実行コマンド：`claude -p "<指示>" --allowedTools Read,Write --output-format json [--model <model>]`、cwd は `<work>/<job-id>`。
- タイムアウト（初期値5分）とキャンセルに対応する。stderr はジョブログに保存する。
- **claude バイナリの解決**：GUI アプリは PATH が最小限になるため、①設定値 ②macOS: `$SHELL -lc 'command -v claude'` / Windows: `where claude`（`.exe` を優先） ③既知の場所（macOS: `~/.claude/local/claude`、`~/.local/bin/claude`、`/opt/homebrew/bin/claude`、`/usr/local/bin/claude`、`~/.npm-global/bin/claude` / Windows: `%USERPROFILE%\.claude\local\claude.exe`、`%USERPROFILE%\.local\bin\claude.exe`、`%APPDATA%\npm\claude.cmd`、`%USERPROFILE%\.npm-global\claude.cmd`）の順で探す。実行可否は Windows では拡張子（`.exe` `.cmd` `.bat` `.com`）、それ以外は実行ビットで判定する。見つからなければ設定画面に誘導する。
- タイムアウト・キャンセル時は、macOS ではプロセスグループ、Windows では `taskkill /T` でプロセスツリーごと終了する。

## 8. 安全設計

1. `~/.claude` とプロジェクトのファイルは、applier 以外から書き込まない。
2. applier の手順：
   - 適用前に対象ファイルの SHA-256 を、改善案を生成した時点の値と比べる。違っていれば中止し、再生成を促す。
   - `<データディレクトリ>/backups/<timestamp>-<hash>/` に元ファイルと `manifest.json`（元パス、ハッシュ、日時、操作）を保存する。
   - 同じディレクトリに temp ファイルを書き、fsync → `os.Rename`。
   - 書き込みを許可するのは、targets が発見したパスと `~/.claude/skills/` 配下だけ（許可リスト方式）。
3. ロールバックも applier 経由で行い、ロールバック自体もバックアップを取る。
4. masking：`sk-ant-…`、`AKIA[0-9A-Z]{16}`、`ghp_` / `github_pat_`、`xox[baprs]-`、`Bearer …`、PEM 秘密鍵、メールアドレス、`(password|secret|token|api_key)\s*[:=]\s*\S+`（引用符つきの値は空白・エスケープ引用符を含めて全体）、引数形式の秘密（`--token 値`・`--api-key=値`・`--auth-token 値` のような接頭辞付きも。`--token-limit` など別語は対象外）を `[MASKED:<種別>]` に置き換える。ユーザーが追加パターンを設定できる。

## 9. 完了の定義（v0.1）

確認日: 2026-11-17（Linux/WSL 上のローカル + GitHub Actions macos-latest）

- [~] MVP（F1・F2・F3・F8・F9）が `wails dev` で動き、テスト用データで一通りの操作ができる
  - 実装済み。Go の E2E（service）と、mock API を使うフロントの画面テストで全操作を検証。
  - **未検証**: 実機の `wails dev`（WebView）での手動操作。WSL には webkit が無く、macOS 実機が必要。
- [x] `go vet ./...`・`go test ./...`・`npm run build`・`npm test` がすべて通る（ローカルと CI の両方）
- [x] collector 84%・digest 83%・masking 94%・applier 82%・runner 79%（`bash scripts/coverage.sh`、CI で強制）
- [x] フェイク claude を使った E2E（digest 生成 → 改善案 → 適用 → ロールバック）が通る（`internal/service` の `TestE2EProposalApplyRollback`）
- [~] `make dmg` で `build/bin/harness-app.dmg` ができ、起動できる
  - macos-latest の CI（dmg-smoke ジョブ）で `make dmg` 成功、universal バイナリと `hdiutil verify` を確認済み。
  - **未検証**: .dmg からの実際の起動（macOS 実機での確認が必要）。
- [~] GitHub Actions（macos-latest）でタグ push 時に .dmg がビルドされ、Release に添付される
  - `release.yml` を作成。ビルド手順は dmg-smoke で検証済み。**タグ push はユーザーが行う**ため Release 添付は未実行。
- [x] README にインストール手順（未署名アプリの開き方を含む）と使い方がある

### v0.1.0 リリース手順（ユーザーが実行）

1. 実機で `.dmg` を開いて起動し、上記の未検証項目を確認する。
2. `git tag v0.1.0 && git push origin v0.1.0`

### Windows 対応（#46）
- [x] Windows でビルドでき、テストが通る（windows-latest の CI。exe は `windows-build-smoke` でビルド）
- [~] Windows 配布物（インストーラ）が Release に添付される（`release.yml` に実装。タグ push 時に実行されるため未実行）
- [ ] Windows 実機で起動し、改善案の生成〜適用〜ロールバックができる（人の確認が必要）
