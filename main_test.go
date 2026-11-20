package main

import "testing"

// 最小サイズを下げる場合は frontend/e2e の検証サイズも見直すこと。
func TestMinWindowSize(t *testing.T) {
	if minWindowWidth < 640 || minWindowHeight < 480 {
		t.Fatalf("最小サイズが小さすぎる: %dx%d", minWindowWidth, minWindowHeight)
	}
}
