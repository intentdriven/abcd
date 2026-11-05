package testreuse

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The check directory's own files, beside the tree so nothing a test reads of
// the module holds them.
const (
	treeDirName  = "tree"
	timesName    = "times.json"
	lockName     = "lock"
	holderName   = "holder"
	incomingGlob = ".abcd-tmp-*"
)

// A file's time is timeBase plus 64 bits of the SHA-256 of its content reduced
// modulo timeWindow whole seconds: twenty years ending before 2021, so every
// time is in the past, where Go will cache a test that opened the file (cmd/go,
// errFileTooNew), and within what APFS and ext4 store. Go keys an opened file on
// its size and time (hashWriteStat), so a same-size edit is reused by accident
// about one time in 630 million; the time changes when the content does and is
// the same in every worktree, where a commit time would survive an amend.
var timeBase = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

const timeWindow = 20 * 365 * 24 * 60 * 60

// contentTime is the time a file or directory with this digest carries.
func contentTime(sum [sha256.Size]byte) time.Time {
	return timeBase.Add(time.Duration(binary.BigEndian.Uint64(sum[:8])%timeWindow) * time.Second)
}

// entry is one path of a commit's tree.
type entry struct {
	mode string
	oid  string
	size int64
	path string
}

// listTree reads a commit's tree. A path that could leave the tree, or lands in
// a directory named .git, is refused: what is copied here is executed.
func listTree(ctx context.Context, repo, commit string) ([]entry, error) {
	out, err := gitOutput(ctx, repo, "ls-tree", "-r", "-z", "-l", "--full-tree", commit)
	if err != nil {
		return nil, err
	}
	var entries []entry
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		meta, p, ok := strings.Cut(rec, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 4 {
			return nil, fmt.Errorf("git ls-tree: a line of an unexpected shape: %q", rec)
		}
		if !treePath(p) {
			return nil, fmt.Errorf("the commit names a path outside its tree: %q", p)
		}
		e := entry{mode: fields[0], oid: fields[2], path: p}
		if fields[3] != "-" {
			if e.size, err = strconv.ParseInt(fields[3], 10, 64); err != nil {
				return nil, fmt.Errorf("git ls-tree: size %q: %w", fields[3], err)
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// treePath reports whether a commit's path stays in the tree and out of a .git.
func treePath(p string) bool {
	return filepath.IsLocal(p) && !slices.Contains(strings.Split(p, "/"), ".git")
}

// gitOutput runs a read-only git question in repo, under no variable an outer
// git set: a hook's GIT_DIR must not answer for another repository.
func gitOutput(ctx context.Context, repo string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repo
	cmd.Env = gitEnv()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, bytes.TrimSpace(stderr.Bytes()))
	}
	return out, nil
}

func gitEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		switch name {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES":
			continue
		}
		env = append(env, kv)
	}
	return env
}

// blobReader is one `git cat-file --batch`, started when first needed.
type blobReader struct {
	ctx  context.Context
	repo string
	cmd  *exec.Cmd
	in   io.WriteCloser
	out  *bufio.Reader
}

func (b *blobReader) read(oid string) ([]byte, error) {
	if b.cmd == nil {
		cmd := exec.CommandContext(b.ctx, "git", "cat-file", "--batch")
		cmd.Dir = b.repo
		cmd.Env = gitEnv()
		in, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}
		out, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		b.cmd, b.in, b.out = cmd, in, bufio.NewReader(out)
	}
	if _, err := io.WriteString(b.in, oid+"\n"); err != nil {
		return nil, err
	}
	head, err := b.out.ReadString('\n')
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(head)
	if len(fields) != 3 || fields[0] != oid {
		return nil, fmt.Errorf("git cat-file: %q for %s", strings.TrimSpace(head), oid)
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, err
	}
	body := make([]byte, size+1) // the content and its newline
	if _, err := io.ReadFull(b.out, body); err != nil {
		return nil, err
	}
	return body[:size], nil
}

func (b *blobReader) close() {
	if b.cmd != nil {
		_ = b.in.Close()
		_ = b.cmd.Wait()
	}
}

// SyncStats is what a sync did to the check directory.
type SyncStats struct {
	Written, Removed, Retimed, Kept int
	Took                            time.Duration
}

// snapshot is the tree a sync left, to read back after the tests have run.
type snapshot struct {
	tree    string
	entries []entry
	times   map[string]int64
}

