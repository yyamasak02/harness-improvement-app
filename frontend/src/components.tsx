import type {ButtonHTMLAttributes, ReactNode} from 'react'

export function Card({title, children}: {title?: string; children: ReactNode}) {
  return (
    <section className="rounded-card border bg-surface p-4 shadow-card">
      {title && <h3 className="mb-3 text-sm font-semibold text-muted">{title}</h3>}
      {children}
    </section>
  )
}

export function Kpi({label, value}: {label: string; value: string | number}) {
  return (
    <div className="rounded-card border bg-surface p-4 shadow-card">
      <div className="text-xs font-medium text-muted">{label}</div>
      <div className="mt-1 text-2xl font-bold tracking-tight" data-testid={`kpi-${label}`}>{value}</div>
    </div>
  )
}

export function ErrorBox({message}: {message: string}) {
  if (!message) return null
  return <div role="alert" className="alert alert-danger">{message}</div>
}

/** 回転するスピナー（装飾。読み上げは周囲の role="status" が担う）。アニメーションは style.css で prefers-reduced-motion を尊重する。 */
export function Spinner({className = ''}: {className?: string}) {
  return (
    <svg className={`spinner inline-block h-4 w-4 align-[-0.15em] ${className}`} viewBox="0 0 24 24" fill="none" aria-hidden="true" data-testid="spinner">
      <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="3" opacity="0.25" />
      <path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
    </svg>
  )
}

/** 読み込み中の表示。スクリーンリーダーにも通知される。 */
export function Loading({label = '読み込み中…'}: {label?: string}) {
  return (
    <p role="status" aria-live="polite" aria-busy="true" className="flex items-center gap-2 text-sm text-muted">
      <Spinner />{label}
    </p>
  )
}

/** 実行中の表示（スピナー＋文言）。AI の実行など時間のかかる操作の進行中に出す。 */
export function Busy({label}: {label: string}) {
  return (
    <p role="status" aria-live="polite" aria-busy="true" className="mt-2 flex items-center gap-2 text-sm text-muted">
      <Spinner />{label}
    </p>
  )
}

interface LoadingButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  busy: boolean
  /** 実行中に表示する文言（省略時は通常のラベルのまま）。 */
  busyLabel?: string
  children: ReactNode
}

/** 実行中は無効化（2 重実行の防止）してスピナーを出すボタン。aria-busy も付ける。 */
export function LoadingButton({busy, busyLabel, children, disabled, className = '', type = 'button', ...rest}: LoadingButtonProps) {
  return (
    <button {...rest} type={type} className={`${className} inline-flex items-center gap-1.5 disabled:cursor-not-allowed disabled:opacity-50`} disabled={disabled || busy} aria-busy={busy || undefined}>
      {busy && <Spinner />}
      {busy && busyLabel ? busyLabel : children}
    </button>
  )
}

type IconName = 'home' | 'sessions' | 'improve' | 'memory' | 'skills' | 'reports' | 'extensions' | 'history' | 'settings'

// 24x24・線画の inline SVG（外部アイコンパッケージは使わない）
const iconPaths: Record<IconName, string> = {
  home: 'M3 11l9-8 9 8M5 10v10h5v-6h4v6h5V10',
  sessions: 'M4 5h16v11H9l-5 4V5z',
  improve: 'M4 20l4-1 11-11-3-3L5 16l-1 4zM14 6l3 3',
  memory: 'M5 4h10l4 4v12H5V4zM14 4v5h5M8 13h8M8 17h6',
  skills: 'M12 3l2.5 5.5L20 9.3l-4 4 1 5.7-5-2.8L7 19l1-5.7-4-4 5.5-.8L12 3z',
  reports: 'M5 4h14v16H5V4zM9 9h6M9 13h6M9 17h3',
  extensions: 'M10 4h4v3a2 2 0 1 0 0 4v3h-4v-3a2 2 0 1 1 0-4V4zM4 14h16v6H4v-6z',
  history: 'M4 12a8 8 0 1 0 3-6.2M4 4v4h4M12 8v4l3 2',
  settings: 'M4 7h10M18 7h2M4 17h2M10 17h10M16 4v6M8 14v6',
}

/** ナビ用のアイコン（装飾。ラベルは隣のテキストが担う）。 */
export function Icon({name, className = ''}: {name: IconName; className?: string}) {
  return (
    <svg className={`h-4 w-4 shrink-0 ${className}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={iconPaths[name]} />
    </svg>
  )
}

export type {IconName}

export const fmt = (n: number) => n.toLocaleString('ja-JP')

export function shortProject(p: string): string {
  return p.replace(/^-/, '').replace(/-/g, '/')
}
