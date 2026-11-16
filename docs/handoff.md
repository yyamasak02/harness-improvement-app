# 引き継ぎメモ

セッションが切れても、どのフェーズからでも再開できるようにするためのメモ。**現実（git / PR / issue）が常に優先**。食い違ったら現実を信じ、再開後にこのメモを直す。
状況の自動確認は `bash scripts/status.sh`（`resume` skill）。更新ルールは `.claude/rules/session.md`。

## 現在の目標
- v0.1.0 リリースの準備。MVP（F1・F2・F3・F8・F9）と macOS / Windows 対応は実装・マージ済み。残りは人の確認とタグ push。

## 進捗（要約）
- 実装: docs/tasks.md の T00〜T44 完了。Phase 5（F4〜F7）は issue（#21〜#24）で管理。
- 品質基盤: pre-commit・CI（lint / test-go -race / test-frontend / test-macos / test-windows / pr-policy / dmg・windows ビルド）・カバレッジ基準・ルール検査・独立レビュー（`review-pr` skill、`scripts/merge-pr.sh`）。
- Windows 対応（#46）完了。配布は release.yml（タグ push 時に macOS の dmg と Windows の zip・インストーラ）。

## 次の一手
1. `bash scripts/status.sh` で現状を確認する。
2. リリースタグの整備は完了（CHANGELOG・`scripts/check-version.sh`・`scripts/release.sh`・`release` skill）。実際のリリースは人の検証（#25 #52）と #47 の後に `release` skill で進める。
3. 人の作業待ち（needs-human）: #25 macOS 実機確認 / #26 dmg 起動確認と v0.1.0 タグ push / #47 API キー登録とブランチ保護 / #52 Windows 実機確認 / #27 個人メールの扱い。

## 未解決・注意
- **タグ push・リリース・シークレット登録・ブランチ保護はエージェントが実行できない**（権限で禁止）。人に依頼する。
- dependabot の PR が未整理で溜まっている（メジャー更新を含む）。マージ前に CI とレビュー（`review-pr`）を通す。方針は issue で管理。
- Wails は v2.10.2 に固定（CLI も v2.10.2 に揃える）。go-webview2 は v1.0.19 に固定。
- マージは必ず `scripts/merge-pr.sh`（head への APPROVE と CI 成功が必要）。`gh pr merge` は deny。
- 独立レビュー後に push したら再レビューが必要（承認は head コミットごと）。

## 更新履歴
- 2026-11-17: 初版（Windows 対応完了時点）
