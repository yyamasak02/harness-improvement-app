import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'
import {writeFileSync} from 'node:fs'
import {resolve} from 'node:path'

// ビルドは dist を空にするが、main.go の go:embed が dist の存在を要求する。
// クリーンな clone で `go vet` が通るよう、プレースホルダーを毎回作り直す。
const keepDist = () => ({
  name: 'keep-dist-gitkeep',
  closeBundle() { writeFileSync(resolve(__dirname, 'dist/.gitkeep'), '') },
})

export default defineConfig({
  plugins: [react(), keepDist()],
  test: {
    exclude: ['e2e/**', 'node_modules/**', 'dist/**'],
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
