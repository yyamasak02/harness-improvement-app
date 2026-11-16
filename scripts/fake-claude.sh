#!/usr/bin/env bash
# テスト用のフェイク claude。cwd（ジョブディレクトリ）の input.json を読み、output.md / rationale.md を書く。
# FAKE_CLAUDE_MODE: ok(既定) | fail(終了コード1) | timeout(長時間sleep) | nooutput(出力ファイルを作らない)
# FAKE_CLAUDE_ARGS_FILE: 指定すると引数を1行1引数で追記する
# FAKE_CLAUDE_SLEEP: timeout モードの秒数（既定 30）
set -u
if [ -n "${FAKE_CLAUDE_ARGS_FILE:-}" ]; then
  { printf 'cwd=%s\n' "$PWD"; for a in "$@"; do printf 'arg=%s\n' "$a"; done; } >> "$FAKE_CLAUDE_ARGS_FILE"
fi
case "${FAKE_CLAUDE_MODE:-ok}" in
  fail)
    echo "fake-claude: forced failure" >&2
    exit 1 ;;
  timeout)
    sleep "${FAKE_CLAUDE_SLEEP:-30}"
    exit 0 ;;
  nooutput)
    echo '{"type":"result","result":"no output"}'
    exit 0 ;;
esac
if [ ! -f input.json ]; then
  echo "fake-claude: input.json not found in $PWD" >&2
  exit 2
fi
cat > output.md <<'MD'
# 改善後の CLAUDE.md（フェイク）

- 応答は日本語
- テストは go test
- フェイクが追加した行
MD
cat > rationale.md <<'MD'
- フェイクによる固定の変更理由
MD
echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
