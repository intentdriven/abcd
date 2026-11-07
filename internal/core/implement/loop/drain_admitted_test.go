package loop

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/landings"
)

// drainRule is the admission the drain fixture's rule record gives its lanes.
var drainRule = Admitted{Kind: landings.KindDrain, Record: "adr-2609300000000001",
	Path: drainrule.ADRsRelDir + "/2609300000000001-drain-rule.md"}

// TestADrainsLaneWithoutAdmissionTakesTheDrainsRule: a drain's lane begun
// before runs recorded their admission takes the drain's rule when the drain
// next moves, which is what the records step's refusal names for it; `abcd
// build <iss-N>` resuming that lane records nothing, since the lane is the
// drain's and not a build's.
func TestADrainsLaneWithoutAdmissionTakesTheDrainsRule(t *testing.T) {
	repo := drainRepo(t)
	res := drainedToLaneOnce(t, repo, DrainOptions{})
	if res.Start == nil {
		t.Fatalf("premise: the drain opened a lane: %+v", res)
	}
	runID := res.Start.RunID
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Admitted = nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	admitted := func() *Admitted {
		t.Helper()
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatal(err)
		}
		return st.Admitted
	}
	if _, err := Start(repo.Root(), eligibleIssue, Options{noReserve: true}); err != nil {
		t.Fatal(err)
	}
	if a := admitted(); a != nil {
		t.Fatalf("`abcd build` resuming a drain's lane records no build admission: %+v", a)
	}
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	if a := admitted(); a == nil || *a != drainRule {
		t.Fatalf("the drain's next move records its rule on its lane: %+v", a)
	}
}

// TestADrainRuleIdSpelledOtherwiseStillAdmits: a rule record whose frontmatter
// spells its id with ordinal zeros or in capitals is the same record, which
// drainrule.Load accepts, and the lanes the drain opens under it are admitted
// under the id in its one spelling rather than refused at their start.
func TestADrainRuleIdSpelledOtherwiseStillAdmits(t *testing.T) {
	repo := drainRepo(t)
	repo.Write(drainRule.Path, "---\nid: ADR-2609300000000001\nslug: drain-rule\nstatus: accepted\ndate: 2026-09-30\n"+
		drainrule.ProposalFrontmatter()+"---\n\n# The drain rule\n")
	repo.Commit("the drain rule's id in capitals")
	res := drainToLane(t, repo, Options{}, DrainOptions{})
	if res.Start == nil {
		t.Fatalf("the drain opens a lane under the record: %+v", res)
	}
	st, err := ReadState(repo.Root(), res.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Admitted == nil || *st.Admitted != drainRule {
		t.Fatalf("the lane is admitted under the rule's canonical id: %+v", st.Admitted)
	}
}

// TestDrainLaneCommitsItsRule (itd-2610301850454860): a lane admitted under the
// drain's rule record commits `Admitted-by: drain <rule record id>` beside its
// `Resolves:` and `Assisted-by:` trailers, from the run's state, and its pull
// request's body carries the same line. TestStartRecordsAdmission holds that the
// drain writes that admission into every lane's state at its start; here the
// landing's records commit reads it back from there.
func TestDrainLaneCommitsItsRule(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validated(t)
	rule := Admitted{Kind: landings.KindDrain, Record: "adr-2609300000000001",
		Path: drainrule.ADRsRelDir + "/2609300000000001-drain-rule.md"}
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Admitted = &rule
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	f.step(t) // prepare
	f.step(t) // records
	l := currentLane(t, f.repo, f.runID)
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	block := trailerBlock(msg)
	at := slices.Index(block, "Admitted-by: drain adr-2609300000000001")
	if at < 0 || !slices.Contains(block[:at], "Resolves: "+f.issue) || !slices.Contains(block[at:], "Assisted-by: Claude:claude-test-5") {
		t.Fatalf("the records commit carries the drain's rule after its resolution and before its disclosure:\n%s", msg)
	}
	if strings.Contains(msg, "Refs:") || strings.Contains(msg, rule.Path) {
		t.Fatalf("a rule record's trailer needs no Refs: line and carries no path:\n%s", msg)
	}
	if a, ok := landings.ParseAdmitted(msg); !ok || a.Kind != landings.KindDrain || a.Record != rule.Record {
		t.Fatalf("the trailer reads back as the drain's rule: %+v %v", a, ok)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if body := prBody(st, l, nil); !strings.Contains(body, "\nAdmitted-by: drain adr-2609300000000001\n") {
		t.Fatalf("the body carries the drain's rule:\n%s", body)
	}
}
