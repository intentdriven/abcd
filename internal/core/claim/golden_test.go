package claim

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// goldenAct is what one reservation act sent and left behind, byte for byte:
// the fetch and push it made, the commit object origin holds, and the stamp
// it wrote.
type goldenAct struct {
	fetch, push  []string
	object       string
	stampRel     string
	stampContent string
}

// goldenScenario reserves iss-5 from a fresh clone with everything an act's
// bytes depend on held fixed: the clock (t0), the git identity, the working
// copy's key and the version the Assisted-by label names.
func goldenScenario(t *testing.T) goldenAct {
	t.Helper()
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	key := filepath.Join(p.dir, filepath.FromSlash(copyKeyRel))
	if err := os.MkdirAll(filepath.Dir(key), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	log := traced(t)
	res, err := Reserve(p.dir, Request{Record: "iss-5", Footprint: []string{"internal/a.go", "docs/b.md"}},
		Options{Now: clockAt(t0), Home: p.home, Version: "1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	var g goldenAct
	for _, c := range log.cmds {
		switch c[0] {
		case "fetch":
			if g.fetch == nil {
				g.fetch = c
			}
		case "push":
			g.push = c
		}
	}
	g.object = runGitRaw(t, w.bare, "cat-file", "commit", res.Commit)
	g.stampRel = fetchedAtRel
	data, err := os.ReadFile(filepath.Join(p.dir, filepath.FromSlash(fetchedAtRel)))
	if err != nil {
		t.Fatal(err)
	}
	g.stampContent = string(data)
	return g
}

// runGitRaw is runGit without trimming its output, for a byte comparison.
func runGitRaw(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v in %s: %v", args, filepath.Base(dir), err)
	}
	return string(out)
}

// TestAReservationActIsByteForByteWhatItWas holds the reservation act to the
// bytes it made before the primitive was generalised (main at baf57cf5e): the
// literals below were captured by running goldenScenario against an export of
// that commit.
func TestAReservationActIsByteForByteWhatItWas(t *testing.T) {
	g := goldenScenario(t)
	const sha = "194d2abe630c556738c07b00670a1e6f55396fda"
	// The one deliberate change since baf57cf5e: every reservations fetch also
	// takes origin's release tags (releaseTagsRefspec, release.go).
	wantFetch := []string{"fetch", "--prune", "--no-tags", "origin",
		"+refs/heads/abcd-reserved/*:refs/remotes/origin/abcd-reserved/*",
		"+refs/tags/v*:refs/abcd/release-tags/v*",
		"+refs/heads/main:refs/remotes/origin/main"}
	if !slices.Equal(g.fetch, wantFetch) {
		t.Errorf("fetch = %q; want %q", g.fetch, wantFetch)
	}
	wantPush := []string{"push", "--porcelain", "origin", sha + ":refs/heads/abcd-reserved/iss-5"}
	if !slices.Equal(g.push, wantPush) {
		t.Errorf("push = %q; want %q", g.push, wantPush)
	}
	const wantObject = "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\n" +
		"author Pat Example <1234+octo@users.noreply.github.com> 1791036000 +0000\n" +
		"committer Pat Example <1234+octo@users.noreply.github.com> 1791036000 +0000\n" +
		"\n" +
		"abcd reservation: iss-5\n" +
		"\n" +
		"Abcd-Reservation: iss-5\n" +
		"Abcd-Holder: octo\n" +
		"Abcd-Copy: 0123456789abcdef0123456789abcdef\n" +
		"Abcd-Footprint: internal/a.go\n" +
		"Abcd-Footprint: docs/b.md\n" +
		"Abcd-Claimed-At: 2026-10-03T14:00:00Z\n" +
		"Abcd-Expires-At: 2026-10-03T16:00:00Z\n" +
		"Assisted-by: abcd:dev\n"
	if g.object != wantObject {
		t.Errorf("origin's commit object =\n%q\nwant\n%q", g.object, wantObject)
	}
	if g.stampRel != ".abcd/.work.local/reservation/fetched-at" || g.stampContent != "2026-10-03T14:00:00Z\n" {
		t.Errorf("stamp %s = %q; want .abcd/.work.local/reservation/fetched-at holding 2026-10-03T14:00:00Z", g.stampRel, g.stampContent)
	}
}

// TestTheStatusFetchIsTheCommandItWas holds `abcd status`'s fetch to its
// bytes: one fetch reading its per-branch refspecs on stdin, with the
// reservations' globbed refspec, their release tags and the default branch
// on the command line, and the refspecs it sends, one forced refspec per
// branch origin lists by a name the reader reads, sorted.
func TestTheStatusFetchIsTheCommandItWas(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	octo := w.clone("6+octo@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	octo.announceAs("merging", "#1")
	maya.announceAs("merging", "#2")
	log := traced(t)
	pat.status(nil)
	var fetches [][]string
	for _, c := range log.cmds {
		if c[0] == "fetch" {
			fetches = append(fetches, c)
		}
	}
	want := []string{"fetch", "--prune", "--no-tags", "--stdin", "origin",
		"+refs/heads/abcd-reserved/*:refs/remotes/origin/abcd-reserved/*",
		"+refs/tags/v*:refs/abcd/release-tags/v*",
		"+refs/heads/main:refs/remotes/origin/main"}
	if len(fetches) != 1 || !slices.Equal(fetches[0], want) {
		t.Fatalf("fetches = %q; want the one command %q", fetches, want)
	}
	names, _, _, err := listStatusBranches(pat.dir)
	if err != nil {
		t.Fatal(err)
	}
	status := statusBranches
	status.only = names
	stdin := []string{
		"+refs/heads/abcd-status/maya:refs/remotes/origin/abcd-status/maya",
		"+refs/heads/abcd-status/octo:refs/remotes/origin/abcd-status/octo",
	}
	if got := status.refspecs(); !slices.Equal(got, stdin) {
		t.Fatalf("the refspecs on stdin = %q; want %q", got, stdin)
	}
}
