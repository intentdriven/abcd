package claim

import (
	"strings"
	"testing"
	"time"
)

// TestADirectoryFootprintIsReservedAndNamedWithItsFileCount
// (iss-2610230016544142): a reservation whose footprint names a directory git
// tracks on origin's default branch is still made, and the result names the
// directory and how many tracked files it covers, so the person is told to
// name the files the work changes instead.
func TestADirectoryFootprintIsReservedAndNamedWithItsFileCount(t *testing.T) {
	w := newWorld(t)
	w.seed.Write("internal/pkg/a.go", "package pkg\n")
	w.seed.Write("internal/pkg/b.go", "package pkg\n")
	w.seed.Write("internal/pkg/sub/c.go", "package sub\n")
	w.seed.Commit("the package")
	w.seed.Git("push", "-q", "origin", "main")
	p := w.clone("1234+octo@users.noreply.github.com", "")

	res, err := p.reserve("iss-5", "internal/pkg", "README.md")
	if err != nil || res.Act != ActReserved {
		t.Fatalf("a directory footprint is reserved, never refused: %+v %v", res, err)
	}
	if len(res.Directories) != 1 {
		t.Fatalf("Directories = %+v; want the one directory", res.Directories)
	}
	d := res.Directories[0]
	if d.Path != "internal/pkg" || d.Files != 3 || d.On != "origin/main" {
		t.Fatalf("Directories[0] = %+v; want internal/pkg covering 3 tracked files on origin/main", d)
	}
	warn := d.Warning()
	for _, want := range []string{`"internal/pkg"`, "3 tracked files", "origin/main", "name the files the work changes"} {
		if !strings.Contains(warn, want) {
			t.Fatalf("the warning %q does not say %q", warn, want)
		}
	}
	// A renewal keeps the tip's footprint, directory and all, and names
	// nothing: the warning was given when the footprint was named.
	w.set(t0.Add(30 * time.Minute))
	res, err = p.do(Request{Record: "iss-5", Renew: true})
	if err != nil || res.Act != ActRenewed || strings.Join(res.Footprint, ",") != "internal/pkg,README.md" {
		t.Fatalf("renewal: %+v %v", res, err)
	}
	if len(res.Directories) != 0 {
		t.Fatalf("a renewal of a directory footprint warns again: %+v", res.Directories)
	}
}

// TestAFileFootprintDrawsNoDirectoryWarning: a footprint naming only files,
// tracked or new, names no directory.
func TestAFileFootprintDrawsNoDirectoryWarning(t *testing.T) {
	w := newWorld(t)
	w.seed.Write("internal/pkg/a.go", "package pkg\n")
	w.seed.Commit("the package")
	w.seed.Git("push", "-q", "origin", "main")
	p := w.clone("1234+octo@users.noreply.github.com", "")

	res, err := p.reserve("iss-5", "internal/pkg/a.go", "internal/pkg/new.go", "README.md")
	if err != nil || res.Act != ActReserved {
		t.Fatalf("reserve: %+v %v", res, err)
	}
	if len(res.Directories) != 0 {
		t.Fatalf("a file-only footprint names no directory: %+v", res.Directories)
	}
}

// TestADirectoryWarningNamesTheBranchTerminalSafe: git allows a bidi or
// zero-width rune in a branch name, so origin's default branch, which the
// warning names, is sanitised before any surface shows it.
func TestADirectoryWarningNamesTheBranchTerminalSafe(t *testing.T) {
	w := newWorld(t)
	const hostile = "ma‮in"
	w.seed.Git("checkout", "-q", "-b", hostile)
	w.seed.Write("internal/pkg/a.go", "package pkg\n")
	w.seed.Commit("the package")
	w.seed.Git("push", "-q", "origin", hostile)
	runGit(t, w.bare, "symbolic-ref", "HEAD", "refs/heads/"+hostile)
	p := w.clone("1234+octo@users.noreply.github.com", "")

	res, err := p.reserve("iss-5", "internal/pkg")
	if err != nil || len(res.Directories) != 1 {
		t.Fatalf("reserve: %+v %v", res, err)
	}
	d := res.Directories[0]
	if d.Files != 1 || !strings.HasPrefix(d.On, "origin/") {
		t.Fatalf("the directory is read from origin's default branch: %+v", d)
	}
	for _, s := range []string{d.On, d.Warning()} {
		if strings.ContainsRune(s, '‮') {
			t.Fatalf("%q carries the branch name's bidi mark unsanitised", s)
		}
	}
}
