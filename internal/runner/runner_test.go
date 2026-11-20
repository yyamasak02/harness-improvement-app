package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode"

	"harness-app/internal/testutil"
)

func fake(t *testing.T) string { return testutil.FakeClaude(t) }

// samePathReal はシンボリックリンク（macOS の /var → /private/var など）を解決して比較する。
func samePathReal(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	return err1 == nil && err2 == nil && ra == rb
}

// newRunner は FakeClaude を使う Runner を返す。env は FAKE_CLAUDE_* を t.Setenv に頼らず渡す（t.Parallel 可）。
func newRunner(t *testing.T, env ...string) *Runner {
	t.Helper()
	return &Runner{ClaudePath: fake(t), WorkDir: t.TempDir(), Model: "m1", Timeout: 5 * time.Second, Env: env}
}

// waitForFile は path が非空で現れるまで短間隔でポーリングする（固定 sleep を避ける）。FakeClaude は起動直後に
// FAKE_CLAUDE_ARGS_FILE へ書くので、その出現を「起動した」マーカーとして使う。
// goroutine からも呼べるよう、t.Fatal ではなくエラーを返す。
func waitForFile(path string) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("FakeClaude の起動マーカー %s が現れない", path)
}

func TestRunOK(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args")
	r := newRunner(t, "FAKE_CLAUDE_ARGS_FILE="+argsFile)
	res, err := r.Run(context.Background(), Input{Digest: "d", CurrentClaudeMd: "c", TargetKind: "project"})
	if err != nil {
		t.Fatal(err)
	}
	argsBytes, _ := os.ReadFile(argsFile)
	args := strings.ReplaceAll(string(argsBytes), "\r\n", "\n")

	t.Run("出力と理由を読み取る", func(t *testing.T) {
		if !strings.Contains(res.Output, "フェイク") || !strings.Contains(res.Rationale, "固定") {
			t.Fatalf("%+v", res)
		}
	})
	t.Run("cwd はジョブディレクトリ", func(t *testing.T) {
		var gotCwd string
		for _, l := range strings.Split(args, "\n") {
			if strings.HasPrefix(l, "cwd=") {
				gotCwd = strings.TrimPrefix(l, "cwd=")
			}
		}
		if !samePathReal(gotCwd, res.JobDir) {
			t.Errorf("cwd がジョブディレクトリでない: got %q want %q", gotCwd, res.JobDir)
		}
	})
	t.Run("許可ツールは Read,Write のみ", func(t *testing.T) {
		for _, want := range []string{"arg=--allowedTools\narg=Read,Write", "arg=--output-format\narg=json", "arg=--model\narg=m1"} {
			if !strings.Contains(args, want) {
				t.Errorf("引数に %q が無い:\n%s", want, args)
			}
		}
		if strings.Contains(args, "Bash") {
			t.Error("Bash を許可してはいけない")
		}
	})
	t.Run("skill が展開される", func(t *testing.T) {
		if _, err := os.Stat(filepath.Join(res.JobDir, ".claude", "skills", "claudemd-improve", "SKILL.md")); err != nil {
			t.Error("skill が展開されていない")
		}
	})
	t.Run("input.json がある", func(t *testing.T) {
		if _, err := os.Stat(filepath.Join(res.JobDir, "input.json")); err != nil {
			t.Error("input.json が無い")
		}
	})
}

func TestRunErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		mode  string
		to    time.Duration
		check func(error) bool
	}{
		{"非ゼロ終了", "fail", 5 * time.Second, func(e error) bool { return e != nil && strings.Contains(e.Error(), "forced failure") }},
		// FakeClaude は 30 秒眠るので、タイムアウトを長めにしても所要時間は変わらない（負荷の高い CI でも起動を待てる）。
		{"タイムアウト", "timeout", time.Second, func(e error) bool { return errors.Is(e, ErrTimeout) }},
		{"出力なし", "nooutput", 5 * time.Second, func(e error) bool { return e != nil && strings.Contains(e.Error(), "output.md") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t, "FAKE_CLAUDE_MODE="+tt.mode)
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
	t.Parallel()
	started := filepath.Join(t.TempDir(), "started")
	r := newRunner(t, "FAKE_CLAUDE_MODE=timeout", "FAKE_CLAUDE_ARGS_FILE="+started)
	r.Timeout = time.Minute // タイムアウトではなくキャンセルで終わることを確かめる
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waitErr := make(chan error, 1)
	go func() {
		err := waitForFile(started)
		cancel() // 待機に失敗しても Run を止めてテストが固まらないようにする
		waitErr <- err
	}()
	_, err := r.Run(ctx, Input{})
	if werr := <-waitErr; werr != nil {
		t.Fatal(werr)
	}
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
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	mk := func(p string) string {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755)
		return p
	}
	none := func() string { return "" }
	cfg := mk(filepath.Join(t.TempDir(), "myclaude"+ext))
	shellBin := mk(filepath.Join(t.TempDir(), "shclaude"+ext))
	local := mk(filepath.Join(home, ".local", "bin", "claude"+ext))
	tests := []struct {
		name    string
		o       ResolveOptions
		want    string
		wantErr bool
	}{
		{"設定値が最優先", ResolveOptions{Configured: cfg, ShellLookup: func() string { return shellBin }, Home: home}, cfg, false},
		{"設定値が不正ならエラー", ResolveOptions{Configured: filepath.Join(home, "nope"+ext), ShellLookup: none, Home: home}, "", true},
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

// OS ごとの探索規則は GOOS を注入して全 OS 上で検証する。
func TestResolveClaudeByOS(t *testing.T) {
	t.Run("windows: 拡張子で実行可否を判定し .exe を既知の場所から見つける", func(t *testing.T) {
		home := t.TempDir()
		exe := filepath.Join(home, ".local", "bin", "claude.exe")
		_ = os.MkdirAll(filepath.Dir(exe), 0o755)
		_ = os.WriteFile(exe, []byte("x"), 0o644) // Windows では実行ビットを見ない
		got, err := ResolveClaude(ResolveOptions{GOOS: "windows", Home: home, ShellLookup: func() string { return "" }})
		if err != nil || got != exe {
			t.Fatalf("got %q err %v", got, err)
		}
	})
	t.Run("windows: 実行拡張子の無いファイルは不可", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "claude.txt")
		_ = os.WriteFile(p, []byte("x"), 0o755)
		if _, err := ResolveClaude(ResolveOptions{GOOS: "windows", Configured: p}); !errors.Is(err, ErrClaudeNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("windows: npm の shim（.cmd）は AppData から見つかる", func(t *testing.T) {
		home, appData := t.TempDir(), t.TempDir()
		shim := filepath.Join(appData, "npm", "claude.cmd")
		_ = os.MkdirAll(filepath.Dir(shim), 0o755)
		_ = os.WriteFile(shim, []byte("x"), 0o644)
		got, err := ResolveClaude(ResolveOptions{GOOS: "windows", Home: home, AppData: appData, ShellLookup: func() string { return "" }})
		if err != nil || got != shim {
			t.Fatalf("got %q err %v", got, err)
		}
	})
	t.Run("候補の順序は .exe が .cmd より先", func(t *testing.T) {
		c := candidates("windows", `C:\Users\u`, `C:\Users\u\AppData\Roaming`)
		var exeIdx, cmdIdx = -1, -1
		for i, p := range c {
			if strings.HasSuffix(p, ".exe") && exeIdx < 0 {
				exeIdx = i
			}
			if strings.HasSuffix(p, ".cmd") && cmdIdx < 0 {
				cmdIdx = i
			}
		}
		if exeIdx < 0 || cmdIdx < 0 || exeIdx > cmdIdx {
			t.Fatalf("%v", c)
		}
	})
	t.Run("linux: 候補に ~/.local/bin と /usr/bin を含み、実行ビットで判定する", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("実行ビットは Windows に無い")
		}
		home := t.TempDir()
		c := candidates("linux", home, "")
		want := map[string]bool{filepath.Join(home, ".local", "bin", "claude"): false, "/usr/bin/claude": false}
		for _, p := range c {
			if _, ok := want[p]; ok {
				want[p] = true
			}
		}
		for p, ok := range want {
			if !ok {
				t.Fatalf("候補に %s が無い: %v", p, c)
			}
		}
		bin := filepath.Join(home, ".local", "bin")
		if err := os.MkdirAll(bin, 0o755); err != nil {
			t.Fatal(err)
		}
		exe := filepath.Join(bin, "claude")
		if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		got, err := ResolveClaude(ResolveOptions{GOOS: "linux", Home: home, ShellLookup: func() string { return "" }})
		if err != nil || got != exe {
			t.Fatalf("got %q err %v", got, err)
		}
	})
	t.Run("unix: 実行ビットが無ければ不可", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "claude")
		_ = os.WriteFile(p, []byte("x"), 0o644)
		if runtime.GOOS == "windows" {
			t.Skip("Windows ではファイルの実行ビットを表現できない")
		}
		if _, err := ResolveClaude(ResolveOptions{GOOS: "linux", Configured: p}); !errors.Is(err, ErrClaudeNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestPromptIsSafeForCmdShim(t *testing.T) {
	if strings.ContainsAny(prompt, "%\"&|<>^\n\r") {
		t.Fatalf("プロンプトに .cmd shim で問題になる文字が含まれている: %q", prompt)
	}
	for _, r := range prompt {
		if r > unicode.MaxASCII {
			t.Fatalf("プロンプトに非 ASCII 文字 %q が含まれている（.cmd shim のコードページで文字化けしうる）: %q", r, prompt)
		}
	}
}

func TestPickWindowsCandidate(t *testing.T) {
	tests := []struct {
		name, out, want string
	}{
		{"exe を最優先", "C:\\npm\\claude\r\nC:\\npm\\claude.cmd\r\nC:\\bin\\claude.exe\r\n", "C:\\bin\\claude.exe"},
		{"拡張子なしの shim は除外し cmd を採用", "C:\\npm\\claude\r\nC:\\npm\\claude.cmd\r\n", "C:\\npm\\claude.cmd"},
		{"拡張子なしだけなら空", "C:\\npm\\claude\r\n", ""},
		{"空出力", "", ""},
		{"大文字の拡張子", "C:\\bin\\CLAUDE.EXE\r\n", "C:\\bin\\CLAUDE.EXE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pickWindowsCandidate(tt.out); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestRunSkillGenerate(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args")
	r := newRunner(t, "FAKE_CLAUDE_ARGS_FILE="+argsFile)
	res, err := r.RunSkillGenerate(context.Background(), SkillInput{SkillName: "s", Memories: []MemoryInput{{Label: "l", Content: "c"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 2 || res.Files[0].Path != "SKILL.md" || res.Files[1].Path != "knowledge/notes.md" || !strings.Contains(res.Rationale, "固定") {
		t.Fatalf("%+v", res)
	}
	args, _ := os.ReadFile(argsFile)
	if !strings.Contains(string(args), "Read,Write") || strings.Contains(string(args), "Bash") {
		t.Errorf("許可ツールは Read,Write のみ:\n%s", args)
	}
}

func TestRunSkillGenerateErrors(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"badskill", "fail", "nooutput"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			if _, err := newRunner(t, "FAKE_CLAUDE_MODE="+mode).RunSkillGenerate(context.Background(), SkillInput{SkillName: "s"}); err == nil {
				t.Fatal("エラーになるべき")
			}
		})
	}
}

func TestRunDailyReport(t *testing.T) {
	t.Parallel()
	argsFile := filepath.Join(t.TempDir(), "args")
	body, err := newRunner(t, "FAKE_CLAUDE_ARGS_FILE="+argsFile).RunDailyReport(context.Background(), DailyInput{Date: "2026-11-10", Digest: "d"})
	if err != nil || !strings.Contains(body, "フェイク") {
		t.Fatalf("%v %q", err, body)
	}
	args, _ := os.ReadFile(argsFile)
	if !strings.Contains(string(args), "daily-report") || !strings.Contains(string(args), "Read,Write") || strings.Contains(string(args), "Bash") {
		t.Errorf("%s", args)
	}
	for _, mode := range []string{"fail", "nooutput"} {
		if _, err := newRunner(t, "FAKE_CLAUDE_MODE="+mode).RunDailyReport(context.Background(), DailyInput{}); err == nil {
			t.Errorf("%s: エラーになるべき", mode)
		}
	}
}

func TestValidSkillRelPath(t *testing.T) {
	cases := map[string]bool{
		"SKILL.md": true, "knowledge/a.md": true, "a/b/c.md": true,
		"": false, "../a.md": false, "/a.md": false, "a/../b.md": false, "a//b.md": false, ".hidden.md": false,
		"x.sh": false, `a\b.md`: false, "C:/a.md": false, "a/.git/b.md": false,
	}
	for p, want := range cases {
		if got := ValidSkillRelPath(p); got != want {
			t.Errorf("%q: got %v want %v", p, got, want)
		}
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	testutil.Cleanup() // フェイク claude の一時ディレクトリを消す
	os.Exit(code)
}
