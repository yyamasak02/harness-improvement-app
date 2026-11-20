// Package archtest は docs/architecture.md のレイヤー方針（依存は外側→内側のみ）を検査する。
// ソースの import を go/parser で静的に読むだけで、実ホーム・外部プロセスは使わない。
package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// layers はプロジェクト内パッケージ（モジュール相対パス。ルートは "."）の層番号。
// 依存は「自分より小さい層」にのみ許される（同層・上位層への import は違反）。
// 表にないパッケージは失敗する（新規パッケージは必ずここで層を決める）。
var layers = map[string]int{
	// L1 基盤: 他のプロジェクト内パッケージに依存しない
	"internal/fsutil":     1,
	"internal/osutil":     1,
	"internal/masking":    1,
	"internal/memory":     1,
	"internal/launchd":    1,
	"internal/scheduler":  1,
	"internal/skills":     1,
	"internal/extensions": 1,
	"internal/targets":    1,
	"internal/logmodel":   1,
	// L2 インフラ: 外部（ファイル・DB・プロセス・FS 監視）への出入口。L1 にのみ依存
	"internal/store":     2,
	"internal/config":    2,
	"internal/applier":   2,
	"internal/collector": 2,
	"internal/runner":    2,
	// L3 加工: インフラの結果を組み立てる
	"internal/digest": 3,
	// L4 ユースケース
	"internal/service": 4,
	// L5 エントリ（Wails バインディング）
	".": 5,
}

// testSupport は本番コードから import されない、テスト補助のパッケージ。層の対象外。
var testSupport = map[string]bool{
	"internal/testutil":            true,
	"internal/testutil/fakeclaude": true,
	"internal/archtest":            true,
}

// thirdPartyOwners は外部ライブラリを import してよいパッケージの一覧。
// 外部依存の入口を 1 か所に閉じ込める（差し替え・テストを容易にする）。
var thirdPartyOwners = map[string][]string{
	"github.com/wailsapp/wails/v2": {"."},
	"modernc.org/sqlite":           {"internal/store"},
	"github.com/fsnotify/fsnotify": {"internal/collector"},
}

// onlyImports は、層の規則に加えて import 先をさらに限定するパッケージ。
// エントリ（app.go）は薄いバインディングとして service だけを呼ぶ。
var onlyImports = map[string][]string{
	".": {"internal/service"},
}

// allowlistCap は許容リストの件数の固定上限。増やす変更は禁止（違反は直す）。
// リストを減らしたら、この値も同じだけ下げる（件数が一致しないと失敗する）。目標は 0。
const allowlistCap = 1

// allowlist は現状の違反（"from -> to"）と理由。リファクタリング（docs/architecture.md の移行計画）で 0 にする。
var allowlist = map[string]string{
	". -> internal/applier": "エントリが service を介さずに applier を直接使う。M3 で service に集約して解消",
}

type pkgInfo struct {
	imports map[string]bool // プロジェクト内の相対パス
	ext     map[string]bool // 外部（標準ライブラリ以外）の import パス
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod が見つからない")
		}
		dir = parent
	}
}

func modulePath(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(l, "module "))
		}
	}
	t.Fatal("go.mod に module がない")
	return ""
}

