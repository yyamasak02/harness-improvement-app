---
name: refactor-campaign
description: 観点別のサブエージェント群でコードをレビューし、指摘を issue にして、修正→独立レビュー→マージまでを小さな PR で自動で回す（大規模リファクタリング）。アーキテクチャ方針（docs/architecture.md）に沿った整理を進めるとき、定期的な構造の点検に使う。
---
メインのセッションは**調整だけ**を担う。観点別のレビューも修正も、新しい文脈のサブエージェントに任せる（文脈を使い切らないため）。共通ルール・出力の定型・停止条件は `docs/refactor-agents.md`、方針は `docs/architecture.md`。

## 前提の確認（満たさなければ始めない）
- `main` がクリーンで CI が成功している。`bash scripts/status.sh` で安全装置の警告が無い。
- `bash scripts/arch-metrics.sh` を実行し、開始時の値（許容リストの件数・巨大ファイル・`time.Now` の直接呼び出し）を記録する。
- 並行して進めている feature の issue があれば、同じファイルを触らない範囲に限る（衝突する指摘は後回し）。

## 1 周回の手順
1. **観点別レビューを並列に起動する**（読み取り専用。Agent で `refactor-review-layering` / `-testability` / `-reuse` / `-structure` / `-robustness` / `-frontend` / `-tests`、`run_in_background`）。依頼文には対象範囲（パッケージ・ディレクトリ）と「`docs/refactor-agents.md` の定型で返す」だけを書く。実装の経緯・意見は渡さない。
2. **集約する**: 指摘を集め、重複を統合し、次に分類する。
   - 自動修正の対象: 挙動が変わらない・見積り S/M・境界ファイル不要。
   - 人の判断に回す: 「挙動が変わる: yes」・仕様との矛盾・境界ファイルが要る・見積り L（分割して再起票）。
3. **起票する**（`create-issue`。`type/tech-debt`、領域は `area/*`、優先度は重大度に対応: high→p2、medium→p3、low→p3。本文に指摘の定型・対象・根拠・方針の節・完了条件）。重複は `bash scripts/related-issues.sh` で確認して起票しない。人の判断に回すものには `status/needs-human` を付ける。
4. **優先度順に修正する**（`parallel-work` skill に従う。最大 3 本並列。Agent で `refactorer`、`isolation: worktree`、`run_in_background`。依頼文は issue 番号と「`.claude/agents/refactorer.md` に従う」だけ）。1 周回の PR は最大 6 本。同じファイル・同じパッケージを触る指摘は並列にしない。
5. **独立レビュー**: PR ごとに、修正した `refactorer` とは別の `code-reviewer` を新しく起動する（`review-pr` skill）。`REQUEST_CHANGES` なら `refactorer` に直させて再レビュー。
6. **マージ**: `bash scripts/merge-pr.sh <N>` を 1 本ずつ（承認マーカーと CI 成功を検査する。`gh pr merge` は使わない）。マージするたびに、次の PR は `origin/main` を取り込んでから ship する。
7. **再計測**: `bash scripts/arch-metrics.sh`・`bash scripts/ux-report.sh`（UX 指標があるとき）・カバレッジを確認し、開始時と比べる。許容リストが減ったら、上限（`allowlistCap` / `ALLOWLIST_CAP`）が下がっていることを確認する。
8. `docs/handoff.md` を更新する（周回の結果・残りの指摘・次の一手）。

## 停止条件・安全弁（守る）
- 同じ PR で CI 失敗か `REQUEST_CHANGES` が 3 回続いたら、`docs/blockers.md` に記録してその指摘は止める（他は続ける）。
- 再計測で、許容リストの増加・巨大ファイルの増加・カバレッジの低下・UX 指標の予算超過のいずれかが出たら、周回を止めて原因の PR を特定する（直せなければ issue にして人に報告）。
- 挙動を変える変更・仕様の変更・境界ファイルの変更を、リファクタリングの名目で通さない。レビューで見つけたら blocker にする。
- テストの削除・skip・期待値の緩和・カバレッジ閾値の引き下げ・許容リストの増加は、理由があっても自動では通さない（人の判断。issue 化）。
- ユーザーの別の依頼が来たら、周回を中断してそちらを優先する（途中の PR は完了させるか、ブランチを残して報告する）。

## 完了の報告
- 周回ごとに、起票した issue・マージした PR・開始時と終了時の計測値・人の判断に回した指摘を、ユーザーに短く報告する。
