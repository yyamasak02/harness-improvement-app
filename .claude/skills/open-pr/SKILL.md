---
name: open-pr
description: 現在のブランチの PR を作成する（マージはしない）。マージは review-pr skill の独立レビュー承認後に scripts/merge-pr.sh で行う。
---
1. `git status` がクリーンで `quality-gate` と `verify --full` が済んでいることを確認。
2. `bash scripts/related-issues.sh` で関連 issue を確認する。
3. `bash scripts/ship.sh "<prefix>: <要約>" "<本文>"`
   - タイトルは Conventional 形式（CI の pr-policy が検査）。本文は `.github/pull_request_template.md` に従い、関連 issue は `Closes #N` / `Refs #N`（無ければ `[no-issue]: 理由`）。
   - verify --full → commit → push → PR 作成まで行う。**マージはしない。**
4. 続けて `review-pr` skill を実行する（独立レビュー → 指摘対応 → 再レビュー → `bash scripts/merge-pr.sh`）。
force push・main 直 push・`gh pr merge` の直接実行は禁止。
