package applier

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// 同じ ExpectedHash での同時 Apply は、ちょうど 1 件だけ成功し、残りは ErrHashMismatch になる（TOCTOU 防止）。
func TestConcurrentApplySameBaseOnlyOneWins(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "CLAUDE.md")
	if err := os.WriteFile(target, []byte("base"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := HashBytes([]byte("base"))
	const n = 16
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// service と同様に、呼び出しごとに別インスタンスを使う
			a := &Applier{DataDir: filepath.Join(root, "data"), Allowed: func(p string) bool { return p == target }}
			_, errs[i] = a.Apply(Request{Path: target, Content: fmt.Sprintf("v%d", i), ExpectedHash: base})
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, e := range errs {
		switch {
		case e == nil:
			ok++
		case !errors.Is(e, ErrHashMismatch):
			t.Errorf("想定外のエラー: %v", e)
		}
	}
	if ok != 1 {
		t.Fatalf("成功は 1 件のはず: %d", ok)
	}
}

// Apply と Rollback を並行に呼んでもデッドロック・競合しない。
func TestConcurrentApplyAndRollback(t *testing.T) {
	a, root := newApplier(t)
	target := filepath.Join(root, "CLAUDE.md")
	a.Allowed = func(p string) bool { return p == target }
	if err := os.WriteFile(target, []byte("v0"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := a.Apply(Request{Path: target, Content: "v1", ExpectedHash: HashBytes([]byte("v0"))})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = a.Rollback(m.ID) }()
		go func(i int) {
			defer wg.Done()
			cur, _ := HashFile(target)
			_, _ = a.Apply(Request{Path: target, Content: fmt.Sprintf("x%d", i), ExpectedHash: cur})
		}(i)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("デッドロックの疑い")
	}
	if _, err := os.ReadFile(target); err != nil {
		t.Fatal(err)
	}
}
