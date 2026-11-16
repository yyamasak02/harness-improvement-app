import {isUnder} from './paths'

test.each([
  ['unix: 配下', '/home/u/app/CLAUDE.md', '/home/u/app', true],
  ['unix: 兄弟ディレクトリは配下でない', '/home/u/app2/CLAUDE.md', '/home/u/app', false],
  ['unix: 大文字小文字は区別する', '/home/u/App/CLAUDE.md', '/home/u/app', false],
  ['unix: dir 自身は配下でない', '/home/u/app', '/home/u/app', false],
  ['windows: バックスラッシュ', 'C:\\Users\\u\\app\\CLAUDE.md', 'C:\\Users\\u\\app', true],
  ['windows: 区切りが混在', 'C:\\Users\\u\\app/.claude/rules/go.md', 'C:\\Users\\u\\app', true],
  ['windows: 大文字小文字を無視', 'c:\\users\\U\\App\\CLAUDE.md', 'C:\\Users\\u\\app', true],
  ['windows: 末尾の区切りを許容', 'C:\\Users\\u\\app\\CLAUDE.md', 'C:\\Users\\u\\app\\', true],
  ['windows: 兄弟は配下でない', 'C:\\Users\\u\\app2\\CLAUDE.md', 'C:\\Users\\u\\app', false],
  ['空の dir は常に false', '/a/b', '', false],
])('isUnder: %s', (_n, path, dir, want) => {
  expect(isUnder(path, dir)).toBe(want)
})
