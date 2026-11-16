import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    globals: true,
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/main.tsx', 'src/test-setup.ts', 'src/types.ts', 'src/vite-env.d.ts'],
      // 品質ゲート。下げる場合は理由を docs/decisions.md に記録すること。
      thresholds: {statements: 80, branches: 75, lines: 80},
    },
  },
})
