"""承認フローの共通処理: 変更サマリの作成（決定論的）と、第一承認者（別エージェント）の評価記録の読み書き。

評価記録は `<git-dir>/harness-approvals/<パスのハッシュ>.json`（コミットされない）。有効期限つきで、
**評価した変更内容のハッシュ（content_sha）に結び付く**（評価後に内容が変わったら無効）。
信頼モデル: 記録は「第一承認者の評価」という目印であり、最終承認者は人。実装者が自分で記録を作ることは
規則で禁止している（レビューのマーカーと同じ扱い）。壊れた記録・読めない記録は「無い」として扱う（fail-closed）。"""
import difflib
import hashlib
import json
import math
import os
import re
import subprocess
import time

TTL_SECONDS = 30 * 60  # 評価の有効期間（30 分）


def git_dir():
    d = os.environ.get("HARNESS_APPROVAL_DIR")
    if d:
        return d
    try:
        g = subprocess.run(["git", "rev-parse", "--git-dir"], capture_output=True, text=True, cwd=os.environ.get("CLAUDE_PROJECT_DIR") or None).stdout.strip()
    except OSError:
        g = ""
    base = g if os.path.isabs(g) else os.path.join(os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd(), g or ".git")
    return os.path.join(base, "harness-approvals")


def sha(text):
    return hashlib.sha256((text or "").encode("utf-8", "replace")).hexdigest()


def canon(path):
    """パスの正規化（./ .. // 末尾の空白と引用符、シンボリックリンク、大文字小文字）。保護の判定と評価記録のキーに使う。"""
    p = (path or "").strip().strip("'\"").replace("\\", "/")
    p = re.sub(r"/+", "/", p)
    try:
        if os.path.isabs(p):
            p = os.path.realpath(p)   # シンボリックリンクを解決（存在しないパスは親までを解決）
        else:
            p = os.path.normpath(p)
    except (OSError, ValueError):
        p = os.path.normpath(p)
    return p.replace("\\", "/").casefold()


def key(path):
    return hashlib.sha256(norm(path).casefold().encode()).hexdigest()[:16]


def norm(path):
    """リポジトリ内の相対パス（区切りは /）。絶対パスは CLAUDE_PROJECT_DIR を基準に相対化する。"""
    p = canon(path) if os.path.isabs((path or "").strip()) else (path or "").strip().replace("\\", "/")
    p = re.sub(r"/+", "/", p)
    root = canon(os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd()).rstrip("/") + "/"
    if p.startswith(root):
        p = p[len(root):]
    # テストなどで別のルートが使われる場合に備え、境界ディレクトリ以降だけを使う
    for marker in (".claude/", "scripts/", ".github/", "docs/approval-log.md", ".git/"):
        i = p.find(marker)
        if i > 0:
            p = p[i:]
            break
    return p


def read_record(path, now=None):
    """(記録 or None, 状態) 状態: none | expired | ok。型が不正・読めない記録は none（fail-closed）。"""
    f = os.path.join(git_dir(), key(path) + ".json")
    try:
        with open(f, encoding="utf-8") as fh:
            rec = json.load(fh)
    except (OSError, ValueError):
        return None, "none"
    c = rec.get("created") if isinstance(rec, dict) else None
    if not isinstance(c, (int, float)) or isinstance(c, bool) or not math.isfinite(c) or c > (now or time.time()) + 60:
        return None, "none"   # 型が不正・非有限・未来の時刻は無効
    if rec.get("verdict") not in ("APPROVE", "REJECT") or not isinstance(rec.get("summary", ""), str):
        return None, "none"
    if (now or time.time()) - rec["created"] > TTL_SECONDS:
        return rec, "expired"
    return rec, "ok"


def write_record(path, verdict, risk, summary, content_sha="", now=None):
    d = git_dir()
    os.makedirs(d, exist_ok=True)
    rec = {"path": norm(path), "verdict": verdict, "risk": risk, "summary": summary, "content_sha": content_sha,
           "created": int(now or time.time())}
    f = os.path.join(d, key(path) + ".json")
    tmp = f + ".tmp"
    with open(tmp, "w", encoding="utf-8") as fh:
        json.dump(rec, fh, ensure_ascii=False)
    os.replace(tmp, f)
    return rec


def line_stats(old, new):
    added = removed = 0
    for l in difflib.unified_diff(old.splitlines(), new.splitlines(), lineterm="", n=0):
        if l.startswith("+") and not l.startswith("+++"):
            added += 1
        elif l.startswith("-") and not l.startswith("---"):
            removed += 1
    return added, removed


