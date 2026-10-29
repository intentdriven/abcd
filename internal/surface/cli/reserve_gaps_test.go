package cli

import (
	"encoding/json"
	"slices"
	"testing"
)

// TestAReserveThatDropsHeldPathsSaysSoInBothRenders (iss-2610290507512835):
// a reserve that renews this copy's own reservation with a footprint leaving
// held paths out is made, exit 0; the plain render warns on stderr naming
// each, and --json carries them as dropped_footprint.
func TestAReserveThatDropsHeldPathsSaysSoInBothRenders(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	if code, stdout, stderr := reserveRun(t, p, "reserve", "iss-1", "--footprint", "a.go", "--footprint", "b.go", "--footprint", "c.go"); code != 0 {
		t.Fatalf("the reservation: exit %d\n%s%s", code, stdout, stderr)
	}
	code, stdout, stderr := reserveRun(t, p, "reserve", "iss-1", "--footprint", "b.go")
	const warning = `abcd reserve: warning: the footprint leaves out "a.go", "c.go", which the reservation it replaced held, so this reservation no longer holds them; name each again with --footprint to keep it` + "\n"
	if code != 0 || stderr != warning {
		t.Fatalf("the renewal leaving a.go and c.go out: exit %d, stdout %q, stderr %q; want it made, warned on stderr", code, stdout, stderr)
	}
	code, raw, stderr := reserveRun(t, p, "reserve", "iss-1", "--footprint", "d.go", "--footprint", "d.go", "--json")
	var got struct {
		Footprint []string `json:"footprint"`
		Dropped   []string `json:"dropped_footprint"`
	}
	if err := json.Unmarshal([]byte(raw), &got); code != 0 || err != nil || !slices.Equal(got.Dropped, []string{"b.go"}) ||
		!slices.Equal(got.Footprint, []string{"d.go"}) || stderr != "" {
		t.Fatalf("the --json renewal: exit %d, %+v (%v), stderr %q; want dropped_footprint [b.go] and d.go once\n%s", code, got, err, stderr, raw)
	}
}
