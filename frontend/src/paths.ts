// OS をまたぐパス判定。cwd は jsonl に記録された元の OS の形式（/Users/x/app または C:\Users\x\app）。

const isWindowsPath = (p: string) => /^[A-Za-z]:[\\/]/.test(p) || p.startsWith('\\\\')

const normalize = (p: string) => p.replace(/\\/g, '/').replace(/\/+$/, '')

/** path が dir の配下にあるか。Windows 形式のパスは区切り文字と大文字小文字を無視して比較する。 */
export function isUnder(path: string, dir: string): boolean {
  if (!dir) return false
  let a = normalize(path)
  let d = normalize(dir)
  if (isWindowsPath(dir) || isWindowsPath(path)) {
    a = a.toLowerCase()
    d = d.toLowerCase()
  }
  return a.startsWith(d + '/')
}
