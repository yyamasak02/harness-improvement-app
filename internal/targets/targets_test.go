package targets

import (
	"path/filepath"
	"testing"
)

const (
	claudeDir = "../../testdata/claude"
	cwd       = "../../testdata/workspace/app"
)

func TestDiscover(t *testing.T) {
	got := Discover(claudeDir, []string{cwd, cwd, ""})
	kinds := map[Kind]int{}
	for _, g := range got {
		kinds[g.Kind]++
		if !g.Exists {
			t.Errorf("存在するはず: %s", g.Path)
		}
	}
	want := map[Kind]int{KindGlobal: 1, KindProject: 1, KindLocal: 1, KindRule: 1}
	for k, n := range want {
		if kinds[k] != n {
			t.Errorf("%s: got %d want %d", k, kinds[k], n)
		}
	}
	if len(got) != 4 {
		t.Fatalf("重複排除されていない: %d", len(got))
	}
}

func TestMissingFilesListedAsNotExisting(t *testing.T) {
	dir := t.TempDir()
	ts := ForProject(dir)
	if len(ts) != 2 || ts[0].Exists || ts[1].Exists {
		t.Fatalf("%+v", ts)
	}
	if g := Global(dir); g.Exists {
		t.Fatal("存在しないはず")
	}
}

func TestAllowed(t *testing.T) {
	found := Discover(claudeDir, []string{cwd})
	skills := "/home/x/.claude/skills"
	tests := []struct {
		path string
		want bool
	}{
		{filepath.Join(cwd, "CLAUDE.md"), true},
		{filepath.Join(cwd, "sub", "..", "CLAUDE.md"), true},
		{"/home/x/.claude/skills/foo/SKILL.md", true},
		{"/home/x/.claude/skills", false},
		{"/home/x/.claude/skills/../settings.json", false},
		{"/home/x/.claude/settings.json", false},
		{"/etc/passwd", false},
	}
	for _, tt := range tests {
		if got := Allowed(found, skills, tt.path); got != tt.want {
			t.Errorf("%s: got %v want %v", tt.path, got, tt.want)
		}
	}
}

func TestPathComparisonFold(t *testing.T) {
	tests := []struct {
		name       string
		a, b       string
		fold, want bool
	}{
		{"大文字小文字を区別する OS では別", "/a/CLAUDE.md", "/a/claude.md", false, false},
		{"区別しない OS では同じ", "/a/CLAUDE.md", "/a/claude.md", true, true},
		{"Clean 後に同一", "/a/b/../CLAUDE.md", "/a/CLAUDE.md", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := samePath(tt.a, tt.b, tt.fold); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestUnderDirAndAllowedFold(t *testing.T) {
	dir := "/home/x/.claude/skills"
	tests := []struct {
		name       string
		path       string
		fold, want bool
	}{
		{"配下", "/home/x/.claude/skills/a/SKILL.md", false, true},
		{"大文字違いは区別する OS では配下でない", "/home/x/.Claude/Skills/a/SKILL.md", false, false},
		{"大文字違いでも区別しない OS では配下", "/home/x/.Claude/Skills/a/SKILL.md", true, true},
		{"dir 自身は含まない", dir, true, false},
		{"親へ抜ける", "/home/x/.claude/skills/../settings.json", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := underDir(dir, tt.path, tt.fold); got != tt.want {
				t.Fatalf("underDir: got %v want %v", got, tt.want)
			}
			if got := allowed(nil, dir, tt.path, tt.fold); got != tt.want {
				t.Fatalf("allowed: got %v want %v", got, tt.want)
			}
		})
	}
	found := []Target{{Path: "/p/CLAUDE.md"}}
	if !allowed(found, "", "/P/claude.md", true) || allowed(found, "", "/P/claude.md", false) {
		t.Fatal("発見済み対象の比較が OS 規則に従っていない")
	}
	if !SamePath("/a/../b", "/b") {
		t.Fatal("SamePath")
	}
}
