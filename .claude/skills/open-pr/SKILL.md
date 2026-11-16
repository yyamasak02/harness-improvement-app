---
name: open-pr
description: 現在のブランチを push し、gh で PR を作成して squash マージし main を更新する。機能単位の完了時に使う。
---
1. `git status` がクリーンで `quality-gate` と `verify --full` が済んでいることを確認。PR タイトルは Conventional 形式（CI の pr-policy が検査）。本文は `.github/pull_request_template.md` のチェックリストに従い、関連 issue は `Closes #N`。
2. `git push -u origin <branch>`
3. `gh pr create --title "<prefix>: <要約>" --body "<概要/タスクID/検証結果>\n\n🤖 Generated with [Claude Code](https://claude.com/claude-code)"`
4. `gh pr checks` で CI があれば成功を確認（無ければスキップ）。
5. `gh pr merge --squash --delete-branch`
6. `git switch main && git pull`
force push・main 直 push は禁止。

## ショートカット
`bash scripts/ship.sh "<prefix>: <要約>" "<概要/タスクID/検証結果>"` が verify → commit → push → PR → squash マージ → main 更新までを実行する（推奨）。
