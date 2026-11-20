// Package service はアプリのユースケースを束ねる。app.go はこれを呼ぶだけの薄いバインディング。
package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"harness-app/internal/applier"
	"harness-app/internal/collector"
	"harness-app/internal/config"
	"harness-app/internal/digest"
	"harness-app/internal/masking"
	"harness-app/internal/osutil"
	"harness-app/internal/runner"
	"harness-app/internal/scheduler"
	"harness-app/internal/store"
	"harness-app/internal/targets"
)

// Env は外部環境（全て注入可能。テストでは TempDir / testdata を渡す）。
type Env struct {
	Home      string // ホーム（claude バイナリ探索用）
	ClaudeDir string // ~/.claude 相当（読み取り専用）
	DataDir   string // アプリデータディレクトリ
	// ShellLookup は claude 探索（nil なら既定）。
	ShellLookup func() string
	Now         func() time.Time
}

// Service はアプリ本体。
type Service struct {
	env Env
	st  *store.Store
	cfg config.Config
	// cfgNewer は config.json が新しい形式で読めなかった（既定値で動作中。Save は拒否される）。起動時に確定し不変。
	cfgNewer bool

	mu      sync.Mutex
	cancel  context.CancelFunc
	reindex sync.Mutex // Reindex の同時実行を直列化する

	reportBusy     bool      // 日報の定期生成中（mu で保護）
	reportFailedAt time.Time // 直近の定期生成の失敗時刻（mu で保護）
	suggestBusy    bool      // 改善案の自動生成中（mu で保護）
}

// New は設定を読み DB を開く。
func New(env Env) (*Service, error) {
	if env.ClaudeDir == "" || env.DataDir == "" {
		return nil, errors.New("ClaudeDir と DataDir は必須です")
	}
	cfg, err := config.Load(env.DataDir)
	newer := errors.Is(err, config.ErrNewerFormat)
	if err != nil && !newer {
		return nil, fmt.Errorf("設定の読み込み: %w", err)
	}
	if err := os.MkdirAll(env.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("データディレクトリ作成: %w", err)
	}
	st, err := store.Open(filepath.Join(env.DataDir, "index.db"))
	if err != nil {
		return nil, err
	}
	return &Service{env: env, st: st, cfg: cfg, cfgNewer: newer}, nil
}

// Close は DB を閉じる。
func (s *Service) Close() error { return s.st.Close() }

// ClaudeDir は監視用に公開する。
func (s *Service) ClaudeDir() string { return s.env.ClaudeDir }

func (s *Service) now() time.Time {
	if s.env.Now != nil {
		return s.env.Now()
	}
	return time.Now()
}

// Target は書き込み・閲覧対象（targets.Target の別名。エントリが targets を直接 import しないための公開）。
type Target = targets.Target

// ---- 設定 ----

// Config はアプリ設定（config.Config の別名。エントリが config を直接 import しないための公開）。
type Config = config.Config

// DefaultDataDir は OS 標準のアプリデータディレクトリを返す。
func DefaultDataDir() (string, error) { return config.DataDir("") }

// OpenPath は path を OS のファイラで開く（osutil.OpenPath への委譲。エントリが osutil を直接 import しないための公開）。
func OpenPath(path string) error { return osutil.OpenPath(path) }

// Settings は現在の設定。
func (s *Service) Settings() config.Config { return s.config() }

// SettingsStatus は設定ファイルの読み込み状態（画面の案内用）。
type SettingsStatus struct {
	NewerFormat bool   `json:"newerFormat"`
	Message     string `json:"message"`
}

// SettingsStatus は設定ファイルが新しい形式で読めなかったときに案内を返す。
func (s *Service) SettingsStatus() SettingsStatus {
	if !s.cfgNewer {
		return SettingsStatus{}
	}
	return SettingsStatus{NewerFormat: true, Message: "設定ファイルが新しい形式のため読み込めません（既定値で動作中で、設定の保存はできません）。新しいバージョンに更新するか、設定ファイル（config.json）を退避してください。"}
}

// config は設定のスナップショット（スケジューラなど別 goroutine からも安全に読める）。
func (s *Service) config() config.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Clone()
}

// SaveSettings は設定を検証して保存する（不正なマスキング正規表現は拒否）。
func (s *Service) SaveSettings(c config.Config) error {
	if _, err := s.masker(c); err != nil {
		return err
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 300
	}
	if c.DailyReportTime != "" {
		if _, _, err := scheduler.ParseClock(c.DailyReportTime); err != nil {
			return err
		}
	}
	if err := config.Save(s.env.DataDir, c); err != nil {
		return err
	}
	s.mu.Lock()
	s.cfg = c.Clone()
	s.mu.Unlock()
	return nil
}

