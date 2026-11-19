package service

import (
	"strings"
	"testing"
)

func TestListExtensionsMasked(t *testing.T) {
	svc, err := New(Env{
		Home:      "../../testdata/extensions/home",
		ClaudeDir: "../../testdata/extensions/claude",
		DataDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	r, err := svc.ListExtensions()
	if err != nil {
		t.Fatal(err)
	}
	if len(r.MCP) != 4 || len(r.Plugins) != 2 || len(r.Skills) != 2 || len(r.Hooks) != 3 {
		t.Fatalf("report=%+v", r)
	}
	for _, m := range r.MCP {
		if strings.Contains(strings.Join(m.Args, " "), "abcd1234efgh5678") {
			t.Errorf("マスクされていない: %+v", m)
		}
	}
}
