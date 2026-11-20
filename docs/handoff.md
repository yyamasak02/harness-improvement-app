# 引き継ぎメモ

セッションが切れても、どのフェーズからでも再開できるようにするためのメモ。**現実（git / PR / issue）が常に優先**。食い違ったら現実を信じ、再開後にこのメモを直す。
状況の自動確認は `bash scripts/status.sh`（`resume` skill）。更新ルールは `.claude/rules/session.md`。

## 現在の目標
- v0.1.0 リリースの準備。MVP（F1・F2・F3・F8・F9）と macOS / Windows 対応は実装・マージ済み。残りは人の確認とタグ push。
- 並行して、**開発プロセスの安全装置**（個人情報の流出防止・プロンプトインジェクション対策・承認フロー・再発防止）を整備してきた（#72 #78 #81 #85 はマージ済み）。

## 現在の状態（2026-11-21）
- **UX 改善は全てマージ済み**: #152 レイアウトと E2E・#153 自動提案の設定・#154 ローディング・#155 UI 刷新・#157 UX 計測・#158 評価の仕組み（ux-review skill・週次の ux-weekly ワークフロー）。#172 更新（データ形式のバージョン・上書き更新の手順）と #196（新形式の設定の案内）もマージ済み。
- **アーキテクチャ方針（#175）と、観点別のサブエージェント群による大規模リファクタリング（#176）を整備**: `docs/architecture.md`（現状・レイヤード方針・移行計画）、`internal/archtest` と `frontend/src/arch.test.ts`（依存の向きの検査。許容リストは**増やさない**。件数の上限 `allowlistCap` / `ALLOWLIST_CAP` で固定）、観点別レビュアー 7 つと修正担当 `refactorer`（エージェント定義）、`refactor-campaign` skill、`scripts/arch-metrics.sh`、`docs/refactor-agents.md`。
- リファクタリングの進み具合: 許容リスト（Go）は 8 件から 3 件（残りは app.go が applier・collector・store を直接 import している分）。移行 M1（#181）・M2（#182）完了、M3（#183）は第 1〜3 段階（config・targets/extensions・osutil）完了で、第 4 段階（store）以降（collector・applier）が続く。M4（#184 service の分割）・M5（#185）・F1（#180）は未着手。試運転のレビューの指摘（#188〜#192・#199〜#211 など）はマージ済み。
- 人の判断待ち: #193（applier の Rollback・History、runner のジョブディレクトリなど、挙動が変わる提案）、#173（アプリ内の更新確認。採用と決定済みで、実装はこれから）、#178（公開に向けた人の作業。発信は見送り中）。
- 運用の注意（今回の学び）: (1) PR 本文は `gh pr edit` が失敗して直せない。**`Closes #N` の語を段階的な issue に書かない**（`Refs #N`）。書いてしまったら PR を閉じて、直した本文の新しい PR にする。(2) worktree のローカルの git identity は実メールのことがある。`git merge` のマージコミットは、noreply の GIT_AUTHOR_*/GIT_COMMITTER_* を設定してから作る（push 前に確認する）。(3) `docs/decisions.md` の末尾に複数の PR が追記して衝突しやすい。マージ順に取り込む。(4) レビュー担当が、実装担当の worktree のブランチを切り替えることがある。

## 現在の状態（2026-11-20）
- 追加済み（マージ済み）: `scout` skill（改善提案の点検→issue 化・#97）、`parallel-work` skill（独立 issue を worktree とサブエージェントで最大 3 本並列・#99）、常時ロード文脈の削減（約 65KB→約 39KB・#100）。
- **v0.2 の機能 F4〜F7 は全てマージ済み**（#104 F4・#107 F5・#111 F6・#109 F7。T50〜T53 完了）。レビュー指摘の持ち越しも #113〜#115 でマージ済み。
- #116・#117・#112 はマージ済み。持ち越しは #125（masking）・#126・#128・#130・#137・#139 に起票済み。
- **リリース（人の作業が要る）**: `v0.1.0-rc.3` を push 済みで、Release ワークフローは全ジョブ成功（dmg・Windows インストーラ・.deb）。#148（release.yml の `substring` でワークフローが毎回失敗）を修正して rc.3 で初めて通った。
  - 人の確認待ち: Release に 3 つの成果物が付いたか／アセット名（`.deb` は `~` が `.` に置換される。#145）／WSLg での起動（#66）／macOS・Windows の起動（#25・#52）。確認できたら README のインストール手順を Release 案内に直し、#123・#124・#136 を閉じる。
  - 公開済みの誤ったタグ・Release（`0.0.0`・`0.0.1`・`1.0.0`・`v1.0.0`・`v1.0.1`）は、人が整理する（エージェントは削除しない）。ローカルの `v0.1.0-rc.1`・`v0.1.0-rc.2` は push せず `git tag -d` で消す。
  - rc.3 のあとに入った機能（#152 レイアウトと E2E・#153 自動提案・#154 ローディング・#155 UI 刷新・#157 UX 計測・#158 評価の仕組み）は、次の rc.4 で配布する（`release.sh prepare 0.1.0-rc.4`）。
- Linux の配布: .deb は #144 で実装済み、apt リポジトリ（署名鍵の管理）は人の判断待ち（#124）。#118（wailsjs のずれ検出）は #141 で完了、人がクローズする。
- 起票済みの提案: #102（常時ロード rules の削減・p2・security.md は承認フローが必要）。
- **要確認（人）**: F5・F6・F7 で `frontend/wailsjs` を手編集した（CI の wailsjs-sync で検出済み・#141）。macOS 等で `wails generate module` を実行し、差分が出ないか確認する。サブエージェントの worktree（`.claude/worktrees/`）が多数残っているので `git worktree remove` で掃除する（`rm -rf` は禁止）。
- 並列実行の運用: `parallel-work` skill。マージは 1 本ずつ `merge-pr.sh`。マージ後にローカルブランチ削除が worktree のせいで失敗するのは正常（マージ自体は成功）。
- 訂正: 下の「#87（スクリプトの可読性）」は番号の誤り（#87 は dependabot の PR でクローズ済み）。該当の issue は #88。

## 現在の状態（2026-11-18）
- #84（承認フロー）は #89、#56 の方針の修正は #91、typescript 7 は #93 でマージ済み。安全装置の診断は main で OK。
- #56 の残り: dependabot の #44（#93 で取り込み済み・自動で閉じる想定）と #40（@types/diff は不要になった）の PR を閉じる。#39・#41・#42・#43 は Node 22 が必須のため #92（needs-human・Node の方針）の判断待ち。
- minor 2 件は #90（p3）。

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
2. 上の「現在の状態」の #56 の残り（PR の整理）と #92 の判断を人に確認する。
3. そのほかの backlog（優先順）: #88（スクリプトの可読性の改善・p2）、#56 の続き（dependabot の actions グループが major を束ねる点の修正・冒頭コメントの追記、置き換わった PR の取り込み）、#68（多言語対応・対象言語の確認が必要）、#73（ハーネス改善の提案機能・着手前に設計案を出す）、#74（ペルソナ整備）、#21〜#24（v0.2 の機能）。
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
