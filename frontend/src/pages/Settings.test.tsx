import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
import {deferred} from '../testUtils'
import Settings, {regexError} from './Settings'

test('claude が見つからないときに案内を表示する', async () => {
  render(<Settings />)
  expect(await screen.findByRole('alert')).toHaveTextContent('claude コマンドが見つかりません')
  expect(screen.getByRole('alert')).toHaveTextContent('which claude')
})

test('設定ファイルが新しい形式のときに案内を表示する', async () => {
  vi.spyOn(api, 'settingsStatus').mockResolvedValue({newerFormat: true, message: '設定ファイルが新しい形式のため読み込めません。新しいバージョンに更新するか、設定ファイル（config.json）を退避してください。'})
  render(<Settings />)
  expect(await screen.findByText(/新しい形式のため読み込めません/)).toBeInTheDocument()
})

test('設定を編集して保存できる', async () => {
  render(<Settings />)
  const model = await screen.findByLabelText('使用モデル')
  await userEvent.type(model, 'sonnet')
  await userEvent.click(screen.getByRole('button', {name: '保存'}))
  expect(await screen.findByText('保存しました')).toBeInTheDocument()
  expect((await api.settings()).model).toBe('sonnet')
})

test('不正な正規表現は保存できない', async () => {
  render(<Settings />)
  await userEvent.click(await screen.findByRole('button', {name: 'パターンを追加'}))
  await userEvent.type(screen.getByLabelText('正規表現 1'), '(')
  expect(screen.getByText(/正規表現が不正です/)).toBeInTheDocument()
  expect(screen.getByRole('button', {name: '保存'})).toBeDisabled()
  await userEvent.click(screen.getByRole('button', {name: '削除'}))
  expect(screen.getByRole('button', {name: '保存'})).toBeEnabled()
})

test('インデックス再構築の結果を表示する', async () => {
  render(<Settings />)
  await userEvent.click(await screen.findByRole('button', {name: 'インデックスを再構築'}))
  expect(await screen.findByText(/再構築しました/)).toBeInTheDocument()
})

test('改善案の自動生成は既定でオフ。切り替えて保存できる（適用の自動化ではないと説明する）', async () => {
  render(<Settings />)
  const box = await screen.findByLabelText('改善案を自動で生成する')
  expect(box).not.toBeChecked()
  expect(screen.getByText(/自動で作るのは提案までです/)).toBeInTheDocument()
  await userEvent.click(box)
  await userEvent.click(screen.getByRole('button', {name: '保存'}))
  expect(await screen.findByText('保存しました')).toBeInTheDocument()
  expect((await api.settings()).autoSuggestEnabled).toBe(true)
})

test('バージョン情報を表示する（通信なし）', async () => {
  vi.spyOn(api, 'version').mockResolvedValue('1.2.3')
  render(<Settings />)
  expect(await screen.findByText('1.2.3')).toBeInTheDocument()
  expect(screen.getByText(/更新の確認を行いません/)).toBeInTheDocument()
})

test('前回の自動生成の失敗を表示する', async () => {
  vi.spyOn(api, 'autoSuggestStatus').mockResolvedValue({enabled: true, running: false, lastAttemptAt: '2026-11-10T00:00:00Z', lastError: 'claude が見つかりません', hasProposal: false})
  render(<Settings />)
  expect(await screen.findByText(/前回の自動生成に失敗しました: claude が見つかりません/)).toBeInTheDocument()
  vi.restoreAllMocks()
})

test('regexError', () => {
  expect(regexError({name: 'a', regex: 'a+'})).toBe('')
  expect(regexError({name: 'a', regex: '('})).not.toBe('')
})

test('インデックス再構築中は実行中表示になり、保存も含めて 2 重に押せない', async () => {
  const d = deferred<Awaited<ReturnType<typeof api.reindex>>>()
  vi.spyOn(api, 'reindex').mockReturnValue(d.promise)
  render(<Settings />)
  await userEvent.click(await screen.findByRole('button', {name: 'インデックスを再構築'}))
  const btn = screen.getByRole('button', {name: /再構築中…/})
  expect(btn).toBeDisabled()
  expect(btn).toHaveAttribute('aria-busy', 'true')
  expect(screen.getByRole('button', {name: '保存'})).toBeDisabled()
  d.resolve({Scanned: 1, Ingested: 1, Skipped: 0, Removed: 0, BrokenLines: 0})
  expect(await screen.findByText(/再構築しました/)).toBeInTheDocument()
  expect(screen.getByRole('button', {name: 'インデックスを再構築'})).toBeEnabled()
  vi.restoreAllMocks()
})

test('保存中は実行中表示になる', async () => {
  const d = deferred<void>()
  vi.spyOn(api, 'saveSettings').mockReturnValue(d.promise)
  render(<Settings />)
  await userEvent.click(await screen.findByRole('button', {name: '保存'}))
  expect(screen.getByRole('button', {name: /保存中…/})).toBeDisabled()
  d.resolve()
  expect(await screen.findByText('保存しました')).toBeInTheDocument()
  vi.restoreAllMocks()
})

test('自動提案の実行中は実行中表示を出す', async () => {
  vi.spyOn(api, 'autoSuggestStatus').mockResolvedValue({enabled: true, running: true, lastAttemptAt: '', lastError: '', hasProposal: false})
  render(<Settings />)
  expect(await screen.findByText('自動提案を実行中…')).toBeInTheDocument()
  vi.restoreAllMocks()
})
