package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"harness-app/internal/config"
	"harness-app/internal/store"
)

const claudeDir = "../../testdata/claude"

func parse(t *testing.T, name string) *Parsed {
	t.Helper()
	p, err := ParseFile(filepath.Join(claudeDir, "projects", "-home-demo-app", name), "-home-demo-app",
		NewCorrectionDetector(config.DefaultCorrectionPatterns()))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseSession1(t *testing.T) {
	p := parse(t, "sess-0001.jsonl")
	s := p.Session
	if s == nil {
		t.Fatal("session nil")
	}
	tests := []struct {
		name      string
		got, want any
	}{
		{"id", s.ID, "sess-0001"},
		{"cwd", s.Cwd, "/home/demo/app"},
		{"broken", p.Broken, 1},
		{"user_msgs(tool_result/isMeta除外)", s.UserMsgs, 3},
		{"assistant_msgs", s.AssistantMsgs, 4},
		{"tool_errors", s.ToolErrors, 1},
		{"corrections", s.Corrections, 1},
		{"input_tokens", s.InputTokens, 10 + 10 + 5 + 10},
		{"cache_read", s.CacheRead, 400},
		{"started", s.StartedAt, "2026-11-01T01:00:00Z"},
		{"ended", s.EndedAt, "2026-11-01T01:03:00Z"},
		{"models", len(p.Usage), 2},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v want %v", tt.name, tt.got, tt.want)
		}
	}
	var errTool string
	for _, m := range p.Messages {
		if m.IsError {
			errTool = m.Tool
		}
	}
	if errTool != "Bash" {
		t.Errorf("エラーのツール名: %q", errTool)
	}
}

func TestParseSession2Unknown(t *testing.T) {
	p := parse(t, "sess-0002.jsonl")
	if p.Session.Corrections != 1 || p.Session.GitBranch != "feat/readme" || p.Broken != 0 {
		t.Fatalf("%+v broken=%d", p.Session, p.Broken)
	}
}

func TestParseEmptyAndMissing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "e.jsonl")
	_ = os.WriteFile(p, []byte("{\"type\":\"summary\"}\n\n"), 0o644)
	got, err := ParseFile(p, "x", nil)
	if err != nil || got.Session != nil {
		t.Fatalf("%v %+v", err, got)
	}
	if _, err := ParseFile(filepath.Join(dir, "none"), "x", nil); err == nil {
		t.Fatal("エラーになるべき")
	}
}

func TestStreamingDuplicateUsage(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "d.jsonl")
	l := `{"type":"assistant","sessionId":"d","message":{"id":"m1","model":"x","content":[],"usage":{"input_tokens":%d,"output_tokens":1}}}` + "\n"
	_ = os.WriteFile(p, []byte(sprintf(l, 1)+sprintf(l, 7)), 0o644)
	got, _ := ParseFile(p, "x", nil)
	if got.Session.AssistantMsgs != 1 || got.Session.InputTokens != 7 {
		t.Fatalf("%+v", got.Session)
	}
}

