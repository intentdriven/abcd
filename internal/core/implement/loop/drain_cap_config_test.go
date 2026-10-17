package loop

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestADrainCountsAgainstTheConfiguredCap: with reservation.max_live 12 in
// the person's machine configuration, a drain holding ten does not wait at
// the cap: it counts against the same cap the reservation act judges.
func TestADrainCountsAgainstTheConfiguredCap(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	f.parkIssueLane(t)
	for i := 1; i <= claim.MaxLive-1; i++ {
		id := fmt.Sprintf("iss-26093000000002%02d", i)
		if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: id, Footprint: []string{fmt.Sprintf("other/f%d.go", i)}}, claim.Options{}); err != nil {
			t.Fatalf("Pat's reservation %d: %v", i, err)
		}
	}
	home := os.Getenv("HOME")
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(`{"reservation": {"max_live": 12}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a drain under a raised cap: %v", err)
	}
	st, _, err := readDrain(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if st.Waiting == DrainWaitCap || strings.Contains(res.Next, "live reservations") {
		t.Fatalf("holding ten under a cap of twelve, the drain waited at the cap: %q, next %s", st.Waiting, res.Next)
	}
}
