package launch

import (
	"errors"
	"strings"
	"testing"
)

// The package's tests register no reading (ReadArmedMerges), so none asks
// the real forge, though the suite runs over this repository's own tree,
// whose origin is on GitHub; a test of the row sets its own.

const (
	headX = "1111111111111111111111111111111111111111"
	headY = "2222222222222222222222222222222222222222"
)

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
	withArmed(t, ArmedMerges{Base: "main", Release: "rel-v0.16.0", Admitted: map[int]string{11: headX}, Armed: []ArmedMerge{
		{Number: 11, Head: headX, HeadRef: "fix/a", AutoMerge: true, Method: "SQUASH"},
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
	withArmed(t, ArmedMerges{Base: "main", Release: "rel-v0.16.0", Admitted: map[int]string{11: headX},
		Armed: []ArmedMerge{{Number: 11, Head: headX, AutoMerge: true, Method: "SQUASH"}}}, nil, "main")
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

// TestTheLaunchCheckExemptsAnAdmittedMergeOnlyAtItsAdmittedHead: a push to
// an admitted pull request since its admission brings commits nobody
// admitted, and the cut is refused naming both heads.
func TestTheLaunchCheckExemptsAnAdmittedMergeOnlyAtItsAdmittedHead(t *testing.T) {
	withArmed(t, ArmedMerges{Base: "main", Release: "rel-v0.16.0", Admitted: map[int]string{11: headX},
		Armed: []ArmedMerge{{Number: 11, Head: headY, AutoMerge: true, Method: "SQUASH"}}}, nil, "main")
	res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip})
	row := mergeHoldRow(t, res)
	if len(row.Findings) != 1 || !mergeHeld(res) ||
		!strings.Contains(row.Findings[0].Detail, "pull request #11 was admitted into rel-v0.16.0 at 111111111111, but its head is now 222222222222") {
		t.Fatalf("row = %+v; want the moved admission refused", row)
	}
}

// TestTheLaunchCheckRefusesALapsedReleaseAndWarnsOfLostWithdrawals: a
// release reservation that lapsed before the release ended holds nothing and
// refuses the cut; withdrawals a later commit dropped are a warning row.
func TestTheLaunchCheckRefusesALapsedReleaseAndWarnsOfLostWithdrawals(t *testing.T) {
	withArmed(t, ArmedMerges{Base: "main", Lapsed: "rel-v0.16.0", LapsedAt: "2026-10-04T14:00:00Z",
		Lost: []string{"#11 withdrawn for rel-v0.15.0 by octo are lost"}}, nil, "main")
	res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip})
	row := mergeHoldRow(t, res)
	if len(row.Findings) != 1 || !mergeHeld(res) ||
		!strings.Contains(row.Findings[0].Detail, "your release reservation rel-v0.16.0 lapsed at 2026-10-04T14:00:00Z before the release ended") {
		t.Fatalf("row = %+v; want the lapse refused", row)
	}
	var lost *GateSummary
	for i := range res.Gates {
		if res.Gates[i].Name == gateMergeHoldLost {
			lost = &res.Gates[i]
		}
	}
	if lost == nil || lost.Tier != TierWarn || len(lost.Findings) != 1 || !strings.Contains(strings.Join(res.Warnings, "\n"), "#11 withdrawn for rel-v0.15.0 by octo are lost") {
		t.Fatalf("lost row = %+v, warnings %v; want a warning naming them", lost, res.Warnings)
	}
}

