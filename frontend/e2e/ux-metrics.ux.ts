import AxeBuilder from '@axe-core/playwright'
import {expect, test, type Page} from '@playwright/test'
import {budgets, installLargeDataBridge, median, pages, record, round} from './ux-helpers'

// ユーザー体験の自動計測（定義・予算は docs/ux-metrics.md）。mockApi / 合成ブリッジで動かし、実 ~/.claude・実 claude には触れない。
test.use({viewport: {width: 1024, height: 700}})

const ready = (page: Page) => expect(page.getByRole('navigation', {name: 'メイン'})).toBeVisible()
const settled = async (page: Page) => {
  await ready(page)
  await expect(page.getByText('読み込み中')).toHaveCount(0)
}

test('速さ: 起動から操作可能になるまで', async ({browser}) => {
  const runs: number[] = []
  for (let i = 0; i < 5; i++) {
    const ctx = await browser.newContext({viewport: {width: 1024, height: 700}})
    const page = await ctx.newPage()
    const t0 = Date.now()
    await page.goto('/#/')
    await settled(page)
    runs.push(Date.now() - t0)
    await ctx.close()
  }
  const value = median(runs)
  record({id: 'startup', label: '起動から操作可能まで（中央値・5 回）', unit: 'ms', value, budget: budgets.startupMs, detail: runs})
  expect(value).toBeLessThanOrEqual(budgets.startupMs)
})

test('速さ: ページ遷移', async ({page}) => {
  await page.goto('/#/')
  await settled(page)
  const times: number[] = []
  for (let round_ = 0; round_ < 3; round_++) {
    for (const p of pages) {
      const t0 = Date.now()
      await page.getByRole('navigation', {name: 'メイン'}).getByRole('link', {name: p.link, exact: true}).click()
      await expect(page.getByRole('heading', {level: 2, name: p.name})).toBeVisible()
      await expect(page.getByText('読み込み中')).toHaveCount(0)
      times.push(Date.now() - t0)
    }
  }
  const med = median(times)
  const max = Math.max(...times)
  record({id: 'transition-median', label: 'ページ遷移（中央値・27 回）', unit: 'ms', value: med, budget: budgets.transitionMedianMs})
  record({id: 'transition-max', label: 'ページ遷移（最大・27 回）', unit: 'ms', value: max, budget: budgets.transitionMaxMs})
  expect(med).toBeLessThanOrEqual(budgets.transitionMedianMs)
  expect(max).toBeLessThanOrEqual(budgets.transitionMaxMs)
})

test('速さ: 大量データ（セッション 10000 件・メッセージ 2000 件）', async ({page}) => {
  await installLargeDataBridge(page, 10000, 2000)
  await page.goto('/#/')
  await ready(page)

  const renders: number[] = []
  const scrolls: number[] = []
  for (let i = 0; i < 3; i++) {
    await page.goto('/#/')
    await ready(page)
    const t0 = Date.now()
    await page.getByRole('navigation', {name: 'メイン'}).getByRole('link', {name: 'セッション', exact: true}).click()
    await expect(page.getByRole('list', {name: 'セッション一覧'}).locator('li')).toHaveCount(10000, {timeout: 30_000})
    renders.push(Date.now() - t0)
    // 一覧を端から端まで 30 ステップでスクロールし、各ステップの描画完了（rAF）までの合計時間を測る
    scrolls.push(await page.evaluate(async () => {
      const list = document.querySelector('[aria-label="セッション一覧"]')!
      let el: Element | null = list
      while (el && el.scrollHeight <= el.clientHeight + 1) el = el.parentElement
      const target = (el ?? document.scrollingElement)!
      const start = performance.now()
      for (let s = 1; s <= 30; s++) {
        target.scrollTop = (target.scrollHeight / 30) * s
        await new Promise((r) => requestAnimationFrame(() => r(null)))
      }
      return performance.now() - start
    }))
  }
  const details: number[] = []
  for (let i = 0; i < 3; i++) {
    await page.goto('/#/sessions')
    const t0 = Date.now()
    await page.getByRole('list', {name: 'セッション一覧'}).locator('li button').first().click()
    await expect(page.getByTestId('message')).toHaveCount(2000, {timeout: 30_000})
    details.push(Date.now() - t0)
  }
  const r = median(renders), s = median(scrolls), d = median(details)
  record({id: 'large-list-render', label: '一覧 10000 件の描画（中央値・3 回）', unit: 'ms', value: r, budget: budgets.largeListRenderMs, detail: renders})
  record({id: 'large-scroll', label: '一覧 10000 件の 30 ステップのスクロール（中央値・3 回）', unit: 'ms', value: round(s), budget: budgets.largeScrollMs, detail: scrolls.map((x) => round(x))})
  record({id: 'large-detail-render', label: '詳細 2000 メッセージの描画（中央値・3 回）', unit: 'ms', value: d, budget: budgets.largeDetailRenderMs, detail: details})
  expect(r).toBeLessThanOrEqual(budgets.largeListRenderMs)
  expect(s).toBeLessThanOrEqual(budgets.largeScrollMs)
  expect(d).toBeLessThanOrEqual(budgets.largeDetailRenderMs)
})

