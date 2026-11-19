import {useState} from 'react'
import Markdown from 'react-markdown'
import {api} from '../api'
import {Card, ErrorBox} from '../components'
import DiffView from '../DiffView'
import {useAsync} from '../hooks'
import type {Manifest, SkillProposal} from '../types'

type Phase = 'select' | 'generating' | 'review' | 'applied'

const base = (p: string) => p.split(/[\\/]/).pop() ?? p

export default function Skills() {
  const memories = useAsync(() => api.listMemories(), [])
  const [picked, setPicked] = useState<string[]>([])
  const [name, setName] = useState('')
  const [days, setDays] = useState(0)
  const [instruction, setInstruction] = useState('')
  const [phase, setPhase] = useState<Phase>('select')
  const [proposal, setProposal] = useState<SkillProposal | null>(null)
  const [edited, setEdited] = useState<Record<string, string>>({})
  const [editing, setEditing] = useState<string>('')
  const [applied, setApplied] = useState<Manifest[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const fail = (e: unknown) => setError(e instanceof Error ? e.message : String(e))
  const toggle = (p: string) => setPicked((cur) => (cur.includes(p) ? cur.filter((x) => x !== p) : [...cur, p]))

  async function generate() {
    setError(''); setPhase('generating')
    try {
      const p = await api.generateSkill({name, memoryPaths: picked, project: '', days, instruction})
      setProposal(p); setEdited(Object.fromEntries(p.files.map((f) => [f.path, f.content]))); setEditing(''); setPhase('review')
    } catch (e) { fail(e); setPhase('select') }
  }

  async function apply() {
    if (!proposal) return
    setError(''); setBusy(true)
    try {
      setApplied(await api.applySkill(proposal.name, proposal.files.map((f) => ({path: f.path, content: edited[f.path], baseHash: f.baseHash}))))
      setPhase('applied')
    } catch (e) { fail(e) } finally { setBusy(false) }
  }

  const discard = () => { setProposal(null); setEditing(''); setPhase('select'); setError('') }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">自己ナレッジ生成</h2>
      <p className="text-sm text-gray-600">メモリから skill セット（SKILL.md と knowledge）を AI で生成します。自動では保存されません。差分を確認して「適用」を押したときだけ <code>~/.claude/skills/&lt;名前&gt;/</code> に書き込まれます（適用前に自動でバックアップされます）。</p>
      <ErrorBox message={error || memories.error} />

      {(phase === 'select' || phase === 'generating') && (
        <Card title="① メモリと条件">
          <fieldset disabled={phase === 'generating'}>
            <legend className="mb-1 text-sm">元にするメモリ（複数選択可）</legend>
            <ul className="max-h-48 space-y-1 overflow-auto" aria-label="メモリ選択">
              {memories.data?.map((m) => (
                <li key={m.target.path}>
                  <label className="text-sm">
                    <input type="checkbox" className="mr-2" checked={picked.includes(m.target.path)} onChange={() => toggle(m.target.path)} />
                    {m.project} / {base(m.target.path)}
                  </label>
                </li>
              ))}
              {memories.data?.length === 0 && <li className="text-sm text-gray-500">メモリはまだありません。</li>}
            </ul>
            <div className="mt-3 grid gap-3 md:grid-cols-2">
              <label className="text-sm">skill 名（小文字・数字・ハイフン）
                <input className="mt-1 w-full rounded border px-2 py-1" value={name} onChange={(e) => setName(e.target.value)} aria-label="skill 名" />
              </label>
              <label className="text-sm">セッション digest の併用
                <select className="mt-1 w-full rounded border px-2 py-1" value={days} onChange={(e) => setDays(Number(e.target.value))} aria-label="digest の期間">
                  <option value={0}>使わない</option>
                  {[7, 30, 90].map((d) => <option key={d} value={d}>直近 {d} 日</option>)}
                </select>
              </label>
            </div>
            <label className="mt-3 block text-sm">追加指示（任意）
              <textarea className="mt-1 w-full rounded border p-2" rows={3} value={instruction} onChange={(e) => setInstruction(e.target.value)} aria-label="追加指示" />
            </label>
          </fieldset>
          <div className="mt-3 flex gap-2">
            <button className="rounded bg-blue-600 px-3 py-1 text-sm text-white disabled:opacity-50" disabled={!name || picked.length === 0 || phase === 'generating'} onClick={() => void generate()}>生成</button>
            {phase === 'generating' && <button className="rounded border border-red-400 px-3 py-1 text-sm text-red-700" onClick={() => void api.cancelGeneration()}>キャンセル</button>}
          </div>
          {phase === 'generating' && <p className="mt-2 text-sm text-gray-600" role="status">生成中…（claude を実行しています）</p>}
        </Card>
      )}

      {phase === 'review' && proposal && (
        <>
          <Card title="作成理由">
            {proposal.rationale ? <div className="prose prose-sm max-w-none text-sm"><Markdown>{proposal.rationale}</Markdown></div> : <p className="text-sm text-gray-500">理由は出力されませんでした。</p>}
          </Card>
          <p className="break-all text-sm text-gray-600">適用先: {proposal.dir}</p>
          {proposal.files.map((f) => (
            <Card key={f.path} title={`${f.path}${f.original ? '（既存を更新）' : '（新規）'}`}>
              <DiffView original={f.original} proposed={edited[f.path]} editing={editing === f.path} onChange={(v) => setEdited((cur) => ({...cur, [f.path]: v}))} />
              <div className="mt-2 flex gap-2">
                <button className="rounded border px-3 py-1 text-sm" onClick={() => setEditing(editing === f.path ? '' : f.path)}>{editing === f.path ? '編集を終えて差分を見る' : '右側を編集'}</button>
                {edited[f.path] !== f.content && <button className="rounded border px-3 py-1 text-sm" onClick={() => setEdited((cur) => ({...cur, [f.path]: f.content}))}>AI の出力に戻す</button>}
              </div>
            </Card>
          ))}
          <div className="flex gap-2">
            <span className="flex-1" />
            <button className="rounded border px-3 py-1 text-sm" onClick={discard}>破棄</button>
            <button className="rounded bg-green-600 px-3 py-1 text-sm text-white disabled:opacity-50" disabled={busy} onClick={() => void apply()}>適用</button>
          </div>
        </>
      )}

      {phase === 'applied' && (
        <Card title="適用しました">
          <p className="text-sm" role="status">{applied.length} ファイルを適用しました。バックアップ ID: <code>{applied.map((m) => m.id).join(', ')}</code>（「変更履歴」から 1 ファイルずつ元に戻せます）</p>
          <button className="mt-3 rounded border px-3 py-1 text-sm" onClick={discard}>続けて生成する</button>
        </Card>
      )}
    </div>
  )
}
