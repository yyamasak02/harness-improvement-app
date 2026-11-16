// Package applier はファイルへの書き込みを一手に引き受ける唯一のパッケージ。
// 手順: 許可リスト確認 → SHA-256 照合 → バックアップ → temp 書き込み(fsync) → rename。
package applier

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"harness-app/internal/fsutil"
)

// AbsentHash は存在しないファイルのハッシュ値。
const AbsentHash = "absent"

var (
	// ErrNotAllowed は許可リスト外のパスへの書き込み要求。
	ErrNotAllowed = errors.New("書き込みが許可されていないパスです")
	// ErrHashMismatch は改善案生成後に対象ファイルが変更されたことを示す。
	ErrHashMismatch = errors.New("対象ファイルが改善案の生成後に変更されています。再生成してください")
	// ErrSymlink はシンボリックリンクへの書き込み拒否。
	ErrSymlink = errors.New("シンボリックリンクには書き込めません")
)

// Applier はバックアップ付きの書き込み器。
type Applier struct {
	DataDir string                 // バックアップは <DataDir>/backups/ に保存する
	Allowed func(path string) bool // 書き込み許可判定（許可リスト）
	Now     func() time.Time       // テスト用に差し替え可能
}

// Manifest はバックアップの manifest.json。
type Manifest struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Op         string `json:"op"` // apply | rollback
	Time       string `json:"time"`
	Existed    bool   `json:"existed"` // 操作前にファイルが存在したか
	OldHash    string `json:"oldHash"` // 操作前のハッシュ
	NewHash    string `json:"newHash"` // 操作後のハッシュ（削除なら AbsentHash）
	RollbackOf string `json:"rollbackOf,omitempty"`
}

// Request は適用要求。
type Request struct {
	Path         string
	Content      string
	ExpectedHash string // 改善案生成時点のハッシュ（HashFile の値）
}

// HashFile はファイルの SHA-256（hex）。存在しなければ AbsentHash。
func HashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return AbsentHash, nil
	}
	if err != nil {
		return "", fmt.Errorf("ハッシュ計算: %w", err)
	}
	return HashBytes(b), nil
}

// HashBytes は SHA-256（hex）。
func HashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (a *Applier) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *Applier) check(path string) error {
	if a.Allowed == nil || !a.Allowed(path) {
		return fmt.Errorf("%s: %w", path, ErrNotAllowed)
	}
	if st, err := os.Lstat(path); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s: %w", path, ErrSymlink)
	}
	return nil
}

func (a *Applier) backupsDir() string { return filepath.Join(a.DataDir, "backups") }

// newBackupDir は <timestamp>-<hash8> のディレクトリを衝突しないよう作る。
func (a *Applier) newBackupDir(hash string) (string, string, error) {
	if err := os.MkdirAll(a.backupsDir(), 0o755); err != nil {
		return "", "", fmt.Errorf("backups 作成: %w", err)
	}
	h8 := hash
	if len(h8) > 8 {
		h8 = h8[:8]
	}
	base := a.now().UTC().Format("20060102T150405.000") + "-" + h8
	for i := 0; i < 1000; i++ {
		id := base
		if i > 0 {
			id = fmt.Sprintf("%s-%d", base, i)
		}
		dir := filepath.Join(a.backupsDir(), id)
		err := os.Mkdir(dir, 0o755)
		if err == nil {
			return id, dir, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", "", fmt.Errorf("バックアップ dir 作成: %w", err)
		}
	}
	return "", "", errors.New("バックアップ dir を作れません")
}

