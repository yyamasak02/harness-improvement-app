package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrClaudeNotFound は claude バイナリが見つからないことを示す。
var ErrClaudeNotFound = errors.New("claude コマンドが見つかりません。設定画面でパスを指定してください")

// ResolveOptions は claude 解決の入力。全て注入可能。
type ResolveOptions struct {
	Configured  string        // ① 設定値
	ShellLookup func() string // ② `$SHELL -lc 'command -v claude'`（nil なら既定実装）
	Home        string        // ③ 候補パスの基準（~ 相当）
}

func isExec(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Mode()&0o111 != 0
}

// DefaultShellLookup はログインシェル経由で claude を探す（GUI 起動で PATH が最小限でも見つけるため）。
func DefaultShellLookup() string {
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

// ResolveClaude は ①設定値 ②ログインシェル ③既知の場所 の順で claude を探す。
func ResolveClaude(o ResolveOptions) (string, error) {
	if o.Configured != "" {
		if isExec(o.Configured) {
			return o.Configured, nil
		}
		return "", fmt.Errorf("設定された claude パス %q が実行できません: %w", o.Configured, ErrClaudeNotFound)
	}
	lookup := o.ShellLookup
	if lookup == nil {
		lookup = DefaultShellLookup
	}
	if p := lookup(); p != "" && isExec(p) {
		return p, nil
	}
	cands := []string{
		filepath.Join(o.Home, ".claude", "local", "claude"),
		filepath.Join(o.Home, ".local", "bin", "claude"),
		"/opt/homebrew/bin/claude",
		"/usr/local/bin/claude",
		filepath.Join(o.Home, ".npm-global", "bin", "claude"),
	}
	for _, c := range cands {
		if isExec(c) {
			return c, nil
		}
	}
	return "", ErrClaudeNotFound
}
