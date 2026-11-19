import {render, screen, within} from '@testing-library/react'
import {api} from '../api'
import Extensions from './Extensions'

test('MCP・プラグイン・skill・hook を一覧表示する', async () => {
  render(<Extensions />)
  const mcp = await screen.findByRole('list', {name: 'MCP サーバー'})
  expect(within(mcp).getByText('demo-stdio')).toBeInTheDocument()
  expect(within(mcp).getByText(/env: DEMO_KEY/)).toBeInTheDocument()
  expect(within(mcp).getByText('https://mcp.example.com/mcp')).toBeInTheDocument()
  const plugins = screen.getByRole('list', {name: 'プラグイン'})
  expect(within(plugins).getByText('有効')).toBeInTheDocument()
  expect(within(plugins).getByText('無効')).toBeInTheDocument()
  expect(within(screen.getByRole('list', {name: 'Skills'})).getByText('demo-skill')).toBeInTheDocument()
  expect(within(screen.getByRole('list', {name: 'Hooks'})).getByText('echo demo-check')).toBeInTheDocument()
})

test('空の項目は案内を出し、警告を表示する', async () => {
  const spy = vi.spyOn(api, 'listExtensions').mockResolvedValue({mcp: null, plugins: null, skills: null, hooks: null, warnings: ['settings.json を読めませんでした']})
  render(<Extensions />)
  expect(await screen.findByText('MCP サーバーは設定されていません。')).toBeInTheDocument()
  expect(screen.getByText('hook は設定されていません。')).toBeInTheDocument()
  expect(screen.getByText(/settings.json を読めませんでした/)).toBeInTheDocument()
  spy.mockRestore()
})

test('読み込み失敗はエラー表示', async () => {
  const spy = vi.spyOn(api, 'listExtensions').mockRejectedValue(new Error('失敗した'))
  render(<Extensions />)
  expect(await screen.findByText(/失敗した/)).toBeInTheDocument()
  spy.mockRestore()
})
