//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

// configureProc は子プロセスを独立したプロセスグループにし、キャンセル時にグループごと終了させる。
func configureProc(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}
