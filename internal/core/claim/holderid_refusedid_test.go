package claim

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestADueWithdrawalUnderARefusedHolderIDIsNotPromisedToTheNextAct: with the
// holder id set to no id, or kept in a file the guard refuses, no act can
// run, so the listing names the refusal's class rather than saying the next
// reservation settles the withdrawals made before the holder id.
func TestADueWithdrawalUnderARefusedHolderIDIsNotPromisedToTheNextAct(t *testing.T) {
	for _, c := range []struct {
		name, class string
		setup       func(t *testing.T, p *participant)
	}{
		{"a variable set to no id", HolderIDEnvNotID, func(t *testing.T, _ *participant) { t.Setenv(HolderIDEnv, "not-an-id") }},
		{"a kept file another account can write", HolderIDsWritableByOthers, func(t *testing.T, p *participant) {
			keepHolderID(t, p.home, studioID)
			if err := os.Chmod(abcdhome.Path(p.home, "holder-id"), 0o620); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, _, octo := loginNamedEndedRelease(t)
			c.setup(t, octo)
			rw := listingOf(t, w, octo, "rel-v0.16.0")
			lines := strings.Join(rw.Lines(), "\n")
			if !slices.Equal(numbers(rw.Due), []string{"#11"}) || rw.DueAwaitsHolderID || rw.HolderIDRefused != c.class ||
				!strings.Contains(lines, HolderIDRefusedLine(c.class)) || strings.Contains(lines, "until your next reservation or announcement") {
				t.Fatalf("octo's listing = %+v %q; want #11 due, the refusal named, and no promise of the next act", rw, rw.Lines())
			}
		})
	}
}
