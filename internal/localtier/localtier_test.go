package localtier_test

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/localtier"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// checkout is a repository that ignores the local tier, with one state file
// in it written but never committed.
func checkout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local", "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".abcd", ".work.local", "run", "drain.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "seed")
	return dir
}

// TestAStateFileThisCheckoutWroteIsRead: an untracked, ignored file in the
// tier is this checkout's own, so it passes; so does one the tier does not
// hold at all, and every path outside a repository.
func TestAStateFileThisCheckoutWroteIsRead(t *testing.T) {
	dir := checkout(t)
	for _, rel := range []string{".abcd/.work.local/run/drain.json", ".abcd/.work.local/run/absent.json", ".abcd/.work.local/run"} {
		if err := localtier.Untracked(dir, rel); err != nil {
			t.Fatalf("Untracked(%s) = %v; want it read", rel, err)
		}
	}
	if err := localtier.Untracked(t.TempDir(), ".abcd/.work.local/run/drain.json"); err != nil {
		t.Fatalf("Untracked outside a repository = %v; want it read", err)
	}
}

// TestAStateFileACommitCarriesIsRefused: a state file git tracks (force-added
// past the ignore rule) came with a commit, not from this checkout, so it is
// refused, naming it; and so is a directory holding one, for a reader that
// lists it.
func TestAStateFileACommitCarriesIsRefused(t *testing.T) {
	dir := checkout(t)
	git(t, dir, "add", "-f", ".abcd/.work.local/run/drain.json")
	git(t, dir, "commit", "-q", "-m", "carry it")
	for _, rel := range []string{".abcd/.work.local/run/drain.json", ".abcd/.work.local/run"} {
		err := localtier.Untracked(dir, rel)
		var r *localtier.Refused
		if !errors.As(err, &r) || r.Cause != nil || len(r.Paths) != 1 || r.Paths[0] != ".abcd/.work.local/run/drain.json" {
			t.Fatalf("Untracked(%s) over a tracked state file = %v; want a refusal naming it", rel, err)
		}
		if !strings.Contains(r.Reason(), "git tracks .abcd/.work.local/run/drain.json") || !strings.Contains(r.Remedy(), "git rm --cached") {
			t.Fatalf("the refusal = %q; want it to name the tracked file", err)
		}
	}
}

