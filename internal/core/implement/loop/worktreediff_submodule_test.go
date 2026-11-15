package loop

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// subFilterRepo is a repository with a checked-out submodule, sub, whose own
// settings name a clean filter for every path, a script that records each
// run at mark, and whose .gitmodules entry sets ignore to none, so every git
// command that honours that entry looks inside it. stale gives the
// submodule's file a stale stat, so a status inside the submodule re-hashes
// it through that filter; live reports whether the filter ran since the last
// stale, and clears the record.
func subFilterRepo(t *testing.T) (r *gittest.Repo, stale func(), live func() bool) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	t.Setenv("HOME", t.TempDir())
	r = gittest.NewRepo(t)
	r.Git("config", "user.name", "Fixture")
	r.Git("config", "user.email", "fixture@example.invalid")
	r.Git("config", "commit.gpgsign", "false")
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	gitDir := r.AddSubmodule("sub")
	r.Git("config", "--file", ".gitmodules", "submodule.sub.ignore", "none")
	r.Commit("the submodule's entry looks inside it")
	mark := filepath.Join(t.TempDir(), "sub-filter-ran")
	script := filepath.Join(t.TempDir(), "sub-clean.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "--file", filepath.Join(gitDir, "config"), "filter.subown.clean", script)
	if err := os.MkdirAll(filepath.Join(gitDir, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "info", "attributes"), []byte("* filter=subown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	n := 0
	stale = func() {
		n++
		later := time.Now().Add(time.Duration(n) * time.Hour)
		if err := os.Chtimes(filepath.Join(r.Root(), "sub", "s.txt"), later, later); err != nil {
			t.Fatal(err)
		}
	}
	live = func() bool {
		_, err := os.Stat(mark)
		_ = os.Remove(mark)
		return err == nil
	}
	stale()
	plain := exec.Command("git", "-C", r.Root(), "status", "--porcelain")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git status: %v\n%s", err, out)
	}
	if !live() {
		t.Fatal("fixture: a plain git status did not run the submodule's clean filter, so this test proves nothing")
	}
	return r, stale, live
}

// moveSub checks a new commit out inside the submodule, which moves its
// pointer, with the submodule's own filter blanked for the commit.
func moveSub(t *testing.T, r *gittest.Repo) {
	t.Helper()
	r.Git("-C", "sub", "-c", "filter.subown.clean=", "commit", "-q", "--allow-empty", "-m", "moved")
}

// TestTheDrivesAddStartsNothingInASubmodule: git's add compares the index
// with the working tree and, for a checked-out submodule whose commit is the
// one the index records, starts a status inside it under its own settings;
// git add takes no --ignore-submodules and no setting stops it, as the plain
// add here shows. The drive's add (addAllGit, and addAllGitIndex on a scratch
// index, as a restart's snapshot and the autoplan digest stage the tree)
// leaves such a submodule out by pathspec, so the filter its own settings
// name does not run, and still stages a moved submodule and every other
// change. Watched failing with addAllArgs giving the plain `add -A`: each
// add ran the submodule's filter.
func TestTheDrivesAddStartsNothingInASubmodule(t *testing.T) {
	r, stale, live := subFilterRepo(t)
	plain := exec.Command("git", "-C", r.Root(), "-c", "diff.ignoreSubmodules=dirty", "-c", "submodule.sub.ignore=dirty", "add", "-A")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git add: %v\n%s", err, out)
	}
	if !live() {
		t.Fatal("fixture: a plain git add -A, with both settings, did not run the submodule's clean filter, so this test proves nothing")
	}

	stale()
	r.Write("b.txt", "b\n")
	if _, err := addAllGit(r.Root(), "."); err != nil {
		t.Fatal(err)
	}
	if live() {
		t.Error("the drive's add ran the submodule's own clean filter")
	}
	if got := r.Git("diff", "--cached", "--name-only"); got != "b.txt" {
		t.Errorf("the drive's add stages the change beside the submodule: %q", got)
	}
	r.Git("reset", "-q")

	head := r.Git("rev-parse", "HEAD")
	for _, moved := range []bool{false, true} {
		if moved {
			moveSub(t, r)
		}
		stale()
		index := filepath.Join(t.TempDir(), "index")
		if _, err := gitutil.RunWithIndex(r.Root(), index, 4096, "read-tree", head); err != nil {
			t.Fatal(err)
		}
		if _, err := addAllGitIndex(r.Root(), index); err != nil {
			t.Fatal(err)
		}
		if live() {
			t.Errorf("submodule moved %v: the drive's add on a scratch index ran the submodule's own clean filter", moved)
		}
		names, err := gitutil.RunWithIndex(r.Root(), index, 4096, "diff", "--cached", "--name-only", head, "--")
		if err != nil {
			t.Fatal(err)
		}
		want := "b.txt\n"
		if moved {
			want = "b.txt\nsub\n"
		}
		if string(names) != want {
			t.Errorf("submodule moved %v: the scratch index stages %q, want %q", moved, names, want)
		}
	}
}

