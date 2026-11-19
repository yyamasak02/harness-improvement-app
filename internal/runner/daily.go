package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// dailyPrompt は daily-report 用（同じ理由で ASCII のみ）。
const dailyPrompt = "Follow the daily-report skill: process input.json in the current directory and write output.md."

// DailyInput は daily-report への入力（input.json。digest はマスク済み）。
type DailyInput struct {
	Date        string `json:"date"`
	Digest      string `json:"digest"`
	Instruction string `json:"instruction"`
}

// RunDailyReport は daily-report を実行し、output.md（日報本文）を返す。ファイルへは保存しない。
func (r *Runner) RunDailyReport(ctx context.Context, in DailyInput) (string, error) {
	res, err := r.execute(ctx, dailyPrompt, in)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(filepath.Join(res.JobDir, "output.md"))
	if err != nil {
		return "", errors.New("output.md が生成されませんでした")
	}
	if strings.TrimSpace(string(b)) == "" {
		return "", errors.New("日報が空でした")
	}
	return string(b), nil
}
