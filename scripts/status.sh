#!/usr/bin/env bash
# 途中再開用: どのフェーズからでも現状を把握し、次の行動を示す。読み取り専用（何も変更しない）。
#   scripts/status.sh           … 詳細
#   scripts/status.sh --brief   … 要点のみ（SessionStart フック用）
# gh が使えない・オフラインでも、取得できた範囲で表示する。
set -uo pipefail
cd "$(dirname "$0")/.."
brief=0; [ "${1:-}" = "--brief" ] && brief=1

have_gh=0
[ -z "${STATUS_NO_GH:-}" ] && command -v gh >/dev/null 2>&1 && have_gh=1   # STATUS_NO_GH=1 でオフライン動作（テスト用）
# gh の呼び出しは可能なら 15 秒でタイムアウトする（macOS には timeout が無いので gtimeout / 無しにフォールバック）。
# 失敗は握りつぶして空を返す。「失敗」と「該当なし」を区別したい呼び出しは gh_rc を使う。
# （--json を付けない gh pr view は Projects (classic) の警告で失敗する）
TO=""; command -v timeout >/dev/null 2>&1 && TO="timeout 15"; [ -z "$TO" ] && command -v gtimeout >/dev/null 2>&1 && TO="gtimeout 15"
g() { if [ $have_gh -eq 1 ]; then $TO gh "$@" 2>/dev/null || true; fi; }
# gh_rc: 標準出力+標準エラーを GH_OUT に、終了コードを GH_RC に入れる
gh_rc() { GH_OUT=$($TO gh "$@" 2>&1); GH_RC=$?; }
section() { printf '\n== %s\n' "$1"; }

branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "?")
dirty=$(git status --porcelain 2>/dev/null)
sha=$(git rev-parse HEAD 2>/dev/null || echo "")
next=""   # 次にすべきこと（最初に決まったものを採用）
decide() { [ -n "$next" ] || next="$1"; }

section "Git"
echo "ブランチ: $branch ($(git rev-parse --short HEAD 2>/dev/null))"
if [ -n "$dirty" ]; then
  echo "未コミットの変更: $(echo "$dirty" | wc -l | tr -d ' ') 件"
  [ $brief -eq 1 ] || echo "$dirty" | head -15 | sed 's/^/  /'
  decide "未コミットの変更がある → 内容を確認し、続きを実装するか、まとめて commit する（scripts/ship.sh）"
else
  echo "作業ツリー: クリーン"
fi
if git rev-parse --abbrev-ref '@{u}' >/dev/null 2>&1; then
  ahead=$(git rev-list --count '@{u}..HEAD' 2>/dev/null || echo 0)
  behind=$(git rev-list --count 'HEAD..@{u}' 2>/dev/null || echo 0)
  echo "リモートとの差: 未 push ${ahead} 件 / 未取り込み ${behind} 件"
  [ "$ahead" -gt 0 ] && decide "未 push のコミットがある → scripts/ship.sh で push する"
else
  [ "$branch" != "main" ] && echo "リモート追跡ブランチ: なし（未 push）"
fi
[ $brief -eq 1 ] || { echo "直近のコミット:"; git log --oneline -5 2>/dev/null | sed 's/^/  /'; }

# ---- GitHub への接続確認（取得失敗と「該当なし」を混同しないため）----
gh_ok=0; owner=""
if [ $have_gh -eq 1 ]; then
  gh_rc repo view --json owner --jq .owner.login
  if [ "$GH_RC" -eq 0 ] && [ -n "$GH_OUT" ]; then gh_ok=1; owner="$GH_OUT"; else
    section "GitHub"
    echo "GitHub の状態を取得できません（未認証・オフライン・レート制限など）: $(echo "$GH_OUT" | head -1)"
    decide "GitHub の状態を取得できない → PR・CI・レビューの判定は不能。git の状態だけで判断し、ユーザーに gh の認証（gh auth status）を確認してもらう。PR を作る・マージする判断はしない"
  fi
else
  section "GitHub"; echo "gh が使えません（未インストール、または STATUS_NO_GH 指定）。PR・issue の情報は表示できません"
  decide "gh が使えない → git の状態だけで判断する。PR・CI・レビューは判定不能"
