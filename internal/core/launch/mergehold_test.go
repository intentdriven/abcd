package launch

import (
	"errors"
	"strings"
	"testing"
)

// The package's tests register no reading (ReadArmedMerges), so none asks
// the real forge, though the suite runs over this repository's own tree,
// whose origin is on GitHub; a test of the row sets its own.

// withArmed sets the row's reading and the checkout's branch for one test.
func withArmed(t *testing.T, a ArmedMerges, err error, branch string) {
	t.Helper()
	prevA, prevB := ReadArmedMerges, currentBranch
	ReadArmedMerges = func(string) (ArmedMerges, error) { return a, err }
	currentBranch = func(string) string { return branch }
	t.Cleanup(func() { ReadArmedMerges, currentBranch = prevA, prevB })
}

func mergeHoldRow(t *testing.T, res suiteResult) GateSummary {
	t.Helper()
	for _, g := range res.Gates {
		if g.Name == gateMergeHold {
			return g
		}
	}
	t.Fatalf("the suite has no %s row: %+v", gateMergeHold, res.Gates)
	return GateSummary{}
}

func mergeHeld(res suiteResult) bool {
	for _, err := range res.errs {
		if errors.Is(err, ErrMergeHeld) {
			return true
		}
	}
	return false
}

// TestTheLaunchCheckRefusesAnUnaccountedArmedMerge: a merge armed or queued
// that the cutting copy's release reservation has not admitted refuses the
// cut, naming it; an admitted one and the release pull request itself pass.
func TestTheLaunchCheckRefusesAnUnaccountedArmedMerge(t *testing.T) {
	withArmed(t, ArmedMerges{Base: "main", Release: "rel-v0.16.0", Admitted: []int{11}, Armed: []ArmedMerge{
		{Number: 11, HeadRef: "fix/a", AutoMerge: true, Method: "SQUASH"},
		{Number: 14, HeadRef: "feat/b", AutoMerge: true, Method: "MERGE"},
		{Number: 15, HeadRef: "release/v0.16.0", Queued: true},
		{Number: 16, HeadRef: "release/v0.16.0", Queued: true, CrossRepository: true},
	}}, nil, "release/v0.16.0")
	res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip})
	row := mergeHoldRow(t, res)
	if row.Status != "ran" || row.Tier != TierHardFail || len(row.Findings) != 2 || !mergeHeld(res) {
		t.Fatalf("row = %+v; want two findings refusing the cut", row)
	}
	if f := row.Findings[0].Detail; !strings.Contains(f, "pull request #14 has auto-merge on (merge) into main, and rel-v0.16.0 has not admitted it") ||
		!strings.Contains(f, "gh pr merge 14 --disable-auto") {
		t.Fatalf("finding = %q", f)
	}
	if f := row.Findings[1].Detail; !strings.Contains(f, "#16 sits in the merge queue") {
		t.Fatalf("a fork's branch of the release's name is judged: %q", f)
	}
	joined := strings.Join(res.Refusals, "\n")
	if !strings.Contains(joined, "merge-hold: pull request #14") || strings.Contains(joined, "#11") || strings.Contains(joined, "#15 ") {
		t.Fatalf("refusals = %v; want #14 and #16 only", res.Refusals)
	}
}

// TestTheLaunchCheckPassesAFrozenBranch: nothing armed but what the
// reservation admitted passes.
func TestTheLaunchCheckPassesAFrozenBranch(t *testing.T) {
	withArmed(t, ArmedMerges{Base: "main", Release: "rel-v0.16.0", Admitted: []int{11},
		Armed: []ArmedMerge{{Number: 11, AutoMerge: true, Method: "SQUASH"}}}, nil, "main")
	res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip})
	if row := mergeHoldRow(t, res); row.Status != "ran" || len(row.Findings) != 0 || mergeHeld(res) {
		t.Fatalf("row = %+v; want it to pass", row)
	}
}

// TestTheLaunchCheckWithoutAReservationNamesTheReservation: a copy that holds
// no release reservation is told to reserve first.
func TestTheLaunchCheckWithoutAReservationNamesTheReservation(t *testing.T) {
	withArmed(t, ArmedMerges{Base: "main", Armed: []ArmedMerge{{Number: 3, Queued: true}}}, nil, "main")
	row := mergeHoldRow(t, runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip}))
	if len(row.Findings) != 1 || !strings.Contains(row.Findings[0].Detail, "this copy holds no release reservation: reserve the release first") {
		t.Fatalf("row = %+v", row)
	}
}

// TestTheLaunchCheckRefusesWhenTheForgeCannotList and reports itself not
// armed off GitHub.
func TestTheLaunchCheckRefusesWhenTheForgeCannotList(t *testing.T) {
	withArmed(t, ArmedMerges{}, errors.New("gh did not answer in time"), "main")
	res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip})
	if row := mergeHoldRow(t, res); len(row.Findings) != 1 || !strings.Contains(row.Findings[0].Detail, "could not be listed (gh did not answer in time)") || !mergeHeld(res) {
		t.Fatalf("row = %+v; want the cut refused", row)
	}
	withArmed(t, ArmedMerges{}, ErrNotOnForge, "main")
	res = runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip})
	if row := mergeHoldRow(t, res); row.Status != "not_armed" || mergeHeld(res) {
		t.Fatalf("row = %+v; want not armed", row)
	}
}

// TestTheLaunchCheckIsNotArmedWithoutAReading: a path that registers no
// reading reports the row not armed and refuses nothing.
func TestTheLaunchCheckIsNotArmedWithoutAReading(t *testing.T) {
	prev := ReadArmedMerges
	ReadArmedMerges = nil
	t.Cleanup(func() { ReadArmedMerges = prev })
	res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip})
	if row := mergeHoldRow(t, res); row.Status != "not_armed" || mergeHeld(res) {
		t.Fatalf("row = %+v; want not armed", row)
	}
}
