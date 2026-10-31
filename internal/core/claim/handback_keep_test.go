package claim

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// A hand-back stands until a person changes the record (spc-2610161044461448
// step 3; dcn-2610161040365515), so a person who reserves the issue over it,
// reads the note and releases it again without changing the record leaves it
// standing: the release's tombstone carries the hand-back's kind and blob
// forward, and no drain on any account judges the issue again.

// handBackOf is the hand-back trailers of commit sha as origin holds it.
func (w *world) handBackOf(sha string) (kind, blob []string) {
	w.t.Helper()
	return w.trailerValues(sha, "Abcd-Handed-Back"), w.trailerValues(sha, "Abcd-Record")
}

// TestAPlainReleaseOverAHandBackKeepsIt: Q's own reserve over P's hand-back,
// renewed, then released plainly, pushes a tombstone carrying the hand-back's
// kind and blob, says it kept it, and R's drain still backs off; once a
// person changes the record, Q's next reserve and release is a plain
// tombstone, and R's drain reserves it.
func TestAPlainReleaseOverAHandBackKeepsIt(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	r := w.clone("77+rosa@users.noreply.github.com", "")
	blob := w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.do(Request{Record: "iss-5", Footprint: []string{"a.go"}, Drain: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible}); err != nil {
		t.Fatal(err)
	}
	if res, err := q.reserve("iss-5", "b.go"); err != nil || res.HandedBack == nil {
		t.Fatalf("Q's reserve over the hand-back = %+v, %v; want it made with the note", res, err)
	}
	w.set(t0.Add(70 * time.Minute))
	if res, err := q.reserve("iss-5", "b.go"); err != nil || res.Act != ActRenewed {
		t.Fatalf("Q's renewal = %+v, %v", res, err)
	}
	rel, err := q.do(Request{Record: "iss-5", Release: true})
	if err != nil || rel.Act != ActReleased {
		t.Fatalf("Q's plain release = %+v, %v", rel, err)
	}
	tomb := w.remoteTip("iss-5")
	if kind, got := w.handBackOf(tomb); !slices.Equal(kind, []string{HandBackUserVisible}) || !slices.Equal(got, []string{blob}) {
		t.Fatalf("Q's tombstone carries hand-back %v over %v; want user-visible over %s, the hand-back kept", kind, got, blob)
	}
	if k := rel.HandBackKept; k == nil || k.Kind != HandBackUserVisible || k.Holder != w.holderIDOf("1234+octo@users.noreply.github.com") || !k.At.Equal(t0) {
		t.Fatalf("the release's kept hand-back = %+v; want P's user-visible hand-back of %s", k, stamp(t0))
	}
	if note := rel.HandBackKept.KeptNote("iss-5"); !strings.Contains(note, "still stands") {
		t.Fatalf("the kept note = %q", note)
	}
	if _, err := r.do(Request{Record: "iss-5", Footprint: []string{"c.go"}, Drain: true}); !errors.Is(err, ErrBackOff) ||
		!strings.Contains(err.Error(), "was handed back by "+w.holderIDOf("99+quinn@users.noreply.github.com")+" as user-visible") {
		t.Fatalf("R's drain over the kept hand-back = %v; want the hand-back's back-off", err)
	}

	w.fileOnMain("iss-5", "The widget breaks; a person settled it: keep the page as it is.")
	if _, err := q.reserve("iss-5", "b.go"); err != nil {
		t.Fatal(err)
	}
	rel, err = q.do(Request{Record: "iss-5", Release: true})
	if err != nil || rel.HandBackKept != nil {
		t.Fatalf("Q's release once the record changed = %+v, %v; want a plain tombstone", rel, err)
	}
	if kind, _ := w.handBackOf(w.remoteTip("iss-5")); kind != nil {
		t.Fatalf("the tombstone over a changed record carries hand-back %v", kind)
	}
	if res, err := r.do(Request{Record: "iss-5", Footprint: []string{"c.go"}, Drain: true}); err != nil || res.Act != ActReserved {
		t.Fatalf("R's drain once the record changed = %+v, %v; want it reserved", res, err)
	}
}

