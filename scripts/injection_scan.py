#!/usr/bin/env python3
"""プロンプトインジェクションらしき文言・不可視文字の検出（フック・gh-read.sh から使う）。

CLI: injection_scan.py   … 標準入力を検査し、検出があれば種別を 1 行ずつ出力して終了コード 1。
検出は「警告のための決定論的な目印」であり、検出されなかったことは安全の保証ではない。
"""
import re
import sys

INVISIBLE = re.compile("[​-‏‪-‮⁠-⁤⁦-⁩﻿\U000e0000-\U000e007f]")

PATTERNS = [
    ("上書き指示", r"ignore\s+(?:all\s+|any\s+|the\s+)?(?:previous|prior|above|earlier)\s+(?:instructions?|prompts?|rules?)"),
    ("上書き指示", r"disregard\s+(?:all\s+|any\s+|the\s+)?(?:previous|prior|above|earlier)?\s*(?:instructions?|prompts?|rules?)"),
    ("上書き指示", r"forget\s+(?:all\s+|everything\s+)?(?:previous|prior|above|earlier)\s+(?:instructions?|prompts?)"),
    ("上書き指示", r"(?:これまで|以前|上記|前)の(?:指示|命令|プロンプト)を(?:すべて|全て)?(?:無視|忘れ|破棄)"),
    ("上書き指示", r"(?:指示|命令)を(?:無視|上書き)して"),
    ("役割の乗っ取り", r"you\s+are\s+now\s+(?:a|an|in)\b|new\s+instructions?\s*:|</?\s*(?:system|assistant|human)\s*>"),
    ("情報の持ち出し", r"(?:reveal|print|show|leak|send|upload|post)\s+(?:me\s+)?(?:your\s+|the\s+|all\s+)?(?:system\s+prompt|api[ _-]?keys?|secrets?|credentials?|tokens?|env(?:ironment)?\s+variables?|\.env)"),
    ("情報の持ち出し", r"システムプロンプトを(?:表示|出力|教え)|(?:秘密|トークン|認証情報|APIキー)を(?:送|出力|表示|投稿)"),
    ("権限の変更", r"--dangerously-skip-permissions|bypassPermissions|dangerouslyDisableSandbox|disable\s+(?:the\s+)?(?:hooks?|sandbox|permissions?)|(?:edit|modify|change|remove)\s+(?:the\s+)?(?:\.claude/settings|settings\.json|deny\s+list)|settings\.json\s*を(?:編集|変更|書き換え)|deny\s*を(?:削除|外)"),
    ("危険なコマンドの指示", r"(?:curl|wget)\b[^\n|]*\|\s*(?:sudo\s+)?(?:ba|z)?sh\b|\bbase64\s+(?:-d|--decode)\b[^\n]*\|\s*(?:ba|z)?sh"),
    ("AI への呼びかけ", r"(?:note|message|instructions?)\s+(?:to|for)\s+(?:the\s+)?(?:ai|assistant|claude|llm|agent)\b|\bdear\s+(?:ai|assistant|claude)\b|(?:AI|アシスタント|Claude)(?:さん)?(?:へ|に)(?:指示|お願い|命令)"),
]
COMPILED = [(k, re.compile(p, re.I)) for k, p in PATTERNS]


def scan(text):
    """検出した種別のリスト（重複なし・出現順）を返す。"""
    found = []
    if INVISIBLE.search(text):
        found.append("不可視文字")
    for kind, rx in COMPILED:
        if rx.search(text) and kind not in found:
            found.append(kind)
    return found


def strip_invisible(text):
    return INVISIBLE.sub("", text)


if __name__ == "__main__":
    kinds = scan(sys.stdin.read())
    for k in kinds:
        print(k)
    sys.exit(1 if kinds else 0)
