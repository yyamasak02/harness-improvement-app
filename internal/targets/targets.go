// Package targets は改善対象（CLAUDE.md / rules）の発見と書き込み許可判定を行う。
package targets

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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

// foldCase は大文字小文字を区別しない OS（Windows）かどうか。
var foldCase = runtime.GOOS == "windows"

func samePath(a, b string, fold bool) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if fold {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// SamePath は 2 つのパスが同じ場所を指すか（OS の大文字小文字規則に従う）を返す。
func SamePath(a, b string) bool { return samePath(a, b, foldCase) }

// underDir は path が dir の厳密に配下（dir 自身は含まない）にあるかを返す。
// filepath.Rel は Windows では大文字小文字を無視する等 OS 依存なので、Clean 済みの文字列の接頭辞で判定する。
func underDir(dir, path string, fold bool) bool {
	dir, path = filepath.Clean(dir), filepath.Clean(path) // ".." は Clean が解決する
	if fold {
		dir, path = strings.ToLower(dir), strings.ToLower(path)
	}
	prefix := strings.TrimRight(dir, string(filepath.Separator)) + string(filepath.Separator)
	return len(path) > len(prefix) && strings.HasPrefix(path, prefix)
}

func allowed(found []Target, skillsDir, path string, fold bool) bool {
	for _, t := range found {
		if samePath(t.Path, path, fold) {
			return true
		}
	}
	return skillsDir != "" && underDir(skillsDir, path, fold)
}

// KindMemory はプロジェクトメモリ（~/.claude/projects/<project>/memory/*.md）。
const KindMemory Kind = "memory"

// Memories は claudeDir/projects/*/memory/*.md を発見する。Label は プロジェクト名/ファイル名。
func Memories(claudeDir string) []Target {
	files, _ := filepath.Glob(filepath.Join(claudeDir, "projects", "*", "memory", "*.md"))
	sort.Strings(files)
	out := make([]Target, 0, len(files))
	for _, f := range files {
		// シンボリックリンクは辿らない（許可リスト外のファイルを読み書きさせない）
		if st, err := os.Lstat(f); err != nil || st.Mode()&os.ModeSymlink != 0 {
			continue
		}
		proj := filepath.Base(filepath.Dir(filepath.Dir(f)))
		out = append(out, Target{Kind: KindMemory, Path: f, Label: proj + "/" + filepath.Base(f), Exists: exists(f)})
	}
	return out
}

// Allowed は path が書き込み許可リスト（発見済み対象、または skillsDir 配下）に含まれるかを返す。
func Allowed(found []Target, skillsDir, path string) bool {
	return allowed(found, skillsDir, path, foldCase)
}
