// Package masking は AI に渡す前のテキストから秘密情報を伏せる。
package masking

import (
	"fmt"
	"regexp"
)

type rule struct {
	kind string
	re   *regexp.Regexp
	repl string
}

// Masker はマスキング処理器。
type Masker struct{ rules []rule }

// Extra はユーザー追加パターン。
type Extra struct{ Name, Regex string }

func mask(kind string) string { return "[MASKED:" + kind + "]" }

func builtin() []rule {
	mk := func(kind, re, repl string) rule {
		return rule{kind: kind, re: regexp.MustCompile(re), repl: repl}
	}
	return []rule{
		mk("private-key", `(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?(-----END [A-Z ]*PRIVATE KEY-----|\z)`, mask("private-key")),
		mk("anthropic-key", `sk-ant-[A-Za-z0-9_\-]+`, mask("anthropic-key")),
		mk("aws-key", `AKIA[0-9A-Z]{16}`, mask("aws-key")),
		mk("github-token", `(?:ghp_|github_pat_)[A-Za-z0-9_]{6,}`, mask("github-token")),
		mk("slack-token", `xox[baprs]-[A-Za-z0-9\-]{6,}`, mask("slack-token")),
		mk("bearer", `(?i)\bBearer\s+[A-Za-z0-9._~+/=\-]{6,}`, mask("bearer")),
		mk("email", `[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`, mask("email")),
		// コマンドライン引数の `--token abc` / `--auth-token abc` / `--api-key=abc`（引用符内のエスケープも考慮。値が `-` で始まる次のフラグは対象外）
		mk("secret", `(?i)(^|\s)(--?(?:[a-z0-9]+[-_])*(?:password|passwd|secret|token|api[-_]?key|access[-_]?token)(?:=|\s+))(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s-]\S*)`, "${1}${2}"+mask("secret")),
		mk("secret", `(?i)((?:password|secret|token|api_key)\s*[:=]\s*)(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|\S+)`, "${1}"+mask("secret")),
	}
}

// New は組み込みパターン＋ユーザー追加パターンの Masker を作る。不正な正規表現はエラー。
func New(extra []Extra) (*Masker, error) {
	m := &Masker{rules: builtin()}
	for _, e := range extra {
		re, err := regexp.Compile(e.Regex)
		if err != nil {
			return nil, fmt.Errorf("マスキングパターン %q: %w", e.Name, err)
		}
		name := e.Name
		if name == "" {
			name = "custom"
		}
		m.rules = append(m.rules, rule{kind: name, re: re, repl: mask(name)})
	}
	return m, nil
}

// Mask は秘密情報を [MASKED:<種別>] に置換する。
func (m *Masker) Mask(s string) string {
	for _, r := range m.rules {
		s = r.re.ReplaceAllString(s, r.repl)
	}
	return s
}
