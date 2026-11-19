package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtract(t *testing.T) {
	dir := t.TempDir()
	if err := Extract(dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, ".claude", "skills", "claudemd-improve", "SKILL.md"))
	if err != nil || !strings.Contains(string(b), "name: claudemd-improve") {
		t.Fatalf("%v %s", err, b)
	}
	names, _ := Names()
	if len(names) != 2 || names[0] != "claudemd-improve" || names[1] != "skill-generate" {
		t.Fatalf("%v", names)
	}
	b, err = os.ReadFile(filepath.Join(dir, ".claude", "skills", "skill-generate", "SKILL.md"))
	if err != nil || !strings.Contains(string(b), "name: skill-generate") {
		t.Fatalf("%v %s", err, b)
	}
}
