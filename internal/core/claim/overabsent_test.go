package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// A released branch is deleted at the next sweep (adr-2610161940004937), so a
// drain whose hand-back was still pending when a person released the issue by
// hand can find no branch left to publish it over. A release asked with
// OverAbsent publishes the hand-back as a new branch's first commit, a plain
// push that origin takes only while the branch is still absent, so a session
// that creates it meanwhile wins and nothing of the release lands.

// sweptAfterAHandRelease is P's own reservation of iss-5 released by hand at
// t0, then its branch deleted on origin, as the sweep does.
func sweptAfterAHandRelease(t *testing.T) (w *world, p *participant, blob string) {
	t.Helper()
	w = newWorld(t)
	p = w.clone("1234+octo@users.noreply.github.com", "")
	blob = w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-5", Release: true}); err != nil {
		t.Fatal(err)
	}
	runGit(t, w.bare, "update-ref", "-d", RefPrefix+"iss-5")
	return w, p, blob
}

// TestAHandBackOverAnAbsentBranchIsPublishedAsItsFirstCommit: the release
// pushes one root tombstone marking the hand-back over the record's blob, and
// another person's drain then backs off it while the record is unchanged.
func TestAHandBackOverAnAbsentBranchIsPublishedAsItsFirstCommit(t *testing.T) {
	w, p, blob := sweptAfterAHandRelease(t)
	w.set(t0.Add(26 * time.Hour))
	res, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, HandBackBlob: blob, OverAbsent: true})
	if err != nil || res.Act != ActReleased || res.HandBackUnpublished != "" {
		t.Fatalf("the release over the absent branch = %+v, %v; want the hand-back published", res, err)
	}
	tip := w.remoteTip("iss-5")
	if tip == "" || tip != res.Commit || w.parentOf(tip) != "" {
		t.Fatalf("origin's tip %s (result %s, parent %q); want the release as the branch's first commit", tip, res.Commit, w.parentOf(tip))
	}
	if kind, got := w.handBackOf(tip); len(kind) != 1 || kind[0] != HandBackUserVisible || len(got) != 1 || got[0] != blob {
		t.Fatalf("the tombstone marks %v over %v; want user-visible over %s", kind, got, blob)
	}
	r := w.clone("77+rosa@users.noreply.github.com", "")
	if _, err := r.do(Request{Record: "iss-5", Footprint: []string{"c.go"}, Drain: true}); !errors.Is(err, ErrBackOff) {
		t.Fatalf("R's drain over the published hand-back = %v; want a back-off", err)
	}
	if w.remoteTip("iss-5") != tip {
		t.Fatal("R's drain pushed over the hand-back")
	}
}

// TestAReleaseOverAnAbsentBranchNeedsOverAbsent: without it a release still
// finds nothing to release, and with it a release that publishes no hand-back,
// or names no blob it was judged over, is refused before anything runs.
func TestAReleaseOverAnAbsentBranchNeedsOverAbsent(t *testing.T) {
	w, p, blob := sweptAfterAHandRelease(t)
	if _, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, HandBackBlob: blob}); err == nil ||
		!strings.Contains(err.Error(), "holds no reservation on origin to release") {
		t.Fatalf("a hand-back release over the absent branch without OverAbsent = %v; want nothing to release", err)
	}
	if _, err := p.do(Request{Record: "iss-5", Release: true, OverAbsent: true}); !errors.Is(err, ErrRefused) ||
		!strings.Contains(err.Error(), "only a release that publishes a hand-back") {
		t.Fatalf("a plain release with OverAbsent = %v; want it refused", err)
	}
	if _, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, OverAbsent: true}); !errors.Is(err, ErrRefused) ||
		!strings.Contains(err.Error(), "names the blob it was judged over") {
		t.Fatalf("a hand-back with OverAbsent and no judged blob = %v; want it refused", err)
	}
	if w.remoteTip("iss-5") != "" {
		t.Fatal("a refused release created the branch")
	}
}

