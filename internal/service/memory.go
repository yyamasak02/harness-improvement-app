package service

import (
	"os"
	"time"

	"harness-app/internal/memory"
	"harness-app/internal/targets"
)

// StaleDays 以上更新されていないメモリを「古い」とみなす。
const StaleDays = 90

// MemoryItem はメモリ 1 件と検出結果。
type MemoryItem struct {
	Target      targets.Target `json:"target"`
	Project     string         `json:"project"`
	Size        int64          `json:"size"`
	ModTime     string         `json:"modTime"`
	Stale       bool           `json:"stale"`
	DuplicateOf []string       `json:"duplicateOf"`
}

// ListMemories はプロジェクトメモリの一覧（読み取りのみ）と、重複・古さの検出結果。
// 編集は ReadTarget → ApplyProposal（applier 経由）で行う。
func (s *Service) ListMemories() ([]MemoryItem, error) {
	ts := targets.Memories(s.env.ClaudeDir)
	files := make([]memory.File, 0, len(ts))
	items := make([]MemoryItem, 0, len(ts))
	for _, t := range ts {
		b, err := os.ReadFile(t.Path)
		if err != nil {
			continue // 消えた・読めないものは一覧から外す
		}
		var mod time.Time
		var size int64
		if st, err := os.Stat(t.Path); err == nil {
			mod, size = st.ModTime(), st.Size()
		}
		files = append(files, memory.File{Path: t.Path, Content: string(b), ModTime: mod})
		items = append(items, MemoryItem{Target: t, Project: projectOf(t.Label), Size: size, ModTime: mod.UTC().Format(time.RFC3339)})
	}
	for i, f := range memory.Analyze(files, s.now(), StaleDays) {
		items[i].Stale, items[i].DuplicateOf = f.Stale, f.DuplicateOf
	}
	return items, nil
}

func projectOf(label string) string {
	for i := 0; i < len(label); i++ {
		if label[i] == '/' {
			return label[:i]
		}
	}
	return label
}
