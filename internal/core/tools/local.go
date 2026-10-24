package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The installed-tools record is how abcd knows a copy is its own: the
// absolute path of each program it installed and the sha256 of the bytes it
// wrote there, in ~/.abcd.noindex/tools.json beside the path-entry record that
// vouches for abcd's own binary. Ownership is recorded provenance, never
// content-guessing (brief invariant 12's rule, applied to the tools): a
// program abcd runs by its recorded path only while its bytes still hash to
// the record, and removes only on that same condition, so a file someone else
// put at that path, or changed since, is never run as abcd's and never
// deleted.

// recordRel is the record's place in the home, in the slash form the
// home-scoped primitives take.
var recordRel = abcdhome.Rel("tools.json")

// maxRecordBytes bounds the record read.
const maxRecordBytes = 64 << 10

// Record is the installed-tools record.
type Record struct {
	SchemaVersion int                  `json:"schema_version"`
	Tools         map[string]Installed `json:"tools"`
}

// Installed is one copy abcd installed.
type Installed struct {
	// Path is the program's absolute path.
	Path string `json:"path"`
	// SHA256 is the digest of the bytes abcd wrote there.
	SHA256 string `json:"sha256"`
	// Version is the pinned release it came from.
	Version string `json:"version"`
	// AssetSHA256 is the digest of the downloaded archive, which matched the
	// pin.
	AssetSHA256 string `json:"asset_sha256"`
	// Checked is how far the download was checked, as the result said it.
	Checked string `json:"checked"`
}

// userBinDir is the folder abcd installs a tool's program into, in uv's order:
// $XDG_BIN_HOME, then $XDG_DATA_HOME/../bin, then ~/.local/bin, each only when
// absolute. It is the person's own folder: nothing here needs admin rights.
// Empty when home is not absolute.
func userBinDir(getenv func(string) string, home string) string {
	if p := getenv("XDG_BIN_HOME"); filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	if p := getenv("XDG_DATA_HOME"); filepath.IsAbs(p) {
		return filepath.Join(p, "..", "bin")
	}
	if !filepath.IsAbs(home) {
		return ""
	}
	return filepath.Join(home, ".local", "bin")
}

// readRecord reads the record from home. An absent record is an empty one; a
// record that does not pass the home-declaration guards (owned by this
// account, writable by no one else, not behind a symlink) or does not parse
// is an error, and reads as no tool installed by abcd.
func readRecord(home string) (Record, error) {
	rec := Record{SchemaVersion: 1, Tools: map[string]Installed{}}
	if !filepath.IsAbs(home) {
		return rec, errors.New("no absolute home directory to read " + abcdhome.Display("tools.json") + " from")
	}
	raw, refusal, err := fsutil.ReadHomeDeclaration(home, recordRel, maxRecordBytes)
	if refusal == fsutil.DeclarationAbsent {
		return rec, nil
	}
	if err != nil {
		return rec, err
	}
	var got Record
	if err := json.Unmarshal(raw, &got); err != nil {
		return rec, errors.New(abcdhome.Display("tools.json") + " does not parse: " + err.Error())
	}
	for name, e := range got.Tools {
		if Known(name) && filepath.IsAbs(e.Path) && hexDigest(e.SHA256) {
			rec.Tools[name] = e
		}
	}
	return rec, nil
}

