import {api, isWails} from './api'

test('Wails ランタイムが無いブラウザ環境では mock を使う', async () => {
  expect(isWails()).toBe(false)
  const o = await api.overview(30)
  expect(o.sessions).toBe(2)
})

test('window.go があれば Wails 環境と判定する', () => {
  ;(window as unknown as {go: unknown}).go = {}
  expect(isWails()).toBe(true)
  delete (window as unknown as {go?: unknown}).go
})
