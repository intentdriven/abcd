package claim

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/shellquote"
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

// goldenHolderID is the holder id the golden scenarios keep.
const goldenHolderID = "h_00112233445566778899aabbccddeeff"

// goldenScenario reserves iss-5 from a fresh clone with everything an act's
// bytes depend on held fixed: the clock (t0), the git identity, the working
// copy's key, the holder id and the version the Assisted-by label names.
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
	keepHolderID(t, p.home, goldenHolderID)
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
	const sha = "855f075567e91378eca82e78a7fc1073437fbaa3"
	// The deliberate changes since baf57cf5e: every reservations fetch also
	// takes origin's release tags (releaseTagsRefspec, release.go), and the
	// holder is named by the holder id, not the login (iss-2610290809526033),
	// which moves the commit's name.
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
	wantObject := "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\n" +
		"author Pat Example <1234+octo@users.noreply.github.com> 1791036000 +0000\n" +
		"committer Pat Example <1234+octo@users.noreply.github.com> 1791036000 +0000\n" +
		"\n" +
		"abcd reservation: iss-5\n" +
		"\n" +
		"Abcd-Reservation: iss-5\n" +
		"Abcd-Holder: " + goldenHolderID + "\n" +
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

// stdinOfFetches puts a stand-in git on PATH that passes every command to the
// real git, and appends to the file it returns every byte a `git fetch
// --stdin` reads on its standard input, as git reads it.
func stdinOfFetches(t *testing.T) string {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "stdin.log")
	script := "#!/bin/sh\nfetch=0 stdin=0\nfor a in \"$@\"; do\n\tcase \"$a\" in fetch) fetch=1 ;; --stdin) stdin=1 ;; esac\ndone\n" +
		"if [ $fetch = 1 ] && [ $stdin = 1 ]; then\n\ttee -a " + shellquote.Single(logPath) + " | " + shellquote.Single(real) + " \"$@\"\n\texit $?\nfi\n" +
		"exec " + shellquote.Single(real) + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// TestTheStatusFetchIsTheCommandItWas holds `abcd status`'s fetch to its
// bytes: one fetch reading its per-branch refspecs on stdin, with the
// reservations' globbed refspec, their release tags and the default branch
// on the command line, and on its standard input, as git read it, one forced
// refspec per branch origin lists by a name the reader reads, sorted.
func TestTheStatusFetchIsTheCommandItWas(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	octo := w.clone("6+octo@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	keepHolderID(t, maya.home, "h_"+strings.Repeat("a", 32))
	keepHolderID(t, octo.home, "h_"+strings.Repeat("b", 32))
	// A status branch named by a login, as one made before the holder id is,
	// is listed and fetched beside them.
	w.pushStatus("quinn", announcementMsg("asking", "#3", "quinn", t0, time.Hour), "")
	octo.announceAs("merging", "#1")
	maya.announceAs("merging", "#2")
	stdinLog := stdinOfFetches(t)
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
	got, err := os.ReadFile(stdinLog)
	if err != nil {
		t.Fatalf("no fetch read its stdin: %v", err)
	}
	a, b := "h_"+strings.Repeat("a", 32), "h_"+strings.Repeat("b", 32)
	stdin := "+refs/heads/abcd-status/" + a + ":refs/remotes/origin/abcd-status/" + a + "\n" +
		"+refs/heads/abcd-status/" + b + ":refs/remotes/origin/abcd-status/" + b + "\n" +
		"+refs/heads/abcd-status/quinn:refs/remotes/origin/abcd-status/quinn\n"
	if string(got) != stdin {
		t.Fatalf("the fetch read on stdin %q; want %q", got, stdin)
	}
}
