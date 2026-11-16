#!/usr/bin/env bash
# コミットメッセージ（1 行目）が Conventional 形式であることを検査する。pre-commit の commit-msg ステージ用。
msg=$(head -n1 "$1")
re='^(feat|fix|refactor|test|docs|chore|ci)(\([a-zA-Z0-9_,./-]+\))?: .+'
if [[ "$msg" =~ $re ]] || [[ "$msg" =~ ^(Merge|Revert) ]]; then exit 0; fi
echo "コミットメッセージは '<feat|fix|refactor|test|docs|chore|ci>(scope)?: 要約' の形式にしてください: $msg"
exit 1
