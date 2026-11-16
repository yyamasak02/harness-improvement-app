---
description: 品質担保・リファクタリング・テスト保守のルール（コードを変更する全ての作業に適用）
---
# 品質ルール

目的: 「ハーネスを安全に改善する」ツールなので、**データを壊さない・秘密を漏らさない・勝手に書かない**を最優先に、変更のたびにテストと設計を健全に保つ。

## 機械的に強制されるもの（外すな・迂回するな）
| 仕組み | 内容 | 場所 |
|---|---|---|
| pre-commit | gofmt / go vet / staticcheck / tsc / ルール検査 / コミットメッセージ形式 / 末尾空白・秘密鍵 | `.pre-commit-config.yaml` |
| pre-push | `verify.sh --full`（lint・テスト・カバレッジ）と「実装を変えたらテストも変える」 | 同上 |
| CI | lint / test-go(-race) / test-frontend / test-macos / pr-policy / dmg-smoke | `.github/workflows/ci.yml` |
| カバレッジ | Go: collector・digest・masking・applier・runner ≥ 70%。フロント: stmts ≥ 80 / branch ≥ 75 / lines ≥ 80 | `scripts/coverage.sh` / `frontend/vite.config.ts` |
| ルール検査 | ホーム解決・実 `~/.claude` 参照・書き込み API の使用箇所・秘密らしき文字列・テスト無効化 | `scripts/check-rules.sh`（自己テスト: `scripts/selftest-checks.sh`） |
| 週次点検 | govulncheck / npm audit | `.github/workflows/maintenance.yml` |

## 変更するときの義務
1. **実装を変えたらテストも変える。** 新機能は正常系＋異常系、バグ修正は「先に失敗する回帰テスト → 修正」（`fix-bug` skill）。
2. **テストを弱めて通さない。** `skip` / `.only` / 期待値の緩和 / カバレッジ閾値の引き下げ / `--no-verify` で通すのは禁止。閾値を下げる正当な理由があるなら `docs/decisions.md` に理由を書き、人間の確認を得る（issue 化）。
3. **意味を失ったテストは直す・消す。** 仕様変更で不要になったテスト、重複テスト、実装詳細に張り付いた壊れやすいテストを放置しない。テストを消す PR では理由を本文に書く。
4. **リファクタリングの機会を毎回検討する。** PR 前に `quality-gate` skill（内部で `/simplify` を使用）を実行する。
   - 触ったコードの重複・長すぎる関数・不自然な命名・古いコメント・未使用コードはその場で直す（範囲は触った周辺まで）。
   - 範囲外の大きなリファクタは、その PR に混ぜず `type/tech-debt` の issue にする。PR 本文でリンクする。
   - 純粋なリファクタ（挙動不変）は `refactor:` プレフィックスで別 PR にする。テストを変えない場合のみ本文に `[no-test-needed]: 理由`。
5. **仕様・ドキュメントを同期する。** 挙動を変えたら `docs/spec.md`・README・CHANGELOG の該当箇所を更新する。`docs/decisions.md` に判断を残す。
6. **テストの書き方**: テーブル駆動。外部（claude・ホーム・時刻）は注入。`testdata/` のダミーのみ。時間依存は固定時刻を注入し、`sleep` に頼る場合は上限を短くして並列で壊れないこと。フレークを見つけたら直ちに issue（`type/test`）。
7. **新しい禁止事項を見つけたら機械化する。** 人が気をつける運用にせず `scripts/check-rules.sh` に検査を足し、`scripts/selftest-checks.sh` に検出テストを足す。

## PR 前チェックリスト（`quality-gate` skill が実行）
- [ ] `bash scripts/verify.sh --full`
- [ ] 実装差分に対応するテスト差分がある（なければ理由）
- [ ] 不要・重複・壊れやすいテストの整理をした
- [ ] `/simplify` 相当のレビューをした（重複・命名・未使用・過剰な抽象化）
- [ ] 持ち越す課題は issue 化した（ラベルは `issues-labels.md`）
- [ ] spec / README / CHANGELOG / decisions を更新した
