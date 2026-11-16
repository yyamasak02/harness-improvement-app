#!/usr/bin/env bash
# PR の JSON（gh pr view --json comments,reviews）を標準入力から読み、指定 SHA に対する最新のレビュー判定を出力する。
#   使い方: gh pr view N --json comments,reviews | scripts/review-verdict.sh <head-sha> <許可する投稿者をカンマ区切り>
#   出力: APPROVE | REQUEST_CHANGES | NONE
# 有効なマーカーの条件: ① 許可された投稿者 ② 本文の最終行がマーカー ③ sha が head と一致。時系列で最新のものを採用する。
set -euo pipefail
SHA="${1:?head sha が必要です}" ALLOWED="${2:?許可する投稿者が必要です}" python3 -I -c '
import json, os, re, sys
sha = os.environ["SHA"]
allowed = {a.strip() for a in os.environ["ALLOWED"].split(",") if a.strip()}
d = json.load(sys.stdin)
items = []
for c in d.get("comments", []):
    items.append((c.get("createdAt") or "", c.get("author") or {}, c.get("body") or ""))
for r in d.get("reviews", []):
    items.append((r.get("submittedAt") or "", r.get("author") or {}, r.get("body") or ""))
items.sort(key=lambda x: x[0])
pat = re.compile(r"^<!-- review-agent: verdict=(APPROVE|REQUEST_CHANGES) sha=([0-9a-f]{7,40}) -->$")
last = "NONE"
for _, author, body in items:
    if author.get("login") not in allowed:
        continue
    lines = [l.strip() for l in body.strip().splitlines()]
    m = pat.match(lines[-1]) if lines else None
    if m and (sha.startswith(m.group(2)) or m.group(2).startswith(sha)):
        last = m.group(1)
print(last)
'
