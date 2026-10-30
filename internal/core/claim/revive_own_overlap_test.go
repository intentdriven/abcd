package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestARevivalJudgesOnesOtherCopyOnTheFilesItsWorkChanged
// (iss-2610261837062463, dcn-2610291352512058): a renew-only act reviving
// one's own lapsed reservation that names the files its work changed
// (Request.OwnOverlapOn) backs off the same person's other copy only where
// that copy holds one of those files; an overlap with that copy on any other
// path goes ahead, named in Result.OwnOverlap, and the reservation keeps its
// whole footprint. An overlap with another person still backs it off on the
// whole footprint, and a revival naming no files is judged as before.
func TestARevivalJudgesOnesOtherCopyOnTheFilesItsWorkChanged(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "itd-82", Footprint: []string{"loop", "docs/a.md"}, Lease: time.Minute}); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(2 * time.Minute))
	if _, err := laptop.reserve("iss-2", "loop/loop.go"); err != nil {
		t.Fatalf("the other copy reserves a path the lapsed reservation named: %v", err)
	}
	tip := w.remoteTip("itd-82")
	w.set(t0.Add(3 * time.Minute))

	_, err := p.do(Request{Record: "itd-82", Renew: true})
	wantBackOff(t, w, err, "itd-99", "loop/loop.go", "iss-2")
	_, err = p.do(Request{Record: "itd-82", Renew: true, OwnOverlapOn: []string{"loop/loop.go"}})
	wantBackOff(t, w, err, "itd-99", "loop/loop.go", "iss-2")
	if got := w.remoteTip("itd-82"); got != tip {
		t.Fatalf("a refused revival moved the branch to %s", got)
	}

	// Another person's reservation still backs it off on the whole footprint.
	if _, err := q.reserve("iss-3", "docs/a.md"); err != nil {
		t.Fatal(err)
	}
	_, err = p.do(Request{Record: "itd-82", Renew: true, OwnOverlapOn: []string{"loop/narrow.go"}})
	wantBackOff(t, w, err, "itd-99", "docs/a.md", "iss-3")
	if _, err := q.do(Request{Record: "iss-3", Release: true}); err != nil {
		t.Fatal(err)
	}

	res, err := p.do(Request{Record: "itd-82", Renew: true, OwnOverlapOn: []string{"loop/narrow.go"}})
	want := []OwnOverlap{{Path: "loop/loop.go", Record: "iss-2", Expiry: t0.Add(2*time.Minute + 2*time.Hour), Copy: "your copy on another computer"}}
	if err != nil || res.Act != ActRenewed || !slices.Equal(res.OwnOverlap, want) {
		t.Fatalf("the revival past one's other copy on a file its work did not change = %+v, %v; want it renewed, naming %+v", res, err, want)
	}
	if strings.Join(res.Footprint, ",") != "loop,docs/a.md" {
		t.Fatalf("the revival's footprint = %v; want its whole footprint kept", res.Footprint)
	}
}

// TestOnlyARenewalNamesTheFilesItsOverlapIsJudgedOn: a reservation that is
// not a renewal naming OwnOverlapOn is refused, and nothing is pushed.
func TestOnlyARenewalNamesTheFilesItsOverlapIsJudgedOn(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	_, err := p.do(Request{Record: "iss-1", Footprint: []string{"a.go"}, OwnOverlapOn: []string{"a.go"}})
	if !errors.Is(err, ErrRefused) || !strings.HasSuffix(err.Error(), nothingPushed) {
		t.Fatalf("a reservation naming OwnOverlapOn = %v; want it refused", err)
	}
	if w.remoteTip("iss-1") != "" {
		t.Fatal("the refused reservation reached origin")
	}
}

// TestAnOtherCopyOverlapReadsEveryHolderOfTheReaders: the overlap a lane
// leaving its landing judges counts the reader's live reservation from
// another copy under their kept id, an id they declare, and their login on
// one made before the id, and, when their id cannot be read here, under the
// id of their own reservation exactly; never another person's, by id or by
// login.
func TestAnOtherCopyOverlapReadsEveryHolderOfTheReaders(t *testing.T) {
	kept := "h_" + strings.Repeat("a", 32)
	reader := Self{ID: kept, Login: "octo", Declared: []string{kept, studioID}}
	until := t0.Add(time.Hour)
	for _, c := range []struct {
		name, holder string
		reader       Self
		want         bool
	}{
		{"under the kept id", kept, reader, true},
		{"under a declared id", studioID, reader, true},
		{"under the login, made before the id", "OCTO", reader, true},
		{"under the own reservation's id, the reader's id not read", kept, Self{}, true},
		{"under another person's id", mayaB, reader, false},
		{"under another person's login", "quinn", reader, false},
	} {
		s := Snapshot{Reservations: []Reservation{
			{Record: "iss-1", Holder: kept, Copy: CopyThis, State: StateLive, Footprint: []string{"loop/a.go"}, EffectiveExpiry: until},
			{Record: "iss-2", Holder: c.holder, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", State: StateLive, Footprint: []string{"loop"}, EffectiveExpiry: until},
		}}
		o, ok := s.OtherCopyOverlap("iss-1", []string{"loop/a.go"}, kept, c.reader)
		if ok != c.want || ok && (o.Path != "loop/a.go" || o.Record != "iss-2") {
			t.Errorf("%s: overlap %+v, %v; want %v", c.name, o, ok, c.want)
		}
	}
}
