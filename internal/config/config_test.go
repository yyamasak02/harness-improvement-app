package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaultWhenMissing(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c, Default()) {
		t.Fatalf("既定値でない: %+v", c)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Default()
	c.Model = "m"
	c.MaskPatterns = []MaskPattern{{Name: "x", Regex: "a+"}}
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil || !reflect.DeepEqual(got, c) {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestLoadBroken(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "config.json"), []byte("{"), 0o600)
	if _, err := Load(dir); err == nil {
		t.Fatal("エラーになるべき")
	}
}

func TestDataDir(t *testing.T) {
	d, err := DataDir("/x")
	if err != nil || d != "/x/harness-app" {
		t.Fatalf("%q %v", d, err)
	}
}
