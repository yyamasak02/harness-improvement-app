package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"harness-app/internal/applier"
	"harness-app/internal/collector"
	"harness-app/internal/config"
	"harness-app/internal/extensions"
	"harness-app/internal/osutil"
	"harness-app/internal/service"
	"harness-app/internal/store"
	"harness-app/internal/targets"
)

// App は Wails バインディング層。ロジックは internal/service に置く。
type App struct {
	ctx     context.Context
	svc     *service.Service
	initErr error
}

// NewApp は App を作る。
func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	home, err := os.UserHomeDir()
	if err != nil {
		a.initErr = fmt.Errorf("ホームディレクトリの解決: %w", err)
		return
	}
	dataDir, err := config.DataDir("")
	if err != nil {
		a.initErr = err
		return
	}
	svc, err := service.New(service.Env{Home: home, ClaudeDir: filepath.Join(home, ".claude"), DataDir: dataDir})
	if err != nil {
		a.initErr = err
		return
	}
	a.svc = svc
	go a.indexAndWatch(ctx)
	go svc.StartReportScheduler(ctx, func(info service.ReportInfo, err error) {
		if err != nil {
			wruntime.EventsEmit(ctx, "report:error", err.Error())
			return
		}
		wruntime.EventsEmit(ctx, "report:done", info)
	})
}

func (a *App) shutdown(ctx context.Context) {
	if a.svc != nil {
		_ = a.svc.Close()
	}
}

// reindex は進捗を Wails イベントで通知しながら差分取り込みする。
func (a *App) reindex(rebuild bool) (collector.Stats, error) {
	wruntime.EventsEmit(a.ctx, "index:start")
	stats, err := a.svc.Reindex(rebuild, func(done, total int) {
		wruntime.EventsEmit(a.ctx, "index:progress", map[string]int{"done": done, "total": total})
	})
	if err != nil {
		wruntime.EventsEmit(a.ctx, "index:error", err.Error())
		return stats, err
	}
	wruntime.EventsEmit(a.ctx, "index:done", stats)
	return stats, nil
}

func (a *App) indexAndWatch(ctx context.Context) {
	a.reindexAndSuggest(ctx)
	_ = collector.Watch(ctx, a.svc.ClaudeDir(), 2*time.Second, func() { a.reindexAndSuggest(ctx) })
}

// reindexAndSuggest は取り込み後に、設定がオンなら改善案を自動生成する（提案まで。適用はしない）。
func (a *App) reindexAndSuggest(ctx context.Context) {
	if _, err := a.reindex(false); err != nil {
		return
	}
	go func() {
		ok, err := a.svc.RunAutoSuggest(ctx)
		switch {
		case err != nil:
			wruntime.EventsEmit(ctx, "autosuggest:error", err.Error())
		case ok:
			wruntime.EventsEmit(ctx, "autosuggest:done")
		}
	}()
}

// AutoSuggestStatus は改善案の自動生成の状態。
func (a *App) AutoSuggestStatus() (service.AutoSuggestStatus, error) {
	if err := a.ready(); err != nil {
		return service.AutoSuggestStatus{}, err
	}
	return a.svc.AutoSuggestStatus(), nil
}

// AutoSuggestion は自動生成された未確認の提案（無ければ null）。
func (a *App) AutoSuggestion() (*service.Proposal, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.AutoSuggestion()
}

// DismissAutoSuggestion は未確認の提案を破棄する。
func (a *App) DismissAutoSuggestion() error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.svc.DismissAutoSuggestion()
}

func (a *App) ready() error {
	if a.initErr != nil {
		return a.initErr
	}
	if a.svc == nil {
		return fmt.Errorf("アプリが初期化されていません")
	}
	return nil
}

// InitError は初期化エラーを返す（無ければ空文字）。
func (a *App) InitError() string {
	if a.initErr != nil {
		return a.initErr.Error()
	}
	return ""
}

func (a *App) Settings() (config.Config, error) {
	if err := a.ready(); err != nil {
		return config.Config{}, err
	}
	return a.svc.Settings(), nil
}

func (a *App) SaveSettings(c config.Config) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.svc.SaveSettings(c)
}

