import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
import {deferred} from '../testUtils'
import Reports from './Reports'

test('日報を生成して一覧と内容に表示する', async () => {
  render(<Reports />)
  await userEvent.click(screen.getByRole('button', {name: '今日の日報を生成'}))
  expect(await screen.findByRole('status')).toHaveTextContent('の日報を保存しました')
  expect(await screen.findByText('mock の日報')).toBeInTheDocument()
  expect((await api.listReports()).length).toBeGreaterThan(0)
  expect(await screen.findByRole('list', {name: '日報一覧'})).toBeInTheDocument()
})

test('plist を表示できる（生成のみ）', async () => {
  render(<Reports />)
  await userEvent.click(screen.getByRole('button', {name: 'launchd の plist を生成'}))
  expect(((await screen.findByLabelText('plist')) as HTMLTextAreaElement).value).toContain('app.harness-app.daily-report')
})

test('生成に失敗するとエラーを表示する', async () => {
  const orig = api.generateReport
  api.generateReport = () => Promise.reject(new Error('claude が見つかりません'))
  try {
    render(<Reports />)
    await userEvent.click(screen.getByRole('button', {name: '今日の日報を生成'}))
    expect(await screen.findByRole('alert')).toHaveTextContent('claude が見つかりません')
  } finally {
    api.generateReport = orig
  }
})

test('日報の生成中はスピナーと実行中表示が出て、ボタンが無効化される', async () => {
  const d = deferred<{date: string; path: string}>()
  const orig = api.generateReport
  api.generateReport = () => d.promise
  try {
    render(<Reports />)
    await userEvent.click(screen.getByRole('button', {name: '今日の日報を生成'}))
    const btn = screen.getByRole('button', {name: /生成中…/})
    expect(btn).toBeDisabled()
    expect(btn).toHaveAttribute('aria-busy', 'true')
    expect(screen.getByTestId('spinner')).toBeInTheDocument()
    d.reject(new Error('失敗'))
    expect(await screen.findByRole('alert')).toHaveTextContent('失敗')
    expect(screen.getByRole('button', {name: '今日の日報を生成'})).toBeEnabled()
  } finally {
    api.generateReport = orig
  }
})
