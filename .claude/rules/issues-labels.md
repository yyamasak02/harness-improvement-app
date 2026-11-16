---
description: GitHub issue / PR のラベル規則と運用（issue を作る・PR を作るときに必ず従う）
---
# ラベル規則

定義の唯一の正は `.github/labels.tsv`。変更したら `bash scripts/sync-labels.sh` で同期する。
GitHub 既定のラベル（bug / enhancement など）は使わない。**必ず次の接頭辞付きラベルを使う。**

| 軸 | ラベル | 付け方 |
|---|---|---|
| 種別 `type/` | feature / bug / tech-debt / test / docs / chore / verification | **必須・ちょうど 1 つ** |
| 領域 `area/` | collector / store / digest / masking / targets / applier / runner / service / frontend / ci / release / harness | **必須・1 つ以上**（変更するパッケージに対応） |
| 優先度 `priority/` | p0 / p1 / p2 / p3 | **issue は必須・ちょうど 1 つ**（PR は不要） |
| 状態 `status/` | ready / blocked / needs-human | 任意。人の作業が要るものは `needs-human`、停止中は `blocked`（理由をコメント） |
| リリース `release/` | v0.1 / v0.2 / later | issue は必須・ちょうど 1 つ |

## 優先度の基準
- p0: データ破壊・秘密情報の流出・リリースを止める不具合
- p1: 次のリリースまでに必須
- p2: 通常のバックログ
- p3: 余裕があれば

## issue の作り方
- 未着手・保留・残課題は **issue にする**（`docs/tasks.md` のチェック漏れや口頭の TODO にしない）。`create-issue` skill を使う。
- タイトルは `<prefix>: <要約>`（prefix は PR と同じ feat/fix/test/refactor/docs/chore）。
- 本文は「背景・やること・完了条件（チェックボックス）」。関連する仕様節やタスク ID（T50 等）を書く。
- 秘密情報・個人情報（メールアドレス等）を本文に書かない。
- リファクタリングやテスト不足を見つけたが今回の PR に含めない場合は、`type/tech-debt` または `type/test` で issue を作り、PR 本文からリンクする。

## PR のラベル
- PR は `type/*` と `area/*` を付ける（`.github/labeler.yml` が area を自動付与、type は PR タイトルの prefix から自動付与）。
- PR タイトルは Conventional 形式: `feat|fix|refactor|test|docs|chore|ci(: |(scope): )要約`。
- 対応する issue があれば本文に `Closes #N` を書く。
