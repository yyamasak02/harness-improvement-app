// Go 側の JSON 形に合わせた型（wailsjs/go/models.ts の class 型に依存しない）。
export interface MaskPattern { name: string; regex: string }
export interface Config {
  claudePath: string
  model: string
  excludedProjects: string[]
  correctionPatterns: string[]
  maskPatterns: MaskPattern[]
  timeoutSeconds: number
}
export interface Session {
  id: string; project: string; filePath: string; cwd: string; gitBranch: string
  startedAt: string; endedAt: string
  userMsgs: number; assistantMsgs: number; toolErrors: number; corrections: number
  inputTokens: number; outputTokens: number; cacheRead: number; cacheCreate: number
}
export interface Message {
  id: number; sessionId: string; project: string; ts: string; role: string
  kind: 'text' | 'tool_use' | 'tool_result'; tool: string; text: string
  isError: boolean; isCorrection: boolean
}
export interface SessionDetail { session: Session; messages: Message[] }
export interface SearchHit { messageId: number; sessionId: string; project: string; ts: string; role: string; text: string }
export interface DailyPoint { date: string; sessions: number; messages: number }
export interface ModelStat { model: string; messages: number; inputTokens: number; outputTokens: number }
export interface ProjectStat { project: string; cwd: string; sessions: number; lastAt: string }
export interface Overview {
  sessions: number; messages: number; inputTokens: number; outputTokens: number
  corrections: number; toolErrors: number; memoryFiles: number
  daily: DailyPoint[] | null; models: ModelStat[] | null; projects: ProjectStat[] | null
}
export type TargetKind = 'global' | 'project' | 'local' | 'rule' | 'memory'
export interface Target { kind: TargetKind; path: string; label: string; exists: boolean }
export interface MemoryItem {
  target: Target; project: string; size: number; modTime: string; stale: boolean; duplicateOf: string[]
}
export interface TargetContent{ target: Target; content: string; hash: string }
export interface ProposalRequest { targetPath: string; project: string; days: number; instruction: string }
export interface Proposal {
  targetPath: string; baseHash: string; original: string; output: string
  rationale: string; digest: string; jobId: string
}
export interface SkillRequest { name: string; memoryPaths: string[]; project: string; days: number; instruction: string }
export interface SkillFileProposal { path: string; absPath: string; original: string; content: string; baseHash: string }
export interface SkillProposal { name: string; dir: string; files: SkillFileProposal[]; rationale: string; jobId: string }
export interface SkillApplyFile { path: string; content: string; baseHash: string }
export interface Manifest {
  id: string; path: string; op: string; time: string; existed: boolean
  oldHash: string; newHash: string; rollbackOf?: string
}
export interface BackupEntry extends Manifest { before: string; after: string }
export interface ClaudeStatus { found: boolean; path: string; error: string }
export interface IndexStats { Scanned: number; Ingested: number; Skipped: number; Removed: number; BrokenLines: number }
