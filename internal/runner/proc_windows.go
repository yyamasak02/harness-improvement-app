//go:build windows

package runner

import (
	"os/exec"
	"strconv"
	"syscall"
)

// configureProc は新しいプロセスグループで起動し、キャンセル時にプロセスツリーごと終了させる。
func configureProc(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow, HideWindow: true}
	cmd.Cancel = func() error {
		// taskkill /T は子孫プロセスも終了する。失敗したら本体だけでも終了させる。
		kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		hideWindow(kill)
		if err := kill.Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}

// createNoWindow は CREATE_NO_WINDOW。GUI アプリから起動するコンソールアプリの窓を出さない。
const createNoWindow = 0x08000000

// hideWindow は補助コマンド（where / taskkill）のコンソール窓を表示しない。
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow, HideWindow: true}
}