func (s *Service) masker(c config.Config) (*masking.Masker, error) {
	var ex []masking.Extra
	for _, p := range c.MaskPatterns {
		ex = append(ex, masking.Extra{Name: p.Name, Regex: p.Regex})
	}
	return masking.New(ex)
}

// ---- インデックス ----

// Reindex は差分取り込みする。rebuild=true なら DB を空にして再構築する。
func (s *Service) Reindex(rebuild bool, progress func(done, total int)) (collector.Stats, error) {
	s.reindex.Lock()
	defer s.reindex.Unlock()
	if rebuild {
		if err := s.st.Reset(); err != nil {
			return collector.Stats{}, err
		}
	}
	cfg := s.config()
	return collector.Ingest(s.st, collector.Options{
		ClaudeDir:          s.env.ClaudeDir,
		CorrectionPatterns: cfg.CorrectionPatterns,
		ExcludedProjects:   cfg.ExcludedProjects,
		Progress:           progress,
		Force:              rebuild,
	})
}

// ---- claude ----

// ClaudeStatus は claude バイナリの検出結果。
type ClaudeStatus struct {
	Found bool   `json:"found"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

// CheckClaude は claude バイナリを解決する。
func (s *Service) CheckClaude() ClaudeStatus {
	cfg := s.config()
	p, err := runner.ResolveClaude(runner.ResolveOptions{Configured: cfg.ClaudePath, ShellLookup: s.env.ShellLookup, Home: s.env.Home})
	if err != nil {
		return ClaudeStatus{Error: err.Error()}
	}
	return ClaudeStatus{Found: true, Path: p}
}

// ---- ターゲットと書き込み許可 ----

func (s *Service) cwds() ([]string, error) {
	rows, err := s.st.DB().Query(`SELECT DISTINCT cwd FROM sessions WHERE cwd<>'' ORDER BY cwd`)
	if err != nil {
		return nil, fmt.Errorf("cwd の取得: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("cwd の読み取り: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListTargets は改善対象の一覧（グローバル＋各プロジェクト）。
func (s *Service) ListTargets() ([]targets.Target, error) {
	c, err := s.cwds()
	if err != nil {
		return nil, err
	}
	return targets.Discover(s.env.ClaudeDir, c), nil
}

func (s *Service) skillsDir() string { return filepath.Join(s.env.ClaudeDir, "skills") }

// allTargets は ListTargets に発見済みのプロジェクトメモリを加えた、読み書きの許可リスト。
func (s *Service) allTargets() ([]targets.Target, error) {
	ts, err := s.ListTargets()
	if err != nil {
		return nil, err
	}
	return append(ts, targets.Memories(s.env.ClaudeDir)...), nil
}

func (s *Service) allowed(path string) bool {
	ts, err := s.allTargets()
	if err != nil {
		return false
	}
	return targets.Allowed(ts, s.skillsDir(), path)
}

func (s *Service) applier() *applier.Applier {
	return &applier.Applier{DataDir: s.env.DataDir, Allowed: s.allowed, Now: s.env.Now}
}

func (s *Service) findTarget(path string) (targets.Target, error) {
	ts, err := s.allTargets()
	if err != nil {
		return targets.Target{}, err
	}
	for _, t := range ts {
		if targets.SamePath(t.Path, path) {
			return t, nil
		}
	}
	return targets.Target{}, fmt.Errorf("%s: %w", path, applier.ErrNotAllowed)
}

// TargetContent は対象ファイルの内容とハッシュ。
type TargetContent struct {
	Target  targets.Target `json:"target"`
	Content string         `json:"content"`
	Hash    string         `json:"hash"`
}

// ReadTarget は発見済み対象だけを読む。
func (s *Service) ReadTarget(path string) (TargetContent, error) {
	t, err := s.findTarget(path)
	if err != nil {
		return TargetContent{}, err
	}
	b, err := os.ReadFile(t.Path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return TargetContent{}, fmt.Errorf("対象の読み込み: %w", err)
	}
	h, err := applier.HashFile(t.Path)
	if err != nil {
		return TargetContent{}, err
	}
	return TargetContent{Target: t, Content: string(b), Hash: h}, nil
}

// ---- digest / 改善案 ----

// BuildDigest は masking 済みの digest を返す。
func (s *Service) BuildDigest(project string, days int) (string, error) {
	m, err := s.masker(s.config())
	if err != nil {
		return "", err
	}
	return digest.Build(s.st, m, digest.Options{Project: project, Days: days, Now: s.now()})
}

// ProposalRequest は改善案の生成要求。
type ProposalRequest struct {
	TargetPath  string `json:"targetPath"`
	Project     string `json:"project"`
	Days        int    `json:"days"`
	Instruction string `json:"instruction"`
}

// Proposal は生成された改善案（保存はしない。適用は ApplyProposal）。
type Proposal struct {
	TargetPath string `json:"targetPath"`
	BaseHash   string `json:"baseHash"`
	Original   string `json:"original"`
	Output     string `json:"output"`
	Rationale  string `json:"rationale"`
	Digest     string `json:"digest"`
	JobID      string `json:"jobId"`
}

// GenerateProposal は digest を作り claude -p で改善案を生成する。ファイルは書き換えない。
func (s *Service) GenerateProposal(ctx context.Context, req ProposalRequest) (*Proposal, error) {
	r, ctx, done, err := s.startRunner(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	return s.propose(ctx, r, req)
}

// propose は digest（masking 済み）と対象ファイルから claude -p で改善案を作る。手動・自動生成で共有する。
func (s *Service) propose(ctx context.Context, r *runner.Runner, req ProposalRequest) (*Proposal, error) {
	tc, err := s.ReadTarget(req.TargetPath)
	if err != nil {
		return nil, err
	}
	d, err := s.BuildDigest(req.Project, req.Days)
	if err != nil {
		return nil, err
	}
	m, err := s.masker(s.config())
	if err != nil {
		return nil, err
	}
	res, err := r.Run(ctx, runner.Input{
		Digest:          d,
		CurrentClaudeMd: m.Mask(tc.Content), // 対象ファイル本文も masking を通して渡す
		Instruction:     m.Mask(req.Instruction),
		TargetKind:      string(tc.Target.Kind),
	})
	if err != nil {
		return nil, err
	}
	return &Proposal{TargetPath: tc.Target.Path, BaseHash: tc.Hash, Original: tc.Content, Output: res.Output,
		Rationale: res.Rationale, Digest: d, JobID: res.JobID}, nil
}

// newRunner は設定のスナップショットから Runner を作る（キャンセル登録はしない）。
func (s *Service) newRunner() (*runner.Runner, error) {
	cfg := s.config()
	bin, err := runner.ResolveClaude(runner.ResolveOptions{Configured: cfg.ClaudePath, ShellLookup: s.env.ShellLookup, Home: s.env.Home})
	if err != nil {
		return nil, err
	}
	return &runner.Runner{ClaudePath: bin, WorkDir: filepath.Join(s.env.DataDir, "work"), Model: cfg.Model,
		Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second, Now: s.env.Now}, nil
}

// startRunner は claude の解決とキャンセル登録を行う。done は必ず呼ぶ。
func (s *Service) startRunner(ctx context.Context) (*runner.Runner, context.Context, func(), error) {
	r, err := s.newRunner()
	if err != nil {
		return nil, nil, nil, err
	}
	cctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	return r, cctx, func() { cancel(); s.mu.Lock(); s.cancel = nil; s.mu.Unlock() }, nil
}

// CancelGeneration は生成中のジョブをキャンセルする。
func (s *Service) CancelGeneration() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// ApplyProposal はユーザーが「適用」を押したときだけ呼ばれる。applier 経由で書き込む。
func (s *Service) ApplyProposal(path, content, baseHash string) (applier.Manifest, error) {
	return s.applier().Apply(applier.Request{Path: path, Content: content, ExpectedHash: baseHash})
}

// ---- 履歴 ----

// ListBackups は変更履歴（新しい順）。
func (s *Service) ListBackups() ([]applier.Entry, error) { return s.applier().History() }

// GetBackup は履歴 1 件（前後の本文付き）。
func (s *Service) GetBackup(id string) (applier.Entry, error) { return s.applier().Get(id) }

// Rollback は履歴を元に戻す。
func (s *Service) Rollback(id string) (applier.Manifest, error) { return s.applier().Rollback(id) }

// Session はセッション一覧の 1 件（store.Session の別名。エントリが store を直接 import しないための公開）。
type Session = store.Session

// SearchHit は検索結果の 1 件（store.SearchHit の別名。同上）。
type SearchHit = store.SearchHit
