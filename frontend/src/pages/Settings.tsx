import {useEffect, useState} from 'react'
import {api} from '../api'
import {Card, ErrorBox, Loading} from '../components'
import {useAsync} from '../hooks'
import type {Config, MaskPattern} from '../types'

const lines = (s: string) => s.split('\n').map((x) => x.trim()).filter(Boolean)

/** 不正な正規表現のメッセージを返す（問題なければ空文字）。 */
export function regexError(p: MaskPattern): string {
  try { new RegExp(p.regex); return '' } catch (e) { return e instanceof Error ? e.message : '不正な正規表現' }
}

export default function Settings() {
  const loaded = useAsync(() => api.settings(), [])
  const claude = useAsync(() => api.checkClaude(), [loaded.data?.claudePath])
  const [cfg, setCfg] = useState<Config | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  useEffect(() => { if (loaded.data) setCfg(loaded.data) }, [loaded.data])

  if (loaded.error) return <ErrorBox message={loaded.error} />
  if (!cfg) return <Loading />

  const set = <K extends keyof Config>(k: K, v: Config[K]) => setCfg({...cfg, [k]: v})
  const badRegex = cfg.maskPatterns.some((p) => regexError(p))

  async function save() {
    setError(''); setNotice('')
    try { await api.saveSettings(cfg as Config); setNotice('保存しました'); } catch (e) { setError(e instanceof Error ? e.message : String(e)) }
  }
  async function rebuild() {
    setError(''); setNotice('')
    try {
      const s = await api.reindex(true)
      setNotice(`インデックスを再構築しました（取り込み ${s.Ingested} 件、壊れた行 ${s.BrokenLines}）`)
    } catch (e) { setError(e instanceof Error ? e.message : String(e)) }
  }

  return (
    <div className="max-w-3xl space-y-4">
      <h2 className="text-xl font-bold">設定</h2>
      <ErrorBox message={error} />
      {notice && <p role="status" className="rounded border border-green-300 bg-green-50 p-2 text-sm">{notice}</p>}

      <Card title="claude コマンド">
        {claude.data && !claude.data.found && (
          <div role="alert" className="mb-3 rounded border border-amber-300 bg-amber-50 p-3 text-sm">
            claude コマンドが見つかりません。Claude Code をインストールするか、下の欄にフルパスを入力して保存してください
            （例: <code>~/.local/bin/claude</code> や <code>/opt/homebrew/bin/claude</code>。ターミナルで <code>which claude</code> を実行すると分かります）。
          </div>
        )}
        {claude.data?.found && <p className="mb-2 text-xs text-green-700">検出済み: {claude.data.path}</p>}
        <label className="block text-sm">パス（空欄なら自動検出）
          <input className="mt-1 w-full rounded border px-2 py-1" aria-label="claude のパス" value={cfg.claudePath} onChange={(e) => set('claudePath', e.target.value)} />
        </label>
        <label className="mt-3 block text-sm">使用モデル（空欄なら既定）
          <input className="mt-1 w-full rounded border px-2 py-1" aria-label="使用モデル" value={cfg.model} onChange={(e) => set('model', e.target.value)} />
        </label>
        <label className="mt-3 block text-sm">タイムアウト（秒）
          <input type="number" min={10} className="mt-1 w-32 rounded border px-2 py-1" aria-label="タイムアウト" value={cfg.timeoutSeconds} onChange={(e) => set('timeoutSeconds', Number(e.target.value))} />
        </label>
      </Card>

      <Card title="日次振り返り">
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" aria-label="日報を自動生成する" checked={cfg.dailyReportEnabled} onChange={(e) => set('dailyReportEnabled', e.target.checked)} />
          アプリ起動中、指定時刻に当日の日報を自動生成する
        </label>
        <label className="mt-3 block text-sm">生成時刻（HH:MM。空欄なら 21:00）
          <input className="mt-1 w-32 rounded border px-2 py-1" aria-label="日報の生成時刻" placeholder="21:00" value={cfg.dailyReportTime} onChange={(e) => set('dailyReportTime', e.target.value)} />
        </label>
      </Card>

      <Card title="取り込み">
        <label className="block text-sm">除外するプロジェクト（プロジェクトディレクトリ名または cwd。1 行 1 件）
          <textarea className="mt-1 w-full rounded border p-2 font-mono text-xs" rows={3} aria-label="除外するプロジェクト" value={cfg.excludedProjects.join('\n')} onChange={(e) => set('excludedProjects', lines(e.target.value))} />
        </label>
        <label className="mt-3 block text-sm">訂正パターン（部分一致・1 行 1 件）
          <textarea className="mt-1 w-full rounded border p-2 font-mono text-xs" rows={5} aria-label="訂正パターン" value={cfg.correctionPatterns.join('\n')} onChange={(e) => set('correctionPatterns', lines(e.target.value))} />
        </label>
        <p className="mt-1 text-xs text-gray-500">変更後に「インデックスを再構築」を実行すると反映されます。</p>
      </Card>

      <Card title="マスキングの追加パターン（正規表現）">
        <ul className="space-y-2">
          {cfg.maskPatterns.map((p, i) => {
            const err = regexError(p)
            return (
              <li key={i} className="flex flex-wrap items-center gap-2">
                <input className="w-32 rounded border px-2 py-1 text-sm" aria-label={`パターン名 ${i + 1}`} placeholder="名前" value={p.name}
                  onChange={(e) => set('maskPatterns', cfg.maskPatterns.map((x, j) => (j === i ? {...x, name: e.target.value} : x)))} />
                <input className="flex-1 rounded border px-2 py-1 font-mono text-sm" aria-label={`正規表現 ${i + 1}`} placeholder="正規表現" value={p.regex}
                  onChange={(e) => set('maskPatterns', cfg.maskPatterns.map((x, j) => (j === i ? {...x, regex: e.target.value} : x)))} />
                <button className="rounded border px-2 py-1 text-sm" onClick={() => set('maskPatterns', cfg.maskPatterns.filter((_, j) => j !== i))}>削除</button>
                {err && <span className="w-full text-xs text-red-700">正規表現が不正です: {err}</span>}
              </li>
            )
          })}
        </ul>
        <button className="mt-2 rounded border px-3 py-1 text-sm" onClick={() => set('maskPatterns', [...cfg.maskPatterns, {name: '', regex: ''}])}>パターンを追加</button>
      </Card>

      <div className="flex flex-wrap gap-2">
        <button className="rounded bg-blue-600 px-4 py-1.5 text-sm text-white disabled:opacity-50" disabled={badRegex} onClick={() => void save()}>保存</button>
        <button className="rounded border px-4 py-1.5 text-sm" onClick={() => void rebuild()}>インデックスを再構築</button>
        <button className="rounded border px-4 py-1.5 text-sm" onClick={() => void api.openDataDir()}>データディレクトリを開く</button>
      </div>
    </div>
  )
}
