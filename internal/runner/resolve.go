package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrClaudeNotFound は claude バイナリが見つからないことを示す。
var ErrClaudeNotFound = errors.New("claude コマンドが見つかりません。設定画面でパスを指定してください")

// ResolveOptions は claude 解決の入力。全て注入可能。
type ResolveOptions struct {
	Configured  string        // ① 設定値
	ShellLookup func() string // ② macOS/Linux: `$SHELL -lc 'command -v claude'` / Windows: `where claude`（nil なら既定実装）
	Home        string        // ③ 候補パスの基準（~ / %USERPROFILE% 相当）
	AppData     string        // Windows の %APPDATA%（npm グローバルの shim 置き場）。空なら環境変数を使う
	GOOS        string        // テスト用。空なら runtime.GOOS
}

func (o ResolveOptions) goos() string {
	if o.GOOS != "" {
		return o.GOOS
	}
	return runtime.GOOS
}

// hasExecExt は Windows で実行可能とみなす拡張子かを返す。
func hasExecExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".cmd", ".bat", ".com":
		return true
	}
	return false
}

// isExecutable はファイルが実行可能かを返す。Windows は拡張子で、それ以外は実行ビットで判定する。
func isExecutable(goos, path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	if goos == "windows" {
		return hasExecExt(path)
	}
	return st.Mode()&0o111 != 0
}

// DefaultShellLookup は GUI 起動で PATH が最小限でも claude を見つけるための既定探索。
func DefaultShellLookup() string { return shellLookup(runtime.GOOS) }

func shellLookup(goos string) string {
	if goos == "windows" {
		cmd := exec.Command("where", "claude")
		hideWindow(cmd)
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		return pickWindowsCandidate(string(out))
	}
	sh := os.Getenv("SHELL")
	if sh == "" {
		return ""
	}
	out, err := exec.Command(sh, "-lc", "command -v claude").Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// pickWindowsCandidate は `where claude` の出力から実行可能なものを選ぶ。
// .exe を優先し、次に .cmd/.bat/.com。npm が作る拡張子なしの shim（sh 用）は実行できないので除外する。
func pickWindowsCandidate(out string) string {
	var other string
	for _, l := range strings.Split(strings.ReplaceAll(out, "\r", ""), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || !hasExecExt(l) {
			continue
		}
		if strings.EqualFold(filepath.Ext(l), ".exe") {
			return l
		}
		if other == "" {
			other = l
		}
	}
	return other
}

// candidates は既知のインストール先（優先順）。
func candidates(goos, home, appData string) []string {
	if goos == "windows" {
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return []string{
			filepath.Join(home, ".claude", "local", "claude.exe"),
			filepath.Join(home, ".local", "bin", "claude.exe"),
			filepath.Join(appData, "npm", "claude.cmd"),
			filepath.Join(home, ".npm-global", "claude.cmd"),
		}
	}
	return []string{
		filepath.Join(home, ".claude", "local", "claude"),
		filepath.Join(home, ".local", "bin", "claude"),
		"/opt/homebrew/bin/claude",
		"/usr/local/bin/claude",
		filepath.Join(home, ".npm-global", "bin", "claude"),
	}
}

// ResolveClaude は ①設定値 ②ログインシェル（Windows は where）③既知の場所 の順で claude を探す。
func ResolveClaude(o ResolveOptions) (string, error) {
	goos := o.goos()
	if o.Configured != "" {
		if isExecutable(goos, o.Configured) {
			return o.Configured, nil
		}
		return "", fmt.Errorf("設定された claude パス %q が実行できません: %w", o.Configured, ErrClaudeNotFound)
	}
	lookup := o.ShellLookup
	if lookup == nil {
		lookup = func() string { return shellLookup(goos) }
	}
	if p := lookup(); p != "" && isExecutable(goos, p) {
		return p, nil
	}
	appData := o.AppData
	if appData == "" && goos == "windows" {
		appData = os.Getenv("APPDATA")
	}
	for _, c := range candidates(goos, o.Home, appData) {
		if isExecutable(goos, c) {
			return c, nil
		}
	}
	return "", ErrClaudeNotFound
}
