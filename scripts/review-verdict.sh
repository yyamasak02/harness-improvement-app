#!/usr/bin/env bash
# PR の JSON（gh pr view --json comments,reviews）を標準入力から読み、指定 SHA に対する最新のレビュー判定を出力する。
#   使い方: gh pr view N --json comments,reviews | scripts/review-verdict.sh <head-sha>
#   出力: APPROVE | REQUEST_CHANGES | NONE
set -euo pipefail
SHA="${1:?head sha が必要です}" python3 -I -c '
import json, os, re, sys
sha = os.environ["SHA"]
d = json.load(sys.stdin)
bodies = [c.get("body") or "" for c in d.get("comments", [])] + [r.get("body") or "" for r in d.get("reviews", [])]
pat = re.compile(r"<!-- review-agent: verdict=(APPROVE|REQUEST_CHANGES) sha=([0-9a-f]{7,40}) -->")
last = "NONE"
for b in bodies:
    for verdict, s in pat.findall(b):
        if sha.startswith(s) or s.startswith(sha):
            last = verdict
print(last)
'
