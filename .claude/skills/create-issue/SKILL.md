---
name: create-issue
description: 未着手課題・保留事項・技術的負債を、ラベル規則（.claude/rules/issues-labels.md）に従って GitHub issue として作成する。
---
1. `.claude/rules/issues-labels.md` と `.github/labels.tsv` を読み、ラベルを決める（type=1つ / area=1つ以上 / priority=1つ / release=1つ、必要なら status）。
2. 重複確認: `gh issue list --state open --search "<キーワード>"`
3. 本文ファイルを作る（背景 / やること / 完了条件のチェックボックス / 関連 T-ID・仕様節）。秘密情報・メールアドレスを書かない。
4. 作成:
   `gh issue create --title "<prefix>: <要約>" --body-file <file> --label type/..,area/..,priority/..,release/..`
5. ラベルが無いと失敗する。その場合は `bash scripts/sync-labels.sh` を先に実行する。
6. 作った issue 番号を PR 本文・`docs/tasks.md` から参照する。
