# 引き継ぎメモ

セッションが切れても、どのフェーズからでも再開できるようにするためのメモ。**現実（git / PR / issue）が常に優先**。食い違ったら現実を信じ、再開後にこのメモを直す。
状況の自動確認は `bash scripts/status.sh`（`resume` skill）。更新ルールは `.claude/rules/session.md`。

## 現在の目標
- v0.1.0 リリースの準備。MVP（F1・F2・F3・F8・F9）と macOS / Windows 対応は実装・マージ済み。残りは人の確認とタグ push。
- 並行して、**開発プロセスの安全装置**（個人情報の流出防止・プロンプトインジェクション対策・承認フロー・再発防止）を整備してきた（#72 #78 #81 #85 はマージ済み）。

## 中断時点の状態（2026-11-18・ユーザーの指示で作業を中止）
**作業ブランチ `chore/approval-hardening-84`（#84「承認フローの誤検出・fail-open・台帳の照合」）が途中。PR は未作成。**
- 反映済み（作業ツリー・コミット前〜ローカルコミット。`git status` で確認）: `scripts/hooks/pre_tool_guard.py`（誤検出の削減・引用符の扱い・実行専用の形）、`scripts/record-approval.sh`（台帳に内容のハッシュ `sha:…`）、`scripts/check-approvals.sh`（差分が取れなければ NG・台帳のハッシュ照合）、`scripts/check_harness.py`（CI/pre-commit の配線の検査・必須 deny の追加）、`scripts/selftest-checks.sh`（#84 のテスト追加）、`docs/approval-log.md`（評価の台帳）。これらは別エージェント（change-approver）の評価（最終 APPROVE）を経て反映した。
- **未反映（再開時にやること）**: `.claude/settings.json` の deny に、main の移動・削除の禁止を追加する。追加する 8 項目: `Bash(git branch -f*)` `Bash(git branch -m*)` `Bash(git branch -D main*)` `Bash(git branch -d main*)` `Bash(git update-ref*)` `Bash(git switch -C*)` `Bash(git switch --force-create*)` `Bash(git checkout -B*)`（deny の末尾に追記。allow の追加は無い）。
  - これが未反映の間は、`check_harness.py` の必須 deny（`git branch -f*`・`git update-ref*`）が足りず、**`bash scripts/check-harness.sh` は NG になる**（ブランチ上の一時的な状態。main では正常）。
  - 手順は `request-approval` skill: 変更後の全文を境界ファイル以外にファイルとして作る → `change-approver` に評価・記録させる（有効 30 分・評価した内容にのみ有効）→ Write ツールで同じ内容を反映（人の確認画面が出る）→ `bash scripts/selftest-checks.sh` と `bash scripts/check-harness.sh` を通す → `bash scripts/ship.sh` で PR → 独立レビュー（`review-pr`）→ `bash scripts/merge-pr.sh`。
  - 評価の記録は 30 分で切れる。切れていたら再評価が要る。
- スクラッチ領域（`/tmp/claude-…/scratchpad/wip84/`）の作業ファイルは消えている前提で進める（必要なら上の説明から作り直す）。

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
2. 上の「中断時点の状態」の #84 を仕上げる（settings.json の deny 追加 → 承認フロー → PR → レビュー → マージ）。
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
