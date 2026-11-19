package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harness-app/internal/applier"
)

func TestMemoryListEditRollback(t *testing.T) {
	f := newFixture(t)
	dir := filepath.Join(f.claudeDir, "projects", "-home-demo-app", "memory")
	// 重複するメモリを足す
	body := read(t, filepath.Join(dir, "feedback.md"))
	dup := filepath.Join(dir, "feedback-copy.md")
	if err := os.WriteFile(dup, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := f.svc.ListMemories()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("items=%d", len(items))
	}
	var fb *MemoryItem
	for i := range items {
		if strings.HasSuffix(items[i].Target.Path, "feedback.md") && !strings.Contains(items[i].Target.Path, "copy") {
			fb = &items[i]
		}
	}
	if fb == nil || len(fb.DuplicateOf) != 1 || fb.DuplicateOf[0] != dup || fb.Project != "-home-demo-app" {
		t.Fatalf("%+v", fb)
	}

	// 閲覧 → applier 経由で編集 → ロールバック
	tc, err := f.svc.ReadTarget(fb.Target.Path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := f.svc.ApplyProposal(fb.Target.Path, tc.Content+"- 追記\n", tc.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(t, fb.Target.Path), "- 追記") {
		t.Fatal("適用されていない")
	}
	if _, err := f.svc.ApplyProposal(fb.Target.Path, "x", tc.Hash); !errors.Is(err, applier.ErrHashMismatch) {
		t.Fatalf("古いハッシュは拒否: %v", err)
	}
	if _, err := f.svc.Rollback(m.ID); err != nil {
		t.Fatal(err)
	}
	if read(t, fb.Target.Path) != tc.Content {
		t.Fatal("ロールバックされていない")
	}
	// 許可リスト外（memory ディレクトリ配下でも .md 以外・発見されない新規ファイル）は拒否
	if _, err := f.svc.ApplyProposal(filepath.Join(dir, "new.txt"), "x", applier.AbsentHash); !errors.Is(err, applier.ErrNotAllowed) {
		t.Fatalf("未発見パスは拒否: %v", err)
	}
}
