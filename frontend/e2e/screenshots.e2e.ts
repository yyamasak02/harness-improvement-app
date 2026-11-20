import {expect, test} from '@playwright/test'

// 見た目の確認用。ライト/ダーク × 通常/小さい幅 × 主要ページのスクリーンショットを test-results/screenshots に出力する
// （gitignore 済み）。あわせて、配色がテーマに追従していることを検証する。
const variants = [
  {name: 'light', colorScheme: 'light' as const, viewport: {width: 1280, height: 800}},
  {name: 'dark', colorScheme: 'dark' as const, viewport: {width: 1280, height: 800}},
  {name: 'small-light', colorScheme: 'light' as const, viewport: {width: 800, height: 600}},
  {name: 'small-dark', colorScheme: 'dark' as const, viewport: {width: 800, height: 600}},
]
const pages = [
  {path: '/', name: 'overview'},
  {path: '/sessions', name: 'sessions'},
  {path: '/improve', name: 'improve'},
  {path: '/skills', name: 'skills'},
  {path: '/memory', name: 'memory'},
  {path: '/reports', name: 'reports'},
  {path: '/history', name: 'history'},
  {path: '/extensions', name: 'extensions'},
  {path: '/settings', name: 'settings'},
]

for (const v of variants) {
  test.describe(`スクリーンショット ${v.name}`, () => {
    test.use({viewport: v.viewport, colorScheme: v.colorScheme})
    for (const p of pages) {
      test(p.name, async ({page}) => {
        await page.goto(`/#${p.path}`)
        await expect(page.getByRole('navigation', {name: 'メイン'})).toBeVisible()
        await page.waitForLoadState('networkidle')
        await page.screenshot({path: `test-results/screenshots/${v.name}-${p.name}.png`})
      })
    }

    test('改善案の差分', async ({page}) => {
      await page.goto('/#/improve')
      const sel = page.getByLabel('対象ファイル')
      await expect(sel.locator('option')).not.toHaveCount(1)
      await sel.selectOption({index: 1})
      await page.getByRole('button', {name: '④ 改善案を生成'}).click()
      await expect(page.getByTestId('diff-stats')).toBeVisible()
      await page.screenshot({path: `test-results/screenshots/${v.name}-improve-diff.png`})
    })

    test('配色がテーマに追従する', async ({page}) => {
      await page.goto('/#/')
      const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor)
      const [r, g, b] = bg.match(/\d+/g)!.map(Number)
      const luminance = (r + g + b) / 3
      if (v.colorScheme === 'dark') expect(luminance).toBeLessThan(60)
      else expect(luminance).toBeGreaterThan(200)
    })
  })
}
