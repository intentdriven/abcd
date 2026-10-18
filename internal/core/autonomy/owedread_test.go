package autonomy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/ghcontract"
	"github.com/intentdriven/abcd/internal/gittest"
)

const fixtureEmail = "fixture@example.invalid"

// fakePR is one open pull request the fake forge lists, in the shape `gh pr
// list --json` prints it.
type fakePR struct {
	Number            int              `json:"number"`
	HeadRefName       string           `json:"headRefName"`
	HeadRefOid        string           `json:"headRefOid"`
	BaseRefName       string           `json:"baseRefName"`
	Author            map[string]any   `json:"author"`
	Body              string           `json:"body"`
	MergeStateStatus  string           `json:"mergeStateStatus"`
	ReviewDecision    string           `json:"reviewDecision"`
	StatusCheckRollup []map[string]any `json:"statusCheckRollup"`
	IsCrossRepository bool             `json:"isCrossRepository"`
}

var (
	rollupGreen   = []map[string]any{{"__typename": "CheckRun", "name": "ci", "status": "COMPLETED", "conclusion": "SUCCESS"}}
	rollupFailing = []map[string]any{{"__typename": "CheckRun", "name": "ci", "status": "COMPLETED", "conclusion": "FAILURE"}}
	rollupRunning = []map[string]any{{"__typename": "CheckRun", "name": "ci", "status": "IN_PROGRESS", "conclusion": ""}}
)

func pr(n int, head, base, author string) fakePR {
	return fakePR{Number: n, HeadRefName: head, BaseRefName: base, Author: map[string]any{"login": author},
		MergeStateStatus: "CLEAN", StatusCheckRollup: rollupGreen}
}

// fakeForge answers the one listing a stop check makes, refusing a field the
// real forge client refuses (internal/ghcontract).
type fakeForge struct {
	t     *testing.T
	mu    sync.Mutex
	prs   []fakePR
	calls int
	hang  bool
	err   error
	ctxs  []context.Context
}

