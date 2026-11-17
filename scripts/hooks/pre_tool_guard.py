#!/usr/bin/env python3
"""PreToolUse フック: セキュリティの境界（権限・フック・検査・マージの仕組み）の変更を、必ず人の確認に回す。
データ内の指示（プロンプトインジェクション）では自動承認されない。外部への送信先の差し替えは拒否する。
出力: permissionDecision = ask | deny。対象外なら何も出力しない。常に終了コード 0。"""
import json
import re
import sys

# セキュリティの境界。これらの変更は人が確認する（日常的に編集するファイルは含めない）
PROTECTED = [
    r"(^|/)\.claude/settings(\.local)?\.json$",
    r"(^|/)\.claude/rules/security\.md$",
    r"(^|/)\.claude/agents/",
    r"(^|/)scripts/hooks/",
    r"(^|/)scripts/(gh-post|gh-read|check-privacy|privacy_scan|injection_scan|merge-pr|review-verdict)\.(sh|py)$",
    r"(^|/)\.github/workflows/claude-review\.yml$",
]
PROT_RX = [re.compile(p) for p in PROTECTED]
# 保護ファイルを書き換えうる Bash の形
WRITE_HINT = re.compile(r"(\bsed\s+-[a-z]*i|\btee\b|\bmv\b|\bcp\b|\brm\b|\bgit\s+(checkout|restore|apply)\b|\bperl\s+-[a-z]*i|\bpython3?\b|\bnode\b|\bchmod\b|\btruncate\b|\bdd\b|\bln\b)")
REDIRECT = re.compile(r">>?\s*(\S+)")
FD_REDIRECT = re.compile(r"\d*>&\d*|&>>?\s*\S+|>\s*/dev/null")
# コマンドの先頭（行頭・; & | ( ` の直後・sudo/xargs の後）にある git だけを見る。文書やコミットメッセージの中の説明文では反応しない
_HEAD = r"(?:^|[;&|(`\n]\s*)(?:(?:sudo|xargs|env)\s+)*"
_GIT = _HEAD + r"git(?:\s+-\S+(?:\s+[^-\s]\S*)?)*\s+"   # git -C dir などのオプションを読み飛ばす
REMOTE = re.compile(_GIT + r"remote\s+(?:add|set-url|rename)\b|" + _GIT + r"push\b[^\n;&|]*?(?:https?://|git@|ssh://)")


def is_protected(path):
    p = (path or "").replace("\\", "/")
    return any(rx.search(p) for rx in PROT_RX)


def decide(ev):
    name = ev.get("tool_name", "")
    ti = ev.get("tool_input") or {}
    if name in ("Edit", "Write", "MultiEdit", "NotebookEdit"):
        path = ti.get("file_path") or ti.get("notebook_path") or ""
        if is_protected(path):
            return "ask", "セキュリティの境界のファイル（%s）の変更です。権限・フック・検査を緩めていないか、人が確認してください。" % path
    if name == "Bash":
        cmd = ti.get("command", "")
        if REMOTE.search(cmd):
            return "deny", "送信先（git remote / push 先）の差し替えは禁止です。"
        plain = FD_REDIRECT.sub(" ", cmd)   # 2>&1 や > /dev/null は書き込みではない
        toks = [t for t in re.split(r"[\s'\"=:]+", plain) if t]
        touches = any(rx.search(t) for t in toks for rx in PROT_RX)
        redirects_to_protected = any(is_protected(m.group(1).strip("'\"")) for m in REDIRECT.finditer(plain))
        if (touches and WRITE_HINT.search(plain)) or redirects_to_protected:
            return "ask", "セキュリティの境界のファイルを書き換えうるコマンドです。人が確認してください。"
    return None, ""


def main():
    try:
        ev = json.load(sys.stdin)
    except ValueError:
        return
    d, why = decide(ev)
    if d:
        print(json.dumps({"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": d, "permissionDecisionReason": why}}, ensure_ascii=False))


if __name__ == "__main__":
    main()
