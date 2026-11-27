package claim

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestAnUnlapsedReleasePrintsNoLapsedAt (iss-2611252133341605): a release
// whose lease has not lapsed has no lapse instant, so its JSON carries no
// lapsed_at at all. omitempty never drops a zero time, which printed
// 0001-01-01T00:00:00Z, read by a consumer as a lapse at that instant. A
// lapsed release still carries the instant it lapsed.
func TestAnUnlapsedReleasePrintsNoLapsedAt(t *testing.T) {
	lapsedAt := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name string
		v    any
		want bool
	}{
		{"withdrawals of an ended release", ReleaseWithdrawals{Subject: "rel-v0.16.0", Ended: true}, false},
		{"withdrawals of a lapsed release", ReleaseWithdrawals{Subject: "rel-v0.16.0", Lapsed: true, LapsedAt: lapsedAt}, true},
		{"armed merges with no lapsed cut", ArmedMerges{Base: "main"}, false},
		{"armed merges with a lapsed cut", ArmedMerges{Base: "main", Lapsed: "rel-v0.16.0", LapsedAt: lapsedAt}, true},
	} {
		b, err := json.Marshal(c.v)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got := string(b)
		if has := strings.Contains(got, `"lapsed_at"`); has != c.want {
			t.Errorf("%s: lapsed_at present = %v, want %v, in %s", c.name, has, c.want, got)
		}
		if strings.Contains(got, "0001-01-01") {
			t.Errorf("%s: a zero time was printed: %s", c.name, got)
		}
		if c.want && !strings.Contains(got, `"lapsed_at":"2026-11-25T12:00:00Z"`) {
			t.Errorf("%s: the lapse instant is missing: %s", c.name, got)
		}
	}
}
