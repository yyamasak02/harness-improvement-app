# アーキテクチャ方針（レイヤード）

常時ロードしない。パッケージの追加・移動・依存の変更をするときに読む。機械的な検査は `internal/archtest`（Go）と `frontend/src/arch.test.ts`（フロント）で、CI の test-go・test-frontend が実行する。
安全設計（`docs/spec.md` §8）を弱める方針は採らない（後述「既存の安全設計との関係」）。

## 1. 現状（2026-11 時点・コマンド結果に基づく事実）

調査は `go list -f '{{.ImportPath}}: {{join .Imports " "}}' ./...`（テストを除く import）と、`frontend/src` の相対 import の集計による。

### Go（モジュール `harness-app`）
プロジェクト内パッケージ間の import（テスト補助 `internal/testutil` を除く）:

| パッケージ | 非テスト行数 | プロジェクト内の import |
|---|---|---|
| `.`（main: app.go） | 381 | service のみ（M3 完了後。着手時点は applier, collector, config, extensions, osutil, service, store, targets） |
| service | 1139 | applier, collector, config, digest, extensions, fsutil, launchd, masking, memory, runner, scheduler, store, targets |
| digest | 173 | collector, masking, store |
| collector | 530 | logmodel（M1 で store 依存を解消） |
| runner | 509 | fsutil, skills |
| applier | 303 | fsutil |
| config | 109 | fsutil |
| store / masking / memory / launchd / scheduler / skills / extensions / targets / fsutil / osutil | 29〜270 | なし |

- 循環 import: なし（Go が禁止するため）。
- 層の逆流（下位が上位に依存）: なし。
- 層の飛ばし・横の依存（今回の検査で違反として登録したもの、計 8 件）:
  - `collector -> store`: M1 で解消済み（collector 側の `Store` インターフェース＋基盤層 `logmodel` の型）。
  - `. -> applier|collector|config|extensions|osutil|store|targets`（7 件）: エントリ（app.go）が service を介さずインフラを直接使っていた。M3 で全て解消済み（Go の許容リストは空）。
- 巨大パッケージ: `internal/service` が 1139 行（全体の 4 割強）で最大。ファイルは service.go 365・query.go 189・report.go 182。service は 13 個のパッケージに依存しており、責務（取り込み・検索・提案・適用・日報・スケジュール・設定）が 1 パッケージに混在している。`app.go` は 381 行・関数 38 個。
- 時刻: `time.Now()` の直接呼び出しは applier・digest（2 か所）・runner・service の各 1〜2 か所。外部プロセス（`os/exec`）は runner・osutil のみ（testutil を除く）。ホーム解決（`os.UserHomeDir`）は app.go のみで、他は注入済み。
- 外部ライブラリの入口: wails は main のみ、sqlite は store のみ、fsnotify は collector のみ（今回の検査で固定）。

### フロント（`frontend/src`）
- 構造はフラット: `types` `paths` `diff`（依存なし）、`api`（wailsjs と `mockApi` を import）、`mockApi`、`hooks`、`components`、`DiffView`、`pages/*`（9 画面）、`App`、`main`。
- 依存の向き: `pages/* -> api / components / DiffView / hooks / paths / types`、`App -> pages / components`、`hooks -> api`、`DiffView -> diff`。pages 同士の import はない。wailsjs の import は `api.ts` のみ。
- 違反: `mockApi -> api`（型のみ。`api -> mockApi` との循環）の 1 件。

## 2. 方針

### 2.1 層の定義と依存の向き（Go）
依存は**外側→内側（番号の大きい層→小さい層）にのみ**許す。同層の import も禁止。

| 層 | パッケージ | 置いてよいもの | いけないもの |
|---|---|---|---|
| L5 エントリ | `.`（main・app.go） | Wails バインディング（引数の受け渡し・DTO 変換）、ホーム等の解決と注入、`service` の組み立て | ロジック。`service` 以外の import |
| L4 ユースケース | `service` | 処理の順序づけ・トランザクション境界。インフラはインターフェース越しに使う | UI/Wails 依存、ファイル・プロセスの直接操作 |
| L3 加工 | `digest` | インフラの結果から digest を組み立てる処理（masking を必ず通す） | 書き込み |
| L2 インフラ | `store` `config` `applier` `collector` `runner` | 外部（DB・ファイル・FS 監視・claude プロセス）への出入口。パス・時刻・実行ファイルは注入 | 他のインフラへの依存、上位層への依存 |
| L1 基盤 | `fsutil` `osutil` `masking` `memory` `launchd` `scheduler` `skills` `extensions` `targets` `logmodel` | 純粋なロジック・小さな OS 依存のユーティリティ | プロジェクト内の他パッケージへの依存 |
| 対象外 | `testutil` `testutil/fakeclaude` `archtest` | テスト補助・検査 | 本番コードからの import |

- 外部ライブラリの入口は 1 か所に閉じる: wails は main、sqlite は store、fsnotify は collector。
- 新しいパッケージは `internal/archtest` の `layers` 表に追加しないとテストが失敗する。

