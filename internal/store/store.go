// Package store は SQLite のスキーマとクエリを提供する。
package store

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode/utf8"

	_ "modernc.org/sqlite" // pure Go driver
)

// Store は索引 DB。
type Store struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS files(path TEXT PRIMARY KEY, size INTEGER, mtime_ns INTEGER, broken_lines INTEGER);
CREATE TABLE IF NOT EXISTS sessions(id TEXT PRIMARY KEY, project TEXT, file_path TEXT, cwd TEXT, git_branch TEXT,
  started_at TEXT, ended_at TEXT, user_msgs INT, assistant_msgs INT, tool_errors INT, corrections INT,
  input_tokens INT, output_tokens INT, cache_read INT, cache_create INT);
CREATE TABLE IF NOT EXISTS model_usage(session_id TEXT, model TEXT, messages INT, input_tokens INT, output_tokens INT);
CREATE TABLE IF NOT EXISTS messages(id INTEGER PRIMARY KEY, session_id TEXT, project TEXT, ts TEXT, role TEXT,
  kind TEXT, tool TEXT, text TEXT, is_error INT, is_correction INT);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at);
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(text, content='messages', content_rowid='id', tokenize='trigram');
CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
  INSERT INTO messages_fts(rowid, text) VALUES (new.id, new.text);
END;
CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, text) VALUES('delete', old.id, old.text);
END;
`

// Open は DB を開きマイグレーションする。path が ":memory:" ならメモリ DB。
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("DB を開く: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("マイグレーション: %w", err)
	}
	return &Store{db: db}, nil
}

// Close は DB を閉じる。
func (s *Store) Close() error { return s.db.Close() }

// DB は生の *sql.DB を返す（同パッケージ外のクエリ用）。
func (s *Store) DB() *sql.DB { return s.db }

// SearchHit は検索結果の 1 件。
type SearchHit struct {
	MessageID int64  `json:"messageId"`
	SessionID string `json:"sessionId"`
	Project   string `json:"project"`
	Ts        string `json:"ts"`
	Role      string `json:"role"`
	Text      string `json:"text"`
}

// Search は全文検索する。3 文字未満は LIKE にフォールバックする。
func (s *Store) Search(query string, limit int) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if utf8.RuneCountInString(query) < 3 {
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
		rows, err = s.db.Query(`SELECT id, session_id, project, ts, role, text FROM messages
			WHERE text LIKE ? ESCAPE '\' ORDER BY ts DESC LIMIT ?`, "%"+esc+"%", limit)
	} else {
		phrase := `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
		rows, err = s.db.Query(`SELECT m.id, m.session_id, m.project, m.ts, m.role, m.text
			FROM messages_fts f JOIN messages m ON m.id = f.rowid
			WHERE messages_fts MATCH ? ORDER BY m.ts DESC LIMIT ?`, phrase, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("検索: %w", err)
	}
	defer rows.Close()
	var out []SearchHit
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.MessageID, &h.SessionID, &h.Project, &h.Ts, &h.Role, &h.Text); err != nil {
			return nil, fmt.Errorf("検索結果の読み取り: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
