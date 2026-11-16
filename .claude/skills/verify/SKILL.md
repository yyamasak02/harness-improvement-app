---
name: verify
description: コミット／PR 前の検証（go vet・go test・フロント build/test）を一括実行する。変更後・コミット前に必ず使う。
---
`bash scripts/verify.sh` を実行する（push・PR 前は `bash scripts/verify.sh --full`：lint・ルール検査・カバレッジ基準まで）。存在するレイヤー（Go / frontend）だけ検証する。
失敗したら原因を直して再実行。同じエラーを 3 回直せなければ `docs/blockers.md` に記録して次タスクへ。
成功するまでコミットしない。
