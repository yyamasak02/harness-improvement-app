# 性能指標（Go ベンチ・ビルドサイズ）

目的: 大量のセッションログを扱うバックエンドの処理時間と、フロントのビルドサイズの退行を数字で検出する。
フロントの操作の速さは `docs/ux-metrics.md`（Playwright）が担当し、ここは Go 側とサイズを担当する。
計測は**開発・CI 専用**。アプリに計測コード・外部送信は入れない。

## 使い方
- 実行: `cd frontend && npm run build`（サイズ計測に必要）→ `bash scripts/perf-report.sh`（`--size-only` でサイズだけ）。
- 結果: `perf-results/bench.tsv`（ベンチ名と ns/op の中央値）・`size.tsv`。gitignore 済み。CI の `perf-metrics` ジョブがジョブサマリーに表を出し、アーティファクト `perf-metrics` に保存する。
- 前回比: 前回のアーティファクトを `perf-previous/` に展開すると、前回比（x 倍）を出し、1.5 倍超なら「⚠ 悪化」と表示する。

## 判定
| 種類 | 扱い |
|---|---|
| 時間系（ns/op） | **警告のみ**（CI は遅く揺れる。3 回の中央値）。悪化が続けば `type/tech-debt` の issue にする |
| ビルドサイズ | 予算超過で CI 失敗。予算は `scripts/perf-report.sh` の `JS_BUDGET` / `CSS_BUDGET`（下表と一致させる） |

## 指標
| 指標 | 場所 | 予算 | 実測の目安 |
|---|---|---|---|
| マスキング（通常文 / 秘密を含む文 各 約 15KB） | `internal/masking/bench_test.go` | なし（警告のみ） | 約 1〜2 ms / 約 2〜4 ms |
| jsonl パース（4000 行） | `internal/collector/bench_test.go` | なし（警告のみ） | 約 20〜35 ms |
| JS バンドル合計 | `frontend/dist/assets/*.js` | 1,500,000 B | 約 860 KB |
| CSS 合計 | `frontend/dist/assets/*.css` | 150,000 B | 約 22 KB |

## ベンチを足すとき
- 入力は合成データのみ（`b.TempDir()`・生成）。実ログは読まない。名前は `Benchmark<対象>`、`b.ReportAllocs()` を付ける。
- 予算を緩めて通さない。緩める場合は `docs/decisions.md` に理由を書き issue にする。

## 未整備
- digest.Build（store が必要）・applier・SQLite 取り込みのベンチ。必要になったら足す。
- 実機（Wails WebView）での計測。
