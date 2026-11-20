// Package digest は期間・プロジェクトを指定して AI に渡す Markdown の digest を作る。
package digest

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"harness-app/internal/collector"
	"harness-app/internal/masking"
	"harness-app/internal/store"
)

// DefaultMaxBytes は digest サイズの既定上限。
const DefaultMaxBytes = 30000

// Options は digest 生成条件。
type Options struct {
	Project  string    // プロジェクトディレクトリ名。空なら全体
	Days     int       // 直近 N 日（0 以下なら 30）
	Now      time.Time // 現在時刻（必須。呼び出し側が注入する。ゼロはエラー）
	MaxBytes int       // 上限（0 なら DefaultMaxBytes）
	MaxItems int       // 各セクションの最大件数（0 なら 20）
}

func excerpt(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > n {
		return string([]rune(s)[:n]) + "…"
	}
	return s
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return excerpt(s, 160)
}

// Build は masking 済みの Markdown を返す。
func Build(st *store.Store, m *masking.Masker, opt Options) (string, error) {
	if m == nil {
		return "", fmt.Errorf("masker が必要です（masking を通さない digest は作れません）")
	}
	if opt.Days <= 0 {
		opt.Days = 30
	}
	if opt.Now.IsZero() {
		return "", fmt.Errorf("Options.Now（現在時刻）が必要です")
	}
	if opt.MaxBytes <= 0 {
		opt.MaxBytes = DefaultMaxBytes
	}
	if opt.MaxItems <= 0 {
		opt.MaxItems = 20
	}
	since := opt.Now.AddDate(0, 0, -opt.Days).UTC().Format(time.RFC3339)
	db := st.DB()

	var sessions, userMsgs, asstMsgs, toolErr, corr, inTok, outTok int
	if err := db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(user_msgs),0),COALESCE(SUM(assistant_msgs),0),COALESCE(SUM(tool_errors),0),
		COALESCE(SUM(corrections),0),COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0)
		FROM sessions WHERE started_at>=? AND (?='' OR project=?)`, since, opt.Project, opt.Project).
		Scan(&sessions, &userMsgs, &asstMsgs, &toolErr, &corr, &inTok, &outTok); err != nil {
		return "", fmt.Errorf("統計の集計: %w", err)
	}

	var b strings.Builder
	proj := opt.Project
	if proj == "" {
		proj = "全プロジェクト"
	}
	fmt.Fprintf(&b, "# セッション digest\n\n- 対象: %s\n- 期間: 直近 %d 日（%s 以降）\n\n", proj, opt.Days, since[:10])
	b.WriteString("## 統計\n\n| 項目 | 値 |\n|---|---|\n")
	fmt.Fprintf(&b, "| セッション数 | %d |\n| ユーザー発言 | %d |\n| アシスタント応答 | %d |\n| 訂正発言 | %d |\n| ツールエラー | %d |\n| 入力トークン | %d |\n| 出力トークン | %d |\n\n",
		sessions, userMsgs, asstMsgs, corr, toolErr, inTok, outTok)

	// 訂正の抜粋
	rows, err := db.Query(`SELECT m.ts, m.text FROM messages m JOIN sessions s ON s.id=m.session_id
		WHERE m.is_correction=1 AND s.started_at>=? AND (?='' OR s.project=?) ORDER BY m.ts DESC LIMIT ?`,
		since, opt.Project, opt.Project, opt.MaxItems)
	if err != nil {
		return "", fmt.Errorf("訂正の取得: %w", err)
	}
	b.WriteString("## 訂正発言の抜粋（新しい順）\n\n")
	n := 0
	for rows.Next() {
		var ts, text string
		if err := rows.Scan(&ts, &text); err != nil {
			rows.Close()
			return "", fmt.Errorf("訂正の読み取り: %w", err)
		}
		fmt.Fprintf(&b, "- [%s] %s\n", safeDate(ts), excerpt(text, 300))
		n++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", fmt.Errorf("訂正の反復: %w", err)
	}
	rows.Close()
	if n == 0 {
		b.WriteString("（なし）\n")
	}

	// エラーのまとめ
	rows, err = db.Query(`SELECT m.tool, m.text FROM messages m JOIN sessions s ON s.id=m.session_id
		WHERE m.is_error=1 AND s.started_at>=? AND (?='' OR s.project=?)`, since, opt.Project, opt.Project)
	if err != nil {
		return "", fmt.Errorf("エラーの取得: %w", err)
	}
	errCounts := map[string]int{}
	for rows.Next() {
		var tool, text string
		if err := rows.Scan(&tool, &text); err != nil {
			rows.Close()
			return "", fmt.Errorf("エラーの読み取り: %w", err)
		}
		if tool == "" {
			tool = "(不明)"
		}
		errCounts[tool+": "+firstLine(text)]++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", fmt.Errorf("エラーの反復: %w", err)
	}
	rows.Close()
	b.WriteString("\n## ツールエラーのまとめ（ツール: 先頭行 / 件数）\n\n")
	for _, c := range collector.TopCounts(errCounts, opt.MaxItems) {
		fmt.Fprintf(&b, "- %s ×%d\n", c.Name, c.Count)
	}
	if len(errCounts) == 0 {
		b.WriteString("（なし）\n")
	}

	// 頻出コマンド
	rows, err = db.Query(`SELECT m.text FROM messages m JOIN sessions s ON s.id=m.session_id
		WHERE m.kind='tool_use' AND m.tool='Bash' AND s.started_at>=? AND (?='' OR s.project=?)`, since, opt.Project, opt.Project)
	if err != nil {
		return "", fmt.Errorf("コマンドの取得: %w", err)
	}
	cmds := map[string]int{}
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			rows.Close()
			return "", fmt.Errorf("コマンドの読み取り: %w", err)
		}
		if t := collector.FirstToken(text); t != "" {
			cmds[t]++
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", fmt.Errorf("コマンドの反復: %w", err)
	}
	rows.Close()
	b.WriteString("\n## 頻出コマンド（Bash 先頭トークン）\n\n")
	for _, c := range collector.TopCounts(cmds, opt.MaxItems) {
		fmt.Fprintf(&b, "- `%s` ×%d\n", c.Name, c.Count)
	}
	if len(cmds) == 0 {
		b.WriteString("（なし）\n")
	}

	out := m.Mask(b.String())
	if len(out) > opt.MaxBytes {
		cut := opt.MaxBytes
		for cut > 0 && !utf8.RuneStart(out[cut]) {
			cut--
		}
		out = out[:cut] + "\n\n…（サイズ上限のため以降を省略）\n"
	}
	return out, nil
}

func safeDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}
