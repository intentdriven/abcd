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

// TestIgnoreReadsRefuseAPartialCloneBelowTheLazyFetchFloor is
// iss-2610091935324732. Below 2.44 the floor exempted check-ignore and
// ls-files as reading no objects, but git reads a skip-worktree .gitignore (or
// .gitattributes) missing from disk out of the object store, so in a partial
// clone those lookups lazy-fetch through the promisor remote's transport. Now
// check-ignore counts as reading objects unless it is --no-index, and ls-files
// does for any flag that consults the exclude files or blob content; the plain
// index listings still run, and a repository with no promisor remote runs
// every form as before.
func TestIgnoreReadsRefuseAPartialCloneBelowTheLazyFetchFloor(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(".gitignore", "secret.txt\n")
	r.Write("kept.txt", "kept\n")
	r.Commit("c0")
	r.Write("secret.txt", "s\n")
	root := r.Root()

	gitutil.SetGitVersionSource(t, func() (string, error) { return "git version 2.39.5 (Apple Git-154)", nil })

	reading := [][]string{
		{"check-ignore", "-z", "-v", "--stdin"},
		{"check-ignore", "secret.txt"},
		{"-c", "core.excludesFile=", "check-ignore", "-v", "secret.txt"},
		{"ls-files", "-o"},
		{"ls-files", "--others"},
		{"ls-files", "-o", "-i", "--exclude-standard"},
		{"ls-files", "-oi", "--exclude-standard"},
		{"ls-files", "--others", "--ignored", "--exclude-standard", "--directory"},
		{"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "x"},
		{"ls-files", "--exclude-standard"},
		{"ls-files", "--exclude-per-directory=.gitignore"},
		{"ls-files", "--exclude-from=.gitignore"},
		{"ls-files", "-X", ".gitignore"},
		{"ls-files", "-x", "*.txt"},
		{"ls-files", "--exclude=*.txt"},
		{"ls-files", "--with-tree=HEAD"},
		{"ls-files", "--with-tree", "HEAD"},
		{"ls-files", "--eol"},
		{"ls-files", "--format=%(eolinfo:index) %(path)"},
		{"ls-files", "-m"},
		{"ls-files", "--modified"},
		{"ls-files", "--", ":(attr:foo)"},
		{"config", "--blob=HEAD:.gitignore", "--list"},
		{"config", "--blob", "HEAD:.gitignore", "--list"},
		// Every listing too: git expands a sparse index the index file itself
		// marks, whatever the config says, by reading tree objects.
		{"ls-files"},
		{"ls-files", "-z"},
		{"ls-files", "--cached", "-z"},
		{"ls-files", "--stage", "-z", "--", ":(glob)**/.gitattributes"},
		{"ls-files", "--", "kept.txt"},
	}
	running := [][]string{
		{"check-ignore", "--no-index", "-v", "secret.txt"},
		{"config", "--get", "remote.origin.promisor"},
	}
	// Only the refusal matters here: a form git itself rejects or that exits 1
	// (check-ignore with no match) still ran.
	run := func(args []string) error {
		t.Helper()
		_, err := gitutil.Run(root, args...)
		return err
	}

	// No promisor declared: every form runs on old git.
	for _, args := range append(append([][]string{}, reading...), running...) {
		if err := run(args); errors.Is(err, gitutil.ErrLazyFetchFloor) {
			t.Errorf("no promisor remote, git %v refused: %v", args, err)
		}
	}
	if !gitutil.IsIgnored(root, "secret.txt") {
		t.Fatal("no promisor remote: IsIgnored no longer answers on old git")
	}
	if got, err := gitutil.IgnoredUnder(root, "."); err != nil || len(got) == 0 {
		t.Fatalf("no promisor remote: IgnoredUnder no longer answers on old git: %q, %v", got, err)
	}

	r.Git("config", "remote.origin.promisor", "true")
	for _, args := range reading {
		if err := run(args); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
			t.Errorf("partial clone on git 2.39: git %v was not refused (err %v)", args, err)
		}
	}
	for _, args := range running {
		if err := run(args); errors.Is(err, gitutil.ErrLazyFetchFloor) {
			t.Errorf("partial clone on git 2.39: git %v reads no object, yet refused: %v", args, err)
		}
	}
	// The helpers go through the same guard: git never runs, so nothing is
	// reported ignored.
	if gitutil.IsIgnored(root, "secret.txt") {
		t.Error("partial clone on git 2.39: CheckIgnored ran git check-ignore")
	}
	// IgnoredUnder returns the refusal, so a caller pruning by it can say so.
	if got, err := gitutil.IgnoredUnder(root, "."); len(got) != 0 || !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Errorf("partial clone on git 2.39: IgnoredUnder = %q, %v; want nothing and ErrLazyFetchFloor", got, err)
	}
}

