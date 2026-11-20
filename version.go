package main

import (
	_ "embed"
	"encoding/json"
)

// バージョンの出典は wails.json（ビルド時に埋め込む。通信はしない）。
//
//go:embed wails.json
var wailsJSON []byte

// versionFrom は wails.json の内容から info.productVersion を取り出す。読めなければ "unknown"。
func versionFrom(b []byte) string {
	var w struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(b, &w); err != nil || w.Info.ProductVersion == "" {
		return "unknown"
	}
	return w.Info.ProductVersion
}

// Version はこのアプリのバージョン（設定画面の「バージョン情報」に表示する）。
func (a *App) Version() string { return versionFrom(wailsJSON) }
