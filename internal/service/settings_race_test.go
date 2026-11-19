package service

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"harness-app/internal/config"
)

// 設定は呼び出し側と共有しない（保存時・取得時に複製）。決定的に検証する。
func TestSettingsSnapshotIsIsolated(t *testing.T) {
	f := newFixture(t)
	c := f.svc.Settings()
	c.ExcludedProjects = []string{"a"}
	c.MaskPatterns = []config.MaskPattern{{Name: "n", Regex: "zzz"}}
	if err := f.svc.SaveSettings(c); err != nil {
		t.Fatal(err)
	}
	c.ExcludedProjects[0] = "changed"
	c.MaskPatterns[0].Regex = "changed"
	got := f.svc.Settings()
	if got.ExcludedProjects[0] != "a" || got.MaskPatterns[0].Regex != "zzz" {
		t.Fatalf("保存後の呼び出し側の変更が波及: %+v", got)
	}
	got.ExcludedProjects[0] = "mut"
	if again := f.svc.Settings(); again.ExcludedProjects[0] != "a" {
		t.Fatalf("取得した値の変更が波及: %+v", again)
	}
}

// Settings の JSON で空スライスが null にならない（フロントの .join/.map が落ちる）。
func TestSettingsJSONHasNoNullSlices(t *testing.T) {
	f := newFixture(t)
	b, err := json.Marshal(f.svc.Settings())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "null") {
		t.Fatalf("JSON に null: %s", b)
	}
}

// SaveSettings と Settings を並行に呼んでも、呼び出し側のスライス書き換えが
// 内部状態に波及せず data race にならない（go test -race）。
func TestSettingsConcurrentAccessRace(t *testing.T) {
	f := newFixture(t)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		c := f.svc.Settings()
		c.ExcludedProjects = []string{"a", "b"}
		c.MaskPatterns = []config.MaskPattern{{Name: "n", Regex: "zzz"}}
		for i := 0; i < 100; i++ {
			// 保存後に呼び出し側が同じスライスを書き換えても、内部には影響しない
			if err := f.svc.SaveSettings(c); err != nil {
				t.Error(err)
				return
			}
			c.ExcludedProjects[0] = "c"
			c.MaskPatterns[0].Regex = "yyy"
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			c := f.svc.Settings()
			for range c.ExcludedProjects {
			}
			if len(c.MaskPatterns) > 0 {
				_ = c.MaskPatterns[0].Regex
			}
			// 返されたスナップショットを書き換えても内部には影響しない
			if len(c.ExcludedProjects) > 0 {
				c.ExcludedProjects[0] = "mut"
			}
		}
	}()
	wg.Wait()
}
