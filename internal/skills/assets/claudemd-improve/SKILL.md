---
name: claudemd-improve
description: セッション digest をもとに CLAUDE.md（または rules）の改善後の全文を output.md に書き、変更理由を rationale.md に書く。
---

# claudemd-improve

カレントディレクトリの `input.json` を Read で読み、改善案を作る。

## 入力 (`input.json`)
- `digest`: セッションログの要約（マスク済み）
- `currentClaudeMd`: 現在のファイル全文
- `instruction`: ユーザーの追加指示（空なら無視）
- `targetKind`: `global` | `project` | `local` | `rule`

## 出力（Write で書く。この 2 ファイルだけ）
1. `output.md` … 改善後の **全文**。前置き・コードフェンス・説明文を付けない。
2. `rationale.md` … 変更理由の箇条書き（1 変更 1 行、根拠となる digest の事実を添える）。

## 守ること
- 既存の記述は理由なく消さない。消す場合は rationale に理由を書く。
- 重複した記述は統合する。行数を増やしすぎない。
- digest の「訂正発言」「ツールエラー」「頻出コマンド」から、繰り返し起きている問題だけを反映する。単発の出来事は入れない。
- 「やってはいけないこと」は具体的・検証可能な形で書く。
- `[MASKED:...]` を含む値を推測して復元しない。
- `input.json` 以外のファイルを読まない。output.md / rationale.md 以外を書かない。
