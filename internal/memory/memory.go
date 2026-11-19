// Package memory はプロジェクトメモリの重複・古さの検出を行う（読み取りのみ）。
package memory

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// File は検査対象のメモリ 1 件。
type File struct {
	Path    string
	Content string
	ModTime time.Time
}

// Finding は 1 件の検出結果。
type Finding struct {
	Path        string   `json:"path"`
	Stale       bool     `json:"stale"`
	DuplicateOf []string `json:"duplicateOf"`
}

// SimilarThreshold 以上の行集合の類似度（Jaccard）を重複とみなす。
const SimilarThreshold = 0.8

// body は frontmatter を除いた本文の、空行を除いた正規化済み行集合。
func body(s string) map[string]bool {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				lines = lines[i+1:]
				break
			}
		}
	}
	set := map[string]bool{}
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			set[t] = true
		}
	}
	return set
}

func similarity(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}

func isIndex(p string) bool {
	return filepath.Base(filepath.ToSlash(p)) == "MEMORY.md" || strings.HasSuffix(filepath.ToSlash(p), "/MEMORY.md")
}

// Analyze は重複（本文の類似）と古さ（staleDays 日以上更新が無い）を検出する。
// MEMORY.md（索引）は重複判定の対象外。入力順と同じ順で返す。
func Analyze(files []File, now time.Time, staleDays int) []Finding {
	sets := make([]map[string]bool, len(files))
	for i, f := range files {
		sets[i] = body(f.Content)
	}
	out := make([]Finding, len(files))
	for i, f := range files {
		out[i] = Finding{Path: f.Path, DuplicateOf: []string{}}
		if staleDays > 0 && !f.ModTime.IsZero() && now.Sub(f.ModTime) >= time.Duration(staleDays)*24*time.Hour {
			out[i].Stale = true
		}
		if isIndex(f.Path) {
			continue
		}
		for j, g := range files {
			if i != j && !isIndex(g.Path) && similarity(sets[i], sets[j]) >= SimilarThreshold {
				out[i].DuplicateOf = append(out[i].DuplicateOf, g.Path)
			}
		}
		sort.Strings(out[i].DuplicateOf)
	}
	return out
}
