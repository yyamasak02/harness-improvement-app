package memory

import (
	"reflect"
	"testing"
	"time"
)

func TestAnalyze(t *testing.T) {
	now := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	fm := "---\nname: x\n---\n"
	files := []File{
		{Path: "a/memory/a.md", Content: fm + "- 日本語で答える\n- テストは go test\n", ModTime: now.Add(-24 * time.Hour)},
		{Path: "a/memory/b.md", Content: "---\nname: y\n---\n\n- テストは go test\n- 日本語で答える\n", ModTime: now.Add(-100 * 24 * time.Hour)},
		{Path: "a/memory/c.md", Content: fm + "- 全く別の内容\n", ModTime: now},
		{Path: "a/memory/MEMORY.md", Content: fm + "- 日本語で答える\n- テストは go test\n", ModTime: now},
		{Path: "a/memory/d.md", Content: "", ModTime: time.Time{}},
	}
	dupAB := [][]string{{"a/memory/b.md"}, {"a/memory/a.md"}, {}, {}, {}}
	tests := []struct {
		name      string
		staleDays int
		wantStale []bool
	}{
		{"90日で古い判定", 90, []bool{false, true, false, false, false}},
		{"無効化(0)", 0, []bool{false, false, false, false, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Analyze(files, now, tt.staleDays)
			for i, g := range got {
				if g.Stale != tt.wantStale[i] {
					t.Errorf("%s stale=%v", g.Path, g.Stale)
				}
				if !reflect.DeepEqual(g.DuplicateOf, dupAB[i]) {
					t.Errorf("%s dup=%v want %v", g.Path, g.DuplicateOf, dupAB[i])
				}
			}
		})
	}
}

func TestSimilarity(t *testing.T) {
	a := map[string]bool{"1": true, "2": true, "3": true, "4": true}
	b := map[string]bool{"1": true, "2": true, "3": true, "5": true}
	if similarity(a, b) >= SimilarThreshold {
		t.Fatal("0.6 は重複でない")
	}
	if similarity(a, map[string]bool{}) != 0 {
		t.Fatal("空は 0")
	}
}