func (f *fakeForge) run(ctx context.Context, dir string, args ...string) (string, error) {
	f.mu.Lock()
	f.calls++
	f.ctxs = append(f.ctxs, ctx)
	f.mu.Unlock()
	if err := ghcontract.Check(args); err != nil {
		f.t.Errorf("the listing asks the forge for a field it refuses: %v", err)
		return "", err
	}
	if len(args) < 2 || args[0] != "pr" || args[1] != "list" {
		f.t.Errorf("the stop check's one forge call is a pull-request listing; got %v", args)
	}
	if f.hang {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if f.err != nil {
		return "", f.err
	}
	raw, err := json.Marshal(f.prs)
	return string(raw), err
}

// fx is a managed checkout with an origin, the default branch pushed and
// fetched, a fake forge and the reservations as last fetched.
type fx struct {
	t      *testing.T
	repo   *gittest.Repo
	root   string
	home   string
	origin string
	now    time.Time
	forge  *fakeForge
	res    []claim.Reservation
	email  string
	files  int // the files commitOn has added, for the next one's name
}

func newFx(t *testing.T) *fx {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	repo.Commit("seed")
	root, err := filepath.EvalSymlinks(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	f := &fx{t: t, repo: repo, root: root, home: home, now: time.Now().Truncate(time.Second), email: fixtureEmail}
	f.forge = &fakeForge{t: t}
	f.origin = filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, f.origin)
	f.git("remote", "add", "origin", f.origin)
	f.git("push", "-q", "origin", "main")
	f.git("remote", "set-head", "origin", "main")
	return f
}

func (f *fx) git(args ...string) string { f.t.Helper(); return f.repo.Git(args...) }

// gitIn runs git in the fixture with stdin, under the fixture's isolated
// environment.
func (f *fx) gitIn(stdin string, args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", f.root}, args...)...)
	cmd.Env = f.repo.Env()
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		f.t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// treeWith is branch's tree with one file added, written to the object store.
func (f *fx) treeWith(branch, name, content string) string {
	f.t.Helper()
	blob := f.gitIn(content, "hash-object", "-w", "--stdin")
	entries := f.git("ls-tree", branch+"^{tree}")
	if entries != "" {
		entries += "\n"
	}
	return f.gitIn(entries+"100644 blob "+blob+"\t"+name+"\n", "mktree")
}

// commitOn records a commit on branch without checking it out, by author
// (the fixture's own when ""), and returns it. The commit adds a file of its
// own, so its change is real: a commit that changes nothing has, by
// definition, already landed wherever its branch is merged.
func (f *fx) commitOn(branch, msg, author string) string {
	f.t.Helper()
	f.files++
	tree := f.treeWith(branch, fmt.Sprintf("f-%d.txt", f.files), msg+"\n")
	args := []string{}
	if author != "" {
		args = append(args, "-c", "user.email="+author, "-c", "user.name="+strings.Split(author, "@")[0])
	}
	args = append(args, "commit-tree", tree, "-p", branch, "-m", msg)
	c := f.git(args...)
	f.git("update-ref", "refs/heads/"+branch, c)
	return c
}

// own checks branch out in this copy and back, so its HEAD reflog holds it.
func (f *fx) own(branch string) {
	f.t.Helper()
	f.git("checkout", "-q", branch)
	f.git("checkout", "-q", "main")
}

func (f *fx) push(branch string) { f.t.Helper(); f.git("push", "-q", "-u", "origin", branch) }

func (f *fx) head(branch string) string {
	f.t.Helper()
	return f.git("rev-parse", "refs/heads/"+branch)
}

func (f *fx) mintReceipts() string {
	f.t.Helper()
	dir := filepath.Join(f.root, ".abcd", ".work.local", "preflight-receipts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		f.t.Fatal(err)
	}
	return dir
}

func (f *fx) receipt(sha string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.mintReceipts(), sha), []byte("ok\n"), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fx) reserve(id, copy string, claimed, expiry time.Time) {
	f.res = append(f.res, claim.Reservation{Record: id, Copy: copy, State: claim.StateLive, ClaimedAt: claimed, EffectiveExpiry: expiry})
}

// reserveFresh is a live reservation under this copy's key, claimed an hour
// ago with a four-hour lease, so its renewal is not due.
func (f *fx) reserveFresh(id string) {
	f.reserve(id, claim.CopyThis, f.now.Add(-time.Hour), f.now.Add(3*time.Hour))
}

func (f *fx) opts() ReadOptions {
	return ReadOptions{
		Root:   f.root,
		Home:   f.home,
		Now:    func() time.Time { return f.now },
		Forge:  f.forge.run,
		Person: "alice",
		Email:  f.email,
		Reservations: func(string, time.Time) (claim.Snapshot, error) {
			return claim.Snapshot{Reservations: f.res}, nil
		},
	}
}

func (f *fx) awayRecord(hold bool, steps ...string) *Record {
	f.t.Helper()
	o := Options{Home: f.home, Now: func() time.Time { return f.now }, Who: func(string) string { return "alice" }}
	rec, err := SetAway(f.root, "", hold, o)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, s := range steps {
		if rec, err = QueueStep(f.root, s, o); err != nil {
			f.t.Fatal(err)
		}
	}
	return rec
}

// read is ReadOwed while away (with a fresh away record) or present.
func (f *fx) read(away bool) Owed {
	f.t.Helper()
	o := f.opts()
	if away {
		o.Record = f.awayRecord(false)
	}
	return f.readWith(o)
}

func (f *fx) readWith(o ReadOptions) Owed {
	f.t.Helper()
	owed, err := ReadOwed(context.Background(), o)
	if err != nil {
		f.t.Fatal(err)
	}
	return owed
}

func has(o Owed, kind OwedKind, key string) bool {
	_, ok := find(o, kind, key)
	return ok
}

func find(o Owed, kind OwedKind, key string) (OwedItem, bool) {
	for _, i := range o.Items {
		if i.Kind == kind && (i.Branch == key || i.Record == key) {
			return i, true
		}
	}
	return OwedItem{}, false
}

func describe(o Owed) string {
	var b strings.Builder
	for _, i := range o.Items {
		b.WriteString(string(i.Kind) + ":" + i.Branch + i.Record + " ")
	}
	b.WriteString("notes=" + strings.Join(o.Notes, " | "))
	return b.String()
}

// TestOwnBranchFromReflogWithoutWindow: a branch checked out before the mode
// was set, with a recent commit of this checkout's, is still own.
func TestOwnBranchFromReflogWithoutWindow(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/a")
	f.commitOn("feat/a", "work", "")
	f.push("feat/a")
	f.own("feat/a")
	f.now = f.now.Add(6 * time.Hour) // the mode is set long after the checkout
	p := pr(1, "feat/a", "main", "alice")
	p.MergeStateStatus = "BEHIND"
	f.forge.prs = []fakePR{p}
	owed := f.read(true)
	if it, ok := find(owed, OwedBehind, "feat/a"); !ok || it.PR != 1 || !it.Capped {
		t.Fatalf("the branch checked out before the window must be own and owed as behind: %s", describe(owed))
	}
}

// TestOwnBranchFromRecordedCreation: a branch cut with `git worktree add -b`,
// absent from this copy's reflog, with no upstream and no record id, is own
// once recorded.
func TestOwnBranchFromRecordedCreation(t *testing.T) {
	f := newFx(t)
	wt := filepath.Join(t.TempDir(), "s")
	f.git("worktree", "add", "-q", "-b", "feat/s", wt)
	if err := os.WriteFile(filepath.Join(wt, "s.txt"), []byte("stacked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.git("-C", wt, "add", "s.txt")
	f.git("-C", wt, "commit", "-q", "-m", "stacked work")
	if owed := f.read(false); has(owed, OwedUnpushed, "feat/s") {
		t.Fatalf("an unrecorded branch in another worktree is not own: %s", describe(owed))
	}
	st, err := OpenStore(f.home, f.root)
	if err != nil {
		t.Fatal(err)
	}
	key, err := claim.CopyKey(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.RecordBranches(key, []BranchChange{{Kind: BranchCreated, Name: "feat/s"}}, f.now); err != nil {
		t.Fatal(err)
	}
	if owed := f.read(false); !has(owed, OwedUnpushed, "feat/s") {
		t.Fatalf("a recorded creation is own, and its unpushed commit owed: %s", describe(owed))
	}
}

// writeRun lays a run state in this copy's local tier: key, its lanes'
// branches and stages, and whether a spec step is still pending.
func (f *fx) writeRun(id, key string, pending bool, lanes ...map[string]any) {
	f.t.Helper()
	dir := filepath.Join(f.root, ".abcd", ".work.local", "run", id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		f.t.Fatal(err)
	}
	st := map[string]any{"schema_version": 9, "run_id": id, "key": key, "lanes": lanes, "pending": []any{}}
	if pending {
		st["pending"] = []any{map[string]any{"spec_step": 2}}
	}
	raw, _ := json.Marshal(st)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), raw, 0o600); err != nil {
		f.t.Fatal(err)
	}
}

// TestOwnBranchFromLiveRunLane: a lane of a run whose reservation is live
// under this copy's key is own; a lane of a run whose reservation lapsed, or
// of a finished run, is not.
func TestOwnBranchFromLiveRunLane(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/lane")
	f.commitOn("feat/lane", "lane work", "bob@example.invalid")
	f.writeRun("run-0000000000000001", "itd-1", false, map[string]any{"id": "lane-1", "branch": "feat/lane", "stage": "validate"})
	f.reserveFresh("itd-1")
	if owed := f.read(false); !has(owed, OwedUnpushed, "feat/lane") {
		t.Fatalf("a live run's lane is own: %s", describe(owed))
	}
	f.res[0].State = claim.StateExpired
	if owed := f.read(false); has(owed, OwedUnpushed, "feat/lane") {
		t.Fatalf("a lane whose run's reservation lapsed is not own: %s", describe(owed))
	}
	f.res[0].State = claim.StateLive
	f.writeRun("run-0000000000000001", "itd-1", false, map[string]any{"id": "lane-1", "branch": "feat/lane", "stage": "done"})
	if owed := f.read(false); has(owed, OwedUnpushed, "feat/lane") {
		t.Fatalf("a finished run's lane is not own: %s", describe(owed))
	}
}

// TestOwnBranchFromLinkedReservation: the head branch of an open pull request
// a closing trailer links to a reservation under this copy's key is own.
func TestOwnBranchFromLinkedReservation(t *testing.T) {
	f := newFx(t)
	f.reserveFresh("iss-7")
	p := pr(3, "fix/other", "main", "alice")
	p.Body = "Fixes the thing.\n\nResolves: iss-7\nAssisted-by: x"
	p.StatusCheckRollup = rollupFailing
	f.forge.prs = []fakePR{p}
	owed := f.read(true)
	if it, ok := find(owed, OwedFailing, "fix/other"); !ok || it.PR != 3 {
		t.Fatalf("the linked pull request's head is own, and failing is owed: %s", describe(owed))
	}
	if has(owed, OwedUncarried, "iss-7") {
		t.Fatalf("the linked pull request carries the reservation's work: %s", describe(owed))
	}
}

// TestStaleOwnBranchNotOwed: a reflog or recorded branch whose newest commit
// of this checkout's is older than 14 days, with no live reservation, is not
// owed; a live reservation linking it keeps it owed.
func TestStaleOwnBranchNotOwed(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/old")
	f.commitOn("feat/old", "old work", "")
	f.own("feat/old")
	f.git("branch", "feat/iss-9-old")
	f.commitOn("feat/iss-9-old", "old linked work", "")
	f.own("feat/iss-9-old")
	f.now = f.now.Add(20 * 24 * time.Hour)
	f.reserveFresh("iss-9")
	owed := f.read(false)
	if has(owed, OwedUnpushed, "feat/old") {
		t.Fatalf("a branch with no commit of this checkout's in 14 days is not owed: %s", describe(owed))
	}
	if !has(owed, OwedUnpushed, "feat/iss-9-old") {
		t.Fatalf("a live reservation linking a stale branch keeps it owed: %s", describe(owed))
	}
}

// TestReviewCheckoutNotOwn: a branch checked out to review, holding only
// another author's commits, and the head of a pull request another person
// authored, are never own.
func TestReviewCheckoutNotOwn(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/review")
	f.commitOn("feat/review", "their work", "bob@example.invalid")
	f.own("feat/review")
	f.git("branch", "feat/theirs")
	f.commitOn("feat/theirs", "their work", "bob@example.invalid")
	f.commitOn("feat/theirs", "my suggestion", "")
	f.own("feat/theirs")
	p := pr(5, "feat/theirs", "main", "bob")
	p.MergeStateStatus = "BEHIND"
	f.forge.prs = []fakePR{p}
	owed := f.read(true)
	for _, b := range []string{"feat/review", "feat/theirs"} {
		for _, i := range owed.Items {
			if i.Branch == b {
				t.Fatalf("%s is never own: %s", b, describe(owed))
			}
		}
	}
}

// TestSiblingBranchNotOwn: a branch a sibling working copy cut and works on
// is none of this copy's routes, so it never holds this session.
func TestSiblingBranchNotOwn(t *testing.T) {
	f := newFx(t)
	wt := filepath.Join(t.TempDir(), "sib")
	f.git("worktree", "add", "-q", "-b", "feat/sib", wt)
	f.git("-C", wt, "commit", "-q", "--allow-empty", "-m", "sibling work")
	st, _ := OpenStore(f.home, f.root)
	if err := st.RecordBranches("fedcba9876543210fedcba9876543210", []BranchChange{{Kind: BranchCreated, Name: "feat/sib"}}, f.now); err != nil {
		t.Fatal(err)
	}
	f.forge.prs = []fakePR{pr(8, "feat/sib", "main", "alice")}
	f.forge.prs[0].StatusCheckRollup = rollupFailing
	owed := f.read(true)
	for _, i := range owed.Items {
		if i.Branch == "feat/sib" {
			t.Fatalf("a sibling copy's branch is not own: %s", describe(owed))
		}
	}
}

// TestReservationCopyKeyFilter: a sibling copy's live reservation is not
// owed; this copy's is, as uncarried work and, half through, as a renewal.
func TestReservationCopyKeyFilter(t *testing.T) {
	f := newFx(t)
	f.reserve("iss-1", "", f.now.Add(-3*time.Hour), f.now.Add(time.Hour))
	if owed := f.read(true); !owed.Empty() {
		t.Fatalf("a sibling copy's reservation is not owed: %s", describe(owed))
	}
	f.res[0].Copy = claim.CopyThis
	owed := f.read(true)
	if !has(owed, OwedUncarried, "iss-1") || !has(owed, OwedRenewal, "iss-1") {
		t.Fatalf("this copy's reservation, half through and carried by nothing: %s", describe(owed))
	}
	if it, _ := find(owed, OwedRenewal, "iss-1"); it.Capped {
		t.Fatalf("a renewal is never capped")
	}
	if it, _ := find(owed, OwedUncarried, "iss-1"); !it.Capped {
		t.Fatalf("uncarried work is capped")
	}
}

// TestNextRenewalEarliestLive: Owed.NextRenewal is the earliest renewal due
// (halfway between the latest claim and the expiry) among the live
// reservations under this copy's key, due or not yet, which the stop
// verdict's deadline needs; a sibling copy's or a lapsed reservation does not
// count, and with none it is zero. It is filled while present too, since it
// is read from disk.
func TestNextRenewalEarliestLive(t *testing.T) {
	f := newFx(t)
	if owed := f.read(true); !owed.NextRenewal.IsZero() {
		t.Fatalf("with no reservation NextRenewal is zero, got %s", owed.NextRenewal)
	}
	// due in 1h30: claimed an hour ago, expiring in four hours
	f.reserve("iss-1", claim.CopyThis, f.now.Add(-time.Hour), f.now.Add(4*time.Hour))
	// due in 30m, but a sibling copy's
	f.reserve("iss-2", "", f.now.Add(-time.Hour), f.now.Add(2*time.Hour))
	// due in 45m: claimed 15 minutes ago, a two-hour lease
	f.reserve("iss-3", claim.CopyThis, f.now.Add(-15*time.Minute), f.now.Add(105*time.Minute))
	// lapsed
	f.res = append(f.res, claim.Reservation{Record: "iss-4", Copy: claim.CopyThis, State: claim.StateExpired,
		ClaimedAt: f.now.Add(-5 * time.Hour), EffectiveExpiry: f.now.Add(-time.Hour)})
	want := f.now.Add(45 * time.Minute)
	for _, away := range []bool{true, false} {
		if got := f.read(away).NextRenewal; !got.Equal(want) {
			t.Fatalf("away=%v: NextRenewal = %s, want iss-3's due %s", away, got, want)
		}
	}
	// a renewal already due still counts: it is the earliest
	f.reserve("iss-5", claim.CopyThis, f.now.Add(-3*time.Hour), f.now.Add(time.Hour))
	if got, want := f.read(true).NextRenewal, f.now.Add(-time.Hour); !got.Equal(want) {
		t.Fatalf("a renewal already due is the earliest: NextRenewal = %s, want %s", got, want)
	}
}

// TestReservationLinkByHeadOrClosingTrailer: the record id in the head branch
// or on a Resolves: or Delivers: line links; a green linked pull request is
// not owed until renewal is due.
func TestReservationLinkByHeadOrClosingTrailer(t *testing.T) {
	f := newFx(t)
	f.reserveFresh("iss-11")
	f.reserveFresh("iss-12")
	f.reserveFresh("itd-13")
	a := pr(1, "fix/iss-11-thing", "main", "alice")
	b := pr(2, "fix/y", "main", "alice")
	b.Body = "Delivers: itd-13"
	c := pr(3, "fix/z", "main", "alice")
	c.Body = "Mentions iss-12 in passing."
	f.forge.prs = []fakePR{a, b, c}
	owed := f.read(true)
	if has(owed, OwedUncarried, "iss-11") || has(owed, OwedUncarried, "itd-13") || !has(owed, OwedUncarried, "iss-12") {
		t.Fatalf("head branch and closing trailer link; a mention does not: %s", describe(owed))
	}
	if len(owed.Items) != 1 {
		t.Fatalf("green linked pull requests are not owed: %s", describe(owed))
	}
	f.now = f.now.Add(2 * time.Hour)
	owed = f.read(true)
	if !has(owed, OwedRenewal, "iss-11") || !has(owed, OwedRenewal, "itd-13") {
		t.Fatalf("once half through, each renewal is due: %s", describe(owed))
	}
}

// TestRecordIDWordBoundary: iss-5 does not match iss-50 in a branch name or a
// trailer.
func TestRecordIDWordBoundary(t *testing.T) {
	f := newFx(t)
	f.reserveFresh("iss-5")
	p := pr(4, "fix/iss-50", "main", "alice")
	p.Body = "Resolves: iss-50"
	p.StatusCheckRollup = rollupFailing
	f.forge.prs = []fakePR{p}
	owed := f.read(true)
	if !has(owed, OwedUncarried, "iss-5") || has(owed, OwedFailing, "fix/iss-50") {
		t.Fatalf("iss-5 is not iss-50: %s", describe(owed))
	}
	if !mentionsID("feat/iss-5", "iss-5") || mentionsID("feat/iss-50", "iss-5") || !mentionsID("Resolves: iss-5, iss-6", "iss-5") {
		t.Fatalf("the word-boundary match is wrong")
	}
}

// TestRefsLineCarriesOnlyOwnBranch: a Refs: line on a sibling copy's pull
// request neither carries the work nor makes its branch own; on an own
// branch's pull request it carries it.
func TestRefsLineCarriesOnlyOwnBranch(t *testing.T) {
	f := newFx(t)
	f.reserveFresh("iss-20")
	sib := pr(1, "feat/sib", "main", "alice")
	sib.Body = "Refs: iss-20"
	sib.StatusCheckRollup = rollupFailing
	f.forge.prs = []fakePR{sib}
	owed := f.read(true)
	if !has(owed, OwedUncarried, "iss-20") || has(owed, OwedFailing, "feat/sib") {
		t.Fatalf("a Refs: line on a branch not own carries nothing and owns nothing: %s", describe(owed))
	}
	f.git("branch", "feat/mine")
	f.commitOn("feat/mine", "work", "")
	f.push("feat/mine")
	f.own("feat/mine")
	mine := pr(2, "feat/mine", "main", "alice")
	mine.Body = "Refs: iss-20"
	f.forge.prs = []fakePR{sib, mine}
	if owed := f.read(true); has(owed, OwedUncarried, "iss-20") {
		t.Fatalf("a Refs: line on an own branch's pull request carries the work: %s", describe(owed))
	}
}

// TestBehindFromMergeState: a pull request whose merge state is BEHIND is
// owed while the local default branch is stale and shows it current; DIRTY is
// owed as behind; CLEAN is not; UNKNOWN reads as not read.
func TestBehindFromMergeState(t *testing.T) {
	for _, c := range []struct {
		state string
		owed  bool
	}{{"BEHIND", true}, {"DIRTY", true}, {"CLEAN", false}, {"UNKNOWN", false}} {
		t.Run(c.state, func(t *testing.T) {
			f := newFx(t)
			f.git("branch", "feat/a")
			f.commitOn("feat/a", "work", "")
			f.push("feat/a")
			f.own("feat/a")
			p := pr(1, "feat/a", "main", "alice")
			p.MergeStateStatus = c.state
			f.forge.prs = []fakePR{p}
			owed := f.read(true)
			if has(owed, OwedBehind, "feat/a") != c.owed {
				t.Fatalf("merge state %s: owed as behind = %v, want %v: %s", c.state, !c.owed, c.owed, describe(owed))
			}
			if c.state == "UNKNOWN" && !strings.Contains(strings.Join(owed.Notes, " "), "not read") {
				t.Fatalf("UNKNOWN must be said as not read: %s", describe(owed))
			}
		})
	}
}

// stack makes feat/base (own, pushed, one commit off main) and feat/top on it
// (own, pushed, one commit more).
func (f *fx) stack() {
	f.t.Helper()
	f.git("branch", "feat/base")
	f.commitOn("feat/base", "base work", "")
	f.push("feat/base")
	f.own("feat/base")
	f.git("branch", "feat/top", "feat/base")
	f.commitOn("feat/top", "top work", "")
	f.push("feat/top")
	f.own("feat/top")
}

// TestStackedBranchFoundWithoutPullRequest: with no pull request, a branch
// whose nearest own ancestor branch is not in the default branch is stacked,
// and owed while behind that base.
func TestStackedBranchFoundWithoutPullRequest(t *testing.T) {
	f := newFx(t)
	f.stack()
	f.commitOn("feat/base", "base moves on", "")
	owed := f.read(true)
	it, ok := find(owed, OwedStacked, "feat/top")
	if !ok || !strings.Contains(it.Reason, "feat/base") {
		t.Fatalf("feat/top is stacked on feat/base and behind it: %s", describe(owed))
	}
	if has(owed, OwedStacked, "feat/base") {
		t.Fatalf("feat/base stands on the default branch, not stacked: %s", describe(owed))
	}
}

// TestStackedBranchClearingRule: a stacked branch is owed while behind its
// base, failing, with a review outstanding, or unverified; waiting on its base
// otherwise.
func TestStackedBranchClearingRule(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*fakePR)
		receipt bool
		owed    bool
	}{
		{"behind", func(p *fakePR) { p.MergeStateStatus = "BEHIND" }, true, true},
		{"failing", func(p *fakePR) { p.StatusCheckRollup = rollupFailing }, true, true},
		{"review", func(p *fakePR) { p.ReviewDecision = "CHANGES_REQUESTED" }, true, true},
		{"unverified", func(p *fakePR) {}, false, true},
		{"waiting", func(p *fakePR) {}, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFx(t)
			f.stack()
			f.mintReceipts()
			if c.receipt {
				f.receipt(f.head("feat/top"))
			}
			base := pr(1, "feat/base", "main", "alice")
			top := pr(2, "feat/top", "feat/base", "alice")
			c.mutate(&top)
			f.forge.prs = []fakePR{base, top}
			owed := f.read(true)
			if has(owed, OwedStacked, "feat/top") != c.owed {
				t.Fatalf("%s: stacked owed = %v, want %v: %s", c.name, !c.owed, c.owed, describe(owed))
			}
			if has(owed, OwedBehind, "feat/top") || has(owed, OwedFailing, "feat/top") {
				t.Fatalf("a stacked branch is judged by the clearing rule alone: %s", describe(owed))
			}
		})
	}
}