fi

# ---- 現在のブランチの PR ----
if [ "$branch" != "main" ] && [ $gh_ok -eq 1 ]; then
  section "現在のブランチの PR"
  gh_rc pr view --json number --jq .number
  pr=""
  if [ "$GH_RC" -eq 0 ] && [ -n "$GH_OUT" ]; then pr="$GH_OUT"
  elif echo "$GH_OUT" | grep -qi "no pull requests found"; then pr="none"
  else
    echo "PR の有無を取得できません: $(echo "$GH_OUT" | head -1)"
    decide "PR の状態を取得できない → 判定不能。しばらくして再実行するか、gh pr list で手動確認する（PR を重複作成しない）"
  fi
  if [ "$pr" = "none" ]; then
    echo "PR なし"
    [ -n "$dirty" ] || decide "ブランチはあるが PR が無い → scripts/ship.sh で PR を作成する（その前に quality-gate と関連 issue の確認）"
  elif [ -n "$pr" ]; then
    state=$(g pr view "$pr" --json state --jq .state); title=$(g pr view "$pr" --json title --jq .title)
    echo "#$pr $title [$state]"
    if [ "$state" = "MERGED" ]; then
      decide "PR #$pr はマージ済み → git switch main && git pull し、不要なブランチを削除する。docs/handoff.md を更新する"
    elif [ "$state" = "CLOSED" ]; then
      decide "PR #$pr はクローズ済み（未マージ）→ 理由を確認（gh pr view $pr --comments）。再開するなら新しい PR を作る"
    else
      head=$(g pr view "$pr" --json headRefOid --jq .headRefOid)
      verdict=$(g pr view "$pr" --json comments,reviews | bash scripts/review-verdict.sh "${head:-none}" "$owner,github-actions,claude" 2>/dev/null || echo NONE)
      [ -n "$verdict" ] || verdict=NONE
      echo "head: ${head:0:7} / 独立レビューの判定（この head に対して）: $verdict"
      checks=$(g pr checks "$pr")
      if [ -n "$checks" ]; then
        fails=$(echo "$checks" | awk -F'\t' '$2=="fail"' | wc -l | tr -d ' ')
        pend=$(echo "$checks" | awk -F'\t' '$2=="pending"' | wc -l | tr -d ' ')
        pass=$(echo "$checks" | awk -F'\t' '$2=="pass"' | wc -l | tr -d ' ')
        echo "CI: 成功 $pass / 失敗 $fails / 実行中 $pend"
        [ $brief -eq 1 ] || echo "$checks" | awk -F'\t' '$2=="fail"{print "  失敗: "$1}'
      else
        fails=0; pend=0; echo "CI: 情報なし"
      fi
      if [ "${fails:-0}" -gt 0 ]; then
        decide "PR #$pr の CI が失敗 → 失敗ジョブのログを読んで原因を直す（gh run view <run-id> --log-failed。実行完了後に取得可）"
      elif [ "${pend:-0}" -gt 0 ]; then
        decide "PR #$pr の CI が実行中 → 完了を待つ（レビューは CI が確定してから）"
      elif [ "$verdict" = "NONE" ]; then
        decide "PR #$pr に現在の head への独立レビューが無い → review-pr skill でレビューを依頼する"
      elif [ "$verdict" = "REQUEST_CHANGES" ]; then
        decide "PR #$pr は REQUEST_CHANGES → 最新のレビューコメント（gh pr view $pr --comments）の blocker/major を直し、再レビューを受ける"
      elif [ "$verdict" = "APPROVE" ]; then
        decide "PR #$pr は APPROVE かつ CI 成功 → bash scripts/merge-pr.sh $pr でマージする"
      fi
    fi
  fi
fi

