package testreuse

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// gitRepo is a small repository for the sync's tests.
type gitRepo struct {
	t    *testing.T
	root string
}

func newGitRepo(t *testing.T) *gitRepo {
	t.Helper()
	r := &gitRepo{t: t, root: t.TempDir()}
	r.git("init", "-q", "-b", "main")
	return r
}

func (r *gitRepo) git(args ...string) string {
	r.t.Helper()
	return runGit(r.t, r.root, args...)
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=Quinn", "-c", "user.email=quinn@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = gittest.Env(t)
	for _, name := range []string{"GIT_AUTHOR_DATE", "GIT_COMMITTER_DATE"} {
		if at := os.Getenv(name); at != "" {
			cmd.Env = append(cmd.Env, name+"="+at)
		}
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *gitRepo) write(rel, body string) {
	r.t.Helper()
	writeFile(r.t, filepath.Join(r.root, rel), body, 0o644)
}

func writeFile(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func (r *gitRepo) commit(msg string) string {
	r.t.Helper()
	r.git("add", "-A")
	r.git("commit", "-q", "-m", msg)
	return r.git("rev-parse", "HEAD")
}

// TestAFileTimeIsInThePastAndFollowsItsContent: every derived time is a whole
// second before 2021, where Go will cache a test that opened the file and APFS
// and ext4 store it exactly; the same content has the same time everywhere and
// different content a different one.
func TestAFileTimeIsInThePastAndFollowsItsContent(t *testing.T) {
	limit := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	seen := map[int64]bool{}
	fractional := 0
	for i := range 2000 {
		at := contentTime(sha256.Sum256([]byte{byte(i), byte(i >> 8)}))
		if at.Before(timeBase) || !at.Before(limit) {
			t.Fatalf("content %d takes %s, outside the window", i, at)
		}
		if at.Nanosecond() != 0 {
			fractional++
		}
		seen[at.UnixNano()] = true
	}
	if len(seen) != 2000 {
		t.Errorf("2000 contents took %d times; the window should spread them", len(seen))
	}
	if fractional < 1990 {
		t.Errorf("%d of 2000 times carry nanosecond bits; a whole second is too few bits to stand on", fractional)
	}
	if a, b := contentTime(sha256.Sum256([]byte("same"))), contentTime(sha256.Sum256([]byte("same"))); !a.Equal(b) {
		t.Errorf("one content took two times: %s and %s", a, b)
	}
}

// TestSyncMakesTheTreeTheCommit: the tree holds the commit's files with their
// modes and derived times, directories with derived times, and no .git. A second
// check directory comes out identical, stat for stat, which is what lets a result
// carry between them on one path.
func TestSyncMakesTheTreeTheCommit(t *testing.T) {
	r := newGitRepo(t)
	r.write("go.mod", "module m\n")
	r.write("a/b/deep.txt", "deep\n")
	writeFile(t, filepath.Join(r.root, "run.sh"), "#!/bin/sh\n", 0o755)
	head := r.commit("one")

	first, second := t.TempDir(), t.TempDir()
	for _, dir := range []string{first, second} {
		if _, err := Sync(context.Background(), r.root, head, dir); err != nil {
			t.Fatal(err)
		}
	}
	tree := filepath.Join(first, treeDirName)
	if raw, err := os.ReadFile(filepath.Join(tree, "a", "b", "deep.txt")); err != nil || string(raw) != "deep\n" {
		t.Fatalf("deep.txt: %q, %v", raw, err)
	}
	info, err := os.Stat(filepath.Join(tree, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("run.sh has mode %o, want the commit's 755", info.Mode().Perm())
	}
	if want := contentTime(sha256.Sum256([]byte("#!/bin/sh\n"))); !info.ModTime().Equal(want) {
		t.Errorf("run.sh's time is %s, want %s", info.ModTime(), want)
	}
	if _, err := os.Lstat(filepath.Join(tree, ".git")); !os.IsNotExist(err) {
		t.Errorf("the tree holds a .git (%v): a test could ask git about the repository there", err)
	}
	for _, rel := range []string{".", "a", "a/b", "go.mod", "a/b/deep.txt", "run.sh"} {
		one, err := os.Stat(filepath.Join(first, treeDirName, rel))
		if err != nil {
			t.Fatal(err)
		}
		two, err := os.Stat(filepath.Join(second, treeDirName, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !one.ModTime().Equal(two.ModTime()) || one.Mode() != two.Mode() || (!one.IsDir() && one.Size() != two.Size()) {
			t.Errorf("%s differs between two syncs of one commit: %s %v, %s %v", rel, one.ModTime(), one.Mode(), two.ModTime(), two.Mode())
		}
		if !one.ModTime().Before(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("%s has the time %s, not a derived one", rel, one.ModTime())
		}
	}
}

// TestASyncWritesOnlyWhatChanged: this repository's tree has thousands of files
// and something on this machine watches file events, so a commit changing one
// file writes one file and leaves every other's inode alone.
func TestASyncWritesOnlyWhatChanged(t *testing.T) {
	r := newGitRepo(t)
	for i := range 50 {
		r.write("d/f"+string(rune('a'+i%26))+string(rune('a'+i/26))+".txt", "body\n")
	}
	r.write("keep.txt", "keep\n")
	r.write("edit.txt", "before\n")
	first := r.commit("one")
	dir := t.TempDir()
	stats, err := Sync(context.Background(), r.root, first, dir)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Written != 52 {
		t.Fatalf("the first sync wrote %d files, want 52", stats.Written)
	}
	again, err := Sync(context.Background(), r.root, first, dir)
	if err != nil {
		t.Fatal(err)
	}
	if again.Written != 0 || again.Removed != 0 || again.Retimed != 0 {
		t.Errorf("a sync of the same commit changed something: %+v", again)
	}

	keep := filepath.Join(dir, treeDirName, "keep.txt")
	before, err := os.Stat(keep)
	if err != nil {
		t.Fatal(err)
	}
	r.write("edit.txt", "after!!\n")
	r.write("new.txt", "new\n")
	r.git("rm", "-q", "--cached", "d/faa.txt")
	if err := os.Remove(filepath.Join(r.root, "d", "faa.txt")); err != nil {
		t.Fatal(err)
	}
	second := r.commit("two")
	stats, err = Sync(context.Background(), r.root, second, dir)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Written != 2 || stats.Removed != 1 {
		t.Errorf("a commit changing two files and removing one wrote %d and removed %d", stats.Written, stats.Removed)
	}
	after, err := os.Stat(keep)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Error("an unchanged file was rewritten")
	}
	if _, err := os.Lstat(filepath.Join(dir, treeDirName, "d", "faa.txt")); !os.IsNotExist(err) {
		t.Errorf("a file the commit removed is still in the tree: %v", err)
	}
}

// TestASameSizeEditMovesTheFilesTime: Go keys an opened file on its size and
// time, so a same-size edit committed within one second, which a commit time
// would miss, must move the file's time.
func TestASameSizeEditMovesTheFilesTime(t *testing.T) {
	r := newGitRepo(t)
	r.write("data.txt", "good\n")
	first := r.commit("one")
	dir := t.TempDir()
	if _, err := Sync(context.Background(), r.root, first, dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, treeDirName, "data.txt")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	r.write("data.txt", "evil\n")
	second := r.commit("two")
	if _, err := Sync(context.Background(), r.root, second, dir); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if before.Size() != after.Size() {
		t.Fatalf("the edit changed the size: %d to %d", before.Size(), after.Size())
	}
	if before.ModTime().Equal(after.ModTime()) {
		t.Errorf("a same-size edit kept the time %s, so go test would replay the old result", after.ModTime())
	}
	if raw, _ := os.ReadFile(path); string(raw) != "evil\n" {
		t.Errorf("the tree holds %q", raw)
	}
}

// TestASyncRecoversADirectoryACrashLeft: a sync or a test killed anywhere
// leaves a directory the next sync makes exact, whatever state it is in.
func TestASyncRecoversADirectoryACrashLeft(t *testing.T) {
	r := newGitRepo(t)
	r.write("go.mod", "module m\n")
	r.write("a/file.txt", "whole file\n")
	r.write("b/file.txt", "other file\n")
	r.write("c/file.txt", "third\n")
	head := r.commit("one")
	dir := t.TempDir()
	if _, err := Sync(context.Background(), r.root, head, dir); err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(dir, treeDirName)

	// A cut-short write, a temporary file, a memo cut short, what a test left,
	// a file where a directory goes and a directory where a file goes, a
	// directory a test made unreadable, and a directory's time moved.
	writeFile(t, filepath.Join(tree, "a", "file.txt"), "whole", 0o644)
	writeFile(t, filepath.Join(dir, ".abcd-tmp-12345"), "partial", 0o600)
	writeFile(t, filepath.Join(dir, "unknown.json"), "{\"cut", 0o600)
	writeFile(t, filepath.Join(tree, "a", "left-behind.txt"), "x", 0o644)
	writeFile(t, filepath.Join(tree, "stray", "deep", "x.txt"), "x", 0o644)
	if err := os.RemoveAll(filepath.Join(tree, "b")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(tree, "b"), "not a directory", 0o644)
	if err := os.Remove(filepath.Join(tree, "go.mod")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(tree, "go.mod", "x"), "a directory", 0o644)
	if err := os.Chmod(filepath.Join(tree, "stray", "deep"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(tree, "stray", "deep"), 0o755) })
	if err := os.Chmod(filepath.Join(tree, "c"), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(tree, "c"), 0o755) })

	if _, err := Sync(context.Background(), r.root, head, dir); err != nil {
		t.Fatalf("the sync did not recover: %v", err)
	}
	clean := t.TempDir()
	if _, err := Sync(context.Background(), r.root, head, clean); err != nil {
		t.Fatal(err)
	}
	var want []string
	if err := filepath.Walk(filepath.Join(clean, treeDirName), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(filepath.Join(clean, treeDirName), p)
		want = append(want, rel)
		got, err := os.Lstat(filepath.Join(tree, rel))
		if err != nil {
			t.Errorf("%s is missing after recovery: %v", rel, err)
			return nil
		}
		if !got.ModTime().Equal(info.ModTime()) || got.Mode() != info.Mode() || (!info.IsDir() && got.Size() != info.Size()) {
			t.Errorf("%s differs after recovery: %s %v %d, want %s %v %d", rel, got.ModTime(), got.Mode(), got.Size(), info.ModTime(), info.Mode(), info.Size())
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := filepath.Walk(tree, func(p string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(tree, p)
		got = append(got, rel)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the recovered tree holds\n%v\nwant\n%v", got, want)
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.Name() != treeDirName {
			t.Errorf("%s remains beside the tree", it.Name())
		}
	}
}

// TestASnapshotNamesATrackedFileARunRewrote: the tree a stage ran over must
// still be the commit when it ends.
func TestASnapshotNamesATrackedFileARunRewrote(t *testing.T) {
	r := newGitRepo(t)
	r.write("a.txt", "a\n")
	r.write("b.txt", "b\n")
	head := r.commit("one")
	dir := t.TempDir()
	_, snap, err := syncTree(context.Background(), r.root, head, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.changedTracked(); len(got) != 0 {
		t.Fatalf("a fresh tree reads as changed: %v", got)
	}
	writeFile(t, filepath.Join(dir, treeDirName, "b.txt"), "B\n", 0o644)
	if got := snap.changedTracked(); len(got) != 1 || got[0] != "b.txt" {
		t.Errorf("a rewritten file reads as %v", got)
	}
}

// TestAPathOutOfTheTreeIsRefused: what the sync writes, the tests execute, so
// nothing outside the tree and nothing in a .git.
func TestAPathOutOfTheTreeIsRefused(t *testing.T) {
	for _, p := range []string{"../escape", "a/../../b", "/abs", ".git/hooks/x", "a/.git/config", ".GIT/hooks/x", "a/.Git/config"} {
		if treePath(p) {
			t.Errorf("%q was taken", p)
		}
	}
	for _, p := range []string{"a", "a/b.txt", ".github/x", ".gitignore"} {
		if !treePath(p) {
			t.Errorf("%q was refused", p)
		}
	}
}

// TestASameSizeFileWithAMatchingTimeIsStillRewritten: a commit's file that shares
// its size and derived time with what is on disk is not taken for it. The sync
// proves the bytes, so a tree it keeps is the commit's.
func TestASameSizeFileWithAMatchingTimeIsStillRewritten(t *testing.T) {
	r := newGitRepo(t)
	r.write("data.txt", "evil\n")
	first := r.commit("one")
	dir := t.TempDir()
	if _, err := Sync(context.Background(), r.root, first, dir); err != nil {
		t.Fatal(err)
	}
	r.write("data.txt", "good\n")
	second := r.commit("two")
	onDisk := filepath.Join(dir, treeDirName, "data.txt")
	at := contentTime(sha256.Sum256([]byte("good\n")))
	if err := os.Chtimes(onDisk, at, at); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(context.Background(), r.root, second, dir); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(onDisk); string(raw) != "good\n" {
		t.Errorf("the tree holds %q under a time that says it is the commit's", raw)
	}

	// The same with both commits' blobs already seen: back to the first commit,
	// with the file made to look like it.
	writeFile(t, onDisk, "good\n", 0o644)
	at = contentTime(sha256.Sum256([]byte("evil\n")))
	if err := os.Chtimes(onDisk, at, at); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(context.Background(), r.root, first, dir); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(onDisk); string(raw) != "evil\n" {
		t.Errorf("the tree holds %q after a sync to the first commit", raw)
	}
}

// TestALinkToAnotherTargetOfTheSameLengthIsRetargeted: a link is the commit's
// when its target is, whatever the length says.
func TestALinkToAnotherTargetOfTheSameLengthIsRetargeted(t *testing.T) {
	r := newGitRepo(t)
	r.write("a.txt", "a\n")
	r.write("b.txt", "b\n")
	if err := os.Symlink("a.txt", filepath.Join(r.root, "link")); err != nil {
		t.Fatal(err)
	}
	first := r.commit("one")
	dir := t.TempDir()
	if _, err := Sync(context.Background(), r.root, first, dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(r.root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("b.txt", filepath.Join(r.root, "link")); err != nil {
		t.Fatal(err)
	}
	second := r.commit("two")
	if _, err := Sync(context.Background(), r.root, second, dir); err != nil {
		t.Fatal(err)
	}
	if got, err := os.Readlink(filepath.Join(dir, treeDirName, "link")); err != nil || got != "b.txt" {
		t.Errorf("the link points at %q (%v), want b.txt", got, err)
	}
}

// TestASnapshotNamesAFileThatOnlyLooksTheSame: a run that rewrites a tracked file
// and puts its size and time back has still left the tree not the commit.
func TestASnapshotNamesAFileThatOnlyLooksTheSame(t *testing.T) {
	r := newGitRepo(t)
	r.write("a.txt", "a\n")
	head := r.commit("one")
	dir := t.TempDir()
	_, snap, err := syncTree(context.Background(), r.root, head, dir)
	if err != nil {
		t.Fatal(err)
	}
	onDisk := filepath.Join(dir, treeDirName, "a.txt")
	info, err := os.Stat(onDisk)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, onDisk, "b\n", 0o644)
	if err := os.Chtimes(onDisk, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if got := snap.changedTracked(); len(got) != 1 || got[0] != "a.txt" {
		t.Errorf("a forged file reads as %v", got)
	}
}

// commitFromIndex commits exactly the given index entries, which git would not
// let a working tree spell.
func commitFromIndex(t *testing.T, r *gitRepo, entries ...string) string {
	t.Helper()
	blob := func(body string) string {
		cmd := exec.Command("git", "hash-object", "-w", "--stdin")
		cmd.Dir = r.root
		cmd.Env = gittest.Env(t)
		cmd.Stdin = strings.NewReader(body)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(out))
	}
	for _, e := range entries {
		mode, rest, _ := strings.Cut(e, " ")
		path, body, _ := strings.Cut(rest, " ")
		r.git("-c", "core.ignorecase=false", "update-index", "--add", "--cacheinfo", mode+","+blob(body)+","+path)
	}
	tree := r.git("-c", "core.ignorecase=false", "write-tree")
	return r.git("commit-tree", tree, "-m", "by hand")
}

// TestACommitThatCouldWriteThroughALinkIsRefused: two paths one file system takes
// for one, the first a link, would land the second wherever the link points. The
// sync refuses the commit and writes nothing.
func TestACommitThatCouldWriteThroughALinkIsRefused(t *testing.T) {
	outside := t.TempDir()
	r := newGitRepo(t)
	for name, entries := range map[string][]string{
		"a link and a path under its other spelling": {"120000 Link " + outside, "100644 link/evil.txt pwned"},
		"one file under two spellings":               {"100644 File.txt one", "100644 file.txt two"},
		"one directory under two spellings":          {"100644 Dir/a.txt one", "100644 dir/b.txt two"},
	} {
		t.Run(name, func(t *testing.T) {
			r.git("read-tree", "--empty")
			commit := commitFromIndex(t, r, entries...)
			dir := t.TempDir()
			if _, err := Sync(context.Background(), r.root, commit, dir); err == nil {
				t.Error("the commit was taken")
			}
			if items, _ := os.ReadDir(outside); len(items) != 0 {
				t.Errorf("the sync wrote outside the tree: %v", items)
			}
		})
	}
}

// TestASyncDoesNotWriteThroughALinkARunLeft: a directory of the commit that a test
// replaced with a link to elsewhere is put back, and elsewhere is untouched.
func TestASyncDoesNotWriteThroughALinkARunLeft(t *testing.T) {
	outside := t.TempDir()
	r := newGitRepo(t)
	r.write("a/file.txt", "commit\n")
	first := r.commit("one")
	dir := t.TempDir()
	if _, err := Sync(context.Background(), r.root, first, dir); err != nil {
		t.Fatal(err)
	}
	inTree := filepath.Join(dir, treeDirName, "a")
	if err := os.RemoveAll(inTree); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, inTree); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(context.Background(), r.root, first, dir); err != nil {
		t.Fatal(err)
	}
	if items, _ := os.ReadDir(outside); len(items) != 0 {
		t.Errorf("the sync wrote through the link: %v", items)
	}
	info, err := os.Lstat(inTree)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Errorf("the directory is still %v", info.Mode())
	}
}

// TestATreeThatIsALinkIsRefused: what a test plants in the tree's own place must
// not take the sync's writes elsewhere.
func TestATreeThatIsALinkIsRefused(t *testing.T) {
	outside := t.TempDir()
	r := newGitRepo(t)
	r.write("a.txt", "a\n")
	head := r.commit("one")
	dir := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, treeDirName)); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(context.Background(), r.root, head, dir); err == nil {
		t.Error("a linked tree was taken")
	}
	if items, _ := os.ReadDir(outside); len(items) != 0 {
		t.Errorf("the sync wrote through the link: %v", items)
	}
	if _, err := Sync(context.Background(), r.root, head, dir); err != nil {
		t.Errorf("the next check does not recover: %v", err)
	}
}

// TestASyncRemovesWhatItDoesNotKnowBesideTheTree: go looks above the module for a
// go.work, and the directory is the tree's parent.
func TestASyncRemovesWhatItDoesNotKnowBesideTheTree(t *testing.T) {
	r := newGitRepo(t)
	r.write("a.txt", "a\n")
	head := r.commit("one")
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.work"), "go 1.21\n", 0o644)
	writeFile(t, filepath.Join(dir, "nested", "x"), "x", 0o644)
	writeFile(t, filepath.Join(dir, holderName), "kept", 0o600)
	writeFile(t, filepath.Join(dir, lockName), "", 0o600)
	if _, err := Sync(context.Background(), r.root, head, dir); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"go.work", "nested"} {
		if _, err := os.Lstat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
			t.Errorf("%s remains: %v", gone, err)
		}
	}
	for _, kept := range []string{holderName, lockName} {
		if _, err := os.Lstat(filepath.Join(dir, kept)); err != nil {
			t.Errorf("%s was removed: %v", kept, err)
		}
	}
}