// write は backup → atomic write を行う。content==nil ならファイル削除（新規作成の取り消し）。
func (a *Applier) write(path string, content []byte, op, rollbackOf string) (Manifest, error) {
	old, err := os.ReadFile(path)
	existed := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, fmt.Errorf("現ファイル読み込み: %w", err)
	}
	oldHash := AbsentHash
	if existed {
		oldHash = HashBytes(old)
	}
	id, dir, err := a.newBackupDir(oldHash)
	if err != nil {
		return Manifest{}, err
	}
	fail := func(e error) (Manifest, error) { _ = os.RemoveAll(dir); return Manifest{}, e }
	if existed {
		if err := fsutil.WriteFileAtomic(filepath.Join(dir, "original"), old, 0o600); err != nil {
			return fail(fmt.Errorf("バックアップ保存: %w", err))
		}
	}
	newHash := AbsentHash
	if content != nil {
		newHash = HashBytes(content)
		if err := fsutil.WriteFileAtomic(filepath.Join(dir, "applied"), content, 0o600); err != nil {
			return fail(fmt.Errorf("適用内容の保存: %w", err))
		}
	}
	m := Manifest{ID: id, Path: path, Op: op, Time: a.now().UTC().Format(time.RFC3339), Existed: existed,
		OldHash: oldHash, NewHash: newHash, RollbackOf: rollbackOf}
	mb, _ := json.MarshalIndent(m, "", "  ")
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, "manifest.json"), mb, 0o600); err != nil {
		return fail(fmt.Errorf("manifest 保存: %w", err))
	}
	if content == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fail(fmt.Errorf("ファイル削除: %w", err))
		}
		return m, nil
	}
	perm := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		perm = st.Mode().Perm()
	}
	if err := fsutil.WriteFileAtomic(path, content, perm); err != nil {
		return fail(fmt.Errorf("書き込み: %w", err))
	}
	return m, nil
}

// Apply は改善案を適用する。
func (a *Applier) Apply(r Request) (Manifest, error) {
	if err := a.check(r.Path); err != nil {
		return Manifest{}, err
	}
	cur, err := HashFile(r.Path)
	if err != nil {
		return Manifest{}, err
	}
	if cur != r.ExpectedHash {
		return Manifest{}, fmt.Errorf("%s: %w", r.Path, ErrHashMismatch)
	}
	return a.write(r.Path, []byte(r.Content), "apply", "")
}

// Entry は履歴 1 件（manifest と保存済み内容）。
type Entry struct {
	Manifest
	Before string `json:"before"` // 操作前の内容（存在しなければ空）
	After  string `json:"after"`  // 操作後の内容（削除なら空）
}

func (a *Applier) readEntry(id string, withBody bool) (Entry, error) {
	dir := filepath.Join(a.backupsDir(), id)
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Entry{}, fmt.Errorf("manifest 読み込み %s: %w", id, err)
	}
	var e Entry
	if err := json.Unmarshal(b, &e.Manifest); err != nil {
		return Entry{}, fmt.Errorf("manifest 解析 %s: %w", id, err)
	}
	if withBody {
		if o, err := os.ReadFile(filepath.Join(dir, "original")); err == nil {
			e.Before = string(o)
		}
		if o, err := os.ReadFile(filepath.Join(dir, "applied")); err == nil {
			e.After = string(o)
		}
	}
	return e, nil
}

// History は履歴を新しい順に返す（本文なし）。
func (a *Applier) History() ([]Entry, error) {
	ents, err := os.ReadDir(a.backupsDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("履歴の列挙: %w", err)
	}
	var out []Entry
	for _, d := range ents {
		if !d.IsDir() {
			continue
		}
		e, err := a.readEntry(d.Name(), false)
		if err != nil {
			continue // 壊れた履歴は無視
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

// Get は履歴 1 件を本文付きで返す。
func (a *Applier) Get(id string) (Entry, error) {
	if id == "" || filepath.Base(id) != id {
		return Entry{}, fmt.Errorf("不正な ID: %q", id)
	}
	return a.readEntry(id, true)
}

// Rollback は履歴 id の操作前の状態へ戻す。ロールバック自体もバックアップを取る。
func (a *Applier) Rollback(id string) (Manifest, error) {
	e, err := a.Get(id)
	if err != nil {
		return Manifest{}, err
	}
	if e.Op != "apply" {
		return Manifest{}, fmt.Errorf("apply 以外の履歴はロールバックできません: %s", e.Op)
	}
	if err := a.check(e.Path); err != nil {
		return Manifest{}, err
	}
	if !e.Existed {
		return a.write(e.Path, nil, "rollback", id)
	}
	return a.write(e.Path, []byte(e.Before), "rollback", id)
}
