// UX 指標の結果（ux-metrics-results/*.json）を 1 つの JSON と Markdown 表にまとめる。
// 使い方: node e2e/summarize-ux.mjs --clean（計測前に結果を消す）/ node e2e/summarize-ux.mjs（集計。予算超過なら終了コード 1）
// GITHUB_STEP_SUMMARY があればジョブサマリーに追記する。外部へは何も送らない。
import {existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync, appendFileSync} from 'node:fs'
import {join} from 'node:path'

const dir = join(process.cwd(), 'ux-metrics-results')

if (process.argv.includes('--clean')) {
  rmSync(dir, {recursive: true, force: true})
  mkdirSync(dir, {recursive: true})
  process.exit(0)
}

const files = existsSync(dir) ? readdirSync(dir).filter((f) => f.endsWith('.json') && f !== 'summary.json') : []
const metrics = files.sort().map((f) => JSON.parse(readFileSync(join(dir, f), 'utf8')))
if (metrics.length === 0) {
  console.error('計測結果がありません（ux-metrics-results/ が空）')
  process.exit(1)
}
const rows = metrics.map((m) => ({...m, ok: m.value <= m.budget}))
const failed = rows.filter((r) => !r.ok)

writeFileSync(join(dir, 'summary.json'), JSON.stringify({generatedAt: new Date().toISOString(), failed: failed.length, metrics: rows}, null, 2) + '\n')

const md = [
  '## UX 指標（docs/ux-metrics.md）',
  '',
  '| 指標 | 実測 | 予算（以下） | 判定 |',
  '|---|---|---|---|',
  ...rows.map((r) => `| ${r.label} | ${r.value}${r.unit ? ' ' + r.unit : ''} | ${r.budget}${r.unit ? ' ' + r.unit : ''} | ${r.ok ? 'OK' : '**超過**'} |`),
  '',
  failed.length ? `予算超過: ${failed.length} 件` : '全指標が予算内です。',
  '',
].join('\n')
console.log(md)
if (process.env.GITHUB_STEP_SUMMARY) appendFileSync(process.env.GITHUB_STEP_SUMMARY, md + '\n')
process.exit(failed.length ? 1 : 0)