// TestClearingRuleReview: CHANGES_REQUESTED in the listing, and a run lane
// with no passing review of its head, are owed.
func TestClearingRuleReview(t *testing.T) {
	f := newFx(t)
	f.stack()
	f.receipt(f.head("feat/top"))
	f.forge.prs = []fakePR{pr(1, "feat/base", "main", "alice"), pr(2, "feat/top", "feat/base", "alice")}
	f.reserveFresh("itd-2")
	lane := func(head string, pass bool) map[string]any {
		return map[string]any{"id": "lane-1", "branch": "feat/top", "stage": "validate", "validation": []any{
			map[string]any{"round": 1, "head_sha": head, "validators": []any{map[string]any{"role": "r", "pass": pass}}},
		}}
	}
	f.writeRun("run-0000000000000002", "itd-2", true, lane("0000000000000000000000000000000000000000", true))
	if owed := f.read(true); !has(owed, OwedStacked, "feat/top") {
		t.Fatalf("a lane with no passing review of its head is owed: %s", describe(owed))
	}
	f.writeRun("run-0000000000000002", "itd-2", true, lane(f.head("feat/top"), false))
	if owed := f.read(true); !has(owed, OwedStacked, "feat/top") {
		t.Fatalf("a lane whose review of its head did not pass is owed: %s", describe(owed))
	}
	f.writeRun("run-0000000000000002", "itd-2", true, lane(f.head("feat/top"), true))
	if owed := f.read(true); has(owed, OwedStacked, "feat/top") {
		t.Fatalf("a lane with a passing review of its head waits on its base: %s", describe(owed))
	}
	f.forge.prs[1].ReviewDecision = "CHANGES_REQUESTED"
	if owed := f.read(true); !has(owed, OwedStacked, "feat/top") {
		t.Fatalf("CHANGES_REQUESTED is owed: %s", describe(owed))
	}
}

