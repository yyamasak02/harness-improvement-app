import {useMemo} from 'react'
import {diffStats, sideBySide} from './diff'
import type {DiffRow} from './diff'

const leftTone: Record<DiffRow['type'], string> = {same: '', add: 'bg-subtle', del: 'bg-danger-soft', change: 'bg-danger-soft'}
const rightTone: Record<DiffRow['type'], string> = {same: '', add: 'bg-success-soft', del: 'bg-subtle', change: 'bg-success-soft'}

interface Props {
  original: string
  proposed: string
  editing: boolean
  onChange: (v: string) => void
}

/** 左右分割の diff。右側は編集モードで textarea になる。 */
export default function DiffView({original, proposed, editing, onChange}: Props) {
  const rows = useMemo(() => sideBySide(original, proposed), [original, proposed])
  const st = diffStats(rows)
  return (
    <div>
      <div className="mb-2 text-xs text-muted" data-testid="diff-stats">
        追加 {st.added} 行 / 削除 {st.removed} 行 / 変更 {st.changed} 行
      </div>
      <div className="grid grid-cols-2 gap-2 font-mono text-xs">
        <div className="overflow-hidden rounded-ctl border bg-surface">
          <div className="border-b bg-subtle px-2 py-1.5 font-sans font-semibold">現在</div>
          <ol aria-label="現在の内容">
            {rows.map((r, i) => (
              <li key={i} className={`whitespace-pre-wrap px-2 ${leftTone[r.type]}`}>{r.left ?? ' '}</li>
            ))}
          </ol>
        </div>
        <div className="overflow-hidden rounded-ctl border bg-surface">
          <div className="border-b bg-subtle px-2 py-1.5 font-sans font-semibold">改善案{editing ? '（編集中）' : ''}</div>
          {editing ? (
            <textarea aria-label="改善案の編集" className="h-96 w-full rounded-none border-0 font-mono text-xs shadow-none" value={proposed} onChange={(e) => onChange(e.target.value)} />
          ) : (
            <ol aria-label="改善案の内容">
              {rows.map((r, i) => (
                <li key={i} className={`whitespace-pre-wrap px-2 ${rightTone[r.type]}`}>{r.right ?? ' '}</li>
              ))}
            </ol>
          )}
        </div>
      </div>
    </div>
  )
}
