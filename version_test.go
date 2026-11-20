package main

import (
	"regexp"
	"testing"
)

func TestVersionFrom(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"正常", `{"info":{"productVersion":"1.2.3-rc.1"}}`, "1.2.3-rc.1"},
		{"欠落", `{"info":{}}`, "unknown"},
		{"壊れた JSON", `{`, "unknown"},
	}
	for _, tt := range tests {
		if got := versionFrom([]byte(tt.in)); got != tt.want {
			t.Errorf("%s: %q != %q", tt.name, got, tt.want)
		}
	}
}

// 埋め込まれた wails.json から SemVer が読める（出典の wails.json と一致する）。
func TestEmbeddedVersion(t *testing.T) {
	v := NewApp().Version()
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`).MatchString(v) {
		t.Fatalf("バージョン形式が不正: %q", v)
	}
}
