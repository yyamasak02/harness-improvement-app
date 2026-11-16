package store

import (
	"database/sql"
	"fmt"
)

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

// Files は取り込み済みファイルの一覧を返す。
func (s *Store) Files() (map[string]FileState, error) {
	rows, err := s.db.Query(`SELECT path,size,mtime_ns,broken_lines FROM files`)
	if err != nil {
		return nil, fmt.Errorf("files の取得: %w", err)
	}
	defer rows.Close()
	out := map[string]FileState{}
	for rows.Next() {
		var f FileState
		if err := rows.Scan(&f.Path, &f.Size, &f.MtimeNs, &f.BrokenLines); err != nil {
			return nil, fmt.Errorf("files の読み取り: %w", err)
		}
		out[f.Path] = f
	}
	return out, rows.Err()
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func deleteByFile(tx *sql.Tx, filePath string) error {
	stmts := []string{
		`DELETE FROM messages WHERE session_id IN (SELECT id FROM sessions WHERE file_path=?)`,
		`DELETE FROM model_usage WHERE session_id IN (SELECT id FROM sessions WHERE file_path=?)`,
		`DELETE FROM sessions WHERE file_path=?`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q, filePath); err != nil {
			return fmt.Errorf("旧データの削除: %w", err)
		}
	}
	return nil
}

// ReplaceFile はファイル由来の既存データを消してから、セッション・メッセージ・使用量・ファイル状態を入れ直す。
func (s *Store) ReplaceFile(f FileState, sess *Session, msgs []Message, usage []ModelUsage) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := deleteByFile(tx, f.Path); err != nil {
		return err
	}
	if sess != nil {
		if _, err := tx.Exec(`DELETE FROM messages WHERE session_id=?`, sess.ID); err != nil {
			return fmt.Errorf("重複セッションの削除: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM model_usage WHERE session_id=?`, sess.ID); err != nil {
			return fmt.Errorf("重複セッションの削除: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM sessions WHERE id=?`, sess.ID); err != nil {
			return fmt.Errorf("重複セッションの削除: %w", err)
		}
		if _, err := tx.Exec(`INSERT INTO sessions(id,project,file_path,cwd,git_branch,started_at,ended_at,user_msgs,assistant_msgs,
			tool_errors,corrections,input_tokens,output_tokens,cache_read,cache_create) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			sess.ID, sess.Project, sess.FilePath, sess.Cwd, sess.GitBranch, sess.StartedAt, sess.EndedAt, sess.UserMsgs,
			sess.AssistantMsgs, sess.ToolErrors, sess.Corrections, sess.InputTokens, sess.OutputTokens, sess.CacheRead,
			sess.CacheCreate); err != nil {
			return fmt.Errorf("セッション挿入: %w", err)
		}
		for _, m := range msgs {
			if _, err := tx.Exec(`INSERT INTO messages(session_id,project,ts,role,kind,tool,text,is_error,is_correction)
				VALUES(?,?,?,?,?,?,?,?,?)`, m.SessionID, m.Project, m.Ts, m.Role, m.Kind, m.Tool, m.Text,
				btoi(m.IsError), btoi(m.IsCorrection)); err != nil {
				return fmt.Errorf("メッセージ挿入: %w", err)
			}
		}
		for _, u := range usage {
			if _, err := tx.Exec(`INSERT INTO model_usage(session_id,model,messages,input_tokens,output_tokens) VALUES(?,?,?,?,?)`,
				u.SessionID, u.Model, u.Messages, u.InputTokens, u.OutputTokens); err != nil {
				return fmt.Errorf("使用量挿入: %w", err)
			}
		}
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO files(path,size,mtime_ns,broken_lines) VALUES(?,?,?,?)`,
		f.Path, f.Size, f.MtimeNs, f.BrokenLines); err != nil {
		return fmt.Errorf("files 更新: %w", err)
	}
	return tx.Commit()
}

// RemoveFile はファイル由来のデータと files 行を消す。
func (s *Store) RemoveFile(path string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := deleteByFile(tx, path); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM files WHERE path=?`, path); err != nil {
		return fmt.Errorf("files 削除: %w", err)
	}
	return tx.Commit()
}

// Reset は全データを消す（再構築用）。
func (s *Store) Reset() error {
	for _, q := range []string{`DELETE FROM messages`, `DELETE FROM model_usage`, `DELETE FROM sessions`, `DELETE FROM files`,
		`INSERT INTO messages_fts(messages_fts) VALUES('rebuild')`} {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("リセット: %w", err)
		}
	}
	return nil
}
