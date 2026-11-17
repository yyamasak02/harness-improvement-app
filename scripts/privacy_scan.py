#!/usr/bin/env python3
"""開発プロセスでの個人情報・秘密の混入を検査する（check-privacy.sh から呼ぶ）。

  privacy_scan.py diff    … 標準入力の unified diff の「追加行」を検査
  privacy_scan.py text    … 標準入力のテキスト（PR/issue 本文・コミットメッセージ）を検査

検出した値そのものは出力しない（CI ログは公開されうる）。種別・場所・伏せた断片だけを出す。
意図したダミー値は、その行に `privacy-allow: 理由` を書いて明示する（レビューで理由を確認する）。
終了コード: 0=問題なし 1=検出あり
"""
import getpass
import os
import re
import subprocess
import sys

PLACEHOLDER_NAMES = {"demo", "user", "username", "you", "runner", "example", "test", "alice", "bob", "carol",
                     "name", "me", "foo", "x", "u", "yourname", "your-name", "someone", "dev", "admin", "root", "work"}
EMAIL_OK = re.compile(r"(@users\.noreply\.github\.com|^noreply@(anthropic|github)\.com|@example\.[a-z.]+|@test\.|^git@github\.com|@localhost)$", re.I)
DOC_IPS = ("192.0.2.", "198.51.100.", "203.0.113.", "127.", "0.")


def personal_values():
    """この環境固有の個人値（OS ユーザー名・ホームのパス・git の実メール）。4 文字以上のみ。"""
    vals = set()
    for v in (getpass.getuser(), os.path.basename(os.path.expanduser("~"))):
        if v and len(v) >= 4 and v.lower() not in PLACEHOLDER_NAMES:
            vals.add(v)
    home = os.path.expanduser("~")
    if len(home) > 6:
        vals.add(home)
    try:
        email = subprocess.run(["git", "config", "user.email"], capture_output=True, text=True).stdout.strip()
        if email and not EMAIL_OK.search(email):
            vals.add(email)
    except OSError:
        pass
    for v in os.environ.get("PRIVACY_EXTRA", "").split(","):
        if len(v.strip()) >= 4:
            vals.add(v.strip())
    return vals


def redact(s):
    return s[:2] + "…(%d文字)" % len(s)


def lux(s):
    d = [int(c) for c in s if c.isdigit()]
    if not 13 <= len(d) <= 19:
        return False
    t, alt = 0, False
    for x in reversed(d):
        if alt:
            x = x * 2 - 9 if x * 2 > 9 else x * 2
        t += x
        alt = not alt
    return t % 10 == 0


RULES = [
    ("email", re.compile(r"[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}"), lambda m: not EMAIL_OK.search(m)),
    ("phone", re.compile(r"\+\d{1,3}[ -]\d{1,4}[ -]\d{2,4}[ -]\d{3,4}\b|\b0\d{1,4}-\d{1,4}-\d{3,4}\b|\b0[5789]0\d{8}\b"), None),
    ("postal-code", re.compile(r"〒\s?\d{3}-?\d{4}"), None),
    ("credit-card", re.compile(r"(?<![\w\-.])(?:\d[ -]?){12,18}\d(?![\w\-])"), lux),
    ("home-path", re.compile(r"(?:/home/|/Users/)([^/\s\"'<>:$*`,;)\]]+)|[A-Za-z]:\\Users\\([^\\\s\"'<>:$*`,;)\]]+)"),  # privacy-allow: 検出用の正規表現
     lambda m: _home_name(m) not in PLACEHOLDER_NAMES and not _home_name(m).startswith(("<", "{", "%"))),
    ("ip-address", re.compile(r"(?<![\d.])(?:\d{1,3}\.){3}\d{1,3}(?![\d.])"),
     lambda m: all(0 <= int(p) <= 255 for p in m.split(".")) and not m.startswith(DOC_IPS)),
    ("secret", re.compile(r"sk-ant-[A-Za-z0-9_\-]{20,}|AKIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|xox[baprs]-[A-Za-z0-9\-]{20,}|eyJ[A-Za-z0-9_\-]{8,}\.eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}|AIza[0-9A-Za-z_\-]{35}|-----BEGIN [A-Z ]*PRIVATE KEY-----"), lambda m: "dummy" not in m.lower()),
]


def _home_name(m):
    g = re.search(r"(?:/home/|/Users/)([^/\s\"'<>:$*`,;)\]]+)|[A-Za-z]:\\Users\\([^\\\s\"'<>:$*`,;)\]]+)", m)  # privacy-allow: 検出用の正規表現
    return (g.group(1) or g.group(2) or "").lower() if g else ""


def scan_line(line, personal):
    out = []
    if "privacy-allow:" in line:
        return out
    for kind, rx, ok in RULES:
        for m in rx.finditer(line):
            s = m.group(0)
            if ok is None or ok(s):
                out.append((kind, redact(s)))
    low = line.lower()
    for v in personal:
        if v.lower() in low:
            out.append(("personal-value", redact(v)))
    return out


def main():
    mode = sys.argv[1] if len(sys.argv) > 1 else "text"
    personal = personal_values()
    hits = []
    if mode == "diff":
        f, ln = "?", 0
        for raw in sys.stdin.read().splitlines():
            if raw.startswith("+++ "):
                f = raw[6:] if raw.startswith("+++ b/") else raw[4:]
                continue
            if raw.startswith("@@"):
                m = re.search(r"\+(\d+)", raw)
                ln = int(m.group(1)) - 1 if m else 0
                continue
            if raw.startswith("+"):
                ln += 1
                for kind, frag in scan_line(raw[1:], personal):
                    hits.append("%s:%d [%s] %s" % (f, ln, kind, frag))
            elif not raw.startswith("-"):
                ln += 1
    else:
        for i, line in enumerate(sys.stdin.read().splitlines(), 1):
            for kind, frag in scan_line(line, personal):
                hits.append("本文 %d 行目 [%s] %s" % (i, kind, frag))
    if hits:
        print("個人情報・秘密らしきものを検出しました（値は伏せています）:")
        for h in hits[:30]:
            print("  " + h)
        print("意図したダミー値なら、その行に 'privacy-allow: 理由' を書く。実在の値ならコミット・投稿しない。")
        sys.exit(1)
    print("check-privacy OK")


if __name__ == "__main__":
    main()
