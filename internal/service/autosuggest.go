package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"harness-app/internal/fsutil"
	"harness-app/internal/targets"
)

// autoSuggestInterval は自動生成の最小間隔（前回の試行から。成功・失敗を問わない）。
// 失敗後も同じ間隔まで再試行しない（無限リトライを避け、トークンを無駄にしない）。
const autoSuggestInterval = 24 * time.Hour

// autoSuggestDays は自動生成が対象にする直近の日数。
const autoSuggestDays = 7

// autoSuggestState は自動生成の状態（<データディレクトリ>/auto-suggest.json）。
// 保存先はアプリのデータ領域のみ。提案（Proposal）は保存するだけで、~/.claude には書かない。
type autoSuggestState struct {
	LastAttemptAt time.Time `json:"lastAttemptAt"`
	LastError     string    `json:"lastError"`
	Proposal      *Proposal `json:"proposal"`
}

// AutoSuggestStatus は設定画面に出す自動生成の状態。
type AutoSuggestStatus struct {
	Enabled       bool   `json:"enabled"`
	Running       bool   `json:"running"`
	LastAttemptAt string `json:"lastAttemptAt"` // RFC3339。未実行なら空
	LastError     string `json:"lastError"`
	HasProposal   bool   `json:"hasProposal"`
}

func (s *Service) autoSuggestPath() string { return filepath.Join(s.env.DataDir, "auto-suggest.json") }

func (s *Service) loadAutoSuggest() (autoSuggestState, error) {
	var st autoSuggestState
	b, err := os.ReadFile(s.autoSuggestPath())
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, fmt.Errorf("自動生成の状態の読み込み: %w", err)
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return autoSuggestState{}, fmt.Errorf("自動生成の状態の解析: %w", err)
	}
	return st, nil
}

func (s *Service) saveAutoSuggest(st autoSuggestState) error {
	b, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("自動生成の状態のシリアライズ: %w", err)
	}
	if err := fsutil.WriteFileAtomic(s.autoSuggestPath(), b, 0o600); err != nil {
		return fmt.Errorf("自動生成の状態の保存: %w", err)
	}
	return nil
}

// AutoSuggestStatus は自動生成の現在の状態。
func (s *Service) AutoSuggestStatus() AutoSuggestStatus {
	st, _ := s.loadAutoSuggest() // 壊れていれば未実行扱い
	s.mu.Lock()
	running := s.suggestBusy
	s.mu.Unlock()
	out := AutoSuggestStatus{Enabled: s.config().AutoSuggestEnabled, Running: running, LastError: st.LastError, HasProposal: st.Proposal != nil}
	if !st.LastAttemptAt.IsZero() {
		out.LastAttemptAt = st.LastAttemptAt.Format(time.RFC3339)
	}
	return out
}

// AutoSuggestion は自動生成された未確認の提案（無ければ nil）。適用はしない（人が差分を見て「適用」を押す）。
func (s *Service) AutoSuggestion() (*Proposal, error) {
	st, err := s.loadAutoSuggest()
	if err != nil {
		return nil, err
	}
	return st.Proposal, nil
}

// DismissAutoSuggestion は未確認の提案を破棄する（適用後・破棄時）。
func (s *Service) DismissAutoSuggestion() error {
	st, err := s.loadAutoSuggest()
	if err != nil {
		return err
	}
	if st.Proposal == nil {
		return nil
	}
	st.Proposal = nil
	return s.saveAutoSuggest(st)
}

// newSessionsSince は since より後のセッションの数。since がゼロなら全件。
func (s *Service) newSessionsSince(since time.Time) (int, error) {
	q, args := `SELECT COUNT(*) FROM sessions`, []any{}
	if !since.IsZero() {
		ts := since.UTC().Format("2006-01-02T15:04:05.000Z")
		q += ` WHERE ended_at > ? OR started_at > ?`
		args = append(args, ts, ts)
	}
	var n int
	if err := s.st.DB().QueryRow(q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("新しいセッションの確認: %w", err)
	}
	return n, nil
}

// RunAutoSuggest は取り込み後・起動時に呼ぶ 1 回分。設定がオンで、前回の試行から 24 時間以上経ち、
// 新しいセッションがあるときだけ、グローバル CLAUDE.md への改善案を生成して保存する（提案まで。適用は絶対にしない）。
// 生成したら true。失敗は状態に記録して返す（次の試行は 24 時間後）。
func (s *Service) RunAutoSuggest(ctx context.Context) (bool, error) {
	if !s.config().AutoSuggestEnabled {
		return false, nil
	}
	now := s.now()
	s.mu.Lock()
	if s.suggestBusy {
		s.mu.Unlock()
		return false, nil
	}
	s.suggestBusy = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.suggestBusy = false; s.mu.Unlock() }()

	st, err := s.loadAutoSuggest()
	if err != nil {
		return false, err
	}
	if !st.LastAttemptAt.IsZero() && now.Sub(st.LastAttemptAt) < autoSuggestInterval {
		return false, nil
	}
	n, err := s.newSessionsSince(st.LastAttemptAt)
	if err != nil || n == 0 {
		return false, err
	}

	// 試行時刻は結果にかかわらず記録する（失敗しても 24 時間は再試行しない）。
	st.LastAttemptAt = now
	// 手動生成のキャンセルと共有しないよう、キャンセル登録をしない Runner を使う。
	r, err := s.newRunner()
	var p *Proposal
	if err == nil {
		p, err = s.propose(ctx, r, ProposalRequest{TargetPath: targets.Global(s.env.ClaudeDir).Path, Days: autoSuggestDays})
	}
	if err != nil {
		st.LastError = err.Error()
		if serr := s.saveAutoSuggest(st); serr != nil {
			return false, errors.Join(err, serr)
		}
		return false, err
	}
	st.LastError, st.Proposal = "", p
	if err := s.saveAutoSuggest(st); err != nil {
		return false, err
	}
	return true, nil
}
