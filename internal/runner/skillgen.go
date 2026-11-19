package runner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 生成物の上限（暴走した出力を UI や適用に渡さない）。
const (
	maxSkillFiles    = 20
	maxSkillFileSize = 200 * 1024
)

// MemoryInput は skill-generate に渡すメモリ 1 件（マスク済みの本文）。
type MemoryInput struct {
	Label   string `json:"label"`
	Content string `json:"content"`
}

// SkillInput は skill-generate への入力（input.json）。
type SkillInput struct {
	SkillName   string        `json:"skillName"`
	Memories    []MemoryInput `json:"memories"`
	Digest      string        `json:"digest"`
	Instruction string        `json:"instruction"`
}

// SkillFile は生成された skill のファイル 1 件（Path は skill ディレクトリからの相対パス。常に / 区切り）。
type SkillFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// SkillResult は skill-generate の結果。
type SkillResult struct {
	JobID     string      `json:"jobId"`
	Files     []SkillFile `json:"files"`
	Rationale string      `json:"rationale"`
}

// RunSkillGenerate は skill-generate を実行し、<job>/output/ 配下の生成物を読み取る。ファイルへは書き込まない。
func (r *Runner) RunSkillGenerate(ctx context.Context, in SkillInput) (*SkillResult, error) {
	res, err := r.execute(ctx, skillPrompt, in)
	if err != nil {
		return nil, err
	}
	files, err := collectSkillFiles(filepath.Join(res.JobDir, "output"))
	if err != nil {
		return nil, err
	}
	return &SkillResult{JobID: res.JobID, Files: files, Rationale: readOptional(filepath.Join(res.JobDir, "rationale.md"))}, nil
}

// ValidSkillRelPath は skill 内の相対パスとして安全か（絶対パス・..・.md 以外・隠しファイルを拒否）。
func ValidSkillRelPath(p string) bool {
	if p == "" || strings.ContainsAny(p, `\:`) || !strings.HasSuffix(p, ".md") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." || strings.HasPrefix(seg, ".") {
			return false
		}
	}
	return true
}

func collectSkillFiles(root string) ([]SkillFile, error) {
	var out []SkillFile
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if !d.Type().IsRegular() || !ValidSkillRelPath(rel) {
			return fmt.Errorf("生成物に不正なファイルがあります: %s", rel)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("生成物の読み込み: %w", err)
		}
		if len(b) > maxSkillFileSize {
			return fmt.Errorf("生成物が大きすぎます: %s", rel)
		}
		out = append(out, SkillFile{Path: rel, Content: string(b)})
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errors.New("output/ が生成されませんでした")
	}
	if err != nil {
		return nil, err
	}
	if len(out) == 0 || len(out) > maxSkillFiles {
		return nil, fmt.Errorf("生成物のファイル数が不正です: %d", len(out))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	if !hasPath(out, "SKILL.md") {
		return nil, errors.New("SKILL.md が生成されませんでした")
	}
	return out, nil
}

func hasPath(files []SkillFile, p string) bool {
	for _, f := range files {
		if f.Path == p {
			return true
		}
	}
	return false
}
