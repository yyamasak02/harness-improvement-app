package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// 形式バージョン導入前（user_version=0）の DB を、データを保ったまま現行に移行できる。
func TestOpenLegacyV0DB(t *testing.T) {
	sqlText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "upgrade", "v0", "index.sql"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "index.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(string(sqlText)); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()

	s, err := Open(path)
	if err != nil {
		t.Fatalf("旧形式を開けない: %v", err)
	}
	defer s.Close()
	hits, err := s.Search("upgradeprobe", 10)
	if err != nil || len(hits) != 1 {
		t.Fatalf("旧データが読めない: hits=%v err=%v", hits, err)
	}
	var ver int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&ver); err != nil || ver != SchemaVersion {
		t.Fatalf("バージョンが上がっていない: %d %v", ver, err)
	}
}

// 新しい形式の DB は古いアプリが開かず、中身も変えない。
func TestOpenNewerFormatRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(fmt.Sprintf(`PRAGMA user_version = %d; CREATE TABLE marker(x INT)`, SchemaVersion+1)); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()

	if _, err := Open(path); !errors.Is(err, ErrNewerFormat) {
		t.Fatalf("ErrNewerFormat を期待: %v", err)
	}
	raw, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var ver, n int
	if err := raw.QueryRow(`PRAGMA user_version`).Scan(&ver); err != nil || ver != SchemaVersion+1 {
		t.Fatalf("バージョンが変わった: %d %v", ver, err)
	}
	if err := raw.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='messages'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("新形式 DB が変更された: n=%d err=%v", n, err)
	}
}
