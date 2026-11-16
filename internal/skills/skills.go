// Package skills は同梱 skill を埋め込み、ジョブディレクトリへ展開する。
package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:assets
var assets embed.FS

// Names は同梱 skill の名前一覧。
func Names() ([]string, error) {
	ents, err := fs.ReadDir(assets, "assets")
	if err != nil {
		return nil, fmt.Errorf("skill 一覧: %w", err)
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// Extract は <dir>/.claude/skills/<name>/ に同梱 skill を展開する。dir はジョブディレクトリ（~/.claude ではない）。
func Extract(dir string) error {
	dest := filepath.Join(dir, ".claude", "skills")
	return fs.WalkDir(assets, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("assets", p)
		if rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := assets.ReadFile(p)
		if err != nil {
			return fmt.Errorf("skill 読み込み %s: %w", p, err)
		}
		if err := os.WriteFile(target, b, 0o644); err != nil {
			return fmt.Errorf("skill 展開 %s: %w", target, err)
		}
		return nil
	})
}