// TestClearingRuleWithoutReceipts: in a repository with no receipt folder,
// passing checks verify; checks still running owe it uncapped; with no pull
// request the reason says it was not verified.
func TestClearingRuleWithoutReceipts(t *testing.T) {
	f := newFx(t)
	f.stack()
	f.forge.prs = []fakePR{pr(1, "feat/base", "main", "alice"), pr(2, "feat/top", "feat/base", "alice")}
	if owed := f.read(true); has(owed, OwedStacked, "feat/top") {
		t.Fatalf("passing checks verify where no receipts are minted: %s", describe(owed))
	}
	f.forge.prs[1].StatusCheckRollup = rollupRunning
	owed := f.read(true)
	it, ok := find(owed, OwedStacked, "feat/top")
	if !ok || it.Capped {
		t.Fatalf("checks still running owe it, not capped: %s", describe(owed))
	}
	f.forge.prs = nil
	owed = f.read(true)
	if has(owed, OwedStacked, "feat/top") || !strings.Contains(strings.Join(owed.Notes, " "), "not verified") {
		t.Fatalf("with no pull request it is judged on the rest, and said not verified: %s", describe(owed))
	}
	f.commitOn("feat/base", "base moves on", "")
	owed = f.read(true)
	if it, ok := find(owed, OwedStacked, "feat/top"); !ok || !strings.Contains(it.Reason, "not verified") {
		t.Fatalf("owed for being behind, the reason says it was not verified: %s", describe(owed))
	}
}

