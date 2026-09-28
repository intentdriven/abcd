package record

import (
	"strings"
	"testing"
)

// readyMember is a planned bundle member the readiness gate passes.
func readyMember(id, slug, specID string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nspec_id: " + specID + "\nkind: bundle-member\nbundle: pair\n---\n\n# " + slug +
		"\n\n## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- **Given** x, **then** y.\n" + recordGrounds
}

const bundleSpec = "---\nid: spc-50\nslug: pair\nintent: itd-30\nintents: [itd-30, itd-31]\nbundle: pair\n---\n# pair\n\nWritten body, ready to build.\n"

// iss-2609261215160156: a bundle's shared spec is described through every
// member it lists, not its `intent:` back-link alone. The links name both
// members, and the readiness move defers to whichever member is not ready.
func TestDescribeBundleSpecReadsEveryMember(t *testing.T) {
	repo := t.TempDir()
	intentFixture(t, repo, "planned", "itd-30", "alpha", readyMember("itd-30", "alpha", "spc-50"))
	// The second member carries no acceptance criteria, so the gate refuses it.
	intentFixture(t, repo, "planned", "itd-31", "beta", strings.Replace(readyMember("itd-31", "beta", "spc-50"), "## Acceptance Criteria\n\n- **Given** x, **then** y.\n", "", 1))
	write(t, repo, ".abcd/development/specs/open/spc-50-pair.md", bundleSpec)

	d, err := Describe(repo, "spc-50")
	if err != nil {
		t.Fatal(err)
	}
	if d.Links["intent"] != "itd-30" || d.Links["intents"] != "itd-30, itd-31" {
		t.Fatalf("a bundle spec's links must name every member: %+v", d.Links)
	}
	moves := strings.Join(d.NextMoves, " ")
	if !strings.Contains(moves, "not ready") || !strings.Contains(moves, "intent ready itd-31") || strings.Contains(moves, "intent ready itd-30") {
		t.Fatalf("the move must defer to the member that is not ready, and to it alone: %v", d.NextMoves)
	}
}

// iss-2609261215160156: after its first member is superseded, a closed bundle
// spec names the member still in force and says the other was passed over —
// never a superseded record as the linked intent.
func TestDescribeClosedBundleSpecPassesOverASupersededMember(t *testing.T) {
	repo := t.TempDir()
	intentFixture(t, repo, "superseded", "itd-30", "alpha",
		"---\nid: itd-30\nslug: alpha\nspec_id: spc-50\nkind: bundle-member\nbundle: null\nbundle_at_supersession: pair\nsuperseded_by: itd-40\nkind_at_supersession: bundle-member\n---\n\n# alpha\n")
	intentFixture(t, repo, "shipped", "itd-31", "beta", strings.Replace(readyMember("itd-31", "beta", "spc-50"), "bundle: pair\n", "bundle: pair\nimpact: additive\n", 1))
	write(t, repo, ".abcd/development/specs/closed/spc-50-pair.md", bundleSpec)

	d, err := Describe(repo, "spc-50")
	if err != nil {
		t.Fatal(err)
	}
	moves := strings.Join(d.NextMoves, " ")
	if !strings.Contains(moves, "itd-31") || !strings.Contains(moves, "itd-30") || !strings.Contains(moves, "superseded") {
		t.Fatalf("the closed page must name the member in force and say the other is superseded: %v", d.NextMoves)
	}
	if strings.Contains(moves, "the linked intent is itd-30") {
		t.Fatalf("a superseded member must not be named as the linked intent: %v", d.NextMoves)
	}
}
