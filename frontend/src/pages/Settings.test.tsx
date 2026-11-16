import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
import Settings, {regexError} from './Settings'

test('claude が見つからないときに案内を表示する', async () => {
  render(<Settings />)
  expect(await screen.findByRole('alert')).toHaveTextContent('claude コマンドが見つかりません')
  expect(screen.getByRole('alert')).toHaveTextContent('which claude')
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

test('regexError', () => {
  expect(regexError({name: 'a', regex: 'a+'})).toBe('')
  expect(regexError({name: 'a', regex: '('})).not.toBe('')
})
