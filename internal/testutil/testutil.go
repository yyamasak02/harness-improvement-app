// Package testutil はテスト用の共有ヘルパー。テストコードからのみ使う。
package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var (
	buildOnce sync.Once
	fakePath  string
	buildErr  error
)

// moduleRoot はこのファイルの位置からモジュールのルートを返す。
func moduleRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// FakeClaude はフェイク claude をビルドして実行ファイルのパスを返す（プロセス内で 1 回だけビルド）。
// 挙動は環境変数（t.Setenv）で切り替える。詳細は fakeclaude/main.go を参照。
func FakeClaude(t testing.TB) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "fakeclaude-")
		if err != nil {
			buildErr = fmt.Errorf("一時ディレクトリ作成: %w", err)
			return
		}
		name := "fake-claude"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		out := filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", out, "./internal/testutil/fakeclaude")
		cmd.Dir = moduleRoot()
		if b, err := cmd.CombinedOutput(); err != nil {
			buildErr = fmt.Errorf("fakeclaude のビルド: %w\n%s", err, b)
			return
		}
		fakePath = out
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	return fakePath
}
