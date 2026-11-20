import {useEffect, useState} from 'react'
import Markdown from 'react-markdown'
import {api} from '../api'
import {Card, ErrorBox, LoadingButton} from '../components'
import {useAsync} from '../hooks'

export default function Reports() {
  const [ver, setVer] = useState(0)
  const list = useAsync(() => api.listReports(), [ver])
  const [selected, setSelected] = useState('')
  const [body, setBody] = useState('')
  const [plist, setPlist] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const fail = (e: unknown) => setError(e instanceof Error ? e.message : String(e))

  useEffect(() => {
    if (!selected) { setBody(''); return }
    let alive = true
    api.readReport(selected).then((b) => alive && setBody(b), (e) => alive && fail(e))
    return () => { alive = false }
  }, [selected])

  useEffect(() => {
    const offDone = api.onEvent('report:done', () => setVer((v) => v + 1))
    const offErr = api.onEvent('report:error', (m) => setError(`日報の自動生成に失敗しました: ${String(m)}`))
    return () => { offDone(); offErr() }
  }, [])

  async function generate() {
    setError(''); setNotice(''); setBusy(true)
    try {
      const r = await api.generateReport('')
      setNotice(`${r.date} の日報を保存しました`); setVer((v) => v + 1); setSelected(r.date)
      setBody(await api.readReport(r.date))
    } catch (e) { fail(e) } finally { setBusy(false) }
  }

  async function showPlist() {
    setError('')
    try { setPlist(await api.launchdPlist()) } catch (e) { fail(e) }
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">日次振り返り</h2>
      <p className="text-sm text-muted">
        当日のセッションから日報と TODO を生成し、アプリのデータディレクトリの reports/ に保存します（~/.claude やプロジェクトには書きません）。
        自動生成は設定画面で有効にします（アプリ起動中のみ。時刻を過ぎて起動した場合は未生成なら作成します）。
      </p>
      <ErrorBox message={error || list.error} />
      {notice && <p role="status" className="alert alert-success">{notice}</p>}
      <div className="flex gap-2">
        <LoadingButton className="btn btn-primary" busy={busy} busyLabel="生成中…" onClick={generate}>今日の日報を生成</LoadingButton>
        <button className="btn btn-secondary" onClick={showPlist}>launchd の plist を生成</button>
      </div>
      {plist && (
        <Card title="launchd の plist（表示のみ。登録・実行はしません）">
          <p className="mb-2 text-xs text-muted">
            使う場合は ~/Library/LaunchAgents/ に保存し、自分で launchctl load してください。指定時刻にアプリを起動します。
          </p>
          <textarea readOnly aria-label="plist" className="w-full font-mono text-xs" rows={14} value={plist} />
        </Card>
      )}
      <div className="grid gap-4 lg:grid-cols-[240px_1fr]">
        <Card title="日報一覧">
          <ul className="space-y-1" aria-label="日報一覧">
            {list.data?.map((r) => (
              <li key={r.date}>
                <button className="list-item" aria-current={selected === r.date ? true : undefined}
                  onClick={() => setSelected(r.date)}>{r.date}</button>
              </li>
            ))}
            {list.data?.length === 0 && <li className="text-sm text-muted">日報はまだありません。</li>}
          </ul>
        </Card>
        <Card title="内容">
          {selected ? <div className="prose max-w-none text-sm"><Markdown>{body}</Markdown></div>
            : <p className="text-sm text-muted">日報を選択してください。</p>}
        </Card>
      </div>
    </div>
  )
}
