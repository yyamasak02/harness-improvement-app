// Package runner は claude -p をヘッドレス実行し、ファイル経由で入出力を受け渡す。
package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"harness-app/internal/fsutil"
	"harness-app/internal/skills"
)

// AllowedTools は claude に許可するツール（固定。変更しない）。
const AllowedTools = "Read,Write"

var (
	// ErrTimeout は実行がタイムアウトしたことを示す。
	ErrTimeout = errors.New("claude の実行がタイムアウトしました")
	// ErrCanceled は実行がキャンセルされたことを示す。
	ErrCanceled = errors.New("claude の実行がキャンセルされました")
)

// Input は skill への入力（input.json）。
type Input struct {
	Digest          string `json:"digest"`
	CurrentClaudeMd string `json:"currentClaudeMd"`
	Instruction     string `json:"instruction"`
	TargetKind      string `json:"targetKind"` // global|project|local|rule
}

// Runner は claude 実行器。
type Runner struct {
	ClaudePath string
	WorkDir    string // ジョブディレクトリの親（アプリの作業ディレクトリ）
	Model      string
	Timeout    time.Duration // 0 なら 5 分
}

// Result は実行結果。
type Result struct {
	JobID     string `json:"jobId"`
	JobDir    string `json:"jobDir"`
	Output    string `json:"output"`
	Rationale string `json:"rationale"`
	Log       string `json:"log"`
}

func newJobID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(b[:])
}

// Run はジョブディレクトリを作り claude を実行する。cwd は常にジョブディレクトリ。
func (r *Runner) Run(ctx context.Context, in Input) (*Result, error) {
	if r.ClaudePath == "" {
		return nil, ErrClaudeNotFound
	}
	if r.WorkDir == "" {
		return nil, errors.New("WorkDir が未設定です")
	}
	id := newJobID()
	dir := filepath.Join(r.WorkDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("ジョブディレクトリ作成: %w", err)
	}
	ib, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("input.json 生成: %w", err)
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, "input.json"), ib, 0o600); err != nil {
		return nil, fmt.Errorf("input.json 書き込み: %w", err)
	}
	if err := skills.Extract(dir); err != nil {
		return nil, fmt.Errorf("skill 展開: %w", err)
	}
	res := &Result{JobID: id, JobDir: dir}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	prompt := "claudemd-improve skill に従い、カレントディレクトリの input.json を処理して output.md と rationale.md を書いてください。"
	args := []string{"-p", prompt, "--allowedTools", AllowedTools, "--output-format", "json"}
	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}
	cmd := exec.CommandContext(cctx, r.ClaudePath, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
	logf, err := os.Create(filepath.Join(dir, "job.log"))
	if err != nil {
		return nil, fmt.Errorf("ジョブログ作成: %w", err)
	}
	defer logf.Close()
	cmd.Stdout = logf
	cmd.Stderr = logf
	res.Log = filepath.Join(dir, "job.log")

	runErr := cmd.Run()
	switch {
	case errors.Is(cctx.Err(), context.DeadlineExceeded):
		return res, ErrTimeout
	case errors.Is(ctx.Err(), context.Canceled):
		return res, ErrCanceled
	case runErr != nil:
		return res, fmt.Errorf("claude の実行に失敗: %w（ログ: %s）", runErr, tail(res.Log))
	}
	out, err := os.ReadFile(filepath.Join(dir, "output.md"))
	if err != nil {
		return res, fmt.Errorf("output.md が生成されませんでした: %w", err)
	}
	res.Output = string(out)
	if rb, err := os.ReadFile(filepath.Join(dir, "rationale.md")); err == nil {
		res.Rationale = string(rb)
	}
	return res, nil
}

func tail(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		s = s[len(s)-300:]
	}
	return s
}
