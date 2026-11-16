import {useState} from 'react'
import {api} from '../api'
import {Card, ErrorBox, Loading} from '../components'
import DiffView from '../DiffView'
import {useAsync} from '../hooks'

const opLabel: Record<string, string> = {apply: '適用', rollback: 'ロールバック'}

function Detail({id, onRollback}: {id: string; onRollback: () => void}) {
  const {data, error, loading} = useAsync(() => api.getBackup(id), [id])
  if (loading && !data) return <Loading />
  if (error) return <ErrorBox message={error} />
  if (!data) return null
  return (
    <div className="space-y-3">
      <div className="text-sm text-gray-600">
        <div>{data.path}</div>
        <div>{opLabel[data.op] ?? data.op}（{data.time.replace('T', ' ').replace('Z', '')}）{!data.existed && ' ※新規作成'}</div>
      </div>
      <DiffView original={data.before} proposed={data.after} editing={false} onChange={() => undefined} />
      {data.op === 'apply' && (
        <button className="rounded border border-red-400 px-3 py-1 text-sm text-red-700" onClick={onRollback}>この変更をロールバック</button>
      )}
    </div>
  )
}

export default function History() {
  const [ver, setVer] = useState(0)
  const list = useAsync(() => api.listBackups(), [ver])
  const [selected, setSelected] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function rollback() {
    setConfirming(false); setError('')
    try {
      const m = await api.rollback(selected)
      setNotice(`ロールバックしました（バックアップ ID: ${m.id}）`)
      setSelected(m.id); setVer((v) => v + 1)
    } catch (e) { setError(e instanceof Error ? e.message : String(e)) }
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">変更履歴</h2>
      <p className="text-sm text-gray-600">アプリが書き込んだ全ファイルのバックアップです。ロールバックも履歴に記録されます。</p>
      <ErrorBox message={error || list.error} />
      {notice && <p role="status" className="rounded border border-green-300 bg-green-50 p-2 text-sm">{notice}</p>}
      <div className="grid gap-4 lg:grid-cols-[360px_1fr]">
        <Card title="履歴">
          <ul className="space-y-2" aria-label="履歴一覧">
            {list.data?.map((h) => (
              <li key={h.id}>
                <button className={`w-full rounded border p-2 text-left text-sm hover:bg-gray-50 ${selected === h.id ? 'border-blue-500' : ''}`} onClick={() => { setSelected(h.id); setNotice('') }}>
                  <div className="text-xs text-gray-500">{h.time.replace('T', ' ').replace('Z', '')} / {opLabel[h.op] ?? h.op}</div>
                  <div className="break-all">{h.path}</div>
                </button>
              </li>
            ))}
            {list.data?.length === 0 && <li className="text-sm text-gray-500">履歴はまだありません。</li>}
          </ul>
        </Card>
        <Card title="差分">
          {selected ? <Detail key={selected} id={selected} onRollback={() => setConfirming(true)} /> : <p className="text-sm text-gray-500">履歴を選択してください。</p>}
        </Card>
      </div>
      {confirming && (
        <div className="fixed inset-0 flex items-center justify-center bg-black/40" role="presentation">
          <div role="dialog" aria-modal="true" aria-label="ロールバックの確認" className="w-96 rounded-lg bg-white p-5 shadow-xl">
            <p className="mb-4 text-sm">この変更を元に戻します。現在の内容もバックアップされます。よろしいですか？</p>
            <div className="flex justify-end gap-2">
              <button className="rounded border px-3 py-1 text-sm" onClick={() => setConfirming(false)}>キャンセル</button>
              <button className="rounded bg-red-600 px-3 py-1 text-sm text-white" onClick={() => void rollback()}>ロールバックする</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
