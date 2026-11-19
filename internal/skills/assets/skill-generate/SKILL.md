---
name: skill-generate
description: プロジェクトメモリとセッション digest から、再利用できる skill セット（SKILL.md と knowledge/*.md）を output/ に書き、作成理由を rationale.md に書く。
---

# skill-generate

カレントディレクトリの `input.json` を Read で読み、skill セットを作る。

## 入力 (`input.json`)
- `skillName`: 作る skill の名前（小文字・数字・ハイフン）
- `memories`: 選ばれたメモリの配列（`label` と `content`。マスク済み）
- `digest`: セッションログの要約（マスク済み。空のことがある）
- `instruction`: ユーザーの追加指示（空なら無視）

## 出力（Write で書く。次のファイルだけ）
1. `output/SKILL.md` … frontmatter（`name` は `skillName` と同じ、`description` は「何をする skill か・いつ使うか」を 1〜2 文）と、手順を簡潔に書く本文。
2. `output/knowledge/<topic>.md` … 繰り返し参照する知識（任意。1 トピック 1 ファイル、最大 5 つ）。SKILL.md から相対パスで参照する。
3. `rationale.md` … その skill を作る理由と、元にしたメモリ・digest の事実の箇条書き。

## 守ること
- 出力は Markdown（`.md`）だけ。実行可能なスクリプトや、`output/` の外へのファイルは作らない。
- メモリに繰り返し現れる、再利用価値のある知識・手順だけを入れる。単発の出来事は入れない。
- `[MASKED:...]` を含む値を推測して復元しない。個人情報・秘密を書かない。
- `input.json` 以外のファイルを読まない。`output/` と `rationale.md` 以外を書かない。
