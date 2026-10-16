package reservesweep

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The sweep against a real bare origin: what a dry run lists and leaves
// alone, what --apply deletes, the compare-and-swap that spares a branch
// re-reserved after it was judged, the default branch's terminal folders,
// and the claim package's own reservations and releases read as the sweep
// classifies them.

type world struct {
	t     *testing.T
	bare  string
	seed  *gittest.Repo
	home  string
	clock time.Time
}

func newWorld(t *testing.T) *world {
	t.Helper()
	seed := gittest.NewRepo(t)
	seed.Write("README.md", "seed\n")
	seed.Write(".abcd/work/issues/resolved/iss-5-the-widget.md", "---\nid: iss-5\n---\n# The widget\n")
	seed.Write(".abcd/work/issues/open/iss-6-the-gadget.md", "---\nid: iss-6\n---\n# The gadget\n")
	seed.Commit("seed")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	seed.Git("remote", "add", "origin", bare)
	seed.Git("push", "-q", "origin", "main")
	return &world{t: t, bare: bare, seed: seed, home: t.TempDir(), clock: t0}
}

// clone is a working copy with a no-reply identity and its local tier.
func (w *world) clone() string {
	w.t.Helper()
	dir := filepath.Join(w.t.TempDir(), "copy")
	w.seed.Git("clone", "-q", w.bare, dir)
	for _, kv := range [][2]string{{"user.name", "Pat Example"}, {"user.email", "1234+octo@users.noreply.github.com"},
		{"gc.auto", "0"}, {"maintenance.auto", "false"}, {"commit.gpgsign", "false"}} {
		runGit(w.t, dir, nil, "config", kv[0], kv[1])
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		w.t.Fatal(err)
	}
	return dir
}

