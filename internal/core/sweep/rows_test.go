package sweep

import (
	"strings"
	"testing"
)

// TestExitCodes is the exit-code mapping (spc-2610230730269295, Rows): 0 when
// anything was or would be acted on, 1 when the run completed and acted on
// nothing, 2 on a refusal or a fault, whatever rows came before it.
func TestExitCodes(t *testing.T) {
	keep := Row{Kind: RowWorktree, Action: ActionKeep}
	remove := Row{Kind: RowWorktree, Action: ActionRemove}
	salvage := Row{Kind: RowWorktree, Action: ActionSavePatch}
	for _, c := range []struct {
		name string
		res  Result
		want int
	}{
		{"no rows", Result{}, ExitNothing},
		{"only keeps", Result{Rows: []Row{keep, {Kind: RowStray, Action: ActionKeep}, {Kind: RowEmbargoStore, Action: ActionKeep}}}, ExitNothing},
		{"a row with no action is no act", Result{Rows: []Row{keep, {Kind: RowWorktree}}}, ExitNothing},
		{"a removal", Result{Rows: []Row{keep, remove}}, ExitActed},
		{"a salvage", Result{Rows: []Row{salvage, keep}}, ExitActed},
		{"a refusal", Result{Refusal: "the worktree store is a symlink"}, ExitRefused},
		{"a refusal outranks an act", Result{Rows: []Row{remove}, Refusal: "fault"}, ExitRefused},
	} {
		if got := c.res.ExitCode(); got != c.want {
			t.Errorf("%s: ExitCode() = %d; want %d", c.name, got, c.want)
		}
	}
	if ExitActed != 0 || ExitNothing != 1 || ExitRefused != 2 {
		t.Fatalf("the exit codes are 0, 1 and 2: %d %d %d", ExitActed, ExitNothing, ExitRefused)
	}
}

// TestNotesSayEachFailedCheckOnce: the result's notes name the process check
// and the reservation read once each, then the forge note, and nothing when
// every check was made.
func TestNotesSayEachFailedCheckOnce(t *testing.T) {
	if n := (Result{}).Notes(); len(n) != 0 {
		t.Fatalf("Notes() = %q; want none when every check was made", n)
	}
	r := Result{ProcessCheckUnavailable: "lsof timed out after 5s", ReservationsUnread: "no fetch is stamped", ForgeNote: "no forge answered"}
	got := r.Notes()
	want := []string{"process check unavailable: lsof timed out after 5s", "reservations unread: no fetch is stamped", "no forge answered"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("Notes() = %q; want %q", got, want)
	}
}
