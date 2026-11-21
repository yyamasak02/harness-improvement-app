#!/usr/bin/env bash
# 性能計測（Go ベンチ＋フロントのビルドサイズ）。開発・CI 専用。読み取り専用（結果は perf-results/ に出す）。
# 使い方: bash scripts/perf-report.sh [--size-only]
# 前回結果を perf-previous/bench.tsv に置くと前回比を出す（CI のアーティファクト perf-metrics を展開）。
# 時間系は揺れるため失敗にしない（警告のみ）。サイズが予算を超えたら失敗する。詳細: docs/perf-metrics.md
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

# 予算（バイト。docs/perf-metrics.md の表と一致させる）
JS_BUDGET=${JS_BUDGET:-1500000}
CSS_BUDGET=${CSS_BUDGET:-150000}
WARN_RATIO=${WARN_RATIO:-1.5} # 前回比でこの倍率を超えたら警告

out=perf-results
mkdir -p "$out"
fail=0

if [ "${1:-}" != "--size-only" ]; then
  echo "== Go ベンチ（ns/op。合成データ）"
  go test -p 1 ./internal/... -run '^$' -bench . -benchtime 20x -count 3 2>&1 \
    | awk '/^Benchmark/ { n=$1; sub(/-[0-9]+$/, "", n); v[n]=v[n] " " $3 } END { for (k in v) { split(substr(v[k],2), a, " "); c=asorti_n(a); print k "\t" med(a, c) } }
      function asorti_n(a,  i, j, t, c) { c=0; for (i in a) c++; for (i=1;i<=c;i++) for (j=i+1;j<=c;j++) if (a[j]+0 < a[i]+0) { t=a[i]; a[i]=a[j]; a[j]=t } return c }
      function med(a, c) { return a[int((c+1)/2)] }' \
    | sort >"$out/bench.tsv"
  prev=perf-previous/bench.tsv
  while IFS=$'\t' read -r name ns; do
    line=$(printf '  %-40s %12d ns/op' "$name" "$ns")
    if [ -f "$prev" ]; then
      p=$(awk -F'\t' -v n="$name" '$1==n {print $2}' "$prev")
      if [ -n "$p" ] && [ "$p" -gt 0 ]; then
        ratio=$(awk -v a="$ns" -v b="$p" 'BEGIN{printf "%.2f", a/b}')
        line="$line  前回比 x$ratio"
        if awk -v r="$ratio" -v w="$WARN_RATIO" 'BEGIN{exit !(r>w)}'; then line="$line  ⚠ 悪化"; fi
      fi
    fi
    echo "$line"
  done <"$out/bench.tsv"
fi

echo "== フロントのビルドサイズ（frontend/dist。バイト）"
if [ -d frontend/dist/assets ]; then
  js=$(cat frontend/dist/assets/*.js 2>/dev/null | wc -c | tr -d ' ')
  css=$(cat frontend/dist/assets/*.css 2>/dev/null | wc -c | tr -d ' ')
  printf '%s\t%s\n%s\t%s\n' js "$js" css "$css" >"$out/size.tsv"
  for pair in "js:$js:$JS_BUDGET" "css:$css:$CSS_BUDGET"; do
    IFS=: read -r k v b <<<"$pair"
    mark=""
    if [ "$v" -gt "$b" ]; then mark="  ✗ 予算超過"; fail=1; fi
    printf '  %-4s %9d / 予算 %9d (%d%%)%s\n' "$k" "$v" "$b" $((v * 100 / b)) "$mark"
  done
else
  echo "  （frontend/dist が無い。cd frontend && npm run build の後に再実行）"
fi
exit "$fail"
