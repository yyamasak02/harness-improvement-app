package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harness-app/internal/config"
)

// 性能計測用（scripts/perf-report.sh）。合成の jsonl を b.TempDir に作る（実ログは読まない）。
func BenchmarkParseFile(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 2000; i++ {
		ts := fmt.Sprintf("2026-11-01T%02d:%02d:00Z", (i/60)%24, i%60)
		fmt.Fprintf(&sb, `{"sessionId":"s","timestamp":"%s","cwd":"/home/demo/app","type":"user","message":{"role":"user","content":"テスト %d を実行して"}}`+"\n", ts, i)
		fmt.Fprintf(&sb, `{"sessionId":"s","timestamp":"%s","type":"assistant","message":{"id":"m%d","model":"claude-opus-4-1","role":"assistant","content":[{"type":"text","text":"実行します"},{"type":"tool_use","id":"t%d","name":"Bash","input":{"command":"go test ./..."}}],"usage":{"input_tokens":10,"output_tokens":20}}}`+"\n", ts, i, i)
	}
	path := filepath.Join(b.TempDir(), "bench.jsonl")
	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		b.Fatal(err)
	}
	det := NewCorrectionDetector(config.DefaultCorrectionPatterns())
	b.SetBytes(int64(sb.Len()))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseFile(path, "-home-demo-app", det); err != nil {
			b.Fatal(err)
		}
	}
}
