package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setReport(t *testing.T, f *fixture, enabled bool, clock string) {
	t.Helper()
	c := f.svc.Settings()
	c.DailyReportEnabled, c.DailyReportTime = enabled, clock
	if err := f.svc.SaveSettings(c); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateReportSavesInDataDir(t *testing.T) {
	f := newFixture(t)
	info, err := f.svc.GenerateReport(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(f.dataDir, "reports", "2026-11-10.md")
	if info.Date != "2026-11-10" || info.Path != want {
		t.Fatalf("%+v", info)
	}
	if !strings.Contains(read(t, want), "フェイク") {
		t.Fatal("本文が保存されていない")
	}
	if _, err := os.Stat(filepath.Join(f.claudeDir, "reports")); err == nil {
		t.Fatal("フィクスチャのクロード設定側には書かない")
	}
	list, err := f.svc.ListReports()
	if err != nil || len(list) != 1 || list[0].Date != "2026-11-10" {
		t.Fatalf("%v %v", list, err)
	}
	if body, err := f.svc.ReadReport("2026-11-10"); err != nil || body == "" {
		t.Fatalf("%v", err)
	}
}

func TestReportInvalidDate(t *testing.T) {
	f := newFixture(t)
	for _, d := range []string{"../x", "2026-13-01", "20261110", "2026-11-10.md"} {
		if _, err := f.svc.GenerateReport(context.Background(), d); err == nil {
			t.Errorf("GenerateReport %q", d)
		}
		if _, err := f.svc.ReadReport(d); err == nil {
			t.Errorf("ReadReport %q", d)
		}
	}
	if l, err := f.svc.ListReports(); err != nil || len(l) != 0 {
		t.Fatalf("%v %v", l, err)
	}
}

func TestScheduledReport(t *testing.T) {
	f := newFixture(t)
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	f.svc.env.Now = func() time.Time { return now }
	ctx := context.Background()
	run := func() bool {
		ok, err := f.svc.RunScheduledReport(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}

	setReport(t, f, false, "09:00")
	now = now.Add(10 * time.Hour)
	if run() {
		t.Fatal("無効なら生成しない")
	}
	setReport(t, f, true, "21:00")
	if run() {
		t.Fatal("予定時刻前は生成しない")
	}
	now = time.Date(2026, 11, 10, 21, 0, 0, 0, time.UTC)
	if !run() {
		t.Fatal("予定時刻に生成する")
	}
	if run() {
		t.Fatal("同じ日に 2 回目は生成しない")
	}
	now = time.Date(2026, 11, 11, 23, 0, 0, 0, time.UTC)
	if !run() {
		t.Fatal("翌日の遅い起動でも取り戻して生成する")
	}
	if l, _ := f.svc.ListReports(); len(l) != 2 || l[0].Date != "2026-11-11" {
		t.Fatalf("%v", l)
	}
}

func TestScheduledReportFailureBacksOff(t *testing.T) {
	f := newFixture(t)
	now := time.Date(2026, 11, 10, 22, 0, 0, 0, time.UTC)
	f.svc.env.Now = func() time.Time { return now }
	t.Setenv("FAKE_CLAUDE_MODE", "fail")
	setReport(t, f, true, "21:00")
	if _, err := f.svc.RunScheduledReport(context.Background()); err == nil {
		t.Fatal("失敗が返る")
	}
	now = now.Add(time.Minute)
	if ok, err := f.svc.RunScheduledReport(context.Background()); ok || err != nil {
		t.Fatalf("間隔内は再試行しない: %v %v", ok, err)
	}
	t.Setenv("FAKE_CLAUDE_MODE", "ok")
	now = now.Add(retryAfter)
	if ok, err := f.svc.RunScheduledReport(context.Background()); !ok || err != nil {
		t.Fatalf("間隔後に成功: %v %v", ok, err)
	}
}

func TestSaveSettingsRejectsBadReportTime(t *testing.T) {
	f := newFixture(t)
	c := f.svc.Settings()
	c.DailyReportTime = "25:00"
	if err := f.svc.SaveSettings(c); err == nil {
		t.Fatal("不正な時刻は拒否")
	}
}

// 設定保存と定期チェックが並行しても data race にならない（go test -race で検出）。
func TestSaveSettingsAndScheduledReportRace(t *testing.T) {
	f := newFixture(t)
	f.svc.env.Now = func() time.Time { return time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC) } // 予定時刻前: 生成はしない
	setReport(t, f, true, "21:00")
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			c := f.svc.Settings()
			c.ExcludedProjects = []string{"x"}
			c.DailyReportTime = "21:00"
			if err := f.svc.SaveSettings(c); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 200; i++ {
		if _, err := f.svc.RunScheduledReport(context.Background()); err != nil {
			t.Error(err)
			break
		}
	}
	<-done
}

func TestGenerateReportRejectsOtherDates(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.GenerateReport(context.Background(), "2026-11-09"); err == nil {
		t.Fatal("今日以外の日付は拒否（digest は当日分のため）")
	}
}

func TestGenerateReportRejectsConcurrent(t *testing.T) {
	f := newFixture(t)
	f.svc.mu.Lock()
	f.svc.reportBusy = true
	f.svc.mu.Unlock()
	if _, err := f.svc.GenerateReport(context.Background(), ""); err == nil {
		t.Fatal("生成中は拒否")
	}
}

func TestLaunchdPlistFromSettings(t *testing.T) {
	f := newFixture(t)
	setReport(t, f, true, "07:30")
	s, err := f.svc.LaunchdPlist()
	if err != nil || !strings.Contains(s, "<integer>7</integer>") || !strings.Contains(s, "<integer>30</integer>") || !strings.Contains(s, LaunchdLabel) {
		t.Fatalf("%v\n%s", err, s)
	}
}
