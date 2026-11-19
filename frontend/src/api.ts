// Wails バインディングはここに集約する。ブラウザ単体（npm run dev）では mock に切り替わる。
import * as App from '../wailsjs/go/main/App'
import {EventsOn} from '../wailsjs/runtime/runtime'
import {mockApi} from './mockApi'
import type * as T from './types'

export interface Api {
  settings(): Promise<T.Config>
  saveSettings(c: T.Config): Promise<void>
  reindex(rebuild: boolean): Promise<T.IndexStats>
  overview(days: number): Promise<T.Overview>
  listProjects(): Promise<T.ProjectStat[]>
  listSessions(project: string, days: number, limit: number): Promise<T.Session[]>
  getSession(id: string): Promise<T.SessionDetail>
  search(q: string): Promise<T.SearchHit[]>
  listTargets(): Promise<T.Target[]>
  listMemories(): Promise<T.MemoryItem[]>
  readTarget(path: string): Promise<T.TargetContent>
  buildDigest(project: string, days: number): Promise<string>
  generateProposal(req: T.ProposalRequest): Promise<T.Proposal>
  cancelGeneration(): Promise<void>
  applyProposal(path: string, content: string, baseHash: string): Promise<T.Manifest>
  generateSkill(req: T.SkillRequest): Promise<T.SkillProposal>
  applySkill(name: string, files: T.SkillApplyFile[]): Promise<T.Manifest[]>
  listBackups(): Promise<T.BackupEntry[]>
  getBackup(id: string): Promise<T.BackupEntry>
  rollback(id: string): Promise<T.Manifest>
  checkClaude(): Promise<T.ClaudeStatus>
  openDataDir(): Promise<void>
  /** index:progress / index:done などのイベント購読。解除関数を返す。 */
  onEvent(name: string, cb: (payload: unknown) => void): () => void
}

const cast = <U>(p: Promise<unknown>) => p as Promise<U>

export const wailsApi: Api = {
  settings: () => cast(App.Settings()),
  saveSettings: (c) => App.SaveSettings(c as never),
  reindex: (r) => cast(App.Reindex(r)),
  overview: (d) => cast(App.Overview(d)),
  listProjects: () => cast(App.ListProjects()),
  listSessions: (p, d, l) => cast(App.ListSessions(p, d, l)),
  getSession: (id) => cast(App.GetSession(id)),
  search: (q) => cast(App.Search(q)),
  listTargets: () => cast(App.ListTargets()),
  listMemories: () => cast(App.ListMemories()),
  readTarget: (p) => cast(App.ReadTarget(p)),
  buildDigest: (p, d) => App.BuildDigest(p, d),
  generateProposal: (r) => cast(App.GenerateProposal(r as never)),
  cancelGeneration: () => App.CancelGeneration(),
  applyProposal: (p, c, h) => cast(App.ApplyProposal(p, c, h)),
  generateSkill: (r) => cast(App.GenerateSkill(r as never)),
  applySkill: (n, f) => cast(App.ApplySkill(n, f as never)),
  listBackups: () => cast(App.ListBackups()),
  getBackup: (id) => cast(App.GetBackup(id)),
  rollback: (id) => cast(App.Rollback(id)),
  checkClaude: () => cast(App.CheckClaude()),
  openDataDir: () => App.OpenDataDir(),
  onEvent: (name, cb) => EventsOn(name, cb),
}

/** Wails のランタイムが注入されていれば本物、無ければ mock。 */
export function isWails(): boolean {
  return typeof window !== 'undefined' && !!(window as unknown as {go?: unknown}).go
}

export const api: Api = isWails() ? wailsApi : mockApi()
