package spotcheck

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// clone is a checkout with one commit on main and origin at bare.
func clone(t *testing.T, bare string) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("the base")
	r.Git("remote", "add", "origin", bare)
	return r
}

// twoClones are two checkouts of one bare remote.
func twoClones(t *testing.T) (a, b *gittest.Repo, bare string) {
	t.Helper()
	bare = filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	return clone(t, bare), clone(t, bare), bare
}

var at0 = time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)

func handBack(issue string, at time.Time) HandBack {
	return HandBack{At: at, Issue: issue, From: "lane", Kind: "stuck", Route: "home", Reason: "the lane kept refusing",
		PR: 12, RunID: "run-2611130703410770", DrainRule: "adr-2609291342092738", Holder: "h_0123456789abcdef0123456789abcdef"}
}

func readLocal(t *testing.T, root string) Records {
	t.Helper()
	rec, ok, err := Read(context.Background(), root, LocalRef)
	if err != nil || !ok {
		t.Fatalf("Read(local) = ok %v, %v", ok, err)
	}
	return rec
}

func remoteFiles(t *testing.T, bare string) []string {
	t.Helper()
	out := gitIn(t, bare, "ls-tree", "-r", "--name-only", BranchRef)
	return strings.Fields(out)
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	r := gittest.OpenRepo(t, dir)
	return r.Git(args...)
}

// A record is one new file on the local copy, written without touching HEAD,
// the index or the working tree, and without reaching origin.
func TestAWriteIsOneFileOnTheLocalCopy(t *testing.T) {
	a, _, bare := twoClones(t)
	head := a.Git("rev-parse", "HEAD")
	wrote, err := WriteHandBack(context.Background(), a.Root(), handBack("iss-1", at0))
	if err != nil || !wrote {
		t.Fatalf("WriteHandBack = %v, %v", wrote, err)
	}
	if got := a.Git("rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD moved: %s", got)
	}
	if st := a.Git("status", "--porcelain"); st != "" {
		t.Fatalf("the working tree changed: %q", st)
	}
	files := strings.Fields(a.Git("ls-tree", "-r", "--name-only", LocalRef))
	if len(files) != 1 || files[0] != "handbacks/20261113T070000Z-iss-1.json" {
		t.Fatalf("local copy holds %v", files)
	}
	if out := gitIn(t, bare, "for-each-ref", BranchRef); out != "" {
		t.Fatalf("a write reached origin: %s", out)
	}
	rec := readLocal(t, a.Root())
	if len(rec.HandBacks) != 1 || rec.HandBacks[0].PR != 12 || rec.HandBacks[0].RunID != "run-2611130703410770" || len(rec.Unreadable) != 0 {
		t.Fatalf("read back %+v", rec)
	}
}

// Writing the same record twice keeps one file and one commit.
func TestAWriteOfARecordAlreadyKeptWritesNothing(t *testing.T) {
	a, _, _ := twoClones(t)
	ctx := context.Background()
	if _, err := WriteHandBack(ctx, a.Root(), handBack("iss-1", at0)); err != nil {
		t.Fatal(err)
	}
	first := a.Git("rev-parse", LocalRef)
	wrote, err := WriteHandBack(ctx, a.Root(), handBack("iss-1", at0))
	if err != nil || wrote {
		t.Fatalf("second write = %v, %v; want nothing written", wrote, err)
	}
	if a.Git("rev-parse", LocalRef) != first {
		t.Fatal("the local copy moved on a record it already held")
	}
	// A merge-gate decision asked again over the same head with the same
	// answer is the one record, whenever it is asked.
	d := Decision{Kind: KindMergeGate, At: at0, Site: SiteLane, PR: 7, Head: strings.Repeat("a", 40), Mode: "auto", Role: "maintainer", Arm: false, Why: "no receipt"}
	if wrote, err := WriteDecision(ctx, a.Root(), d); err != nil || !wrote {
		t.Fatalf("decision = %v, %v", wrote, err)
	}
	d.At = at0.Add(time.Hour)
	if wrote, err := WriteDecision(ctx, a.Root(), d); err != nil || wrote {
		t.Fatalf("the same decision an hour later = %v, %v; want nothing written", wrote, err)
	}
	d.Arm = true
	if wrote, err := WriteDecision(ctx, a.Root(), d); err != nil || !wrote {
		t.Fatalf("the gate's other answer = %v, %v; want it written", wrote, err)
	}
}

// A checkout with no origin keeps no record.
func TestAWriteWithNoOriginWritesNothing(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("the base")
	wrote, err := WriteHandBack(context.Background(), r.Root(), handBack("iss-1", at0))
	if err != ErrNoOrigin || wrote {
		t.Fatalf("WriteHandBack = %v, %v; want ErrNoOrigin", wrote, err)
	}
	if out := r.Git("for-each-ref", LocalRef); out != "" {
		t.Fatalf("a ref was written: %s", out)
	}
}

