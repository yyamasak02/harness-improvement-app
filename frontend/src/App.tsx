import {HashRouter, NavLink, Route, Routes} from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import Sessions from './pages/Sessions'
import Improve from './pages/Improve'
import Memory from './pages/Memory'
import Skills from './pages/Skills'
import Reports from './pages/Reports'
import Extensions from './pages/Extensions'
import History from './pages/History'
import Settings from './pages/Settings'
import {Icon} from './components'
import type {IconName} from './components'

const nav: {to: string; label: string; icon: IconName}[] = [
  {to: '/', label: '概要', icon: 'home'},
  {to: '/sessions', label: 'セッション', icon: 'sessions'},
  {to: '/improve', label: 'CLAUDE.md 改善', icon: 'improve'},
  {to: '/memory', label: 'メモリ', icon: 'memory'},
  {to: '/skills', label: 'ナレッジ生成', icon: 'skills'},
  {to: '/reports', label: '日次振り返り', icon: 'reports'},
  {to: '/extensions', label: '拡張機能', icon: 'extensions'},
  {to: '/history', label: '変更履歴', icon: 'history'},
  {to: '/settings', label: '設定', icon: 'settings'},
]

export default function App() {
  return (
    <HashRouter>
      <div className="flex h-screen overflow-hidden bg-canvas text-fg">
        <aside className="w-40 shrink-0 overflow-y-auto border-r bg-surface p-3 md:w-56 md:p-4">
          <h1 className="mb-0.5 text-lg font-bold tracking-tight">harness-app</h1>
          <p className="mb-5 text-xs text-muted">Claude Code ハーネス改善</p>
          <nav className="flex flex-col gap-1" aria-label="メイン">
            {nav.map((n) => (
              <NavLink key={n.to} to={n.to} end={n.to === '/'}
                className={({isActive}) => `flex items-center gap-2.5 rounded-ctl px-3 py-2 text-sm font-medium transition-colors ${isActive ? 'bg-accent text-white shadow-sm' : 'text-muted hover:bg-subtle hover:text-fg'}`}>
                <Icon name={n.icon} />
                <span className="min-w-0 break-words">{n.label}</span>
              </NavLink>
            ))}
          </nav>
        </aside>
        <main className="min-w-0 flex-1 overflow-auto p-4 md:p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/sessions" element={<Sessions />} />
            <Route path="/improve" element={<Improve />} />
            <Route path="/memory" element={<Memory />} />
            <Route path="/skills" element={<Skills />} />
            <Route path="/reports" element={<Reports />} />
            <Route path="/extensions" element={<Extensions />} />
            <Route path="/history" element={<History />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </HashRouter>
  )
}
