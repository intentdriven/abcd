package testreuse

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	"sync"
	"syscall"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The check directory's own files, beside the tree so nothing a test reads of
// the module holds them. A sync removes anything else it finds there.
const (
	treeDirName = "tree"
	lockName    = "lock"
	holderName  = "holder"
)

// A file's time is timeBase plus bits of the SHA-256 of its content: 64 reduced
// modulo timeWindow whole seconds, twenty years ending before 2021, so every time
// is in the past, where Go will cache a test that opened the file (cmd/go,
// errFileTooNew), and 64 more reduced to the nanoseconds of that second. Go keys
// an opened file on its size and time (hashWriteStat), so the key moves when the
// content does and is the same in every worktree, where a commit time would
// survive an amend. The time is never how the sync knows a file is the commit's:
// it reads the bytes (onDisk).
var timeBase = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

const timeWindow = 20 * 365 * 24 * 60 * 60

// contentTime is the time a file or directory with this digest carries.
func contentTime(sum [sha256.Size]byte) time.Time {
	sec := binary.BigEndian.Uint64(sum[:8]) % timeWindow
	nsec := binary.BigEndian.Uint64(sum[8:16]) % uint64(time.Second)
	return timeBase.Add(time.Duration(sec)*time.Second + time.Duration(nsec))
}

// blobOID is the object name git gives this content, in the format oid is in.
func blobOID(oid string, body []byte) string {
	head := fmt.Sprintf("blob %d\x00", len(body))
	switch len(oid) {
	case sha1.Size * 2:
		h := sha1.New()
		h.Write([]byte(head))
		h.Write(body)
		return hex.EncodeToString(h.Sum(nil))
	case sha256.Size * 2:
		h := sha256.New()
		h.Write([]byte(head))
		h.Write(body)
		return hex.EncodeToString(h.Sum(nil))
	}
	return ""
}

// entry is one path of a commit's tree.
type entry struct {
	mode string
	oid  string
	size int64
	path string
}

