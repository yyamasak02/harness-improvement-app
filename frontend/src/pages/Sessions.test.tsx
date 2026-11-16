import {render, screen, waitFor, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Sessions, {MessageRow} from './Sessions'

test('一覧から詳細を開き、訂正とエラーをハイライトする', async () => {
  render(<Sessions />)
  const list = await screen.findByRole('list', {name: 'セッション一覧'})
  const buttons = await waitFor(() => {
    const b = within(list).getAllByRole('button')
    expect(b).toHaveLength(2)
    return b
  })
  await userEvent.click(buttons[1]) // sess-0001
  const msgs = await screen.findAllByTestId('message')
  expect(msgs).toHaveLength(4)
  expect(screen.getAllByText('訂正').length).toBeGreaterThan(0)
  expect(screen.getAllByText('エラー').length).toBeGreaterThan(0)
})

test('全文検索の結果から詳細へ移動できる', async () => {
  render(<Sessions />)
  await userEvent.type(await screen.findByLabelText('全文検索'), 'go vet')
  await userEvent.click(screen.getByRole('button', {name: '検索'}))
  const res = await screen.findByRole('list', {name: '検索結果'})
  await userEvent.click(within(res).getAllByRole('button')[0])
  expect(await screen.findAllByTestId('message')).not.toHaveLength(0)
})

test('検索で該当が無ければ該当なしを表示する', async () => {
  render(<Sessions />)
  await userEvent.type(await screen.findByLabelText('全文検索'), 'zzzzzz')
  await userEvent.click(screen.getByRole('button', {name: '検索'}))
  expect(await screen.findByText('該当なし')).toBeInTheDocument()
})

test('MessageRow は種別を日本語で表示する', () => {
  render(<ul><MessageRow m={{id: 1, sessionId: 's', project: 'p', ts: '2026-11-01T00:00:00Z', role: 'assistant', kind: 'tool_use', tool: 'Bash', text: 'ls', isError: false, isCorrection: false}} /></ul>)
  expect(screen.getByText('ツール呼び出し (Bash)')).toBeInTheDocument()
})
