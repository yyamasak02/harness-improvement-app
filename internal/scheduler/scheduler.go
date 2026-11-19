// Package scheduler は「毎日 HH:MM 以降に 1 回」の判定と、定期チェックのループを提供する。時刻は注入できる。
package scheduler

import (
	"context"
	"fmt"
	"time"
)

// ParseClock は "HH:MM" を時・分に分解する。
func ParseClock(s string) (hour, min int, err error) {
	var extra string
	if n, _ := fmt.Sscanf(s, "%d:%d%s", &hour, &min, &extra); n != 2 || len(s) != 5 || s[2] != ':' ||
		hour < 0 || hour > 23 || min < 0 || min > 59 {
		return 0, 0, fmt.Errorf("時刻は HH:MM 形式で指定してください: %q", s)
	}
	return hour, min, nil
}

// Due は今日の予定時刻を過ぎていて、まだ今日の分を実行していなければ true（起動が遅れた場合の取り戻しを含む）。
func Due(now time.Time, clock string, doneToday bool) bool {
	h, m, err := ParseClock(clock)
	if err != nil || doneToday {
		return false
	}
	at := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
	return !now.Before(at)
}

// Loop は開始直後に 1 回、その後 interval ごとに fn を呼ぶ。ctx が終わると戻る。
func Loop(ctx context.Context, interval time.Duration, fn func(context.Context)) {
	fn(ctx)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn(ctx)
		}
	}
}