def _hook_set(cfg):
    """hooks 設定を (イベント, matcher, コマンド) の集合にする。"""
    out = set()
    for ev, groups in ((cfg or {}).get("hooks") or {}).items():
        for g in groups or []:
            for h in (g or {}).get("hooks") or []:
                out.add((ev, (g or {}).get("matcher", ""), (h or {}).get("command", "")))
    return out


def settings_delta(old, new):
    """settings.json の変更を調べる。(説明の行リスト, 判定) 判定: loosen | tighten | neutral | unknown
    権限（allow/deny/ask）・フック・その他すべての設定キーを見る。権限を緩めうるものは loosen（保守的）。"""
    try:
        o, n = json.loads(old or "{}"), json.loads(new)
    except ValueError:
        return ["（JSON として読めないため、権限の増減は判定できません）"], "unknown"
    if not isinstance(o, dict) or not isinstance(n, dict):
        return ["（設定の形式が想定と違うため判定できません）"], "unknown"
    lines, loosen, tighten = [], False, False

    def perm(d, k):
        p = d.get("permissions")
        return list(p.get(k) or []) if isinstance(p, dict) else []

    for kind in ("deny", "allow", "ask"):
        before, after = set(perm(o, kind)), set(perm(n, kind))
        rem, add = sorted(before - after), sorted(after - before)
        if rem:
            lines.append("%s を %d 件削除: %s" % (kind, len(rem), " / ".join(rem[:6]) + (" …" if len(rem) > 6 else "")))
        if add:
            lines.append("%s を %d 件追加: %s" % (kind, len(add), " / ".join(add[:6]) + (" …" if len(add) > 6 else "")))
        if kind == "deny":
            loosen |= bool(rem)
            tighten |= bool(add)
        elif kind == "allow":
            loosen |= bool(add)
            tighten |= bool(rem)
        else:  # ask: 増えれば確認が増える（締める）、減れば緩める
            loosen |= bool(rem)
            tighten |= bool(add)
    # permissions の allow/deny/ask 以外（defaultMode・additionalDirectories など）
    po = {k: v for k, v in (o.get("permissions") or {}).items() if k not in ("allow", "deny", "ask")} if isinstance(o.get("permissions"), dict) else {}
    pn = {k: v for k, v in (n.get("permissions") or {}).items() if k not in ("allow", "deny", "ask")} if isinstance(n.get("permissions"), dict) else {}
    for k in sorted(set(po) | set(pn)):
        if po.get(k) != pn.get(k):
            lines.append("permissions.%s が変わります（権限の動作に関わる設定）" % k)
            loosen = True
    # フック: 既存のフックが消えた・コマンドが変わった → 緩める。追加のみ → 締める
    ho, hn = _hook_set(o), _hook_set(n)
    if ho != hn:
        gone, new = sorted(ho - hn), sorted(hn - ho)
        if gone:
            lines.append("フックが %d 件なくなる／変わります" % len(gone))
            loosen = True
        if new:
            lines.append("フックが %d 件増えます／入れ替わります（追加されるスクリプトの中身も確認が必要）" % len(new))
            loosen = True
    # そのほかの設定キー（disableAllHooks・env・enableAllProjectMcpServers など）はすべて「緩めうる」として扱う
    for k in sorted((set(o) | set(n)) - {"permissions", "hooks"}):
        if o.get(k) != n.get(k):
            lines.append("設定 %s が変わります（権限・動作に関わりうる設定）" % k)
            loosen = True
    if not lines:
        return ["権限・フック・設定の変更はありません"], "neutral"
    return lines, ("loosen" if loosen else "tighten" if tighten else "neutral")


RISK_JA = {"loosen": "権限・検査を**緩める**（または緩めうる）変更", "tighten": "権限・検査を**締める**変更",
           "neutral": "権限・検査の強さは変わらない変更", "unknown": "影響を判定できない変更"}


def summarize(path, old, new):
    """人が読む変更サマリ。(行のリスト, 判定)"""
    a, r = line_stats(old or "", new or "")
    lines = ["対象: %s（+%d / -%d 行）" % (norm(path), a, r)]
    risk = "unknown"
    n = norm(path)
    if n.endswith("settings.json") or n.endswith("settings.local.json"):
        d, risk = settings_delta(old, new)
        lines += ["権限の変化: " + x for x in d]
    else:
        diff = [l for l in difflib.unified_diff((old or "").splitlines(), (new or "").splitlines(), lineterm="", n=0) if l[:1] in "+-" and l[:3] not in ("+++", "---")]
        lines += ["  " + l[:140] for l in diff[:20]]
        if len(diff) > 20:
            lines.append("  …（ほか %d 行。変更が大きいので、評価者の要約と合わせて慎重に確認してください）" % (len(diff) - 20))
    return lines, risk
