package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// The release subject (rel-vX.Y.Z, ended by its tag on origin) and the
// hand-back (a tombstone naming a record's blob on origin's default branch)
// meet in the view and in the reservation's judgement: a release subject is
// derived from the tags and carries no record blob, and a record is derived
// from the default branch's records, whose blob a hand-back is judged by.

// TestOneViewEndsAReleaseByItsTagAndJudgesAHandBackByItsBlob: one listing
// holding a release subject, a handed-back issue and a finished issue shows
// each released by what released it, and the hand-back's record as changed
// only once a person changes it on origin's default branch.
func TestOneViewEndsAReleaseByItsTagAndJudgesAHandBackByItsBlob(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.do(Request{Record: "iss-5", Footprint: []string{"a.go"}, Drain: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.reserve("iss-6", "b.go"); err != nil {
		t.Fatal(err)
	}
	w.finishOnMain("iss-6", "resolved")
	w.tagOrigin("v0.16.0")
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	at := t0.Add(time.Hour)
	snap, err := View(q.dir, at)
	if err != nil {
		t.Fatal(err)
	}
	for id, by := range map[string]string{"rel-v0.16.0": ReleasedByTag, "iss-5": ReleasedByTombstone, "iss-6": ReleasedByDefaultBranch} {
		if r, ok := snap.Lookup(id); !ok || r.State != StateReleased || r.ReleasedBy != by {
			t.Fatalf("%s in the view = %+v; want released by %s", id, r, by)
		}
	}
	if r, _ := snap.Lookup("rel-v0.16.0"); r.HandedBack != "" || r.RecordChanged {
		t.Fatalf("the release subject = %+v; want no hand-back", r)
	}
	if r, _ := snap.Lookup("iss-5"); r.HandedBack != HandBackUserVisible || r.RecordChanged {
		t.Fatalf("the handed-back issue = %+v; want its hand-back standing", r)
	}
	if why := snap.HandBack("iss-5", at); why == "" {
		t.Fatal("the view names no hand-back over iss-5")
	}

	w.fileOnMain("iss-5", "The widget breaks; a person settled it.")
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err = View(q.dir, at)
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := snap.Lookup("iss-5"); !r.RecordChanged || snap.HandBack("iss-5", at) != "" {
		t.Fatalf("the handed-back issue once its record changed = %+v; want the hand-back no longer standing", r)
	}
	if r, _ := snap.Lookup("rel-v0.16.0"); r.ReleasedBy != ReleasedByTag {
		t.Fatalf("the release subject = %+v; want it still released by its tag", r)
	}
}

// TestADerivedReleaseBacksOffSayingWhatReleasedIt: the reservation table's
// derived row names what released the subject: a release's tag on origin, or
// a record finished on origin's default branch.
func TestADerivedReleaseBacksOffSayingWhatReleasedIt(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	for _, c := range []struct {
		req  Request
		want string
	}{
		{Request{Record: "rel-v0.16.0"}, "v0.16.0 is tagged on origin, so there is nothing to reserve; take other work; nothing was pushed"},
		{Request{Record: "iss-5", Footprint: []string{"a.go"}}, "iss-5 is finished on origin's default branch, so there is nothing to reserve; take other work; nothing was pushed"},
	} {
		pr, err := prepare(p.dir, c.req, p.opts())
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, err = decideReserve(pr, judged{derived: true})
		if !errors.Is(err, ErrBackOff) || err.Error() != c.want {
			t.Errorf("reserving %s, released outside its tip = %v; want %q", c.req.Record, err, c.want)
		}
	}
}

// TestAReleaseSubjectIsJudgedByItsTagNotByTheDefaultBranch: a release
// subject's judgement reads its tag, and names no record blob, so a drain's
// reservation of an untagged release is made, and its renewal once tagged is
// told the tag released it.
func TestAReleaseSubjectIsJudgedByItsTagNotByTheDefaultBranch(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.do(Request{Record: "rel-v0.16.0", Drain: true})
	if err != nil || res.Act != ActReserved {
		t.Fatalf("a drain reserving an untagged release = %+v, %v; want it reserved", res, err)
	}
	w.tagOrigin("v0.16.0")
	_, err = p.do(Request{Record: "rel-v0.16.0", Renew: true})
	var nh *NotHeld
	if !errors.As(err, &nh) || !strings.Contains(err.Error(), "the reservation on rel-v0.16.0 is released: v0.16.0 is tagged on origin") {
		t.Fatalf("renewing a tagged release = %v; want it told its tag released it", err)
	}
}
