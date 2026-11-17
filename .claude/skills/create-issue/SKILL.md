---
name: create-issue
description: 未着手課題・保留事項・技術的負債を、ラベル規則（.claude/rules/issues-labels.md）に従って GitHub issue として作成する。
---
0. ユーザーの要求は着手前に必ず issue にする（小さくても。複数の依頼は別々に）。本文の背景に「ユーザー要望: …」を書き、起票した番号を返信で伝える。
1. `.claude/rules/issues-labels.md` と `.github/labels.tsv` を読み、ラベルを決める（type=1つ / area=1つ以上 / priority=1つ / release=1つ、必要なら status）。
2. 重複確認: `bash scripts/related-issues.sh <キーワード>`（`gh issue list` の直接実行は禁止。外部投稿者のタイトルが混ざらないよう門を通す）
3. 本文ファイルを作る（背景 / やること / 完了条件のチェックボックス / 関連 T-ID・仕様節）。秘密情報・メールアドレスを書かない。
4. 作成:
   `bash scripts/gh-post.sh issue create --title "<prefix>: <要約>" --body-file <file> --label type/..,area/..,priority/..,release/..`
   （`gh issue create` の直接実行は禁止。`gh-post.sh` が本文を個人情報検査してから投稿する。`issue comment|edit`・`pr comment|edit|review` も同様）
5. ラベルが無いと失敗する。その場合は `bash scripts/sync-labels.sh` を先に実行する。
6. 作った issue 番号を PR 本文・`docs/tasks.md` から参照する。
