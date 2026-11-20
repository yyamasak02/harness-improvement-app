// Package config はアプリ設定の読み書きを行う。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"harness-app/internal/fsutil"
)

// MaskPattern はユーザー追加のマスキングパターン。
type MaskPattern struct {
	Name  string `json:"name"`
	Regex string `json:"regex"`
}

// FormatVersion は config.json の形式バージョン。0（無し）は導入前の形式で、フィールドは現行と互換。
// 形式を変えるときは上げ、Load に移行を足す。
const FormatVersion = 1

// ErrNewerFormat は設定ファイルがこのアプリより新しい形式のときのエラー（古いアプリで上書きして壊さないため）。
var ErrNewerFormat = errors.New("設定ファイルがこのバージョンより新しい形式です。アプリを新しいバージョンに更新してください")

// Config はユーザー設定。
type Config struct {
	// FormatVersion は設定ファイルの形式バージョン（保存時にアプリが設定する。画面からは編集しない）。
	FormatVersion      int           `json:"formatVersion"`
	ClaudePath         string        `json:"claudePath"`
	Model              string        `json:"model"`
	ExcludedProjects   []string      `json:"excludedProjects"`
	CorrectionPatterns []string      `json:"correctionPatterns"`
	MaskPatterns       []MaskPattern `json:"maskPatterns"`
	TimeoutSeconds     int           `json:"timeoutSeconds"`
	// 日次振り返り（F6）。DailyReportTime は "HH:MM"（空なら 21:00）。
	DailyReportEnabled bool   `json:"dailyReportEnabled"`
	DailyReportTime    string `json:"dailyReportTime"`
	// 改善案の自動生成（既定オフ。claude -p を使うためトークンを消費する）。生成までで、適用は人の操作が必須。
	AutoSuggestEnabled bool `json:"autoSuggestEnabled"`
}

// Clone はスライスも含めて複製する（共有による data race を避けるため）。
func (c Config) Clone() Config {
	c.ExcludedProjects = cloneSlice(c.ExcludedProjects)
	c.CorrectionPatterns = cloneSlice(c.CorrectionPatterns)
	c.MaskPatterns = cloneSlice(c.MaskPatterns)
	return c
}

// cloneSlice は nil は nil、空は空のまま複製する（JSON で null と [] を区別するため）。
func cloneSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	return append(make([]T, 0, len(s)), s...)
}

// DefaultCorrectionPatterns は訂正発言の初期パターン。
func DefaultCorrectionPatterns() []string {
	return []string{"違う", "そうじゃな", "じゃなくて", "やめて", "間違", "no,", "don't", "instead", "actually"}
}

// Default は初期設定を返す。
func Default() Config {
	return Config{
		FormatVersion:      FormatVersion,
		CorrectionPatterns: DefaultCorrectionPatterns(),
		ExcludedProjects:   []string{},
		MaskPatterns:       []MaskPattern{},
		TimeoutSeconds:     300,
	}
}

// DataDir はアプリデータディレクトリを返す。base が空なら OS 標準の設定ディレクトリを使う。
func DataDir(base string) (string, error) {
	if base == "" {
		d, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("設定ディレクトリの解決: %w", err)
		}
		base = d
	}
	return filepath.Join(base, "harness-app"), nil
}

func path(dataDir string) string { return filepath.Join(dataDir, "config.json") }

// Load は設定を読む。無ければ既定値を返す。
func Load(dataDir string) (Config, error) {
	c := Default()
	b, err := os.ReadFile(path(dataDir))
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("設定読み込み: %w", err)
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return Default(), fmt.Errorf("設定の解析: %w", err)
	}
	if c.FormatVersion > FormatVersion {
		return Default(), fmt.Errorf("%w（設定 v%d、このアプリ v%d）", ErrNewerFormat, c.FormatVersion, FormatVersion)
	}
	c.FormatVersion = FormatVersion
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 300
	}
	return c, nil
}

// saveMu は Save の「形式バージョンの確認」から「書き込み」までを直列化する（確認後に別の Save が割り込まない）。
var saveMu sync.Mutex

// テスト用のフック（本番では nil）。確認直後・書き込み直後に呼ばれる。
var testHookAfterCheck, testHookAfterWrite func()

// Save は設定を temp → rename で保存する。
// 既存ファイルが新しい形式なら上書きしない。確認と書き込みは同じ排他の中で行う。
func Save(dataDir string, c Config) error {
	saveMu.Lock()
	defer saveMu.Unlock()
	if _, err := Load(dataDir); errors.Is(err, ErrNewerFormat) {
		return err
	}
	if testHookAfterCheck != nil {
		testHookAfterCheck()
	}
	c.FormatVersion = FormatVersion
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("設定のシリアライズ: %w", err)
	}
	if err := fsutil.WriteFileAtomic(path(dataDir), b, 0o600); err != nil {
		return fmt.Errorf("設定保存: %w", err)
	}
	if testHookAfterWrite != nil {
		testHookAfterWrite()
	}
	return nil
}
