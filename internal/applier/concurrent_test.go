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
	final, err := HashFile(target)
	if err != nil {
		t.Fatal(err)
	}
	assertHistoryChain(t, a, m.OldHash, final) // 履歴には最初の Apply（v0→v1）も含まれる
}

// assertHistoryChain は、並行実行後の履歴が「start から final へ至る 1 本の直列な操作列」と整合することを検証する。
// 各履歴は manifest・バックアップ（original/applied）が欠けず、そのハッシュが OldHash/NewHash と一致する。
// 順序は ID（時刻）に頼らず、ハッシュ遷移のグラフで検査する: 各ハッシュで（出る辺 − 入る辺）が、
// start では +1、final では -1（start==final なら全て 0）、他は 0 であること。
// 排他が破れて操作が交錯すると、OldHash が直前の NewHash と連鎖せず、この釣り合いが崩れる。
func assertHistoryChain(t *testing.T, a *Applier, start, final string) {
	t.Helper()
	hist, err := a.History()
	if err != nil {
		t.Fatal(err)
	}
	dirs, err := os.ReadDir(a.backupsDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != len(dirs) {
		t.Fatalf("バックアップ dir %d 件に対し、読める履歴が %d 件（manifest の欠け）", len(dirs), len(hist))
	}
	balance := map[string]int{}
	for _, e := range hist {
		full, err := a.Get(e.ID)
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(a.backupsDir(), e.ID)
		if e.Existed {
			if got := HashBytes([]byte(full.Before)); got != e.OldHash {
				t.Errorf("%s: original のハッシュが OldHash と不一致", e.ID)
			}
		} else if _, err := os.Stat(filepath.Join(dir, "original")); err == nil || e.OldHash != AbsentHash {
			t.Errorf("%s: 存在しなかったのに original がある、または OldHash が absent でない", e.ID)
		}
		if e.NewHash != AbsentHash {
			if got := HashBytes([]byte(full.After)); got != e.NewHash {
				t.Errorf("%s: applied のハッシュが NewHash と不一致", e.ID)
			}
		}
		balance[e.OldHash]++
		balance[e.NewHash]--
	}
	if start != final {
		balance[start]--
		balance[final]++
	}
	for h, b := range balance {
		if b != 0 {
			t.Errorf("履歴の OldHash/NewHash が連鎖していない（ハッシュ %.8s の釣り合い %d）", h, b)
		}
	}
}
