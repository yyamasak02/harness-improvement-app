---
name: daily-report
description: 当日のセッション digest から、日報と翌日の TODO を Markdown で output.md に書く。
---

# daily-report

カレントディレクトリの `input.json` を Read で読み、日報を作る。

## 入力 (`input.json`)
- `date`: 対象日（YYYY-MM-DD）
- `digest`: 当日のセッションログの要約（マスク済み）
- `instruction`: ユーザーの追加指示（空なら無視）

## 出力（Write で書く。次のファイルだけ）
- `output.md` … 次の見出しを持つ Markdown。
  1. `# 日報 <date>`
  2. `## やったこと` … digest から読み取れる事実の箇条書き
  3. `## つまずき・訂正` … ツールエラーや訂正発言から分かる課題
  4. `## TODO` … 翌日以降にやることのチェックボックス（`- [ ]`）

## 守ること
- digest にない事実を作らない。推測は「推測」と明記する。
- `[MASKED:...]` を含む値を推測して復元しない。個人情報・秘密を書かない。
- `input.json` 以外のファイルを読まない。`output.md` 以外を書かない。
