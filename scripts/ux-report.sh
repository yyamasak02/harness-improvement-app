#!/usr/bin/env bash
# UX 指標の計測結果（JSON）から、予算に対する達成率・前回比・悪化した指標を表にする。読み取り専用（何も書かない・送らない）。
# 使い方: bash scripts/ux-report.sh [--current DIR|FILE] [--previous DIR|FILE]
#   current  既定: frontend/ux-metrics-results（summary.json、無ければ個別 *.json）
#   previous 既定: frontend/ux-metrics-previous（CI の ux-metrics アーティファクトを展開して置く。docs/ux-metrics.md）
# 結果が無ければその旨を表示して正常終了する。予算超過があっても終了コードは 0（判定は CI の e2e:ux が行う）。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
cur=frontend/ux-metrics-results
prev=frontend/ux-metrics-previous
while [ $# -gt 0 ]; do
  case "$1" in
    --current) cur="${2:?--current に値が必要}"; shift 2 ;;
    --previous) prev="${2:?--previous に値が必要}"; shift 2 ;;
    -h|--help) sed -n '2,7p' "$0"; exit 0 ;;
    *) echo "不明な引数: $1" >&2; exit 2 ;;
  esac
done
python3 -I - "$cur" "$prev" <<'PY'
import json, os, sys

def load(path):
    """summary.json か、個別 *.json のディレクトリ/ファイルから {id: metric} を返す。無ければ None。"""
    if os.path.isdir(path):
        s = os.path.join(path, 'summary.json')
        if os.path.isfile(s):
            path = s
        else:
            ms = {}
            for f in sorted(os.listdir(path)):
                if f.endswith('.json'):
                    try:
                        with open(os.path.join(path, f), encoding='utf-8') as fh:
                            m = json.load(fh)
                    except (OSError, ValueError):
                        continue
                    if isinstance(m, dict) and 'id' in m and 'value' in m:
                        ms[m['id']] = m
            return ms or None
    if not os.path.isfile(path):
        return None
    try:
        with open(path, encoding='utf-8') as fh:
            d = json.load(fh)
    except (OSError, ValueError):
        return None
    items = d.get('metrics', []) if isinstance(d, dict) else []
    ms = {m['id']: m for m in items if isinstance(m, dict) and 'id' in m and 'value' in m}
    return ms or None

cur, prev = load(sys.argv[1]), load(sys.argv[2])
if cur is None:
    print(f'UX 計測結果がありません（{sys.argv[1]}）。cd frontend && npm run e2e:ux で計測するか、--current で指定してください。')
    sys.exit(0)
if prev is None:
    print(f'前回の結果がありません（{sys.argv[2]}）。前回比は出しません（CI の ux-metrics アーティファクトを --previous に置くと比較できます）。')

def fmt(v):
    return str(int(v)) if float(v) == int(v) else f'{v:g}'

print('| 指標 | 実測 | 予算（以下） | 達成率 | 前回 | 前回比 | 判定 |')
print('|---|---|---|---|---|---|---|')
over, worse = [], []
for id_, m in cur.items():
    v, b, u = float(m['value']), float(m['budget']), m.get('unit', '')
    ratio = '-' if b == 0 else f'{v / b * 100:.0f}%'
    p = prev.get(id_) if prev else None
    if p is None:
        pv, diff = '-', '-'
    else:
        pvv = float(p['value'])
        pv = fmt(pvv)
        d = v - pvv
        diff = '±0' if d == 0 else f'{d:+g}' + ('' if pvv == 0 else f' ({d / pvv * 100:+.0f}%)')
        # 悪化 = 値が増えた（全指標「小さいほど良い」）。時間系は揺れるので 20% 超かつ 10ms 以上の増加のみ悪化とみなす
        if d > 0 and (u != 'ms' or (d >= 10 and (pvv == 0 or d / pvv > 0.2))):
            worse.append(m.get('label', id_))
    if v > b:
        over.append(m.get('label', id_))
    unit = ' ' + u if u else ''
    print(f"| {m.get('label', id_)} | {fmt(v)}{unit} | {fmt(b)}{unit} | {ratio} | {pv} | {diff} | {'OK' if v <= b else '**超過**'} |")
print()
print(f'予算超過: {len(over)} 件' + (' — ' + ', '.join(over) if over else ''))
if prev is not None:
    print(f'悪化（前回比）: {len(worse)} 件' + (' — ' + ', '.join(worse) if worse else ''))
print('改善の余地の目安: 達成率が高い指標から見る（優先度の基準は docs/ux-metrics.md）。')
PY
