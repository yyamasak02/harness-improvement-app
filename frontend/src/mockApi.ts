// ブラウザ単体（npm run dev）やテスト用の mock 実装。状態はメモリ上のみ。
import type {Api} from './api'
import type * as T from './types'

const sessions: T.Session[] = [
  {id: 'sess-0002', project: '-home-demo-app', filePath: '', cwd: '/home/demo/app', gitBranch: 'feat/readme', startedAt: '2026-11-02T09:00:00Z', endedAt: '2026-11-02T09:01:10Z', userMsgs: 2, assistantMsgs: 2, toolErrors: 0, corrections: 1, inputTokens: 6, outputTokens: 6, cacheRead: 200, cacheCreate: 100},
  {id: 'sess-0001', project: '-home-demo-app', filePath: '', cwd: '/home/demo/app', gitBranch: 'main', startedAt: '2026-11-01T01:00:00Z', endedAt: '2026-11-01T01:03:00Z', userMsgs: 3, assistantMsgs: 4, toolErrors: 1, corrections: 1, inputTokens: 35, outputTokens: 70, cacheRead: 400, cacheCreate: 200},
]

const messages: Record<string, T.Message[]> = {
  'sess-0001': [
    {id: 1, sessionId: 'sess-0001', project: '-home-demo-app', ts: '2026-11-01T01:00:00Z', role: 'user', kind: 'text', tool: '', text: 'テストを実行して', isError: false, isCorrection: false},
    {id: 2, sessionId: 'sess-0001', project: '-home-demo-app', ts: '2026-11-01T01:00:05Z', role: 'assistant', kind: 'tool_use', tool: 'Bash', text: 'go test ./...', isError: false, isCorrection: false},
    {id: 3, sessionId: 'sess-0001', project: '-home-demo-app', ts: '2026-11-01T01:00:10Z', role: 'user', kind: 'tool_result', tool: 'Bash', text: 'FAIL: TestFoo', isError: true, isCorrection: false},
    {id: 4, sessionId: 'sess-0001', project: '-home-demo-app', ts: '2026-11-01T01:01:00Z', role: 'user', kind: 'text', tool: '', text: '違う、そうじゃなくて先に go vet を実行して', isError: false, isCorrection: true},
  ],
}

const targets: T.Target[] = [
  {kind: 'global', path: '/home/demo/.claude/CLAUDE.md', label: 'グローバル CLAUDE.md', exists: true},
  {kind: 'project', path: '/home/demo/app/CLAUDE.md', label: 'CLAUDE.md', exists: true},
  {kind: 'local', path: '/home/demo/app/CLAUDE.local.md', label: 'CLAUDE.local.md', exists: false},
  {kind: 'rule', path: '/home/demo/app/.claude/rules/go.md', label: '.claude/rules/go.md', exists: true},
]

