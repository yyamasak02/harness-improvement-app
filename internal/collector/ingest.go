package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"harness-app/internal/logmodel"
)

// Store は取り込みが必要とする保存先の機能（利用側の最小のインターフェース）。実体は main/service が結線する。
type Store interface {
	Files() (map[string]logmodel.FileState, error)
	ReplaceFile(f logmodel.FileState, sess *logmodel.Session, msgs []logmodel.Message, usage []logmodel.ModelUsage) error
	RemoveFile(path string) error
}

// Options は取り込み設定。
type Options struct {
	ClaudeDir          string // ~/.claude 相当（注入可能）
	CorrectionPatterns []string
	ExcludedProjects   []string // プロジェクトディレクトリ名 or cwd
	Progress           func(done, total int)
	Force              bool // true なら差分判定を無視して全件取り込む
}

// Stats は取り込み結果。
type Stats struct {
	Scanned, Ingested, Skipped, Removed, BrokenLines int
}

func excluded(opt Options, project, cwd string) bool {
	for _, e := range opt.ExcludedProjects {
		if e != "" && (e == project || e == cwd) {
			return true
		}
	}
	return false
}

// Ingest は <ClaudeDir>/projects/*/*.jsonl を差分取り込みする。読み取り専用。
func Ingest(st Store, opt Options) (Stats, error) {
	var stats Stats
	files, err := filepath.Glob(filepath.Join(opt.ClaudeDir, "projects", "*", "*.jsonl"))
	if err != nil {
		return stats, fmt.Errorf("jsonl の列挙: %w", err)
	}
	sort.Strings(files)
	known, err := st.Files()
	if err != nil {
		return stats, err
	}
	det := NewCorrectionDetector(opt.CorrectionPatterns)
	present := map[string]bool{}
	for i, path := range files {
		present[path] = true
		stats.Scanned++
		if opt.Progress != nil {
			opt.Progress(i, len(files))
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		fs := logmodel.FileState{Path: path, Size: info.Size(), MtimeNs: info.ModTime().UnixNano()}
		if k, ok := known[path]; ok && !opt.Force && k.Size == fs.Size && k.MtimeNs == fs.MtimeNs {
			stats.Skipped++
			continue
		}
		project := filepath.Base(filepath.Dir(path))
		parsed, err := ParseFile(path, project, det)
		if err != nil {
			return stats, fmt.Errorf("%s: %w", path, err)
		}
		fs.BrokenLines = parsed.Broken
		stats.BrokenLines += parsed.Broken
		sess := parsed.Session
		if sess != nil && excluded(opt, project, sess.Cwd) {
			sess = nil
		}
		if err := st.ReplaceFile(fs, sess, parsed.Messages, parsed.Usage); err != nil {
			return stats, fmt.Errorf("%s: %w", path, err)
		}
		stats.Ingested++
	}
	for path := range known {
		if !present[path] {
			if err := st.RemoveFile(path); err != nil {
				return stats, err
			}
			stats.Removed++
		}
	}
	if opt.Progress != nil {
		opt.Progress(len(files), len(files))
	}
	return stats, nil
}
