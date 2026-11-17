---
name: release
description: リリース（バージョン確定・CHANGELOG・タグ作成・Release 確認）の手順。リリース準備を頼まれたとき、または着手できる issue が無くリリース可否を確認するときに使う。タグの push は人が行う。
---
前提: `.claude/rules/release.md` を読む。

## 1. 事前確認
- `bash scripts/status.sh`（open PR が無い・main が最新・CI 成功）
- `bash scripts/status.sh`（open な issue を優先度順に表示。`gh issue list` の直接実行は禁止）で p0 と p1。着手できるもの（needs-human 以外）は先に片付ける。人の作業待ち（macOS / Windows 実機確認など）が残っていれば、**リリースを止めてユーザーに報告**する。
- `[Unreleased]` が空でないこと（`CHANGELOG.md`）。

## 2. リリース準備 PR
1. `bash scripts/release.sh prepare X.Y.Z`（main 上で実行。`release/vX.Y.Z` ブランチを作り、バージョンと CHANGELOG を更新して commit）
2. `bash scripts/ship.sh "chore: vX.Y.Z のリリース準備" "<本文>\nRefs #<リリース issue>\n[no-test-needed]: バージョンと CHANGELOG のみ"`
3. `review-pr` skill（独立レビュー → 指摘対応 → 再レビュー）→ `bash scripts/merge-pr.sh`

## 3. タグ作成
1. `git switch main && git pull`
2. `bash scripts/release.sh tag X.Y.Z`（クリーンな main・CI 成功・バージョン整合・CHANGELOG の節を検査し、ローカルに注釈付きタグを作る）
3. **ユーザーに依頼する**: 「`git push origin vX.Y.Z` を実行してください」（エージェントは push しない）。未解決の p0/p1 の一覧も添える。

## 4. push 後の確認（ユーザーが push したあと）
- `gh run list --workflow Release --limit 1` → 完了まで待つ（実行 ID を固定して Monitor）
- `gh release view vX.Y.Z` で、`harness-app.dmg`・`*installer.exe` が添付され、ノートが CHANGELOG と一致していること
- 失敗したら原因を直して **PATCH 版**で出し直す（既存タグは変更しない）。`docs/handoff.md` を更新し、結果を関連 issue（#26 など）にコメントする。
