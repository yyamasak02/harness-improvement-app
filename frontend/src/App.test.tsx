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
