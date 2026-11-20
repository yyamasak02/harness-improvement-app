package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 形式バージョン導入前の config.json（formatVersion 無し）を読め、値が保たれ、保存で形式が付く。
func TestLoadLegacyV0Config(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "upgrade", "v0", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("旧形式を読めない: %v", err)
	}
	if c.FormatVersion != FormatVersion || c.Model != "demo-model" || c.TimeoutSeconds != 120 || !c.DailyReportEnabled ||
		c.DailyReportTime != "20:30" || len(c.MaskPatterns) != 1 || c.MaskPatterns[0].Name != "demo" {
		t.Fatalf("値が保たれていない: %+v", c)
	}
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	saved, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if !strings.Contains(string(saved), `"formatVersion": 1`) {
		t.Fatalf("保存に形式バージョンが無い: %s", saved)
	}
}

// 新しい形式の config.json は、読み込みを拒否し、上書きもしない。
func TestNewerFormatConfigNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	orig := `{"formatVersion": 999, "model": "future-demo", "futureField": true}`
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); !errors.Is(err, ErrNewerFormat) {
		t.Fatalf("ErrNewerFormat を期待: %v", err)
	}
	if err := Save(dir, Default()); !errors.Is(err, ErrNewerFormat) {
		t.Fatalf("Save は ErrNewerFormat で拒否するはず: %v", err)
	}
	if got, _ := os.ReadFile(p); string(got) != orig {
		t.Fatalf("ファイルが変更された: %s", got)
	}
}
