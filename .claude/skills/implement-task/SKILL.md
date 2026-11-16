---
name: implement-task
description: docs/tasks.md の次の未完了タスク（または機能グループ）を、テスト先行で実装し PR マージまで進める標準手順。
---
1. `docs/tasks.md` で最初の未完了 `[ ]` を特定し、同じ PR にまとめるべき関連タスクを決める。`docs/spec.md` の該当節を読む。
2. **関連 issue を探す**: `bash scripts/related-issues.sh`（該当 issue は PR 本文で `Closes/Refs #N`。無ければ `create-issue`）。
3. `git switch main && git pull && git switch -c feat/<id>-<slug>`
4. テストを書く（テーブル駆動、testdata / 注入パスのみ）→ 実装。`.claude/rules/` に従う。
5. `quality-gate` skill（テストの過不足・リファクタリング・ドキュメント同期）→ `verify` skill（`bash scripts/verify.sh --full`）を実行して全て通す。
6. `docs/tasks.md` を `[x]` に更新。迷った判断は `docs/decisions.md` に 1 行。
7. `git add` → `git commit`（日本語、Co-Authored-By 付き）。
8. `review-pr` skill: `scripts/ship.sh` で PR 作成 → 独立レビュー → 指摘対応・再レビュー → `scripts/merge-pr.sh`。
9. 次のタスクへ。止まらない。CLAUDE.md の「やってはいけないこと」を毎回確認する。
