package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setAutoSuggest(t *testing.T, f *fixture, on bool) {
	t.Helper()
	c := f.svc.Settings()
	c.AutoSuggestEnabled = on
	if err := f.svc.SaveSettings(c); err != nil {
		t.Fatal(err)
	}
}

func TestAutoSuggestDefaultOffDoesNothing(t *testing.T) {
	f := newFixture(t)
	if f.svc.Settings().AutoSuggestEnabled {
		t.Fatal("既定はオフ")
	}
	ok, err := f.svc.RunAutoSuggest(context.Background())
	if ok || err != nil {
		t.Fatalf("%v %v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(f.dataDir, "work")); err == nil {
		t.Fatal("オフなら claude を実行しない（ジョブディレクトリができている）")
	}
	if st := f.svc.AutoSuggestStatus(); st.Enabled || st.HasProposal || st.LastAttemptAt != "" {
		t.Fatalf("%+v", st)
	}
}

// 提案までで、対象ファイルは書き換えない（適用は人の操作）。24 時間の間隔と新しいセッションの条件を守る。
func TestAutoSuggestGeneratesProposalOnlyWithLimits(t *testing.T) {
	f := newFixture(t)
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	f.svc.env.Now = func() time.Time { return now }
	setAutoSuggest(t, f, true)
	ctx := context.Background()
	global := filepath.Join(f.claudeDir, "CLAUDE.md")
	before, beforeErr := os.ReadFile(global)

	ok, err := f.svc.RunAutoSuggest(ctx)
	if !ok || err != nil {
		t.Fatalf("初回は生成: %v %v", ok, err)
	}
	p, err := f.svc.AutoSuggestion()
	if err != nil || p == nil || !strings.Contains(p.Output, "フェイクが追加した行") || p.TargetPath != global {
		t.Fatalf("%+v %v", p, err)
	}
	after, afterErr := os.ReadFile(global)
	if string(before) != string(after) || (beforeErr == nil) != (afterErr == nil) {
		t.Fatal("自動生成が対象ファイルを書き換えた（適用は自動化しない）")
	}
	if st := f.svc.AutoSuggestStatus(); !st.Enabled || !st.HasProposal || st.LastError != "" || st.LastAttemptAt == "" {
		t.Fatalf("%+v", st)
	}

	// 24 時間未満は実行しない。
	now = now.Add(23 * time.Hour)
	if ok, err := f.svc.RunAutoSuggest(ctx); ok || err != nil {
		t.Fatalf("間隔内: %v %v", ok, err)
	}
	// 24 時間経っても新しいセッションが無ければ実行しない。
	now = now.Add(2 * time.Hour)
	if ok, err := f.svc.RunAutoSuggest(ctx); ok || err != nil {
		t.Fatalf("新しいセッション無し: %v %v", ok, err)
	}
	// 破棄できる。
	if err := f.svc.DismissAutoSuggestion(); err != nil {
		t.Fatal(err)
	}
	if p, err := f.svc.AutoSuggestion(); p != nil || err != nil {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestAutoSuggestRunsAgainWithNewSession(t *testing.T) {
	f := newFixture(t)
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	f.svc.env.Now = func() time.Time { return now }
	setAutoSuggest(t, f, true)
	if ok, err := f.svc.RunAutoSuggest(context.Background()); !ok || err != nil {
		t.Fatalf("%v %v", ok, err)
	}
	now = now.Add(25 * time.Hour)
	if _, err := f.svc.st.DB().Exec(`UPDATE sessions SET ended_at='2026-11-10T12:00:00.000Z' WHERE id=(SELECT id FROM sessions LIMIT 1)`); err != nil {
		t.Fatal(err)
	}
	if ok, err := f.svc.RunAutoSuggest(context.Background()); !ok || err != nil {
		t.Fatalf("新しいセッションがあれば再実行: %v %v", ok, err)
	}
}

// claude が失敗したら諦めてエラーを残し、間隔内は再試行しない（無限リトライしない）。
func TestAutoSuggestFailureRecordedNoRetry(t *testing.T) {
	f := newFixture(t)
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	f.svc.env.Now = func() time.Time { return now }
	setAutoSuggest(t, f, true)
	t.Setenv("FAKE_CLAUDE_MODE", "fail")
	ok, err := f.svc.RunAutoSuggest(context.Background())
	if ok || err == nil {
		t.Fatalf("失敗が返る: %v %v", ok, err)
	}
	st := f.svc.AutoSuggestStatus()
	if st.LastError == "" || st.HasProposal || st.Running {
		t.Fatalf("%+v", st)
	}
	t.Setenv("FAKE_CLAUDE_MODE", "ok")
	now = now.Add(time.Hour)
	if ok, err := f.svc.RunAutoSuggest(context.Background()); ok || err != nil {
		t.Fatalf("失敗後も間隔内は再試行しない: %v %v", ok, err)
	}
}

// claude が見つからなくてもクラッシュせず、エラーを残す。
func TestAutoSuggestClaudeMissing(t *testing.T) {
	f := newFixture(t)
	setAutoSuggest(t, f, true)
	c := f.svc.Settings()
	c.ClaudePath = filepath.Join(f.dataDir, "no-such-claude")
	if err := f.svc.SaveSettings(c); err != nil {
		t.Fatal(err)
	}
	ok, err := f.svc.RunAutoSuggest(context.Background())
	if ok || err == nil {
		t.Fatalf("%v %v", ok, err)
	}
	if st := f.svc.AutoSuggestStatus(); st.LastError == "" || st.HasProposal {
		t.Fatalf("%+v", st)
	}
}
