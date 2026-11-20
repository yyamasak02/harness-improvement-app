package digest

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
	const marker = "（サイズ上限のため以降を省略）"
	const max = 200
	out, err := Build(st, m, Options{Days: 30, Now: now, MaxBytes: max})
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, marker); n != 1 {
		t.Fatalf("省略マーカーは 1 回だけ: %d\n%s", n, out)
	}
	body := strings.SplitN(out, "\n\n…"+marker, 2)[0]
	if len(body) > max {
		t.Fatalf("本文が MaxBytes を超えた: %d", len(body))
	}
	if !utf8.ValidString(out) {
		t.Fatal("マルチバイト文字の途中で切れている")
	}
}

func TestRequiresMasker(t *testing.T) {
	st, _ := setup(t)
	if _, err := Build(st, nil, Options{}); err == nil {
		t.Fatal("masker なしはエラー")
	}
}

func TestExcerpt(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"空白の正規化", "a  b\nc", 10, "a b c"},
		{"マルチバイトの切り詰め", "あいうえお", 3, "あいう…"},
		{"n=0", "abc", 0, "…"},
		{"n=0 で空文字", "", 0, ""},
		{"ちょうど n 文字", "abcde", 5, "abcde"},
		{"n+1 文字", "abcdef", 5, "abcde…"},
		{"マルチバイトでちょうど n 文字", "あいう", 3, "あいう"},
		{"空白を詰めた後の長さで判定", "a   b", 3, "a b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := excerpt(tt.in, tt.n); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestSafeDate(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"空文字", "", ""},
		{"短い文字列", "2026-11", "2026-11"},
		{"9 文字", "2026-11-1", "2026-11-1"},
		{"ちょうど 10 文字", "2026-11-10", "2026-11-10"},
		{"RFC3339", "2026-11-10T12:34:56Z", "2026-11-10"},
		{"不正形式でも先頭 10 文字", "not-a-date-at-all", "not-a-date"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := safeDate(tt.in); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestBuildMasksAdditionalKinds(t *testing.T) {
	st, m := setup(t)
	_, err := st.DB().Exec(`INSERT INTO messages (session_id, ts, kind, text, is_correction) SELECT id, '2026-11-09T00:00:00Z', 'user', ?, 1 FROM sessions LIMIT 1`,
		"違う Bearer abcdef1234567890 と ghp_abcdef123456")
	if err != nil {
		t.Fatal(err)
	}
	out, err := Build(st, m, Options{Days: 30, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"abcdef1234567890", "ghp_abcdef123456"} {
		if strings.Contains(out, leak) {
			t.Errorf("マスクされていない: %s", leak)
		}
	}
	if !strings.Contains(out, "[MASKED:bearer]") {
		t.Errorf("bearer がマスクされるはず\n%s", out)
	}
}
