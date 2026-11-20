// UX 指標の計測の共通部品。開発・CI 専用（配布物には含まれない。送信・永続化は結果 JSON のローカル書き出しのみ）。
import {mkdirSync, writeFileSync} from 'node:fs'
import {join} from 'node:path'
import type {Page} from '@playwright/test'

/** 予算（docs/ux-metrics.md と一致させる）。緩める場合は docs/decisions.md に理由を書き issue 化する。 */
export const budgets = {
  startupMs: 3000,
  transitionMedianMs: 500,
  transitionMaxMs: 1500,
  largeListRenderMs: 5000,
  largeScrollMs: 4000,
  largeDetailRenderMs: 5000,
  overflowCount: 0,
  clsPerPage: 0.1,
  axeSeriousCount: 0,
  clicksFindSession: 2,
  clicksApplyProposal: 5,
  clicksRollback: 4,
  keyboardTabsToSessions: 12,
} as const

export interface MetricRecord {
  id: string
  label: string
  unit: string
  value: number
  budget: number
  detail?: unknown
}

const outDir = join(process.cwd(), 'ux-metrics-results')

/** 1 指標 = 1 ファイル（並列・再実行でも壊れない）。集計は e2e/summarize-ux.mjs。 */
export function record(m: MetricRecord) {
  mkdirSync(outDir, {recursive: true})
  writeFileSync(join(outDir, `${m.id}.json`), JSON.stringify(m, null, 2) + '\n')
}

export function median(xs: number[]): number {
  const s = [...xs].sort((a, b) => a - b)
  const mid = Math.floor(s.length / 2)
  return s.length % 2 ? s[mid] : (s[mid - 1] + s[mid]) / 2
}

export const round = (n: number, d = 1) => Math.round(n * 10 ** d) / 10 ** d

export const pages = [
  {path: '/', name: '概要', link: '概要'},
  {path: '/sessions', name: 'セッション', link: 'セッション'},
  {path: '/improve', name: 'CLAUDE.md 改善', link: 'CLAUDE.md 改善'},
  {path: '/memory', name: 'メモリ', link: 'メモリ'},
  {path: '/skills', name: '自己ナレッジ生成', link: 'ナレッジ生成'},
  {path: '/reports', name: '日次振り返り', link: '日次振り返り'},
  {path: '/extensions', name: '拡張機能', link: '拡張機能'},
  {path: '/history', name: '変更履歴', link: '変更履歴'},
  {path: '/settings', name: '設定', link: '設定'},
]

/** 大量データ用の合成ブリッジ。window.go を差し込むと api.ts が本物の経路（wailsApi）を使う。実データ・実 ~/.claude は使わない。 */
export async function installLargeDataBridge(page: Page, sessionCount: number, messageCount: number) {
  await page.addInitScript(({sessionCount, messageCount}) => {
    const mkSession = (i: number) => ({
      id: `sess-${String(i).padStart(5, '0')}`, project: '-home-demo-app', filePath: '', cwd: '/home/demo/app',
      gitBranch: `feat/branch-${i % 50}`, startedAt: new Date(Date.UTC(2026, 0, 1) + i * 60000).toISOString(),
      endedAt: '', userMsgs: 3, assistantMsgs: 4, toolErrors: i % 7 === 0 ? 1 : 0, corrections: i % 5 === 0 ? 1 : 0,
      inputTokens: 10, outputTokens: 20, cacheRead: 0, cacheCreate: 0,
    })
    const sessions = Array.from({length: sessionCount}, (_, i) => mkSession(i))
    const messages = Array.from({length: messageCount}, (_, i) => ({
      id: i + 1, sessionId: 'sess-00000', project: '-home-demo-app', ts: '2026-01-01T00:00:00Z',
      role: i % 2 ? 'assistant' : 'user', kind: 'text', tool: '', text: `合成メッセージ ${i} `.repeat(1 + (i % 8)),
      isError: false, isCorrection: false,
    }))
    const handlers: Record<string, (...a: unknown[]) => unknown> = {
      ListProjects: () => [{project: '-home-demo-app', cwd: '/home/demo/app', sessions: sessionCount, lastAt: '2026-01-01T00:00:00Z'}],
      ListSessions: () => sessions,
      GetSession: () => ({session: sessions[0], messages}),
    }
    const w = window as unknown as Record<string, unknown>
    w.go = {main: {App: new Proxy({}, {get: (_t, name: string) => async (...a: unknown[]) => (handlers[name] ? handlers[name](...a) : null)})}}
    w.runtime = {EventsOnMultiple: () => () => {}, EventsOff: () => {}}
  }, {sessionCount, messageCount})
}
