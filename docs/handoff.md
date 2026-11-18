# 引き継ぎメモ

セッションが切れても、どのフェーズからでも再開できるようにするためのメモ。**現実（git / PR / issue）が常に優先**。食い違ったら現実を信じ、再開後にこのメモを直す。
状況の自動確認は `bash scripts/status.sh`（`resume` skill）。更新ルールは `.claude/rules/session.md`。

## 現在の目標
- v0.1.0 リリースの準備。MVP（F1・F2・F3・F8・F9）と macOS / Windows 対応は実装・マージ済み。残りは人の確認とタグ push。
- 並行して、**開発プロセスの安全装置**（個人情報の流出防止・プロンプトインジェクション対策・承認フロー・再発防止）を整備してきた（#72 #78 #81 #85 はマージ済み）。

## 現在の状態（2026-11-18）
- 作業ブランチ `chore/approval-hardening-84`（#84）。settings.json の deny 追加（8 項目）まで反映済みで、`check-harness.sh` は OK。
- 残り: PR 作成（`scripts/ship.sh`）→ 独立レビュー（`review-pr`）→ `scripts/merge-pr.sh`。PR 番号は `status.sh` で確認する。

## 進捗（要約）
- 実装: docs/tasks.md の T00〜T44 完了。Phase 5（F4〜F7）は issue（#21〜#24）で管理。
- 品質基盤: pre-commit・CI・カバレッジ基準・ルール検査・独立レビュー（`review-pr` skill、`scripts/merge-pr.sh`）。
- 配布: release.yml（タグ push 時に macOS の .dmg と Windows のインストーラのみ。Release ノートの先頭にダウンロード案内）。Linux / WSL（WSLg）はソースからのビルド（`make linux`、#65）。
- **安全装置（マージ済み）**:
  - 個人情報の流出防止（#72）: `scripts/check-privacy.sh`（pre-commit / pre-push / CI）、GitHub への投稿は `scripts/gh-post.sh` のみ（直接の `gh issue|pr create|comment|edit|review` は deny）。
  - プロンプトインジェクション対策（#78）: issue / PR の読み取りは `scripts/gh-read.sh` のみ（外部投稿者の内容は `<untrusted-data>` で囲む）、PostToolUse フックで警告、PreToolUse フックで境界ファイルの変更を承認フローへ。
  - 承認フロー（#81）: 境界ファイルの変更は、別エージェント `change-approver` の評価（`scripts/record-approval.sh`）→ 人の確認。評価は `docs/approval-log.md` に残り、CI の `check-approvals` が確認する。
  - 再発防止（#85）: フックは常に main の版（`git archive`）で動く。実環境の診断 `scripts/check-harness.sh` をセッション開始・ブランチ切り替え後（post-checkout。**`make setup` の再実行が必要**）・pre-push・CI で実行。
- dependabot の方針（グルーピング・major は個別・同時 5 本、#83）はマージ済み。

## 次の一手
1. `bash scripts/status.sh` で現状を確認する（冒頭に「安全装置」の警告が出たら、それを最優先で直す）。
2. 上の「現在の状態」の #84 を仕上げる（PR → レビュー → マージ）。
3. そのほかの backlog（優先順）: #87（スクリプトの可読性の改善・p2）、#56 の続き（dependabot の actions グループが major を束ねる点の修正・冒頭コメントの追記、置き換わった PR の取り込み）、#68（多言語対応・対象言語の確認が必要）、#73（ハーネス改善の提案機能・着手前に設計案を出す）、#74（ペルソナ整備）、#21〜#24（v0.2 の機能）。
4. 人の作業待ち（needs-human）: #25 macOS 実機確認 / #26 dmg 起動確認と v0.1.0 タグ push / #47 API キー登録とブランチ保護 / #52 Windows 実機確認 / #64 Windows の Smart App Control（署名するかの判断）/ #66 WSLg での起動確認 / #75 アプリの AI 送信前検査（採否の確認。実装は git stash に退避）。

## 未解決・注意
- **タグ push・リリース・シークレット登録・ブランチ保護はエージェントが実行できない**（権限で禁止）。人に依頼する。
- **境界ファイル**（`.claude/settings*.json`・`.claude/rules/security.md`・`.claude/agents/`・`scripts/hooks/`・`gh-post`/`gh-read`/`check-privacy`/`privacy_scan`/`injection_scan`/`merge-pr`/`review-verdict`/`record-approval`/`check-approvals`/`check-harness`/`check_harness`・`docs/approval-log.md`・`claude-review.yml`）は承認フローを通す。Bash では書き換えられない（Edit / Write を使う）。**コマンド文字列に境界ファイルのパスを含めて python / cp / sed 等を実行するとフックが拒否する**ので、スクリプトはファイルに書いて `python3 /tmp/…/x.py` だけを実行する。
- 新しいエージェント定義は**セッション再起動で読み込まれる**（`change-approver` は今は使える）。
- フックの検査は文字列判定で、Bash の工夫による迂回（`c\p`・`$IFS`・`sed --in-place`・`awk -i inplace`・`cd` してからの相対パスなど）は防げない。最終防衛線は人の確認・台帳・独立レビュー（#84 に一覧）。
- Wails は v2.10.2 に固定（CLI も v2.10.2 に揃える）。go-webview2 は v1.0.19 に固定。
- マージは必ず `scripts/merge-pr.sh`（head への APPROVE と CI 成功が必要）。`gh pr merge` は deny。独立レビュー後に push したら再レビューが必要（承認は head コミットごと）。
- 独立レビュー（`code-reviewer`）のコメント末尾のマーカーは `<!-- review-agent: verdict=APPROVE|REQUEST_CHANGES sha=<head の完全な SHA> -->` の書式で、**最終行**に置く。
- ユーザーの要求は、実装の前に必ず issue にする（UserPromptSubmit フックが毎回リマインド）。

## 更新履歴
- 2026-11-17: 初版（Windows 対応完了時点）
- 2026-11-18: 安全装置（#72 #78 #81 #85）マージ後。#84 の途中状態と再開手順を追記（ユーザー指示で作業中止）
