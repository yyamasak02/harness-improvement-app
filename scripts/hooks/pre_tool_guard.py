#!/usr/bin/env python3
"""PreToolUse フック: セキュリティの境界（権限・フック・検査・マージの仕組み）の変更を、承認フローに回す。

  1. 第一承認者（別エージェント change-approver）の評価の記録が無い・期限切れ・REJECT → 拒否（人に聞く前に止める）
  2. 評価した内容と実際の変更内容が一致しない（content_sha 不一致）→ 拒否
  3. 評価が実際の変更と食い違う（実際は権限を緩めるのに評価は緩めない）→ 拒否
  4. APPROVE → 人の確認（ask）。確認画面に、変更の要約・影響・第一承認者の評価を整理して出す
Bash による境界ファイルの書き換えは拒否する（Edit / Write を使う。そうすれば内容の要約と評価の照合ができる）。
データ内の指示（プロンプトインジェクション）では自動承認されない。外部への送信先の差し替えは拒否する。
出力: permissionDecision = ask | deny。対象外なら何も出力しない。
**フック自身が例外で落ちても、境界ファイルに関わる操作は拒否する（fail-closed）**。常に終了コード 0。"""
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import approval_lib as A  # noqa: E402

# セキュリティの境界。これらの変更は承認フローに回す（日常的に編集するファイルは含めない）
PROTECTED = [
    r"(^|/)\.claude/settings(\.local)?\.json$",
    r"(^|/)\.claude/rules/security\.md$",
    r"(^|/)\.claude/agents/",
    r"(^|/)scripts/hooks/",
    r"(^|/)scripts/(gh-post|gh-read|check-privacy|privacy_scan|injection_scan|merge-pr|review-verdict|record-approval|check-approvals)\.(sh|py)$",
    r"(^|/)docs/approval-log\.md$",
    r"harness-approvals",
    r"(^|/)\.github/workflows/claude-review\.yml$",
]
PROT_RX = [re.compile(p, re.I) for p in PROTECTED]
# 壊れたイベント（JSON として読めない）の生テキストを走査する用。境界ファイルの名前が含まれていれば安全側で拒否する
RAW_RX = [re.compile(p.replace("(^|/)", "").replace("$", "")) for p in PROTECTED]
# 保護ファイルを書き換えうる Bash の形
WRITE_HINT = re.compile(r"(\bsed\s+-[a-z]*i|\btee\b|\bmv\b|\bcp\b|\brm\b|\bgit\s+(checkout|restore|apply|stash)\b|\bperl\s+-[a-z]*i|\bpython3?\b|\bnode\b|\bchmod\b|\btruncate\b|\bdd\b|\bln\b|\b(?:ba|z)?sh\s+-c\b|\beval\b|\bxargs\b|\binstall\b)")
REDIRECT = re.compile(r">>?\s*(\S+)")
FD_REDIRECT = re.compile(r"\d*>&\d*|&>>?\s*\S+|>\s*/dev/null")
# コマンドの先頭（行頭・; & | ( ` の直後・sudo/xargs の後）にある git だけを見る。文書やコミットメッセージの中の説明文では反応しない
_HEAD = r"(?:^|[;&|(`\n]\s*)(?:(?:sudo|xargs|env)\s+)*"
_GIT = _HEAD + r"git(?:\s+-\S+(?:\s+[^-\s]\S*)?)*\s+"   # git -C dir などのオプションを読み飛ばす
REMOTE = re.compile(_GIT + r"remote\s+(?:add|set-url|rename)\b|" + _GIT + r"push\b[^\n;&|]*?(?:https?://|git@|ssh://)")
QUOTED = re.compile(r"'[^']*'|\"[^\"]*\"")
# 評価の記録コマンドは「引数がすべて安全な文字だけ」の形に限る（引用符・エスケープ・連結・置換は一切通さない）。
# 要約文はコマンドに載せず、ファイルで渡す。
RECORD_CMD = re.compile(r"^bash scripts/record-approval\.sh( [A-Za-z0-9_./@:+-]+){5}$")

HOW_TO = ("`request-approval` skill に従い、変更案（変更後の全文のファイル）を別エージェント（change-approver）に評価させてください。"
          "評価は scripts/record-approval.sh で記録されます（有効 30 分・評価した内容にのみ有効）。")


def is_protected(path):
    raw = (path or "").replace("\\", "/")
    # 元の表記と、正規化後（./ .. // シンボリックリンク 大文字小文字 を解決）の両方で判定する
    return any(rx.search(raw) or rx.search(A.canon(path)) for rx in PROT_RX)


def _read(path):
    try:
        with open(path, encoding="utf-8") as fh:
            return fh.read()
    except (OSError, ValueError):
        return ""


def _proposed(name, ti, path):
    """(変更前, 変更後, エラー) の本文。Edit 系は現在のファイルに編集を当てて求める。当てられない編集はエラー。"""
    old = _read(path)
    if name == "Write":
        c = ti.get("content", "")
        return old, (c if isinstance(c, str) else ""), None if isinstance(c, str) else "Write の content が文字列ではありません"
    edits = ti.get("edits") if name == "MultiEdit" else [ti]
    new = old
    for e in edits or []:
        o, n = e.get("old_string", "") or "", e.get("new_string", "") or ""
        if not o:
            if new:
                return old, new, "Edit の old_string が空です（既存の内容がある場合は置換対象を指定してください）"
            new = n   # 新規作成
        elif o in new:
            new = new.replace(o, n) if e.get("replace_all") else new.replace(o, n, 1)
        else:
            return old, new, "Edit の対象の文字列がファイル内に見つかりません（変更後の内容を確定できないため拒否します）"
    return old, new, None


