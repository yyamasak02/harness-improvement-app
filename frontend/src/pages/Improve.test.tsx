import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Improve from './Improve'

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
