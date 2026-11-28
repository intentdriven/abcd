package cli

import (
	"strings"
	"testing"
)

// TestReserveHelpNamesTheActsThatReArm: once a release's tag is on origin,
// only a reservation act or `abcd peers` re-arms what it withdrew; `abcd
// status` fetches the reservations and re-arms nothing, so the help names
// those acts rather than any act that fetches the reservations.
func TestReserveHelpNamesTheActsThatReArm(t *testing.T) {
	var asJSON bool
	long := strings.Join(strings.Fields(newReserveCommand(&asJSON).Long), " ")
	if strings.Contains(long, "that fetches the reservations") {
		t.Errorf("abcd reserve's help says any act that fetches the reservations re-arms, which abcd status does not")
	}
	if !strings.Contains(long, "the next reservation act or abcd peers of a copy of the same person does so (abcd status re-arms nothing)") {
		t.Errorf("abcd reserve's help does not name the acts that re-arm once the tag is on origin")
	}
}