// loadTimes reads the memo of each blob's derived time, so a sync reads only
// blobs it has not seen. It is a memo: a file a crash cut short, or any other
// that does not read, costs one read of every blob and nothing else.
func loadTimes(dir string) map[string]int64 {
	times := map[string]int64{}
	raw, err := os.ReadFile(filepath.Join(dir, timesName))
	if err != nil {
		return times
	}
	if err := json.Unmarshal(raw, &times); err != nil {
		return map[string]int64{}
	}
	return times
}

func saveTimes(dir string, times map[string]int64) error {
	raw, err := json.Marshal(times)
	if err != nil {
		return err
	}
	return fsutil.WriteRebuildable(filepath.Join(dir, timesName), raw, 0o600, time.Now())
}

// Sync makes dir's tree exactly the commit's: every tracked file with its mode
// and content-derived time, every directory with a time derived from the names
// in it, and nothing else. It writes only what differs, so a commit that changes
// one file writes one file and a repository whose tree has thousands of files
// creates that churn once. A sync interrupted anywhere leaves a tree the next one
// repairs: a file is renamed into place with its time already set, what a test
// or a crash left is removed, and a directory's time is set last.
func Sync(ctx context.Context, repo, commit, dir string) (SyncStats, error) {
	stats, _, err := syncTree(ctx, repo, commit, dir)
	return stats, err
}

func syncTree(ctx context.Context, repo, commit, dir string) (SyncStats, *snapshot, error) {
	began := time.Now()
	var stats SyncStats
	leftovers, err := filepath.Glob(filepath.Join(dir, incomingGlob))
	if err != nil {
		return stats, nil, err
	}
	for _, l := range leftovers {
		if err := os.Remove(l); err != nil {
			return stats, nil, err
		}
	}
	entries, err := listTree(ctx, repo, commit)
	if err != nil {
		return stats, nil, err
	}
	tree := filepath.Join(dir, treeDirName)
	if err := os.MkdirAll(tree, 0o755); err != nil {
		return stats, nil, err
	}

	wantFile := make(map[string]bool, len(entries))
	wantDir := map[string]map[string]string{"": {}}
	for _, e := range entries {
		wantFile[e.path] = true
		child := e.path
		if e.mode == "160000" {
			wantDir[e.path] = map[string]string{}
			delete(wantFile, e.path)
		}
		for child != "." {
			parent := path.Dir(child)
			if parent == "." {
				parent = ""
			}
			kids := wantDir[parent]
			if kids == nil {
				kids = map[string]string{}
				wantDir[parent] = kids
			}
			kind := "dir"
			if child == e.path && e.mode != "160000" {
				kind = e.mode
			}
			kids[path.Base(child)] = kind
			if parent == "" {
				break
			}
			child = parent
		}
	}
	if err := removeExtras(tree, "", wantFile, wantDir, &stats); err != nil {
		return stats, nil, err
	}

	times := loadTimes(dir)
	seen := make(map[string]int64, len(entries))
	blobs := &blobReader{ctx: ctx, repo: repo}
	defer blobs.close()
	for _, e := range entries {
		if e.mode == "160000" {
			continue
		}
		abs := filepath.Join(tree, filepath.FromSlash(e.path))
		info, statErr := os.Lstat(abs)
		sec, known := times[e.oid]
		if known && statErr == nil && e.matches(info, sec, abs) {
			seen[e.oid] = sec
			stats.Kept++
			continue
		}
		body, err := blobs.read(e.oid)
		if err != nil {
			return stats, nil, err
		}
		sec = contentTime(sha256.Sum256(body)).Unix()
		seen[e.oid] = sec
		if statErr == nil && e.matches(info, sec, abs) {
			stats.Kept++
			continue
		}
		if err := place(abs, e, body, sec); err != nil {
			return stats, nil, err
		}
		stats.Written++
	}

	dirs := make([]string, 0, len(wantDir))
	for d := range wantDir {
		dirs = append(dirs, d)
	}
	slices.Sort(dirs)
	for _, d := range dirs {
		abs := filepath.Join(tree, filepath.FromSlash(d))
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return stats, nil, err
		}
		info, err := os.Lstat(abs)
		if err != nil {
			return stats, nil, err
		}
		if info.Mode().Perm() != 0o755 {
			if err := os.Chmod(abs, 0o755); err != nil {
				return stats, nil, err
			}
		}
		at := dirTime(wantDir[d])
		if !info.ModTime().Equal(at) {
			if err := os.Chtimes(abs, at, at); err != nil {
				return stats, nil, err
			}
			stats.Retimed++
		}
	}
	if err := saveTimes(dir, seen); err != nil {
		return stats, nil, err
	}
	stats.Took = time.Since(began)
	return stats, &snapshot{tree: tree, entries: entries, times: seen}, nil
}