// TestTheLaunchCheckNeverPointsAQueuedMergeAtABareDisableAuto: turning
// auto-merge off does not take a pull request out of the merge queue, so a
// queued one is pointed at the dequeue (or --admit), never at a bare
// --disable-auto, and every hand remedy says nothing re-arms it.
func TestTheLaunchCheckNeverPointsAQueuedMergeAtABareDisableAuto(t *testing.T) {
	withArmed(t, ArmedMerges{Base: "main", Release: "rel-v0.16.0", Admitted: map[int]string{12: headX}, Armed: []ArmedMerge{
		{Number: 12, Head: headY, HeadRef: "fix/q", Queued: true},
		{Number: 13, HeadRef: "fix/r", Queued: true},
		{Number: 14, HeadRef: "fix/s", Queued: true, AutoMerge: true, Method: "MERGE"},
	}}, nil, "release/v0.16.0")
	row := mergeHoldRow(t, runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip}))
	if len(row.Findings) != 3 {
		t.Fatalf("row = %+v", row)
	}
	for i, f := range row.Findings {
		if !strings.Contains(f.Detail, "take it out of the merge queue") || !strings.Contains(f.Detail, "nothing re-arms it") {
			t.Errorf("finding %d = %q; want the dequeue named", i, f.Detail)
		}
		if i < 2 && strings.Contains(f.Detail, "--disable-auto") {
			t.Errorf("finding %d = %q; want no bare --disable-auto for a queued pull request", i, f.Detail)
		}
	}
	if !strings.Contains(row.Findings[1].Detail, "--admit 13") || !strings.Contains(row.Findings[2].Detail, "gh pr merge 14 --disable-auto") {
		t.Fatalf("findings = %+v", row.Findings)
	}
}

// TestTheLaunchCheckDoesNotExemptAnUnverifiedAdmission: an admission the
// forge's record of pushes does not yet show the reader pushed is not
// exempt, and the finding says why.
func TestTheLaunchCheckDoesNotExemptAnUnverifiedAdmission(t *testing.T) {
	why := "the forge's record says another account pushed the commit that recorded the admission: check the pull request by hand"
	withArmed(t, ArmedMerges{Base: "main", Release: "rel-v0.16.0", Unverified: map[int]string{11: why},
		Armed: []ArmedMerge{{Number: 11, Head: headX, HeadRef: "fix/a", AutoMerge: true, Method: "SQUASH"}}}, nil, "release/v0.16.0")
	res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip})
	row := mergeHoldRow(t, res)
	if len(row.Findings) != 1 || !mergeHeld(res) || !strings.Contains(row.Findings[0].Detail, "is recorded admitted into rel-v0.16.0, but it is not exempt") ||
		!strings.HasSuffix(row.Findings[0].Detail, why) || strings.Contains(row.Findings[0].Detail, "in a minute") {
		t.Fatalf("row = %+v; want the reason and its own remedy, and no reading again for a verdict that does not change", row)
	}
}

// TestTheLaunchCheckPointsAKeptWithdrawalAtARenewal: a pull request the
// forge refused to withdraw is left armed and recorded kept, and --admit
// refuses a kept one, so the finding names the renewal that withdraws it
// again, never --admit.
func TestTheLaunchCheckPointsAKeptWithdrawalAtARenewal(t *testing.T) {
	withArmed(t, ArmedMerges{Base: "main", Release: "rel-v0.16.0", Kept: map[int]bool{11: true},
		Armed: []ArmedMerge{{Number: 11, Head: headX, HeadRef: "fix/a", AutoMerge: true, Method: "SQUASH"}}}, nil, "release/v0.16.0")
	row := mergeHoldRow(t, runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip}))
	if len(row.Findings) != 1 || !strings.Contains(row.Findings[0].Detail, "the forge refused rel-v0.16.0's withdrawal of it") ||
		!strings.Contains(row.Findings[0].Detail, "reserve rel-v0.16.0 again from this copy (abcd reserve rel-v0.16.0)") ||
		strings.Contains(row.Findings[0].Detail, "--admit") {
		t.Fatalf("row = %+v; want the renewal named, never --admit", row)
	}
}

// TestAVerifyRenderJudgesNoArmedMerge: a render that states MergeHoldSkip
// (`launch archive --verify`) reports the row not armed and refuses nothing,
// even where the forge cannot list the merges; the zero value still refuses.
func TestAVerifyRenderJudgesNoArmedMerge(t *testing.T) {
	withArmed(t, ArmedMerges{}, errors.New("gh: no token"), "main")
	res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip, MergeHold: MergeHoldSkip})
	if row := mergeHoldRow(t, res); row.Status != "not_armed" || mergeHeld(res) {
		t.Fatalf("row = %+v, refused = %v; want not armed and no refusal", row, mergeHeld(res))
	}
	if res := runGateSuite(suiteRequest{RepoRoot: t.TempDir(), Dirty: DirtySkip}); !mergeHeld(res) {
		t.Fatalf("the zero policy must still refuse when the merges cannot be listed: %+v", res.Gates)
	}
}
