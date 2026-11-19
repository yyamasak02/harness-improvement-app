package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseClock(t *testing.T) {
	for _, c := range []struct {
		in   string
		h, m int
		ok   bool
	}{{"21:00", 21, 0, true}, {"00:05", 0, 5, true}, {"23:59", 23, 59, true}, {"24:00", 0, 0, false}, {"9:00", 0, 0, false},
		{"12:60", 0, 0, false}, {"abc", 0, 0, false}, {"", 0, 0, false}, {"12:00x", 0, 0, false}} {
		h, m, err := ParseClock(c.in)
		if (err == nil) != c.ok || (c.ok && (h != c.h || m != c.m)) {
			t.Errorf("%q: %d %d %v", c.in, h, m, err)
		}
	}
}

func TestDue(t *testing.T) {
	d := func(h, m int) time.Time { return time.Date(2026, 11, 10, h, m, 0, 0, time.UTC) }
	for _, c := range []struct {
		name  string
		now   time.Time
		clock string
		done  bool
		want  bool
	}{
		{"予定前", d(20, 59), "21:00", false, false},
		{"ちょうど", d(21, 0), "21:00", false, true},
		{"過ぎた（取り戻し）", d(23, 30), "21:00", false, true},
		{"実行済み", d(22, 0), "21:00", true, false},
		{"不正な時刻", d(22, 0), "xx", false, false},
	} {
		if got := Due(c.now, c.clock, c.done); got != c.want {
			t.Errorf("%s: %v", c.name, got)
		}
	}
}

func TestLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var n atomic.Int32
	done := make(chan struct{})
	go func() {
		Loop(ctx, 5*time.Millisecond, func(context.Context) {
			if n.Add(1) >= 3 {
				cancel()
			}
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Loop が終了しない")
	}
	if n.Load() < 3 {
		t.Fatal(n.Load())
	}
}