// scan は root 以下の非テスト Go ファイルの import をパッケージごとに集める。
func scan(t *testing.T, root string) map[string]*pkgInfo {
	t.Helper()
	mod := modulePath(t, root)
	skip := map[string]bool{"frontend": true, "node_modules": true, "testdata": true, ".git": true, ".claude": true, "build": true, "docs": true, "scripts": true}
	pkgs := map[string]*pkgInfo{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		rel = filepath.ToSlash(rel)
		p := pkgs[rel]
		if p == nil {
			p = &pkgInfo{imports: map[string]bool{}, ext: map[string]bool{}}
			pkgs[rel] = p
		}
		for _, im := range f.Imports {
			ip, _ := strconv.Unquote(im.Path.Value)
			switch {
			case ip == mod:
				p.imports["."] = true
			case strings.HasPrefix(ip, mod+"/"):
				p.imports[strings.TrimPrefix(ip, mod+"/")] = true
			case strings.Contains(strings.Split(ip, "/")[0], "."):
				p.ext[ip] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return pkgs
}

func onlyAllowed(from, to string) bool {
	only, ok := onlyImports[from]
	if !ok {
		return true
	}
	for _, o := range only {
		if o == to {
			return true
		}
	}
	return false
}

// violations は層の規則に反する "from -> to" を返す（許容リストは考慮しない）。
func violations(pkgs map[string]*pkgInfo) (found map[string]bool, problems []string) {
	found = map[string]bool{}
	for from, p := range pkgs {
		if testSupport[from] {
			continue
		}
		lf, ok := layers[from]
		if !ok {
			problems = append(problems, "層が未定義のパッケージ: "+from+"（archtest の layers に追加する）")
			continue
		}
		for to := range p.imports {
			if testSupport[to] {
				problems = append(problems, from+" がテスト補助 "+to+" を本番コードから import している")
				continue
			}
			lt, ok := layers[to]
			if !ok {
				problems = append(problems, "層が未定義の import 先: "+to)
				continue
			}
			if lt >= lf || !onlyAllowed(from, to) {
				found[from+" -> "+to] = true
			}
		}
		for e := range p.ext {
			for prefix, owners := range thirdPartyOwners {
				if e != prefix && !strings.HasPrefix(e, prefix+"/") {
					continue
				}
				allowed := false
				for _, o := range owners {
					allowed = allowed || o == from
				}
				if !allowed {
					problems = append(problems, from+" が外部ライブラリ "+e+" を import している（許可: "+strings.Join(owners, ", ")+"）")
				}
			}
		}
	}
	return found, problems
}

func TestLayerDependencies(t *testing.T) {
	pkgs := scan(t, repoRoot(t))
	found, problems := violations(pkgs)
	for v := range found {
		if _, ok := allowlist[v]; !ok {
			problems = append(problems, "層の規則違反（許容リストへの追加は禁止。依存の向きを直す）: "+v)
		}
	}
	for v := range allowlist {
		if !found[v] {
			problems = append(problems, "許容リストに残っているが、もう違反ではない（リストと allowlistCap から削除する）: "+v)
		}
	}
	sort.Strings(problems)
	for _, p := range problems {
		t.Error(p)
	}
}

func TestAllowlistCapIsFixed(t *testing.T) {
	if len(allowlist) != allowlistCap {
		t.Errorf("許容リストは %d 件、上限は %d 件。増やすのは禁止。減らしたら allowlistCap も同じだけ下げる", len(allowlist), allowlistCap)
	}
	for k, reason := range allowlist {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("許容リストの %q に理由がない", k)
		}
	}
}

// 検査自体が違反を検出できること（層の向き・未定義・外部ライブラリ）。
func TestViolationsDetectsProblems(t *testing.T) {
	pkgs := map[string]*pkgInfo{
		"internal/masking": {imports: map[string]bool{"internal/service": true}, ext: map[string]bool{}},
		"internal/store":   {imports: map[string]bool{"internal/runner": true}, ext: map[string]bool{"github.com/fsnotify/fsnotify": true}},
		"internal/newpkg":  {imports: map[string]bool{}, ext: map[string]bool{}},
		"internal/service": {imports: map[string]bool{"internal/store": true}, ext: map[string]bool{}},
	}
	found, problems := violations(pkgs)
	for _, want := range []string{"internal/masking -> internal/service", "internal/store -> internal/runner"} {
		if !found[want] {
			t.Errorf("違反 %q を検出できていない", want)
		}
	}
	if found["internal/service -> internal/store"] {
		t.Error("下位層への import を違反にしている")
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"層が未定義のパッケージ: internal/newpkg", "fsnotify"} {
		if !strings.Contains(joined, want) {
			t.Errorf("問題 %q を検出できていない: %s", want, joined)
		}
	}
}
