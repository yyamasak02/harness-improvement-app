package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harness-app/internal/applier"
)

func TestE2ESkillGenerateApplyRollback(t *testing.T) {
	f := newFixture(t)
	mem := filepath.Join(f.claudeDir, "projects", "-home-demo-app", "memory", "feedback.md")
	req := SkillRequest{Name: "my-skill", MemoryPaths: []string{mem}, Days: 30}
	sp, err := f.svc.GenerateSkill(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(f.claudeDir, "skills", "my-skill")
	if _, err := os.Stat(skillDir); err == nil {
		t.Fatal("生成しただけでは書き込まれない")
	}
	if len(sp.Files) != 2 || sp.Files[0].Path != "SKILL.md" || sp.Files[1].Path != "knowledge/notes.md" || sp.Files[0].BaseHash != applier.AbsentHash {
		t.Fatalf("%+v", sp.Files)
	}
	if sp.Rationale == "" || sp.Dir != skillDir {
		t.Fatalf("%+v", sp)
	}

	var files []SkillApplyFile
	for _, pf := range sp.Files {
		files = append(files, SkillApplyFile{Path: pf.Path, Content: pf.Content, BaseHash: pf.BaseHash})
	}
	ms, err := f.svc.ApplySkill(sp.Name, files)
	if err != nil || len(ms) != 2 {
		t.Fatalf("%v %v", ms, err)
	}
	if !strings.Contains(read(t, filepath.Join(skillDir, "SKILL.md")), "name: fake-skill") {
		t.Fatal("適用されていない")
	}
	// 同じ古いハッシュ（新規扱い）での再適用は拒否される
	if _, err := f.svc.ApplySkill(sp.Name, files); !errors.Is(err, applier.ErrHashMismatch) {
		t.Fatalf("再適用は拒否: %v", err)
	}
	// ロールバックで新規ファイルは消える
	for _, m := range ms {
		if _, err := f.svc.Rollback(m.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err == nil {
		t.Fatal("ロールバックされていない")
	}
}

func TestSkillApplyRollsBackOnFailure(t *testing.T) {
	f := newFixture(t)
	skillDir := filepath.Join(f.claudeDir, "skills", "partial")
	files := []SkillApplyFile{
		{Path: "SKILL.md", Content: "# a\n", BaseHash: applier.AbsentHash},
		{Path: "knowledge/x.md", Content: "x", BaseHash: "stale-hash"},
	}
	if _, err := f.svc.ApplySkill("partial", files); !errors.Is(err, applier.ErrHashMismatch) {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err == nil {
		t.Fatal("途中失敗時は適用済み分が戻る")
	}
}

func TestSkillRejectsBadInput(t *testing.T) {
	f := newFixture(t)
	ok := SkillApplyFile{Path: "SKILL.md", Content: "x", BaseHash: applier.AbsentHash}
	cases := []struct {
		name  string
		skill string
		files []SkillApplyFile
	}{
		{"名前に ..", "..", []SkillApplyFile{ok}},
		{"名前に区切り", "a/b", []SkillApplyFile{ok}},
		{"大文字", "Bad", []SkillApplyFile{ok}},
		{"空名", "", []SkillApplyFile{ok}},
		{"パス脱出", "s", []SkillApplyFile{ok, {Path: "../../CLAUDE.md", Content: "x", BaseHash: applier.AbsentHash}}},
		{"絶対パス", "s", []SkillApplyFile{ok, {Path: "/etc/x.md", Content: "x"}}},
		{"スクリプト", "s", []SkillApplyFile{ok, {Path: "run.sh", Content: "x"}}},
		{"SKILL.md なし", "s", []SkillApplyFile{{Path: "a.md", Content: "x", BaseHash: applier.AbsentHash}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := f.svc.ApplySkill(c.skill, c.files); err == nil {
				t.Fatal("拒否されるべき")
			}
		})
	}
	if _, err := os.Stat(filepath.Join(f.claudeDir, "skills")); err == nil {
		t.Fatal("何も書き込まれてはいけない")
	}
}

func TestGenerateSkillErrors(t *testing.T) {
	f := newFixture(t)
	mem := filepath.Join(f.claudeDir, "projects", "-home-demo-app", "memory", "feedback.md")
	ctx := context.Background()
	if _, err := f.svc.GenerateSkill(ctx, SkillRequest{Name: "ok-name"}); err == nil {
		t.Error("メモリ未選択は拒否")
	}
	if _, err := f.svc.GenerateSkill(ctx, SkillRequest{Name: "ok-name", MemoryPaths: []string{filepath.Join(f.ws, "secret.md")}}); !errors.Is(err, applier.ErrNotAllowed) {
		t.Errorf("未発見パスは拒否: %v", err)
	}
	if _, err := f.svc.GenerateSkill(ctx, SkillRequest{Name: "Bad Name", MemoryPaths: []string{mem}}); err == nil {
		t.Error("不正な名前は拒否")
	}
	t.Setenv("FAKE_CLAUDE_MODE", "badskill")
	if _, err := f.svc.GenerateSkill(ctx, SkillRequest{Name: "ok-name", MemoryPaths: []string{mem}}); err == nil {
		t.Error("不正な生成物は拒否")
	}
	t.Setenv("FAKE_CLAUDE_MODE", "fail")
	if _, err := f.svc.GenerateSkill(ctx, SkillRequest{Name: "ok-name", MemoryPaths: []string{mem}}); err == nil {
		t.Error("claude 失敗はエラー")
	}
}

func TestSkillInputIsMasked(t *testing.T) {
	f := newFixture(t)
	mem := filepath.Join(f.claudeDir, "projects", "-home-demo-app", "memory", "feedback.md")
	secret := "sk-ant-" + strings.Repeat("a", 30)
	if err := os.WriteFile(mem, []byte("token "+secret+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.GenerateSkill(context.Background(), SkillRequest{Name: "m", MemoryPaths: []string{mem}, Instruction: secret}); err != nil {
		t.Fatal(err)
	}
	var found bool
	_ = filepath.Walk(filepath.Join(f.dataDir, "work"), func(p string, info os.FileInfo, err error) error {
		if err == nil && info.Name() == "input.json" {
			found = true
			if strings.Contains(read(t, p), secret) {
				t.Error("input.json に秘密が含まれている")
			}
		}
		return nil
	})
	if !found {
		t.Fatal("input.json が無い")
	}
}
