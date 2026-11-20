// Wails バインディングはここに集約する。ブラウザ単体（npm run dev）では mock に切り替わる。
import * as App from '../wailsjs/go/main/App'
import {EventsOn} from '../wailsjs/runtime/runtime'
import {mockApi} from './mockApi'
import type * as T from './types'

export interface Api {
  settings(): Promise<T.Config>
  settingsStatus(): Promise<T.SettingsStatus>
  saveSettings(c: T.Config): Promise<void>
  reindex(rebuild: boolean): Promise<T.IndexStats>
  overview(days: number): Promise<T.Overview>
  listProjects(): Promise<T.ProjectStat[]>
  listSessions(project: string, days: number, limit: number): Promise<T.Session[]>
  getSession(id: string): Promise<T.SessionDetail>
  search(q: string): Promise<T.SearchHit[]>
  listTargets(): Promise<T.Target[]>
  listMemories(): Promise<T.MemoryItem[]>
  listExtensions(): Promise<T.ExtensionsReport>
  readTarget(path: string): Promise<T.TargetContent>
  autoSuggestStatus(): Promise<T.AutoSuggestStatus>
  /** 自動生成された未確認の提案（無ければ null）。適用はしない。 */
  autoSuggestion(): Promise<T.Proposal | null>
  dismissAutoSuggestion(): Promise<void>
  buildDigest(project: string, days: number): Promise<string>
  generateProposal(req: T.ProposalRequest): Promise<T.Proposal>
  cancelGeneration(): Promise<void>
  applyProposal(path: string, content: string, baseHash: string): Promise<T.Manifest>
  generateSkill(req: T.SkillRequest): Promise<T.SkillProposal>
  applySkill(name: string, files: T.SkillApplyFile[]): Promise<T.Manifest[]>
  generateReport(date: string): Promise<T.ReportInfo>
  listReports(): Promise<T.ReportInfo[]>
  readReport(date: string): Promise<string>
  launchdPlist(): Promise<string>
  listBackups(): Promise<T.BackupEntry[]>
  getBackup(id: string): Promise<T.BackupEntry>
  rollback(id: string): Promise<T.Manifest>
  checkClaude(): Promise<T.ClaudeStatus>
  openDataDir(): Promise<void>
  /** アプリのバージョン（wails.json 由来。通信なし）。 */
  version(): Promise<string>
  /** index:progress / index:done などのイベント購読。解除関数を返す。 */
  onEvent(name: string, cb: (payload: unknown) => void): () => void
}

const cast = <U>(p: Promise<unknown>) => p as Promise<U>

export const wailsApi: Api = {
  settings: () => cast(App.Settings()),
  settingsStatus: () => cast(App.SettingsStatus()),
  saveSettings: (c) => App.SaveSettings(c as never),
  reindex: (r) => cast(App.Reindex(r)),
  overview: (d) => cast(App.Overview(d)),
  listProjects: () => cast(App.ListProjects()),
  listSessions: (p, d, l) => cast(App.ListSessions(p, d, l)),
  getSession: (id) => cast(App.GetSession(id)),
  search: (q) => cast(App.Search(q)),
  listTargets: () => cast(App.ListTargets()),
  listMemories: () => cast(App.ListMemories()),
  listExtensions: () => cast(App.ListExtensions()),
  readTarget: (p) => cast(App.ReadTarget(p)),
  autoSuggestStatus: () => cast(App.AutoSuggestStatus()),
  autoSuggestion: () => cast(App.AutoSuggestion()),
  dismissAutoSuggestion: () => App.DismissAutoSuggestion(),
  buildDigest: (p, d) => App.BuildDigest(p, d),
  generateProposal: (r) => cast(App.GenerateProposal(r as never)),
  cancelGeneration: () => App.CancelGeneration(),
  applyProposal: (p, c, h) => cast(App.ApplyProposal(p, c, h)),
  generateSkill: (r) => cast(App.GenerateSkill(r as never)),
  applySkill: (n, f) => cast(App.ApplySkill(n, f as never)),
  generateReport: (d) => cast(App.GenerateReport(d)),
  listReports: () => cast(App.ListReports()),
  readReport: (d) => App.ReadReport(d),
  launchdPlist: () => App.LaunchdPlist(),
  listBackups: () => cast(App.ListBackups()),
  getBackup: (id) => cast(App.GetBackup(id)),
  rollback: (id) => cast(App.Rollback(id)),
  checkClaude: () => cast(App.CheckClaude()),
  openDataDir: () => App.OpenDataDir(),
  version: () => App.Version(),
  onEvent: (name, cb) => EventsOn(name, cb),
}

/** Wails のランタイムが注入されていれば本物、無ければ mock。 */
export function isWails(): boolean {
  return typeof window !== 'undefined' && !!(window as unknown as {go?: unknown}).go
}

export const api: Api = isWails() ? wailsApi : mockApi()