// writeRecord writes rec into home, creating ~/.abcd.noindex at the account's
// own mode and writing through its descriptor; an empty record removes the
// file.
func writeRecord(home string, rec Record) error {
	if err := fsutil.HomeScopeLink(home, recordRel); err != nil {
		return err
	}
	dir, err := fsutil.EnsureHomeScope(home, path.Dir(recordRel), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer dir.Close()
	if len(rec.Tools) == 0 {
		if err := dir.Remove(path.Base(recordRel)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	rec.SchemaVersion = 1
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(dir, path.Base(recordRel), append(body, '\n'), abcdhome.FileMode)
}

func hexDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && strings.ToLower(s) == s
}

// fileDigest hashes the regular file at p, never following a symlink at the
// leaf.
func fileDigest(p string) (string, bool) {
	fi, err := os.Lstat(p)
	if err != nil || !fi.Mode().IsRegular() {
		return "", false
	}
	f, err := os.Open(p)
	if err != nil {
		return "", false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, io.LimitReader(f, maxProgramBytes+1)); err != nil {
		return "", false
	}
	return hex.EncodeToString(h.Sum(nil)), true
}

// intactCache remembers a digest judgement for one (path, size, mtime), so a
// verb that runs gh a dozen times hashes it once.
var intactCache sync.Map

type intactKey struct {
	path  string
	size  int64
	mtime int64
}

// intact reports whether e's program is still the bytes abcd wrote.
func intact(e Installed) bool {
	fi, err := os.Lstat(e.Path)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	k := intactKey{e.Path, fi.Size(), fi.ModTime().UnixNano()}
	if v, ok := intactCache.Load(k); ok {
		return v.(string) == e.SHA256
	}
	got, ok := fileDigest(e.Path)
	if !ok {
		return false
	}
	intactCache.Store(k, got)
	return got == e.SHA256
}

// owned is abcd's own recorded copy of name in home, when the record names one
// and its bytes still hash to the record.
func owned(home, name string) (Installed, bool) {
	rec, err := readRecord(home)
	if err != nil {
		return Installed{}, false
	}
	e, ok := rec.Tools[name]
	if !ok || !intact(e) {
		return Installed{}, false
	}
	return e, true
}

// userHome is the person's home directory, a seam for tests.
var userHome = os.UserHomeDir

// pathLookup is the PATH lookup Resolve falls back to, a seam for tests.
var pathLookup = exec.LookPath

// Resolve is where abcd runs name from: abcd's own recorded copy when there is
// one and it is intact, whatever comes first on PATH (the intent's fourth
// criterion), and otherwise the PATH lookup. Every caller that runs a
// registered tool resolves it here, never by a bare exec.LookPath.
func Resolve(name string) (string, error) {
	if home, err := userHome(); err == nil {
		if e, ok := owned(home, name); ok {
			return e.Path, nil
		}
	}
	return pathLookup(name)
}

// Removal reports what uninstall did with one recorded copy.
type Removal struct {
	Tool    string `json:"tool"`
	Path    string `json:"path"`
	Removed bool   `json:"removed"`
	Note    string `json:"note"`
}

// Uninstall removes the copies the record in home names, and only those, and
// only while each is still the bytes abcd wrote: a file changed since, or
// replaced by another, is left where it is and said so. The record goes once
// nothing it names is still abcd's to remove. Nothing else is touched: a copy
// installed some other way was never recorded, so it is never reached.
func Uninstall(home string) ([]Removal, error) {
	rec, err := readRecord(home)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rec.Tools))
	for n := range rec.Tools {
		names = append(names, n)
	}
	sort.Strings(names)
	out := []Removal{}
	keep := map[string]Installed{}
	for _, n := range names {
		e := rec.Tools[n]
		r := Removal{Tool: n, Path: displayIn(home, e.Path)}
		switch {
		case !exists(e.Path):
			r.Note = "already gone"
		case !intact(e):
			r.Note = "changed since abcd installed it, so it is not abcd's to remove; left untouched"
		default:
			if err := os.Remove(e.Path); err != nil {
				r.Note = "remove failed: " + err.Error()
				keep[n] = e
			} else {
				r.Removed = true
				r.Note = "removed abcd's copy"
			}
		}
		out = append(out, r)
	}
	if len(names) > 0 {
		if err := writeRecord(home, Record{Tools: keep}); err != nil {
			return out, err
		}
	}
	return out, nil
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// displayIn is p in the tilde form under home, so a message pasted into an
// issue never carries the username.
func displayIn(home, p string) string {
	if home != "" && filepath.IsAbs(home) {
		if p == home {
			return "~"
		}
		if strings.HasPrefix(p, home+string(filepath.Separator)) {
			return "~" + p[len(home):]
		}
	}
	return p
}
