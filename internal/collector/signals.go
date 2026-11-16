package collector

import (
	"sort"
	"strings"
)

// CorrectionDetector は訂正発言の検出器（部分一致・大文字小文字無視）。
type CorrectionDetector struct{ patterns []string }

// NewCorrectionDetector は空パターンを除いて検出器を作る。
func NewCorrectionDetector(patterns []string) *CorrectionDetector {
	d := &CorrectionDetector{}
	for _, p := range patterns {
		if p = strings.TrimSpace(p); p != "" {
			d.patterns = append(d.patterns, strings.ToLower(p))
		}
	}
	return d
}

// Match は s が訂正パターンのどれかを含むかを返す。
func (d *CorrectionDetector) Match(s string) bool {
	if d == nil {
		return false
	}
	l := strings.ToLower(s)
	for _, p := range d.patterns {
		if strings.Contains(l, p) {
			return true
		}
	}
	return false
}

// FirstToken は Bash コマンドの先頭トークン（環境変数代入は読み飛ばす）。
func FirstToken(cmd string) string {
	for _, f := range strings.Fields(cmd) {
		if strings.Contains(f, "=") && !strings.HasPrefix(f, "-") && !strings.Contains(f, "/") {
			continue
		}
		return f
	}
	return ""
}

// Count は名前と件数。
type Count struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// TopCounts は件数降順（同数は名前順）で上位 n 件を返す。
func TopCounts(m map[string]int, n int) []Count {
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}
