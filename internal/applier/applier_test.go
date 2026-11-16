package applier

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newApplier(t *testing.T, allowed ...string) (*Applier, string) {
	t.Helper()
	root := t.TempDir()
	tick := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	a := &Applier{
		DataDir: filepath.Join(root, "data"),
		Allowed: func(p string) bool {
			for _, x := range allowed {
				if filepath.Clean(x) == filepath.Clean(p) {
					return true
				}
			}
			return false
		},
		Now: func() time.Time { tick = tick.Add(time.Second); return tick },
	}
	return a, root
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestApplyAndRollback(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "CLAUDE.md")
	_ = os.WriteFile(target, []byte("old"), 0o640)
	a, _ := newApplier(t, target)
	h, _ := HashFile(target)
	m, err := a.Apply(Request{Path: target, Content: "new", ExpectedHash: h})
	if err != nil {
		t.Fatal(err)
	}
	if read(t, target) != "new" {
		t.Fatal("適用されていない")
	}
	if st, _ := os.Stat(target); st.Mode().Perm() != 0o640 {
		t.Fatalf("パーミッションが保持されていない: %v", st.Mode())
	}
	e, _ := a.Get(m.ID)
	if e.Before != "old" || e.After != "new" || e.OldHash != h || e.NewHash != HashBytes([]byte("new")) {
		t.Fatalf("%+v", e)
	}
	rb, err := a.Rollback(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, target) != "old" {
		t.Fatal("ロールバックされていない")
	}
	hist, _ := a.History()
	if len(hist) != 2 || hist[0].ID != rb.ID || hist[0].Op != "rollback" || hist[0].RollbackOf != m.ID {
		t.Fatalf("履歴: %+v", hist)
	}
	if got, _ := a.Get(rb.ID); got.Before != "new" {
		t.Fatal("ロールバック前の内容がバックアップされていない")
	}
	if _, err := a.Rollback(rb.ID); err == nil {
		t.Fatal("rollback の rollback は不可")
	}
}

func TestNewFileRollbackRemoves(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sub", "CLAUDE.local.md")
	a, _ := newApplier(t, target)
	m, err := a.Apply(Request{Path: target, Content: "x", ExpectedHash: AbsentHash})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Rollback(m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("削除されるはず")
	}
}

func TestRejects(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a.md")
	other := filepath.Join(root, "b.md")
	link := filepath.Join(root, "link.md")
	_ = os.WriteFile(target, []byte("old"), 0o644)
	_ = os.WriteFile(other, []byte("o"), 0o644)
	_ = os.Symlink(other, link)
	a, _ := newApplier(t, target, link)
	h, _ := HashFile(target)
	tests := []struct {
		name string
		req  Request
		want error
	}{
		{"許可外", Request{Path: other, Content: "x", ExpectedHash: "h"}, ErrNotAllowed},
		{"ハッシュ不一致", Request{Path: target, Content: "x", ExpectedHash: "deadbeef"}, ErrHashMismatch},
		{"シンボリックリンク", Request{Path: link, Content: "x", ExpectedHash: "h"}, ErrSymlink},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := a.Apply(tt.req); !errors.Is(err, tt.want) {
				t.Fatalf("got %v want %v", err, tt.want)
			}
		})
	}
	if read(t, target) != "old" || read(t, other) != "o" {
		t.Fatal("拒否されたのに変更されている")
	}
	if hist, _ := a.History(); len(hist) != 0 {
		t.Fatalf("拒否時にバックアップが残っている: %d", len(hist))
	}
	_ = h
}

func TestNilAllowedDeniesAll(t *testing.T) {
	a := &Applier{DataDir: t.TempDir()}
	if _, err := a.Apply(Request{Path: "/x"}); !errors.Is(err, ErrNotAllowed) {
		t.Fatal(err)
	}
}

func TestWriteFailureLeavesOriginalAndNoBackup(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root では権限エラーを再現できない")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "ro")
	_ = os.Mkdir(dir, 0o755)
	target := filepath.Join(dir, "a.md")
	_ = os.WriteFile(target, []byte("old"), 0o644)
	_ = os.Chmod(dir, 0o555)
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	a, _ := newApplier(t, target)
	h, _ := HashFile(target)
	if _, err := a.Apply(Request{Path: target, Content: "new", ExpectedHash: h}); err == nil {
		t.Fatal("失敗するはず")
	}
	if read(t, target) != "old" {
		t.Fatal("元ファイルが壊れた")
	}
	if hist, _ := a.History(); len(hist) != 0 {
		t.Fatal("失敗時のバックアップは消えるべき")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp が残っている: %v", entries)
	}
}

func TestGetValidation(t *testing.T) {
	a, _ := newApplier(t)
	for _, id := range []string{"", "../x", "a/b"} {
		if _, err := a.Get(id); err == nil {
			t.Errorf("%q はエラー", id)
		}
	}
	if h, err := a.History(); err != nil || h != nil {
		t.Fatalf("履歴なしは nil: %v %v", h, err)
	}
}

func TestSameTimestampNoCollision(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a.md")
	a, _ := newApplier(t, target)
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return fixed }
	prev := AbsentHash
	for _, c := range []string{"1", "2", "3"} {
		if _, err := a.Apply(Request{Path: target, Content: c, ExpectedHash: prev}); err != nil {
			t.Fatal(err)
		}
		prev, _ = HashFile(target)
	}
	if hist, _ := a.History(); len(hist) != 3 {
		t.Fatalf("履歴: %d", len(hist))
	}
}