// TestUnpushedSkipsGoneUpstream: a squash-merged branch whose upstream is gone
// is not owed.
func TestUnpushedSkipsGoneUpstream(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/g")
	f.commitOn("feat/g", "work", "")
	f.push("feat/g")
	f.own("feat/g")
	f.commitOn("feat/g", "more", "")
	if owed := f.read(false); !has(owed, OwedUnpushed, "feat/g") {
		t.Fatalf("a commit not on the upstream is owed: %s", describe(owed))
	}
	f.git("push", "-q", "origin", "--delete", "feat/g")
	f.git("fetch", "-q", "--prune", "origin")
	if owed := f.read(false); has(owed, OwedUnpushed, "feat/g") {
		t.Fatalf("a branch whose upstream is gone is not owed: %s", describe(owed))
	}
}

// TestQueuedPushHandsItemBack: without --hold-pushes a queued step hands
// nothing back; with it, a step naming an unpushed branch whose head contains
// the base's last-fetched tip and has a receipt hands that branch's unpushed
// item back, recorded once per branch and head; nothing else is handed back.
func TestQueuedPushHandsItemBack(t *testing.T) {
	setup := func(t *testing.T) *fx {
		f := newFx(t)
		f.git("branch", "feat/p")
		f.commitOn("feat/p", "work", "")
		f.own("feat/p")
		f.receipt(f.head("feat/p"))
		return f
	}
	readWith := func(f *fx, hold bool, steps ...string) Owed {
		o := f.opts()
		o.Record = f.awayRecord(hold, steps...)
		return f.readWith(o)
	}
	handedBack := func(t *testing.T, f *fx) []HandBack {
		st, _ := OpenStore(f.home, f.root)
		rec, err := st.Read()
		if err != nil {
			t.Fatal(err)
		}
		return rec.HandedBack
	}

	t.Run("no hold-pushes", func(t *testing.T) {
		f := setup(t)
		if owed := readWith(f, false, "push feat/p"); !has(owed, OwedUnpushed, "feat/p") {
			t.Fatalf("without --hold-pushes a queued push hands nothing back: %s", describe(owed))
		}
		if hb := handedBack(t, f); len(hb) != 0 {
			t.Fatalf("nothing is recorded as handed back: %v", hb)
		}
	})
	t.Run("hold-pushes verified", func(t *testing.T) {
		f := setup(t)
		owed := readWith(f, true, "push feat/p")
		if has(owed, OwedUnpushed, "feat/p") {
			t.Fatalf("a verified branch's queued push is handed back: %s", describe(owed))
		}
		if len(owed.HandedBack) != 1 || owed.HandedBack[0].Kind != OwedUnpushed ||
			owed.HandedBack[0].Branch != "feat/p" || owed.HandedBack[0].Head != f.head("feat/p") {
			t.Fatalf("the handed-back item goes to the verdict in Owed.HandedBack, with its branch and head: %+v", owed.HandedBack)
		}
		// The verdict accepts a hand-back and the stop hook records it; the
		// reader records nothing, so one owner writes it once.
		if hb := handedBack(t, f); len(hb) != 0 {
			t.Fatalf("the reader records no hand-back itself: %+v", hb)
		}
	})
	t.Run("hold-pushes base moved", func(t *testing.T) {
		f := setup(t)
		f.commitOn("main", "main moves on", "")
		f.git("push", "-q", "origin", "main")
		if owed := readWith(f, true, "push feat/p"); !has(owed, OwedUnpushed, "feat/p") {
			t.Fatalf("a head lacking the base's tip is not handed back: %s", describe(owed))
		}
	})
	t.Run("hold-pushes no receipt", func(t *testing.T) {
		f := setup(t)
		f.commitOn("feat/p", "more", "")
		if owed := readWith(f, true, "push feat/p"); !has(owed, OwedUnpushed, "feat/p") {
			t.Fatalf("a head with no receipt is not handed back: %s", describe(owed))
		}
	})
	t.Run("hold-pushes mints no receipts", func(t *testing.T) {
		f := setup(t)
		if err := os.RemoveAll(f.mintReceipts()); err != nil {
			t.Fatal(err)
		}
		if owed := readWith(f, true, "push feat/p"); !has(owed, OwedUnpushed, "feat/p") {
			t.Fatalf("a repository that mints no receipts hands nothing back: %s", describe(owed))
		}
	})
	t.Run("hold-pushes behind and failing pull requests", func(t *testing.T) {
		f := setup(t)
		for _, b := range []string{"feat/b", "feat/f"} {
			f.git("branch", b)
			f.commitOn(b, "work", "")
			f.push(b)
			f.own(b)
			f.commitOn(b, "local fix", "")
			f.receipt(f.head(b))
		}
		behind := pr(1, "feat/b", "main", "alice")
		behind.MergeStateStatus = "DIRTY"
		failing := pr(2, "feat/f", "main", "alice")
		failing.StatusCheckRollup = rollupFailing
		f.forge.prs = []fakePR{behind, failing}
		owed := readWith(f, true, "push feat/b", "push feat/f")
		if !has(owed, OwedBehind, "feat/b") || !has(owed, OwedFailing, "feat/f") {
			t.Fatalf("a behind or failing pull request stays owed though its fix is queued: %s", describe(owed))
		}
	})
	t.Run("hold-pushes stacked and review", func(t *testing.T) {
		f := setup(t)
		f.stack()
		f.commitOn("feat/top", "unpushed", "")
		f.receipt(f.head("feat/top"))
		top := pr(2, "feat/top", "feat/base", "alice")
		top.ReviewDecision = "CHANGES_REQUESTED"
		f.forge.prs = []fakePR{pr(1, "feat/base", "main", "alice"), top}
		owed := readWith(f, true, "push feat/top")
		if !has(owed, OwedStacked, "feat/top") {
			t.Fatalf("a stacked branch the clearing rule owes, a review outstanding, stays owed: %s", describe(owed))
		}
	})
	t.Run("hold-pushes reservation and renewal", func(t *testing.T) {
		f := setup(t)
		f.reserve("iss-30", claim.CopyThis, f.now.Add(-3*time.Hour), f.now.Add(time.Hour))
		owed := readWith(f, true, "open the pull request for iss-30", "renew iss-30")
		if !has(owed, OwedUncarried, "iss-30") || !has(owed, OwedRenewal, "iss-30") {
			t.Fatalf("a reservation queued by its id, and a renewal, are never handed back: %s", describe(owed))
		}
	})
	t.Run("hold-pushes word boundary", func(t *testing.T) {
		f := setup(t)
		f.git("branch", "feat/iss-50")
		f.commitOn("feat/iss-50", "work", "")
		f.own("feat/iss-50")
		f.receipt(f.head("feat/iss-50"))
		if owed := readWith(f, true, "push feat/iss-5"); !has(owed, OwedUnpushed, "feat/iss-50") {
			t.Fatalf("iss-5 queued does not hand back iss-50: %s", describe(owed))
		}
	})
}

