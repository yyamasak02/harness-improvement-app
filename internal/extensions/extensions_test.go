package extensions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	tdClaude = "../../testdata/extensions/claude"
	tdJSON   = "../../testdata/extensions/home/.claude.json" // privacy-allow: testdata 内の相対パス（実ホームではない）
)

func TestScanTestdata(t *testing.T) {
	r := Scan(tdClaude, tdJSON, func(s string) string { return strings.ReplaceAll(s, "abcd1234efgh5678", "[M]") })

	if len(r.MCP) != 4 {
		t.Fatalf("mcp=%+v", r.MCP)
	}
	var stdio, http, proj *MCPServer
	for i := range r.MCP {
		switch r.MCP[i].Name {
		case "demo-stdio":
			stdio = &r.MCP[i]
		case "demo-http":
			http = &r.MCP[i]
		case "proj-srv":
			proj = &r.MCP[i]
		}
	}
	if stdio == nil || stdio.Type != "stdio" || stdio.Command != "demo-mcp" || len(stdio.EnvKeys) != 1 || stdio.EnvKeys[0] != "DEMO_SECRET" {
		t.Errorf("stdio=%+v", stdio)
	}
	if stdio != nil && strings.Contains(strings.Join(stdio.Args, " "), "abcd1234") {
		t.Errorf("mask が適用されていない: %v", stdio.Args)
	}
	if http == nil || http.Type != "http" || http.URL == "" {
		t.Errorf("http=%+v", http)
	}
	if proj == nil || proj.Scope != "project" || proj.Project != "/demo/app" {
		t.Errorf("proj=%+v", proj)
	}

	if len(r.Plugins) != 2 || r.Plugins[0].ID != "demo-plugin@demo-market" || !r.Plugins[0].Enabled || r.Plugins[0].Version != "1.2.0" || r.Plugins[1].Enabled {
		t.Errorf("plugins=%+v", r.Plugins)
	}
	if len(r.Skills) != 2 || r.Skills[0].Name != "demo-skill" || r.Skills[0].Description != "デモ用の skill です" || r.Skills[1].Name != "no-meta" {
		t.Errorf("skills=%+v", r.Skills)
	}
	if len(r.Hooks) != 3 || r.Hooks[0].Event != "PreToolUse" || r.Hooks[0].Matcher != "Bash" || r.Hooks[1].Event != "Stop" {
		t.Errorf("hooks=%+v", r.Hooks)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("warnings=%v", r.Warnings)
	}
}

func TestScanMissingAndBroken(t *testing.T) {
	d := t.TempDir()
	r := Scan(filepath.Join(d, "none"), filepath.Join(d, "none.json"), nil)
	if len(r.MCP)+len(r.Plugins)+len(r.Skills)+len(r.Hooks)+len(r.Warnings) != 0 {
		t.Errorf("存在しないパスは空になる: %+v", r)
	}
	if err := os.WriteFile(filepath.Join(d, "settings.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	r = Scan(d, filepath.Join(d, "none.json"), nil)
	if len(r.Warnings) != 1 || strings.Contains(r.Warnings[0], "broken") {
		t.Errorf("warnings=%v", r.Warnings)
	}
}

func TestFrontmatter(t *testing.T) {
	cases := []struct{ in, name, desc string }{
		{"---\nname: a\ndescription: \"b c\"\n---\nbody", "a", "b c"},
		{"---\r\nname: a\r\n---\r\n", "a", ""},
		{"no front", "", ""},
		{"---\nname: a\n", "", ""},
	}
	for _, c := range cases {
		n, d := frontmatter(c.in)
		if n != c.name || d != c.desc {
			t.Errorf("%q -> %q,%q", c.in, n, d)
		}
	}
}
