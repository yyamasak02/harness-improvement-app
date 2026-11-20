-- バージョン導入前（PRAGMA user_version = 0）の索引 DB を再現するダミー。個人情報・実ログは含まない。
CREATE TABLE files(path TEXT PRIMARY KEY, size INTEGER, mtime_ns INTEGER, broken_lines INTEGER);
CREATE TABLE sessions(id TEXT PRIMARY KEY, project TEXT, file_path TEXT, cwd TEXT, git_branch TEXT,
  started_at TEXT, ended_at TEXT, user_msgs INT, assistant_msgs INT, tool_errors INT, corrections INT,
  input_tokens INT, output_tokens INT, cache_read INT, cache_create INT);
CREATE TABLE model_usage(session_id TEXT, model TEXT, messages INT, input_tokens INT, output_tokens INT);
CREATE TABLE messages(id INTEGER PRIMARY KEY, session_id TEXT, project TEXT, ts TEXT, role TEXT,
  kind TEXT, tool TEXT, text TEXT, is_error INT, is_correction INT);
CREATE INDEX idx_messages_session ON messages(session_id);
CREATE INDEX idx_sessions_project ON sessions(project);
CREATE INDEX idx_sessions_started ON sessions(started_at);
CREATE VIRTUAL TABLE messages_fts USING fts5(text, content='messages', content_rowid='id', tokenize='trigram');
CREATE TRIGGER messages_ai AFTER INSERT ON messages BEGIN
  INSERT INTO messages_fts(rowid, text) VALUES (new.id, new.text);
END;
CREATE TRIGGER messages_ad AFTER DELETE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, text) VALUES('delete', old.id, old.text);
END;
INSERT INTO messages(id, session_id, project, ts, role, kind, text) VALUES (1, 's-demo', '-home-demo-app', '2026-11-01T00:00:00Z', 'user', 'text', 'デモの古い形式のメッセージ upgradeprobe');
