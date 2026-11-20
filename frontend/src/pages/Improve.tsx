import {useMemo, useState} from 'react'
import Markdown from 'react-markdown'
import {api} from '../api'
import {Busy, Card, ErrorBox, Loading, LoadingButton} from '../components'
import DiffView from '../DiffView'
import {isUnder} from '../paths'
import {useAsync} from '../hooks'
import type {Manifest, Proposal, Target, TargetContent} from '../types'

type Phase = 'select' | 'generating' | 'review' | 'applied'

export default function Improve() {
  const targets = useAsync(() => api.listTargets(), [])
  const projects = useAsync(() => api.listProjects(), [])
  const auto = useAsync(() => api.autoSuggestion(), [])
  const [autoHidden, setAutoHidden] = useState(false)
  const [path, setPath] = useState('')
  const [days, setDays] = useState(30)
  const [instruction, setInstruction] = useState('')
  const [current, setCurrent] = useState<TargetContent | null>(null)
  const [digest, setDigest] = useState('')
  const [phase, setPhase] = useState<Phase>('select')
  const [proposal, setProposal] = useState<Proposal | null>(null)
  const [edited, setEdited] = useState('')
  const [editing, setEditing] = useState(false)
  const [error, setError] = useState('')
  const [applied, setApplied] = useState<Manifest | null>(null)
  const [digestBusy, setDigestBusy] = useState(false)
  const [applyBusy, setApplyBusy] = useState(false)
  const [dismissBusy, setDismissBusy] = useState(false)

  const target: Target | undefined = targets.data?.find((t) => t.path === path)
  const project = useMemo(() => {
    if (!target || target.kind === 'global') return ''
    const hit = projects.data?.find((p) => isUnder(target.path, p.cwd))
    return hit?.project ?? ''
  }, [target, projects.data])

  const fail = (e: unknown) => setError(e instanceof Error ? e.message : String(e))

  async function selectTarget(p: string) {
    setPath(p); setDigest(''); setError(''); setProposal(null); setPhase('select'); setCurrent(null)
    if (!p) return
    try { setCurrent(await api.readTarget(p)) } catch (e) { fail(e) }
  }

  async function previewDigest() {
    setError(''); setDigestBusy(true)
    try { setDigest(await api.buildDigest(project, days)) } catch (e) { fail(e) } finally { setDigestBusy(false) }
  }

  async function generate() {
    setError(''); setPhase('generating')
    try {
      const p = await api.generateProposal({targetPath: path, project, days, instruction})
      setProposal(p); setEdited(p.output); setEditing(false); setPhase('review')
    } catch (e) {
      fail(e); setPhase('select')
    }
  }

  async function apply() {
    if (!proposal) return
    setError(''); setApplyBusy(true)
    try {
      setApplied(await api.applyProposal(proposal.targetPath, edited, proposal.baseHash))
      if (auto.data && auto.data.jobId === proposal.jobId) await api.dismissAutoSuggestion()
      setPhase('applied')
    } catch (e) { fail(e) } finally { setApplyBusy(false) }
  }

  /** 自動生成された提案を確認画面に開く（適用は差分を見て人が押す）。 */
  function openAuto(p: Proposal) {
    setPath(p.targetPath); setProposal(p); setEdited(p.output); setEditing(false); setError(''); setPhase('review'); setAutoHidden(true)
  }

  async function dismissAuto() {
    setDismissBusy(true)
    try { await api.dismissAutoSuggestion(); setAutoHidden(true) } catch (e) { fail(e) } finally { setDismissBusy(false) }
  }

  function discard() {
    setProposal(null); setEditing(false); setPhase('select'); setError('')
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">CLAUDE.md 改善</h2>
      <p className="text-sm text-gray-600">AI の出力は自動では保存されません。差分を確認して「適用」を押したときだけ書き込まれます（適用前に自動でバックアップされます）。</p>
      <ErrorBox message={error || targets.error} />

      {auto.data && !autoHidden && phase === 'select' && (
        <Card title="自動生成された提案があります">
          <p className="text-sm">設定で有効にした自動生成により、{auto.data.targetPath} の改善案が作られています。まだ何も書き込まれていません。差分を確認して、よければ「適用」を押してください。</p>
          <div className="mt-3 flex gap-2">
            <button className="rounded bg-blue-600 px-3 py-1 text-sm text-white" onClick={() => openAuto(auto.data as Proposal)}>差分を確認する</button>
            <LoadingButton className="rounded border px-3 py-1 text-sm disabled:opacity-50" busy={dismissBusy} onClick={() => void dismissAuto()}>破棄</LoadingButton>
          </div>
        </Card>
      )}

      {(phase === 'select' || phase === 'generating') && (
        <Card title="① 対象と条件">
          <div className="grid gap-3 md:grid-cols-2">
            <label className="text-sm">対象ファイル
              <select className="mt-1 w-full rounded border px-2 py-1" value={path} onChange={(e) => void selectTarget(e.target.value)} aria-label="対象ファイル" disabled={phase === 'generating'}>
                <option value="">選択してください</option>
                {targets.data?.map((t) => <option key={t.path} value={t.path}>{t.label}（{t.path}）{t.exists ? '' : ' ※未作成'}</option>)}
              </select>
            </label>
            <label className="text-sm">対象期間
              <select className="mt-1 w-full rounded border px-2 py-1" value={days} onChange={(e) => setDays(Number(e.target.value))} aria-label="対象期間" disabled={phase === 'generating'}>
                {[7, 30, 90].map((d) => <option key={d} value={d}>直近 {d} 日</option>)}
              </select>
            </label>
          </div>
          <label className="mt-3 block text-sm">追加指示（任意）
            <textarea className="mt-1 w-full rounded border p-2" rows={3} value={instruction} onChange={(e) => setInstruction(e.target.value)} aria-label="追加指示" disabled={phase === 'generating'} />
          </label>
          {current && <p className="mt-2 text-xs text-gray-500">現在の内容：{current.content.split('\n').length} 行</p>}
          <div className="mt-3 flex gap-2">
            <LoadingButton className="rounded border px-3 py-1 text-sm disabled:opacity-50" busy={digestBusy} busyLabel="digest を作成中…" disabled={!path || phase === 'generating'} onClick={() => void previewDigest()}>② digest をプレビュー</LoadingButton>
            <LoadingButton className="rounded bg-blue-600 px-3 py-1 text-sm text-white disabled:opacity-50" busy={phase === 'generating'} busyLabel="生成中…" disabled={!path || digestBusy} onClick={() => void generate()}>④ 改善案を生成</LoadingButton>
            {phase === 'generating' && <button className="rounded border border-red-400 px-3 py-1 text-sm text-red-700" onClick={() => void api.cancelGeneration()}>キャンセル</button>}
          </div>
          {phase === 'generating' && <Busy label="生成中…（claude を実行しています）" />}
        </Card>
      )}

      {digest && (phase === 'select' || phase === 'generating') && (
        <Card title="③ digest プレビュー（マスク済み・AI に渡す内容）">
          <pre className="max-h-80 overflow-auto whitespace-pre-wrap text-xs" data-testid="digest">{digest}</pre>
        </Card>
      )}

      {phase === 'review' && proposal && (
        <>
          <Card title="変更理由">
            {proposal.rationale ? <div className="prose prose-sm max-w-none text-sm"><Markdown>{proposal.rationale}</Markdown></div> : <p className="text-sm text-gray-500">理由は出力されませんでした。</p>}
          </Card>
          <Card title="⑤ 差分（右側は編集できます）">
            <DiffView original={proposal.original} proposed={edited} editing={editing} onChange={setEdited} />
            <div className="mt-3 flex gap-2">
              <button className="rounded border px-3 py-1 text-sm" onClick={() => setEditing((v) => !v)}>{editing ? '編集を終えて差分を見る' : '右側を編集'}</button>
              {edited !== proposal.output && <button className="rounded border px-3 py-1 text-sm" onClick={() => setEdited(proposal.output)}>AI の出力に戻す</button>}
              <span className="flex-1" />
              <button className="rounded border px-3 py-1 text-sm disabled:opacity-50" disabled={applyBusy} onClick={discard}>破棄</button>
              <LoadingButton className="rounded bg-green-600 px-3 py-1 text-sm text-white disabled:opacity-50" busy={applyBusy} busyLabel="適用中…" disabled={edited === proposal.original} onClick={() => void apply()}>適用</LoadingButton>
            </div>
          </Card>
        </>
      )}

      {phase === 'applied' && applied && (
        <Card title="適用しました">
          <p className="text-sm" role="status">{applied.path} に適用しました。バックアップ ID: <code>{applied.id}</code>（「変更履歴」から元に戻せます）</p>
          <button className="mt-3 rounded border px-3 py-1 text-sm" onClick={() => void selectTarget(path)}>続けて改善する</button>
        </Card>
      )}
      {targets.loading && !targets.data && <Loading />}
    </div>
  )
}
