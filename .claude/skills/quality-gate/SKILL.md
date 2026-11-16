---
name: quality-gate
description: PR 前の品質ゲート。テストの過不足・リファクタリング機会・ドキュメント同期を点検し、持ち越し課題を issue 化する。コードを変更したら PR 作成前に必ず実行する。
---
前提: `.claude/rules/quality.md` を読む。

1. 差分を把握する: `git diff origin/main...HEAD --stat` と `git diff origin/main...HEAD`
2. **テストの点検**
   - 変更した実装ごとに、対応するテスト（正常系・異常系・境界）があるか確認し、足りなければ書く。
   - 仕様変更で意味を失った／重複した／実装詳細に依存するテストを直す・消す（消す場合は理由を PR 本文に）。
   - `skip` / `.only` / 閾値の引き下げで通していないことを確認する。
3. **リファクタリングの点検**
   - `/simplify`（Skill）を実行して、重複・命名・未使用コード・過剰な抽象化を直す（触った周辺まで）。
   - 範囲外の大きな改善は `create-issue` で `type/tech-debt` の issue にして、PR 本文からリンクする。
4. **ドキュメント同期**: `docs/spec.md` / README / CHANGELOG（ユーザーに見える変更のみ）/ `docs/decisions.md`。
5. **機械化**: 今回見つけた新しい禁止事項・再発しうる不具合は `scripts/check-rules.sh` と `scripts/selftest-checks.sh` に足す。
6. `bash scripts/verify.sh --full` を実行し、全て通すまで PR を作らない。
7. 結果（直した点・issue 化した点）を PR 本文の品質チェックリストに反映する。
