package claim

import (
	"errors"
	"testing"
	"time"
)

// TestAReleaseNamingItsTipReleasesThatCommitAlone: a release naming the
// reservation commit it took releases it while it is the tip; once this
// person renews it by hand the tip has moved, and the same release is refused
// with NotHeldMoved, nothing pushed. A tip named without a release, or not as
// a full object name, is refused before anything runs.
func TestAReleaseNamingItsTipReleasesThatCommitAlone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	first, err := p.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(10 * time.Minute))
	renewed, err := p.reserve("iss-5", "a.go")
	if err != nil || renewed.Act != ActRenewed {
		t.Fatalf("the renewal by hand = %+v, %v", renewed, err)
	}
	_, err = p.do(Request{Record: "iss-5", Release: true, Tip: first.Commit})
	var nh *NotHeld
	if !errors.As(err, &nh) || nh.Why != NotHeldMoved || !errors.Is(err, ErrRefused) || w.remoteTip("iss-5") != renewed.Commit {
		t.Fatalf("a release naming the moved tip = %v; want NotHeldMoved, the renewal left at the tip", err)
	}
	res, err := p.do(Request{Record: "iss-5", Release: true, Tip: renewed.Commit})
	if err != nil || res.Act != ActReleased || w.parentOf(w.remoteTip("iss-5")) != renewed.Commit {
		t.Fatalf("a release naming the tip = %+v, %v; want a tombstone over %s", res, err, renewed.Commit)
	}
	for _, req := range []Request{{Record: "iss-6", Footprint: []string{"a.go"}, Tip: first.Commit}, {Record: "iss-5", Release: true, Tip: "HEAD"}} {
		if _, err := p.do(req); !errors.Is(err, ErrRefused) {
			t.Fatalf("%+v = %v; want it refused", req, err)
		}
	}
}
