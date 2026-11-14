package loop

// compare_reads_test.go holds iss-2611140432160343: the comparison of what a
// drive holds never asks itself, read from the command lines it makes; and a
// git that was not run is a failed read before any exit status.

import (
	"context"
	"errors"
	"fmt"
	"os"
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

// dashLedIncludes are include paths that start with a dash: one git reads
// as a word of its own of single-dash letters, one as a prefix of --edit.
var dashLedIncludes = []string{"-local/extra.cfg", "--ed"}

// includedRepo is a checkout whose own settings include value, a path
// relative to .git/config that starts with a dash, naming a file there.
// It returns the checkout and the included file.
func includedRepo(t *testing.T, value string) (root, included string) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	included = filepath.Join(repo.Root(), ".git", value)
	mkdirs(t, filepath.Dir(included))
	writeMode(t, included, "[core]\n\teditor = /bin/true\n", 0o644)
	cfg := filepath.Join(repo.Root(), ".git", "config")
	f, err := os.OpenFile(cfg, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("[include]\n\tpath = " + value + "\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return repo.Root(), included
}

// TestTheComparisonIsAskedOnceWhateverAnIncludePathHolds: with an include
// path that starts with a dash in the checkout's own settings, which every
// environment the drive's git runs in reads, the drive's check, its
// comparison of what it holds, is asked exactly once by a git from outside
// and never by a read of its own; with the included file changed since the
// drive started, the git is not run and the stop names the changed file, not
// a comparison that did not finish. The check refuses a second ask, so a
// comparison that asks itself ends at once. Watched failing on 5dd7deff:
// the include read asked the check from inside.
func TestTheComparisonIsAskedOnceWhateverAnIncludePathHolds(t *testing.T) {
	for _, value := range dashLedIncludes {
		t.Run(value, func(t *testing.T) {
			root, included := includedRepo(t, value)
			id, err := readRemoteIdentity(root)
			if err != nil {
				t.Fatal(err)
			}
			writeMode(t, included, "[core]\n\teditor = /bin/false\n", 0o644)
			var mu sync.Mutex
			calls := 0
			unset, err := gitutil.SetBeforeHookedGit(func([]string) error {
				mu.Lock()
				calls++
				n := calls
				mu.Unlock()
				if n > 1 {
					return errors.New("the check was asked again while it compared")
				}
				return heldBeforeGit(root, id)
			})
			if err != nil {
				t.Fatal(err)
			}
			defer unset()
			_, err = gitutil.LocalGit(root, "status")
			mu.Lock()
			n := calls
			mu.Unlock()
			if n != 1 {
				t.Fatalf("the check was asked %d times, want once from outside and never from inside: %v", n, err)
			}
			r := mustRefusal(t, err)
			if r.Stage != StageConfig || !strings.Contains(r.Reason, filepath.Base(value)) || strings.Contains(r.Reason, "did not finish") {
				t.Fatalf("the stop names the changed include file %q: %+v", value, r)
			}
		})
	}
}

// TestAnIncludeReadAsksNothingInAnyEnvironment: the read of an include path
// that starts with a dash, in each environment the drive's git runs in,
// asks the drive's check nothing and resolves the path git names. Watched
// failing on 5dd7deff: in abcd's isolated reads it asked.
func TestAnIncludeReadAsksNothingInAnyEnvironment(t *testing.T) {
	for _, value := range dashLedIncludes {
		root, included := includedRepo(t, value)
		for _, e := range driveEnvs() {
			t.Run(value+" in "+e.name, func(t *testing.T) {
				var mu sync.Mutex
				var asked [][]string
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
				r := scopedSetting{scope: "local", origin: "file:.git/config", key: "include.path", value: value}
				p, err := includePath(e, root, "salt", r, boundsNow(e))
				mu.Lock()
				defer mu.Unlock()
				if len(asked) != 0 {
					t.Fatalf("the include read asked the check: %q", asked)
				}
				if err != nil || p != filepath.Clean(included) {
					t.Fatalf("the include read resolves %q: %q, %v", included, p, err)
				}
			})
		}
	}
}

// TestADashLedIdIsSearchedAsAPattern: the search of the decisions folder for
// an id read from the run's state gives git the id as -e's own word, so an
// id that starts with a dash is the pattern, never an option, to git or to
// the check's reading of the command line: the search asks the drive's check
// nothing and answers. Watched failing on 5dd7deff: the id was read as
// --textconv, and the search asked.
func TestADashLedIdIsSearchedAsAPattern(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	at := baseTree{root: repo.Root(), sha: strings.TrimSpace(repo.Git("rev-parse", "HEAD"))}
	var mu sync.Mutex
	var asked [][]string
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
	if _, err := at.decisionsNaming(at.sha, at.blob, "--textconv"); err != nil {
		t.Fatalf("the search answers: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(asked) != 0 {
		t.Fatalf("the search asked the check: %q", asked)
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
