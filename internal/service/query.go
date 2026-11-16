package service

import (
	"fmt"
	"path/filepath"
	"time"

	"harness-app/internal/store"
)

func (s *Service) since(days int) string {
	if days <= 0 {
		days = 30
	}
	return s.now().AddDate(0, 0, -days).UTC().Format(time.RFC3339)
}

// Overview は概要ダッシュボード用の集計。
type Overview struct {
	Sessions     int           `json:"sessions"`
	Messages     int           `json:"messages"`
	InputTokens  int           `json:"inputTokens"`
	OutputTokens int           `json:"outputTokens"`
	Corrections  int           `json:"corrections"`
	ToolErrors   int           `json:"toolErrors"`
	MemoryFiles  int           `json:"memoryFiles"`
	Daily        []DailyPoint  `json:"daily"`
	Models       []ModelStat   `json:"models"`
	Projects     []ProjectStat `json:"projects"`
}

// DailyPoint は日別の活動。
type DailyPoint struct {
	Date     string `json:"date"`
	Sessions int    `json:"sessions"`
	Messages int    `json:"messages"`
}

// ModelStat はモデル別使用量。
type ModelStat struct {
	Model        string `json:"model"`
	Messages     int    `json:"messages"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

// ProjectStat はプロジェクト別セッション数。
type ProjectStat struct {
	Project  string `json:"project"`
	Cwd      string `json:"cwd"`
	Sessions int    `json:"sessions"`
	LastAt   string `json:"lastAt"`
}

// Overview は期間（日数）の集計を返す。
func (s *Service) Overview(days int) (Overview, error) {
	since := s.since(days)
	db := s.st.DB()
	var o Overview
	if err := db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(user_msgs+assistant_msgs),0),COALESCE(SUM(input_tokens),0),
		COALESCE(SUM(output_tokens),0),COALESCE(SUM(corrections),0),COALESCE(SUM(tool_errors),0)
		FROM sessions WHERE started_at>=?`, since).
		Scan(&o.Sessions, &o.Messages, &o.InputTokens, &o.OutputTokens, &o.Corrections, &o.ToolErrors); err != nil {
		return o, fmt.Errorf("KPI 集計: %w", err)
	}
	rows, err := db.Query(`SELECT substr(started_at,1,10) d, COUNT(*), COALESCE(SUM(user_msgs+assistant_msgs),0)
		FROM sessions WHERE started_at>=? GROUP BY d ORDER BY d`, since)
	if err != nil {
		return o, fmt.Errorf("日別集計: %w", err)
	}
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Sessions, &p.Messages); err != nil {
			rows.Close()
			return o, fmt.Errorf("日別の読み取り: %w", err)
		}
		o.Daily = append(o.Daily, p)
	}
	rows.Close()
	rows, err = db.Query(`SELECT mu.model, SUM(mu.messages), SUM(mu.input_tokens), SUM(mu.output_tokens)
		FROM model_usage mu JOIN sessions s ON s.id=mu.session_id WHERE s.started_at>=?
		GROUP BY mu.model ORDER BY SUM(mu.messages) DESC`, since)
	if err != nil {
		return o, fmt.Errorf("モデル別集計: %w", err)
	}
	for rows.Next() {
		var m ModelStat
		if err := rows.Scan(&m.Model, &m.Messages, &m.InputTokens, &m.OutputTokens); err != nil {
			rows.Close()
			return o, fmt.Errorf("モデル別の読み取り: %w", err)
		}
		o.Models = append(o.Models, m)
	}
	rows.Close()
	ps, err := s.projects(since)
	if err != nil {
		return o, err
	}
	o.Projects = ps
	mem, _ := filepath.Glob(filepath.Join(s.env.ClaudeDir, "projects", "*", "memory", "*.md"))
	o.MemoryFiles = len(mem)
	return o, nil
}

