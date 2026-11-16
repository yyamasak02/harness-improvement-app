import {DependencyList, useEffect, useState} from 'react'
import {api} from './api'

export interface AsyncState<V> { data: V | null; error: string; loading: boolean }

/** 非同期関数の結果を state にする。deps が変わると再実行し、古い結果は捨てる。 */
export function useAsync<V>(fn: () => Promise<V>, deps: DependencyList): AsyncState<V> {
  const [state, setState] = useState<AsyncState<V>>({data: null, error: '', loading: true})
  useEffect(() => {
    let alive = true
    setState((s) => ({...s, loading: true, error: ''}))
    fn().then(
      (data) => alive && setState({data, error: '', loading: false}),
      (e: unknown) => alive && setState({data: null, error: e instanceof Error ? e.message : String(e), loading: false}),
    )
    return () => { alive = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)
  return state
}

/** インデックス更新完了のたびに増える数値。依存配列に入れて再取得に使う。 */
export function useIndexVersion(): number {
  const [v, setV] = useState(0)
  useEffect(() => api.onEvent('index:done', () => setV((x) => x + 1)), [])
  return v
}