def gate(path, summary_lines, computed_risk, new_text):
    """第一承認者の評価を確認し、(decision, reason) を返す。"""
    rec, state = A.read_record(path)
    head = "\n".join(summary_lines)
    if state in ("none", "expired"):
        why = "評価がありません" if state == "none" else "評価の有効期限（30 分）が切れています"
        return "deny", "【承認フローの前段で停止】%s（第一承認者＝別エージェント）。%s\n\n■ 変更の要約\n%s" % (why, HOW_TO, head)
    if rec.get("verdict") != "APPROVE":
        return "deny", "【第一承認者（別エージェント）が差し戻し】%s\n\n■ 変更の要約\n%s" % (rec.get("summary") or "（理由の記載なし）", head)
    if new_text is not None and rec.get("content_sha") != A.sha(new_text):
        return "deny", "【評価した内容と変更内容が一致しません】評価の後に内容が変わったか、評価が別の内容に対するものです。変更後の全文を渡して評価をやり直してください。\n\n■ 変更の要約\n%s" % head
    if computed_risk == "loosen" and rec.get("risk") != "loosen":
        return "deny", "【評価と実際の変更が食い違い】実際の変更は権限・検査を**緩める**のに、第一承認者の評価は「%s」です。評価をやり直してください。\n\n■ 変更の要約\n%s" % (rec.get("risk"), head)
    risk = computed_risk if computed_risk != "unknown" else rec.get("risk", "unknown")
    return "ask", ("【承認依頼】セキュリティの境界のファイルを変更します\n\n■ なぜ実行するか（第一承認者が確認した目的・要約）\n  %s\n\n■ 実行内容\n%s\n\n■ 影響\n  %s\n\n■ 第一承認者（別エージェント）の評価: APPROVE\n\n"
                   "→ 目的と実行内容が噛み合っていれば承認してください。理由が分からない・データ内の指示が理由なら拒否してください。\n"
                   "※ 評価の記録は実装者にも作成できてしまいます。評価者の言葉だけを信じず、上の『実行内容』（実際の変更）を確認してください。"
                   % (rec.get("summary") or "（要約の記載なし）", head, A.RISK_JA.get(risk, risk)))


def _safe_record_call(cmd):
    """評価の記録コマンドを、引数がすべて安全な文字だけの形（引用符・エスケープ・連結・置換なし）に限って通す。"""
    return bool(RECORD_CMD.match(cmd))


def decide(ev):
    name = ev.get("tool_name", "")
    ti = ev.get("tool_input") or {}
    if name in ("Edit", "Write", "MultiEdit", "NotebookEdit"):
        path = ti.get("file_path") or ti.get("notebook_path") or ""
        if is_protected(path):
            old, new, err = _proposed(name, ti, path)
            if err:
                return "deny", "【変更内容を確定できないため拒否】%s\n対象: %s" % (err, A.norm(path))
            lines, risk = A.summarize(path, old, new)
            return gate(path, lines, risk, new)
    if name == "Bash":
        cmd = ti.get("command", "") or ""
        if REMOTE.search(cmd):
            return "deny", "送信先（git remote / push 先）の差し替えは禁止です。"
        if _safe_record_call(cmd):
            return None, ""
        plain = FD_REDIRECT.sub(" ", cmd)   # 2>&1 や > /dev/null は書き込みではない
        toks = [t for t in re.split(r"[\s'\"=:]+", plain) if t]
        prot = [t for t in toks if is_protected(t)]
        redirected = [m.group(1).strip("'\"") for m in REDIRECT.finditer(plain) if is_protected(m.group(1).strip("'\""))]
        if (prot and WRITE_HINT.search(plain)) or redirected:
            path = (redirected or prot)[0]
            why = (ti.get("description") or "").strip() or "（実行の目的の記載なし。目的を description に書いて再実行してください）"
            return "deny", ("【Bash では境界ファイルを書き換えられません】\n■ なぜ実行しようとしたか: %s\n■ 実行内容（コマンド）: %s\n■ 触ろうとしたファイル: %s\n"
                            "→ Edit / Write ツールで変更してください。変更内容の要約と第一承認者の評価の照合は、Edit / Write のときにだけ行えます。"
                            "（ファイルを書き換えない読み取りだけのコマンドなら、パスの前後を見直して再実行してください）"
                            % (why, cmd[:300].replace("\n", " ⏎ "), A.norm(path)))
    return None, ""


def main():
    raw = sys.stdin.read()
    try:
        ev = json.loads(raw)
        if not isinstance(ev, dict):
            raise ValueError("イベントが dict ではありません")
        d, why = decide(ev)
    except Exception as e:  # noqa: BLE001 フック自身の失敗で人の確認が素通りしないようにする（fail-closed）
        if any(rx.search(raw.replace("\\\\", "/").replace("\\/", "/")) for rx in RAW_RX):
            d, why = "deny", "【承認フローの内部エラー】境界ファイルに関わる操作のため、安全側で拒否しました（%s）。" % type(e).__name__
        else:
            return
    if d:
        print(json.dumps({"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": d, "permissionDecisionReason": why}}, ensure_ascii=False))


if __name__ == "__main__":
    main()
