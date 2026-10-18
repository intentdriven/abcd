package claim

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The reservation tips a view reads come from origin, written by whoever can
// push to the reservation namespace, so reading them in one batch is parsing
// at a trust boundary (iss-2610180545134593). The batch is a shortcut and
// nothing more: a tip it cannot read cleanly is read on its own, as before,
// with every bound and refusal that read applies, and one bad tip never moves
// another's body under its name.

const (
	shaA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	shaB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	shaC = "cccccccccccccccccccccccccccccccccccccccc"
)

// TestTheCommitBatchIsCutOnlyWhereItReadsAsAsked holds splitCommitBatch to
// the answers it may trust: a body is kept only under the name and size the
// check read for it, a missing object costs that object alone, and the first
// answer it cannot read ends the cut, leaving every later object out to be
// read on its own rather than shifted under another's name.
func TestTheCommitBatchIsCutOnlyWhereItReadsAsAsked(t *testing.T) {
	wants := []batchWant{{shaA, 3}, {shaB, 4}, {shaC, 2}}
	ans := func(name, typ string, body string) string {
		return name + " " + typ + " " + strconv.Itoa(len(body)) + "\n" + body + "\n"
	}
	for _, c := range []struct {
		name string
		out  string
		want map[string]string
	}{
		{"all three", ans(shaA, "commit", "aaa") + ans(shaB, "commit", "bbbb") + ans(shaC, "commit", "cc"),
			map[string]string{shaA: "aaa", shaB: "bbbb", shaC: "cc"}},
		{"one missing in the middle", ans(shaA, "commit", "aaa") + shaB + " missing\n" + ans(shaC, "commit", "cc"),
			map[string]string{shaA: "aaa", shaC: "cc"}},
		{"a size other than the check read", ans(shaA, "commit", "aaa") + ans(shaB, "commit", "bbbbbb") + ans(shaC, "commit", "cc"),
			map[string]string{shaA: "aaa"}},
		{"a type that is not a commit", ans(shaA, "commit", "aaa") + ans(shaB, "blob", "bbbb") + ans(shaC, "commit", "cc"),
			map[string]string{shaA: "aaa"}},
		{"an answer for another object", ans(shaA, "commit", "aaa") + ans(shaC, "commit", "cc") + ans(shaB, "commit", "bbbb"),
			map[string]string{shaA: "aaa"}},
		{"a short read", ans(shaA, "commit", "aaa") + shaB + " commit 4\nbb",
			map[string]string{shaA: "aaa"}},
		{"a body not ended by a newline", ans(shaA, "commit", "aaa") + shaB + " commit 4\nbbbbX" + ans(shaC, "commit", "cc"),
			map[string]string{shaA: "aaa"}},
		{"no answer at all", "", map[string]string{}},
		{"bytes past the last answer", ans(shaA, "commit", "aaa") + ans(shaB, "commit", "bbbb") + ans(shaC, "commit", "cc") + "extra",
			map[string]string{shaA: "aaa", shaB: "bbbb", shaC: "cc"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := splitCommitBatch(c.out, wants); !maps.Equal(got, c.want) {
				t.Errorf("splitCommitBatch = %q; want %q", got, c.want)
			}
		})
	}
}

