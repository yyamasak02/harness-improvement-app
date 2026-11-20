package service

import (
	"path/filepath"

	"harness-app/internal/extensions"
	"harness-app/internal/masking"
)

// ExtensionsReport は拡張機能の一覧（extensions.Report の別名。エントリが extensions を直接 import しないための公開）。
type ExtensionsReport = extensions.Report

// ListExtensions は MCP / plugins / skills / hooks の状態（読み取りのみ）。
// コマンド・引数・URL はマスキングを通して返す。
func (s *Service) ListExtensions() (extensions.Report, error) {
	m, err := s.masker(s.config())
	if err != nil {
		// 追加パターンが不正でも一覧は出す（組み込みパターンだけで伏せる）
		if m, err = masking.New(nil); err != nil {
			return extensions.Report{}, err
		}
	}
	return extensions.Scan(s.env.ClaudeDir, filepath.Join(s.env.Home, ".claude.json"), m.Mask), nil
}
