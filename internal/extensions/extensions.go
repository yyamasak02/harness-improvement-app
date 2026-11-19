// Package extensions は MCP / plugins / skills / hooks の状態を読み取り専用で集める（F7）。
// 書き込み API は一切使わない。パスは呼び出し側が注入する。
package extensions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MCPServer は MCP サーバー 1 件。env・headers の値は秘密を含みうるため読まない（キー名のみ）。
type MCPServer struct {
	Name    string   `json:"name"`
	Scope   string   `json:"scope"` // user | project | settings
	Project string   `json:"project"`
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	URL     string   `json:"url"`
	EnvKeys []string `json:"envKeys"`
}

// Plugin はインストール済み plugin。
type Plugin struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Scope   string `json:"scope"`
	Enabled bool   `json:"enabled"`
}

// Skill はユーザー skill（~/.claude/skills/<name>/SKILL.md）。
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// Hook は settings.json の hooks 1 件。
type Hook struct {
	Event   string `json:"event"`
	Matcher string `json:"matcher"`
	Type    string `json:"type"`
	Command string `json:"command"`
}

// Report は拡張機能の状態一式。
type Report struct {
	MCP     []MCPServer `json:"mcp"`
	Plugins []Plugin    `json:"plugins"`
	Skills  []Skill     `json:"skills"`
	Hooks   []Hook      `json:"hooks"`
	// Warnings は読めなかった・壊れていたファイルの説明（内容は含めない）。
	Warnings []string `json:"warnings"`
}

type rawMCP struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	URL     string            `json:"url"`
	Env     map[string]string `json:"env"`
}

func readJSON(path string, v any) (exists bool, err error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return true, err
	}
	return true, json.Unmarshal(b, v)
}

func toServers(m map[string]rawMCP, scope, project string, mask func(string) string) []MCPServer {
	out := make([]MCPServer, 0, len(m))
	for name, r := range m {
		s := MCPServer{Name: name, Scope: scope, Project: project, Type: r.Type, Command: mask(r.Command), URL: mask(r.URL)}
		if s.Type == "" {
			s.Type = "stdio"
			if r.URL != "" {
				s.Type = "http"
			}
		}
		for _, a := range r.Args {
			s.Args = append(s.Args, mask(a))
		}
		for k := range r.Env {
			s.EnvKeys = append(s.EnvKeys, k)
		}
		sort.Strings(s.EnvKeys)
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Scan は claudeDir（~/.claude 相当）と claudeJSON（~/.claude.json 相当）を読む。
// mask は表示するコマンド・引数・URL に適用する（nil なら無加工）。壊れたファイルは Warnings に記録して続行する。
func Scan(claudeDir, claudeJSON string, mask func(string) string) Report {
	if mask == nil {
		mask = func(s string) string { return s }
	}
	r := Report{MCP: []MCPServer{}, Plugins: []Plugin{}, Skills: []Skill{}, Hooks: []Hook{}, Warnings: []string{}}
	warn := func(name string, err error) {
		if err != nil {
			r.Warnings = append(r.Warnings, name+" を読めませんでした")
		}
	}

	var cj struct {
		MCPServers map[string]rawMCP `json:"mcpServers"`
		Projects   map[string]struct {
			MCPServers map[string]rawMCP `json:"mcpServers"`
		} `json:"projects"`
	}
	_, err := readJSON(claudeJSON, &cj)
	warn(".claude.json", err)
	r.MCP = append(r.MCP, toServers(cj.MCPServers, "user", "", mask)...)
	projs := make([]string, 0, len(cj.Projects))
	for p := range cj.Projects {
		projs = append(projs, p)
	}
	sort.Strings(projs)
	for _, p := range projs {
		r.MCP = append(r.MCP, toServers(cj.Projects[p].MCPServers, "project", p, mask)...)
	}

	var st struct {
		MCPServers     map[string]rawMCP `json:"mcpServers"`
		EnabledPlugins map[string]bool   `json:"enabledPlugins"`
		Hooks          map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	_, err = readJSON(filepath.Join(claudeDir, "settings.json"), &st)
	warn("settings.json", err)
	r.MCP = append(r.MCP, toServers(st.MCPServers, "settings", "", mask)...)
	events := make([]string, 0, len(st.Hooks))
	for e := range st.Hooks {
		events = append(events, e)
	}
	sort.Strings(events)
	for _, e := range events {
		for _, g := range st.Hooks[e] {
			for _, h := range g.Hooks {
				r.Hooks = append(r.Hooks, Hook{Event: e, Matcher: g.Matcher, Type: h.Type, Command: mask(h.Command)})
			}
		}
	}

	var ip struct {
		Plugins map[string][]struct {
			Scope   string `json:"scope"`
			Version string `json:"version"`
		} `json:"plugins"`
	}
	_, err = readJSON(filepath.Join(claudeDir, "plugins", "installed_plugins.json"), &ip)
	warn("installed_plugins.json", err)
	seen := map[string]bool{}
	for id, entries := range ip.Plugins {
		seen[id] = true
		p := Plugin{ID: id, Enabled: st.EnabledPlugins[id]}
		if len(entries) > 0 {
			p.Version, p.Scope = entries[0].Version, entries[0].Scope
		}
		r.Plugins = append(r.Plugins, p)
	}
	for id, en := range st.EnabledPlugins { // 有効化されているが記録の無いもの
		if !seen[id] {
			r.Plugins = append(r.Plugins, Plugin{ID: id, Enabled: en})
		}
	}
	sort.Slice(r.Plugins, func(i, j int) bool { return r.Plugins[i].ID < r.Plugins[j].ID })

	r.Skills = scanSkills(filepath.Join(claudeDir, "skills"), warn)
	return r
}

func scanSkills(dir string, warn func(string, error)) []Skill {
	out := []Skill{}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			warn("skills", err)
		}
		return out
	}
	for _, e := range ents {
		p := filepath.Join(dir, e.Name(), "SKILL.md")
		b, err := os.ReadFile(p)
		if err != nil {
			continue // SKILL.md の無いディレクトリは skill ではない
		}
		name, desc := frontmatter(string(b))
		if name == "" {
			name = e.Name()
		}
		out = append(out, Skill{Name: name, Description: desc, Path: p})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// frontmatter は先頭の --- で囲まれた name / description を単純に取り出す。
func frontmatter(s string) (name, desc string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return "", ""
	}
	rest := s[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", ""
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch strings.TrimSpace(k) {
		case "name":
			name = v
		case "description":
			desc = v
		}
	}
	return name, desc
}
