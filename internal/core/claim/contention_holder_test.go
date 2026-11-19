package claim

import (
	"strings"
	"testing"
	"time"
)

// TestContentionExemptsThisCopysReservationOnlyUnderTheReadersOwnHolder is
// iss-2611190011141331: a reservation made from this copy holds nothing off
// only when its holder is the reader's own (Self.Owns), the test crowding
// makes. After the holder id is re-minted, one this copy made under the old id
// is another holder's to the reservation act, which backs off over it, so it
// holds the record and every path it shares off in Contention too; one under
// the reader's own id, kept or declared, still holds nothing off.
func TestContentionExemptsThisCopysReservationOnlyUnderTheReadersOwnHolder(t *testing.T) {
	now := t0
	until := now.Add(time.Hour)
	const (
		old      = "h_00000000000000000000000000000001"
		kept     = "h_00000000000000000000000000000002"
		declared = "h_00000000000000000000000000000003"
	)
	s := Snapshot{FetchedAt: now.Add(-time.Minute), Reservations: []Reservation{
		{Record: "iss-1", Holder: old, Copy: CopyThis, Footprint: []string{"internal/a"}, EffectiveExpiry: until, State: StateLive},
		{Record: "iss-2", Holder: kept, Copy: CopyThis, Footprint: []string{"internal/b"}, EffectiveExpiry: until, State: StateLive},
		{Record: "iss-3", Holder: declared, Copy: CopyThis, Footprint: []string{"internal/c"}, EffectiveExpiry: until, State: StateLive},
	}}
	reader := Self{ID: kept, Declared: []string{declared}}
	for _, tc := range []struct {
		name      string
		id        string
		footprint []string
		reader    Self
		takeOver  bool
		want      string
	}{
		{"a path this copy holds under an old id", "iss-9", []string{"internal/a/x.go"}, reader, false,
			`"internal/a/x.go" is in the footprint of iss-1, reserved by ` + old + " until"},
		{"the same under a take-over", "iss-9", []string{"internal/a/x.go"}, reader, true,
			`"internal/a/x.go" is in the footprint of iss-1, reserved by ` + old + " until"},
		{"the record this copy holds under an old id", "iss-1", []string{"internal/z"}, reader, false,
			"iss-1 is reserved by " + old + " until"},
		{"the same under a take-over", "iss-1", []string{"internal/z"}, reader, true,
			"iss-1 is reserved by " + old + " until"},
		{"a path this copy holds under the kept id", "iss-9", []string{"internal/b/x.go"}, reader, false, ""},
		{"a path this copy holds under a declared id", "iss-9", []string{"internal/c/x.go"}, reader, false, ""},
		{"the record this copy holds under the kept id", "iss-2", []string{"internal/b"}, reader, false, ""},
		{"a reader who cannot be told", "iss-9", []string{"internal/b/x.go"}, Self{}, false,
			`"internal/b/x.go" is in the footprint of iss-2, reserved by ` + kept + " until"},
	} {
		got := s.Contention(tc.id, tc.footprint, now, tc.reader, tc.takeOver)
		if tc.want == "" && got != "" || tc.want != "" && !strings.HasPrefix(got, tc.want) {
			t.Errorf("%s: Contention(%s, %v) = %q; want %q", tc.name, tc.id, tc.footprint, got, tc.want)
		}
	}
}

// TestContentionAgreesWithTheReservationActOverAReMintedHolder: this copy
// reserves iss-1 under one holder id, then its home is emptied and the next
// act mints another; Contention, read for the new holder from the view as
// fetched, holds a reservation sharing iss-1's path off exactly when the
// reservation act backs off, and one under the reader's own id passes alike.
func TestContentionAgreesWithTheReservationActOverAReMintedHolder(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "internal/a"); err != nil {
		t.Fatal(err)
	}
	p.home = t.TempDir() // the home emptied: the next act mints a new id
	if _, err := p.reserve("iss-2", "internal/b"); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	reader := ReaderSelf(p.dir, p.opts())
	if !reader.Known() {
		t.Fatalf("the reader names no holder: %+v", reader)
	}
	for _, c := range []struct {
		id      string
		fp      []string
		contend bool
	}{
		{"iss-3", []string{"internal/a/x.go"}, true},
		{"iss-4", []string{"internal/b/x.go"}, false},
	} {
		why := snap.Contention(c.id, c.fp, w.clock(), reader, false)
		_, err := p.reserve(c.id, c.fp...)
		if (why != "") != c.contend || (why == "") != (err == nil) {
			t.Fatalf("%s %v: Contention = %q, but the reservation act = %v", c.id, c.fp, why, err)
		}
	}
}
