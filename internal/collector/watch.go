package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch は <claudeDir>/projects 配下の jsonl 変更を監視し、debounce 後に onChange を呼ぶ。ctx 終了で停止する。
func Watch(ctx context.Context, claudeDir string, debounce time.Duration, onChange func()) error {
	root := filepath.Join(claudeDir, "projects")
	if _, err := os.Stat(root); err != nil {
		return fmt.Errorf("監視対象が無い: %w", err)
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watcher 作成: %w", err)
	}
	if err := w.Add(root); err != nil {
		_ = w.Close()
		return fmt.Errorf("watch 追加: %w", err)
	}
	subs, _ := filepath.Glob(filepath.Join(root, "*"))
	for _, s := range subs {
		if st, err := os.Stat(s); err == nil && st.IsDir() {
			_ = w.Add(s)
		}
	}
	go func() {
		defer w.Close()
		var mu sync.Mutex
		var timer *time.Timer
		trigger := func() {
			mu.Lock()
			defer mu.Unlock()
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, onChange)
		}
		for {
			select {
			case <-ctx.Done():
				mu.Lock()
				if timer != nil {
					timer.Stop()
				}
				mu.Unlock()
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Op&fsnotify.Create != 0 {
					if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
						_ = w.Add(ev.Name)
					}
				}
				if strings.HasSuffix(ev.Name, ".jsonl") {
					trigger()
				}
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return nil
}