// TestPresentReadsDiskOnly: while the person is present the check makes no
// forge call, and reads renewals due and unpushed own branches only.
func TestPresentReadsDiskOnly(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/a")
	f.commitOn("feat/a", "work", "")
	f.own("feat/a")
	f.reserve("iss-3", claim.CopyThis, f.now.Add(-3*time.Hour), f.now.Add(time.Hour))
	f.forge.prs = []fakePR{pr(1, "feat/a", "main", "alice")}
	owed := f.read(false)
	if f.forge.calls != 0 {
		t.Fatalf("present: %d forge calls, want none", f.forge.calls)
	}
	if !has(owed, OwedUnpushed, "feat/a") || !has(owed, OwedRenewal, "iss-3") || has(owed, OwedUncarried, "iss-3") {
		t.Fatalf("present reads renewals due and unpushed branches only: %s", describe(owed))
	}
	// A record past its end is not away either.
	o := f.opts()
	rec := f.awayRecord(false)
	o.Record = rec
	o.Now = func() time.Time { return rec.Ends().Add(time.Minute) }
	f.readWith(o)
	if f.forge.calls != 0 {
		t.Fatalf("a lapsed record is not away: %d forge calls", f.forge.calls)
	}
}

// TestForgeListingDeadline: the listing runs under a ten-second deadline; a
// hung listing ends at it, and the check fails open to the disk kinds with a
// "not read" note: a branch with a pull request is compared on disk naming
// the fetch time, and a reservation is never reported as uncarried.
func TestForgeListingDeadline(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/a")
	f.commitOn("feat/a", "work", "")
	f.push("feat/a")
	f.own("feat/a")
	f.commitOn("main", "main moves on", "")
	f.git("push", "-q", "origin", "main")
	f.reserveFresh("iss-4")

	f.forge.err = errors.New("gh: not signed in")
	o := f.opts()
	o.Record = f.awayRecord(false)
	start := time.Now()
	f.readWith(o)
	if len(f.forge.ctxs) != 1 {
		t.Fatalf("one listing per check, got %d", len(f.forge.ctxs))
	}
	dl, ok := f.forge.ctxs[0].Deadline()
	if !ok || dl.Sub(start) > ForgeDeadline+time.Second || dl.Sub(start) < ForgeDeadline-time.Second {
		t.Fatalf("the listing's deadline is %v from the start, want %v", dl.Sub(start), ForgeDeadline)
	}

	f.forge.err = nil
	f.forge.hang = true
	o.ForgeDeadline = 300 * time.Millisecond
	start = time.Now()
	owed := f.readWith(o)
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("a hung listing must end at its deadline; the check took %v", took)
	}
	notes := strings.Join(owed.Notes, " ")
	if !strings.Contains(notes, "not read") {
		t.Fatalf("a failed listing is named as not read: %s", describe(owed))
	}
	it, ok := find(owed, OwedBehind, "feat/a")
	if !ok || !strings.Contains(it.Reason, "origin/main") || !strings.Contains(it.Reason, "fetched") {
		t.Fatalf("a pushed branch is compared on disk with the default branch, naming the fetch: %s", describe(owed))
	}
	if has(owed, OwedUncarried, "iss-4") {
		t.Fatalf("a reservation is never reported uncarried when the listing failed: %s", describe(owed))
	}
}

