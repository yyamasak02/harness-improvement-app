import {HashRouter, NavLink, Route, Routes} from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import Sessions from './pages/Sessions'
import Improve from './pages/Improve'
import Memory from './pages/Memory'
import Skills from './pages/Skills'
import Extensions from './pages/Extensions'
import History from './pages/History'
import Settings from './pages/Settings'

const nav = [
  {to: '/', label: '概要'},
  {to: '/sessions', label: 'セッション'},
  {to: '/improve', label: 'CLAUDE.md 改善'},
  {to: '/memory', label: 'メモリ'},
  {to: '/skills', label: 'ナレッジ生成'},
  {to: '/extensions', label: '拡張機能'},
  {to: '/history', label: '変更履歴'},
  {to: '/settings', label: '設定'},
]

export default function App() {
  return (
    <HashRouter>
      <div className="flex h-screen bg-gray-50 text-gray-900">
        <aside className="w-52 shrink-0 border-r bg-white p-4">
          <h1 className="mb-1 text-lg font-bold">harness-app</h1>
          <p className="mb-4 text-xs text-gray-500">Claude Code ハーネス改善</p>
          <nav className="flex flex-col gap-1" aria-label="メイン">
            {nav.map((n) => (
              <NavLink key={n.to} to={n.to} end={n.to === '/'}
                className={({isActive}) => `rounded px-3 py-2 text-sm ${isActive ? 'bg-blue-600 text-white' : 'hover:bg-gray-100'}`}>
                {n.label}
              </NavLink>
            ))}
          </nav>
        </aside>
        <main className="flex-1 overflow-auto p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/sessions" element={<Sessions />} />
            <Route path="/improve" element={<Improve />} />
            <Route path="/memory" element={<Memory />} />
            <Route path="/skills" element={<Skills />} />
            <Route path="/extensions" element={<Extensions />} />
            <Route path="/history" element={<History />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </HashRouter>
  )
}
