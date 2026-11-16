// Package osutil は OS ごとの差異を吸収する小さな関数を提供する。
package osutil

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenCommand は path を OS のファイラで開くコマンドと引数を返す。
func OpenCommand(goos, path string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{path}
	case "windows":
		return "explorer", []string{path}
	default:
		return "xdg-open", []string{path}
	}
}

// OpenPath は path を OS のファイラで開く（終了を待たない。explorer は成功時も終了コード 1 を返すため）。
func OpenPath(path string) error {
	name, args := OpenCommand(runtime.GOOS, path)
	if err := exec.Command(name, args...).Start(); err != nil {
		return fmt.Errorf("%s の起動: %w", name, err)
	}
	return nil
}