func (a *App) Reindex(rebuild bool) (collector.Stats, error) {
	if err := a.ready(); err != nil {
		return collector.Stats{}, err
	}
	return a.reindex(rebuild)
}

func (a *App) Overview(days int) (service.Overview, error) {
	if err := a.ready(); err != nil {
		return service.Overview{}, err
	}
	return a.svc.Overview(days)
}

func (a *App) ListProjects() ([]service.ProjectStat, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ListProjects()
}

func (a *App) ListSessions(project string, days, limit int) ([]store.Session, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ListSessions(project, days, limit)
}

func (a *App) GetSession(id string) (service.SessionDetail, error) {
	if err := a.ready(); err != nil {
		return service.SessionDetail{}, err
	}
	return a.svc.GetSession(id)
}

func (a *App) Search(q string) ([]store.SearchHit, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.Search(q)
}

func (a *App) ListTargets() ([]targets.Target, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ListTargets()
}

func (a *App) ListMemories() ([]service.MemoryItem, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ListMemories()
}

func (a *App) ListExtensions() (extensions.Report, error) {
	if err := a.ready(); err != nil {
		return extensions.Report{}, err
	}
	return a.svc.ListExtensions()
}

func (a *App) ReadTarget(path string) (service.TargetContent, error) {
	if err := a.ready(); err != nil {
		return service.TargetContent{}, err
	}
	return a.svc.ReadTarget(path)
}

func (a *App) BuildDigest(project string, days int) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	return a.svc.BuildDigest(project, days)
}

func (a *App) GenerateProposal(req service.ProposalRequest) (*service.Proposal, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.GenerateProposal(context.Background(), req)
}

func (a *App) CancelGeneration() {
	if a.svc != nil {
		a.svc.CancelGeneration()
	}
}

// ApplyProposal はユーザーが UI で「適用」を押したときだけ呼ばれる。
func (a *App) ApplyProposal(path, content, baseHash string) (applier.Manifest, error) {
	if err := a.ready(); err != nil {
		return applier.Manifest{}, err
	}
	return a.svc.ApplyProposal(path, content, baseHash)
}

func (a *App) GenerateSkill(req service.SkillRequest) (*service.SkillProposal, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.GenerateSkill(context.Background(), req)
}

// ApplySkill はユーザーが UI で「適用」を押したときだけ呼ばれる。
func (a *App) ApplySkill(name string, files []service.SkillApplyFile) ([]applier.Manifest, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ApplySkill(name, files)
}

// GenerateReport は日報を生成して <データディレクトリ>/reports/ に保存する（date が空なら今日）。
func (a *App) GenerateReport(date string) (service.ReportInfo, error) {
	if err := a.ready(); err != nil {
		return service.ReportInfo{}, err
	}
	return a.svc.GenerateReport(context.Background(), date)
}

func (a *App) ListReports() ([]service.ReportInfo, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ListReports()
}

func (a *App) ReadReport(date string) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	return a.svc.ReadReport(date)
}

// LaunchdPlist は launchd 用の plist を文字列で返す（生成のみ。登録・実行はしない）。
func (a *App) LaunchdPlist() (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	return a.svc.LaunchdPlist()
}

func (a *App) ListBackups() ([]applier.Entry, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.svc.ListBackups()
}

func (a *App) GetBackup(id string) (applier.Entry, error) {
	if err := a.ready(); err != nil {
		return applier.Entry{}, err
	}
	return a.svc.GetBackup(id)
}

func (a *App) Rollback(id string) (applier.Manifest, error) {
	if err := a.ready(); err != nil {
		return applier.Manifest{}, err
	}
	return a.svc.Rollback(id)
}

func (a *App) CheckClaude() (service.ClaudeStatus, error) {
	if err := a.ready(); err != nil {
		return service.ClaudeStatus{}, err
	}
	return a.svc.CheckClaude(), nil
}

// OpenDataDir はアプリデータディレクトリを OS のファイラで開く。
func (a *App) OpenDataDir() error {
	dir, err := config.DataDir("")
	if err != nil {
		return err
	}
	return osutil.OpenPath(dir)
}
