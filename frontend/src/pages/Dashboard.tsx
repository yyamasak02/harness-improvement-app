import {useState} from 'react'
import {Bar, BarChart, CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts'
import {api} from '../api'
import {Card, ErrorBox, fmt, Kpi, Loading, shortProject} from '../components'
import {useAsync, useIndexVersion} from '../hooks'

const periods = [7, 30, 90]

export default function Dashboard() {
  const [days, setDays] = useState(30)
  const ver = useIndexVersion()
  const {data, error, loading} = useAsync(() => api.overview(days), [days, ver])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold">概要</h2>
        <label className="text-sm">期間：
          <select className="ml-1 rounded border px-2 py-1" value={days} onChange={(e) => setDays(Number(e.target.value))} aria-label="期間">
            {periods.map((p) => <option key={p} value={p}>直近 {p} 日</option>)}
          </select>
        </label>
      </div>
      <ErrorBox message={error} />
      {loading && !data && <Loading />}
      {data && (
        <>
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4 lg:grid-cols-7">
            <Kpi label="セッション" value={fmt(data.sessions)} />
            <Kpi label="メッセージ" value={fmt(data.messages)} />
            <Kpi label="入力トークン" value={fmt(data.inputTokens)} />
            <Kpi label="出力トークン" value={fmt(data.outputTokens)} />
            <Kpi label="訂正" value={fmt(data.corrections)} />
            <Kpi label="ツールエラー" value={fmt(data.toolErrors)} />
            <Kpi label="メモリファイル" value={fmt(data.memoryFiles)} />
          </div>
          <Card title="日別の活動">
            {data.daily?.length ? (
              <div style={{width: '100%', height: 240}}>
                <ResponsiveContainer>
                  <LineChart data={data.daily}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis allowDecimals={false} />
                    <Tooltip />
                    <Legend />
                    <Line type="monotone" dataKey="sessions" name="セッション" stroke="#2563eb" />
                    <Line type="monotone" dataKey="messages" name="メッセージ" stroke="#16a34a" />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            ) : <p className="text-sm text-gray-500">この期間のデータはありません。</p>}
          </Card>
          <div className="grid gap-4 md:grid-cols-2">
            <Card title="モデル別の使用量">
              {data.models?.length ? (
                <div style={{width: '100%', height: 220}}>
                  <ResponsiveContainer>
                    <BarChart data={data.models}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="model" tick={{fontSize: 11}} />
                      <YAxis allowDecimals={false} />
                      <Tooltip />
                      <Bar dataKey="messages" name="応答数" fill="#7c3aed" />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              ) : <p className="text-sm text-gray-500">データなし</p>}
              <table className="mt-2 w-full text-xs">
                <tbody>
                  {data.models?.map((m) => (
                    <tr key={m.model}><td>{m.model}</td><td className="text-right">{fmt(m.messages)} 応答</td>
                      <td className="text-right">{fmt(m.inputTokens + m.outputTokens)} tok</td></tr>
                  ))}
                </tbody>
              </table>
            </Card>
            <Card title="プロジェクト別の分布">
              {data.projects?.length ? (
                <ul className="space-y-1 text-sm">
                  {data.projects.map((p) => (
                    <li key={p.project} className="flex justify-between">
                      <span title={p.cwd}>{shortProject(p.project)}</span><span>{fmt(p.sessions)} セッション</span>
                    </li>
                  ))}
                </ul>
              ) : <p className="text-sm text-gray-500">データなし</p>}
            </Card>
          </div>
        </>
      )}
    </div>
  )
}
