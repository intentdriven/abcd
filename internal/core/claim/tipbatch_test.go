package claim

import (
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
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
