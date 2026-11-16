---
description: ブランチ・コミット・PR・マージの手順
---
- ブランチ名: `feat/<task-id>-<slug>` / `chore/…` / `test/…`。
- 1 機能 = 1 PR。タスク完了ごとに `[x]` を付けてコミット。
- 手順: `git switch -c` → 実装 → `scripts/verify.sh` → `git add`/`commit` → `git push -u origin <branch>` → `gh pr create` → `gh pr merge --squash --delete-branch` → `git switch main && git pull`。
- コミット末尾に Co-Authored-By 行、PR 本文末尾に Claude Code 生成表記を付ける。
- PR 本文: 概要・対象タスク ID・検証結果。
- CI があれば `gh pr checks --watch` 相当（`gh pr checks`）を確認してからマージ。
