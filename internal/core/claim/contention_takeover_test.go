package claim

import (
	"strings"
	"testing"
	"time"
)

// TestContentionUnderATakeOverLetsThisPersonsOtherCopyThrough is
// spc-2610161044461448 step 15's note: under a take-over for octo, octo's own
// reservation of the record from another copy no longer holds it off, so the
// act that takes it over is reached; another person's reservation, an
// unreadable one, and octo's other copy's reservation of another record whose
// footprint the act shares still do.
func TestContentionUnderATakeOverLetsThisPersonsOtherCopyThrough(t *testing.T) {
	now := t0
	until := now.Add(time.Hour)
	s := Snapshot{FetchedAt: now.Add(-time.Minute), Reservations: []Reservation{
		{Record: "iss-1", Holder: "quinn", Footprint: []string{"a.go"}, EffectiveExpiry: until, State: StateLive},
		{Record: "iss-2", Holder: "Octo", Footprint: []string{"b.go"}, EffectiveExpiry: until, State: StateLive},
		{Record: "iss-3", Unreadable: "no trailers", EffectiveExpiry: until, State: StateUnreadable},
		{Record: "iss-4", Holder: "octo", Footprint: []string{"d.go"}, EffectiveExpiry: until, State: StateLive},
	}}
	for _, tc := range []struct {
		id, takeOver string
		footprint    []string
		want         string
	}{
		{"iss-2", "octo", []string{"b.go"}, ""},
		{"iss-2", "", []string{"b.go"}, "iss-2 is reserved by Octo until"},
		{"iss-2", "quinn", []string{"b.go"}, "iss-2 is reserved by Octo until"},
		{"iss-1", "octo", []string{"a.go"}, "iss-1 is reserved by quinn until"},
		{"iss-3", "octo", []string{"c.go"}, "iss-3 has an unreadable reservation holding it until"},
		{"iss-2", "octo", []string{"b.go", "d.go"}, `"d.go" is in the footprint of iss-4, reserved by octo until`},
	} {
		got := s.Contention(tc.id, tc.footprint, now, Self{Login: tc.takeOver})
		if tc.want == "" && got != "" || tc.want != "" && !strings.HasPrefix(got, tc.want) {
			t.Errorf("Contention(%s, %v, take-over %q) = %q; want %q", tc.id, tc.footprint, tc.takeOver, got, tc.want)
		}
	}
}