// dirTime derives a directory's time from the names in it and each one's kind,
// as a file system's own directory time moves when an entry comes or goes and
// not when a file in it is rewritten.
func dirTime(kids map[string]string) time.Time {
	names := make([]string, 0, len(kids))
	for n := range kids {
		names = append(names, n)
	}
	slices.Sort(names)
	h := sha256.New()
	for _, n := range names {
		fmt.Fprintf(h, "%s\x00%s\x00", n, kids[n])
	}
	var sum [sha256.Size]byte
	h.Sum(sum[:0])
	return contentTime(sum)
}

func (e entry) perm() fs.FileMode {
	if e.mode == "100755" {
		return 0o755
	}
	return 0o644
}

// matches reports whether what is on disk is this entry: a regular file of its
// size, mode and derived time, or a link to its target.
func (e entry) matches(info fs.FileInfo, sec int64, abs string) bool {
	if e.mode == "120000" {
		if info.Mode()&fs.ModeSymlink == 0 {
			return false
		}
		target, err := os.Readlink(abs)
		return err == nil && int64(len(target)) == e.size
	}
	return info.Mode().IsRegular() && info.Mode().Perm() == e.perm() && info.Size() == e.size &&
		info.ModTime().Unix() == sec && info.ModTime().Nanosecond() == 0
}

// place puts one entry at abs, a file with its time already set, so a crash
// leaves the old file or the new one and never a part of one with the time that
// would pass it as whole.
func place(abs string, e entry, body []byte, sec int64) error {
	if info, err := os.Lstat(abs); err == nil && info.IsDir() {
		if err := forceRemove(abs); err != nil {
			return err
		}
	}
	if e.mode == "120000" {
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		if err := os.RemoveAll(abs); err != nil {
			return err
		}
		return os.Symlink(string(body), abs)
	}
	return fsutil.WriteRebuildable(abs, body, e.perm(), time.Unix(sec, 0))
}

// removeExtras removes everything under dir the commit does not hold.
func removeExtras(root, rel string, wantFile map[string]bool, wantDir map[string]map[string]string, stats *SyncStats) error {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.Chmod(abs, 0o755); err != nil {
		return err
	}
	items, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	for _, it := range items {
		p := path.Join(rel, it.Name())
		_, isDir := wantDir[p]
		switch {
		case it.IsDir() && isDir:
			if err := removeExtras(root, p, wantFile, wantDir, stats); err != nil {
				return err
			}
		case !it.IsDir() && wantFile[p]:
		default:
			if err := forceRemove(filepath.Join(abs, it.Name())); err != nil {
				return err
			}
			stats.Removed++
		}
	}
	return nil
}

// forceRemove removes a path a test may have made unwritable.
func forceRemove(p string) error {
	err := os.RemoveAll(p)
	if err == nil || !errors.Is(err, fs.ErrPermission) {
		return err
	}
	_ = filepath.WalkDir(p, func(sub string, d fs.DirEntry, walkErr error) error {
		if d != nil && d.IsDir() {
			_ = os.Chmod(sub, 0o755)
		}
		return nil
	})
	return os.RemoveAll(p)
}

// leftBehind names what is in the tree that the commit does not hold: what a test
// left there. Each moved the time of the directory that holds it, so a test that
// walks the tree hashes that time and does not replay on the next check.
func (s *snapshot) leftBehind() []string {
	tracked := make(map[string]bool, len(s.entries))
	for _, e := range s.entries {
		tracked[e.path] = true
	}
	var extra []string
	_ = filepath.WalkDir(s.tree, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if rel, err := filepath.Rel(s.tree, p); err == nil && !tracked[filepath.ToSlash(rel)] {
			extra = append(extra, filepath.ToSlash(rel))
		}
		return nil
	})
	return extra
}

// changedTracked names the tracked files that no longer match the commit: what
// a test run rewrote in place, which leaves what it read no longer the commit.
func (s *snapshot) changedTracked() []string {
	var changed []string
	for _, e := range s.entries {
		if e.mode == "160000" {
			continue
		}
		abs := filepath.Join(s.tree, filepath.FromSlash(e.path))
		info, err := os.Lstat(abs)
		if err != nil || !e.matches(info, s.times[e.oid], abs) {
			changed = append(changed, e.path)
		}
	}
	return changed
}
