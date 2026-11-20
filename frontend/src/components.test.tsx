import {render, screen} from '@testing-library/react'
import {Busy, Icon, Loading, LoadingButton, Spinner} from './components'

test('Loading は status として通知され、スピナーは装飾（aria-hidden）', () => {
  render(<Loading />)
  const status = screen.getByRole('status')
  expect(status).toHaveAttribute('aria-busy', 'true')
  expect(status).toHaveAttribute('aria-live', 'polite')
  expect(status).toHaveTextContent('読み込み中…')
  expect(screen.getByTestId('spinner')).toHaveAttribute('aria-hidden', 'true')
})

test('Busy は文言を status として出す', () => {
  render(<Busy label="生成中…" />)
  expect(screen.getByRole('status')).toHaveTextContent('生成中…')
})

test('LoadingButton は実行中に無効化・aria-busy・スピナー・実行中の文言を出す', () => {
  const {rerender} = render(<LoadingButton busy={false} busyLabel="実行中…">実行</LoadingButton>)
  const idle = screen.getByRole('button', {name: '実行'})
  expect(idle).toBeEnabled()
  expect(idle).not.toHaveAttribute('aria-busy')
  expect(screen.queryByTestId('spinner')).toBeNull()

  rerender(<LoadingButton busy busyLabel="実行中…">実行</LoadingButton>)
  const busy = screen.getByRole('button', {name: '実行中…'})
  expect(busy).toBeDisabled()
  expect(busy).toHaveAttribute('aria-busy', 'true')
  expect(screen.getByTestId('spinner')).toBeInTheDocument()
})

test('LoadingButton は busyLabel が無ければラベルを保ち、disabled も尊重する', () => {
  const {rerender} = render(<LoadingButton busy disabled={false}>実行</LoadingButton>)
  expect(screen.getByRole('button', {name: '実行'})).toBeDisabled()
  rerender(<LoadingButton busy={false} disabled>実行</LoadingButton>)
  expect(screen.getByRole('button', {name: '実行'})).toBeDisabled()
})

test('Spinner は className を受け取る', () => {
  render(<Spinner className="x" />)
  expect(screen.getByTestId('spinner')).toHaveClass('x')
})

test('Icon は装飾（aria-hidden）の SVG を描画する', () => {
  const {container} = render(<Icon name="home" className="x" />)
  const svg = container.querySelector('svg')!
  expect(svg).toHaveAttribute('aria-hidden', 'true')
  expect(svg).toHaveClass('x')
  expect(svg.querySelector('path')).toHaveAttribute('d')
})