# ---- 全 open PR ----
if [ $gh_ok -eq 1 ]; then
  section "open な PR"
  prs=$(g pr list --state open --json number,title,headRefName --jq '.[]|"#\(.number) \(.title) (\(.headRefName))"')
  [ -n "$prs" ] && echo "$prs" | sed 's/^/  /' || echo "  なし"

  section "open な issue（優先度順）"
  command -v python3 >/dev/null 2>&1 || echo "  （python3 が無いため issue 一覧を整形できません）"
  issues=$(g issue list --state open --limit 100 --json number,title,labels | python3 -I -c '
import json, sys
try:
    items = json.load(sys.stdin)
except ValueError:
    sys.exit(0)
rows = []
for i in items:
    ls = [l["name"] for l in i["labels"]]
    pr = next((l.split("/", 1)[1] for l in ls if l.startswith("priority/")), "?")
    tags = [l.split("/", 1)[1] for l in ls if l in ("status/needs-human", "status/blocked") or l.startswith("release/")]
    rows.append((pr, i["number"], i["title"], ",".join(tags)))
for pr, n, t, tags in sorted(rows):
    print("%s\t#%d\t%s\t%s" % (pr, n, t, tags))
')
  if [ -z "$issues" ]; then echo "  なし（または取得できません）"; else
    echo "$issues" | awk -F'\t' '{printf "  [%s] %s %s  (%s)\n",$1,$2,$3,$4}' | { if [ $brief -eq 1 ]; then head -12; else cat; fi; }
    human=$(echo "$issues" | grep -c 'needs-human' || true)
    echo "  → 人の作業が必要（needs-human）: ${human} 件。エージェントが進められるのは、それ以外の p0/p1 から"
  fi
  # main 上でクリーンなら、次に着手する issue の候補を示す
  if [ "$branch" = "main" ] && [ -z "$dirty" ] && [ -z "$next" ]; then
    cand=$(echo "$issues" | grep -v 'needs-human\|blocked' | head -1)
    if [ -n "$cand" ]; then
      decide "main がクリーン → 次の issue に着手: $(echo "$cand" | awk -F'\t' '{print $2" "$3}')（related-issues.sh → ブランチ作成 → implement-task）"
    else
      decide "着手できる issue が無い → 人の作業待ち（needs-human）の一覧を報告する。リリース準備（release skill）の確認"
    fi
  fi
fi

# ---- タスクと blockers ----
section "docs/tasks.md"
if [ -f docs/tasks.md ]; then
  total=$(grep -cE '^- \[[ x]\] T' docs/tasks.md || true); done_=$(grep -cE '^- \[x\] T' docs/tasks.md || true)
  echo "完了 ${done_} / ${total}"
  [ $brief -eq 1 ] || grep -E '^- \[ \] T' docs/tasks.md | head -5 | sed 's/^/  未完了: /'
fi
if [ -f docs/blockers.md ] && [ "$(grep -vE '^(#|\s*$|（なし）)' docs/blockers.md | wc -l | tr -d ' ')" -gt 0 ]; then
  section "docs/blockers.md（未解決のブロッカー）"
  grep -vE '^\s*$' docs/blockers.md | head -20 | sed 's/^/  /'
fi

# ---- 引き継ぎメモ ----
section "docs/handoff.md（引き継ぎメモ。鵜呑みにせず上の現実と照合する）"
if [ -f docs/handoff.md ]; then
  last=$(git log -1 --format=%h -- docs/handoff.md 2>/dev/null)
  if [ -n "$last" ]; then
    behind_commits=$(git rev-list --count "$last..HEAD" 2>/dev/null || echo 0)
    echo "最終更新: $(git log -1 --format=%cd --date=format:%Y-%m-%d\ %H:%M -- docs/handoff.md)（以降のコミット ${behind_commits} 件）"
    [ "$behind_commits" -gt 8 ] && echo "警告: 引き継ぎメモが古い可能性があります。現実（git log / PR / issue）を優先し、再開後に更新してください"
  fi
  if [ $brief -eq 1 ]; then sed -n '/^## 次の一手/,/^## /p' docs/handoff.md | sed '$d' | head -12; else sed 's/^/  /' docs/handoff.md | head -60; fi
else
  echo "  なし（docs/handoff.md を作成してください）"
fi

section "次にすべきこと"
if [ -n "$next" ]; then echo "  → $next"; else echo "  → 状況から自動判定できません。上の情報から判断してください（resume skill 参照）"; fi
echo
echo "（再開手順: .claude/skills/resume/SKILL.md）"