func (s *Service) projects(since string) ([]ProjectStat, error) {
	rows, err := s.st.DB().Query(`SELECT project, COALESCE(MAX(cwd),''), COUNT(*), COALESCE(MAX(started_at),'')
		FROM sessions WHERE started_at>=? GROUP BY project ORDER BY COUNT(*) DESC, project`, since)
	if err != nil {
		return nil, fmt.Errorf("プロジェクト集計: %w", err)
	}
	defer rows.Close()
	var out []ProjectStat
	for rows.Next() {
		var p ProjectStat
		if err := rows.Scan(&p.Project, &p.Cwd, &p.Sessions, &p.LastAt); err != nil {
			return nil, fmt.Errorf("プロジェクトの読み取り: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListProjects は全期間のプロジェクト一覧。
func (s *Service) ListProjects() ([]ProjectStat, error) { return s.projects("") }

// ListSessions はプロジェクト（空なら全て）・期間（0 なら全期間）で絞った一覧（新しい順）。
func (s *Service) ListSessions(project string, days, limit int) ([]store.Session, error) {
	since := ""
	if days > 0 {
		since = s.since(days)
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.st.DB().Query(`SELECT id,project,file_path,cwd,git_branch,started_at,ended_at,user_msgs,assistant_msgs,
		tool_errors,corrections,input_tokens,output_tokens,cache_read,cache_create FROM sessions
		WHERE started_at>=? AND (?='' OR project=?) ORDER BY started_at DESC LIMIT ?`, since, project, project, limit)
	if err != nil {
		return nil, fmt.Errorf("セッション一覧: %w", err)
	}
	defer rows.Close()
	var out []store.Session
	for rows.Next() {
		var x store.Session
		if err := rows.Scan(&x.ID, &x.Project, &x.FilePath, &x.Cwd, &x.GitBranch, &x.StartedAt, &x.EndedAt, &x.UserMsgs,
			&x.AssistantMsgs, &x.ToolErrors, &x.Corrections, &x.InputTokens, &x.OutputTokens, &x.CacheRead, &x.CacheCreate); err != nil {
			return nil, fmt.Errorf("セッションの読み取り: %w", err)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// SessionDetail はセッションと時系列メッセージ。
type SessionDetail struct {
	Session  store.Session   `json:"session"`
	Messages []store.Message `json:"messages"`
}

// GetSession はセッション詳細。
func (s *Service) GetSession(id string) (SessionDetail, error) {
	var d SessionDetail
	row := s.st.DB().QueryRow(`SELECT id,project,file_path,cwd,git_branch,started_at,ended_at,user_msgs,assistant_msgs,
		tool_errors,corrections,input_tokens,output_tokens,cache_read,cache_create FROM sessions WHERE id=?`, id)
	x := &d.Session
	if err := row.Scan(&x.ID, &x.Project, &x.FilePath, &x.Cwd, &x.GitBranch, &x.StartedAt, &x.EndedAt, &x.UserMsgs,
		&x.AssistantMsgs, &x.ToolErrors, &x.Corrections, &x.InputTokens, &x.OutputTokens, &x.CacheRead, &x.CacheCreate); err != nil {
		return d, fmt.Errorf("セッション %q: %w", id, err)
	}
	rows, err := s.st.DB().Query(`SELECT id,session_id,project,ts,role,kind,tool,text,is_error,is_correction
		FROM messages WHERE session_id=? ORDER BY id`, id)
	if err != nil {
		return d, fmt.Errorf("メッセージ取得: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m store.Message
		var e, c int
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Project, &m.Ts, &m.Role, &m.Kind, &m.Tool, &m.Text, &e, &c); err != nil {
			return d, fmt.Errorf("メッセージの読み取り: %w", err)
		}
		m.IsError, m.IsCorrection = e == 1, c == 1
		d.Messages = append(d.Messages, m)
	}
	return d, rows.Err()
}

// Search は全文検索。
func (s *Service) Search(q string) ([]store.SearchHit, error) { return s.st.Search(q, 50) }
