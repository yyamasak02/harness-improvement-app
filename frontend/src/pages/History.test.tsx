import {render, screen, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
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
