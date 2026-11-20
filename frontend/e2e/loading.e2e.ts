import {expect, test} from '@playwright/test'

// mockApi の応答を遅らせ（window.__mockSlowMs）、実行中の表示とボタンの無効化、完了後の復帰を確認する。
// 実 claude・実 ~/.claude には触れない。
test('インデックス再構築の実行中はスピナー表示・ボタン無効化になり、完了すると戻る', async ({page}) => {
  await page.addInitScript(() => { (window as unknown as {__mockSlowMs: number}).__mockSlowMs = 1500 })
  await page.goto('/#/settings')
  await page.getByRole('button', {name: 'インデックスを再構築'}).click()
  const busy = page.getByRole('button', {name: /再構築中…/})
  await expect(busy).toBeDisabled()
  await expect(busy).toHaveAttribute('aria-busy', 'true')
  await expect(busy.getByTestId('spinner')).toBeVisible()
  await expect(page.getByRole('button', {name: '保存'})).toBeDisabled()
  await expect(page.getByText(/再構築しました/)).toBeVisible()
  await expect(page.getByRole('button', {name: 'インデックスを再構築'})).toBeEnabled()
})

test('日報の生成中も実行中表示が出る', async ({page}) => {
  await page.addInitScript(() => { (window as unknown as {__mockSlowMs: number}).__mockSlowMs = 1500 })
  await page.goto('/#/reports')
  await page.getByRole('button', {name: '今日の日報を生成'}).click()
  await expect(page.getByRole('button', {name: /生成中…/})).toBeDisabled()
  await expect(page.getByText(/の日報を保存しました/)).toBeVisible()
})
