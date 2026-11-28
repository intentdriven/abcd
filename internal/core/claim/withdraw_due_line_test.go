package claim

import (
	"strings"
	"testing"
)

// TestTheDueLineNamesTheActsThatReArm: `abcd status` shows the same due
// line as `abcd peers` and fetches the reservations, yet re-arms nothing, so
// the line names the acts that do (a listing by `abcd peers`, which settles
// after its fetch, and a reservation act, which sweeps every ended release)
// rather than any act that fetches.
func TestTheDueLineNamesTheActsThatReArm(t *testing.T) {
	r := ReleaseWithdrawals{Subject: "rel-v0.16.0", Due: []Withdrawal{{Number: 11}}}
	lines := strings.Join(r.Lines(), "\n")
	if strings.Contains(lines, "act that fetches the reservations") {
		t.Errorf("the due line says any act that fetches the reservations re-arms them, which `abcd status` does not: %q", lines)
	}
	if !strings.Contains(lines, "#11 you withdrew for it are not re-armed yet: your next `abcd peers` or reservation act re-arms them") {
		t.Errorf("the due line does not name the acts that re-arm: %q", lines)
	}
}
