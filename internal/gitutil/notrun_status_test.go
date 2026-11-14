package gitutil_test

// notrun_status_test.go holds iss-2611140432160343: a git that was not run is
// a failed read, by its type, before any exit status is read from what it
// wraps; and the networked runners' errors keep their lead.

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// exitedWith is a real *exec.ExitError of status code, wrapped as a cause
// can carry one (the read of core.sshCommand wraps its git's).
func exitedWith(t *testing.T, code int) error {
	t.Helper()
	err := exec.Command("/bin/sh", "-c", fmt.Sprintf("exit %d", code)).Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != code {
		t.Fatalf("fixture: exit %d gave %v", code, err)
	}
	return fmt.Errorf("the read failed: %w", err)
}

// TestAGitNotRunIsAFailedReadWhateverStatusItWraps: a merge-tree the check
// stops, with a refusal that wraps a git's exit status 1 or 129, is an error,
// never merge-tree's "it conflicts" (status 1); and ExitCode reads it as no
// status at all. Watched failing before: the stop was answered as a conflict.
func TestAGitNotRunIsAFailedReadWhateverStatusItWraps(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("one")
	for _, code := range []int{1, 129} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			refused := exitedWith(t, code)
			asked := countingCheck(t, refused)
			out, one, err := gitutil.RunCappedExitOne(r.Root(), 1<<20, "merge-tree", "--write-tree", "HEAD", "HEAD")
			var nr *gitutil.NotRunError
			if !errors.As(err, &nr) || one || out != nil || len(*asked) != 1 {
				t.Fatalf("a stopped merge-tree is a failed read: one=%v, %v, asked %q", one, err, *asked)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			out, one, err = gitutil.RunCappedExitOneContext(ctx, r.Root(), 1<<20, "merge-tree", "--write-tree", "HEAD", "HEAD")
			if !errors.As(err, &nr) || one || out != nil {
				t.Fatalf("a stopped merge-tree is a failed read under a context too: one=%v, %v", one, err)
			}
			if c := gitutil.ExitCode(err); c != -1 {
				t.Fatalf("a git not run has no exit status: %d", c)
			}
			if c := gitutil.ExitCode(refused); c != code {
				t.Fatalf("a git that ran has its own: %d", c)
			}
		})
	}
}

// TestANetworkedRunnersEarlyErrorKeepsItsLead: a command exec cannot start
// (no git on PATH) comes back from LocalGit and NetAskContext led by the git
// command it is, as a failure of cmd.Run is. Watched failing before: exec's
// own words came back bare.
func TestANetworkedRunnersEarlyErrorKeepsItsLead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_SSH_COMMAND", "ssh")
	t.Setenv("PATH", t.TempDir())
	if _, err := gitutil.LocalGit(dir, "commit", "-q"); err == nil || !strings.HasPrefix(err.Error(), "git commit: ") {
		t.Fatalf("LocalGit's error is led by its command: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := gitutil.NetAskContext(ctx, dir, 4096, "var", "GIT_ATTR_GLOBAL"); err == nil || !strings.HasPrefix(err.Error(), "git var: ") {
		t.Fatalf("NetAskContext's error is led by its command: %v", err)
	}
	if _, err := gitutil.Net(dir, "fetch", "origin"); err == nil || !strings.HasPrefix(err.Error(), "git fetch: ") {
		t.Fatalf("Net's error is led by its command: %v", err)
	}
}

// TestAnAncestryCheckRefusesANameGitWouldReadAsAnOption: IsAncestor, whose
// callers pass names read from stored state, refuses a name that is empty or
// starts with a dash before git runs, naming it, and answers a real pair as
// before. Watched failing on 5dd7deff: git was run and its own usage error
// came back.
func TestAnAncestryCheckRefusesANameGitWouldReadAsAnOption(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("one")
	for _, pair := range [][2]string{{"--all", "HEAD"}, {"HEAD", "-x"}, {"", "HEAD"}} {
		ok, err := gitutil.IsAncestor(r.Root(), pair[0], pair[1])
		if ok || err == nil || !strings.Contains(err.Error(), "refusing ref") {
			t.Errorf("%q: the name is refused before git runs: %v %v", pair, ok, err)
		}
	}
	if ok, err := gitutil.IsAncestor(r.Root(), "HEAD", "HEAD"); !ok || err != nil {
		t.Fatalf("a real pair is answered: %v %v", ok, err)
	}
}
