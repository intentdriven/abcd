package loop

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// driverFixture is a repository whose main and side branches both changed
// f.txt from a shared base, with a merge driver configured that writes its own
// bytes into the result and leaves a mark when it runs.
type driverFixture struct {
	r            *gittest.Repo
	ours, theirs string
	mark         string
}

func newDriverFixture(t *testing.T, base, ours, theirs, attributes string, cfg map[string]string) driverFixture {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Git("config", "user.name", "Fixture")
	r.Git("config", "user.email", "fixture@example.invalid")
	r.Write("f.txt", base)
	r.Commit("base")
	r.Git("checkout", "-q", "-b", "side")
	r.Write("f.txt", theirs)
	r.Commit("side")
	theirsSHA := r.Git("rev-parse", "HEAD")
	r.Git("checkout", "-q", "main")
	r.Write("f.txt", ours)
	r.Commit("main")
	oursSHA := r.Git("rev-parse", "HEAD")

	mark := filepath.Join(t.TempDir(), "driver-ran")
	driver := filepath.Join(t.TempDir(), "evil-driver.sh")
	if err := os.WriteFile(driver, []byte("#!/bin/sh\ntouch "+mark+"\necho DRIVER > \"$1\"\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for k, v := range cfg {
		r.Git("config", k, strings.ReplaceAll(v, "DRIVER", driver+" %A"))
	}
	if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "attributes"), []byte(attributes), 0o644); err != nil {
		t.Fatal(err)
	}
	return driverFixture{r: r, ours: oursSHA, theirs: theirsSHA, mark: mark}
}

// plainMerge is a merge the operator runs, outside pickGit, under the same
// environment the loop's git gets; it reports whether git merged cleanly.
func (f driverFixture) plainMerge(t *testing.T) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", f.r.Root(), "-c", "core.hooksPath=/dev/null", "merge", "--no-ff", "--no-edit", "-m", "plain", f.theirs)
	cmd.Env = gitutil.ScrubbedEnv()
	err := cmd.Run()
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		t.Fatal(err)
	}
	return err == nil
}

// undo puts main back at its own commit with a clean tree.
func (f driverFixture) undo(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(f.r.Root(), ".git", "MERGE_HEAD")); err == nil {
		f.r.Git("merge", "--abort")
	}
	f.r.Git("reset", "-q", "--hard", f.ours)
}

func (f driverFixture) result(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.r.Root(), "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestTheSyncMergeRunsNoConfiguredMergeDriver is iss-2610090821510097. The
// sync's merge goes through pickGit, which keeps the repository's config, so a
// merge driver the repository configures (named by an attribute, or by
// merge.default) runs as the operator wherever both sides changed a path, and
// its bytes become the merge result. The sync must always use git's built-in
// merge: the driver does not start, and the merged file, clean or conflicted,
// is byte for byte what git's built-in merge makes of the same two commits. A
// merge the operator runs outside pickGit still honours the driver.
func TestTheSyncMergeRunsNoConfiguredMergeDriver(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	const base = "a\nb\nc\nd\ne\nf\ng\n"
	for _, tc := range []struct {
		name         string
		ours, theirs string
		attributes   string
		cfg          map[string]string
		unset        []string
		conflict     bool
	}{
		{"an attribute names the driver, clean", "a\nB\nc\nd\ne\nf\ng\n", "a\nb\nc\nd\ne\nF\ng\n",
			"* merge=evil\n", map[string]string{"merge.evil.driver": "DRIVER"}, []string{"merge.evil.driver"}, false},
		{"an attribute names the driver, conflicting", "a\nOURS\nc\nd\ne\nf\ng\n", "a\nTHEIRS\nc\nd\ne\nf\ng\n",
			"* merge=evil\n", map[string]string{"merge.evil.driver": "DRIVER"}, []string{"merge.evil.driver"}, true},
		{"merge.default names the driver", "a\nB\nc\nd\ne\nf\ng\n", "a\nb\nc\nd\ne\nF\ng\n",
			"", map[string]string{"merge.evil.driver": "DRIVER", "merge.default": "evil"}, []string{"merge.evil.driver", "merge.default"}, false},
		{"a dotted driver name", "a\nB\nc\nd\ne\nf\ng\n", "a\nb\nc\nd\ne\nF\ng\n",
			"* merge=e.vil\n", map[string]string{"merge.e.vil.driver": "DRIVER"}, []string{"merge.e.vil.driver"}, false},
		{"a driver named like the built-in", "a\nB\nc\nd\ne\nf\ng\n", "a\nb\nc\nd\ne\nF\ng\n",
			"* merge=text\n", map[string]string{"merge.text.driver": "DRIVER"}, []string{"merge.text.driver"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDriverFixture(t, base, tc.ours, tc.theirs, tc.attributes, tc.cfg)

			// The fixture is live: a merge the operator runs starts the driver.
			f.plainMerge(t)
			if _, err := os.Stat(f.mark); err != nil {
				t.Fatal("fixture: a plain merge did not start the configured driver, so this test proves nothing")
			}
			if err := os.Remove(f.mark); err != nil {
				t.Fatal(err)
			}
			f.undo(t)

			err := syncMerge(f.r.Root(), "sync", f.theirs)
			if _, serr := os.Stat(f.mark); serr == nil {
				t.Fatal("the sync merge started the repository's merge driver")
			}
			if (err != nil) != tc.conflict {
				t.Fatalf("the sync merge's outcome: err=%v, want conflict=%v", err, tc.conflict)
			}
			got := f.result(t)
			f.undo(t)

			// git's built-in merge of the same two commits, with the driver gone.
			for _, k := range tc.unset {
				f.r.Git("config", "--unset", k)
			}
			if clean := f.plainMerge(t); clean == tc.conflict {
				t.Fatalf("fixture: the built-in merge's outcome is clean=%v, want conflict=%v", clean, tc.conflict)
			}
			if want := f.result(t); got != want {
				t.Fatalf("the sync merge's result is not git's built-in result:\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// TestTheSyncMergeRefusesADriverItCannotSwitchOff: a driver name `git -c`
// cannot carry intact is refused before any merge, rather than overridden
// under a different key and left live.
func TestTheSyncMergeRefusesADriverItCannotSwitchOff(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	f := newDriverFixture(t, "a\nb\nc\n", "A\nb\nc\n", "a\nb\nC\n", "* merge=x=y\n", map[string]string{"merge.x=y.driver": "DRIVER"})
	if err := syncMerge(f.r.Root(), "sync", f.theirs); err == nil {
		t.Fatal("a driver name holding '=' must refuse the sync merge")
	}
	if _, err := os.Stat(f.mark); err == nil {
		t.Fatal("the refused sync merge started the driver")
	}
	if head := f.r.Git("rev-parse", "HEAD"); head != f.ours {
		t.Fatalf("the refused sync merge moved HEAD to %s", head)
	}
}
