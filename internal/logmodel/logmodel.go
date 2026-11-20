// Package logmodel は、セッションログの取り込み結果を表す値の型（collector が作り、store が保存する）。
// 依存の向きを逆転するため、どちらにも属さない基盤層に置く。
package logmodel

// Session は 1 セッションの集計。
type Session struct {
	ID            string `json:"id"`
	Project       string `json:"project"`
	FilePath      string `json:"filePath"`
	Cwd           string `json:"cwd"`
	GitBranch     string `json:"gitBranch"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt"`
	UserMsgs      int    `json:"userMsgs"`
	AssistantMsgs int    `json:"assistantMsgs"`
	ToolErrors    int    `json:"toolErrors"`
	Corrections   int    `json:"corrections"`
	InputTokens   int    `json:"inputTokens"`
	OutputTokens  int    `json:"outputTokens"`
	CacheRead     int    `json:"cacheRead"`
	CacheCreate   int    `json:"cacheCreate"`
}

// Message は 1 メッセージ（ブロック単位）。
type Message struct {
	ID           int64  `json:"id"`
	SessionID    string `json:"sessionId"`
	Project      string `json:"project"`
	Ts           string `json:"ts"`
	Role         string `json:"role"`
	Kind         string `json:"kind"` // text | tool_use | tool_result
	Tool         string `json:"tool"`
	Text         string `json:"text"`
	IsError      bool   `json:"isError"`
	IsCorrection bool   `json:"isCorrection"`
}

// ModelUsage はセッション内のモデル別使用量。
type ModelUsage struct {
	SessionID    string `json:"sessionId"`
	Model        string `json:"model"`
	Messages     int    `json:"messages"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

// FileState は取り込み済みファイルの状態。
type FileState struct {
	Path        string
	Size        int64
	MtimeNs     int64
	BrokenLines int
}
