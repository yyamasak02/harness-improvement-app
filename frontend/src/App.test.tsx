import {render, screen} from '@testing-library/react'
import App from './App'

test('アプリ名を表示する', () => {
  render(<App />)
  expect(screen.getByText('harness-app')).toBeInTheDocument()
})
