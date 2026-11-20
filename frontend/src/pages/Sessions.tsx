import {FormEvent, useState} from 'react'
import {api} from '../api'
import {Card, ErrorBox, fmt, Loading, shortProject} from '../components'
import {useAsync, useIndexVersion} from '../hooks'
import type {Message} from '../types'

const periods = [{v: 0, l: '全期間'}, {v: 7, l: '直近 7 日'}, {v: 30, l: '直近 30 日'}, {v: 90, l: '直近 90 日'}]

export function MessageRow({m}: {m: Message}) {
  const tone = m.isError ? 'border-danger-line bg-danger-soft' : m.isCorrection ? 'border-warning-line bg-warning-soft'
    : m.role === 'user' ? 'border-accent/40 bg-accent-soft' : 'border-line bg-surface'
  const kind = m.kind === 'text' ? (m.role === 'user' ? 'ユーザー' : 'アシスタント') : m.kind === 'tool_use' ? `ツール呼び出し (${m.tool})` : `ツール結果 (${m.tool || '?'})`
  return (
    <li className={`rounded border-l-4 p-2 text-sm ${tone}`} data-testid="message">
      <div className="mb-1 flex gap-2 text-xs text-muted">
        <span>{m.ts.replace('T', ' ').replace('Z', '')}</span><span>{kind}</span>
        {m.isCorrection && <span className="font-semibold text-warning-fg">訂正</span>}
        {m.isError && <span className="font-semibold text-danger-fg">エラー</span>}
      </div>
      <pre className="whitespace-pre-wrap break-words font-sans">{m.text}</pre>
    </li>
  )
}

function Detail({id}: {id: string}) {
  const {data, error, loading} = useAsync(() => api.getSession(id), [id])
  if (loading && !data) return <Loading />
  if (error) return <ErrorBox message={error} />
  if (!data) return null
  const s = data.session
  return (
    <div className="space-y-3">
      <div className="text-sm text-muted">
        <div>{s.id}（{s.gitBranch || 'ブランチ不明'}）</div>
        <div>{s.cwd}</div>
        <div>訂正 {s.corrections} / エラー {s.toolErrors} / トークン {fmt(s.inputTokens + s.outputTokens)}</div>
      </div>
      <ul className="space-y-2">{data.messages.map((m) => <MessageRow key={m.id} m={m} />)}</ul>
    </div>
  )
}

export default function Sessions() {
  const [project, setProject] = useState('')
  const [days, setDays] = useState(0)
  const [query, setQuery] = useState('')
  const [submitted, setSubmitted] = useState('')
  const [selected, setSelected] = useState('')
  const ver = useIndexVersion()
  const projects = useAsync(() => api.listProjects(), [ver])
  const list = useAsync(() => api.listSessions(project, days, 200), [project, days, ver])
  const hits = useAsync(() => (submitted ? api.search(submitted) : Promise.resolve([])), [submitted, ver])

  const onSearch = (e: FormEvent) => { e.preventDefault(); setSubmitted(query.trim()) }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">セッション</h2>
      <div className="flex flex-wrap items-end gap-3">
        <label className="text-sm">プロジェクト
          <select className="ml-1" value={project} onChange={(e) => setProject(e.target.value)} aria-label="プロジェクト">
            <option value="">すべて</option>
            {projects.data?.map((p) => <option key={p.project} value={p.project}>{shortProject(p.project)}</option>)}
          </select>
        </label>
        <label className="text-sm">期間
          <select className="ml-1" value={days} onChange={(e) => setDays(Number(e.target.value))} aria-label="期間">
            {periods.map((p) => <option key={p.v} value={p.v}>{p.l}</option>)}
          </select>
        </label>
        <form onSubmit={onSearch} className="flex gap-2">
          <input placeholder="全文検索" aria-label="全文検索" value={query} onChange={(e) => setQuery(e.target.value)} />
          <button className="btn btn-primary" type="submit">検索</button>
          {submitted && <button type="button" className="btn btn-secondary" onClick={() => { setQuery(''); setSubmitted('') }}>クリア</button>}
        </form>
      </div>
      <ErrorBox message={list.error || hits.error} />
      <div className="grid gap-4 lg:grid-cols-[360px_1fr]">
        <Card title={submitted ? `検索結果「${submitted}」` : 'セッション一覧'}>
          {submitted ? (
            <ul className="space-y-2" aria-label="検索結果">
              {hits.data?.map((h) => (
                <li key={h.messageId}>
                  <button className="list-item" onClick={() => setSelected(h.sessionId)}>
                    <div className="text-xs text-muted">{h.ts.slice(0, 10)} {h.role} {h.sessionId}</div>
                    <div className="line-clamp-2">{h.text}</div>
                  </button>
                </li>
              ))}
              {hits.data?.length === 0 && <li className="text-sm text-muted">該当なし</li>}
            </ul>
          ) : (
            <ul className="space-y-2" aria-label="セッション一覧">
              {list.data?.map((s) => (
                <li key={s.id}>
                  <button className="list-item" aria-current={selected === s.id ? true : undefined} onClick={() => setSelected(s.id)}>
                    <div className="font-medium">{s.startedAt.slice(0, 16).replace('T', ' ')}</div>
                    <div className="text-xs text-muted">{shortProject(s.project)} / {s.gitBranch}</div>
                    <div className="text-xs">
                      発言 {s.userMsgs + s.assistantMsgs}
                      {s.corrections > 0 && <span className="ml-2 text-warning-fg">訂正 {s.corrections}</span>}
                      {s.toolErrors > 0 && <span className="ml-2 text-danger-fg">エラー {s.toolErrors}</span>}
                    </div>
                  </button>
                </li>
              ))}
              {list.data?.length === 0 && <li className="text-sm text-muted">セッションがありません。</li>}
            </ul>
          )}
        </Card>
        <Card title="セッション詳細">
          {selected ? <Detail id={selected} /> : <p className="text-sm text-muted">左の一覧から選択してください。</p>}
        </Card>
      </div>
    </div>
  )
}