test('画面の健全性: はみ出し数とレイアウトのずれ（CLS）', async ({page}) => {
  await page.addInitScript(() => {
    const w = window as unknown as {__cls: number}
    w.__cls = 0
    new PerformanceObserver((list) => {
      for (const e of list.getEntries() as unknown as {value: number; hadRecentInput: boolean}[]) if (!e.hadRecentInput) w.__cls += e.value
    }).observe({type: 'layout-shift', buffered: true})
  })
  let overflow = 0
  let worstCls = 0
  const perPage: Record<string, number> = {}
  for (const size of [{width: 640, height: 480}, {width: 1024, height: 700}]) {
    await page.setViewportSize(size)
    for (const p of pages) {
      await page.goto(`/#${p.path}`)
      await page.reload()
      await settled(page)
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(300)
      const r = await page.evaluate(() => {
        const de = document.documentElement
        const main = document.querySelector('main')
        const bad = de.scrollWidth > de.clientWidth || de.scrollHeight > de.clientHeight || (!!main && main.scrollWidth > main.clientWidth)
        return {bad, cls: (window as unknown as {__cls: number}).__cls}
      })
      if (r.bad) overflow++
      perPage[`${size.width}x${size.height} ${p.name}`] = round(r.cls, 4)
      worstCls = Math.max(worstCls, r.cls)
    }
  }
  record({id: 'overflow', label: 'はみ出したページ数（9 ページ × 2 サイズ）', unit: '件', value: overflow, budget: budgets.overflowCount})
  record({id: 'cls', label: 'CLS（ページごとの最大）', unit: '', value: round(worstCls, 4), budget: budgets.clsPerPage, detail: perPage})
  expect(overflow).toBeLessThanOrEqual(budgets.overflowCount)
  expect(worstCls).toBeLessThanOrEqual(budgets.clsPerPage)
})

test('アクセシビリティ: axe-core の重大・深刻な違反', async ({page}) => {
  const found: Record<string, string[]> = {}
  let count = 0
  for (const p of pages) {
    await page.goto(`/#${p.path}`)
    await page.reload()
    await settled(page)
    const res = await new AxeBuilder({page}).analyze()
    const serious = res.violations.filter((v) => v.impact === 'critical' || v.impact === 'serious')
    count += serious.reduce((n, v) => n + v.nodes.length, 0)
    if (serious.length) found[p.name] = serious.map((v) => `${v.id}(${v.impact}) x${v.nodes.length}`)
  }
  record({id: 'axe', label: 'axe-core の重大・深刻な違反（要素数・9 ページ）', unit: '件', value: count, budget: budgets.axeSeriousCount, detail: found})
  expect(count, JSON.stringify(found)).toBeLessThanOrEqual(budgets.axeSeriousCount)
})

test('操作のしやすさ: 主要タスクのクリック数', async ({page}) => {
  let clicks = 0
  const click = async (fn: () => Promise<unknown>) => { clicks++; await fn() }
  const nav = (name: string) => page.getByRole('navigation', {name: 'メイン'}).getByRole('link', {name, exact: true})

  await page.goto('/#/')
  await settled(page)

  clicks = 0
  await click(() => nav('セッション').click())
  await click(() => page.getByRole('list', {name: 'セッション一覧'}).locator('li button').first().click())
  await expect(page.getByText('左の一覧から選択してください。')).toHaveCount(0)
  const find = clicks

  clicks = 0
  await click(() => nav('CLAUDE.md 改善').click())
  const sel = page.getByLabel('対象ファイル')
  await expect(sel.locator('option')).not.toHaveCount(1)
  await click(() => sel.selectOption({index: 1}))
  await click(() => page.getByRole('button', {name: '④ 改善案を生成'}).click())
  await click(() => page.getByRole('button', {name: '適用', exact: true}).click())
  await expect(page.getByRole('status')).toContainText('適用しました')
  const apply = clicks

  clicks = 0
  await click(() => nav('変更履歴').click())
  await click(() => page.locator('main li button').first().click())
  await click(() => page.getByRole('button', {name: 'この変更をロールバック'}).click())
  await click(() => page.getByRole('button', {name: 'ロールバックする'}).click())
  await expect(page.getByRole('status')).toBeVisible()
  const rollback = clicks

  record({id: 'clicks-find-session', label: 'セッションを探して開く（クリック数）', unit: '回', value: find, budget: budgets.clicksFindSession})
  record({id: 'clicks-apply', label: '改善案を適用する（クリック・選択数）', unit: '回', value: apply, budget: budgets.clicksApplyProposal})
  record({id: 'clicks-rollback', label: 'ロールバックする（クリック数）', unit: '回', value: rollback, budget: budgets.clicksRollback})
  expect(find).toBeLessThanOrEqual(budgets.clicksFindSession)
  expect(apply).toBeLessThanOrEqual(budgets.clicksApplyProposal)
  expect(rollback).toBeLessThanOrEqual(budgets.clicksRollback)
})

test('操作のしやすさ: キーボードだけでセッションを開ける', async ({page}) => {
  await page.goto('/#/')
  await settled(page)
  let tabs = 0
  // サイドバーの「セッション」リンクまで Tab で到達する
  while (tabs < 30) {
    await page.keyboard.press('Tab')
    tabs++
    const name = await page.evaluate(() => document.activeElement?.textContent?.trim() ?? '')
    if (name === 'セッション') break
  }
  await page.keyboard.press('Enter')
  await expect(page.getByRole('heading', {level: 2, name: 'セッション'})).toBeVisible()
  // 一覧の最初のセッションまで Tab で到達して Enter で開く
  const first = page.getByRole('list', {name: 'セッション一覧'}).locator('li button').first()
  let guard = 0
  while (guard++ < 30 && !(await first.evaluate((el) => el === document.activeElement))) await page.keyboard.press('Tab')
  await page.keyboard.press('Enter')
  await expect(page.getByText('左の一覧から選択してください。')).toHaveCount(0)
  record({id: 'keyboard-tabs', label: 'キーボードで「セッション」リンクに着くまでの Tab 数（開くまで完了できること）', unit: '回', value: tabs, budget: budgets.keyboardTabsToSessions})
  expect(tabs).toBeLessThanOrEqual(budgets.keyboardTabsToSessions)
})
