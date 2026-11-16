---
name: implement-task
description: docs/tasks.md の次の未完了タスク（または機能グループ）を、テスト先行で実装し PR マージまで進める標準手順。
---
1. `docs/tasks.md` で最初の未完了 `[ ]` を特定し、同じ PR にまとめるべき関連タスクを決める。`docs/spec.md` の該当節を読む。
2. `git switch main && git pull && git switch -c feat/<id>-<slug>`
3. テストを書く（テーブル駆動、testdata / 注入パスのみ）→ 実装。`.claude/rules/` に従う。
4. `quality-gate` skill（テストの過不足・リファクタリング・ドキュメント同期）→ `verify` skill（`bash scripts/verify.sh --full`）を実行して全て通す。
5. `docs/tasks.md` を `[x]` に更新。迷った判断は `docs/decisions.md` に 1 行。
6. `git add` → `git commit`（日本語、Co-Authored-By 付き）。
7. `open-pr` skill で push → PR → squash マージ。
8. 次のタスクへ。止まらない。CLAUDE.md の「やってはいけないこと」を毎回確認する。
