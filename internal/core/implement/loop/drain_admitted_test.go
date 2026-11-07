package loop

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/landings"
)

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