export function mockApi(): Api {
  const files: Record<string, string> = {
    '/home/demo/.claude/CLAUDE.md': '# グローバル\n\n- 応答は日本語\n',
    '/home/demo/app/CLAUDE.md': '# app\n\n- テストは go test\n',
    '/home/demo/app/.claude/rules/go.md': '# Go\n\n- エラーはラップする\n',
  }
  const hashOf = (p: string) => (p in files ? 'h:' + files[p].length + ':' + files[p].slice(0, 8) : 'absent')
  let history: T.BackupEntry[] = []
  let cfg: T.Config = {claudePath: '', model: '', excludedProjects: [], correctionPatterns: ['違う', "don't"], maskPatterns: [], timeoutSeconds: 300}
  let cancelFn: (() => void) | null = null
  const delay = <V,>(v: V, ms = 0) => new Promise<V>((res) => setTimeout(() => res(v), ms))

  return {
    settings: () => delay(cfg),
    saveSettings: async (c) => {
      for (const m of c.maskPatterns) new RegExp(m.regex) // 不正な正規表現は例外
      cfg = c
    },
    reindex: () => delay({Scanned: 2, Ingested: 2, Skipped: 0, Removed: 0, BrokenLines: 1}),
    overview: () => delay({
      sessions: 2, messages: 11, inputTokens: 41, outputTokens: 76, corrections: 2, toolErrors: 1, memoryFiles: 2,
      daily: [{date: '2026-11-01', sessions: 1, messages: 7}, {date: '2026-11-02', sessions: 1, messages: 4}],
      models: [{model: 'claude-sonnet-4-5', messages: 4, inputTokens: 16, outputTokens: 26}, {model: 'claude-opus-4-1', messages: 2, inputTokens: 20, outputTokens: 40}],
      projects: [{project: '-home-demo-app', cwd: '/home/demo/app', sessions: 2, lastAt: '2026-11-02T09:00:00Z'}],
    }),
    listProjects: () => delay([{project: '-home-demo-app', cwd: '/home/demo/app', sessions: 2, lastAt: '2026-11-02T09:00:00Z'}]),
    listSessions: (p) => delay(sessions.filter((s) => !p || s.project === p)),
    getSession: async (id) => {
      const s = sessions.find((x) => x.id === id)
      if (!s) throw new Error('セッションが見つかりません: ' + id)
      return {session: s, messages: messages[id] ?? []}
    },
    search: async (q) => Object.values(messages).flat().filter((m) => q && m.text.includes(q)).map((m) => ({
      messageId: m.id, sessionId: m.sessionId, project: m.project, ts: m.ts, role: m.role, text: m.text})),
    listTargets: () => delay(targets),
    readTarget: async (p) => {
      const t = targets.find((x) => x.path === p)
      if (!t) throw new Error('許可されていないパスです')
      return {target: t, content: files[p] ?? '', hash: hashOf(p)}
    },
    buildDigest: (p, d) => delay(`# セッション digest\n\n- 対象: ${p || '全プロジェクト'}\n- 期間: 直近 ${d} 日\n\n## 統計\n\n| セッション数 | 2 |\n\n## 訂正発言の抜粋\n\n- 違う、そうじゃなくて先に go vet を実行して\n`),
    generateProposal: (req) => new Promise((resolve, reject) => {
      const t = setTimeout(() => {
        cancelFn = null
        const orig = files[req.targetPath] ?? ''
        resolve({targetPath: req.targetPath, baseHash: hashOf(req.targetPath), original: orig,
          output: orig + '- コミット前に go vet を先に実行する\n', rationale: '- 訂正発言「先に go vet」が繰り返されたため追記\n',
          digest: '(mock)', jobId: 'mock-job'})
      }, 300)
      cancelFn = () => { clearTimeout(t); cancelFn = null; reject(new Error('claude の実行がキャンセルされました')) }
    }),
    cancelGeneration: async () => { cancelFn?.() },
    applyProposal: async (p, c, h) => {
      if (!targets.some((t) => t.path === p)) throw new Error('書き込みが許可されていないパスです')
      if (hashOf(p) !== h) throw new Error('対象ファイルが改善案の生成後に変更されています。再生成してください')
      const before = files[p] ?? ''
      const m: T.BackupEntry = {id: `mock-${history.length + 1}`, path: p, op: 'apply', time: new Date().toISOString(),
        existed: p in files, oldHash: h, newHash: 'n', before, after: c}
      files[p] = c
      history = [m, ...history]
      return m
    },
    listBackups: () => delay(history.map((h) => ({...h, before: '', after: ''}))),
    getBackup: async (id) => {
      const h = history.find((x) => x.id === id)
      if (!h) throw new Error('履歴が見つかりません')
      return h
    },
    rollback: async (id) => {
      const h = history.find((x) => x.id === id)
      if (!h) throw new Error('履歴が見つかりません')
      if (h.existed) files[h.path] = h.before
      else delete files[h.path]
      const m: T.BackupEntry = {...h, id: `mock-${history.length + 1}`, op: 'rollback', rollbackOf: id, before: h.after, after: h.before}
      history = [m, ...history]
      return m
    },
    checkClaude: () => delay({found: false, path: '', error: 'claude コマンドが見つかりません。設定画面でパスを指定してください'}),
    openDataDir: async () => undefined,
    onEvent: () => () => undefined,
  }
}