// TestABatchedViewReadsEveryTipAsItsOwnReadWould: tips of every shape a writer
// can push, a readable one either side of an oversized one, a malformed one
// and a blob, read in one batch read exactly as each reads on its own.
func TestABatchedViewReadsEveryTipAsItsOwnReadWould(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := q.reserve("iss-1", "a"); err != nil {
		t.Fatal(err)
	}
	w.pushHostile("iss-2", "abcd reservation: iss-2\n\n"+strings.Repeat("x", maxObjectBytes+100)+"\n", t0)
	w.pushHostile("iss-3", "abcd reservation: iss-3\n\nAbcd-Reservation: iss-3\nAbcd-Claimed-At: now\n", t0)
	if _, err := q.reserve("iss-5", "b"); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	// A blob as a tip: git refuses to push one to a branch, but a copy's own
	// remote-tracking ref can name anything.
	blob := strings.TrimSpace(p.git("hash-object", "-w", "--", "README.md"))
	p.git("update-ref", TrackingPrefix+"iss-4", blob)
	tree, err := emptyTree(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	for _, id := range []string{"iss-1", "iss-2", "iss-3", "iss-4", "iss-5"} {
		refs = append(refs, TrackingPrefix+id)
	}
	at := t0.Add(time.Minute)
	got := readSeenTips(p.dir, refs, tree, at, false)
	for i, ref := range refs {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		tp, ok, err := readSeen(p.dir, id, ref, tree, at, false)
		want := tipRead{t: tp, ok: ok, err: err}
		if !reflect.DeepEqual(got[i], want) {
			t.Errorf("%s batched = %+v; read on its own = %+v", id, got[i], want)
		}
	}
	if got[1].t.Unreadable != reasonObjectTooBig || got[0].t.Unreadable != "" || got[4].t.Unreadable != "" {
		t.Errorf("want the oversized tip unreadable and the tips either side of it read: %+v", got)
	}
	// The batch, not a fallback, read the commits within the bound; the
	// oversized commit and the blob it left to be read on their own.
	bodies := batchedTipBodies(p.dir, refs)
	for i, inBatch := range []bool{true, false, true, false, true} {
		if _, ok := bodies[refs[i]]; ok != inBatch {
			t.Errorf("%s read in the batch = %v; want %v", refs[i], ok, inBatch)
		}
	}
}

// localTip makes a commit on the empty tree with msg, dated t0, in p's own
// object store, and sets id's remote-tracking reservation ref to it: a copy's
// own tracking ref can name anything, which is all a view reads.
func (p *participant) localTip(id, msg string) string {
	p.w.t.Helper()
	path := filepath.Join(p.w.t.TempDir(), "msg")
	if err := os.WriteFile(path, []byte(msg), 0o600); err != nil {
		p.w.t.Fatal(err)
	}
	tree := strings.TrimSpace(p.git("hash-object", "-w", "-t", "tree", "/dev/null"))
	date := "@" + strconv.FormatInt(t0.Unix(), 10) + " +0000"
	cmd := exec.Command("git", "-C", p.dir, "-c", "user.name=Mal", "-c", "user.email=mal@example.invalid",
		"-c", "commit.gpgsign=false", "commit-tree", tree, "-F", path)
	cmd.Env = append(gittest.Env(p.w.t), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	out, err := cmd.Output()
	if err != nil {
		p.w.t.Fatal(err)
	}
	sha := strings.TrimSpace(string(out))
	p.git("update-ref", TrackingPrefix+id, sha)
	return sha
}

// TestABatchedReadAgreesWithItsOwnReadAtEveryBound: the tips where a batched
// read could part from a read on its own, each read both ways and held
// equal. Messages either side of the message cap, raw and as git trims
// them (a body of the cap plus its newline reads as at the cap, not past it),
// and one ending in trailing blanks;
// commits of exactly the object bound and one byte past it; two refs on one
// commit; and an annotated tag as a tip.
func TestABatchedReadAgreesWithItsOwnReadAtEveryBound(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "a"); err != nil {
		t.Fatal(err)
	}
	var refs []string
	add := func(id string) { refs = append(refs, TrackingPrefix+id) }
	add("iss-1")
	head := func(id string) string { return "abcd reservation: " + id + "\n\nAbcd-Reservation: " + id + "\n" }
	for i, n := range []int{MaxMessageBytes - 1, MaxMessageBytes, MaxMessageBytes + 1, MaxMessageBytes + 2} {
		id := "iss-" + strconv.Itoa(10+i)
		msg := head(id)
		msg += strings.Repeat("x", n-len(msg)-1) + "\n"
		p.localTip(id, msg)
		add(id)
	}
	// A reservation whose message ends in trailing blanks: read as stored, the
	// blanks are trimmed, and it names its holder; read untrimmed, its last
	// trailer would carry them.
	p.localTip("iss-15", "abcd reservation: iss-15\n\nAbcd-Reservation: iss-15\nAbcd-Holder: octo\n\n \t\n")
	add("iss-15")
	// The object bound: a commit of exactly maxObjectBytes, and one past it.
	for i, want := range []int{maxObjectBytes, maxObjectBytes + 1} {
		id := "iss-" + strconv.Itoa(20+i)
		probe := p.localTip(id, head(id))
		size, _ := strconv.Atoi(strings.TrimSpace(p.git("cat-file", "-s", probe)))
		p.localTip(id, head(id)+strings.Repeat("y", want-size))
		if got, _ := strconv.Atoi(strings.TrimSpace(p.git("cat-file", "-s", TrackingPrefix+id))); got != want {
			t.Fatalf("%s is %d bytes; want %d", id, got, want)
		}
		add(id)
	}
	// Two refs on one commit, and an annotated tag as a tip.
	p.git("update-ref", TrackingPrefix+"iss-30", strings.TrimSpace(p.git("rev-parse", TrackingPrefix+"iss-1")))
	add("iss-30")
	p.git("-c", "user.name=Mal", "-c", "user.email=mal@example.invalid", "tag", "-a", "-m", "a tag", "t31", TrackingPrefix+"iss-1")
	p.git("update-ref", TrackingPrefix+"iss-31", strings.TrimSpace(p.git("rev-parse", "t31")))
	add("iss-31")

	tree, err := emptyTree(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	at := t0.Add(time.Minute)
	got := readSeenTips(p.dir, refs, tree, at, false)
	for i, ref := range refs {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		tp, ok, err := readSeen(p.dir, id, ref, tree, at, false)
		if want := (tipRead{t: tp, ok: ok, err: err}); !reflect.DeepEqual(got[i], want) {
			t.Errorf("%s batched = %+v\n read on its own = %+v", id, got[i], want)
		}
	}
	bodies := batchedTipBodies(p.dir, refs)
	for _, id := range []string{"iss-1", "iss-10", "iss-11", "iss-12", "iss-13", "iss-15", "iss-20", "iss-30"} {
		if _, ok := bodies[TrackingPrefix+id]; !ok {
			t.Errorf("%s was not read in the batch; the comparison above did not exercise it", id)
		}
	}
	for _, id := range []string{"iss-21", "iss-31"} {
		if _, ok := bodies[TrackingPrefix+id]; ok {
			t.Errorf("%s was read in the batch; past the bound or not a commit, it is read on its own", id)
		}
	}
}

// TestAViewReadsItsTipsInAFixedNumberOfProcesses: a view of several
// reservations reads their tips in one listing, one check and one batch, not
// three git processes per tip.
func TestAViewReadsItsTipsInAFixedNumberOfProcesses(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	for _, id := range []string{"iss-1", "iss-2", "iss-3", "iss-4"} {
		if _, err := q.reserve(id, "docs/"+id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	log := traced(t)
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Reservations) != 4 {
		t.Fatalf("the view lists %d reservations; want 4", len(snap.Reservations))
	}
	subs := log.subcommands()
	if n := strings.Count(strings.Join(subs, " "), "cat-file"); n != 2 {
		t.Errorf("the view ran %d cat-file processes for 4 tips; want 2, one check and one batch: %q", n, subs)
	}
	if n := len(slices.DeleteFunc(slices.Clone(subs), func(s string) bool { return s != "for-each-ref" })); n > 2 {
		t.Errorf("the view ran %d for-each-ref processes; want at most 2, the listing and the tips' names: %q", n, subs)
	}
}