// TestSparseListingsRefuseAPartialCloneBelowTheLazyFetchFloor is the
// coordinator's ruling on iss-2610091935324732: with a sparse checkout or a
// sparse index, ls-files can expand the index by reading tree objects, so
// below 2.44 a partial clone whose config enables core.sparseCheckout or
// index.sparse (read through the same isolated config view, or set on the
// command line) refuses EVERY ls-files, the plain index listings included. A
// partial clone without those settings keeps the index-only allowance, a
// repository with no promisor remote runs as before, and at the floor the
// sparse partial clone lists again.
func TestSparseListingsRefuseAPartialCloneBelowTheLazyFetchFloor(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("kept.txt", "kept\n")
	r.Commit("c0")
	root := r.Root()

	gitutil.SetGitVersionSource(t, func() (string, error) { return "git version 2.39.5 (Apple Git-154)", nil })

	listings := [][]string{
		{"ls-files"},
		{"ls-files", "-z"},
		{"ls-files", "--cached", "-z"},
		{"ls-files", "--stage", "-z", "--", ":(glob)**/.gitattributes"},
		{"ls-files", "--sparse"},
		{"ls-files", "--", "kept.txt"},
	}
	others := [][]string{
		{"rev-parse", "--show-toplevel"},
		{"config", "--get", "core.sparsecheckout"},
		{"check-ignore", "--no-index", "kept.txt"},
	}
	run := func(args []string) error {
		t.Helper()
		_, err := gitutil.Run(root, args...)
		return err
	}
	allRun := func(label string, set [][]string) {
		t.Helper()
		for _, args := range set {
			if err := run(args); errors.Is(err, gitutil.ErrLazyFetchFloor) {
				t.Errorf("%s: git %v refused: %v", label, args, err)
			}
		}
	}
	allRefused := func(label string, set [][]string) {
		t.Helper()
		for _, args := range set {
			if err := run(args); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
				t.Errorf("%s: git %v was not refused (err %v)", label, args, err)
			}
		}
	}

	// Sparse but no promisor remote: nothing can lazy-fetch.
	r.Git("config", "core.sparseCheckout", "true")
	r.Git("config", "index.sparse", "true")
	allRun("sparse, no promisor remote", listings)

	// A partial clone that is not sparse by its config is refused too: the
	// on-disk index, not the config, decides whether git expands it.
	r.Git("config", "--unset", "core.sparseCheckout")
	r.Git("config", "--unset", "index.sparse")
	r.Git("config", "remote.origin.promisor", "true")
	allRefused("partial clone, not sparse", listings)
	r.Git("config", "core.sparseCheckout", "false")
	r.Git("config", "index.sparse", "0")
	allRefused("partial clone, sparse settings false", listings)

	// core.sparseCheckout alone.
	r.Git("config", "core.sparseCheckout", "true")
	allRefused("partial clone, core.sparseCheckout", listings)
	allRun("partial clone, core.sparseCheckout, not ls-files", others)
	r.Git("config", "core.sparseCheckout", "false")

	// index.sparse alone, as a bare (true) key in a different case.
	r.Git("config", "index.Sparse", "yes")
	allRefused("partial clone, index.sparse", listings)
	r.Git("config", "index.sparse", "false")

	// The same settings on the command line count too.
	for _, kv := range []string{"core.sparseCheckout=true", "index.sparse", "INDEX.SPARSE=on"} {
		if err := run([]string{"-c", kv, "ls-files", "-z"}); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
			t.Errorf("partial clone, -c %s ls-files: not refused (err %v)", kv, err)
		}
	}
	// A -c turning the setting off cannot vouch for the index on disk either.
	if err := run([]string{"-c", "index.sparse=false", "ls-files", "-z"}); !errors.Is(err, gitutil.ErrLazyFetchFloor) {
		t.Errorf("partial clone, -c index.sparse=false ls-files: not refused (err %v)", err)
	}

	// At the floor the sparse partial clone lists.
	r.Git("config", "core.sparseCheckout", "true")
	gitutil.SetGitVersionSource(t, func() (string, error) { return "git version 2.44.0", nil })
	allRun("sparse partial clone on git 2.44", listings)
}
