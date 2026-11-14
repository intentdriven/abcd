package loop

// compare_reads_test.go holds iss-2611140432160343: the comparison of what a
// drive holds never asks itself, read from the command lines it makes; and a
// git that was not run is a failed read before any exit status.

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// recordAsks wraps the ask of every environment a drive's git runs in, for
// the rest of the test, so each command line a reading makes is recorded.
func recordAsks(t *testing.T) func() [][]string {
	t.Helper()
	var mu sync.Mutex
	var lines [][]string
	for _, e := range driveEnvs() {
		ask := e.ask
		e.ask = func(ctx context.Context, env []string, dir string, limit int, args ...string) (string, error) {
			mu.Lock()
			lines = append(lines, append([]string(nil), args...))
			mu.Unlock()
			return ask(ctx, env, dir, limit, args...)
		}
		t.Cleanup(func() { e.ask = ask })
	}
	return func() [][]string {
		mu.Lock()
		defer mu.Unlock()
		return append([][]string(nil), lines...)
	}
}

// TestTheComparisonsOwnReadsNeverAsk: every command line the comparison of
// what a drive holds makes, read as it makes them, in a repository with an
// include path and on a git that knows the attributes files' names and one
// that does not, is one gitutil.LeavesContentProgramsOn says starts nothing,
// and the check SetBeforeHookedGit set is asked nothing while the drive takes
// what it holds and compares it. So a change to the table that would make the
// comparison ask itself fails here, not as a comparison without end. Watched
// failing with "var" taken off gitutil's starts-nothing list.
func TestTheComparisonsOwnReadsNeverAsk(t *testing.T) {
	for _, older := range []bool{false, true} {
		t.Run(fmt.Sprintf("git older than 2.42: %v", older), func(t *testing.T) {
			if older {
				standInGit(t, `*"var GIT_ATTR_GLOBAL"|*"var GIT_ATTR_SYSTEM") echo 'usage: git var (-l | <variable>)' >&2; exit 129 ;;`)
			}
			repo := briefRepo(t, agentsMarked)
			writeMode(t, filepath.Join(repo.Root(), ".git", "inc.cfg"), "[core]\n\teditor = /bin/true\n", 0o644)
			repo.Git("config", "include.path", "inc.cfg")
			lines := recordAsks(t)
			var asked [][]string
			var mu sync.Mutex
			unset, err := gitutil.SetBeforeHookedGit(func(args []string) error {
				mu.Lock()
				asked = append(asked, args)
				mu.Unlock()
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			defer unset()
			id, err := readRemoteIdentity(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			if err := heldBeforeGit(repo.Root(), id); err != nil {
				t.Fatal(err)
			}
			got := lines()
			verbs := map[string]bool{}
			for _, l := range got {
				verbs[gitutil.Subcommand(l)] = true
				if gitutil.LeavesContentProgramsOn(l) {
					t.Errorf("the comparison's read %q would ask the comparison", l)
				}
			}
			for _, v := range []string{"config", "var", "rev-parse"} {
				if !verbs[v] {
					t.Errorf("the comparison made no %s read: %q", v, got)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if len(asked) != 0 {
				t.Fatalf("the comparison asked the check: %q", asked)
			}
		})
	}
}

// exitedAs is a git's real exit of status code, wrapped as a cause can
// carry it.
func exitedAs(t *testing.T, code int) error {
	t.Helper()
	err := exec.Command("/bin/sh", "-c", fmt.Sprintf("exit %d", code)).Run()
	if exitStatus(err) != code {
		t.Fatalf("fixture: exit %d gave %v", code, err)
	}
	return fmt.Errorf("the read of core.sshCommand failed: %w", err)
}

// TestAGitNotRunIsAFailedReadBeforeAnyStatus: a read whose git was not run
// (a gitutil.NotRunError) is a readFailure naming the read, whatever exit
// status its cause wraps: never git's "there is none" (status 1, nothing
// written) nor a git too old to know the name (129). Watched failing before:
// both were answers of that status.
func TestAGitNotRunIsAFailedReadBeforeAnyStatus(t *testing.T) {
	for _, code := range []int{1, 129} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			notRun := &gitutil.NotRunError{Verb: "var", Err: exitedAs(t, code)}
			e := &gitEnv{name: "a test's environment", environ: func() []string { return nil },
				ask: func(context.Context, []string, string, int, ...string) (string, error) { return "", notRun }}
			a, err := e.read(t.TempDir(), "asking git for its global attributes file", 4096, "var", "GIT_ATTR_GLOBAL")
			var rf *readFailure
			if !errors.As(err, &rf) || a.code != -1 || !strings.Contains(err.Error(), "was not run") {
				t.Fatalf("a git not run is a failed read: %+v, %v", a, err)
			}
			if _, err := attributesFiles(e, t.TempDir(), ""); !errors.As(err, &rf) {
				t.Fatalf("its attributes files are not read as none: %v", err)
			}
		})
	}
}
