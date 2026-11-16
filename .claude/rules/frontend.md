---
description: フロントエンド規約
paths: ["frontend/**"]
---
- TypeScript strict。Wails バインディングは `frontend/src/api.ts` に集約し、ブラウザ単体では mock に切り替える。
- UI 文言は日本語。diff は jsdiff で計算し自前描画（Monaco 不可）。
- Tailwind / Recharts / react-markdown / Vitest。外部 CDN を使わない。
