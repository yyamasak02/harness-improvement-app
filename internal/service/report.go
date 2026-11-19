package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"harness-app/internal/fsutil"
	"harness-app/internal/launchd"
	"harness-app/internal/runner"
	"harness-app/internal/scheduler"
)

var reportDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// DefaultReportTime は日報を生成する既定の時刻。
const DefaultReportTime = "21:00"

// retryAfter は定期実行が失敗したあと、次に試すまでの間隔。
const retryAfter = 30 * time.Minute

// LaunchdLabel は生成する plist のラベル。
const LaunchdLabel = "app.harness-app.daily-report"

// ReportInfo は保存済みの日報 1 件。
type ReportInfo struct {
	Date string `json:"date"` // YYYY-MM-DD
	Path string `json:"path"`
}

func (s *Service) reportsDir() string { return filepath.Join(s.env.DataDir, "reports") }

func (s *Service) reportPath(date string) (string, error) {
	if !reportDateRe.MatchString(date) {
		return "", fmt.Errorf("日付は YYYY-MM-DD 形式で指定してください: %q", date)
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", fmt.Errorf("日付が不正です: %q", date)
	}
	return filepath.Join(s.reportsDir(), date+".md"), nil
}

// GenerateReport は当日分の digest（マスク済み）から日報を生成し、<データディレクトリ>/reports/<date>.md に保存する。
// 保存先はアプリのデータディレクトリのみ（~/.claude やプロジェクトには書かない）。date が空なら今日。
func (s *Service) GenerateReport(ctx context.Context, date string) (ReportInfo, error) {
	if date == "" {
		date = s.now().Format("2006-01-02")
	}
	path, err := s.reportPath(date)
	if err != nil {
		return ReportInfo{}, err
	}
	d, err := s.BuildDigest("", 1)
	if err != nil {
		return ReportInfo{}, err
	}
	r, rctx, done, err := s.startRunner(ctx)
	if err != nil {
		return ReportInfo{}, err
	}
	defer done()
	body, err := r.RunDailyReport(rctx, runner.DailyInput{Date: date, Digest: d})
	if err != nil {
		return ReportInfo{}, err
	}
	if err := fsutil.WriteFileAtomic(path, []byte(body), 0o600); err != nil {
		return ReportInfo{}, fmt.Errorf("日報の保存: %w", err)
	}
	return ReportInfo{Date: date, Path: path}, nil
}

// ListReports は保存済みの日報（新しい順）。
func (s *Service) ListReports() ([]ReportInfo, error) {
	ents, err := os.ReadDir(s.reportsDir())
	if errors.Is(err, os.ErrNotExist) {
		return []ReportInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("日報一覧: %w", err)
	}
	out := []ReportInfo{}
	for _, e := range ents {
		date := strings.TrimSuffix(e.Name(), ".md")
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".md") && reportDateRe.MatchString(date) {
			out = append(out, ReportInfo{Date: date, Path: filepath.Join(s.reportsDir(), e.Name())})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out, nil
}

// ReadReport は日報の本文を返す。
func (s *Service) ReadReport(date string) (string, error) {
	p, err := s.reportPath(date)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("日報の読み込み: %w", err)
	}
	return string(b), nil
}

func (s *Service) reportClock() string {
	if s.cfg.DailyReportTime == "" {
		return DefaultReportTime
	}
	return s.cfg.DailyReportTime
}

// RunScheduledReport は定期チェック 1 回分。有効で、今日の予定時刻を過ぎ、今日の日報が無ければ生成する（取り戻しを含む）。
// 生成したら true。失敗後は retryAfter 経つまで再試行しない。
func (s *Service) RunScheduledReport(ctx context.Context) (bool, error) {
	if !s.cfg.DailyReportEnabled {
		return false, nil
	}
	now := s.now()
	today := now.Format("2006-01-02")
	p, err := s.reportPath(today)
	if err != nil {
		return false, err
	}
	_, statErr := os.Stat(p)
	if !scheduler.Due(now, s.reportClock(), statErr == nil) {
		return false, nil
	}
	s.mu.Lock()
	if s.reportBusy || (!s.reportFailedAt.IsZero() && now.Sub(s.reportFailedAt) < retryAfter) {
		s.mu.Unlock()
		return false, nil
	}
	s.reportBusy = true
	s.mu.Unlock()
	_, err = s.GenerateReport(ctx, today)
	s.mu.Lock()
	s.reportBusy = false
	s.reportFailedAt = time.Time{}
	if err != nil {
		s.reportFailedAt = now
	}
	s.mu.Unlock()
	return err == nil, err
}

// StartReportScheduler は ctx が終わるまで 1 分ごとに RunScheduledReport を呼ぶ。onResult は生成成功・失敗の通知用（nil 可）。
func (s *Service) StartReportScheduler(ctx context.Context, onResult func(ReportInfo, error)) {
	scheduler.Loop(ctx, time.Minute, func(c context.Context) {
		ok, err := s.RunScheduledReport(c)
		if onResult == nil || (!ok && err == nil) {
			return
		}
		onResult(ReportInfo{Date: s.now().Format("2006-01-02")}, err)
	})
}

// LaunchdPlist は日報用の LaunchAgent の plist を文字列で返す（生成のみ。ファイルへの保存・登録・実行はしない）。
// 指定時刻にアプリを起動し、起動後の内部スケジューラが未生成なら日報を作る。
func (s *Service) LaunchdPlist() (string, error) {
	h, m, err := scheduler.ParseClock(s.reportClock())
	if err != nil {
		return "", err
	}
	return launchd.Plist(launchd.Options{Label: LaunchdLabel, Program: []string{"/usr/bin/open", "-a", "harness-app"}, Hour: h, Minute: m})
}
