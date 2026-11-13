package gitutil_test

// filter_ask_test.go holds ruling 1 of fix round 5 of iss-2611131308266006:
// an isolated git that can start a program the repository names which
// ExecPins leaves on (a content filter, a textconv or external diff program,
// a merge driver) asks the check SetBeforeHookedGit set, as the hooked
// runners do; one that blanks them, or reads no content through them, does
// not, and with no check set nothing changes.

import (
	"errors"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// countingCheck sets, for the rest of the test, a check that records each
// command it is asked about and refuses with refused (nil lets git run).
func countingCheck(t *testing.T, refused error) *[][]string {
	t.Helper()
	var asked [][]string
	unset, err := gitutil.SetBeforeHookedGit(func(args []string) error {
		asked = append(asked, append([]string(nil), args...))
		return refused
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unset)
	return &asked
}

// TestAnIsolatedReadThatCanRunAFilterAsksTheCheck: in a repository whose
// settings name a clean filter for its file, an isolated hash-object of that
// file and a status with the filters switched back on (filter-roots) each
// ask the check before git runs; under a refusing check each fails with a
// NotRunError naming its subcommand, and the filter does not run. A
// hash-object with --no-filters and a status with the filters off ask
// nothing. With no check set the hash-object runs the filter, as it always
// has. Watched failing before isolatedGit asked: both reads ran the filter
// under the refusing check.
func TestAnIsolatedReadThatCanRunAFilterAsksTheCheck(t *testing.T) {
	refused := errors.New("held setting changed: filter.evil.clean")
	t.Run("hash-object", func(t *testing.T) {
		r, _, mark := filterFixture(t)
		asked := countingCheck(t, refused)
		_, err := gitutil.Run(r.Root(), "hash-object", "--", "a.txt")
		var nr *gitutil.NotRunError
		if !errors.As(err, &nr) || nr.Verb != "hash-object" || !errors.Is(err, refused) || len(*asked) != 1 {
			t.Fatalf("the read asks the check and is not run: %v, asked %q", err, *asked)
		}
		if ran(mark) {
			t.Fatal("the clean filter ran under a refusing check")
		}
		if _, err := gitutil.Run(r.Root(), "hash-object", "--no-filters", "--", "a.txt"); err != nil || len(*asked) != 1 {
			t.Fatalf("a read with the filters off asks nothing: %v, asked %q", err, *asked)
		}
	})
	t.Run("status with the filters on", func(t *testing.T) {
		r, home, mark := filterFixture(t)
		declare(t, home, r.Root()+"\n", 0o600)
		asked := countingCheck(t, refused)
		_, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{})
		var nr *gitutil.NotRunError
		if !errors.As(err, &nr) || nr.Verb != "status" || len(*asked) != 1 {
			t.Fatalf("the status asks the check and is not run: %v, asked %q", err, *asked)
		}
		if ran(mark) {
			t.Fatal("the clean filter ran under a refusing check")
		}
	})
	t.Run("status with the filters off", func(t *testing.T) {
		r, _, mark := filterFixture(t)
		asked := countingCheck(t, refused)
		if _, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{}); err != nil || len(*asked) != 0 {
			t.Fatalf("a status that starts no filter asks nothing: %v, asked %q", err, *asked)
		}
		if ran(mark) {
			t.Fatal("the filter is off")
		}
	})
	t.Run("no check set", func(t *testing.T) {
		r, _, mark := filterFixture(t)
		if _, err := gitutil.Run(r.Root(), "hash-object", "--", "a.txt"); err != nil {
			t.Fatal(err)
		}
		if !ran(mark) {
			t.Fatal("with no check set the read runs as it always has, filter and all")
		}
	})
}