func runGit(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(gittest.Env(t), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit writes an empty-tree commit in dir dated at, on parent.
func commit(t *testing.T, dir, parent, msg string, at time.Time) string {
	t.Helper()
	tree := runGit(t, dir, nil, "hash-object", "-w", "-t", "tree", "/dev/null")
	args := []string{"commit-tree", tree, "-m", strings.TrimRight(msg, "\n")}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	date := "@" + strconv.FormatInt(at.Unix(), 10) + " +0000"
	return runGit(t, dir, []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}, args...)
}

// put points the record's reservation branch on origin at sha.
func put(t *testing.T, dir, id, sha string) {
	t.Helper()
	runGit(t, dir, nil, "push", "-q", "--force", "origin", sha+":"+claim.RefPrefix+id)
}

func (w *world) tip(id string) string {
	w.t.Helper()
	return runGit(w.t, w.bare, nil, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id)
}

func (w *world) opts(apply bool) Options {
	return Options{Now: func() time.Time { return w.clock }, Apply: apply}
}

func find(t *testing.T, rep Report, id string) Branch {
	t.Helper()
	for _, b := range rep.Branches {
		if b.Record == id {
			return b
		}
	}
	t.Fatalf("the report names no branch for %s: %+v", id, rep.Branches)
	return Branch{}
}

// released writes a reservation and its plain tombstone for id, released at.
func released(t *testing.T, dir, id string, at time.Time) string {
	t.Helper()
	r := commit(t, dir, "", reservationText(id, at.Add(-time.Hour)), at.Add(-time.Hour))
	return commit(t, dir, r, tombstoneText(id, at.Add(-time.Hour), at, ""), at)
}

// TestADryRunListsWhatItWouldDeleteAndDeletesNothing: three branches, one
// spent by its plain release, one spent by its record's resolution, one kept
// by a hand-back; the dry run names each with its reason and every branch is
// still on origin afterwards.
func TestADryRunListsWhatItWouldDeleteAndDeletesNothing(t *testing.T) {
	w := newWorld(t)
	dir := w.clone()
	put(t, dir, "iss-7", released(t, dir, "iss-7", t0))
	put(t, dir, "iss-5", commit(t, dir, "", reservationText("iss-5", t0), t0))
	r := commit(t, dir, "", reservationText("iss-6", t0.Add(-time.Hour)), t0.Add(-time.Hour))
	put(t, dir, "iss-6", commit(t, dir, r, handBackText("iss-6", t0.Add(-time.Hour), t0), t0))
	before := map[string]string{"iss-5": w.tip("iss-5"), "iss-6": w.tip("iss-6"), "iss-7": w.tip("iss-7")}

	w.clock = t0.Add(2 * week)
	rep, err := Sweep(dir, w.opts(false))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Applied || rep.DefaultUnread != "" {
		t.Fatalf("report = %+v; want a dry run that read the default branch", rep)
	}
	for id, want := range map[string]string{"iss-7": SpentReleased, "iss-5": SpentTerminal, "iss-6": KeptHandBack} {
		b := find(t, rep, id)
		if b.Reason != want || b.Spent != (want != KeptHandBack) || b.Outcome != "" {
			t.Fatalf("%s = %+v; want %s and no deletion", id, b, want)
		}
	}
	for id, sha := range before {
		if got := w.tip(id); got != sha {
			t.Fatalf("after a dry run %s's tip is %q; want %s untouched", id, got, sha)
		}
	}
}

// TestApplyDeletesOnlyTheSpentBranches: a spent release is deleted, while a
// recent release and a reservation the claim package holds live stay.
func TestApplyDeletesOnlyTheSpentBranches(t *testing.T) {
	w := newWorld(t)
	dir := w.clone()
	put(t, dir, "iss-7", released(t, dir, "iss-7", t0))
	put(t, dir, "iss-8", released(t, dir, "iss-8", t0.Add(week)))
	w.clock = t0.Add(week + time.Hour)
	if _, err := claim.Reserve(dir, claim.Request{Record: "iss-9", Footprint: []string{"x.go"}},
		claim.Options{Now: func() time.Time { return w.clock }, Home: w.home}); err != nil {
		t.Fatal(err)
	}
	live := w.tip("iss-9")

	w.clock = t0.Add(week + 2*time.Hour)
	rep, err := Sweep(dir, w.opts(true))
	if err != nil {
		t.Fatal(err)
	}
	if b := find(t, rep, "iss-7"); !b.Spent || b.Outcome != Deleted {
		t.Fatalf("iss-7 = %+v; want deleted", b)
	}
	if w.tip("iss-7") != "" {
		t.Fatal("iss-7 is still on origin after --apply")
	}
	for _, id := range []string{"iss-8", "iss-9"} {
		if b := find(t, rep, id); b.Spent || b.Outcome != "" {
			t.Fatalf("%s = %+v; want kept and untouched", id, b)
		}
	}
	if w.tip("iss-9") != live || w.tip("iss-8") == "" {
		t.Fatal("a kept branch moved or went")
	}
	if _, _, deleted := rep.Counts(); deleted != 1 || rep.Failed() {
		t.Fatalf("counts: %d deleted, failed %v; want one and none", deleted, rep.Failed())
	}
}

// TestABranchReReservedAfterItWasJudgedSurvives: between the judgement and
// the deletion another copy reserves the record again; the lease on the
// judged tip refuses the deletion and the new reservation stands.
func TestABranchReReservedAfterItWasJudgedSurvives(t *testing.T) {
	w := newWorld(t)
	dir := w.clone()
	other := w.clone()
	put(t, dir, "iss-7", released(t, dir, "iss-7", t0))
	w.clock = t0.Add(2 * week)
	var moved string
	beforeDelete = func(string) {
		if _, err := claim.Reserve(other, claim.Request{Record: "iss-7", Footprint: []string{"x.go"}},
			claim.Options{Now: func() time.Time { return w.clock }, Home: w.home}); err != nil {
			t.Fatal(err)
		}
		moved = w.tip("iss-7")
	}
	t.Cleanup(func() { beforeDelete = nil })
	rep, err := Sweep(dir, w.opts(true))
	if err != nil {
		t.Fatal(err)
	}
	if b := find(t, rep, "iss-7"); !b.Spent || b.Outcome != Moved {
		t.Fatalf("iss-7 = %+v; want judged spent and its deletion refused as moved", b)
	}
	if got := w.tip("iss-7"); got == "" || got != moved {
		t.Fatalf("iss-7's tip after the sweep = %q; want the new reservation %s", got, moved)
	}
	if rep.Failed() {
		t.Fatal("a refused compare-and-swap counted as a failure")
	}
}

// TestTheDefaultBranchDecidesTerminal: a lapsed reservation never released is
// spent while its record sits in resolved/ on origin's default branch, and
// kept once the record is back in open/ there.
func TestTheDefaultBranchDecidesTerminal(t *testing.T) {
	w := newWorld(t)
	dir := w.clone()
	put(t, dir, "iss-5", commit(t, dir, "", reservationText("iss-5", t0), t0))
	w.clock = t0.Add(2 * week)
	rep, err := Sweep(dir, w.opts(false))
	if err != nil {
		t.Fatal(err)
	}
	if b := find(t, rep, "iss-5"); !b.Spent || b.Reason != SpentTerminal {
		t.Fatalf("iss-5 resolved on main = %+v; want spent as terminal", b)
	}
	w.seed.Git("mv", ".abcd/work/issues/resolved/iss-5-the-widget.md", ".abcd/work/issues/open/iss-5-the-widget.md")
	w.seed.Commit("reopen iss-5")
	w.seed.Git("push", "-q", "origin", "main")
	rep, err = Sweep(dir, w.opts(false))
	if err != nil {
		t.Fatal(err)
	}
	if b := find(t, rep, "iss-5"); b.Spent || b.Reason != KeptNeverRelease {
		t.Fatalf("iss-5 reopened on main = %+v; want kept as never released", b)
	}
}

// TestTheClaimPackagesOwnReleaseReadsAsPlain: a reservation and release made
// through internal/core/claim classify as a reservation and a plain
// tombstone, and the branch is spent a week after the release, not before.
func TestTheClaimPackagesOwnReleaseReadsAsPlain(t *testing.T) {
	w := newWorld(t)
	dir := w.clone()
	o := claim.Options{Now: func() time.Time { return w.clock }, Home: w.home}
	if _, err := claim.Reserve(dir, claim.Request{Record: "itd-9", Footprint: []string{"x.go"}}, o); err != nil {
		t.Fatal(err)
	}
	w.clock = t0.Add(time.Hour)
	if _, err := claim.Reserve(dir, claim.Request{Record: "itd-9", Release: true}, o); err != nil {
		t.Fatal(err)
	}
	w.clock = t0.Add(time.Hour + week)
	rep, err := Sweep(dir, w.opts(false))
	if err != nil {
		t.Fatal(err)
	}
	if b := find(t, rep, "itd-9"); b.Spent || b.Kind != KindPlainTombstone || b.Reason != KeptRecent {
		t.Fatalf("a week after the release, inside the grace = %+v; want a plain tombstone kept as recent", b)
	}
	w.clock = t0.Add(time.Hour + week + claim.TakeoverMargin)
	rep, err = Sweep(dir, w.opts(false))
	if err != nil {
		t.Fatal(err)
	}
	if b := find(t, rep, "itd-9"); !b.Spent || b.Reason != SpentReleased {
		t.Fatalf("past the grace = %+v; want spent as released", b)
	}
}

// TestAnUnknownKindOfTombstoneIsKept: a tombstone carrying a trailer the
// sweep does not know, on a record still open, is kept and the report says
// why.
func TestAnUnknownKindOfTombstoneIsKept(t *testing.T) {
	w := newWorld(t)
	dir := w.clone()
	r := commit(t, dir, "", reservationText("iss-6", t0.Add(-time.Hour)), t0.Add(-time.Hour))
	put(t, dir, "iss-6", commit(t, dir, r, tombstoneText("iss-6", t0.Add(-time.Hour), t0, "Abcd-Released-Kind: handed-over\n"), t0))
	w.clock = t0.Add(10 * week)
	rep, err := Sweep(dir, w.opts(true))
	if err != nil {
		t.Fatal(err)
	}
	if b := find(t, rep, "iss-6"); b.Spent || b.Reason != KeptUnknown || b.Why == "" || b.Outcome != "" {
		t.Fatalf("an unknown kind = %+v; want kept as unknown, with why", b)
	}
	if w.tip("iss-6") == "" {
		t.Fatal("an unknown kind was deleted")
	}
}

// TestAnAgeShorterThanADayIsRefused: no age shorter than the longest lease.
func TestAnAgeShorterThanADayIsRefused(t *testing.T) {
	w := newWorld(t)
	dir := w.clone()
	if _, err := Sweep(dir, Options{Age: time.Hour}); err == nil {
		t.Fatal("an hour's age was taken; want refused")
	}
}
