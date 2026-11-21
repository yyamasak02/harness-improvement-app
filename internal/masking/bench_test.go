package masking

import (
	"strings"
	"testing"
)

// 性能計測用（scripts/perf-report.sh）。入力は合成データのみ。
func BenchmarkMask(b *testing.B) {
	m, err := New(nil)
	if err != nil {
		b.Fatal(err)
	}
	clean := strings.Repeat("go test ./... を実行して結果を確認します。特に秘密は含まない通常の文章です。\n", 200)
	dirty := strings.Repeat("curl -H 'Authorization: Bearer abcdef123456' --token=xyz mail demo@example.com\n", 200)
	for name, in := range map[string]string{"clean": clean, "dirty": dirty} {
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(in)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = m.Mask(in)
			}
		})
	}
}
