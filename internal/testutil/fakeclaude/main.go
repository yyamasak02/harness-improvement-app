// Command fakeclaude はテスト用のフェイク claude。カレントディレクトリ（ジョブディレクトリ）の input.json を読み、
// output.md / rationale.md を書く。Windows でも動くよう Go で実装している。
//
// FAKE_CLAUDE_MODE: ok（既定）| fail（終了コード 1）| timeout（長時間 sleep）| nooutput（出力ファイルを作らない）
// skill-generate（引数に "skill-generate" を含む）のときは output/SKILL.md と output/knowledge/notes.md を書く。
// FAKE_CLAUDE_MODE=badskill なら output/run.sh（不正な拡張子）も書く。
// FAKE_CLAUDE_ARGS_FILE: 指定すると cwd と引数を追記する（"cwd=..." / "arg=..." の行）
// FAKE_CLAUDE_SLEEP: timeout モードの秒数（既定 30）
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const output = "# 改善後の CLAUDE.md（フェイク）\n\n- 応答は日本語\n- テストは go test\n- フェイクが追加した行\n"
const rationale = "- フェイクによる固定の変更理由\n"

func main() { os.Exit(run()) }

func run() int {
	if f := os.Getenv("FAKE_CLAUDE_ARGS_FILE"); f != "" {
		fh, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			cwd, _ := os.Getwd()
			fmt.Fprintf(fh, "cwd=%s\n", cwd)
			for _, a := range os.Args[1:] {
				fmt.Fprintf(fh, "arg=%s\n", a)
			}
			_ = fh.Close()
		}
	}
	switch os.Getenv("FAKE_CLAUDE_MODE") {
	case "fail":
		fmt.Fprintln(os.Stderr, "fake-claude: forced failure")
		return 1
	case "timeout":
		sec := 30
		if v, err := strconv.Atoi(os.Getenv("FAKE_CLAUDE_SLEEP")); err == nil {
			sec = v
		}
		time.Sleep(time.Duration(sec) * time.Second)
		return 0
	case "nooutput":
		fmt.Println(`{"type":"result","result":"no output"}`)
		return 0
	}
	if _, err := os.Stat("input.json"); err != nil {
		fmt.Fprintln(os.Stderr, "fake-claude: input.json not found")
		return 2
	}
	for _, a := range os.Args[1:] {
		if strings.Contains(a, "skill-generate") {
			return writeSkill()
		}
	}
	if err := os.WriteFile("output.md", []byte(output), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	if err := os.WriteFile("rationale.md", []byte(rationale), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"done"}`)
	return 0
}

const skillMd = "---\nname: fake-skill\ndescription: フェイクが作った skill\n---\n\n# fake-skill\n\n詳細は knowledge/notes.md を参照。\n"

func writeSkill() int {
	files := map[string]string{
		"output/SKILL.md":           skillMd,
		"output/knowledge/notes.md": "# メモ\n\n- フェイクの知識\n",
		"rationale.md":              rationale,
	}
	if os.Getenv("FAKE_CLAUDE_MODE") == "badskill" {
		files["output/run.sh"] = "echo bad\n"
	}
	for p, c := range files {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
	}
	fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"done"}`)
	return 0
}
