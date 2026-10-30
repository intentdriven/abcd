package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestReserveOverOnesOtherCopyWarnsInBothRenders (spc-2610161044461448 step
// 11): a person's own `abcd reserve` that shares a path with their
// reservation from another copy is made, exit 0; the plain render warns on
// stderr, naming the path, the other record and the copy it is held from
// (iss-2610290507526253), and --json carries it as own_overlap.
func TestReserveOverOnesOtherCopyWarnsInBothRenders(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	laptop := w.clone(t, "laptop", octoMail)
	if code, stdout, stderr := reserveRun(t, p, "reserve", "iss-1", "--footprint", "internal/x"); code != 0 {
		t.Fatalf("P's reservation: exit %d\n%s%s", code, stdout, stderr)
	}
	code, stdout, stderr := reserveRun(t, laptop, "reserve", "iss-2", "--footprint", "internal/x/a.go")
	const warning = `abcd reserve: warning: "internal/x/a.go" is also in the footprint of iss-1, which you hold from your copy on another computer until `
	if code != 0 || !strings.Contains(stdout, "reserved iss-2 for octo") || !strings.HasPrefix(stderr, warning) ||
		!strings.HasSuffix(stderr, "; reserved anyway, since both are yours\n") || strings.Contains(stdout, "warning") {
		t.Fatalf("the reserve over one's other copy: exit %d, stdout %q, stderr %q; want it made, warned on stderr", code, stdout, stderr)
	}
	if w.remoteTip(t, "iss-2") == "" {
		t.Fatal("the reservation that warned did not reach origin")
	}
	code, raw, stderr := reserveRun(t, laptop, "reserve", "iss-3", "--footprint", "internal/x/b.go", "--json")
	var got struct {
		OwnOverlap []struct {
			Path            string `json:"path"`
			Record          string `json:"record"`
			EffectiveExpiry string `json:"effective_expiry"`
			Copy            string `json:"copy"`
		} `json:"own_overlap"`
	}
	if err := json.Unmarshal([]byte(raw), &got); code != 0 || err != nil || len(got.OwnOverlap) != 1 ||
		got.OwnOverlap[0].Path != "internal/x/b.go" || got.OwnOverlap[0].Record != "iss-1" || got.OwnOverlap[0].EffectiveExpiry == "" ||
		got.OwnOverlap[0].Copy != "your copy on another computer" {
		t.Fatalf("the --json reserve: exit %d, %+v (%v); want own_overlap naming internal/x/b.go in iss-1\n%s%s", code, got.OwnOverlap, err, raw, stderr)
	}
	if stderr != "" {
		t.Fatalf("the --json reserve wrote to stderr: %q", stderr)
	}
}