// TestOwedOrderFixed: the items come in the fixed order, and the next item is
// the first of a renewal due, a failing pull request, a behind one, a stacked
// branch, an unpushed branch and unhanded work.
func TestOwedOrderFixed(t *testing.T) {
	f := newFx(t)
	f.stack()
	f.commitOn("feat/base", "base moves on", "")
	f.git("branch", "feat/u")
	f.commitOn("feat/u", "work", "")
	f.own("feat/u")
	for _, b := range []string{"feat/fail", "feat/behind"} {
		f.git("branch", b)
		f.commitOn(b, "work", "")
		f.push(b)
		f.own(b)
	}
	failing := pr(3, "feat/fail", "main", "alice")
	failing.StatusCheckRollup = rollupFailing
	behind := pr(4, "feat/behind", "main", "alice")
	behind.MergeStateStatus = "BEHIND"
	f.forge.prs = []fakePR{behind, failing}
	f.reserve("iss-40", claim.CopyThis, f.now.Add(-3*time.Hour), f.now.Add(time.Hour))
	owed := f.read(true)
	var kinds []OwedKind
	for _, i := range owed.Items {
		if len(kinds) == 0 || kinds[len(kinds)-1] != i.Kind {
			kinds = append(kinds, i.Kind)
		}
	}
	want := OwedKinds()
	if len(kinds) != len(want) {
		t.Fatalf("kinds %v, want each of %v once in order: %s", kinds, want, describe(owed))
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds %v, want %v", kinds, want)
		}
	}
	if next, _ := owed.Next(); next.Kind != OwedRenewal {
		t.Fatalf("the next item is the renewal: %+v", next)
	}
}

// change commits content to file on branch, checking it out and back to
// main, and returns the commit.
func (f *fx) change(branch, file, content string) string {
	f.t.Helper()
	f.git("checkout", "-q", branch)
	f.repo.Write(file, content)
	f.git("add", "--", file)
	f.git("commit", "-q", "-m", "change "+file+" on "+branch)
	c := f.git("rev-parse", "HEAD")
	f.git("checkout", "-q", "main")
	return c
}

// squashLand squash-merges branch into main, pushes main, and deletes and
// prunes the remote branch the branch was pushed to without -u.
func (f *fx) squashLand(branch string) {
	f.t.Helper()
	f.git("push", "-q", "origin", branch) // no -u: no upstream configured
	f.git("merge", "-q", "--squash", branch)
	f.git("commit", "-q", "-m", "squash "+branch)
	f.git("push", "-q", "origin", "main")
	f.git("push", "-q", "origin", "--delete", branch)
	f.git("fetch", "-q", "--prune", "origin")
}

// TestNoUpstreamLandedBranchNotOwed: a branch pushed without -u, then
// squash-merged with its remote branch deleted and pruned, has no upstream
// and no commit in any remote ref, but its work is already in the default
// branch: it is not owed, whether it held several commits (the squash's
// patch matches none of them) or one that main has since edited over.
func TestNoUpstreamLandedBranchNotOwed(t *testing.T) {
	t.Run("multi-commit squash", func(t *testing.T) {
		f := newFx(t)
		f.git("branch", "feat/sq")
		f.change("feat/sq", "a.txt", "one\n")
		f.change("feat/sq", "b.txt", "two\n")
		f.own("feat/sq")
		f.squashLand("feat/sq")
		f.change("main", "c.txt", "later\n")
		f.git("push", "-q", "origin", "main")
		f.git("fetch", "-q", "origin")
		if owed := f.read(false); has(owed, OwedUnpushed, "feat/sq") || strings.Contains(describe(owed), "feat/sq") {
			t.Fatalf("a squash-merged, pruned branch is not owed: %s", describe(owed))
		}
	})
	t.Run("one commit edited over since", func(t *testing.T) {
		f := newFx(t)
		f.git("branch", "feat/one")
		f.change("feat/one", "a.txt", "one\n")
		f.own("feat/one")
		f.squashLand("feat/one")
		f.change("main", "a.txt", "one, edited\n")
		f.git("push", "-q", "origin", "main")
		f.git("fetch", "-q", "origin")
		if owed := f.read(false); has(owed, OwedUnpushed, "feat/one") || strings.Contains(describe(owed), "feat/one") {
			t.Fatalf("a landed commit main has edited over since is not owed: %s", describe(owed))
		}
	})
}

// TestNoUpstreamUnlandedBranchOwed: a no-upstream branch holding a commit
// whose change the default branch lacks is still owed.
func TestNoUpstreamUnlandedBranchOwed(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/new")
	f.change("feat/new", "n.txt", "new\n")
	f.own("feat/new")
	f.change("main", "m.txt", "moved on\n")
	f.git("push", "-q", "origin", "main")
	f.git("fetch", "-q", "origin")
	if owed := f.read(false); !has(owed, OwedUnpushed, "feat/new") {
		t.Fatalf("an unpushed, unlanded branch is owed: %s", describe(owed))
	}
}

// TestNoUpstreamUndecidedBranchNoted: a no-upstream branch whose change
// conflicts with the default branch and whose patch is not upstream cannot
// be told landed or not: it is named in a note, not owed.
func TestNoUpstreamUndecidedBranchNoted(t *testing.T) {
	f := newFx(t)
	f.change("main", "a.txt", "base\n")
	f.git("push", "-q", "origin", "main")
	f.git("branch", "feat/c")
	f.change("feat/c", "a.txt", "theirs\n")
	f.own("feat/c")
	f.change("main", "a.txt", "ours\n")
	f.git("push", "-q", "origin", "main")
	f.git("fetch", "-q", "origin")
	owed := f.read(false)
	if has(owed, OwedUnpushed, "feat/c") {
		t.Fatalf("an undecidable branch is not owed: %s", describe(owed))
	}
	if !strings.Contains(strings.Join(owed.Notes, " "), "feat/c") {
		t.Fatalf("an undecidable branch is named in a note: %s", describe(owed))
	}
}

