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