func TestIngestIncremental(t *testing.T) {
	// 実ファイルを触らないよう testdata を TempDir にコピーする
	dir := t.TempDir()
	proj := filepath.Join(dir, "projects", "-home-demo-app")
	_ = os.MkdirAll(proj, 0o755)
	for _, n := range []string{"sess-0001.jsonl", "sess-0002.jsonl"} {
		b, _ := os.ReadFile(filepath.Join(claudeDir, "projects", "-home-demo-app", n))
		_ = os.WriteFile(filepath.Join(proj, n), b, 0o644)
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	opt := Options{ClaudeDir: dir, CorrectionPatterns: config.DefaultCorrectionPatterns()}

	s1, err := Ingest(st, opt)
	if err != nil || s1.Ingested != 2 || s1.BrokenLines != 1 {
		t.Fatalf("%+v %v", s1, err)
	}
	s2, _ := Ingest(st, opt)
	if s2.Ingested != 0 || s2.Skipped != 2 {
		t.Fatalf("差分なしのはず: %+v", s2)
	}
	// 追記 → そのファイルだけ再取り込み。セッションが重複しない
	f := filepath.Join(proj, "sess-0002.jsonl")
	fh, _ := os.OpenFile(f, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = fh.WriteString(`{"type":"user","sessionId":"sess-0002","timestamp":"2026-11-02T10:00:00Z","message":{"role":"user","content":"追記"}}` + "\n")
	fh.Close()
	s3, _ := Ingest(st, opt)
	if s3.Ingested != 1 || s3.Skipped != 1 {
		t.Fatalf("%+v", s3)
	}
	var n, um int
	_ = st.DB().QueryRow(`SELECT COUNT(*), SUM(user_msgs) FROM sessions WHERE id='sess-0002'`).Scan(&n, &um)
	if n != 1 || um != 3 {
		t.Fatalf("sessions=%d user_msgs=%d", n, um)
	}
	// FTS に入っている
	if hits, _ := st.Search("追記", 5); len(hits) != 1 {
		t.Fatalf("hits=%d", len(hits))
	}
	// 削除追従
	_ = os.Remove(f)
	s4, _ := Ingest(st, opt)
	if s4.Removed != 1 {
		t.Fatalf("%+v", s4)
	}
	_ = st.DB().QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n != 1 {
		t.Fatalf("sessions=%d", n)
	}
	// 除外プロジェクト
	opt.ExcludedProjects = []string{"-home-demo-app"}
	opt.Force = true
	_, _ = Ingest(st, opt)
	_ = st.DB().QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n != 0 {
		t.Fatalf("除外されていない: %d", n)
	}
}

func TestSignals(t *testing.T) {
	d := NewCorrectionDetector([]string{"違う", " ", "No,"})
	if !d.Match("no, wrong") || !d.Match("それは違うよ") || d.Match("ok") {
		t.Fatal("Match")
	}
	var nilDet *CorrectionDetector
	if nilDet.Match("違う") {
		t.Fatal("nil は false")
	}
	for in, want := range map[string]string{"go test ./...": "go", "FOO=1 npm run x": "npm", "": "", "ls": "ls"} {
		if got := FirstToken(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
	top := TopCounts(map[string]int{"a": 1, "b": 3, "c": 3}, 2)
	if len(top) != 2 || top[0].Name != "b" || top[1].Name != "c" {
		t.Fatalf("%+v", top)
	}
}

// eventually は cond が真になるまで最大 d 待つ（固定 sleep に頼らない）。
func eventually(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

func TestWatchDebounce(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "projects", "p")
	_ = os.MkdirAll(proj, 0o755)
	const debounce = 300 * time.Millisecond // 遅い CI（-race・Windows）でも連続書き込みが窓に収まる余裕
	var n atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := Watch(ctx, dir, debounce, func() { n.Add(1) }); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(proj, "a.jsonl")
	for i := 0; i < 5; i++ { // 待たずに連続で書く
		_ = os.WriteFile(f, []byte{byte('a' + i)}, 0o644)
	}
	// 新規サブディレクトリも監視対象になる
	_ = os.MkdirAll(filepath.Join(dir, "projects", "q"), 0o755)
	if !eventually(5*time.Second, func() bool { return n.Load() >= 1 }) {
		t.Fatal("変更が通知されなかった")
	}
	time.Sleep(2 * debounce) // 余分な通知が来ないこと
	if n.Load() != 1 {
		t.Fatalf("連続した変更は 1 回にまとまるはず: %d", n.Load())
	}
	// 窓を過ぎた後の変更は別の通知になる
	_ = os.WriteFile(f, []byte("later"), 0o644)
	if !eventually(5*time.Second, func() bool { return n.Load() == 2 }) {
		t.Fatalf("2 回目の通知が来ない: %d", n.Load())
	}
	if err := Watch(ctx, filepath.Join(dir, "nope"), time.Second, func() {}); err == nil {
		t.Fatal("存在しないディレクトリはエラー")
	}
}

func sprintf(f string, a ...any) string { return fmt.Sprintf(f, a...) }