// TestForgeListingAtLimitTruncated: a listing that returns exactly its limit
// may be truncated, so a note says so and a reservation no listed pull
// request carries is not reported as uncarried; one short of the limit is
// whole, and the reservation is.
func TestForgeListingAtLimitTruncated(t *testing.T) {
	for _, n := range []int{prListLimit, prListLimit - 1} {
		f := newFx(t)
		f.reserveFresh("iss-77")
		for i := 0; i < n; i++ {
			f.forge.prs = append(f.forge.prs, pr(1000+i, fmt.Sprintf("other/%d", i), "main", "bob"))
		}
		owed := f.read(true)
		notes := strings.Join(owed.Notes, " ")
		if n == prListLimit {
			if has(owed, OwedUncarried, "iss-77") || !strings.Contains(notes, "truncated") {
				t.Errorf("a listing at its limit of %d: %s", n, describe(owed))
			}
			continue
		}
		if !has(owed, OwedUncarried, "iss-77") || strings.Contains(notes, "truncated") {
			t.Errorf("a listing of %d, under its limit: %s", n, describe(owed))
		}
	}
}

// manyOwnBranches makes n own branches, each with a commit of the fixture's
// own not in any remote, none with a pull request.
func (f *fx) manyOwnBranches(n int) {
	f.t.Helper()
	for i := 0; i < n; i++ {
		b := fmt.Sprintf("feat/m%02d", i)
		f.git("branch", b)
		f.commitOn(b, "work "+b, "")
		f.own(b)
	}
}

// countGit puts a git on PATH that counts its calls into the returned file.
func countGit(t *testing.T) func() int {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	bin := t.TempDir()
	log := filepath.Join(t.TempDir(), "calls")
	script := "#!/bin/sh\necho x >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() int {
		raw, _ := os.ReadFile(log)
		_ = os.WriteFile(log, nil, 0o600)
		return strings.Count(string(raw), "\n")
	}
}

// TestStackBaseSearchLinear: finding the bases of own branches with no pull
// request (the listing failed) costs git calls in proportion to the
// branches, not to their pairs: doubling the branches far less than
// quadruples the calls.
func TestStackBaseSearchLinear(t *testing.T) {
	calls := map[int]int{}
	for _, n := range []int{10, 20} {
		f := newFx(t)
		f.manyOwnBranches(n)
		f.forge.err = errors.New("listing failed")
		count := countGit(t)
		count()
		f.read(true)
		calls[n] = count()
	}
	if calls[20] > calls[10]*3 {
		t.Fatalf("git calls: %d for 10 branches, %d for 20; want them to grow linearly", calls[10], calls[20])
	}
}

// TestReadOwedHonoursDeadline: a check whose time budget runs out stops
// reading, returns promptly, marks the reading incomplete and says what was
// not read.
func TestReadOwedHonoursDeadline(t *testing.T) {
	f := newFx(t)
	f.manyOwnBranches(30)
	f.forge.err = errors.New("listing failed")
	o := f.opts()
	o.Record = f.awayRecord(false)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	owed, err := ReadOwed(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("the read took %s past a 150ms budget", took)
	}
	if !owed.Incomplete || !strings.Contains(strings.Join(owed.Notes, " "), "ran out of its time budget") {
		t.Fatalf("a read cut short is marked incomplete with a note: incomplete %v, %s", owed.Incomplete, describe(owed))
	}
}

// TestDueRenewalReleasesOnWake: a set of only a renewal already due, as the
// reader produces it (its renewal time past, its lease not yet lapsed), is
// released by a wake that fires before the lease lapses less the skew, as
// the spec's "a renewal only due keeps releasing" says; a wake past the
// lapse does not qualify.
func TestDueRenewalReleasesOnWake(t *testing.T) {
	f := newFx(t)
	f.now = at(t, "2026-10-18 21:01")
	f.reserve("iss-9", claim.CopyThis, f.now.Add(-3*time.Hour), f.now.Add(time.Hour))
	f.forge.prs = []fakePR{pr(1, "fix/iss-9", "main", "alice")}
	rec := f.awayRecord(false)
	o := f.opts()
	o.Record = rec
	owed := f.readWith(o)
	if len(owed.Items) != 1 || owed.Items[0].Kind != OwedRenewal {
		t.Fatalf("want only the renewal due: %s", describe(owed))
	}
	if d := Deadline(owed, f.now); !d.After(f.now) {
		t.Fatalf("the deadline for a due renewal is %s, already past at %s", d, f.now)
	}
	st := SessionState{}
	wake := StopInput{SessionCrons: crons("*/10 * * * *"), Location: london(t)}
	v := Judge(rec, owed, wake, st, f.now)
	if v.Row != 2 {
		t.Fatalf("first stop row %d, want 2", v.Row)
	}
	for i := 0; i < 3; i++ {
		v = Judge(rec, owed, wake, v.State, f.now)
		if v.Block {
			t.Fatalf("wake %d: a due renewal alone blocked (row %d); want it released", i+1, v.Row)
		}
	}
	late := StopInput{SessionCrons: crons("0 23 * * *"), Location: london(t)}
	if v := Judge(rec, owed, late, v.State, f.now); !v.Block {
		t.Errorf("a wake past the lease's lapse released the stop")
	}
}

// TestReviewCheckoutSameEmailNotOwn: on one machine sibling copies share the
// author email, so a branch this copy once checked out to review, which a
// sibling worktree now has checked out, holds commits of "this checkout's"
// email; it is the sibling's, not this copy's own, and is not owed here.
func TestReviewCheckoutSameEmailNotOwn(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/sibling")
	f.commitOn("feat/sibling", "sibling's work", "")
	f.own("feat/sibling") // checked out here once, to look at it
	sib := filepath.Join(t.TempDir(), "sib")
	f.git("worktree", "add", "-q", sib, "feat/sibling")
	if owed := f.read(false); has(owed, OwedUnpushed, "feat/sibling") {
		t.Fatalf("a branch a sibling worktree has checked out is not this copy's own: %s", describe(owed))
	}
}

// TestCarriedOnlyByOwnPullRequest: a reservation is carried only by a pull
// request findOwn would count: in this repository and the person's; a fork's
// or another person's pull request naming it does not carry it.
func TestCarriedOnlyByOwnPullRequest(t *testing.T) {
	for _, c := range []struct {
		name string
		p    fakePR
	}{
		{"another person's", func() fakePR { p := pr(1, "fix/x", "main", "bob"); p.Body = "Resolves: iss-21"; return p }()},
		{"a fork's", func() fakePR {
			p := pr(2, "fix/iss-21", "main", "alice")
			p.IsCrossRepository = true
			return p
		}()},
	} {
		f := newFx(t)
		f.reserveFresh("iss-21")
		f.forge.prs = []fakePR{c.p}
		if owed := f.read(true); !has(owed, OwedUncarried, "iss-21") {
			t.Errorf("%s pull request carried the reservation: %s", c.name, describe(owed))
		}
	}
}
