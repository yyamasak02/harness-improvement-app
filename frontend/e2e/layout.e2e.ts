import {expect, test, type Page} from '@playwright/test'

const sizes = [
  {width: 480, height: 360},
  {width: 640, height: 480},
  {width: 800, height: 600},
  {width: 1024, height: 700},
  {width: 1440, height: 900},
]

const pages = [
  {path: '/', name: '概要'},
  {path: '/sessions', name: 'セッション'},
  {path: '/improve', name: 'CLAUDE.md 改善'},
  {path: '/memory', name: 'メモリ'},
  {path: '/skills', name: 'ナレッジ生成'},
  {path: '/reports', name: '日次振り返り'},
  {path: '/extensions', name: '拡張機能'},
  {path: '/history', name: '変更履歴'},
  {path: '/settings', name: '設定'},
]

// 横方向のはみ出しを集める。自前でスクロールする祖先（overflow-x: auto など）の中は許容する。
async function overflows(page: Page) {
  return page.evaluate(() => {
    const out: string[] = []
    const de = document.documentElement
    if (de.scrollWidth > de.clientWidth) out.push(`document: scrollWidth ${de.scrollWidth} > clientWidth ${de.clientWidth}`)
    if (de.scrollHeight > de.clientHeight) out.push(`document: scrollHeight ${de.scrollHeight} > clientHeight ${de.clientHeight}`)
    const main = document.querySelector('main')
    if (main && main.scrollWidth > main.clientWidth) out.push(`main: scrollWidth ${main.scrollWidth} > clientWidth ${main.clientWidth}`)
    const clipped = (el: Element) => {
      for (let p = el.parentElement; p && p !== document.body; p = p.parentElement) {
        if (getComputedStyle(p).overflowX !== 'visible') return true
      }
      return false
    }
    const vw = de.clientWidth
    for (const el of Array.from(document.body.querySelectorAll('*'))) {
      const r = el.getBoundingClientRect()
      if (r.width === 0 || r.height === 0) continue
      if ((r.right > vw + 1 || r.left < -1) && !clipped(el)) {
        out.push(`${el.tagName.toLowerCase()}.${String(el.className).slice(0, 60)} right=${Math.round(r.right)} > ${vw}`)
      }
    }
    return out.slice(0, 10)
  })
}

for (const size of sizes) {
  test.describe(`${size.width}x${size.height}`, () => {
    test.use({viewport: size})
    for (const p of pages) {
      test(`${p.name} がはみ出さない`, async ({page}) => {
        await page.goto(`/#${p.path}`)
        await expect(page.getByRole('navigation', {name: 'メイン'})).toBeVisible()
        await page.waitForLoadState('networkidle')
        expect(await overflows(page)).toEqual([])
      })
    }

    test('改善案の差分表示と編集がはみ出さない', async ({page}) => {
      await page.goto('/#/improve')
      const sel = page.getByLabel('対象ファイル')
      await expect(sel.locator('option')).not.toHaveCount(1)
      await sel.selectOption({index: 1})
      await page.getByRole('button', {name: '② digest をプレビュー'}).click()
      await expect(page.getByTestId('digest')).toBeVisible()
      expect(await overflows(page)).toEqual([])
      await page.getByRole('button', {name: '④ 改善案を生成'}).click()
      await expect(page.getByTestId('diff-stats')).toBeVisible()
      expect(await overflows(page)).toEqual([])
      await page.getByRole('button', {name: '右側を編集'}).click()
      expect(await overflows(page)).toEqual([])
    })

    test('一覧から詳細を開いてもはみ出さない', async ({page}) => {
      for (const path of ['/sessions', '/memory', '/reports']) {
        await page.goto(`/#${path}`)
        await page.waitForLoadState('networkidle')
        if (path === '/reports') {
          await page.getByRole('button', {name: '今日の日報を生成'}).click()
          await page.getByRole('button', {name: 'launchd の plist を生成'}).click()
        }
        const item = page.locator('main li button').first()
        await item.click()
        await page.waitForLoadState('networkidle')
        expect(await overflows(page), path).toEqual([])
      }
    })

    test('ロールバック確認ダイアログが枠内に収まる', async ({page}) => {
      // mockApi の履歴は空から始まるので、まず改善案を適用して履歴を作る（画面内の遷移で状態を保つ）
      await page.goto('/#/improve')
      const sel = page.getByLabel('対象ファイル')
      await expect(sel.locator('option')).not.toHaveCount(1)
      await sel.selectOption({index: 1})
      await page.getByRole('button', {name: '④ 改善案を生成'}).click()
      await page.getByRole('button', {name: '適用', exact: true}).click()
      await expect(page.getByRole('status')).toContainText('適用しました')
      await page.getByRole('link', {name: '変更履歴'}).click()
      await page.locator('main li button').first().click()
      expect(await overflows(page)).toEqual([])
      await page.getByRole('button', {name: 'この変更をロールバック'}).click()
      const box = await page.getByRole('dialog').boundingBox()
      expect(box!.x).toBeGreaterThanOrEqual(0)
      expect(box!.x + box!.width).toBeLessThanOrEqual(size.width)
      expect(box!.y + box!.height).toBeLessThanOrEqual(size.height)
    })

    test('サイドバーの全項目に到達できる', async ({page}) => {
      await page.goto('/#/')
      const last = page.getByRole('link', {name: '設定'})
      await last.scrollIntoViewIfNeeded()
      await expect(last).toBeInViewport()
    })
  })
}
