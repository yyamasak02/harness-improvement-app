# タスク

上から順に実装する。1タスク = 1コミット。完了したら `[x]` にする。
各タスクは「テストを書く → 実装 → `go vet`/`go test`/フロントのビルドを通す → コミット」の順で進める。

## Phase 0: 準備

- [x] T00 `wails init -n harness-app -t react-ts` で雛形を作る。Tailwind・Vitest・Recharts・jsdiff・react-markdown を導入する。`Makefile`（dev / test / build / dmg）を作る
- [x] T01 実環境の `~/.claude/projects/` の jsonl を数件**読むだけ**で構造を確認する。匿名化したサンプルを `testdata/claude/projects/...` に作る（正常行、壊れた行、tool_use、tool_result のエラー、訂正発言、複数モデルを含める）。サンプルのメモリと CLAUDE.md も置く
- [x] T02 `scripts/fake-claude.sh` を作る：引数を記録し、`input.json` を読んで決まった `output.md` と `rationale.md` を書く。異常系（非ゼロ終了、タイムアウト）を環境変数で切り替えられるようにする

## Phase 1: バックエンドの基盤

- [x] T10 `internal/config`：設定の構造体、アプリデータディレクトリの解決、JSON の読み書き（temp → rename）
- [x] T11 `internal/store`：SQLite のオープン、マイグレーション、FTS5 trigram の動作確認テスト
- [ ] T12 `internal/collector`：jsonl を寛容にパースし、sessions / messages / model_usage を生成。訂正検出と、エラー・頻出コマンドの抽出
- [ ] T13 `internal/collector`：差分取り込み（files テーブル）。fsnotify による監視（デバウンス付き）
- [ ] T14 `internal/masking`：パターン群とユーザー追加パターン。テーブル駆動テスト
- [ ] T15 `internal/targets`：プロジェクトの cwd から CLAUDE.md / CLAUDE.local.md / rules を発見する。グローバルの CLAUDE.md も含める
- [ ] T16 `internal/digest`：プロジェクト・期間を指定して Markdown の digest を生成（統計、訂正の抜粋、エラーのまとめ、頻出コマンド）。サイズ上限付き、masking 済み
- [ ] T17 `internal/applier`：許可リスト、ハッシュ確認、バックアップ、temp → rename、ロールバック、履歴一覧
- [ ] T18 `internal/skills`：`claudemd-improve` の SKILL.md を embed し、展開する処理
- [ ] T19 `internal/runner`：claude バイナリの解決、ジョブディレクトリ、実行、タイムアウト・キャンセル、出力の読み取り。fake-claude でテスト

## Phase 2: バインディング

- [ ] T20 `app.go`：Settings / Reindex / Overview / ListProjects / ListSessions / GetSession / Search / ListTargets / ReadTarget / BuildDigest / GenerateProposal / ApplyProposal / ListBackups / Rollback / CheckClaude を公開する
- [ ] T21 起動時に自動でインデックスを作り、進捗を Wails イベントで通知する
- [ ] T22 E2E テスト（Go）：testdata + fake-claude で digest 生成 → 改善案 → 適用 → ロールバックまで通す

## Phase 3: フロントエンド

- [ ] T30 `src/api.ts`：Wails バインディングのラッパーと、ブラウザ用の mock。共通レイアウト（サイドバー＋ルーティング）
- [ ] T31 F1 概要ダッシュボード
- [ ] T32 F2 セッションブラウザ（一覧・検索・詳細）
- [ ] T33 F3 CLAUDE.md 改善：対象選択 → digest プレビュー → 生成中表示（キャンセル可） → diff ビュー（左右分割、右側は編集可） → 適用・破棄。rationale も表示する
- [ ] T34 F8 変更履歴：一覧、diff、ロールバック（確認ダイアログあり）
- [ ] T35 F9 設定画面。claude が見つからないときの案内
- [ ] T36 Vitest：diff ユーティリティ、api mock、主要コンポーネントのテスト

## Phase 4: 配布

- [ ] T40 アプリ名・アイコン・`wails.json` の info を整える。`make dmg`（`wails build -platform darwin/universal -clean` → `hdiutil create`）
- [ ] T41 `.github/workflows/ci.yml`：push 時に macos-latest で go test と npm test を実行
- [ ] T42 `.github/workflows/release.yml`：`v*` タグの push で .dmg をビルドし、GitHub Release に添付
- [ ] T43 README：機能、スクリーンショット欄、インストール（未署名のため右クリック→開く、または `xattr -dr com.apple.quarantine`）、使い方、データの保存場所、安全設計
- [ ] T44 完了の定義（spec.md §9）を全項目確認し、`v0.1.0` タグの準備をする（タグの push はユーザーが行う）

## Phase 5: MVP 後（v0.2 以降）

- [ ] T50 F4 メモリビューア
- [ ] T51 F5 自己ナレッジ生成（`skill-generate`）
- [ ] T52 F6 日次振り返り（`daily-report`、スケジューラ、launchd の plist 生成）
- [ ] T53 F7 拡張機能一覧
