import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'

test('サイドバーから画面を移動できる', async () => {
  window.location.hash = '#/'
  render(<App />)
  expect(screen.getByRole('heading', {name: '概要'})).toBeInTheDocument()
  await userEvent.click(screen.getByRole('link', {name: '変更履歴'}))
  expect(screen.getByRole('heading', {name: '変更履歴'})).toBeInTheDocument()
})

test('小さいウィンドウでもはみ出さないよう、サイドバーはスクロール・本文は縮められる（実レイアウトは E2E で検証）', () => {
  window.location.hash = '#/'
  const {container} = render(<App />)
  expect(container.querySelector('aside')).toHaveClass('overflow-y-auto')
  expect(container.querySelector('main')).toHaveClass('min-w-0')
})
