import {defineConfig} from '@playwright/test'

// UX 指標の計測用（docs/ux-metrics.md）。時間系の計測が並列実行で乱れないよう workers=1。
// レイアウトの E2E（playwright.config.ts・*.e2e.ts）とは別に、*.ux.ts だけを実行する。
export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.ux.ts',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 120_000,
  reporter: 'list',
  use: {baseURL: 'http://127.0.0.1:4173', browserName: 'chromium'},
  webServer: {
    command: 'npx vite preview --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
  },
})
