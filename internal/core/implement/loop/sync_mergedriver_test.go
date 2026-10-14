package loop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// driverFixture is a repository whose main and side branches both changed
// f.txt from a shared base, with a merge driver configured that writes its own
// bytes into the result and leaves a mark when it runs.
type driverFixture struct {
	r      *gittest.Repo
	theirs string
	mark   string
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
	return driverFixture{r: r, theirs: theirsSHA, mark: mark}
}

func (f driverFixture) result(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.r.Root(), "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestTheSyncMergeUsesTheRepositorysMergeDriver is iss-2610111729399920. A
// merge driver the repository configures (named by an attribute, or by
// merge.default) is the repository's own merge tool: the person's own `git
// merge` runs it wherever both sides changed a path, so the sync's merge runs
// it too, and its bytes are the merge result (principle
// abcd-fixes-what-it-adds).
func TestTheSyncMergeUsesTheRepositorysMergeDriver(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX driver script")
	}
	withoutPersonalConfig(t)
	const base = "a\nb\nc\nd\ne\nf\ng\n"
	for _, tc := range []struct {
		name         string
		ours, theirs string
		attributes   string
		cfg          map[string]string
	}{
		{"an attribute names the driver", "a\nOURS\nc\nd\ne\nf\ng\n", "a\nTHEIRS\nc\nd\ne\nf\ng\n",
			"* merge=lock\n", map[string]string{"merge.lock.driver": "DRIVER"}},
		{"merge.default names the driver", "a\nB\nc\nd\ne\nf\ng\n", "a\nb\nc\nd\ne\nF\ng\n",
			"", map[string]string{"merge.lock.driver": "DRIVER", "merge.default": "lock"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDriverFixture(t, base, tc.ours, tc.theirs, tc.attributes, tc.cfg)
			if err := syncMerge(f.r.Root(), "sync", f.theirs); err != nil {
				t.Fatalf("the sync merge must be made with the repository's driver: %v", err)
			}
			if _, err := os.Stat(f.mark); err != nil {
				t.Fatal("the sync merge did not run the repository's merge driver")
			}
			if got := f.result(t); got != "DRIVER\n" {
				t.Fatalf("the sync merge's result is not the driver's: %q", got)
			}
		})
	}
}

// TestASyncTheRepositoryRefusesIsRefusedWithGitsReason is
// iss-2610111729399920: the sync's merge runs under the repository's own
// settings, and one that refuses the merge before it starts (here
// merge.verifySignatures over the landed sibling's unsigned commit) leaves no
// merge to abort. The step is refused naming what git reported, and the lane's
// branch and worktree are left as they were.
func TestASyncTheRepositoryRefusesIsRefusedWithGitsReason(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.implement(t, "lane-2", "two.txt")
	f.roundPassed(t, "lane-1")
	f.roundPassed(t, "lane-2")
	f.landed(t, "lane-1")
	judged := f.lane(t, "lane-2").HeadSHA

	f.repo.Git("config", "merge.verifySignatures", "true")
	_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "git could not merge") || !strings.Contains(r.Reason, "GPG signature") {
		t.Fatalf("the sync is refused with git's reason: %+v", r)
	}
	l2 := f.lane(t, "lane-2")
	if len(l2.Syncs) != 0 || l2.HeadSHA != judged {
		t.Fatalf("the refused sync records nothing: %+v", l2)
	}
	if tip := f.repo.Git("rev-parse", "refs/heads/"+l2.Branch); tip != judged {
		t.Fatalf("the branch stays at the judged head: %s", tip)
	}
	if st := f.repo.Git("-C", l2.Worktree, "status", "--porcelain"); st != "" {
		t.Fatalf("the refused merge leaves the worktree clean: %s", st)
	}
}
