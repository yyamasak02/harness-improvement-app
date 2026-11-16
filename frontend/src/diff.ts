import {diffLines} from 'diff'

export type RowType = 'same' | 'add' | 'del' | 'change'
export interface DiffRow { type: RowType; left: string | null; right: string | null }

function splitLines(v: string): string[] {
  if (v === '') return []
  const lines = v.split('\n')
  if (lines[lines.length - 1] === '') lines.pop()
  return lines
}

/** 行 diff を左右対応の行に変換する。連続する削除と追加は「変更」行として並べる。 */
export function sideBySide(oldText: string, newText: string): DiffRow[] {
  const parts = diffLines(oldText, newText)
  const rows: DiffRow[] = []
  for (let i = 0; i < parts.length; i++) {
    const p = parts[i]
    if (!p.added && !p.removed) {
      for (const l of splitLines(p.value)) rows.push({type: 'same', left: l, right: l})
      continue
    }
    if (p.removed) {
      const del = splitLines(p.value)
      const next = parts[i + 1]
      const add = next && next.added ? splitLines(next.value) : []
      if (next && next.added) i++
      const n = Math.max(del.length, add.length)
      for (let k = 0; k < n; k++) {
        const l = k < del.length ? del[k] : null
        const r = k < add.length ? add[k] : null
        rows.push({type: l !== null && r !== null ? 'change' : l !== null ? 'del' : 'add', left: l, right: r})
      }
      continue
    }
    for (const l of splitLines(p.value)) rows.push({type: 'add', left: null, right: l})
  }
  return rows
}

export interface DiffStats { added: number; removed: number; changed: number }

export function diffStats(rows: DiffRow[]): DiffStats {
  const s = {added: 0, removed: 0, changed: 0}
  for (const r of rows) {
    if (r.type === 'add') s.added++
    else if (r.type === 'del') s.removed++
    else if (r.type === 'change') s.changed++
  }
  return s
}

export function hasChanges(oldText: string, newText: string): boolean {
  return oldText !== newText
}