// No home directory, absolute path or control character reaches a record,
// and each text is bounded.
func TestARecordCarriesNoPathAndIsBounded(t *testing.T) {
	a, _, _ := twoClones(t)
	home, _ := os.UserHomeDir()
	h := handBack("iss-1", at0)
	h.Reason = "failed in " + home + "/work/x.go and /private/var/tmp/run/y.txt, see https://example.invalid/a/b \x1b[31mred\x1b[0m " + strings.Repeat("z", 3*maxTextBytes)
	h.Holder = "someone"
	if _, err := WriteHandBack(context.Background(), a.Root(), h); err != nil {
		t.Fatal(err)
	}
	got := readLocal(t, a.Root()).HandBacks[0]
	for _, bad := range []string{home, "/private/var", "\x1b", "someone"} {
		if bad != "" && strings.Contains(got.Reason+got.Holder, bad) {
			t.Fatalf("the record carries %q: %q", bad, got.Reason)
		}
	}
	if !strings.Contains(got.Reason, "https://example.invalid/a/b") {
		t.Fatalf("a URL was taken for a path: %q", got.Reason)
	}
	if len(got.Reason) > maxTextBytes+len("…") {
		t.Fatalf("the reason is %d bytes", len(got.Reason))
	}
}

// A store whose ref names anything but a commit is unreadable: a read and a
// write each refuse, and neither reads it as empty.
func TestAnUnreadableStoreIsNeverEmpty(t *testing.T) {
	a, _, _ := twoClones(t)
	blob := strings.TrimSpace(a.Git("hash-object", "-w", "a.txt"))
	a.Git("update-ref", TrackingRef, blob)
	if _, _, err := Read(context.Background(), a.Root(), TrackingRef); err == nil {
		t.Fatal("Read of a ref naming a blob succeeded")
	}
	if _, err := WriteHandBack(context.Background(), a.Root(), handBack("iss-1", at0)); err == nil {
		t.Fatal("a write over an unreadable store succeeded")
	}
	if out := a.Git("for-each-ref", LocalRef); out != "" {
		t.Fatalf("a write over an unreadable store moved the local copy: %s", out)
	}
}

// A file another writer put on the branch is read only when it is exactly
// what this package writes under that name: an extra field, another version
// or a name its content does not give is named unreadable, and the rest is
// still read.
func TestAFileNotWrittenAsAbcdWritesItIsUnreadable(t *testing.T) {
	a, _, _ := twoClones(t)
	ctx := context.Background()
	if _, err := WriteHandBack(ctx, a.Root(), handBack("iss-1", at0)); err != nil {
		t.Fatal(err)
	}
	good, _ := encode(mustNormal(t, handBack("iss-2", at0)))
	planted := map[string]string{
		"20261113T070000Z-iss-3.json": strings.Replace(string(good), `"issue": "iss-2"`, `"issue": "iss-3", "extra": 1`, 1),
		"20261113T070000Z-iss-4.json": string(good),
		"20261113T070000Z-iss-5.json": strings.Replace(strings.Replace(string(good), `"issue": "iss-2"`, `"issue": "iss-5"`, 1), `"schema_version": 1`, `"schema_version": 2`, 1),
	}
	for name, body := range planted {
		blob := strings.TrimSpace(gitStdin(t, a, body, "hash-object", "-w", "--stdin"))
		r, _ := readRefs(ctx, a.Root())
		top, entries, err := folderEntries(ctx, a.Root(), r.local, HandBacksDir)
		if err != nil {
			t.Fatal(err)
		}
		tree, err := rootWith(ctx, a.Root(), top, HandBacksDir, append(entries, treeEntry{"100644", "blob", blob, name}))
		if err != nil {
			t.Fatal(err)
		}
		c, err := commitTree(ctx, a.Root(), tree, r.local, "planted")
		if err != nil {
			t.Fatal(err)
		}
		if err := moveLocal(ctx, a.Root(), c, r.local); err != nil {
			t.Fatal(err)
		}
	}
	rec := readLocal(t, a.Root())
	if len(rec.HandBacks) != 1 || rec.HandBacks[0].Issue != "iss-1" || len(rec.Unreadable) != 3 {
		t.Fatalf("read %d hand-backs, unreadable %v", len(rec.HandBacks), rec.Unreadable)
	}
}

