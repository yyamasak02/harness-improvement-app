package store

import (
	"path/filepath"
	"testing"
)

func seed(t *testing.T, s *Store) {
	t.Helper()
	for i, txt := range []string{"go test を実行して", "README を書く", "ab"} {
		if _, err := s.db.Exec(`INSERT INTO messages(session_id,project,ts,role,kind,text) VALUES('s','p',?, 'user','text',?)`,
			"2026-11-0"+string(rune('1'+i)), txt); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSearch(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "i.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seed(t, s)
	tests := []struct {
		name, q string
		want    int
	}{
		{"trigram", "実行して", 1},
		{"英数 trigram", "README", 1},
		{"2文字は LIKE", "ab", 1},
		{"1文字は LIKE", "を", 2},
		{"該当なし", "zzzzz", 0},
		{"空", "", 0},
		{"引用符を含む", `"x`, 0},
		{"LIKE ワイルドカードはリテラル", "%", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Search(tt.q, 10)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tt.want {
				t.Fatalf("got %d want %d", len(got), tt.want)
			}
		})
	}
}

func TestFTSDeleteTrigger(t *testing.T) {
	s, _ := Open(":memory:")
	defer s.Close()
	seed(t, s)
	if _, err := s.db.Exec(`DELETE FROM messages`); err != nil {
		t.Fatal(err)
	}
	got, err := s.Search("実行して", 10)
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "i.db")
	for i := 0; i < 2; i++ {
		s, err := Open(p)
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}
}
