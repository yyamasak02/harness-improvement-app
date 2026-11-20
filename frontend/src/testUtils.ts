/** 解決・拒否を手動で制御できる Promise。実行中の表示を確認するテストで使う。 */
export function deferred<V>() {
  let resolve!: (v: V) => void
  let reject!: (e: Error) => void
  const promise = new Promise<V>((res, rej) => { resolve = res; reject = rej })
  return {promise, resolve, reject}
}
