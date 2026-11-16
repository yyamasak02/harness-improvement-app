#!/usr/bin/env bash
# 完了の定義: collector・digest・masking・applier・runner のカバレッジが 70% 以上であること。
set -euo pipefail
cd "$(dirname "$0")/.."
fail=0
for pkg in collector digest masking applier runner; do
  line=$(go test -cover "./internal/$pkg" | grep -o 'coverage: [0-9.]*%')
  pct=${line#coverage: }; pct=${pct%\%}
  printf '%-10s %s%%\n' "$pkg" "$pct"
  awk -v p="$pct" 'BEGIN{exit !(p<70)}' && { echo "  -> 70% 未満"; fail=1; }
done
exit $fail
