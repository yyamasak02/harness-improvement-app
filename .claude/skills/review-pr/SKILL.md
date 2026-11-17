---
name: review-pr
description: PR を独立レビューエージェント（code-reviewer）にレビューさせ、指摘を直し、承認を得てからマージする。コード変更の PR は必ずこの手順を経る。
---
実装者本人のコンテキストでの自己レビューは**レビューとして数えない**。必ず別のエージェントに実行させる。

1. PR を作る（まだマージしない）: `bash scripts/ship.sh "<prefix>: <要約>" "<本文: Closes #N / 概要 / 品質チェック>"`
   - 本文に関連 issue（`Closes #N` / `Refs #N`、依頼由来の変更は必ず issue 化。例外は `[no-issue]: <依存更新|誤字修正|生成物の更新|リリース準備>: 理由`）が必要。探し方: `bash scripts/related-issues.sh`
2. **独立レビューを起動する**: Agent ツールで `subagent_type: "code-reviewer"`（定義: `.claude/agents/code-reviewer.md`）を起動し、プロンプトに PR 番号を渡す。
   - `code-reviewer` が使えない環境では general-purpose エージェントに「`.claude/agents/code-reviewer.md` と `docs/review-guidelines.md` に完全に従ってレビューし、GitHub にコメントを投稿して」と指示する。
   - 実装時の文脈（意図の説明・言い訳）を渡さない。渡すのは PR 番号だけ。
3. 結果を GitHub で確認する: `bash scripts/gh-read.sh pr <N>`（最後のコメントのマーカー `verdict=` を見る）。GitHub Actions の `claude-review` が有効なら、そのコメントも同じ基準で扱う。
4. `REQUEST_CHANGES` なら、blocker / major を全て直す（テストも）。minor は直すか `create-issue`（`type/tech-debt`）にする。直したら `bash scripts/ship.sh ...`（既存 PR には push だけ行う）→ **手順 2 に戻って再レビュー**（head が進むと過去の承認は無効）。
   - 3 周しても APPROVE にならない場合は `docs/blockers.md` に記録して次へ。
5. APPROVE を得たら `bash scripts/merge-pr.sh [N]`（head への承認マーカーと CI 成功を検査してから squash マージ。`gh pr merge` の直接実行は禁止）。
6. レビューが挙げたフォローアップ候補は `create-issue` で起票する。レビューで見つかった再発しうる問題は `scripts/check-rules.sh` に機械化する（`quality.md` 7）。

## 1 MR 1 レビューの意味
- 各 PR は必ず独立レビューを受ける。承認は head コミットに紐づくので、修正を push したら新しい head に対して再度レビューを受ける。
- 起動するたびに新しいレビューエージェントを使い、実装時の文脈は渡さない。
