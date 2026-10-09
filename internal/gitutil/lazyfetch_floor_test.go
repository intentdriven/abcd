package gitutil_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestIsolatedObjectReadRefusesAPartialCloneBelowTheLazyFetchFloor is the
// second half of iss-2610090821527948. GIT_NO_LAZY_FETCH, which the isolated
// environment sets so a missing object in a partial clone is an error rather
// than a fetch through the repository's configured transport, is honoured from
// git 2.44; an older git (Apple's Command Line Tools ship 2.39) ignores it. So
// below the floor an isolated command that reads objects refuses a repository
// that declares a promisor remote (extensions.partialClone, or any
// remote.<name>.promisor true), naming the floor; a repository that declares
// none is unaffected, root discovery still answers, and at the floor the
// promisor repository reads as before. The version is injected, so both sides
// are tested whatever git is installed.
func TestIsolatedObjectReadRefusesAPartialCloneBelowTheLazyFetchFloor(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("kept.txt", "kept\n")
	r.Commit("c0")
	r.Git("tag", "v1.0.0")
	root := r.Root()

	read := func() error {
		t.Helper()
		_, err := gitutil.Run(root, "cat-file", "blob", "HEAD:kept.txt")
		return err
	}
	wantRefused := func(label string, err error) {
		t.Helper()
		if !errors.Is(err, gitutil.ErrLazyFetchFloor) {
			t.Fatalf("%s: want ErrLazyFetchFloor, got %v", label, err)
		}
		if !strings.Contains(err.Error(), "2.44") {
			t.Errorf("%s: the refusal does not name the 2.44 floor: %v", label, err)
		}
	}

	gitutil.SetGitVersionSource(t, func() (string, error) { return "git version 2.39.5 (Apple Git-154)", nil })

	// No promisor declared: old git reads as before.
	if err := read(); err != nil {
		t.Fatalf("a repository with no promisor remote no longer reads on old git: %v", err)
	}

	r.Git("config", "remote.origin.promisor", "false")
	if err := read(); err != nil {
		t.Fatalf("remote.origin.promisor=false is not a promisor remote, yet the read refused: %v", err)
	}

	// remote.<name>.promisor=true alone (subsection case kept as written).
	r.Git("config", "remote.Upstream.promisor", "true")
	wantRefused("remote.Upstream.promisor=true, cat-file", read())
	_, err := gitutil.RunLimited(root, 1<<20, "show", "HEAD:kept.txt")
	wantRefused("RunLimited show", err)
	_, err = gitutil.RunCapped(root, 1<<20, "log", "--format=%H")
	wantRefused("RunCapped log", err)
	_, err = gitutil.Run(root, "tag", "--list", "v*")
	wantRefused("tag --list", err)
	_, err = gitutil.ArchiveTree(root, "HEAD")
	wantRefused("ArchiveTree", err)
	if _, err := gitutil.IsAncestor(root, "HEAD", "HEAD"); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Fatalf("IsAncestor: want ErrLazyFetchFloor, got %v", err)
	}
	// Root discovery reads no object and still answers.
	if _, err := gitutil.Toplevel(root); err != nil {
		t.Fatalf("Toplevel refused on a promisor repository below the floor: %v", err)
	}
	r.Git("config", "--unset", "remote.Upstream.promisor")
	if err := read(); err != nil {
		t.Fatalf("with the promisor key removed the read still refuses: %v", err)
	}

	// extensions.partialClone alone.
	r.Git("config", "extensions.partialClone", "origin")
	wantRefused("extensions.partialClone", read())

	// An unreadable version is treated as below the floor.
	gitutil.SetGitVersionSource(t, func() (string, error) { return "", errors.New("no git version") })
	wantRefused("unreadable version", read())

	// At the floor the same promisor repository reads.
	gitutil.SetGitVersionSource(t, func() (string, error) { return "git version 2.44.0", nil })
	if err := read(); err != nil {
		t.Fatalf("git 2.44 honours GIT_NO_LAZY_FETCH, yet the read refused: %v", err)
	}
	gitutil.SetGitVersionSource(t, func() (string, error) { return "git version 3.0.1.windows.1", nil })
	if err := read(); err != nil {
		t.Fatalf("git 3.0 is past the floor, yet the read refused: %v", err)
	}
}
