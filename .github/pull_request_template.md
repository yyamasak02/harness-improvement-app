<!-- タイトルは Conventional 形式: feat|fix|refactor|test|docs|chore|ci(scope): 要約 -->
<!-- ラベル規則: .claude/rules/issues-labels.md（type/ と area/ は自動付与されます） -->

## 概要

## 関連 issue
Closes #

## 品質チェック（該当しない項目は理由を書く）
- [ ] テストを追加・更新した（バグ修正は先に失敗する回帰テストを書いた）
- [ ] 不要になった／意味の薄くなったテストを整理した（`skip` や閾値の引き下げで通していない）
- [ ] 触った周辺のリファクタリングを検討した（`simplify` / `quality-gate` skill）。今回やらない分は issue（`type/tech-debt`）にした → #
- [ ] 仕様・README・docs・CHANGELOG の更新が必要か確認した
- [ ] `bash scripts/verify.sh --full` が通る
- [ ] 安全設計（`~/.claude` 読み取り専用・applier 経由・masking・`claude -p` の制約）に反していない

<!-- 挙動が変わらない純粋なリファクタリングでテストを変えない場合のみ、下の行を残して理由を書く（CI の pr-policy が読みます）:
[no-test-needed]: 理由
-->
