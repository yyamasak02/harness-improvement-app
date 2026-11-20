package digest

import (
	"fmt"
	"strings"
	"testing"
)

// 反復中（rows.Next の途中）に DB がエラーを返しても、欠けた digest を成功扱いにしない。
// messages を、2 行目の text 評価で整数オーバーフローが起きるビューに差し替えて再現する。
func TestBuild_IterationError(t *testing.T) {
	cases := []struct {
		name    string
		isErr   int
		isCorr  int
		kind    string
		tool    string
		wantMsg string
	}{
		{"ツールエラー", 1, 0, "tool_result", "Bash", "エラーの反復"},
		{"頻出コマンド", 0, 0, "tool_use", "Bash", "コマンドの反復"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st, m := setup(t)
			var sid string
			if err := st.DB().QueryRow(`SELECT id FROM sessions LIMIT 1`).Scan(&sid); err != nil {
				t.Fatal(err)
			}
			view := fmt.Sprintf(`DROP TABLE messages;
CREATE VIEW messages AS
SELECT 1 AS id, '%[1]s' AS session_id, '' AS project, '2026-11-09T00:00:00Z' AS ts, 'user' AS role,
  '%[2]s' AS kind, '%[3]s' AS tool, 'ok' AS text, %[4]d AS is_error, %[5]d AS is_correction
UNION ALL
SELECT 2, '%[1]s', '', '2026-11-09T00:00:01Z', 'user', '%[2]s', '%[3]s',
  abs(-9223372036854775807 - 1), %[4]d, %[5]d`, sid, tc.kind, tc.tool, tc.isErr, tc.isCorr)
			if _, err := st.DB().Exec(view); err != nil {
				t.Fatal(err)
			}
			out, err := Build(st, m, Options{Days: 30, Now: now})
			if err == nil {
				t.Fatalf("反復中のエラーが握りつぶされた:\n%s", out)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("エラー文言に %q が必要: %v", tc.wantMsg, err)
			}
		})
	}
}

// 訂正の抜粋は ORDER BY のソートで全行の text が Query の時点で評価されるため、ビューの式では
// 反復中のエラーにならない。ts にインデックスを持つ実テーブルの上のビューにして、ソートを避け、
// 行ごとに遅延評価される text（古い行だけオーバーフロー）で rows.Err() の分岐を通す。
func TestBuild_IterationError_Correction(t *testing.T) {
	st, m := setup(t)
	var sid string
	if err := st.DB().QueryRow(`SELECT id FROM sessions LIMIT 1`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	setupSQL := fmt.Sprintf(`DROP TABLE messages;
CREATE TABLE raw_messages(id INTEGER PRIMARY KEY, session_id TEXT, project TEXT, ts TEXT, role TEXT,
  kind TEXT, tool TEXT, text TEXT, is_error INT, is_correction INT);
CREATE INDEX idx_raw_ts ON raw_messages(ts);
INSERT INTO raw_messages VALUES
  (1, '%[1]s', '', '2026-11-09T00:00:02Z', 'user', 'text', '', 'ok', 0, 1),
  (2, '%[1]s', '', '2026-11-09T00:00:01Z', 'user', 'text', '', 'ok', 0, 1);
CREATE VIEW messages AS
SELECT id, session_id, project, ts, role, kind, tool,
  CASE WHEN id = 2 THEN abs(-9223372036854775807 - 1) ELSE text END AS text, is_error, is_correction
FROM raw_messages`, sid)
	if _, err := st.DB().Exec(setupSQL); err != nil {
		t.Fatal(err)
	}
	out, err := Build(st, m, Options{Days: 30, Now: now})
	if err == nil {
		t.Fatalf("反復中のエラーが握りつぶされた:\n%s", out)
	}
	if !strings.Contains(err.Error(), "訂正の反復") {
		t.Errorf("rows.Err() の分岐（訂正の反復）のエラーになっていない（Query・Scan で失敗している可能性）: %v", err)
	}
}
