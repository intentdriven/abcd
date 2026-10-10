package lifeboat

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The default scan's ignore listing fails CLOSED and loud on the lazy-fetch
// floor (coordinator ruling on iss-2610091935324732). Below git 2.44 a partial
// clone refuses the `ls-files --others --exclude-standard` that decides which
// paths git ignores, because reading the ignore rules can fetch a
// skip-worktree .gitignore through the repository's configured transport. The
// walk used to read that refusal as "git could not answer" and narrow nothing,
// so a default scan silently read every ignored file. Now Probe and Plan
// refuse with the floor named, and a context that walks anyway narrows every
// path rather than widening. The wide scan, asked for, still runs.
func TestTheIgnoreScopeRefusesOnTheLazyFetchFloor(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(".gitignore", "secret-notes.md\n")
	r.Write("tracked.go", "package a // TODO: tracked\n")
	r.Write("secret-notes.md", "TODO: a private note\n")
	r.Commit("a repo with an ignored file")
	r.Git("config", "remote.origin.promisor", "true")
	restore := gitutil.SwapGitVersionForTest(func() (string, error) { return "git version 2.39.5 (Apple Git-154)", nil })
	defer restore()

	if _, err := Probe(r.Root()); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Errorf("Probe on a partial clone below the floor: want ErrLazyFetchFloor, got %v", err)
	} else if !strings.Contains(err.Error(), "2.44") {
		t.Errorf("the refusal does not name the 2.44 floor: %v", err)
	}
	if _, err := Plan(r.Root()); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Errorf("Plan on a partial clone below the floor: want ErrLazyFetchFloor, got %v", err)
	}

	ctx, err := newSourceContext(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	if got := strings.Join(mustWalk(t, ctx), "\n"); strings.Contains(got, "secret-notes.md") {
		t.Errorf("the ignore listing was refused and the walk widened to an ignored file: %q", got)
	}
	if err := ctx.ignoreScopeErr(); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Errorf("ignoreScopeErr: want ErrLazyFetchFloor, got %v", err)
	}

	// The wide scan needs no ignore listing, so the opt-in still probes.
	if _, err := Probe(r.Root(), IncludeIgnored()); err != nil {
		t.Errorf("Probe with IncludeIgnored refused: %v", err)
	}

	// At the floor the default scan runs and narrows as before.
	restore2 := gitutil.SwapGitVersionForTest(func() (string, error) { return "git version 2.44.0", nil })
	defer restore2()
	if _, err := Probe(r.Root()); err != nil {
		t.Fatalf("Probe at the floor: %v", err)
	}
}
