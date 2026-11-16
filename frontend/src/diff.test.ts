import {diffStats, sideBySide} from './diff'

test.each([
  ['同一', 'a\nb\n', 'a\nb\n', {added: 0, removed: 0, changed: 0}],
  ['追加', 'a\n', 'a\nb\n', {added: 1, removed: 0, changed: 0}],
  ['削除', 'a\nb\n', 'a\n', {added: 0, removed: 1, changed: 0}],
  ['変更', 'a\nb\nc\n', 'a\nB\nc\n', {added: 0, removed: 0, changed: 1}],
  ['空から', '', 'x\ny\n', {added: 2, removed: 0, changed: 0}],
  ['空へ', 'x\n', '', {added: 0, removed: 1, changed: 0}],
  ['1行→2行', 'a\n', 'b\nc\n', {added: 1, removed: 0, changed: 1}],
])('diffStats: %s', (_n, a, b, want) => {
  expect(diffStats(sideBySide(a, b))).toEqual(want)
})

test('左右の行が対応する', () => {
  const rows = sideBySide('a\nb\n', 'a\nX\nc\n')
  expect(rows[0]).toEqual({type: 'same', left: 'a', right: 'a'})
  expect(rows[1]).toEqual({type: 'change', left: 'b', right: 'X'})
  expect(rows[2]).toEqual({type: 'add', left: null, right: 'c'})
})

test('末尾改行なしでも動く', () => {
  expect(diffStats(sideBySide('a', 'a\nb'))).toEqual({added: 1, removed: 0, changed: 1})
})
