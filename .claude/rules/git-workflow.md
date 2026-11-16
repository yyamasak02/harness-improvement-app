---
description: ブランチ・コミット・PR・マージの手順
---
- ブランチ名: `feat/<task-id>-<slug>` / `chore/…` / `test/…`。
- 1 機能 = 1 PR。タスク完了ごとに `[x]` を付けてコミット。
- 手順: 関連 issue を探す（`scripts/related-issues.sh`）→ `git switch -c` → 実装 → `quality-gate` → `scripts/ship.sh`（verify --full・commit・push・PR 作成。**マージしない**）→ 独立レビュー（`review-pr` skill）→ 指摘対応・再レビュー → `scripts/merge-pr.sh`（承認と CI を検査して squash マージ）。
- `gh pr merge` を直接実行しない（権限で禁止）。レビュー前・承認前にマージしない。
- コミット末尾に Co-Authored-By 行、PR 本文末尾に Claude Code 生成表記を付ける。
- PR 本文: 概要・対象タスク ID・検証結果。
- CI があれば `gh pr checks --watch` 相当（`gh pr checks`）を確認してからマージ。
