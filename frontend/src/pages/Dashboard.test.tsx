import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Dashboard from './Dashboard'

test('KPI・モデル・プロジェクトを表示し、期間を切り替えられる', async () => {
  render(<Dashboard />)
  expect(await screen.findByTestId('kpi-セッション')).toHaveTextContent('2')
  expect(screen.getByTestId('kpi-訂正')).toHaveTextContent('2')
  expect(screen.getByTestId('kpi-メモリファイル')).toHaveTextContent('2')
  expect(screen.getAllByText('claude-sonnet-4-5').length).toBeGreaterThan(0)
  expect(screen.getByText('home/demo/app')).toBeInTheDocument()
  await userEvent.selectOptions(screen.getByLabelText('期間'), '7')
  expect(screen.getByLabelText('期間')).toHaveValue('7')
})
