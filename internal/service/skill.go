package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"harness-app/internal/applier"
	"harness-app/internal/runner"
	"harness-app/internal/targets"
)

var skillNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// SkillRequest は自己ナレッジ（skill セット）の生成要求。
type SkillRequest struct {
	Name        string   `json:"name"`
	MemoryPaths []string `json:"memoryPaths"`
	Project     string   `json:"project"`
	Days        int      `json:"days"` // 0 なら digest を渡さない
	Instruction string   `json:"instruction"`
}

// SkillFileProposal は生成された skill ファイル 1 件と、適用先の現状。
type SkillFileProposal struct {
	Path     string `json:"path"` // skill ディレクトリからの相対パス（/ 区切り）
	AbsPath  string `json:"absPath"`
	Original string `json:"original"` // 既存内容（新規なら空）
	Content  string `json:"content"`
	BaseHash string `json:"baseHash"` // 生成時点の既存ファイルのハッシュ（新規は applier.AbsentHash）
}

// SkillProposal は生成された skill セット（保存はしない。適用は ApplySkill）。
type SkillProposal struct {
	Name      string              `json:"name"`
	Dir       string              `json:"dir"`
	Files     []SkillFileProposal `json:"files"`
	Rationale string              `json:"rationale"`
	JobID     string              `json:"jobId"`
}

// SkillApplyFile は適用する 1 ファイル（ユーザーが編集した内容を含む）。
type SkillApplyFile struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	BaseHash string `json:"baseHash"`
}

func (s *Service) skillPath(name, rel string) (string, error) {
	if !skillNameRe.MatchString(name) {
		return "", fmt.Errorf("skill 名が不正です（小文字・数字・ハイフンのみ、64 文字まで）: %q", name)
	}
	if !runner.ValidSkillRelPath(rel) {
		return "", fmt.Errorf("skill 内のパスが不正です: %q", rel)
	}
	return filepath.Join(s.skillsDir(), name, filepath.FromSlash(rel)), nil
}

func readFileOrEmpty(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("既存ファイルの読み込み: %w", err)
	}
	return string(b), nil
}

// GenerateSkill は選んだメモリ（と任意で digest）から skill セットを生成する。ファイルは書き換えない。
// メモリ本文・digest・指示は masking を通して渡す。
func (s *Service) GenerateSkill(ctx context.Context, req SkillRequest) (*SkillProposal, error) {
	if _, err := s.skillPath(req.Name, "SKILL.md"); err != nil {
		return nil, err
	}
	if len(req.MemoryPaths) == 0 {
		return nil, errors.New("メモリを 1 つ以上選んでください")
	}
	m, err := s.masker(s.config())
	if err != nil {
		return nil, err
	}
	in := runner.SkillInput{SkillName: req.Name, Instruction: m.Mask(req.Instruction)}
	for _, p := range req.MemoryPaths {
		tc, err := s.ReadTarget(p) // 発見済みの対象だけ読める
		if err != nil {
			return nil, err
		}
		if tc.Target.Kind != targets.KindMemory {
			return nil, fmt.Errorf("生成元にはメモリだけを選べます: %s", p)
		}
		in.Memories = append(in.Memories, runner.MemoryInput{Label: tc.Target.Label, Content: m.Mask(tc.Content)})
	}
	if req.Days > 0 {
		if in.Digest, err = s.BuildDigest(req.Project, req.Days); err != nil {
			return nil, err
		}
	}
	r, rctx, done, err := s.startRunner(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	res, err := r.RunSkillGenerate(rctx, in)
	if err != nil {
		return nil, err
	}
	sp := &SkillProposal{Name: req.Name, Dir: filepath.Join(s.skillsDir(), req.Name), Rationale: res.Rationale, JobID: res.JobID}
	for _, f := range res.Files {
		abs, err := s.skillPath(req.Name, f.Path)
		if err != nil {
			return nil, err
		}
		orig, err := readFileOrEmpty(abs)
		if err != nil {
			return nil, err
		}
		h, err := applier.HashFile(abs)
		if err != nil {
			return nil, err
		}
		sp.Files = append(sp.Files, SkillFileProposal{Path: f.Path, AbsPath: abs, Original: orig, Content: f.Content, BaseHash: h})
	}
	return sp, nil
}

// ApplySkill はユーザーが「適用」を押したときだけ呼ばれる。各ファイルを applier 経由で書き込む。
// 途中で失敗したら、それまでに適用した分をロールバックする。
func (s *Service) ApplySkill(name string, files []SkillApplyFile) ([]applier.Manifest, error) {
	paths := make([]string, len(files))
	hasSkillMd := false
	for i, f := range files {
		p, err := s.skillPath(name, f.Path)
		if err != nil {
			return nil, err
		}
		for _, q := range paths[:i] {
			if targets.SamePath(p, q) { // OS ごとの大文字小文字の扱いに従う
				return nil, fmt.Errorf("適用するファイルのパスが重複しています: %q", f.Path)
			}
		}
		paths[i] = p
		hasSkillMd = hasSkillMd || f.Path == "SKILL.md"
	}
	if !hasSkillMd {
		return nil, errors.New("SKILL.md が含まれていません")
	}
	var done []applier.Manifest
	for i, f := range files {
		m, err := s.applier().Apply(applier.Request{Path: paths[i], Content: f.Content, ExpectedHash: f.BaseHash})
		if err != nil {
			for j := len(done) - 1; j >= 0; j-- {
				if _, rerr := s.applier().Rollback(done[j].ID); rerr != nil {
					err = fmt.Errorf("%w（ロールバックにも失敗: %v）", err, rerr)
				}
			}
			return nil, err
		}
		done = append(done, m)
	}
	return done, nil
}
