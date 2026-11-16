package digest

import (
	"strings"
	"testing"
	"time"

	"harness-app/internal/collector"
	"harness-app/internal/config"
	"harness-app/internal/masking"
	"harness-app/internal/store"
)

func setup(t *testing.T) (*store.Store, *masking.Masker) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := collector.Ingest(st, collector.Options{ClaudeDir: "../../testdata/claude",
		CorrectionPatterns: config.DefaultCorrectionPatterns()}); err != nil {
		t.Fatal(err)
	}
	m, _ := masking.New(nil)
	return st, m
}

var now = time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

func TestBuild(t *testing.T) {
	st, m := setup(t)
	out, err := Build(st, m, Options{Days: 30, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"セッション数 | 2", "訂正発言 | 2", "ツールエラー | 1",
		"違う、そうじゃなくて", "Bash: FAIL: TestFoo ×1", "`git` ×2", "`go` ×2"} {
		if !strings.Contains(out, want) {
			t.Errorf("含まれるはず: %q\n%s", want, out)
		}
	}
	for _, leak := range []string{"sk-ant-api03-DUMMY", "dev@example.com"} {
		if strings.Contains(out, leak) {
			t.Errorf("マスクされていない: %s", leak)
		}
	}
}

func TestBuildFilters(t *testing.T) {
	st, m := setup(t)
	tests := []struct {
		name string
		opt  Options
		want string
	}{
		{"期間外", Options{Days: 1, Now: now}, "セッション数 | 0"},
		{"存在しないプロジェクト", Options{Project: "zzz", Days: 30, Now: now}, "セッション数 | 0"},
		{"対象プロジェクト", Options{Project: "-home-demo-app", Days: 30, Now: now}, "セッション数 | 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := Build(st, m, tt.opt)
			if err != nil || !strings.Contains(out, tt.want) {
				t.Fatalf("%v\n%s", err, out)
			}
		})
	}
}

func TestSizeLimit(t *testing.T) {
	st, m := setup(t)
	out, _ := Build(st, m, Options{Days: 30, Now: now, MaxBytes: 200})
	if len(out) > 300 || !strings.Contains(out, "省略") {
		t.Fatalf("len=%d", len(out))
	}
}

func TestRequiresMasker(t *testing.T) {
	st, _ := setup(t)
	if _, err := Build(st, nil, Options{}); err == nil {
		t.Fatal("masker なしはエラー")
	}
}

func TestExcerpt(t *testing.T) {
	if got := excerpt("a  b\nc", 10); got != "a b c" {
		t.Fatal(got)
	}
	if got := excerpt("あいうえお", 3); got != "あいう…" {
		t.Fatal(got)
	}
}
