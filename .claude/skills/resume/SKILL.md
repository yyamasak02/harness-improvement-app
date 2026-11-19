---
name: resume
description: セッションの開始・再開時に、どのフェーズからでも現状を把握して次の行動を決める。作業の途中から再開するとき、状況が分からないとき、最初に必ず使う。
---
1. **状況を取得する**: `bash scripts/status.sh`（詳細）。SessionStart フックで要約は既に表示されている。
2. **メモと照合する**: `docs/handoff.md` を読み、status.sh が示す現実（git・PR・issue）と食い違う点を確認する。食い違いは現実を信じる。
3. **フェーズを判定して再開する**（status.sh の「次にすべきこと」と一致するはず。違うなら理由を調べる）:

| 状態 | 次の行動 |
|---|---|
| 未コミットの変更がある | `git diff` で内容を確認。意図が分かれば続きを実装、分からなければ `git log`・handoff・関連 issue から文脈を復元。完成していれば `quality-gate` → `scripts/ship.sh` |
| 未 push のコミットがある | `scripts/ship.sh`（verify --full → push） |
| ブランチはあるが PR が無い | `quality-gate` → 関連 issue 確認 → `scripts/ship.sh` で PR 作成 |
| GitHub の状態を取得できない（判定不能） | PR の作成・マージ判断をしない。git の状態だけで判断し、一時的な障害なら再実行、認証なら `gh auth status` をユーザーに確認してもらう。`gh pr list` で既存 PR を手動確認して重複作成を避ける |
| 現在のブランチの PR がマージ済み・クローズ済み | マージ済み: `git switch main && git pull`、不要なブランチを削除、handoff を更新。クローズ済み: 理由をコメントで確認 |
| PR の CI が失敗 | 完了後に `gh run view <run-id> --log-failed` で原因を特定し、直して push（`fix-bug` の要領で回帰テストも） |
| PR の CI が実行中 | 完了を待つ（Monitor で run-id を指定して待つ。`--limit 1` で最新を拾うと古い実行を拾うことがあるので ID を固定する）。**待つだけにしない**: 独立した issue があれば `parallel-work` で並行して進める |
| head への独立レビューが無い | `review-pr` skill（`code-reviewer` に PR 番号だけ渡す） |
| REQUEST_CHANGES | 最新コメントの blocker/major を全て直す → push → CI 確定後に再レビュー |
| APPROVE かつ CI 成功 | `bash scripts/merge-pr.sh <N>` |
| main がクリーンで PR も無い | `implement-task`: 着手できる issue（needs-human 以外の p0/p1）を `related-issues.sh` で確認して着手 |
| 着手できる issue が無い | 人の作業待ち。needs-human の一覧をユーザーに報告し、`release` skill でリリース準備を確認 |

4. **進める**。確認のために止まらない（CLAUDE.md の自律実行ルール）。止まってよい条件に当たるときだけ止まる。
5. **メモを更新する**: マージ・フェーズ変更・待ち状態に入る前に `docs/handoff.md` を更新する（`.claude/rules/session.md`）。
