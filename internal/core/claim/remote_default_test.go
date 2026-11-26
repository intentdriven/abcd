package claim

import (
	"fmt"
	"strings"
	"testing"
)

// headNamedBranches gives the bare repository at bare branches named
// HEAD/<n>/<padding>/HEAD at its main, until `git ls-remote --symref origin
// HEAD`, which lists every ref whose last component is HEAD beside HEAD
// itself, would write more than want bytes. The refs are made in one
// update-ref transaction rather than pushed one by one.
func headNamedBranches(t *testing.T, bare string, want int) {
	t.Helper()
	sha := runGit(t, bare, "rev-parse", "refs/heads/main")
	pad := strings.Repeat("x", 200)
	var in strings.Builder
	for n, listed := 0, 0; listed <= want; n++ {
		ref := fmt.Sprintf("refs/heads/HEAD/%d/%s/HEAD", n, pad)
		fmt.Fprintf(&in, "create %s %s\n", ref, sha)
		listed += len(sha) + 1 + len(ref) + 1
	}
	runGitStdin(t, bare, in.String(), "update-ref", "--stdin")
	runGit(t, bare, "pack-refs", "--all")
}

// TestTheDefaultBranchIsReadFromTheHEADLinesAlone: an origin holding enough
// branches whose names end in /HEAD that `git ls-remote --symref origin HEAD`
// lists more than a networked read keeps whole still names main as its
// default, at main's tip, both to an act asking for the ceiling (originHead)
// and to a clone that names no default branch of its own (remoteDefault):
// only the two lines whose ref is HEAD itself are read.
func TestTheDefaultBranchIsReadFromTheHEADLinesAlone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	const bound = 1 << 20
	headNamedBranches(t, w.bare, bound)
	if listed := runGit(t, p.dir, "ls-remote", "--symref", Remote, "HEAD"); len(listed) <= bound {
		t.Fatalf("origin's listing of HEAD runs to %d bytes; want more than %d", len(listed), bound)
	}
	main := runGit(t, w.bare, "rev-parse", "refs/heads/main")
	branch, tip, none, err := originHead(p.dir)
	if err != nil || branch != "main" || tip != main || none != "" {
		t.Fatalf("originHead = %q, %q, %q, %v; want main at %s", branch, tip, none, err, main)
	}
	if ref, err := remoteDefault(p.dir); err != nil || ref != "refs/remotes/"+Remote+"/main" {
		t.Fatalf("remoteDefault = %q, %v; want refs/remotes/%s/main", ref, err, Remote)
	}
}
