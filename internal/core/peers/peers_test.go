package peers_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/gittest"
)

// fixture is one repository ("here" is its main checkout) with worktrees laid
// under a temp HOME, so a test can also prove the paths are the kind a surface
// redacts.
type fixture struct {
	t    *testing.T
	repo *gittest.Repo
	home string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	f := &fixture{t: t, repo: gittest.NewRepo(t), home: home}
	// The base ledger: one open issue and one planned intent here, committed, so
	// every peer cut from main starts with the same holdings.
	f.write(f.here(), ".abcd/work/issues/open/iss-1-the-first-finding.md", issue("iss-1", "The first finding"))
	f.write(f.here(), ".abcd/work/issues/open/iss-2-the-second-finding.md", issue("iss-2", "The second finding"))
	f.write(f.here(), ".abcd/development/intents/planned/itd-9-a-planned-intent.md", intentRec("itd-9", "A planned intent"))
	f.repo.Commit("base ledger")
	return f
}

func (f *fixture) here() string { return f.repo.Root() }

// git runs git in dir under the fixture's hermetic environment and identity.
func (f *fixture) git(dir string, args ...string) string {
	f.t.Helper()
	full := append([]string{"-C", dir,
		"-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = f.repo.Env()
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// worktree adds a linked worktree on a new branch cut from main, under HOME.
func (f *fixture) worktree(name, branch string) string {
	f.t.Helper()
	dir := filepath.Join(f.home, "wt", name)
	f.git(f.here(), "worktree", "add", "-q", "-b", branch, dir, "main")
	return dir
}

// resolve moves an issue file from open/ to resolved/ in dir, through git.
func (f *fixture) resolve(dir, name string) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd/work/issues/resolved"), 0o755); err != nil {
		f.t.Fatal(err)
	}
	f.git(dir, "mv", ".abcd/work/issues/open/"+name, ".abcd/work/issues/resolved/"+name)
}

func (f *fixture) write(dir, rel, content string) {
	f.t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) read() peers.Report {
	f.t.Helper()
	rep, err := peers.Read(f.here())
	if err != nil {
		f.t.Fatalf("peers.Read: %v", err)
	}
	return rep
}

func issue(id, title string) string {
	return "---\nschema_version: 1\nid: \"" + id + "\"\nslug: \"x\"\nseverity: \"minor\"\n---\n\n" + title + "\n\nmore body.\n"
}

func intentRec(id, title string) string {
	return "---\nid: " + id + "\nslug: x\n---\n\n# " + title + "\n\n## Press Release\n"
}

func samePath(t *testing.T, a, b string) bool {
	t.Helper()
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	return err1 == nil && err2 == nil && ra == rb
}

func findPeer(rep peers.Report, branch string) (peers.Peer, bool) {
	for _, p := range rep.Peers {
		if p.Branch == branch {
			return p, true
		}
	}
	return peers.Peer{}, false
}

func porcelain(f *fixture, dir string) string {
	return f.git(dir, "status", "--porcelain", "--untracked-files=all")
}

// Criterion 1: a record captured, uncommitted, under a sibling worktree's open/
// and absent here is one row under that peer, named with its title, and the read
// changes nothing in either tree.
func TestACaptureOnASiblingDiskIsARowUnderThatPeer(t *testing.T) {
	f := newFixture(t)
	a := f.worktree("a", "feat/a")
	f.write(a, ".abcd/work/issues/open/iss-100-a-peer-finding.md", issue("iss-100", "A finding the peer captured"))

	beforeHere, beforeA := porcelain(f, f.here()), porcelain(f, a)
	rep := f.read()
	if porcelain(f, f.here()) != beforeHere || porcelain(f, a) != beforeA {
		t.Fatal("the read changed a tree's status; the reader must write nothing")
	}

	p, ok := findPeer(rep, "feat/a")
	if !ok {
		t.Fatalf("the worktree holding an uncommitted capture is not a peer: %+v", rep)
	}
	if p.Source != peers.SourceWorktree || !samePath(t, p.Path, a) {
		t.Fatalf("peer = %+v, want a worktree peer at %s", p, a)
	}
	if p.NotRead != "" {
		t.Fatalf("peer not read: %s", p.NotRead)
	}
	want := peers.Row{ID: "iss-100", Kind: peers.KindOpenThere, Folder: "open", Title: "A finding the peer captured"}
	if len(p.Rows) != 1 || p.Rows[0] != want {
		t.Fatalf("rows = %+v, want [%+v]", p.Rows, want)
	}
	if rep.Live() != 1 || rep.IDCount() != 1 {
		t.Fatalf("live=%d ids=%d, want 1 and 1", rep.Live(), rep.IDCount())
	}
}

// Criterion 2: a record open here and resolved or won't-fixed in a peer is
// listed under that peer, naming the terminal folder.
func TestARecordOpenHereAndTerminalThereNamesTheFolder(t *testing.T) {
	f := newFixture(t)
	b := f.worktree("b", "feat/b")
	f.resolve(b, "iss-1-the-first-finding.md")
	f.git(b, "commit", "-q", "-m", "resolve iss-1")
	if err := os.MkdirAll(filepath.Join(b, ".abcd/work/issues/wontfix"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(b, ".abcd/work/issues/open/iss-2-the-second-finding.md"),
		filepath.Join(b, ".abcd/work/issues/wontfix/iss-2-the-second-finding.md")); err != nil {
		t.Fatal(err)
	}

	p, ok := findPeer(f.read(), "feat/b")
	if !ok {
		t.Fatal("feat/b is not a peer")
	}
	want := []peers.Row{
		{ID: "iss-1", Kind: peers.KindTerminalThere, Folder: "resolved", Title: "The first finding"},
		{ID: "iss-2", Kind: peers.KindTerminalThere, Folder: "wontfix", Title: "The second finding"},
	}
	if len(p.Rows) != 2 || p.Rows[0] != want[0] || p.Rows[1] != want[1] {
		t.Fatalf("rows = %+v, want %+v", p.Rows, want)
	}
}

// Criterion 3: a local branch no worktree has checked out is a peer, read from
// the common dir.
func TestABranchCheckedOutNowhereIsAPeer(t *testing.T) {
	f := newFixture(t)
	f.git(f.here(), "switch", "-q", "-c", "side")
	f.resolve(f.here(), "iss-1-the-first-finding.md")
	f.git(f.here(), "commit", "-q", "-m", "resolve iss-1 on a branch")
	f.git(f.here(), "switch", "-q", "main")

	p, ok := findPeer(f.read(), "side")
	if !ok {
		t.Fatal("a branch checked out nowhere is not a peer")
	}
	if p.Source != peers.SourceBranch || p.Path != "" {
		t.Fatalf("peer = %+v, want a branch peer with no path", p)
	}
	want := peers.Row{ID: "iss-1", Kind: peers.KindTerminalThere, Folder: "resolved", Title: "The first finding"}
	if len(p.Rows) != 1 || p.Rows[0] != want {
		t.Fatalf("rows = %+v, want [%+v]", p.Rows, want)
	}
}

// Criterion 4: an intent draft in a peer with no record of that id here is a
// draft row; a draft of an id this tree already planned is not.
func TestAPeerDraftAbsentHereIsADraftRow(t *testing.T) {
	f := newFixture(t)
	c := f.worktree("c", "feat/c")
	f.write(c, ".abcd/development/intents/drafts/itd-300-a-new-idea.md", intentRec("itd-300", "A new idea"))
	f.write(c, ".abcd/development/intents/drafts/itd-9-a-planned-intent.md", intentRec("itd-9", "A planned intent"))
	if err := os.Remove(filepath.Join(c, ".abcd/development/intents/planned/itd-9-a-planned-intent.md")); err != nil {
		t.Fatal(err)
	}

	p, ok := findPeer(f.read(), "feat/c")
	if !ok {
		t.Fatal("feat/c is not a peer")
	}
	want := peers.Row{ID: "itd-300", Kind: peers.KindDraftThere, Folder: "drafts", Title: "A new idea"}
	if len(p.Rows) != 1 || p.Rows[0] != want {
		t.Fatalf("rows = %+v, want [%+v]", p.Rows, want)
	}
}

// Criterion 6: a peer whose branch is merged into the default branch, or whose
// worktree directory is gone, contributes no rows and is counted as skipped.
func TestSpentPeersAreSkippedAndCounted(t *testing.T) {
	f := newFixture(t)
	f.worktree("merged", "feat/merged") // at main's tip, nothing uncommitted
	gone := f.worktree("gone", "feat/gone")
	f.write(gone, ".abcd/work/issues/open/iss-400-lost.md", issue("iss-400", "Lost"))
	f.git(gone, "add", "-A")
	f.git(gone, "commit", "-q", "-m", "a capture on a worktree about to vanish")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	f.git(f.here(), "branch", "old") // a branch at main's tip, checked out nowhere

	rep := f.read()
	// The gone worktree's own branch carries an unmerged commit, so it is read
	// through the branch source; nothing is read through a spent worktree.
	for _, p := range rep.Peers {
		if p.Source == peers.SourceWorktree || p.Branch != "feat/gone" {
			t.Fatalf("a spent peer was read: %+v", p)
		}
	}
	reasons := map[string]string{}
	for _, s := range rep.Skipped {
		reasons[s.Branch] = s.Reason
	}
	want := map[string]string{"feat/merged": peers.SkipMerged, "feat/gone": peers.SkipGone, "old": peers.SkipMerged}
	for br, r := range want {
		if reasons[br] != r {
			t.Errorf("skipped[%s] = %q, want %q (all: %+v)", br, reasons[br], r, rep.Skipped)
		}
	}
}

// A worktree whose directory is gone, or which git will not read, still leaves
// its branch in this repository's object store: an unmerged commit on it is
// read through the branch source rather than hidden with the worktree.
func TestAGoneOrRefusedWorktreesUnmergedBranchIsReadFromTheStore(t *testing.T) {
	f := newFixture(t)
	gone := f.worktree("gone", "feat/gone")
	f.write(gone, ".abcd/work/issues/open/iss-400-kept-in-the-store.md", issue("iss-400", "Kept in the store"))
	f.git(gone, "add", "-A")
	f.git(gone, "commit", "-q", "-m", "a capture on a worktree about to vanish")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	refused := f.worktree("refused", "feat/refused")
	f.write(refused, ".abcd/work/issues/open/iss-401-committed-before-the-refusal.md", issue("iss-401", "Committed before the refusal"))
	f.git(refused, "add", "-A")
	f.git(refused, "commit", "-q", "-m", "a capture before the worktree broke")
	if err := os.WriteFile(filepath.Join(refused, ".git"), []byte("gitdir: "+filepath.Join(f.home, "nowhere")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rep := f.read()
	for id, branch := range map[string]string{"iss-400": "feat/gone", "iss-401": "feat/refused"} {
		locs := rep.Locate(id)
		if len(locs) != 1 || locs[0].Source != peers.SourceBranch || locs[0].Branch != branch {
			t.Errorf("Locate(%s) = %+v, want %s read through the branch source", id, locs, branch)
		}
	}
}

// A worktree whose branch sits at the default branch's tip is merged by
// ancestry, yet an uncommitted capture in it is exactly what the reader exists
// to show: it stays live while its record folders are dirty.
func TestAMergedWorktreeWithAnUncommittedCaptureStaysLive(t *testing.T) {
	f := newFixture(t)
	fresh := f.worktree("fresh", "feat/fresh")
	f.write(fresh, ".abcd/work/issues/open/iss-101-fresh.md", issue("iss-101", "Fresh"))
	if _, ok := findPeer(f.read(), "feat/fresh"); !ok {
		t.Fatal("a worktree at the default tip holding an uncommitted capture was skipped")
	}
}

// Criterion 7: a peer whose ledger holds one id in two status folders is marked,
// not read, naming the id and the remedy; every other peer renders normally.
func TestASplitLedgerPeerIsMarkedAndOthersRender(t *testing.T) {
	f := newFixture(t)
	s := f.worktree("split", "feat/split")
	f.write(s, ".abcd/work/issues/resolved/iss-1-the-first-finding.md", issue("iss-1", "The first finding"))
	f.write(s, ".abcd/work/issues/open/iss-102-hidden.md", issue("iss-102", "Hidden"))
	a := f.worktree("a", "feat/a")
	f.write(a, ".abcd/work/issues/open/iss-100-seen.md", issue("iss-100", "Seen"))

	rep := f.read()
	sp, ok := findPeer(rep, "feat/split")
	if !ok {
		t.Fatal("the split peer is not named at all")
	}
	for _, want := range []string{"iss-1", "open", "resolved", "move or remove"} {
		if !strings.Contains(sp.NotRead, want) {
			t.Errorf("not_read = %q, want it to name %q", sp.NotRead, want)
		}
	}
	if len(sp.Rows) != 0 {
		t.Errorf("a split peer carried rows: %+v", sp.Rows)
	}
	ap, ok := findPeer(rep, "feat/a")
	if !ok || ap.NotRead != "" || len(ap.Rows) != 1 || ap.Rows[0].ID != "iss-100" {
		t.Fatalf("the healthy peer did not render normally: %+v", ap)
	}
}

// Criterion 8: a directory git lists whose own common dir is not this
// checkout's, or for which git refuses to answer, is named and not read.
func TestAForeignOrRefusedWorktreeIsNamedNotRead(t *testing.T) {
	f := newFixture(t)
	foreign := f.worktree("foreign", "feat/foreign")
	if err := os.Remove(filepath.Join(foreign, ".git")); err != nil {
		t.Fatal(err)
	}
	f.git(foreign, "init", "-q")
	f.write(foreign, ".abcd/work/issues/open/iss-103-other-repo.md", issue("iss-103", "Other repo"))

	refused := f.worktree("refused", "feat/refused")
	if err := os.WriteFile(filepath.Join(refused, ".git"), []byte("gitdir: "+filepath.Join(f.home, "nowhere")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.write(refused, ".abcd/work/issues/open/iss-104-refused.md", issue("iss-104", "Refused"))

	rep := f.read()
	fp, ok := findPeer(rep, "feat/foreign")
	if !ok || !strings.Contains(fp.NotRead, "common dir") || len(fp.Rows) != 0 {
		t.Fatalf("foreign worktree = %+v (ok=%v), want named, not read, citing its common dir", fp, ok)
	}
	rp, ok := findPeer(rep, "feat/refused")
	if !ok || !strings.Contains(rp.NotRead, "git") || len(rp.Rows) != 0 {
		t.Fatalf("refused worktree = %+v (ok=%v), want named, not read, citing git", rp, ok)
	}
}

// Criterion 9: a checkout with no peers reports none.
func TestACheckoutWithNoPeersReportsNone(t *testing.T) {
	f := newFixture(t)
	rep := f.read()
	if len(rep.Peers) != 0 || len(rep.Skipped) != 0 || rep.IDCount() != 0 {
		t.Fatalf("report = %+v, want no peers", rep)
	}
	if len(rep.Sources) != 2 {
		t.Fatalf("sources = %v, want the two local sources", rep.Sources)
	}
}

// Criterion 12: an unreadable or malformed record file lists its id without a
// title, and the read succeeds.
func TestAnUnreadableOrMalformedRecordListsTheIDAlone(t *testing.T) {
	f := newFixture(t)
	d := f.worktree("d", "feat/d")
	f.write(d, ".abcd/work/issues/open/iss-120-malformed.md", "no frontmatter at all\n")
	f.write(d, ".abcd/work/issues/open/iss-121-unreadable.md", issue("iss-121", "Unreadable"))
	if err := os.Chmod(filepath.Join(d, ".abcd/work/issues/open/iss-121-unreadable.md"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(d, ".abcd/work/issues/open/iss-121-unreadable.md"), 0o644) })
	if os.Geteuid() == 0 {
		t.Log("running as root: the mode-0 file is readable, so only the malformed case is exercised")
	}

	p, ok := findPeer(f.read(), "feat/d")
	if !ok {
		t.Fatal("feat/d is not a peer")
	}
	got := map[string]string{}
	for _, r := range p.Rows {
		got[r.ID] = r.Title
	}
	if title, ok := got["iss-120"]; !ok || title != "" {
		t.Errorf("malformed iss-120: present=%v title=%q, want present with no title", ok, title)
	}
	if _, ok := got["iss-121"]; !ok {
		t.Errorf("unreadable iss-121 is missing: %+v", p.Rows)
	}
	if os.Geteuid() != 0 && got["iss-121"] != "" {
		t.Errorf("unreadable iss-121 carries a title %q", got["iss-121"])
	}
}

// Locate answers the not-found paths: which live peer holds an id, in which
// folder, through either source, across every intent bucket.
func TestLocateNamesThePeerThatHoldsAnID(t *testing.T) {
	f := newFixture(t)
	a := f.worktree("a", "feat/a")
	f.write(a, ".abcd/work/issues/open/iss-100-a-peer-finding.md", issue("iss-100", "A peer finding"))
	f.git(f.here(), "switch", "-q", "-c", "side")
	f.write(f.here(), ".abcd/development/intents/shipped/itd-77-shipped-elsewhere.md", intentRec("itd-77", "Shipped elsewhere"))
	f.git(f.here(), "add", "-A")
	f.git(f.here(), "commit", "-q", "-m", "ship itd-77 on a branch")
	f.git(f.here(), "switch", "-q", "main")

	rep := f.read()
	locs := rep.Locate("iss-100")
	if len(locs) != 1 || locs[0].Branch != "feat/a" || locs[0].Folder != "open" || !samePath(t, locs[0].Path, a) {
		t.Fatalf("Locate(iss-100) = %+v, want feat/a's open/", locs)
	}
	locs = rep.Locate("itd-77")
	if len(locs) != 1 || locs[0].Branch != "side" || locs[0].Folder != "shipped" || locs[0].Source != peers.SourceBranch {
		t.Fatalf("Locate(itd-77) = %+v, want side's shipped/", locs)
	}
	if !rep.HeldHere("iss-1") || rep.HeldHere("iss-100") {
		t.Fatal("HeldHere must answer from this tree's own folders")
	}
}

// Scan is Read with no record file opened: the same rows and locations, no
// titles, for the callers that only count or locate.
func TestScanIsReadWithoutTitles(t *testing.T) {
	f := newFixture(t)
	a := f.worktree("a", "feat/a")
	f.write(a, ".abcd/work/issues/open/iss-100-a-peer-finding.md", issue("iss-100", "A peer finding"))
	rep, err := peers.Scan(f.here())
	if err != nil {
		t.Fatal(err)
	}
	p, ok := findPeer(rep, "feat/a")
	want := peers.Row{ID: "iss-100", Kind: peers.KindOpenThere, Folder: "open"}
	if !ok || len(p.Rows) != 1 || p.Rows[0] != want {
		t.Fatalf("Scan rows = %+v, want [%+v]", p.Rows, want)
	}
	if locs := rep.Locate("iss-100"); len(locs) != 1 || rep.IDCount() != 1 {
		t.Fatalf("Scan locations = %+v, ids = %d", locs, rep.IDCount())
	}
}

// oldGit puts a git on PATH that behaves as a git older than 2.36 does on the
// two calls the reader makes of a newer one: `worktree list --porcelain -z` is
// refused as an unknown switch (-z arrived in 2.36; Ubuntu 22.04 ships 2.34),
// and `rev-parse --path-format=absolute` (2.31) is echoed to stdout, exit 0,
// with the path then answered in its relative form.
func oldGit(t *testing.T) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nREAL_GIT=" + real + `
case "$*" in
*"worktree list --porcelain -z"*)
	echo "error: unknown switch 'z'" >&2
	exit 129;;
*rev-parse*--path-format=*)
	echo "--path-format=absolute"
	for a do
		shift
		case "$a" in --path-format=*) ;; *) set -- "$@" "$a";; esac
	done;;
esac
exec "$REAL_GIT" "$@"
`
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// A git older than 2.36 has no -z on `worktree list`, and one older than 2.31
// no --path-format: the reader falls back to the forms every supported git
// answers, so the peer is still found and read rather than the whole listing
// failing (and the board line and the not-found hints with it).
func TestAnOlderGitStillListsAndReadsThePeers(t *testing.T) {
	f := newFixture(t)
	a := f.worktree("a", "feat/a")
	f.write(a, ".abcd/work/issues/open/iss-100-a-peer-finding.md", issue("iss-100", "A finding the peer captured"))
	oldGit(t)

	rep := f.read()
	p, ok := findPeer(rep, "feat/a")
	if !ok {
		t.Fatalf("an older git lost the worktree peer: %+v", rep)
	}
	if p.NotRead != "" || !samePath(t, p.Path, a) {
		t.Fatalf("peer = %+v, want feat/a read at %s", p, a)
	}
	if len(p.Rows) != 1 || p.Rows[0].ID != "iss-100" {
		t.Fatalf("rows = %+v, want iss-100", p.Rows)
	}
}
