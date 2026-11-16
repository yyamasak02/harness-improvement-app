---
name: fix-bug
description: バグ修正の標準手順。先に失敗する回帰テストを書き、修正し、再発防止まで行う。
---
1. バグの issue（`type/bug`）を確認する。無ければ `create-issue` で作る（再現手順と期待結果を書く）。
2. `git switch -c fix/<issue番号>-<slug>`
3. **先に回帰テストを書き、失敗することを確認する**（`go test ./internal/<pkg> -run <Name>` / `npx vitest run <file>`）。
4. 最小の修正でテストを通す。関連コードに同種の問題がないか周辺を点検する。
5. 同種の再発が機械検出できるなら `scripts/check-rules.sh` + `scripts/selftest-checks.sh` に検査を足す。
6. `quality-gate` skill を実行する。
7. `bash scripts/ship.sh "fix: <要約>" "Closes #<番号> / 原因 / 回帰テスト / 検証結果"`
