package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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

func TestNewJobIDUsesInjectedNow(t *testing.T) {
	r := &Runner{Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }}
	if id := r.newJobID(); !strings.HasPrefix(id, "20260102T030405-") {
		t.Fatalf("id=%s", id)
	}
	if id := (&Runner{}).newJobID(); len(id) != len("20260102T030405-")+8 {
		t.Fatalf("既定の形式が違う: %s", id)
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

// skipNoExecBit は実行ビットで判定する非 Windows の解決を、Windows ホストでは検証できないためスキップする。
func skipNoExecBit(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Windows のファイルに実行ビットは無い")
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
		skipNoExecBit(t)
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
		_, err := ResolveClaude(ResolveOptions{GOOS: "linux", Home: home, FixedPaths: none,
			Env: func(string) string { return "" },
			Exec: func(string, ...string) ([]byte, error) {
				t.Fatal("呼ばれてはならない")
				return nil, nil
			}})
		if !errors.Is(err, ErrClaudeNotFound) { // 現行の挙動: シェル探索は空、固定パスも無いので見つからない
			t.Fatalf("got %v", err)
		}
	})
	t.Run("固定パスの差し替え", func(t *testing.T) {
		skipNoExecBit(t)
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

// shellLookup は GOOS ごとの探索コマンドと出力の解釈を、Env と Exec を注入して検証する（実シェルは起動しない）。
func TestShellLookup(t *testing.T) {
	envSH := func(k string) string {
		if k == "SHELL" {
			return "/bin/fakesh"
		}
		return ""
	}
	tests := []struct {
		name     string
		goos     string
		env      func(string) string
		out      string
		execErr  error
		want     string
		wantName string
		wantArgs []string
	}{
		{"windows: where の出力から exe を選ぶ", "windows", envSH, "C:\\npm\\claude\r\nC:\\bin\\claude.exe\r\n", nil, "C:\\bin\\claude.exe", "where", []string{"claude"}},
		{"windows: where が失敗なら空", "windows", envSH, "", errors.New("no where"), "", "where", []string{"claude"}},
		{"linux: 最終行を採用し空白を除く", "linux", envSH, "noise\n /usr/bin/claude \n", nil, "/usr/bin/claude", "/bin/fakesh", []string{"-lc", "command -v claude"}},
		{"darwin: 単一行", "darwin", envSH, "/opt/homebrew/bin/claude\n", nil, "/opt/homebrew/bin/claude", "/bin/fakesh", []string{"-lc", "command -v claude"}},
		{"linux: シェルが失敗なら空", "linux", envSH, "", errors.New("exit 1"), "", "/bin/fakesh", []string{"-lc", "command -v claude"}},
		{"linux: SHELL 未設定なら空で exec しない", "linux", func(string) string { return "" }, "", nil, "", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotName string
			var gotArgs []string
			o := ResolveOptions{Env: tt.env, Exec: func(name string, args ...string) ([]byte, error) {
				gotName, gotArgs = name, args
				return []byte(tt.out), tt.execErr
			}}
			if got := o.shellLookup(tt.goos); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
			if gotName != tt.wantName || !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Fatalf("exec %q %v want %q %v", gotName, gotArgs, tt.wantName, tt.wantArgs)
			}
		})
	}
}
