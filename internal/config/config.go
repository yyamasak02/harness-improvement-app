// Package config はアプリ設定の読み書きを行う。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"harness-app/internal/fsutil"
)

// MaskPattern はユーザー追加のマスキングパターン。
type MaskPattern struct {
	Name  string `json:"name"`
	Regex string `json:"regex"`
}

// Config はユーザー設定。
type Config struct {
	ClaudePath         string        `json:"claudePath"`
	Model              string        `json:"model"`
	ExcludedProjects   []string      `json:"excludedProjects"`
	CorrectionPatterns []string      `json:"correctionPatterns"`
	MaskPatterns       []MaskPattern `json:"maskPatterns"`
	TimeoutSeconds     int           `json:"timeoutSeconds"`
	// 日次振り返り（F6）。DailyReportTime は "HH:MM"（空なら 21:00）。
	DailyReportEnabled bool   `json:"dailyReportEnabled"`
	DailyReportTime    string `json:"dailyReportTime"`
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
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 300
	}
	return c, nil
}

// Save は設定を temp → rename で保存する。
func Save(dataDir string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("設定のシリアライズ: %w", err)
	}
	if err := fsutil.WriteFileAtomic(path(dataDir), b, 0o600); err != nil {
		return fmt.Errorf("設定保存: %w", err)
	}
	return nil
}
