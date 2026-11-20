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

## インストール

> **現在、正式リリース（インストーラ付きの Release）はまだ公開されていません。** 公開されるまでは、先に「[リリースが出るまでの暫定手順（ソースからビルド）](#リリースが出るまでの暫定手順ソースからビルド)」を使ってください。以降の macOS / Windows の手順は **リリース公開後** のものです。
>
> Release ページには GitHub が自動で「Source code (zip / tar.gz)」を付けますが、それはソースコードです。リリース公開後に使うのは、下記のインストーラ（`.dmg` / `-installer.exe`）だけです。

### リリースが出るまでの暫定手順（ソースからビルド）

リポジトリを取得し、前提ツール（Go 1.24 以降、Node.js、[Wails v2](https://wails.io/)）を入れてからビルドします。

```sh
git clone <このリポジトリの URL>
cd harness-improvement-app
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
make dmg      # macOS: build/bin/harness-app.dmg ができる（未署名。開き方は下の macOS 節の手順 2）
make linux    # Linux / WSL: build/bin/harness-app ができる（詳細は下の Linux 節）
```

Windows は、Wails の前提（WebView2 など）を満たした環境で `wails build -platform windows/amd64` を実行します。開発用に試すだけなら `wails dev` でも起動できます。

### macOS（リリース公開後）

1. [Releases](../../releases) から `harness-app.dmg` をダウンロードして開き、`harness-app.app` を「アプリケーション」へドラッグします。
2. **本アプリは署名・公証をしていません。** 初回は次のどちらかで開きます。
   - Finder で `harness-app.app` を **右クリック →「開く」** → 確認ダイアログで「開く」
   - またはターミナルで隔離属性を外します。
     ```sh
     xattr -dr com.apple.quarantine /Applications/harness-app.app
     ```
3. [Claude Code](https://claude.com/claude-code) がインストールされている必要があります。見つからない場合は「設定」でフルパスを指定してください（`which claude` で分かります）。

### Windows（10 以降。リリース公開後）

1. [Releases](../../releases) から `harness-app-amd64-installer.exe`（インストーラ）をダウンロードして実行します。WebView2 ランタイムが無ければ自動で案内されます。
2. **本アプリは署名していません。** 初回は SmartScreen が「Windows によって PC が保護されました」と表示します。**「詳細情報」→「実行」** を選んでください。
3. [Claude Code](https://claude.com/claude-code) をインストールしておきます。自動検出されない場合は「設定」でフルパスを指定してください（PowerShell で `where.exe claude` を実行すると分かります。`claude.exe` を指定してください）。

### Linux / WSL（WSLg）

#### .deb で入れる（Ubuntu 22.04 以降・Debian 系、リリース公開後）

[Releases](../../releases) から `harness-app_<バージョン>_amd64.deb` をダウンロードし、`sudo apt install ./harness-app_*_amd64.deb` で入れます（`libgtk-3-0` と `libwebkit2gtk-4.1-0` は自動で入ります）。本パッケージは署名していません。apt リポジトリは提供していません（検討中: #124）。

#### ソースからビルドする（WSL2 + WSLg の Windows 11、または Linux デスクトップ）

```sh
sudo apt install build-essential libgtk-3-dev libwebkit2gtk-4.1-dev   # Ubuntu 22.04 以降
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
make linux                                                           # build/bin/harness-app ができる
./build/bin/harness-app
```

- WSL では、Windows 側ではなく **WSL 内の `~/.claude`**（Claude Code を WSL で使っている場合のログ・CLAUDE.md）が対象になります。`claude` も WSL 内のものを使います（見つからなければ「設定」でフルパス指定）。
- WSLg が無い環境（Windows 10 など）ではウィンドウが開きません。Windows 版を使ってください。
- Windows 版の Smart App Control の影響は受けません（Linux のバイナリなので）。
- データの保存場所は `~/.config/harness-app/`（`XDG_CONFIG_HOME` があればその配下）です。フォルダを開く機能は `xdg-open` を使います（WSL では `sudo apt install xdg-utils wslu` などで開けるようにしてください）。

## 使い方

1. 起動すると `~/.claude/projects/` を自動で取り込みます（差分のみ。以降はファイル変更を監視して追従します）。
2. 「概要」「セッション」で状況を眺めます。
3. 「CLAUDE.md 改善」で対象（グローバル / プロジェクトの CLAUDE.md / CLAUDE.local.md / rules）と期間、必要なら追加指示を選び、digest を確認してから「改善案を生成」します。
4. 差分を確認し、必要なら右側を直接編集して「適用」します。気に入らなければ「破棄」します。
5. 元に戻したいときは「変更履歴」からロールバックします。

## データの保存場所

| 内容 | 場所（macOS。Windows は `%APPDATA%\harness-app\` 配下の同名） |
|---|---|
| 設定 | `~/Library/Application Support/harness-app/config.json`（Windows: `%APPDATA%\harness-app\config.json`） |
| 索引 DB（消しても再構築できます） | `~/Library/Application Support/harness-app/index.db` |
| バックアップ・変更履歴 | `~/Library/Application Support/harness-app/backups/<時刻>-<ハッシュ>/` |
| AI 実行のジョブ（入出力・ログ） | `~/Library/Application Support/harness-app/work/<job-id>/` |

## 更新の方法

新しいバージョンは [Releases](../../releases) から入手し、**今の導入の上から入れ直す**だけです。設定・索引 DB・バックアップ/変更履歴は、上の「データの保存場所」（インストール先とは別の場所）にあるため、更新でもアンインストールでも消えません。現在のバージョンは、アプリの「設定」の「バージョン情報」で確認できます。

> **アプリは更新の確認も自動ダウンロードもしません**（外部通信は `claude -p` の実行時だけという方針のため）。新しいバージョンが出たかどうかは Releases ページで確認してください（アプリ内通知の検討: #173）。

更新の前に、アプリを終了してください（起動したままだと、実行ファイルを置き換えられません）。

| OS | 手順 | 確認状況 |
|---|---|---|
| macOS | 新しい `harness-app.dmg` を開き、`harness-app.app` を「アプリケーション」へドラッグして**置き換え**ます（「置き換える」を選択）。署名していないため、初回は上の macOS 節の手順 2 で開きます。 | **未検証**（macOS 実機では未確認。データがインストール先の外にあることはコードで確認済み） |
| Windows | 新しい `harness-app-amd64-installer.exe` を実行します。同じインストール先に上書きされ、ショートカットとアンインストール情報も更新されます（旧バージョンを先に消す必要はありません）。 | **未検証**（NSIS の標準の上書きに依存。CI でのサイレント実行はハングしたため入れていない。実機でも未確認。データがインストール先の外にあることはコードで確認済み） |
| Linux / WSL（.deb） | `sudo apt install ./harness-app_<新しいバージョン>_amd64.deb`（または `sudo dpkg -i`）。`0.1.0~rc.3` < `0.1.0~rc.4` < `0.1.0` の順で新しい版と認識されます。 | CI（ubuntu）で、同じ版の再導入 → 新しい版への更新、バージョン順序、ユーザーデータの残存を確認 |
| Linux（ソースからビルド） | 最新のソースを取得して `make linux` し直します。 | — |

### データの形式と、新旧バージョンの関係
- 設定（`config.json`）と索引 DB（`index.db`）は形式バージョンを持ちます。**古い形式は新しいアプリが自動で読み込み・移行**します（バージョン導入前の `0.1.0-rc.3` 以前のデータも読めます）。
- **新しい形式のデータは、古いアプリが開きません**（エラーを表示し、ファイルは書き換えません）。古いバージョンに戻す場合は、新しいバージョンで作られた設定ファイルを退避するか、アプリを新しいバージョンに戻してください。索引 DB は `~/.claude` から再構築できます。

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

コミット・push・PR 作成時には、個人情報・秘密（実在のメール・電話・ホームのパス・OS ユーザー名・トークンなど）の混入検査も走ります（`scripts/check-privacy.sh`）。GitHub への投稿は `scripts/gh-post.sh` 経由で本文を検査します。詳しくは [.claude/rules/security.md](.claude/rules/security.md)。

必要なもの: Go、Node.js 20+、[Wails v2](https://wails.io)（`go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`。**CLI のバージョンは必ず v2.10.2 に揃える**：別版の CLI は `frontend/wailsjs` や `build/` の生成物を書き換えます）。

```sh
make dev            # wails dev
make setup          # pre-commit フックの導入（初回のみ）
make test           # go vet / go test / フロントの build と test
make verify-full    # lint・ルール検査・カバレッジ基準まで（push 前・CI 相当）
bash scripts/coverage.sh   # collector・digest・masking・applier・runner のカバレッジ 70% 以上を確認
make bindings       # Go のメソッドを変えたら Wails バインディングを再生成
make dmg            # macOS 専用: build/bin/harness-app.dmg を出力
make linux          # Linux / WSL 用バイナリ（GTK3・WebKitGTK 4.1 が必要）
make windows        # Windows 用 exe（他 OS からもクロスビルド可）。インストーラは make windows-installer（NSIS が必要）
```

- ブラウザ単体の UI 確認: `cd frontend && npm run dev`（Wails が無いときは mock API に切り替わります）。
- テストは `testdata/` のダミーデータだけを使い、実際の `~/.claude` には触れません。`claude` はフェイク（`internal/testutil/fakeclaude`、Go 製でテスト時にビルド）に差し替えます。
- 仕様は [docs/spec.md](docs/spec.md)、タスクは [docs/tasks.md](docs/tasks.md)、開発ルールは [CLAUDE.md](CLAUDE.md) と `.claude/` を参照してください。

## リリース

バージョンの出典は `wails.json`、変更履歴は [CHANGELOG.md](CHANGELOG.md)（Keep a Changelog・SemVer）です。手順の詳細は [.claude/rules/release.md](.claude/rules/release.md) と `release` skill にあります。

```sh
bash scripts/release.sh prepare 0.1.0   # リリース用ブランチ作成・バージョン更新・CHANGELOG 確定 → PR → レビュー → マージ
bash scripts/release.sh tag 0.1.0       # main・CI 成功・整合を検査し、ローカルに注釈付きタグ v0.1.0 を作る
git push origin v0.1.0                  # タグの push は人が行う
```

タグを push すると GitHub Actions（`release.yml`）がバージョンと CHANGELOG の整合を検証し、macOS（`.dmg`）と Windows（インストーラ）をビルドして、CHANGELOG の該当節をノートとして Release に添付します（`-` を含むタグは pre-release）。

GitHub の Release 画面（Releases → Draft a new release）から作ることもできます。Release を**公開（Publish）**すると同じワークフローが起動し、インストーラをその Release に添付します（下書きのままでは起動しません）。タグ名は `v0.1.0` でも `0.1.0` でもよく、`wails.json` のバージョンと CHANGELOG の節（`## [0.1.0]`）が揃っている必要があります。この場合ノートは上書きされません。ビルドに失敗すると、Release のノートの先頭に失敗の表示と Actions へのリンクが付きます（再実行で成功すると取り除かれます）。

注意: ワークフローの定義は**タグが指すコミット**のものが使われます。この Release 画面からの添付に対応する前のコミットを指す既存のタグ（`v0.1.0-rc.1` など）では、公開してもインストーラは添付されません。修正を含む最新の `main` のコミットで新しいタグ（例 `v0.1.0-rc.2`）を作ってください。
