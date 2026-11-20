// Package collector は Claude Code のセッションログ(jsonl)を寛容にパースし SQLite に取り込む（読み取り専用）。
package collector

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"harness-app/internal/logmodel"
)

const maxTextRunes = 4000

type rawLine struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	Timestamp string          `json:"timestamp"`
	Cwd       string          `json:"cwd"`
	GitBranch string          `json:"gitBranch"`
	IsMeta    bool            `json:"isMeta"`
	Message   json.RawMessage `json:"message"`
}

type rawMessage struct {
	Role    string          `json:"role"`
	Model   string          `json:"model"`
	ID      string          `json:"id"`
	Content json.RawMessage `json:"content"`
	Usage   *struct {
		Input       int `json:"input_tokens"`
		Output      int `json:"output_tokens"`
		CacheRead   int `json:"cache_read_input_tokens"`
		CacheCreate int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

type rawBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

// Parsed はパース結果。
type Parsed struct {
	Session  *logmodel.Session // メッセージが 1 件も無ければ nil
	Messages []logmodel.Message
	Usage    []logmodel.ModelUsage
	Broken   int
}

func truncate(s string) string {
	if utf8.RuneCountInString(s) <= maxTextRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxTextRunes]) + "…"
}

// textOf は文字列 or ブロック配列(text)からテキストを取り出す。
func textOf(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []rawBlock
	if json.Unmarshal(raw, &blocks) == nil {
		var parts []string
		for _, b := range blocks {
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func isSystemTag(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "<command-") || strings.HasPrefix(t, "<local-command") ||
		strings.HasPrefix(t, "<system-reminder") || strings.HasPrefix(t, "Caveat:")
}

// ParseFile は jsonl を読む。project はプロジェクトディレクトリ名。壊れた行は数えてスキップする。
func ParseFile(path, project string, det *CorrectionDetector) (*Parsed, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("jsonl を開く: %w", err)
	}
	defer f.Close()

	p := &Parsed{}
	sess := logmodel.Session{Project: project, FilePath: path}
	toolNames := map[string]string{}
	type usage struct {
		model           string
		in, out, cr, cc int
	}
	usageByMsg := map[string]usage{}
	var anonUsage []usage
	assistantSeen := map[string]bool{}
	modelCount := map[string]*logmodel.ModelUsage{}

	r := bufio.NewReaderSize(f, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			p.handleLine(line, &sess, toolNames, det, func(m rawMessage, isAssistant bool) {
				if !isAssistant {
					return
				}
				u := usage{model: m.Model}
				if m.Usage != nil {
					u.in, u.out, u.cr, u.cc = m.Usage.Input, m.Usage.Output, m.Usage.CacheRead, m.Usage.CacheCreate
				}
				if m.ID == "" {
					anonUsage = append(anonUsage, u)
					return
				}
				usageByMsg[m.ID] = u // 同一 ID は最後の usage を採用（ストリーミング重複対策）
				assistantSeen[m.ID] = true
			})
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("jsonl 読み込み: %w", err)
		}
	}

	all := anonUsage
	for _, u := range usageByMsg {
		all = append(all, u)
	}
	sess.AssistantMsgs = len(all)
	for _, u := range all {
		sess.InputTokens += u.in
		sess.OutputTokens += u.out
		sess.CacheRead += u.cr
		sess.CacheCreate += u.cc
		model := u.model
		if model == "" {
			model = "unknown"
		}
		mu := modelCount[model]
		if mu == nil {
			mu = &logmodel.ModelUsage{Model: model}
			modelCount[model] = mu
		}
		mu.Messages++
		mu.InputTokens += u.in
		mu.OutputTokens += u.out
	}

	if len(p.Messages) == 0 && sess.AssistantMsgs == 0 {
		return p, nil
	}
	if sess.ID == "" {
		sess.ID = strings.TrimSuffix(baseName(path), ".jsonl")
	}
	for i := range p.Messages {
		p.Messages[i].SessionID = sess.ID
		if p.Messages[i].IsError {
			sess.ToolErrors++
		}
		if p.Messages[i].IsCorrection {
			sess.Corrections++
		}
	}
	for _, mu := range modelCount {
		mu.SessionID = sess.ID
		p.Usage = append(p.Usage, *mu)
	}
	p.Session = &sess
	return p, nil
}

func baseName(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

func (p *Parsed) handleLine(line []byte, sess *logmodel.Session, toolNames map[string]string, det *CorrectionDetector,
	onMsg func(rawMessage, bool)) {
	var l rawLine
	if err := json.Unmarshal(line, &l); err != nil {
		if len(strings.TrimSpace(string(line))) > 0 {
			p.Broken++
		}
		return
	}
	if l.Type != "user" && l.Type != "assistant" {
		return
	}
	if sess.ID == "" && l.SessionID != "" {
		sess.ID = l.SessionID
	}
	if sess.Cwd == "" {
		sess.Cwd = l.Cwd
	}
	if sess.GitBranch == "" {
		sess.GitBranch = l.GitBranch
	}
	if l.Timestamp != "" {
		if sess.StartedAt == "" || l.Timestamp < sess.StartedAt {
			sess.StartedAt = l.Timestamp
		}
		if l.Timestamp > sess.EndedAt {
			sess.EndedAt = l.Timestamp
		}
	}
	var m rawMessage
	if len(l.Message) == 0 || json.Unmarshal(l.Message, &m) != nil {
		return
	}
	role := l.Type
	onMsg(m, role == "assistant")

	add := func(kind, tool, text string, isErr, isCorr bool) {
		p.Messages = append(p.Messages, logmodel.Message{Project: sess.Project, Ts: l.Timestamp, Role: role, Kind: kind,
			Tool: tool, Text: truncate(text), IsError: isErr, IsCorrection: isCorr})
	}

	var s string
	if json.Unmarshal(m.Content, &s) == nil {
		if role == "user" && (l.IsMeta || isSystemTag(s)) {
			return
		}
		if s != "" {
			add("text", "", s, false, role == "user" && det.Match(s))
			if role == "user" {
				sess.UserMsgs++
			}
		}
		return
	}
	var blocks []rawBlock
	if json.Unmarshal(m.Content, &blocks) != nil {
		return
	}
	userText := false
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text == "" {
				continue
			}
			if role == "user" {
				if l.IsMeta || isSystemTag(b.Text) {
					continue
				}
				add("text", "", b.Text, false, det.Match(b.Text))
				userText = true
			} else {
				add("text", "", b.Text, false, false)
			}
		case "tool_use":
			toolNames[b.ID] = b.Name
			add("tool_use", b.Name, toolInputText(b), false, false)
		case "tool_result":
			add("tool_result", toolNames[b.ToolUseID], textOf(b.Content), b.IsError, false)
		}
	}
	if userText {
		sess.UserMsgs++
	}
}

func toolInputText(b rawBlock) string {
	var in struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(b.Input, &in) == nil && in.Command != "" {
		return in.Command
	}
	return string(b.Input)
}
