package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name, model string
		want        []string
	}{
		{"モデルなし", "", []string{"-p", "P", "--allowedTools", "Read,Write", "--output-format", "json"}},
		{"モデルあり", "m1", []string{"-p", "P", "--allowedTools", "Read,Write", "--output-format", "json", "--model", "m1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildArgs("P", tt.model); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveTimeout(t *testing.T) {
	for in, want := range map[time.Duration]time.Duration{0: 5 * time.Minute, -1: 5 * time.Minute, time.Second: time.Second} {
		if got := effectiveTimeout(in); got != want {
			t.Errorf("effectiveTimeout(%v)=%v want %v", in, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	boom := errors.New("boom")
	log := filepath.Join(t.TempDir(), "job.log")
	if err := os.WriteFile(log, []byte("tail-text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := classify(context.DeadlineExceeded, context.Canceled, boom, log); !errors.Is(err, ErrTimeout) {
		t.Errorf("timeout 優先: %v", err)
	}
	if err := classify(nil, context.Canceled, boom, log); !errors.Is(err, ErrCanceled) {
		t.Errorf("canceled: %v", err)
	}
	err := classify(nil, nil, boom, log)
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "tail-text") {
		t.Errorf("実行失敗: %v", err)
	}
	if err := classify(nil, nil, nil, log); err != nil {
		t.Errorf("成功: %v", err)
	}
}

func TestResolveClaudeInjectedHost(t *testing.T) {
	home := t.TempDir()
	none := func(string) []string { return nil }
	t.Run("何も見つからない", func(t *testing.T) {
		_, err := ResolveClaude(ResolveOptions{GOOS: "linux", Home: home, FixedPaths: none,
			Env: func(string) string { return "" }})
		if !errors.Is(err, ErrClaudeNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("シェル実行を注入", func(t *testing.T) {
		bin := filepath.Join(home, "sh-claude")
		if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
		var gotName string
		var gotArgs []string
		got, err := ResolveClaude(ResolveOptions{GOOS: "linux", Home: home, FixedPaths: none,
			Env: func(k string) string {
				if k == "SHELL" {
					return "/bin/fakesh"
				}
				return ""
			},
			Exec: func(name string, args ...string) ([]byte, error) {
				gotName, gotArgs = name, args
				return []byte("noise\n" + bin + "\n"), nil
			}})
		if err != nil || got != bin {
			t.Fatalf("got %q %v", got, err)
		}
		if gotName != "/bin/fakesh" || !reflect.DeepEqual(gotArgs, []string{"-lc", "command -v claude"}) {
			t.Fatalf("exec %q %v", gotName, gotArgs)
		}
	})
	t.Run("SHELL 未設定ならシェルを呼ばない", func(t *testing.T) {
		_, _ = ResolveClaude(ResolveOptions{GOOS: "linux", Home: home, FixedPaths: none,
			Env: func(string) string { return "" },
			Exec: func(string, ...string) ([]byte, error) {
				t.Fatal("呼ばれてはならない")
				return nil, nil
			}})
	})
	t.Run("固定パスの差し替え", func(t *testing.T) {
		bin := filepath.Join(home, "fixed-claude")
		if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
		got, err := ResolveClaude(ResolveOptions{GOOS: "linux", Home: home, ShellLookup: func() string { return "" },
			FixedPaths: func(string) []string { return []string{bin} }})
		if err != nil || got != bin {
			t.Fatalf("got %q %v", got, err)
		}
	})
	t.Run("windows の APPDATA と where を注入", func(t *testing.T) {
		app := t.TempDir()
		cmd := filepath.Join(app, "npm", "claude.cmd")
		if err := os.MkdirAll(filepath.Dir(cmd), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cmd, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := ResolveClaude(ResolveOptions{GOOS: "windows", Home: home,
			Env:  func(k string) string { return map[string]string{"APPDATA": app}[k] },
			Exec: func(string, ...string) ([]byte, error) { return nil, errors.New("no where") }})
		if err != nil || got != cmd {
			t.Fatalf("got %q %v", got, err)
		}
	})
}
