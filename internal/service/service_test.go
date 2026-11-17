package service

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"harness-app/internal/applier"
	"harness-app/internal/config"
	"harness-app/internal/testutil"
)

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

type fixture struct {
	svc       *Service
	claudeDir string
	ws        string // プロジェクトの cwd（テスト用の一時ディレクトリ）
	dataDir   string
}

func copyTree(t *testing.T, src, dst string, rewrite func(string) string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if rewrite != nil && strings.HasSuffix(p, ".jsonl") {
			b = []byte(rewrite(string(b)))
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// newFixture は testdata を TempDir に複製し、jsonl の cwd を TempDir 内に書き換える。
// 実ホームの .claude や /home/demo には一切触れない。
func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{claudeDir: filepath.Join(root, "claude"), ws: filepath.Join(root, "ws", "app"), dataDir: filepath.Join(root, "data")}
	copyTree(t, "../../testdata/workspace/app", f.ws, nil)
	copyTree(t, "../../testdata/claude", f.claudeDir, func(s string) string {
		return strings.ReplaceAll(s, "/home/demo/app", jsonEscape(f.ws)) // Windows のパス区切り（\）を JSON 用にエスケープ
	})
	fakeClaude := testutil.FakeClaude(t)
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	svc, err := New(Env{Home: root, ClaudeDir: f.claudeDir, DataDir: f.dataDir, Now: func() time.Time { return now },
		ShellLookup: func() string { return "" }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	c := svc.Settings()
	c.ClaudePath = fakeClaude
	if err := svc.SaveSettings(c); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reindex(false, nil); err != nil {
		t.Fatal(err)
	}
	f.svc = svc
	return f
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// T22: digest 生成 → 改善案 → 適用 → ロールバックまでを通す。
func TestE2EProposalApplyRollback(t *testing.T) {
	f := newFixture(t)
	target := filepath.Join(f.ws, "CLAUDE.md")
	original := read(t, target)

	ts, err := f.svc.ListTargets()
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, x := range ts {
		found = found || x.Path == target
	}
	if !found {
		t.Fatalf("プロジェクトの CLAUDE.md が発見されていない: %+v", ts)
	}

	p, err := f.svc.GenerateProposal(context.Background(), ProposalRequest{TargetPath: target, Project: "-home-demo-app", Days: 30, Instruction: "token=abc123 を含む指示"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Output, "フェイクが追加した行") || p.Rationale == "" {
		t.Fatalf("%+v", p)
	}
	if read(t, target) != original {
		t.Fatal("適用前にファイルが書き換わっている（AI 出力は自動保存してはいけない）")
	}
	in := read(t, filepath.Join(f.dataDir, "work", p.JobID, "input.json"))
	for _, leak := range []string{"sk-ant-api03", "dev@example.com", "abc123"} {
		if strings.Contains(in, leak) || strings.Contains(p.Digest, leak) {
			t.Errorf("AI への入力がマスクされていない: %s", leak)
		}
	}

	m, err := f.svc.ApplyProposal(p.TargetPath, p.Output, p.BaseHash)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, target) != p.Output {
		t.Fatal("適用されていない")
	}
	// 同じ BaseHash での再適用は拒否される（ファイルが変わっている）
	if _, err := f.svc.ApplyProposal(p.TargetPath, "x", p.BaseHash); !errors.Is(err, applier.ErrHashMismatch) {
		t.Fatalf("err=%v", err)
	}
	hist, _ := f.svc.ListBackups()
	if len(hist) != 1 || hist[0].ID != m.ID {
		t.Fatalf("%+v", hist)
	}
	if e, _ := f.svc.GetBackup(m.ID); e.Before != original || e.After != p.Output {
		t.Fatal("履歴の前後内容が不正")
	}
	if _, err := f.svc.Rollback(m.ID); err != nil {
		t.Fatal(err)
	}
	if read(t, target) != original {
		t.Fatal("ロールバックで元に戻っていない")
	}
	if hist, _ := f.svc.ListBackups(); len(hist) != 2 {
		t.Fatalf("ロールバックもバックアップされるはず: %d", len(hist))
	}
}

func TestWriteRestrictions(t *testing.T) {
	f := newFixture(t)
	outside := filepath.Join(t.TempDir(), "x.md")
	if _, err := f.svc.ReadTarget("/etc/passwd"); !errors.Is(err, applier.ErrNotAllowed) {
		t.Fatalf("err=%v", err)
	}
	if _, err := f.svc.ApplyProposal(outside, "x", applier.AbsentHash); !errors.Is(err, applier.ErrNotAllowed) {
		t.Fatalf("err=%v", err)
	}
	if _, err := f.svc.ApplyProposal(filepath.Join(f.claudeDir, "settings.json"), "{}", "h"); !errors.Is(err, applier.ErrNotAllowed) {
		t.Fatalf("settings.json は許可外: %v", err)
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("許可外に書き込まれた")
	}
}

func TestNewTargetFileCreatedAndRolledBack(t *testing.T) {
	f := newFixture(t)
	g := filepath.Join(f.claudeDir, "CLAUDE.md")
	_ = os.Remove(g)
	tc, err := f.svc.ReadTarget(g)
	if err != nil || tc.Target.Exists || tc.Hash != applier.AbsentHash {
		t.Fatalf("%+v %v", tc, err)
	}
	m, err := f.svc.ApplyProposal(g, "new", tc.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Rollback(m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(g); !os.IsNotExist(err) {
		t.Fatal("新規作成のロールバックで削除されるはず")
	}
}

func TestGenerateFailureModes(t *testing.T) {
	target := func(f *fixture) string { return filepath.Join(f.ws, "CLAUDE.md") }
	t.Run("claude 失敗", func(t *testing.T) {
		t.Setenv("FAKE_CLAUDE_MODE", "fail")
		f := newFixture(t)
		if _, err := f.svc.GenerateProposal(context.Background(), ProposalRequest{TargetPath: target(f), Days: 30}); err == nil {
			t.Fatal("エラーになるべき")
		}
	})
	t.Run("キャンセル", func(t *testing.T) {
		t.Setenv("FAKE_CLAUDE_MODE", "timeout")
		f := newFixture(t)
		go func() { time.Sleep(300 * time.Millisecond); f.svc.CancelGeneration() }()
		if _, err := f.svc.GenerateProposal(context.Background(), ProposalRequest{TargetPath: target(f), Days: 30}); err == nil {
			t.Fatal("キャンセルでエラーになるべき")
		}
	})
	t.Run("claude 未検出", func(t *testing.T) {
		f := newFixture(t)
		c := f.svc.Settings()
		c.ClaudePath = ""
		_ = f.svc.SaveSettings(c)
		st := f.svc.CheckClaude()
		if st.Found || st.Error == "" {
			// ホスト環境に claude がある場合は検出されうるので、Found=true なら Path を確認するのみ
			if st.Found && st.Path == "" {
				t.Fatal("Path が空")
			}
		}
	})
}

func TestSettingsValidation(t *testing.T) {
	f := newFixture(t)
	c := f.svc.Settings()
	c.MaskPatterns = []config.MaskPattern{{Name: "bad", Regex: "("}}
	if err := f.svc.SaveSettings(c); err == nil {
		t.Fatal("不正な正規表現は拒否")
	}
	c.MaskPatterns = []config.MaskPattern{{Name: "proj", Regex: `PRJ-\d+`}}
	c.TimeoutSeconds = 0
	if err := f.svc.SaveSettings(c); err != nil || f.svc.Settings().TimeoutSeconds != 300 {
		t.Fatalf("%v", err)
	}
	if st := f.svc.CheckClaude(); !st.Found {
		t.Fatalf("設定した fake claude が見つかるはず: %+v", st)
	}
}

func TestQueries(t *testing.T) {
	f := newFixture(t)
	o, err := f.svc.Overview(30)
	if err != nil {
		t.Fatal(err)
	}
	if o.Sessions != 2 || o.Corrections != 2 || o.ToolErrors != 1 || o.MemoryFiles != 2 || len(o.Daily) != 2 || len(o.Models) != 3 || len(o.Projects) != 1 {
		t.Fatalf("%+v", o)
	}
	if o7, _ := f.svc.Overview(1); o7.Sessions != 0 {
		t.Fatal("期間外")
	}
	ps, _ := f.svc.ListProjects()
	if len(ps) != 1 || ps[0].Sessions != 2 {
		t.Fatalf("%+v", ps)
	}
	ss, _ := f.svc.ListSessions("-home-demo-app", 30, 10)
	if len(ss) != 2 || ss[0].ID != "sess-0002" {
		t.Fatalf("新しい順: %+v", ss)
	}
	if none, _ := f.svc.ListSessions("zzz", 0, 10); len(none) != 0 {
		t.Fatal("プロジェクト絞り込み")
	}
	d, err := f.svc.GetSession("sess-0001")
	if err != nil || len(d.Messages) == 0 {
		t.Fatalf("%v %+v", err, d)
	}
	var corr, errs int
	for _, m := range d.Messages {
		if m.IsCorrection {
			corr++
		}
		if m.IsError {
			errs++
		}
	}
	if corr != 1 || errs != 1 {
		t.Fatalf("corr=%d errs=%d", corr, errs)
	}
	if _, err := f.svc.GetSession("nope"); err == nil {
		t.Fatal("存在しない ID")
	}
	if hits, _ := f.svc.Search("README"); len(hits) != 1 {
		t.Fatalf("hits=%d", len(hits))
	}
}

func TestReindexRebuild(t *testing.T) {
	f := newFixture(t)
	st, err := f.svc.Reindex(true, func(done, total int) {})
	if err != nil || st.Ingested != 2 {
		t.Fatalf("%+v %v", st, err)
	}
	if hits, _ := f.svc.Search("README"); len(hits) != 1 {
		t.Fatal("再構築後も FTS が使える")
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := New(Env{}); err == nil {
		t.Fatal("必須パラメータなしはエラー")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	testutil.Cleanup() // フェイク claude の一時ディレクトリを消す
	os.Exit(code)
}
