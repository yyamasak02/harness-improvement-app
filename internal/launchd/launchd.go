// Package launchd は macOS の launchd 用 plist を文字列として生成する（生成のみ。登録・実行はしない）。
package launchd

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
)

// Options は plist の内容。
type Options struct {
	Label   string   // 例: app.harness-app.daily-report
	Program []string // 起動コマンド（先頭が実行ファイル）
	Hour    int
	Minute  int
}

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// Plist は毎日 Hour:Minute にコマンドを実行する LaunchAgent の plist を返す。
func Plist(o Options) (string, error) {
	if o.Label == "" || len(o.Program) == 0 {
		return "", errors.New("label と program は必須です")
	}
	if o.Hour < 0 || o.Hour > 23 || o.Minute < 0 || o.Minute > 59 {
		return "", fmt.Errorf("時刻が不正です: %d:%d", o.Hour, o.Minute)
	}
	var args bytes.Buffer
	for _, a := range o.Program {
		fmt.Fprintf(&args, "    <string>%s</string>\n", esc(a))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
%s  </array>
  <key>StartCalendarInterval</key>
  <dict>
    <key>Hour</key>
    <integer>%d</integer>
    <key>Minute</key>
    <integer>%d</integer>
  </dict>
</dict>
</plist>
`, esc(o.Label), args.String(), o.Hour, o.Minute), nil
}