// TestAStateFileGitCannotAnswerForIsRefused: in a repository git cannot read
// (a corrupt index), whether the file is tracked is unknown, so it is refused.
func TestAStateFileGitCannotAnswerForIsRefused(t *testing.T) {
	dir := checkout(t)
	if err := os.WriteFile(filepath.Join(dir, ".git", "index"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := localtier.Untracked(dir, ".abcd/.work.local/run/drain.json")
	var r *localtier.Refused
	if !errors.As(err, &r) || r.Cause == nil || r.Paths != nil || !strings.Contains(r.Reason(), "could not ask git") {
		t.Fatalf("Untracked over an unreadable index = %v; want a refusal saying git could not answer", err)
	}
}

// TestAReaderHoldingARootOnTheTierIsGuardedToo: a reader with an os.Root
// opened on a directory inside the tier, its path relative to that root, reads
// what this checkout wrote and is refused what a commit carries, by a refusal
// that is never taken for an absent file.
func TestAReaderHoldingARootOnTheTierIsGuardedToo(t *testing.T) {
	dir := checkout(t)
	open := func() *os.Root {
		t.Helper()
		root, err := os.OpenRoot(filepath.Join(dir, ".abcd", ".work.local", "run"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { root.Close() })
		return root
	}
	root := open()
	if data, err := localtier.ReadInRoot(root, "drain.json", 1<<10); err != nil || string(data) != "{}" {
		t.Fatalf("ReadInRoot(this checkout's own state) = %q, %v; want it read", data, err)
	}
	if _, err := localtier.LstatInRoot(root, "drain.json"); err != nil {
		t.Fatalf("LstatInRoot(this checkout's own state) = %v; want it found", err)
	}
	git(t, dir, "add", "-f", ".abcd/.work.local/run/drain.json")
	git(t, dir, "commit", "-q", "-m", "carry it")
	var r *localtier.Refused
	if _, err := localtier.ReadInRoot(root, "drain.json", 1<<10); !errors.As(err, &r) || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadInRoot(a state file a commit carries) = %v; want a refusal", err)
	}
	// Named from the working tree's top, so the remedy runs from there.
	if want := ".abcd/.work.local/run/drain.json"; r.Rel != want || len(r.Paths) != 1 || r.Paths[0] != want || !strings.Contains(r.Reason(), want) {
		t.Fatalf("the refusal names %q and %q, reason %q; want each repository-relative (%s)", r.Rel, r.Paths, r.Reason(), want)
	}
	if _, err := localtier.LstatInRoot(root, "drain.json"); !errors.As(err, &r) {
		t.Fatalf("LstatInRoot(a state file a commit carries) = %v; want a refusal", err)
	}
}

// gitOut is git's trimmed output in dir.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// TestWhatReachesTheTierByAnotherNameIsRefused: what git tracks is judged by
// the file it names, not by its spelling. A spelling a case-folding filesystem
// opens as the tier (a capital, or the Kelvin sign for k), a link or a
// submodule standing in for the tier, and one standing in for .abcd each
// refuse a read of the tier's own state, though no entry has the tier's name.
func TestWhatReachesTheTierByAnotherNameIsRefused(t *testing.T) {
	for name, entry := range map[string]struct{ mode, path string }{
		"a capital":        {"100644", ".abcd/.Work.Local/run/drain.json"},
		"the Kelvin sign":  {"100644", ".abcd/.wor\u212a.local/run/drain.json"},
		"a link":           {"120000", ".abcd/.work.local"},
		"a submodule":      {"160000", ".abcd/.work.local"},
		"a link for .abcd": {"120000", ".abcd"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := checkout(t)
			blob := gitOut(t, dir, "rev-parse", "HEAD:.gitignore")
			if entry.mode == "160000" {
				blob = gitOut(t, dir, "rev-parse", "HEAD")
			}
			git(t, dir, "update-index", "--add", "--cacheinfo", entry.mode+","+blob+","+entry.path)
			err := localtier.Untracked(dir, ".abcd/.work.local/run/drain.json")
			var r *localtier.Refused
			if !errors.As(err, &r) || r.Cause != nil || !slices.Contains(r.Paths, entry.path) {
				t.Fatalf("Untracked with %s tracked = %v; want a refusal naming it", entry.path, err)
			}
			if got, err := localtier.Carried(dir); err != nil || !slices.Contains(got, entry.path) {
				t.Fatalf("Carried with %s tracked = %q, %v; want it listed", entry.path, got, err)
			}
		})
	}
}

// TestWhatTheTierCarriesElsewhereLeavesThisReadAlone: a refusal is for the
// file being read. An entry elsewhere in the tier, by its own name or another
// spelling of the tier's, does not refuse a read it supplies nothing to, and
// does refuse a read of the whole tier; an entry that opens the file, or a
// directory above it, by another letter case of the path inside the tier is
// refused as the file's own name is.
func TestWhatTheTierCarriesElsewhereLeavesThisReadAlone(t *testing.T) {
	for _, c := range []struct {
		mode, path string
		refused    bool
	}{
		{"100644", ".abcd/.work.local/scratch/note.md", false},
		{"100644", ".abcd/.Work.Local/ledger/entry.md", false},
		{"100644", ".abcd/.work.local/runs/drain.json", false},
		{"100644", ".abcd/.Work.Local/RUN/Drain.json", true},
		{"120000", ".abcd/.worK.local/Run", true},
	} {
		dir := checkout(t)
		blob := gitOut(t, dir, "rev-parse", "HEAD:.gitignore")
		git(t, dir, "update-index", "--add", "--cacheinfo", c.mode+","+blob+","+c.path)
		err := localtier.Untracked(dir, ".abcd/.work.local/run/drain.json")
		if refused := err != nil; refused != c.refused {
			t.Errorf("reading run/drain.json with %s tracked: %v; want refused %v", c.path, err, c.refused)
		}
		var r *localtier.Refused
		if err := localtier.Untracked(dir, localtier.RelDir); !errors.As(err, &r) || !slices.Contains(r.Paths, c.path) {
			t.Errorf("reading the whole tier with %s tracked = %v; want a refusal naming it", c.path, err)
		}
	}
}

// TestASpellingOnlyTheFilesystemFoldsIsRefused: below the tier a name is judged
// by the file it opens too, so a spelling that full case folding opens as the
// file being read (a ligature for its st), which simple folding of the names
// alone does not match, is refused where the filesystem folds it.
func TestASpellingOnlyTheFilesystemFoldsIsRefused(t *testing.T) {
	dir := checkout(t)
	state := filepath.Join(dir, ".abcd", ".work.local", "run", "state.json")
	if err := os.WriteFile(state, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	spelled := ".abcd/.work.local/run/ﬆate.json"
	real, errR := os.Lstat(state)
	folded, errF := os.Lstat(filepath.Join(dir, filepath.FromSlash(spelled)))
	if errR != nil || errF != nil || !os.SameFile(real, folded) {
		t.Skip("this filesystem does not fold the st ligature onto st")
	}
	blob := gitOut(t, dir, "rev-parse", "HEAD:.gitignore")
	git(t, dir, "update-index", "--add", "--cacheinfo", "100644,"+blob+","+spelled)
	var r *localtier.Refused
	if err := localtier.Untracked(dir, ".abcd/.work.local/run/state.json"); !errors.As(err, &r) || !slices.Contains(r.Paths, spelled) {
		t.Fatalf("Untracked with %s tracked = %v; want a refusal naming it", spelled, err)
	}
	if err := localtier.Untracked(dir, ".abcd/.work.local/run/drain.json"); err != nil {
		t.Fatalf("the spelling opens state.json, not drain.json: %v", err)
	}
}