// TestTheDrivesAddReadsItsPathspecsAsAbcdWritesThem: GIT_LITERAL_PATHSPECS in
// the environment makes git read every pathspec as a file name, which would
// turn the staging's excludes into names of files that do not exist, so the
// checked-out submodule would be staged through its own filter. The drive's
// add passes --no-literal-pathspecs, so the excludes hold with it set.
// Watched failing with the add passing no such option: the filter ran.
func TestTheDrivesAddReadsItsPathspecsAsAbcdWritesThem(t *testing.T) {
	r, stale, live := subFilterRepo(t)
	t.Setenv("GIT_LITERAL_PATHSPECS", "1")
	stale()
	r.Write("b.txt", "b\n")
	if _, err := addAllGit(r.Root(), "."); err != nil {
		t.Fatal(err)
	}
	if live() {
		t.Error("with GIT_LITERAL_PATHSPECS set, the drive's add ran the submodule's own clean filter")
	}
	if got := r.Git("diff", "--cached", "--name-only"); got != "b.txt" {
		t.Errorf("the drive's add stages the change beside the submodule: %q", got)
	}
}

// TestTheAddsProgramsSeeLiteralPathspecsOffAndNoICase: git given
// --no-literal-pathspecs sets GIT_LITERAL_PATHSPECS=0 for every program it
// starts, as a post-index-change hook of git's own add on the drive's add's
// command line records with the variable set to 1. The drive's add runs no
// hook (core.hooksPath is pinned to /dev/null) and starts a content filter,
// which sees GIT_LITERAL_PATHSPECS=0 and no GIT_ICASE_PATHSPECS (addCommand).
// Watched failing with addAllArgs passing no --no-literal-pathspecs (the hook
// and the filter saw 1) and with addCommand keeping GIT_ICASE_PATHSPECS (the
// filter saw it).
func TestTheAddsProgramsSeeLiteralPathspecsOffAndNoICase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Git("config", "commit.gpgsign", "false")
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	dir := t.TempDir()
	hookMark, filterMark := filepath.Join(dir, "hook"), filepath.Join(dir, "filter")
	hooks := filepath.Join(r.Root(), ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\nprintf %s \"$GIT_LITERAL_PATHSPECS\" > " + hookMark + "\n"
	if err := os.WriteFile(filepath.Join(hooks, "post-index-change"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	filter := filepath.Join(dir, "clean.sh")
	script := "#!/bin/sh\nprintf '%s %s' \"$GIT_LITERAL_PATHSPECS\" \"${GIT_ICASE_PATHSPECS-unset}\" > " + filterMark + "\ncat\n"
	if err := os.WriteFile(filter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "filter.rec.clean", filter)
	if err := os.MkdirAll(filepath.Join(r.Root(), ".git", "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "attributes"), []byte("c.txt filter=rec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if errors.Is(err, os.ErrNotExist) {
			return "(did not run)"
		}
		if err != nil {
			t.Fatal(err)
		}
		_ = os.Remove(p)
		return string(b)
	}

	r.Write("b.txt", "b\n")
	args, stdin, err := addAllArgs(r.Root(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	own := exec.Command("git", append([]string{"-C", r.Root()}, args...)...)
	own.Env = append(r.Env(), "GIT_LITERAL_PATHSPECS=1")
	own.Stdin = strings.NewReader(stdin)
	if out, err := own.CombinedOutput(); err != nil {
		t.Fatalf("git's own add on the drive's add's command line: %v\n%s", err, out)
	}
	if got := read(hookMark); got != "0" {
		t.Errorf("a hook of git's own add on the drive's add's command line saw GIT_LITERAL_PATHSPECS=%s, want 0", got)
	}
	r.Git("reset", "-q")
	_ = read(hookMark)

	t.Setenv("GIT_LITERAL_PATHSPECS", "1")
	t.Setenv("GIT_ICASE_PATHSPECS", "1")
	r.Write("c.txt", "c\n")
	if _, err := addAllGit(r.Root(), "."); err != nil {
		t.Fatal(err)
	}
	if got := read(filterMark); got != "0 unset" {
		t.Errorf("the drive's add's content filter saw GIT_LITERAL_PATHSPECS and GIT_ICASE_PATHSPECS as %q, want \"0 unset\"", got)
	}
	if got := read(hookMark); got != "(did not run)" {
		t.Errorf("the drive's add ran the repository's post-index-change hook (it saw %q)", got)
	}
}

// TestTheDrivesAddExcludesTheSubmodulesOwnPathAlone: with
// GIT_ICASE_PATHSPECS set, git matches every pathspec without regard to case,
// so the exclude of an unmoved submodule at lib also leaves out LIB/f.txt.
// The drive's add runs without the variable (addCommand), so the exclude
// matches lib alone, as git's own answer shows: ls-files given the add's
// pathspecs, under the add's command and environment, on its own index and
// on a scratch one, lists LIB/f.txt. The index is checked with
// core.ignoreCase off, so no filesystem's folding of case decides it.
// Watched failing with addCommand keeping GIT_ICASE_PATHSPECS: LIB/f.txt was
// left out.
func TestTheDrivesAddExcludesTheSubmodulesOwnPathAlone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Git("config", "commit.gpgsign", "false")
	r.Git("config", "core.ignoreCase", "false")
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	r.AddSubmodule("lib")
	blob := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(blob, []byte("f\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oid := r.Git("hash-object", "-w", blob)
	r.Git("update-index", "--add", "--cacheinfo", "100644,"+oid+",LIB/f.txt")
	scratch := filepath.Join(t.TempDir(), "index")
	own, err := os.ReadFile(filepath.Join(r.Root(), ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scratch, own, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_ICASE_PATHSPECS", "1")

	for _, index := range []string{"", scratch} {
		args, stdin, err := addAllArgs(r.Root(), index, nil)
		if err != nil {
			t.Fatal(err)
		}
		specs := strings.Split(strings.TrimSuffix(stdin, "\x00"), "\x00")
		if !slices.Contains(specs, ":(top,exclude,literal)lib") {
			t.Fatalf("fixture: the add does not exclude the unmoved submodule (%q), so this test proves nothing", specs)
		}
		ls := append([]string{args[0], "ls-files", "-z", "--"}, specs...)
		plain := exec.Command("git", append([]string{"-C", r.Root()}, ls...)...)
		plain.Env = append(r.Env(), "GIT_ICASE_PATHSPECS=1")
		if index != "" {
			plain.Env = append(plain.Env, "GIT_INDEX_FILE="+index)
		}
		kept, err := gitutil.RunTreeCommand(plain, ls)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(kept, "LIB/f.txt\x00") {
			t.Fatalf("fixture: with GIT_ICASE_PATHSPECS set, git's ls-files kept LIB/f.txt (%q), so this test proves nothing", kept)
		}
		cmd, err := addCommand(r.Root(), index, "", ls)
		if err != nil {
			t.Fatal(err)
		}
		got, err := gitutil.RunTreeCommand(cmd, ls)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "LIB/f.txt\x00") || strings.Contains(got, "\x00lib\x00") || strings.HasPrefix(got, "lib\x00") {
			t.Errorf("index %q: the drive's add's pathspecs leave out LIB/f.txt, or keep lib: %q", index, got)
		}
	}
}

// TestTheDrivesAddIsBoundedByNoCountOfSubmodules: an index holding more
// gitlinks than any bound a listing once had (4097, none checked out) is
// staged, its excludes passed to git on its standard input rather than its
// command line, and every gitlink is kept. Watched failing on the branch
// before: the listing refused the index.
func TestTheDrivesAddIsBoundedByNoCountOfSubmodules(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Git("config", "commit.gpgsign", "false")
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	head := r.Git("rev-parse", "HEAD")
	const n = 4097
	var info strings.Builder
	for i := range n {
		p := fmt.Sprintf("links/%04d-%s", i, strings.Repeat("x", 200))
		fmt.Fprintf(&info, "160000 %s 0\t%s\n", head, p)
		if err := os.MkdirAll(filepath.Join(r.Root(), filepath.FromSlash(p)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("git", "-C", r.Root(), "update-index", "--index-info")
	cmd.Env, cmd.Stdin = r.Env(), strings.NewReader(info.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git update-index: %v\n%s", err, out)
	}
	r.Git("commit", "-q", "-m", "many gitlinks")
	r.Write("b.txt", "b\n")
	if _, err := addAllGit(r.Root(), "."); err != nil {
		t.Fatalf("the drive's add over %d gitlinks: %v", n, err)
	}
	if got := r.Git("diff", "--cached", "--name-only"); got != "b.txt" {
		t.Errorf("the drive's add stages the change and keeps every gitlink: %q", got)
	}
}

// embeddedFilterRepo is subFilterRepo's shape for a repository embedded in
// the tree and added by path, which records a gitlink that no .gitmodules
// entry names: git has no submodule.<name>.ignore to read for it, and looks
// inside it by default. Its own settings name a clean filter for every path;
// stale and live are subFilterRepo's.
func embeddedFilterRepo(t *testing.T) (r *gittest.Repo, stale func(), live func() bool) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	t.Setenv("HOME", t.TempDir())
	r = gittest.NewRepo(t)
	r.Git("config", "user.name", "Fixture")
	r.Git("config", "user.email", "fixture@example.invalid")
	r.Git("config", "commit.gpgsign", "false")
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	emb := filepath.Join(r.Root(), "emb")
	r.Git("init", "-q", emb)
	r.Write("emb/s.txt", "s\n")
	r.Git("-C", emb, "add", "s.txt")
	r.Git("-C", emb, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-q", "-m", "embedded seed")
	r.Git("-c", "advice.addEmbeddedRepo=false", "add", "emb")
	r.Git("commit", "-q", "-m", "an embedded repository, no .gitmodules entry")
	mark := filepath.Join(t.TempDir(), "emb-filter-ran")
	script := filepath.Join(t.TempDir(), "emb-clean.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("-C", emb, "config", "filter.embown.clean", script)
	if err := os.WriteFile(filepath.Join(emb, ".git", "info", "attributes"), []byte("* filter=embown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	n := 0
	stale = func() {
		n++
		later := time.Now().Add(time.Duration(n) * time.Hour)
		if err := os.Chtimes(filepath.Join(emb, "s.txt"), later, later); err != nil {
			t.Fatal(err)
		}
	}
	live = func() bool {
		_, err := os.Stat(mark)
		_ = os.Remove(mark)
		return err == nil
	}
	return r, stale, live
}

// sideBranches makes a branch, side-<file>, for each file, holding that file
// on top of HEAD, with every filter blanked for the fixture's own commits.
func sideBranches(t *testing.T, r *gittest.Repo, files ...string) {
	t.Helper()
	for _, file := range files {
		r.Git("checkout", "-q", "-b", "side-"+file)
		r.Write(file, file+"\n")
		r.Git("-c", "filter.subown.clean=", "-c", "filter.embown.clean=", "add", file)
		r.Git("-c", "filter.subown.clean=", "-c", "filter.embown.clean=", "commit", "-q", "-m", file)
		r.Git("checkout", "-q", "-")
	}
}

// TestASyncMergeIsRefusedOverACheckedOutSubmodule: git looks inside a
// checked-out submodule, under that submodule's own settings, as its merge
// saves the lane's changes first where the .gitmodules entry sets its
// ignore to none (the plain merge here), and by default for a repository
// embedded in the tree, which no entry names (the plain status here; git's
// merge looks inside it too once the changes it saves hold a tracked file's).
// No command-line form reaches what the merge runs and not the merge's
// hooks, so the sync's merge (syncMerge) is not made where the lane's index
// records a gitlink with a repository checked out at its path, whether or
// not .gitmodules names it: it is refused, naming the path, and nothing is
// merged or run. A gitlink with nothing checked out merges as before.
// Watched failing with the -c pins removed and nothing in their place: both
// merges were made, the named submodule's running its own filter.
func TestASyncMergeIsRefusedOverACheckedOutSubmodule(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		repo       func(t *testing.T) (*gittest.Repo, func(), func() bool)
		looks      []string
	}{
		{"a submodule .gitmodules names", "sub", subFilterRepo,
			[]string{"-c", "diff.ignoreSubmodules=dirty", "merge", "--no-ff", "--no-edit", "-m", "plain", "side-plain.txt"}},
		{"a repository embedded with no entry", "emb", embeddedFilterRepo, []string{"status", "--porcelain"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, stale, live := tc.repo(t)
			sideBranches(t, r, "plain.txt", "sync.txt")
			_ = live()
			stale()
			plain := exec.Command("git", append([]string{"-C", r.Root()}, tc.looks...)...)
			plain.Env = r.Env()
			if out, err := plain.CombinedOutput(); err != nil {
				t.Fatalf("fixture: git %s: %v\n%s", tc.looks[0], err, out)
			}
			if !live() {
				t.Fatalf("fixture: a plain git %q did not run the submodule's clean filter, so this test proves nothing", tc.looks)
			}

			head := r.Git("rev-parse", "HEAD")
			stale()
			err := syncMerge(r.Root(), "sync", "side-sync.txt")
			var checkedOut *checkedOutSubmodule
			if !errors.As(err, &checkedOut) || checkedOut.Path != tc.path || !strings.Contains(err.Error(), `"`+tc.path+`"`) {
				t.Fatalf("the sync's merge over a checked-out submodule is refused, naming %s: %v", tc.path, err)
			}
			if live() {
				t.Error("the refused sync ran the submodule's own clean filter")
			}
			if got := r.Git("rev-parse", "HEAD"); got != head {
				t.Errorf("the refused sync merged: HEAD moved from %s to %s", head, got)
			}
		})
	}
	t.Run("a gitlink with nothing checked out", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		r := gittest.NewRepo(t)
		r.Git("config", "commit.gpgsign", "false")
		r.Write("a.txt", "a\n")
		r.Commit("seed")
		r.Git("update-index", "--add", "--cacheinfo", "160000,"+r.Git("rev-parse", "HEAD")+",unpop")
		r.Git("commit", "-q", "-m", "a gitlink with no checkout")
		if err := os.Mkdir(filepath.Join(r.Root(), "unpop"), 0o755); err != nil {
			t.Fatal(err)
		}
		sideBranches(t, r, "sync.txt")
		if err := syncMerge(r.Root(), "sync", "side-sync.txt"); err != nil {
			t.Fatal(err)
		}
		if got := r.Git("log", "-1", "--format=%s%n%P"); !strings.HasPrefix(got, "sync\n") || len(strings.Fields(got)) != 3 {
			t.Errorf("the sync's merge makes its merge commit: %q", got)
		}
	})
}

// TestAFailedSyncsUnmergedReadStartsNothingInASubmodule: after a sync's
// merge stops on a conflict, the drive reads the unmerged paths with a diff
// of the index with the working tree (unmergedPaths, through
// gitutil.DiffWorktreeArgs), which starts nothing inside the submodule, and
// still lists the conflicted file. A plain diff of the same shape runs the
// submodule's filter. Watched failing with the read's plain `git diff`.
func TestAFailedSyncsUnmergedReadStartsNothingInASubmodule(t *testing.T) {
	r, stale, live := subFilterRepo(t)
	r.Git("checkout", "-q", "-b", "side")
	r.Write("a.txt", "side\n")
	r.Git("-c", "filter.subown.clean=", "commit", "-q", "-am", "side")
	r.Git("checkout", "-q", "-")
	r.Write("a.txt", "main\n")
	r.Git("-c", "filter.subown.clean=", "commit", "-q", "-am", "main")
	merge := exec.Command("git", "-C", r.Root(), "-c", "filter.subown.clean=", "merge", "-q", "--no-edit", "side")
	merge.Env = r.Env()
	if out, err := merge.CombinedOutput(); err == nil {
		t.Fatalf("fixture: the merge conflicts: %s", out)
	}
	_ = live()
	stale()
	plain := exec.Command("git", "-C", r.Root(), "diff", "--name-only", "--diff-filter=U", "-z")
	plain.Env = r.Env()
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("fixture: git diff: %v\n%s", err, out)
	}
	if !live() {
		t.Fatal("fixture: a plain git diff of the working tree did not run the submodule's clean filter, so this test proves nothing")
	}
	stale()
	got, err := unmergedPaths(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if live() {
		t.Error("the read of the unmerged paths ran the submodule's own clean filter")
	}
	if got != "a.txt\x00" {
		t.Errorf("the conflicted file is listed: %q", got)
	}
}

// TestTheDrivesOtherWorkingTreeCommandsStartNothingInASubmodule: with the
// submodule's .gitmodules entry set to look inside it, the drive's other git
// commands that write or judge a working tree, as the drive runs them, start
// nothing inside the submodule: the pick commit, which always has its
// record to commit (git prints a status, looking inside the submodule, only
// when a commit has nothing to commit, which the fixture shows), the
// landing's records commit, made with --allow-empty when nothing is staged,
// the reset of the local tier out of the landing's index, a restart's clean
// and reset --hard, the abort of a failed sync's merge, and the removal of a
// merged lane's worktree, which git refuses for a worktree holding a
// checked-out submodule before it reads that worktree's status.
func TestTheDrivesOtherWorkingTreeCommandsStartNothingInASubmodule(t *testing.T) {
	r, stale, live := subFilterRepo(t)
	stale()
	if _, err := commitGit(r.Root(), "commit", "-q", "-m", "nothing", "--", "a.txt"); err == nil {
		t.Fatal("fixture: a commit with nothing to commit fails")
	}
	if !live() {
		t.Fatal("fixture: a commit with nothing to commit did not run the submodule's clean filter, so this test proves nothing")
	}
	run := func(name string, prep func(), cmd func() error) {
		t.Helper()
		if prep != nil {
			prep()
		}
		stale()
		if err := cmd(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if live() {
			t.Errorf("%s ran the submodule's own clean filter", name)
		}
	}
	run("the pick commit", func() { r.Write("a.txt", "picked\n") }, func() error {
		_, err := commitGit(r.Root(), "commit", "-q", "-m", "pick", "--", "a.txt")
		return err
	})
	run("the landing's records commit", nil, func() error {
		_, err := gitutil.LocalGit(r.Root(), "commit", "-q", "--allow-empty", "-m", "records")
		return err
	})
	run("the reset of a path out of the index", func() {
		r.Write("a.txt", "staged\n")
		r.Git("-c", "filter.subown.clean=", "add", "a.txt")
	}, func() error {
		_, err := pickGit(r.Root(), "reset", "-q", "--", "a.txt")
		return err
	})
	run("a restart's clean", func() { r.Write("junk/j.txt", "j\n") }, func() error {
		_, err := pickGit(r.Root(), "clean", "-f", "-d", "--quiet", "--")
		return err
	})
	run("a restart's reset --hard", nil, func() error { return resetLaneTree(r.Root(), r.Git("rev-parse", "HEAD")) })
	run("the abort of a failed merge", func() {
		r.Git("checkout", "-q", "-b", "side")
		r.Write("a.txt", "side\n")
		r.Git("-c", "filter.subown.clean=", "commit", "-q", "-am", "side")
		r.Git("checkout", "-q", "-")
		r.Write("a.txt", "main\n")
		r.Git("-c", "filter.subown.clean=", "commit", "-q", "-am", "main")
		merge := exec.Command("git", "-C", r.Root(), "-c", "filter.subown.clean=", "merge", "-q", "--no-edit", "side")
		merge.Env = r.Env()
		if out, err := merge.CombinedOutput(); err == nil {
			t.Fatalf("fixture: the merge conflicts: %s", out)
		}
		_ = live()
	}, func() error {
		_, err := pickGit(r.Root(), "merge", "--abort")
		return err
	})

	lane := filepath.Join(t.TempDir(), "lane")
	r.Git("worktree", "add", "-q", "--detach", lane, "HEAD")
	r.Git("-C", lane, "-c", "protocol.file.allow=always", "submodule", "--quiet", "update", "--init")
	subDir := r.Git("-C", filepath.Join(lane, "sub"), "rev-parse", "--path-format=absolute", "--git-dir")
	script := r.Git("config", "--file", filepath.Join(r.Root(), ".git", "modules", "sub", "config"), "filter.subown.clean")
	r.Git("config", "--file", filepath.Join(subDir, "config"), "filter.subown.clean", script)
	if err := os.MkdirAll(filepath.Join(subDir, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "info", "attributes"), []byte("* filter=subown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(48 * time.Hour)
	if err := os.Chtimes(filepath.Join(lane, "sub", "s.txt"), later, later); err != nil {
		t.Fatal(err)
	}
	_ = live()
	_, err := pickGit(r.Root(), "worktree", "remove", "--", lane)
	if err == nil || !strings.Contains(err.Error(), "submodules cannot be moved or removed") {
		t.Errorf("git refuses to remove a worktree holding a checked-out submodule: %v", err)
	}
	if live() {
		t.Error("the removal of a lane's worktree ran its submodule's own clean filter")
	}
}

// TestTheSubmoduleHandBackPrintsNoControlCharacter: a gitlink's path is
// whatever the lane's index records, any byte but NUL, the default branch's
// name is origin's and the branch is the lane's; the sync's hand-back over a
// checked-out submodule (submoduleRefusal, checkedOutSubmodule) puts each
// through termsafe.Sanitize, so neither a newline nor an escape sequence
// reaches the terminal. Watched failing with the remedy writing the path,
// the default branch and the branch raw.
func TestTheSubmoduleHandBackPrintsNoControlCharacter(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Git("config", "commit.gpgsign", "false")
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	path := "sub\nline\x1b[31m"
	r.Git("update-index", "--add", "--cacheinfo", "160000,"+r.Git("rev-parse", "HEAD")+","+path)
	r.Git("commit", "-q", "-m", "a gitlink with a newline and an escape in its path")
	dir := filepath.Join(r.Root(), path)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lane := Lane{ID: "lane-1", Branch: "build/\x1b[2Jlane\n1", Worktree: r.Root()}
	err := syncSubmoduleRefusal(lane, "ma\x1b]0;in\n")
	rf, ok := AsRefusal(err)
	if !ok {
		t.Fatalf("the sync over a checked-out submodule is handed back: %v", err)
	}
	var cs *checkedOutSubmodule
	if !errors.As(checkedOutGitlink(r.Root()), &cs) || cs.Path != path {
		t.Fatalf("the gitlink is found checked out at its own path: %+v", cs)
	}
	for name, text := range map[string]string{"reason": rf.Reason, "remedy": rf.Remedy, "error": cs.Error()} {
		if strings.ContainsAny(text, "\n\x1b") {
			t.Errorf("the hand-back's %s prints a control character: %q", name, text)
		}
		if !strings.Contains(text, "sub?line?[31m") {
			t.Errorf("the hand-back's %s names the path, sanitized: %q", name, text)
		}
	}
	if !strings.Contains(rf.Remedy, "ma?]0;in?") || !strings.Contains(rf.Remedy, "build/?[2Jlane?1") {
		t.Errorf("the remedy names the default branch and the branch, sanitized: %q", rf.Remedy)
	}
}