// TestWhatAnIsolatedCommandCanStartIsReadFromItsCommandLine: the commands a
// drive's isolated reads run, and their neighbours, are judged by what they
// can start.
func TestWhatAnIsolatedCommandCanStartIsReadFromItsCommandLine(t *testing.T) {
	blank := []string{"-c", "filter.lfs.clean=", "-c", "filter.lfs.smudge=", "-c", "filter.lfs.process="}
	drivers := []string{"-c", "merge.default=git merge-file %A %O %B"}
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"hash-object", "--", "a"}, true},
		{[]string{"hash-object", "--no-filters", "--", "a"}, false},
		{[]string{"hash-object", "-w", "-t", "tree", "--stdin"}, false},
		{[]string{"hash-object", "-w", "--stdin"}, false},
		{[]string{"hash-object", "--stdin", "--path", "a"}, true},
		{append(blank, "hash-object", "--", "a"), false},
		{[]string{"--no-optional-locks", "status", "--porcelain=v1"}, true},
		{append(blank, "--no-optional-locks", "status", "--porcelain=v1"), false},
		{[]string{"diff", "--quiet"}, true},
		{[]string{"diff", "--cached", "--raw", "--no-ext-diff", "HEAD", "--"}, false},
		{[]string{"diff", "--cached", "--binary", "--no-ext-diff", "--no-textconv", "HEAD", "--"}, false},
		{[]string{"diff", "--cached", "--binary", "HEAD", "--"}, true},
		{[]string{"diff", "--name-only", "a", "b", "--"}, false},
		{[]string{"diff", "--text", "--no-ext-diff", "--no-textconv", "a..b", "--"}, false},
		{[]string{"diff", "a..b"}, true},
		{[]string{"diff-tree", "-r", "--raw", "a", "b"}, false},
		{[]string{"diff-tree", "-p", "--textconv", "a", "b"}, true},
		{[]string{"diff-files", "-p"}, true},
		{[]string{"diff-index", "--cached", "-p", "HEAD"}, false},
		{[]string{"show", "-s", "--format=%H", "HEAD"}, false},
		{[]string{"show", "HEAD:a.txt"}, false},
		{[]string{"show", "--textconv", "HEAD:a.txt"}, true},
		{[]string{"show", "HEAD"}, true},
		{[]string{"show", "--no-textconv", "--no-ext-diff", "HEAD"}, false},
		{[]string{"log", "--format=%H", "-n", "1"}, false},
		{[]string{"log", "-p", "--no-ext-diff", "--no-textconv"}, false},
		{[]string{"log", "-p"}, true},
		{[]string{"log", "-Gneedle"}, true},
		{[]string{"cat-file", "--batch"}, false},
		{[]string{"cat-file", "--filters", "HEAD:a"}, true},
		{[]string{"cat-file", "--textconv", "HEAD:a"}, true},
		{[]string{"grep", "-n", "x"}, false},
		{[]string{"blame", "--no-textconv", "HEAD", "--", "a"}, false},
		{[]string{"blame", "--", "a"}, true},
		{[]string{"archive", "HEAD"}, true},
		{[]string{"apply", "--cached"}, false},
		{[]string{"apply"}, true},
		{[]string{"merge-tree", "--write-tree", "a", "b"}, true},
		{append(drivers, "merge-tree", "--write-tree", "a", "b"), false},
		{[]string{"worktree", "add", "x"}, true},
		{[]string{"worktree", "list", "--porcelain"}, false},
		{[]string{"reset", "--soft", "HEAD"}, false},
		{[]string{"reset", "-q", "--", "a"}, true},
		{[]string{"ls-files", "-z"}, false},
		{[]string{"ls-files", "-m"}, true},
		{[]string{"merge-file", "a", "b", "c"}, false},
		{[]string{"rev-parse", "HEAD"}, false},
		{[]string{"-c", "core.splitIndex=false", "config", "--get", "x"}, false},
	} {
		if got := gitutil.LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.args, got, tc.want)
		}
	}
	if gitutil.Subcommand([]string{"-c", "a=b", "--no-optional-locks", "-C", "x", "status"}) != "status" {
		t.Error("the subcommand is read past git's global options")
	}
}
