import {defineConfig} from '@playwright/test'

// E2E は vite のビルド済み成果物を preview で配信し、mockApi（window.go が無いとき自動で有効）で動かす。
// 実 ~/.claude・実 claude には触れない。
export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  fullyParallel: true,
  retries: 0,
  reporter: 'list',
  use: {baseURL: 'http://127.0.0.1:4173', browserName: 'chromium'},
  webServer: {
    command: 'npx vite preview --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
  },
})