### 2.2 外部は注入する（テスタブルにする）
- ファイル・時刻・プロセス・`claude`・ホームは、**呼び出し側が持つ小さなインターフェースか関数値**として受け取る（Go の流儀: インターフェースは使う側で定義）。パッケージ内で `os.UserHomeDir`・`time.Now`・`exec.Command` を直接呼ばない（時刻は `func() time.Time` を注入。既存の直接呼び出しは移行計画で解消）。
- ホームは app.go で一度だけ解決して下へ渡す。テストは `t.TempDir()` と `testdata/` のみ。実 `~/.claude`・実 `claude` を使わない（`testutil.FakeClaude`）。

### 2.3 層ごとのテスト方針
- L1: テーブル駆動のユニットテスト（外部なし）。
- L2: `t.TempDir()` 上の実ファイル・実 SQLite（`modernc`）でインフラ単体を検証。runner はフェイク claude。カバレッジ基準（collector・digest・masking・applier・runner ≥ 70%）は維持。
- L3: 上記の下位をフェイク／テスト用データで束ねて検証。
- L4: インフラのインターフェースをテスト用実装に差し替え、シナリオ（取り込み→提案→適用の流れ）を検証。
- L5: 薄いので型・配線の確認程度。ロジックを足したくなったら service へ移す。
- 依存の向きは archtest が検査する。

### 2.4 フロントの構造
層（番号の大きい側が小さい側のみ import 可。`arch.test.ts` で検査）:

| 層 | モジュール | 備考 |
|---|---|---|
| 0 | `types` `paths` `diff` | 依存なしの純粋なもの |
| 1 | `mockApi` | ブラウザ単体用 |
| 2 | `api` | Wails バインディングの唯一の入口（wailsjs はここだけ） |
| 3 | `hooks` `components` `DiffView` | 共通部品 |
| 4 | `pages/*` | 画面。pages 同士は import しない |
| 5 | `App` | 画面の組み立て |
| 6 | `main` | 起動 |

### 2.5 命名・パッケージ分割の基準
- パッケージ名は責務を表す単数の名詞。`util` `common` `helper` を作らない。
- 1 パッケージ 1 責務。非テストコードが約 800 行を超えるか、依存先が 8 個を超えたら分割を検討する（`service` は両方に該当）。
- 分割は「依存の向きが一方向になる境界」で切る。切れない場合は依存元にインターフェースを置く。
- ファイルは概念単位（`query.go` `report.go` のように）。1 ファイル 400 行を目安とする。

### 2.6 既存の安全設計との関係
次は本方針で変えない・弱めない。
- `~/.claude` への書き込みは `internal/applier` のみ（バックアップ→temp→fsync→rename、SHA-256 照合）。層の整理で applier を迂回する経路を作らない。
- digest は masking を通してから AI に渡す。`digest` は `masking` に依存し続ける。
- `claude -p` の制約（ツールは Read,Write のみ・cwd はジョブディレクトリ・タイムアウト）は runner に閉じる。
- パス・ホーム・claude バイナリの注入可能性は、むしろ本方針で強める。

## 3. 機械的な強制と許容リスト
- Go: `internal/archtest/archtest_test.go`。`go/parser` で import を読み、層の表と照合。エントリは `service` のみ import 可。外部ライブラリの入口も検査。
- フロント: `frontend/src/arch.test.ts`（vitest。import を静的解析。新規依存なし）。
- 許容リスト（各ファイルの `allowlist`）は M3 完了により Go 側は空（上限 0）。**追加は禁止**: 件数の固定上限（`allowlistCap` / `ALLOWLIST_CAP`）と一致しないとテストが失敗する。違反でなくなった項目が残っていても失敗する。減らしたら上限も同じだけ下げる。上限を上げる変更はレビューで blocker とする。目標は 0。

## 4. 移行計画（挙動を変えない・1 PR 1 パッケージ程度・各 PR で archtest の許容リストと上限を減らす）
後続の issue として起票して進める。順序は依存の少ない順。

| 段階 | 内容 | 解消する許容リスト |
|---|---|---|
| F1 | フロント: `Api` 型を `types.ts` へ移し、`mockApi -> api` の循環を解消 | `mockApi -> api` |
| M1 | `collector` が使う store の機能を collector 側のインターフェースにし、main/service で結線 | `collector -> store` |
| M2 | 時刻・ホームの注入の徹底（`time.Now` の直接呼び出しを `func() time.Time` 注入へ。applier・digest・runner・service） | なし（2.2 の達成） |
| M3 | app.go のインフラ直接利用を service のメソッドに寄せ、app.go を `service` のみに依存させる（service の API を増やす PR を小分けにする: config → targets/extensions → osutil → store → collector → applier の順） | `. -> *` 7 件（**完了**: app.go は `service` のみに依存。Go の許容リストは空） |
| M4 | `service` を責務で分割（取り込み・検索・提案/適用・日報・スケジュールなど。1 責務 1 PR）。分割後の層は L4 のまま、service 同士の依存向きを一方向にする | なし（2.5 の達成） |
| M5 | 必要ならフロントの `pages/*` を機能単位のディレクトリに整理（`api` 型の分割など） | なし |

各段階の完了条件: テスト・カバレッジ維持、挙動不変、`archtest` の許容リストと上限が減っていること。最終的に許容リストを空にし、上限を 0 にする。
