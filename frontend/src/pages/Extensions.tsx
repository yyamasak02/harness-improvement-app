import type {ReactNode} from 'react'
import {api} from '../api'
import {Card, ErrorBox, Loading} from '../components'
import {useAsync} from '../hooks'

function Empty({n, children}: {n: number; children: ReactNode}) {
  return n === 0 ? <p className="text-sm text-muted">{children}</p> : null
}

export default function Extensions() {
  const r = useAsync(() => api.listExtensions(), [])
  if (r.error) return <ErrorBox message={r.error} />
  if (!r.data) return <Loading />
  const {mcp, plugins, skills, hooks, warnings} = {
    mcp: r.data.mcp ?? [], plugins: r.data.plugins ?? [], skills: r.data.skills ?? [],
    hooks: r.data.hooks ?? [], warnings: r.data.warnings ?? [],
  }
  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">拡張機能</h2>
      <p className="text-xs text-muted">読み取り専用の一覧です。変更はこのアプリからは行いません（env・ヘッダーの値は読み込みません）。</p>
      {warnings.length > 0 && <ErrorBox message={warnings.join(' / ')} />}
      <Card title={`MCP サーバー（${mcp.length}）`}>
        <Empty n={mcp.length}>MCP サーバーは設定されていません。</Empty>
        <ul aria-label="MCP サーバー" className="space-y-1 text-sm">
          {mcp.map((m) => (
            <li key={`${m.scope}:${m.project}:${m.name}`}>
              <span className="font-medium">{m.name}</span>{' '}
              <span className="text-xs text-muted">{m.type} / {m.scope}{m.project ? `（${m.project}）` : ''}</span>
              <div className="break-all font-mono text-xs text-muted">{m.url || [m.command, ...(m.args ?? [])].join(' ')}</div>
              {(m.envKeys ?? []).length > 0 && <div className="text-xs text-muted">env: {(m.envKeys ?? []).join(', ')}</div>}
            </li>
          ))}
        </ul>
      </Card>
      <Card title={`プラグイン（${plugins.length}）`}>
        <Empty n={plugins.length}>プラグインはありません。</Empty>
        <ul aria-label="プラグイン" className="space-y-1 text-sm">
          {plugins.map((p) => (
            <li key={p.id}>
              <span className="font-medium">{p.id}</span>{' '}
              <span className="text-xs text-muted">{p.version}</span>{' '}
              <span className={`badge ${p.enabled ? 'badge-success' : 'badge-neutral'}`}>{p.enabled ? '有効' : '無効'}</span>
            </li>
          ))}
        </ul>
      </Card>
      <Card title={`Skills（${skills.length}）`}>
        <Empty n={skills.length}>skill はありません。</Empty>
        <ul aria-label="Skills" className="space-y-1 text-sm">
          {skills.map((s) => (
            <li key={s.path}><span className="font-medium">{s.name}</span>{s.description && <span className="text-muted"> — {s.description}</span>}</li>
          ))}
        </ul>
      </Card>
      <Card title={`Hooks（${hooks.length}）`}>
        <Empty n={hooks.length}>hook は設定されていません。</Empty>
        <ul aria-label="Hooks" className="space-y-1 text-sm">
          {hooks.map((h, i) => (
            <li key={i}>
              <span className="font-medium">{h.event}</span>{h.matcher && <span className="text-xs text-muted"> [{h.matcher}]</span>}
              <div className="break-all font-mono text-xs text-muted">{h.command}</div>
            </li>
          ))}
        </ul>
      </Card>
    </div>
  )
}
