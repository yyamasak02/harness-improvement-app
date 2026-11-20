import {useState} from 'react'
import {api} from '../api'
import {Card, ErrorBox, Loading, LoadingButton} from '../components'
import DiffView from '../DiffView'
import {useAsync, useIndexVersion} from '../hooks'
import type {TargetContent} from '../types'

const base = (p: string) => p.split(/[\\/]/).pop() ?? p

function Editor({path, onApplied}: {path: string; onApplied: (id: string) => void}) {
  const [ver, setVer] = useState(0)
  const cur = useAsync(() => api.readTarget(path), [path, ver])
  return cur.data ? <Inner key={cur.data.hash} cur={cur.data} onApplied={(id) => { onApplied(id); setVer((v) => v + 1) }} />
    : cur.error ? <ErrorBox message={cur.error} /> : <Loading />
}

function Inner({cur, onApplied}: {cur: TargetContent; onApplied: (id: string) => void}) {
  const [text, setText] = useState(cur.content)
  const [editing, setEditing] = useState(false)
  const [error, setError] = useState('')
  const [applying, setApplying] = useState(false)
  const changed = text !== cur.content

  async function apply() {
    setError(''); setApplying(true)
    try {
      const m = await api.applyProposal(cur.target.path, text, cur.hash)
      onApplied(m.id)
    } catch (e) { setError(e instanceof Error ? e.message : String(e)); setApplying(false) }
  }

  return (
    <div className="space-y-3">
      <ErrorBox message={error} />
      <div className="break-all text-sm text-gray-600">{cur.target.path}</div>
      {editing ? (
        <>
          <DiffView original={cur.content} proposed={text} editing onChange={setText} />
          <div className="flex gap-2">
            <LoadingButton className="rounded bg-blue-600 px-3 py-1 text-sm text-white disabled:opacity-40" busy={applying} busyLabel="適用中…" disabled={!changed} onClick={() => void apply()}>適用</LoadingButton>
            <button className="rounded border px-3 py-1 text-sm disabled:opacity-50" disabled={applying} onClick={() => { setText(cur.content); setEditing(false) }}>破棄</button>
          </div>
          <p className="text-xs text-gray-500">適用すると変更前の内容がバックアップされ、変更履歴からロールバックできます。</p>
        </>
      ) : (
        <>
          <pre data-testid="memory-body" className="max-h-96 overflow-auto whitespace-pre-wrap rounded border bg-gray-50 p-3 text-sm">{cur.content}</pre>
          <button className="rounded border px-3 py-1 text-sm" onClick={() => setEditing(true)}>編集</button>
        </>
      )}
    </div>
  )
}

export default function Memory() {
  const idxVer = useIndexVersion()
  const [ver, setVer] = useState(0)
  const list = useAsync(() => api.listMemories(), [idxVer, ver])
  const [selected, setSelected] = useState('')
  const [notice, setNotice] = useState('')

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">メモリ</h2>
      <p className="text-sm text-gray-600">プロジェクトメモリの閲覧と編集です。編集は差分を確認してから適用します。重複・古いメモリには印が付きます。</p>
      <ErrorBox message={list.error} />
      {notice && <p role="status" className="rounded border border-green-300 bg-green-50 p-2 text-sm">{notice}</p>}
      <div className="grid gap-4 lg:grid-cols-[360px_1fr]">
        <Card title="メモリ一覧">
          <ul className="space-y-2" aria-label="メモリ一覧">
            {list.data?.map((m) => (
              <li key={m.target.path}>
                <button className={`w-full rounded border p-2 text-left text-sm hover:bg-gray-50 ${selected === m.target.path ? 'border-blue-500' : ''}`}
                  onClick={() => { setSelected(m.target.path); setNotice('') }}>
                  <div className="text-xs text-gray-500">{m.project}</div>
                  <div className="break-all">{base(m.target.path)}</div>
                  <div className="mt-1 flex gap-1 text-xs">
                    {m.stale && <span className="rounded bg-yellow-100 px-1">古い</span>}
                    {m.duplicateOf.length > 0 && (
                      <span className="rounded bg-orange-100 px-1" title={m.duplicateOf.join('\n')}>重複: {m.duplicateOf.map(base).join(', ')}</span>
                    )}
                  </div>
                </button>
              </li>
            ))}
            {list.data?.length === 0 && <li className="text-sm text-gray-500">メモリはまだありません。</li>}
          </ul>
        </Card>
        <Card title="内容">
          {selected ? <Editor key={selected} path={selected} onApplied={(id) => { setNotice(`適用しました（バックアップ ID: ${id}）`); setVer((v) => v + 1) }} />
            : <p className="text-sm text-gray-500">メモリを選択してください。</p>}
        </Card>
      </div>
    </div>
  )
}
