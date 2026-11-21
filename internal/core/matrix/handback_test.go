package matrix

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// syntheticToken is a GitHub-token-shaped value built at runtime, so no
// literal reaches the tree.
func syntheticToken(seed uint64) string { return "ghp_" + testsecret.Synthetic(seed, 36) }

// handOffIn records copy key's hand-off in rec's run as state, the way the
// Launcher does, with no trial behind it.
func handOffIn(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, key TrialKey, kind TaskKind, state HandOffState, reason string) {
	t.Helper()
	h := HandOff{SchemaVersion: HandOffSchema, Run: rec.RunID, Key: key, Kind: kind, Source: pinOf(t, rec, key.Cell).Source,
		State: state, Brief: "make the copy", Reason: reason}
	if state != HandOffRefused {
		h.Clone = "/store/" + rec.RunID + "/copy_" + strings.Repeat("a", 32)
	}
	data, err := encodeCheck(h)
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	if err := proc.WriteState(handOffRel(rec.RunID, key), data); err != nil {
		t.Fatal(err)
	}
}

// unreturnedOf is the report's account of task's copies on cell in state.
func unreturnedOf(rep Report, task, cell string, state UnreturnedState) *UnreturnedCopies {
	for i, u := range rep.Unreturned {
		if u.Task == task && u.Cell == cell && u.State == state {
			return &rep.Unreturned[i]
		}
	}
	return nil
}

// Every copy the run handed off is accounted for: a host copy never handed
// back, a runner cell's copy waiting for its runner, a refused cell's copy and
// a pilot copy the run's start never reached are each shown, in no rate, and
// each named in the limitations.
func TestTheReportCountsEveryCopyHandedOffThatHasNoTrial(t *testing.T) {
	r, _, m, rec := reportLab(t)
	for rep := 1; rep <= 3; rep++ {
		handOffIn(t, r, m, rec, TrialKey{"feature-a", "host-big", rep}, KindCode, HandOffWaiting, "")
	}
	handOffIn(t, r, m, rec, TrialKey{"feature-a", "host-big", 4}, KindCode, HandOffWaiting, "")
	handOffIn(t, r, m, rec, TrialKey{"feature-a", "or-cheap", 6}, KindCode, HandOffAwaitingRunner, "nothing launches the runner yet")
	handOffIn(t, r, m, rec, TrialKey{"doc-page", "or-cheap", 1}, KindWriting, HandOffRefused, "the clone carries a credential")

	rep, err := BuildReport(r.Root(), m.ID, rec.RunID)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	big := codeCell(t, rep, "host-big")
	if big.PassAt1.Passed != 3 || big.PassAt1.Attempts != 3 || big.NotHandedBack != 1 {
		t.Fatalf("host-big = pass@1 %d of %d, %d not handed back; want 3 of 3 with the fourth copy shown apart",
			big.PassAt1.Passed, big.PassAt1.Attempts, big.NotHandedBack)
	}
	cheap := codeCell(t, rep, "or-cheap")
	if cheap.PassAt1.Attempts != 3 || cheap.AwaitingRunner != 1 {
		t.Fatalf("or-cheap = %d attempts, %d awaiting its runner; want 3 and 1", cheap.PassAt1.Attempts, cheap.AwaitingRunner)
	}
	for _, want := range []struct {
		task, cell string
		state      UnreturnedState
		repeats    []int
	}{
		{"feature-a", "host-big", UnreturnedNotHandedBack, []int{4}},
		{"feature-a", "or-cheap", UnreturnedAwaitingRunner, []int{6}},
		{"doc-page", "or-cheap", UnreturnedRefused, []int{1}},
		{"doc-page", "or-cheap", UnreturnedNeverHandedOff, []int{2, 3}},
		{"fidelity", "host-big", UnreturnedNeverHandedOff, []int{1, 2, 3}},
	} {
		u := unreturnedOf(rep, want.task, want.cell, want.state)
		if u == nil || len(u.Repeats) != len(want.repeats) {
			t.Errorf("the report accounts %s on %s as %s: %+v; want repeats %v", want.task, want.cell, want.state, u, want.repeats)
			continue
		}
		for i := range want.repeats {
			if u.Repeats[i] != want.repeats[i] {
				t.Errorf("%s on %s %s = repeats %v; want %v", want.task, want.cell, want.state, u.Repeats, want.repeats)
			}
		}
	}
	if u := unreturnedOf(rep, "feature-a", "host-big", UnreturnedNeverHandedOff); u != nil {
		t.Errorf("host-big's returned copies are accounted as unreturned: %+v", u)
	}
	all := strings.Join(rep.Limitations, "\n")
	for _, want := range []string{
		"1 copies of feature-a on host-big (repeat 4) were handed to the host and never handed back",
		"wait for the cell's runner",
		"were refused before they started (the clone carries a credential)",
		"never handed off",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the limitations say nothing of %q:\n%s", want, all)
		}
	}
}

// A review or a judging is drawn once, so it waits while a copy of its task is
// handed to the host and not back: a copy handed back later could never join
// it. A runner cell's copy, which nothing makes yet, does not hold it.
func TestADrawWaitsForEveryCopyHandedToTheHost(t *testing.T) {
	r, home, m, rec := reportLab(t)
	handOffIn(t, r, m, rec, TrialKey{"feature-a", "host-big", 4}, KindCode, HandOffWaiting, "")
	handOffIn(t, r, m, rec, TrialKey{"doc-page", "host-big", 2}, KindWriting, HandOffWaiting, "")
	_, err := ReviewRequests(r.Root(), home, m.ID, rec.RunID, "feature-a")
	if err == nil || !strings.Contains(err.Error(), "feature-a/host-big/4") || !strings.Contains(err.Error(), "not handed back") {
		t.Fatalf("ReviewRequests with a copy out = %v; want a refusal naming it", err)
	}
	_, _, err = DrawWriting(r.Root(), home, m.ID, rec.RunID, "doc-page", "good", "bad", 1)
	if err == nil || !strings.Contains(err.Error(), "doc-page/host-big/2") || !strings.Contains(err.Error(), "not handed back") {
		t.Fatalf("DrawWriting with a copy out = %v; want a refusal naming it", err)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	for _, rel := range []string{reviewAskRel(rec.RunID, "feature-a", "asked.json"), judgingRel(rec.RunID, "doc-page", "packet.json")} {
		if _, err := proc.ReadState(rel); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("a refused draw wrote %s: %v", rel, err)
		}
	}
}

// writeJudging screens the packet and the key before it writes them, as a
// trial is screened: a calibration example carrying a credential refuses the
// draw, and nothing is written.
func TestTheJudgingPacketPassesTheSecretScreen(t *testing.T) {
	r, _, m, rec := reportLab(t)
	in := writingInputs(2, 3, 1)
	in.Run = rec.RunID
	for i := range in.Copies {
		in.Copies[i].Key.Cell = []string{"host-big", "or-cheap"}[i/3]
	}
	in.Good = "A good page, keyed " + syntheticToken(21) + ".\n"
	p, k := draw(t, in, seedA)
	if err := writeJudging(r.Root(), m.ID, p, k); err == nil || !strings.Contains(err.Error(), "packet.json") {
		t.Fatalf("writeJudging with a credential in the packet = %v; want the screen's refusal", err)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	if _, err := proc.ReadState(judgingRel(rec.RunID, "doc-page", "packet.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a refused packet was written: %v", err)
	}
}
