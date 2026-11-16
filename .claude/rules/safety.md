---
description: 書き込み・AI 実行・マスキングに関する安全設計（常に適用）
---
# 安全設計ルール（spec §8）
- 書き込み許可パスは `internal/targets` が発見したもの＋`~/.claude/skills/` 配下のみ（許可リスト）。
- applier: SHA-256 照合 → `<data>/backups/<ts>-<hash>/`（元ファイル＋manifest.json）→ temp → fsync → rename。ロールバックもバックアップを取る。
- masking 対象: sk-ant-、AKIA…、ghp_/github_pat_、xox[baprs]-、Bearer、PEM 秘密鍵、メール、(password|secret|token|api_key)=値。置換は `[MASKED:<種別>]`。
- runner: `claude -p ... --allowedTools Read,Write --output-format json`、cwd=`<work>/<job-id>`、タイムアウト既定 5 分、stderr はジョブログへ。
- パス・ホーム・claude バイナリは全て注入可能にし、テストは `t.TempDir()` と `testdata/` のみ。