func mustNormal(t *testing.T, h HandBack) HandBack {
	t.Helper()
	n, err := h.normalise()
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func gitStdin(t *testing.T, r *gittest.Repo, stdin string, args ...string) string {
	t.Helper()
	out, err := git(context.Background(), r.Root(), stdin, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Two writers of one remote each push their records, and neither drops the
// other's: the second lays its records over the first's tip.
func TestTwoWritersBothKeepTheirRecords(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	if _, err := WriteHandBack(ctx, a.Root(), handBack("iss-1", at0)); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteHandBack(ctx, b.Root(), handBack("iss-2", at0)); err != nil {
		t.Fatal(err)
	}
	if res, err := Push(ctx, a.Root()); err != nil || res.Pushed != 1 {
		t.Fatalf("a's push = %+v, %v", res, err)
	}
	first := gitIn(t, bare, "rev-parse", BranchRef)
	res, err := Push(ctx, b.Root())
	if err != nil || res.Pushed != 1 {
		t.Fatalf("b's push = %+v, %v", res, err)
	}
	if got := remoteFiles(t, bare); len(got) != 2 {
		t.Fatalf("origin holds %v, want both records", got)
	}
	// No force: a's commit is still on the branch's history.
	gitIn(t, bare, "merge-base", "--is-ancestor", first, BranchRef)
	// A push with nothing new pushes nothing.
	if res, err := Push(ctx, b.Root()); err != nil || res.Commit != "" {
		t.Fatalf("a second push = %+v, %v; want nothing pushed", res, err)
	}
}

// A writer that loses the race (another pushed between its fetch and its
// push) fetches again and lays its records over the new tip, once.
func TestALostRaceIsTriedOnceMore(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	if _, err := WriteHandBack(ctx, b.Root(), handBack("iss-2", at0)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { beforePush = nil })
	races := 0
	beforePush = func(attempt int) {
		if attempt != 1 {
			return
		}
		races++
		beforePush = nil
		if _, err := WriteHandBack(ctx, a.Root(), handBack("iss-1", at0)); err != nil {
			t.Fatal(err)
		}
		if _, err := Push(ctx, a.Root()); err != nil {
			t.Fatal(err)
		}
		beforePush = func(int) {}
	}
	res, err := Push(ctx, b.Root())
	if err != nil || res.Pushed != 1 || races != 1 {
		t.Fatalf("b's push = %+v, %v (races %d)", res, err, races)
	}
	if got := remoteFiles(t, bare); len(got) != 2 {
		t.Fatalf("origin holds %v, want both records", got)
	}
}

// A writer that loses the race twice gives up, saying so, with its records
// still on its local copy and origin's untouched.
func TestASecondLostRaceKeepsTheRecordsForTheNextPush(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	if _, err := WriteHandBack(ctx, b.Root(), handBack("iss-2", at0)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { beforePush = nil })
	n := 0
	beforePush = func(int) {
		n++
		saved := beforePush
		beforePush = nil
		if _, err := WriteHandBack(ctx, a.Root(), handBack("iss-1", at0.Add(time.Duration(n)*time.Minute))); err != nil {
			t.Fatal(err)
		}
		if _, err := Push(ctx, a.Root()); err != nil {
			t.Fatal(err)
		}
		beforePush = saved
	}
	if _, err := Push(ctx, b.Root()); err == nil {
		t.Fatal("a push that lost twice succeeded")
	}
	beforePush = nil
	if got := remoteFiles(t, bare); len(got) != 2 {
		t.Fatalf("origin holds %v, want a's two records alone", got)
	}
	// b's local copy holds its own record still, laid over origin's.
	if rec := readLocal(t, b.Root()); !slices.ContainsFunc(rec.HandBacks, func(h HandBack) bool { return h.Issue == "iss-2" }) {
		t.Fatalf("b's local copy lost its record: %+v", rec.HandBacks)
	}
	if res, err := Push(ctx, b.Root()); err != nil || res.Pushed != 1 {
		t.Fatalf("the next push = %+v, %v", res, err)
	}
	if got := remoteFiles(t, bare); len(got) != 3 {
		t.Fatalf("origin holds %v, want all three", got)
	}
}

// A record whose name origin already holds with other content is kept as
// origin holds it, and the push names it dropped.
func TestARecordOriginAlreadyHoldsIsKeptAsOriginsAndNamed(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	ha, hb := handBack("iss-1", at0), handBack("iss-1", at0)
	hb.Reason = "another reason"
	if _, err := WriteHandBack(ctx, a.Root(), ha); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteHandBack(ctx, b.Root(), hb); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteHandBack(ctx, b.Root(), handBack("iss-2", at0)); err != nil {
		t.Fatal(err)
	}
	if _, err := Push(ctx, a.Root()); err != nil {
		t.Fatal(err)
	}
	res, err := Push(ctx, b.Root())
	if err != nil || res.Pushed != 1 || len(res.Dropped) != 1 || res.Dropped[0] != "handbacks/20261113T070000Z-iss-1.json" {
		t.Fatalf("b's push = %+v, %v", res, err)
	}
	rec, _, err := Read(ctx, b.Root(), TrackingRef)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range rec.HandBacks {
		if h.Issue == "iss-1" && h.Reason != ha.Reason {
			t.Fatalf("origin's record was replaced: %q", h.Reason)
		}
	}
	if len(remoteFiles(t, bare)) != 2 {
		t.Fatalf("origin holds %v", remoteFiles(t, bare))
	}
}
