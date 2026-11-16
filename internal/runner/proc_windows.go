//go:build windows

package runner

import (
	"os/exec"
	"strconv"
	"syscall"
)

// configureProc は新しいプロセスグループで起動し、キャンセル時にプロセスツリーごと終了させる。
func configureProc(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	cmd.Cancel = func() error {
		// taskkill /T は子孫プロセスも終了する。失敗したら本体だけでも終了させる。
		if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
