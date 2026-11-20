// docs/architecture.md のフロントの層（依存は下位層にのみ）を静的に検査する。
// ソースを文字列として読み import を解析するだけで、新規依存は不要。
import {describe, expect, it} from 'vitest'

const sources = import.meta.glob<string>(['./**/*.{ts,tsx}', '!./**/*.test.{ts,tsx}', '!./testUtils.ts', '!./test-setup.ts', '!./vite-env.d.ts'], {
  query: '?raw',
  import: 'default',
  eager: true,
})

// 層（拡張子なしの src 相対パス）。import は「自分より小さい層」にのみ許される。
export const layerOf = (mod: string): number | undefined => {
  if (mod.startsWith('pages/')) return 4
  return ({types: 0, paths: 0, diff: 0, mockApi: 1, api: 2, hooks: 3, components: 3, DiffView: 3, App: 5, main: 6} as Record<string, number>)[mod]
}

// 現状の違反 "from -> to" と理由。増やすのは禁止（上限は ALLOWLIST_CAP で固定。減らしたら同じだけ下げる）。目標は 0。
const ALLOWLIST: Record<string, string> = {
  'mockApi -> api': 'mockApi が Api 型を api.ts から import する（型のみ。api.ts も mockApi を import する循環）。型を types.ts へ移して解消（移行計画 F1）',
}
const ALLOWLIST_CAP = 1

// import 文（型 import・副作用 import・複数行）から相対パス指定子を取り出す。
export const relativeSpecifiers = (code: string): string[] =>
  [...code.matchAll(/(?:^|\n)\s*(?:import|export)\s[^'"]*?from\s*['"](\.[^'"]*)['"]|(?:^|\n)\s*import\s*['"](\.[^'"]*)['"]/g)].map((m) => m[1] ?? m[2])

const normalize = (fromMod: string, spec: string): string => {
  const parts = fromMod.split('/').slice(0, -1)
  for (const seg of spec.split('/')) {
    if (seg === '.' || seg === '') continue
    if (seg === '..') parts.pop()
    else parts.push(seg)
  }
  return parts.join('/')
}

const modules = Object.entries(sources).map(([path, code]) => ({mod: path.replace(/^\.\//, '').replace(/\.(ts|tsx)$/, ''), code}))

const collect = () => {
  const violations = new Set<string>()
  const problems: string[] = []
  for (const {mod, code} of modules) {
    const lf = layerOf(mod)
    if (lf === undefined) {
      problems.push(`層が未定義のモジュール: ${mod}（arch.test.ts の layerOf に追加する）`)
      continue
    }
    for (const spec of relativeSpecifiers(code)) {
      const target = normalize(mod, spec)
      if (/\.(css|svg|png)$/.test(target)) continue
      if (target.startsWith('wailsjs/')) {
        if (mod !== 'api') problems.push(`${mod} が wailsjs を直接 import している（api.ts に集約）`)
        continue
      }
      const lt = layerOf(target)
      if (lt === undefined) {
        problems.push(`層が未定義の import 先: ${mod} -> ${target}`)
      } else if (lt >= lf) {
        violations.add(`${mod} -> ${target}`)
      }
    }
  }
  return {violations, problems}
}

describe('フロントの層の依存方向', () => {
  it('層の規則に違反する import がない（許容リスト以外）', () => {
    const {violations, problems} = collect()
    for (const v of violations) if (!(v in ALLOWLIST)) problems.push(`層の規則違反（許容リストへの追加は禁止。依存の向きを直す）: ${v}`)
    for (const v of Object.keys(ALLOWLIST)) if (!violations.has(v)) problems.push(`許容リストに残っているが、もう違反ではない（リストと ALLOWLIST_CAP から削除する）: ${v}`)
    expect(problems).toEqual([])
  })

  it('許容リストの件数は固定上限と一致し、すべて理由がある', () => {
    expect(Object.keys(ALLOWLIST).length).toBe(ALLOWLIST_CAP)
    for (const reason of Object.values(ALLOWLIST)) expect(reason.trim()).not.toBe('')
  })

  it('モジュールを実際に読めている（検査が空振りしない）', () => {
    expect(modules.length).toBeGreaterThan(10)
  })

  it('import 指定子の抽出（型 import・複数行・副作用 import）', () => {
    const code = "import type {A} from './a'\nimport {\n  B,\n  C,\n} from '../b'\nimport './side'\nimport x from 'react'\nexport {D} from './d'\n"
    expect(relativeSpecifiers(code)).toEqual(['./a', '../b', './side', './d'])
  })

  it('検査が逆流を検出できる（下位→上位は違反）', () => {
    expect(layerOf('hooks')! >= layerOf('pages/Dashboard')!).toBe(false)
    expect(layerOf('pages/A')! >= layerOf('pages/B')!).toBe(true)
    expect(normalize('pages/Improve', '../api')).toBe('api')
  })
})
