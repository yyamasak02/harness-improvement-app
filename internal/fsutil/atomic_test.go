package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "a.txt")
	for _, content := range []string{"one", "two"} {
		if err := WriteFileAtomic(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(p)
		if string(got) != content {
			t.Fatalf("got %q want %q", got, content)
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Fatalf("temp ファイルが残っている: %v", entries)
	}
}