// TestAHandBackOverAnAbsentBranchPublishesNothingItMayNot: a record finished
// on the default branch, or changed since the hand-back was judged, is not
// published over, and the branch stays absent.
func TestAHandBackOverAnAbsentBranchPublishesNothingItMayNot(t *testing.T) {
	t.Run("changed", func(t *testing.T) {
		w, p, blob := sweptAfterAHandRelease(t)
		w.fileOnMain("iss-5", "The widget breaks; a person settled it.")
		res, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, HandBackBlob: blob, OverAbsent: true})
		if err != nil || res.Act != ActAlreadyReleased || !strings.Contains(res.HandBackUnpublished, "has changed since the hand-back was judged") {
			t.Fatalf("over a changed record = %+v, %v; want nothing published, and why", res, err)
		}
		if w.remoteTip("iss-5") != "" {
			t.Fatal("the branch was created over a changed record")
		}
	})
	t.Run("finished", func(t *testing.T) {
		w, p, blob := sweptAfterAHandRelease(t)
		// The record moves unchanged, so its blob is still the one judged:
		// only the finished folder keeps the hand-back off.
		w.seed.Git("mv", ".abcd/work/issues/open", ".abcd/work/issues/resolved")
		w.seed.Commit("resolve iss-5")
		w.seed.Git("push", "-q", "origin", "main")
		res, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, HandBackBlob: blob, OverAbsent: true})
		if err != nil || res.Act != ActAlreadyReleased || !strings.Contains(res.HandBackUnpublished, "iss-5 is finished on origin's default branch") {
			t.Fatalf("over a finished record = %+v, %v; want nothing published, and why", res, err)
		}
		if w.remoteTip("iss-5") != "" {
			t.Fatal("the branch was created over a finished record")
		}
	})
}

// TestAHandBackOverAnAbsentBranchLosesToASessionThatCreatesIt: Q reserves
// iss-5 between P's judgement and P's push; origin refuses P's push, which
// cannot fast-forward Q's branch, and Q's reservation stands. A branch Q
// created before P's fetch is left as it stands too.
func TestAHandBackOverAnAbsentBranchLosesToASessionThatCreatesIt(t *testing.T) {
	t.Run("while the push is on its way", func(t *testing.T) {
		w, p, blob := sweptAfterAHandRelease(t)
		q := w.clone("99+quinn@users.noreply.github.com", "")
		var rival string
		beforePush = func(root string) {
			if root != p.dir {
				return
			}
			beforePush = nil
			res, err := q.reserve("iss-5", "b.go")
			if err != nil {
				t.Errorf("Q's reserve: %v", err)
			}
			rival = res.Commit
		}
		t.Cleanup(func() { beforePush = nil })
		_, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, HandBackBlob: blob, OverAbsent: true})
		var nh *NotHeld
		if !errors.As(err, &nh) || nh.Why != NotHeldMoved {
			t.Fatalf("P's release after Q created the branch = %v; want it left as it stands", err)
		}
		if tip := w.remoteTip("iss-5"); tip == "" || tip != rival {
			t.Fatalf("origin's tip %s; want Q's reservation %s", tip, rival)
		}
	})
	t.Run("by this copy before the fetch", func(t *testing.T) {
		// The drain's own person and copy hold it now: the hand-back still
		// does not release that reservation.
		w, p, blob := sweptAfterAHandRelease(t)
		res, err := p.reserve("iss-5", "b.go")
		if err != nil {
			t.Fatal(err)
		}
		_, err = p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, HandBackBlob: blob, OverAbsent: true})
		var nh *NotHeld
		if !errors.As(err, &nh) || nh.Why != NotHeldMoved {
			t.Fatalf("the release over this copy's new reservation = %v; want it left as it stands", err)
		}
		if w.remoteTip("iss-5") != res.Commit {
			t.Fatal("the release pushed over this copy's own reservation")
		}
	})
	t.Run("before the fetch", func(t *testing.T) {
		w, p, blob := sweptAfterAHandRelease(t)
		q := w.clone("99+quinn@users.noreply.github.com", "")
		res, err := q.reserve("iss-5", "b.go")
		if err != nil {
			t.Fatal(err)
		}
		_, err = p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible, HandBackBlob: blob, OverAbsent: true})
		var nh *NotHeld
		if !errors.As(err, &nh) || nh.Why != NotHeldMoved {
			t.Fatalf("P's release over Q's new branch = %v; want it left as it stands", err)
		}
		if w.remoteTip("iss-5") != res.Commit {
			t.Fatal("P's release pushed over Q's reservation")
		}
	})
}
