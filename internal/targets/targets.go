// Package targets は改善対象（CLAUDE.md / rules）の発見と書き込み許可判定を行う。
package targets

import (
	"os"
	"path/filepath"
	"sort"
)

// Kind は対象の種別。
type Kind string

const (
	KindGlobal  Kind = "global"
	KindProject Kind = "project"
	KindLocal   Kind = "local"
	KindRule    Kind = "rule"
)

// Target は改善対象ファイル。
type Target struct {
	Kind   Kind   `json:"kind"`
	Path   string `json:"path"`
	Label  string `json:"label"`
	Exists bool   `json:"exists"`
}

func exists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// Global はグローバルの CLAUDE.md（claudeDir は ~/.claude 相当）。
func Global(claudeDir string) Target {
	p := filepath.Join(claudeDir, "CLAUDE.md")
	return Target{Kind: KindGlobal, Path: p, Label: "グローバル CLAUDE.md", Exists: exists(p)}
}

// ForProject は cwd 配下の CLAUDE.md / CLAUDE.local.md / .claude/rules/*.md を返す。
// CLAUDE.md と CLAUDE.local.md は存在しなくても（新規作成対象として）含める。rules は存在するものだけ。
func ForProject(cwd string) []Target {
	if cwd == "" {
		return nil
	}
	p := filepath.Join(cwd, "CLAUDE.md")
	l := filepath.Join(cwd, "CLAUDE.local.md")
	out := []Target{
		{Kind: KindProject, Path: p, Label: "CLAUDE.md", Exists: exists(p)},
		{Kind: KindLocal, Path: l, Label: "CLAUDE.local.md", Exists: exists(l)},
	}
	rules, _ := filepath.Glob(filepath.Join(cwd, ".claude", "rules", "*.md"))
	sort.Strings(rules)
	for _, r := range rules {
		out = append(out, Target{Kind: KindRule, Path: r, Label: ".claude/rules/" + filepath.Base(r), Exists: true})
	}
	return out
}

// Discover はグローバル＋各 cwd の対象を重複なく返す。
func Discover(claudeDir string, cwds []string) []Target {
	out := []Target{Global(claudeDir)}
	seen := map[string]bool{out[0].Path: true}
	for _, c := range cwds {
		for _, t := range ForProject(c) {
			if !seen[t.Path] {
				seen[t.Path] = true
				out = append(out, t)
			}
		}
	}
	return out
}

// Allowed は path が書き込み許可リスト（発見済み対象、または skillsDir 配下）に含まれるかを返す。
func Allowed(found []Target, skillsDir, path string) bool {
	clean := filepath.Clean(path)
	for _, t := range found {
		if filepath.Clean(t.Path) == clean {
			return true
		}
	}
	if skillsDir != "" {
		rel, err := filepath.Rel(filepath.Clean(skillsDir), clean)
		if err == nil && rel != "." && rel != ".." && !startsWithDotDot(rel) {
			return true
		}
	}
	return false
}

func startsWithDotDot(rel string) bool {
	return len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)
}