// listTree reads a commit's tree. What is copied here is executed, so it refuses
// a path that could leave the tree, that lands in a directory named .git in any
// spelling, or that a file system could take for another of the commit's paths.
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
	if err := oneSpelling(entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// treePath reports whether a commit's path stays in the tree and out of a .git.
func treePath(p string) bool {
	if !filepath.IsLocal(p) {
		return false
	}
	return !slices.ContainsFunc(strings.Split(p, "/"), func(part string) bool {
		return strings.EqualFold(norm.NFC.String(part), ".git")
	})
}

// oneSpelling refuses a commit two of whose paths one file system would take for
// one, by letter case or by Unicode form, and a path under a file or a link: on
// such a file system the second would land wherever the first points.
func oneSpelling(entries []entry) error {
	fold := func(p string) string { return strings.ToLower(norm.NFC.String(p)) }
	files := make(map[string]string, len(entries))
	dirs := map[string]string{}
	for _, e := range entries {
		key := fold(e.path)
		if other, dup := files[key]; dup {
			return fmt.Errorf("the commit names one file twice: %q and %q", other, e.path)
		}
		files[key] = e.path
		for d := path.Dir(e.path); d != "."; d = path.Dir(d) {
			key := fold(d)
			if other, dup := dirs[key]; dup && other != d {
				return fmt.Errorf("the commit names one directory twice: %q and %q", other, d)
			}
			dirs[key] = d
		}
	}
	for key, d := range dirs {
		if f, under := files[key]; under {
			return fmt.Errorf("the commit names %q as a file and as the directory %q", f, d)
		}
	}
	return nil
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
}

// Sync makes dir's tree exactly the commit's: every tracked file with its mode
// and content-derived time, every directory with a time derived from the names
// in it, and nothing else. It reads every file back and keeps one only when its
// bytes are the commit's, and it writes only what differs, so a commit that
// changes one file writes one file and a repository whose tree has thousands of
// files creates that churn once. Every write goes through an os.Root on the tree,
// so nothing a test or a commit left there takes one elsewhere. A sync interrupted
// anywhere leaves a tree the next one repairs: a file is renamed into place with
// its time already set, what a test or a crash left is removed, and a directory's
// time is set last.
func Sync(ctx context.Context, repo, commit, dir string) (SyncStats, error) {
	stats, _, err := syncTree(ctx, repo, commit, dir)
	return stats, err
}

// clearBeside readies the check directory: anything but the tree and the two
// notes goes. A go.work here would be read above the module, and a link in the
// tree's place would take the sync's writes elsewhere, so the check that finds one
// refuses and removes it, and the next one starts clean.
func clearBeside(top *os.Root) error {
	f, err := top.Open(".")
	if err != nil {
		return err
	}
	items, err := f.ReadDir(-1)
	_ = f.Close()
	if err != nil {
		return err
	}
	for _, it := range items {
		switch it.Name() {
		case lockName, holderName:
		case treeDirName:
			if it.IsDir() {
				continue
			}
			kind := it.Type()
			if err := top.RemoveAll(treeDirName); err != nil {
				return err
			}
			if kind&fs.ModeSymlink != 0 {
				return errors.New("the check directory's tree was a link, which was removed")
			}
		default:
			if err := forceRemove(top, it.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}

func syncTree(ctx context.Context, repo, commit, dir string) (SyncStats, *snapshot, error) {
	began := time.Now()
	var stats SyncStats
	entries, err := listTree(ctx, repo, commit)
	if err != nil {
		return stats, nil, err
	}
	top, err := os.OpenRoot(dir)
	if err != nil {
		return stats, nil, err
	}
	defer top.Close()
	if err := clearBeside(top); err != nil {
		return stats, nil, err
	}
	if err := top.Mkdir(treeDirName, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return stats, nil, err
	}
	root, err := top.OpenRoot(treeDirName)
	if err != nil {
		return stats, nil, err
	}
	defer root.Close()

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
	if err := removeExtras(root, "", wantFile, wantDir, &stats); err != nil {
		return stats, nil, err
	}

	wrong := differing(root, entries)
	stats.Kept = len(entries) - len(wrong)
	blobs := &blobReader{ctx: ctx, repo: repo}
	defer blobs.close()
	for _, e := range wrong {
		if e.mode == "160000" {
			stats.Kept++
			continue
		}
		body, err := blobs.read(e.oid)
		if err != nil {
			return stats, nil, err
		}
		if err := place(root, e, body); err != nil {
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
		rel := d
		if rel == "" {
			rel = "."
		} else if err := root.MkdirAll(rel, 0o755); err != nil {
			return stats, nil, err
		}
		info, err := root.Lstat(rel)
		if err != nil {
			return stats, nil, err
		}
		if info.Mode().Perm() != 0o755 {
			if err := root.Chmod(rel, 0o755); err != nil {
				return stats, nil, err
			}
		}
		at := dirTime(wantDir[d])
		if !info.ModTime().Equal(at) {
			if err := root.Chtimes(rel, at, at); err != nil {
				return stats, nil, err
			}
			stats.Retimed++
		}
	}
	stats.Took = time.Since(began)
	return stats, &snapshot{tree: filepath.Join(dir, treeDirName), entries: entries}, nil
}

// differing is the entries the tree does not hold, read on a few goroutines: a sync
// reads every file of the commit, and a run reads them all again.
func differing(root *os.Root, entries []entry) []entry {
	wrong := make([]bool, len(entries))
	var wg sync.WaitGroup
	next := make(chan int)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				wrong[i] = !entries[i].onDisk(root)
			}
		}()
	}
	for i := range entries {
		next <- i
	}
	close(next)
	wg.Wait()
	var out []entry
	for i, e := range entries {
		if wrong[i] {
			out = append(out, e)
		}
	}
	return out
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

// onDisk reports whether the tree holds this entry: a regular file of its mode
// whose bytes are the blob and whose time derives from them, or a link whose target
// is the blob, or a directory where the commit has a submodule.
func (e entry) onDisk(root *os.Root) bool {
	info, err := root.Lstat(e.path)
	if err != nil {
		return false
	}
	switch e.mode {
	case "160000":
		return info.IsDir()
	case "120000":
		if info.Mode()&fs.ModeSymlink == 0 {
			return false
		}
		target, err := root.Readlink(e.path)
		return err == nil && blobOID(e.oid, []byte(target)) == e.oid
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != e.perm() || info.Size() != e.size {
		return false
	}
	body, err := root.ReadFile(e.path)
	if err != nil {
		return false
	}
	return blobOID(e.oid, body) == e.oid && info.ModTime().Equal(contentTime(sha256.Sum256(body)))
}

// place puts one entry in the tree, a file with its time already set, so a crash
// leaves the old file or the new one and never a part of one with the time that
// would pass it as whole.
func place(root *os.Root, e entry, body []byte) error {
	if info, err := root.Lstat(e.path); err == nil && info.IsDir() {
		if err := forceRemove(root, e.path); err != nil {
			return err
		}
	}
	if e.mode == "120000" {
		if dir := path.Dir(e.path); dir != "." {
			if err := root.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		if err := root.RemoveAll(e.path); err != nil {
			return err
		}
		return root.Symlink(string(body), e.path)
	}
	return fsutil.WriteRebuildableAtInRoot(root, e.path, body, e.perm(), contentTime(sha256.Sum256(body)))
}

// removeExtras removes everything under rel the commit does not hold.
func removeExtras(root *os.Root, rel string, wantFile map[string]bool, wantDir map[string]map[string]string, stats *SyncStats) error {
	name := rel
	if name == "" {
		name = "."
	}
	if err := root.Chmod(name, 0o755); err != nil {
		return err
	}
	f, err := root.Open(name)
	if err != nil {
		return err
	}
	items, err := f.ReadDir(-1)
	_ = f.Close()
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
			if err := forceRemove(root, p); err != nil {
				return err
			}
			stats.Removed++
		}
	}
	return nil
}

// forceRemove removes a path a test may have made unwritable.
func forceRemove(root *os.Root, rel string) error {
	err := root.RemoveAll(rel)
	if err == nil || !errors.Is(err, fs.ErrPermission) {
		return err
	}
	var open func(string)
	open = func(d string) {
		if root.Chmod(d, 0o755) != nil {
			return
		}
		f, err := root.Open(d)
		if err != nil {
			return
		}
		items, _ := f.ReadDir(-1)
		_ = f.Close()
		for _, it := range items {
			if it.IsDir() {
				open(path.Join(d, it.Name()))
			}
		}
	}
	open(rel)
	return root.RemoveAll(rel)
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

// changedTracked names the tracked files whose bytes, mode or time are no longer
// the commit's: what a test run rewrote, which leaves what it read not the commit.
func (s *snapshot) changedTracked() []string {
	root, err := os.OpenRoot(s.tree)
	if err != nil {
		return []string{treeDirName}
	}
	defer root.Close()
	var changed []string
	for _, e := range differing(root, s.entries) {
		changed = append(changed, e.path)
	}
	return changed
}

// openNoFollow opens a note beside the tree, never through a link in its place.
func openNoFollow(path string, flag int) (*os.File, error) {
	return os.OpenFile(path, flag|syscall.O_NOFOLLOW, 0o600)
}
