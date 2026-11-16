package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fake(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../scripts/fake-claude.sh")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func newRunner(t *testing.T) *Runner {
	return &Runner{ClaudePath: fake(t), WorkDir: t.TempDir(), Model: "m1", Timeout: 5 * time.Second}
}

func TestRunOK(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	t.Setenv("FAKE_CLAUDE_ARGS_FILE", argsFile)
	r := newRunner(t)
	res, err := r.Run(context.Background(), Input{Digest: "d", CurrentClaudeMd: "c", TargetKind: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Output, "フェイク") || !strings.Contains(res.Rationale, "固定") {
		t.Fatalf("%+v", res)
	}
	args, _ := os.ReadFile(argsFile)
	s := string(args)
	for _, want := range []string{"cwd=" + res.JobDir, "arg=--allowedTools\narg=Read,Write", "arg=--output-format\narg=json", "arg=--model\narg=m1"} {
		if !strings.Contains(s, want) {
			t.Errorf("引数に %q が無い:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Bash") {
		t.Error("Bash を許可してはいけない")
	}
	if _, err := os.Stat(filepath.Join(res.JobDir, ".claude", "skills", "claudemd-improve", "SKILL.md")); err != nil {
		t.Error("skill が展開されていない")
	}
	if _, err := os.Stat(filepath.Join(res.JobDir, "input.json")); err != nil {
		t.Error("input.json が無い")
	}
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name  string
		mode  string
		to    time.Duration
		check func(error) bool
	}{
		{"非ゼロ終了", "fail", 5 * time.Second, func(e error) bool { return e != nil && strings.Contains(e.Error(), "forced failure") }},
		{"タイムアウト", "timeout", 300 * time.Millisecond, func(e error) bool { return errors.Is(e, ErrTimeout) }},
		{"出力なし", "nooutput", 5 * time.Second, func(e error) bool { return e != nil && strings.Contains(e.Error(), "output.md") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("FAKE_CLAUDE_MODE", tt.mode)
			r := newRunner(t)
			r.Timeout = tt.to
			start := time.Now()
			_, err := r.Run(context.Background(), Input{})
			if !tt.check(err) {
				t.Fatalf("err=%v", err)
			}
			if time.Since(start) > 10*time.Second {
				t.Fatal("終了が遅すぎる（子プロセスが残っている）")
			}
		})
	}
}

func TestCancel(t *testing.T) {
	t.Setenv("FAKE_CLAUDE_MODE", "timeout")
	r := newRunner(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(200 * time.Millisecond); cancel() }()
	_, err := r.Run(ctx, Input{})
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunPreconditions(t *testing.T) {
	if _, err := (&Runner{WorkDir: t.TempDir()}).Run(context.Background(), Input{}); !errors.Is(err, ErrClaudeNotFound) {
		t.Fatal(err)
	}
	if _, err := (&Runner{ClaudePath: "x"}).Run(context.Background(), Input{}); err == nil {
		t.Fatal("WorkDir なしはエラー")
	}
}

func TestResolveClaude(t *testing.T) {
	home := t.TempDir()
	mk := func(p string) string {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755)
		return p
	}
	none := func() string { return "" }
	cfg := mk(filepath.Join(t.TempDir(), "myclaude"))
	shellBin := mk(filepath.Join(t.TempDir(), "shclaude"))
	local := mk(filepath.Join(home, ".local", "bin", "claude"))
	tests := []struct {
		name    string
		o       ResolveOptions
		want    string
		wantErr bool
	}{
		{"設定値が最優先", ResolveOptions{Configured: cfg, ShellLookup: func() string { return shellBin }, Home: home}, cfg, false},
		{"設定値が不正ならエラー", ResolveOptions{Configured: "/nope", ShellLookup: none, Home: home}, "", true},
		{"シェル経由", ResolveOptions{ShellLookup: func() string { return shellBin }, Home: home}, shellBin, false},
		{"既知の場所", ResolveOptions{ShellLookup: none, Home: home}, local, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveClaude(tt.o)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("got %q err %v", got, err)
			}
		})
	}
	// 何も無ければ ErrClaudeNotFound（候補にホスト固有のパスがあり得るため Home 外は検証しない）
	if _, err := ResolveClaude(ResolveOptions{ShellLookup: none, Home: t.TempDir()}); err != nil && !errors.Is(err, ErrClaudeNotFound) {
		t.Fatal(err)
	}
}
