import {render, screen, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
import {deferred} from '../testUtils'
import Memory from './Memory'

test('一覧に重複・古い印が出て、閲覧できる', async () => {
  render(<Memory />)
  const list = await screen.findByRole('list', {name: 'メモリ一覧'})
  const items = await within(list).findAllByRole('button')
  expect(items).toHaveLength(2)
  expect(within(list).getByText('古い')).toBeInTheDocument()
  expect(within(list).getAllByText(/重複:/)).toHaveLength(2)
  await userEvent.click(items[0])
  expect(await screen.findByTestId('memory-body')).toHaveTextContent('応答は日本語で行う。')
})

test('編集 → 適用すると内容が変わり履歴に残る。破棄すると変わらない', async () => {
  render(<Memory />)
  const list = await screen.findByRole('list', {name: 'メモリ一覧'})
  await userEvent.click((await within(list).findAllByRole('button'))[0])
  await userEvent.click(await screen.findByRole('button', {name: '編集'}))
  expect(screen.getByRole('button', {name: '適用'})).toBeDisabled()
  await userEvent.type(screen.getByRole('textbox'), '追記')
  await userEvent.click(screen.getByRole('button', {name: '破棄'}))
  expect(screen.getByTestId('memory-body')).not.toHaveTextContent('追記')

  await userEvent.click(screen.getByRole('button', {name: '編集'}))
  await userEvent.type(screen.getByRole('textbox'), '追記')
  await userEvent.click(screen.getByRole('button', {name: '適用'}))
  expect(await screen.findByText(/適用しました/)).toBeInTheDocument()
  expect(await screen.findByTestId('memory-body')).toHaveTextContent('追記')
  expect((await api.listBackups()).length).toBeGreaterThan(0)
})

test('適用中は実行中表示になり、2 重に押せない', async () => {
  const d = deferred<Awaited<ReturnType<typeof api.applyProposal>>>()
  const apply = vi.spyOn(api, 'applyProposal').mockReturnValue(d.promise)
  render(<Memory />)
  const list = await screen.findByRole('list', {name: 'メモリ一覧'})
  await userEvent.click((await within(list).findAllByRole('button'))[0])
  await userEvent.click(await screen.findByRole('button', {name: '編集'}))
  await userEvent.type(screen.getByRole('textbox'), '追記')
  await userEvent.click(screen.getByRole('button', {name: '適用'}))
  const busy = screen.getByRole('button', {name: /適用中…/})
  expect(busy).toBeDisabled()
  expect(busy).toHaveAttribute('aria-busy', 'true')
  await userEvent.click(busy)
  expect(apply).toHaveBeenCalledTimes(1)
  d.reject(new Error('失敗した'))
  expect(await screen.findByRole('alert')).toHaveTextContent('失敗した')
  expect(screen.getByRole('button', {name: '適用'})).toBeEnabled()
  vi.restoreAllMocks()
})
