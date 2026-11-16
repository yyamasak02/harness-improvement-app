# testdata
全て匿名化したダミー。実 `~/.claude` の内容は含まない。
- `claude/` … ダミーの `~/.claude`（projects/-home-demo-app に 2 セッション、壊れた行・未知 type・isMeta・訂正・ツールエラー・複数モデル・偽の秘密情報を含む）
- `workspace/app/` … cwd 相当（CLAUDE.md / CLAUDE.local.md / .claude/rules）
jsonl の cwd は `/home/demo/app`（実在しない）。targets のテストは `workspace/app` を明示的に渡す。