// TestAHandBackIsPublishedOverAPlainTombstone: a release asked to hand back
// over a tip already released by a plain tombstone publishes the hand-back as
// that tombstone's child, fenced on the commit it names: the tombstone itself,
// or the reservation it released; a tip that moved past both is refused with
// nothing pushed, and a tombstone already carrying a hand-back is reported as
// already released.
func TestAHandBackIsPublishedOverAPlainTombstone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	blob := w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	held := w.remoteTip("iss-5")
	if _, err := p.do(Request{Record: "iss-5", Release: true}); err != nil {
		t.Fatal(err)
	}
	plain := w.remoteTip("iss-5")
	res, err := p.do(Request{Record: "iss-5", Release: true, Tip: held, HandBack: HandBackTrustRule})
	if err != nil || res.Act != ActReleased || res.HandBackUnpublished != "" {
		t.Fatalf("the hand-back over the plain tombstone = %+v, %v; want it published", res, err)
	}
	tip := w.remoteTip("iss-5")
	if w.parentOf(tip) != plain {
		t.Fatalf("the hand-back's tombstone %s is not the plain tombstone's child", tip)
	}
	if kind, got := w.handBackOf(tip); !slices.Equal(kind, []string{HandBackTrustRule}) || !slices.Equal(got, []string{blob}) {
		t.Fatalf("the published hand-back = %v over %v", kind, got)
	}
	if res, err := p.do(Request{Record: "iss-5", Release: true, Tip: tip, HandBack: HandBackStuck}); err != nil || res.Act != ActAlreadyReleased ||
		w.remoteTip("iss-5") != tip {
		t.Fatalf("a hand-back over a hand-back = %+v, %v; want it already released, nothing pushed", res, err)
	}

	// The tombstone named itself as the fence.
	if _, err := p.reserve("iss-6", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.fileOnMain("iss-6", "The gadget breaks.")
	if _, err := p.do(Request{Record: "iss-6", Release: true}); err != nil {
		t.Fatal(err)
	}
	plain6 := w.remoteTip("iss-6")
	if res, err := p.do(Request{Record: "iss-6", Release: true, Tip: plain6, HandBack: HandBackStuck}); err != nil || res.Act != ActReleased {
		t.Fatalf("a hand-back fenced on the tombstone itself = %+v, %v", res, err)
	}

	// A tip that moved past the commit the release names is left as it stands.
	if _, err := p.reserve("iss-7", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.fileOnMain("iss-7", "The gizmo breaks.")
	first := w.remoteTip("iss-7")
	w.set(t0.Add(70 * time.Minute))
	if _, err := p.reserve("iss-7", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-7", Release: true}); err != nil {
		t.Fatal(err)
	}
	before := w.remoteTip("iss-7")
	_, err = p.do(Request{Record: "iss-7", Release: true, Tip: first, HandBack: HandBackStuck})
	var nh *NotHeld
	if !errors.As(err, &nh) || nh.Why != NotHeldMoved || w.remoteTip("iss-7") != before {
		t.Fatalf("a hand-back over a tombstone of a later reservation = %v; want NotHeldMoved, nothing pushed", err)
	}
}

// TestAHandBackJudgedOverAnotherBlobIsNotPublished: a release naming the
// blob its hand-back was judged over publishes none once the record on
// origin's default branch is another, and says so.
func TestAHandBackJudgedOverAnotherBlobIsNotPublished(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	judged := w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.fileOnMain("iss-5", "The widget breaks; a person rewrote the remedy.")
	res, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, HandBackBlob: judged})
	if err != nil || res.Act != ActReleased || !strings.Contains(res.HandBackUnpublished, "changed since") {
		t.Fatalf("the release = %+v, %v; want a plain tombstone saying the record changed", res, err)
	}
	if kind, _ := w.handBackOf(w.remoteTip("iss-5")); kind != nil {
		t.Fatalf("the tombstone carries hand-back %v over a record changed since", kind)
	}
}

// TestAHandBackStandsOverARecordOriginHoldsInNoOneFile: a hand-back is lifted
// only by a blob read on origin's default branch that differs from the one it
// names. A record that branch holds in two folders, or only as a file that is
// no record (a symbolic link), names no blob, so the hand-back stands: a
// drain's reservation backs off it, and the view still names it.
func TestAHandBackStandsOverARecordOriginHoldsInNoOneFile(t *testing.T) {
	for name, change := range map[string]func(w *world, rel string){
		"two folders": func(w *world, rel string) {
			data, err := os.ReadFile(filepath.Join(w.seed.Root(), rel))
			if err != nil {
				w.t.Fatal(err)
			}
			w.seed.Write(strings.Replace(rel, "/open/", "/resolved/", 1), string(data)+"\nresolved on a branch, kept open here\n")
		},
		"a symbolic link": func(w *world, rel string) {
			abs := filepath.Join(w.seed.Root(), rel)
			if err := os.Remove(abs); err != nil {
				w.t.Fatal(err)
			}
			if err := os.Symlink("../../../../README.md", abs); err != nil {
				w.t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			q := w.clone("99+quinn@users.noreply.github.com", "")
			w.fileOnMain("iss-5", "The widget breaks.")
			if _, err := p.do(Request{Record: "iss-5", Footprint: []string{"a.go"}, Drain: true}); err != nil {
				t.Fatal(err)
			}
			if _, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackTrustRule}); err != nil {
				t.Fatal(err)
			}
			tomb := w.remoteTip("iss-5")
			change(w, ".abcd/work/issues/open/iss-5-the-widget.md")
			w.seed.Commit("change iss-5's record")
			w.seed.Git("push", "-q", "origin", "main")
			_, err := q.do(Request{Record: "iss-5", Footprint: []string{"b.go"}, Drain: true})
			if !errors.Is(err, ErrBackOff) || w.remoteTip("iss-5") != tomb {
				t.Fatalf("Q's drain over the hand-back = %v; want it to back off, nothing pushed", err)
			}
			snap, err := View(q.dir, w.clock())
			if err != nil {
				t.Fatal(err)
			}
			if why := snap.HandBack("iss-5", w.clock()); !strings.Contains(why, "was handed back by "+p.keptHolderID()+" as trust-rule") {
				t.Fatalf("the view's hand-back = %q; want it standing", why)
			}
		})
	}
}

// TestAHandBackOverAnUnreadDefaultBranchSaysItCouldNotBeRead: a release asked
// to hand back in a checkout that names no default branch of origin pushes a
// plain tombstone, and says the branch could not be read, not that it holds
// no record of the issue.
func TestAHandBackOverAnUnreadDefaultBranchSaysItCouldNotBeRead(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	p.git("symbolic-ref", "-d", "refs/remotes/origin/HEAD")
	p.git("update-ref", "-d", "refs/remotes/origin/main")
	res, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackStuck})
	if err != nil || res.Act != ActReleased {
		t.Fatalf("the release = %+v, %v", res, err)
	}
	if !strings.Contains(res.HandBackUnpublished, "could not be read") || strings.Contains(res.HandBackUnpublished, "holds no record") {
		t.Fatalf("the release says %q; want it to say the default branch could not be read", res.HandBackUnpublished)
	}
}
