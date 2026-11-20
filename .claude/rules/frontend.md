---
description: フロントエンド規約
paths: ["frontend/**"]
---
- TypeScript strict。Wails バインディングは `frontend/src/api.ts` に集約し、ブラウザ単体では mock に切り替える。
- UI 文言は日本語。diff は jsdiff で計算し自前描画（Monaco 不可）。
- Tailwind / Recharts / react-markdown / Vitest。外部 CDN を使わない。
- パスの判定は `src/paths.ts`（`isUnder` など）を使う。`/` 固定の文字列比較をしない（Windows の cwd は `C:\...`）。
- 層の依存方向は `docs/architecture.md` §2.4 に従う（`src/arch.test.ts` が検査。許容リストへの追加は禁止）。
