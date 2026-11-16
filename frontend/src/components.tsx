import type {ReactNode} from 'react'

export function Card({title, children}: {title?: string; children: ReactNode}) {
  return (
    <section className="rounded-lg border bg-white p-4 shadow-sm">
      {title && <h3 className="mb-3 text-sm font-semibold text-gray-600">{title}</h3>}
      {children}
    </section>
  )
}

export function Kpi({label, value}: {label: string; value: string | number}) {
  return (
    <div className="rounded-lg border bg-white p-4 shadow-sm">
      <div className="text-xs text-gray-500">{label}</div>
      <div className="mt-1 text-2xl font-bold" data-testid={`kpi-${label}`}>{value}</div>
    </div>
  )
}

export function ErrorBox({message}: {message: string}) {
  if (!message) return null
  return <div role="alert" className="rounded border border-red-300 bg-red-50 p-3 text-sm text-red-800">{message}</div>
}

export function Loading() {
  return <p className="text-sm text-gray-500">読み込み中…</p>
}

export const fmt = (n: number) => n.toLocaleString('ja-JP')

export function shortProject(p: string): string {
  return p.replace(/^-/, '').replace(/-/g, '/')
}
