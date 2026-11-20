import type {ButtonHTMLAttributes, ReactNode} from 'react'

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
    <p role="status" aria-live="polite" aria-busy="true" className="flex items-center gap-2 text-sm text-gray-500">
      <Spinner />{label}
    </p>
  )
}

/** 実行中の表示（スピナー＋文言）。AI の実行など時間のかかる操作の進行中に出す。 */
export function Busy({label}: {label: string}) {
  return (
    <p role="status" aria-live="polite" aria-busy="true" className="mt-2 flex items-center gap-2 text-sm text-gray-600">
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
    <button {...rest} type={type} className={`${className} inline-flex items-center gap-1.5 disabled:cursor-not-allowed`} disabled={disabled || busy} aria-busy={busy || undefined}>
      {busy && <Spinner />}
      {busy && busyLabel ? busyLabel : children}
    </button>
  )
}

export const fmt = (n: number) => n.toLocaleString('ja-JP')

export function shortProject(p: string): string {
  return p.replace(/^-/, '').replace(/-/g, '/')
}
