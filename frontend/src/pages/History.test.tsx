import {render, screen, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
import {deferred} from '../testUtils'
import History from './History'

const target = '/home/demo/app/CLAUDE.md'

test('履歴が無いときの表示 → 適用後に履歴が出て、確認ダイアログ経由でロールバックできる', async () => {
  const {unmount} = render(<History />)
  expect(await screen.findByText('履歴はまだありません。')).toBeInTheDocument()
  unmount()

  const tc = await api.readTarget(target)
  await api.applyProposal(target, tc.content + '- 追加\n', tc.hash)

  render(<History />)
  const list = await screen.findByRole('list', {name: '履歴一覧'})
  await userEvent.click(await within(list).findByRole('button'))
  expect(await screen.findByTestId('diff-stats')).toHaveTextContent('追加 1 行')

  await userEvent.click(screen.getByRole('button', {name: 'この変更をロールバック'}))
  const dlg = screen.getByRole('dialog', {name: 'ロールバックの確認'})
  await userEvent.click(within(dlg).getByRole('button', {name: 'キャンセル'}))
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  expect((await api.readTarget(target)).content).toContain('- 追加')

  await userEvent.click(screen.getByRole('button', {name: 'この変更をロールバック'}))
  await userEvent.click(screen.getByRole('button', {name: 'ロールバックする'}))
  expect(await screen.findByText(/ロールバックしました/)).toBeInTheDocument()
  expect((await api.readTarget(target)).content).toBe(tc.content)
})

test('ロールバック中は実行中表示が出て、ロールバックのボタンが無効化される', async () => {
  const tc = await api.readTarget(target)
  await api.applyProposal(target, tc.content + '- 追加2\n', tc.hash)
  const d = deferred<Awaited<ReturnType<typeof api.rollback>>>()
  vi.spyOn(api, 'rollback').mockReturnValue(d.promise)
  render(<History />)
  const list = await screen.findByRole('list', {name: '履歴一覧'})
  await userEvent.click((await within(list).findAllByRole('button'))[0])
  await userEvent.click(await screen.findByRole('button', {name: 'この変更をロールバック'}))
  await userEvent.click(screen.getByRole('button', {name: 'ロールバックする'}))
  expect(screen.getByRole('status')).toHaveTextContent('ロールバック中')
  expect(screen.getByRole('button', {name: 'この変更をロールバック'})).toBeDisabled()
  d.reject(new Error('失敗した'))
  expect(await screen.findByRole('alert')).toHaveTextContent('失敗した')
  expect(screen.getByRole('button', {name: 'この変更をロールバック'})).toBeEnabled()
  vi.restoreAllMocks()
})
