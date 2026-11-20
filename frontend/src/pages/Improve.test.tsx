import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
import {deferred} from '../testUtils'
import Improve from './Improve'

afterEach(() => { vi.restoreAllMocks() })

const autoProposal = {targetPath: '/home/demo/app/CLAUDE.md', baseHash: 'h', original: '# a\n', output: '# a\n- 追加\n', rationale: '理由', digest: 'd', jobId: 'job-auto'}

test('自動生成された提案は確認するまで適用されず、適用すると破棄扱いになる', async () => {
  vi.spyOn(api, 'autoSuggestion').mockResolvedValue(autoProposal)
  const apply = vi.spyOn(api, 'applyProposal').mockResolvedValue({id: 'b1', path: autoProposal.targetPath} as never)
  const dismiss = vi.spyOn(api, 'dismissAutoSuggestion')
  render(<Improve />)
  await userEvent.click(await screen.findByRole('button', {name: '差分を確認する'}))
  expect(apply).not.toHaveBeenCalled()
  expect(await screen.findByTestId('diff-stats')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', {name: '適用'}))
  expect(await screen.findByText(/に適用しました/)).toBeInTheDocument()
  expect(apply).toHaveBeenCalledTimes(1)
  expect(dismiss).toHaveBeenCalled()
  vi.restoreAllMocks()
})

test('自動生成された提案を破棄できる', async () => {
  vi.spyOn(api, 'autoSuggestion').mockResolvedValue(autoProposal)
  const apply = vi.spyOn(api, 'applyProposal')
  const dismiss = vi.spyOn(api, 'dismissAutoSuggestion')
  render(<Improve />)
  await userEvent.click(await screen.findByRole('button', {name: '破棄'}))
  expect(dismiss).toHaveBeenCalled()
  expect(apply).not.toHaveBeenCalled()
  expect(screen.queryByText('自動生成された提案があります')).toBeNull()
  vi.restoreAllMocks()
})

test('自動生成された提案が無ければ案内を出さない', async () => {
  render(<Improve />)
  await screen.findByLabelText('対象ファイル')
  expect(screen.queryByText('自動生成された提案があります')).toBeNull()
})

const proj = '/home/demo/app/CLAUDE.md'

async function pick() {
  const sel = await screen.findByLabelText('対象ファイル')
  await screen.findByRole('option', {name: /CLAUDE.md（\/home\/demo\/app\/CLAUDE.md）/})
  await userEvent.selectOptions(sel, proj)
}

test('digest をプレビューできる', async () => {
  render(<Improve />)
  await pick()
  await userEvent.click(screen.getByRole('button', {name: /digest をプレビュー/}))
  expect(await screen.findByTestId('digest')).toHaveTextContent('訂正発言')
})

test('生成 → 差分 → 編集 → 適用', async () => {
  render(<Improve />)
  await pick()
  await userEvent.click(screen.getByRole('button', {name: /改善案を生成/}))
  expect(screen.getByRole('status')).toHaveTextContent('生成中')
  expect(await screen.findByText(/変更理由/)).toBeInTheDocument()
  expect(await screen.findByTestId('diff-stats')).toHaveTextContent('追加 1 行')
  expect(screen.getByText(/先に go vet/)).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', {name: '右側を編集'}))
  const ta = screen.getByLabelText('改善案の編集')
  await userEvent.type(ta, '追記')
  await userEvent.click(screen.getByRole('button', {name: '編集を終えて差分を見る'}))
  await userEvent.click(screen.getByRole('button', {name: '適用'}))
  expect(await screen.findByText(/に適用しました/)).toBeInTheDocument()
})

test('生成中にキャンセルできる', async () => {
  render(<Improve />)
  await pick()
  await userEvent.click(screen.getByRole('button', {name: /改善案を生成/}))
  await userEvent.click(screen.getByRole('button', {name: 'キャンセル'}))
  expect(await screen.findByRole('alert')).toHaveTextContent('キャンセル')
  expect(screen.getByRole('button', {name: /改善案を生成/})).toBeEnabled()
})

test('破棄すると選択画面に戻り、何も適用されない', async () => {
  render(<Improve />)
  await pick()
  await userEvent.click(screen.getByRole('button', {name: /改善案を生成/}))
  await screen.findByTestId('diff-stats')
  await userEvent.click(screen.getByRole('button', {name: '破棄'}))
  expect(screen.getByLabelText('対象ファイル')).toBeInTheDocument()
  expect(screen.queryByText(/に適用しました/)).not.toBeInTheDocument()
})

test('改善案の生成中はスピナーと実行中表示が出て、ボタンが無効化される。完了で消える', async () => {
  const d = deferred<Awaited<ReturnType<typeof api.generateProposal>>>()
  vi.spyOn(api, 'generateProposal').mockReturnValue(d.promise)
  render(<Improve />)
  await pick()
  await userEvent.click(screen.getByRole('button', {name: /改善案を生成/}))
  const status = screen.getByRole('status')
  expect(status).toHaveTextContent('生成中')
  expect(status).toHaveAttribute('aria-busy', 'true')
  const btn = screen.getByRole('button', {name: /生成中…/})
  expect(btn).toBeDisabled()
  expect(btn).toHaveAttribute('aria-busy', 'true')
  expect(screen.getAllByTestId('spinner').length).toBeGreaterThan(0)
  d.resolve({targetPath: proj, baseHash: 'h', original: '# a\n', output: '# a\n- x\n', rationale: '', digest: '', jobId: 'j'})
  expect(await screen.findByText(/変更理由/)).toBeInTheDocument()
  expect(screen.queryByRole('status')).toBeNull()
})

test('digest 作成中と適用中は実行中表示になり、2 重に押せない', async () => {
  const dg = deferred<string>()
  vi.spyOn(api, 'buildDigest').mockReturnValue(dg.promise)
  const {unmount} = render(<Improve />)
  await pick()
  await userEvent.click(screen.getByRole('button', {name: /digest をプレビュー/}))
  expect(screen.getByRole('button', {name: /digest を作成中/})).toBeDisabled()
  expect(screen.getByRole('button', {name: /改善案を生成/})).toBeDisabled()
  dg.resolve('# d')
  expect(await screen.findByTestId('digest')).toBeInTheDocument()
  unmount()

  const ap = deferred<never>()
  const apply = vi.spyOn(api, 'applyProposal').mockReturnValue(ap.promise)
  render(<Improve />)
  await pick()
  await userEvent.click(screen.getByRole('button', {name: /改善案を生成/}))
  await userEvent.click(await screen.findByRole('button', {name: '適用'}))
  const busy = screen.getByRole('button', {name: /適用中…/})
  expect(busy).toBeDisabled()
  await userEvent.click(busy)
  expect(apply).toHaveBeenCalledTimes(1)
  ap.reject(new Error('失敗した'))
  expect(await screen.findByRole('alert')).toHaveTextContent('失敗した')
  expect(screen.getByRole('button', {name: '適用'})).toBeEnabled()
})
