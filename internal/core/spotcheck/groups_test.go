package spotcheck

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func row(commit, class string, delivers ...string) LandingRow {
	return LandingRow{Commit: commit, When: at0, PR: 7, Class: class, RuleKind: "drain", RuleID: "adr-2609291342092738",
		RulePath: ".abcd/development/decisions/adrs/2609291342092738-x.md", Mode: "auto", Delivers: delivers, Unattended: true}
}

func sha(c byte) string { return strings.Repeat(string(c), 40) }

func keysOf(cs []candidate) []string {
	var ks []string
	for _, c := range cs {
		ks = append(ks, c.key)
	}
	return ks
}

// TestFourGroupsFromTheirSources: rows classed drain and automatic and the
// hand-backs and decisions in the window make the four groups; a row of any
// other class, person-approved among them, and a record outside the window,
// are never drawn.
func TestFourGroupsFromTheirSources(t *testing.T) {
	from, to := at0.Add(-time.Hour), at0.Add(time.Hour)
	rows := []LandingRow{row(sha('a'), ClassDrain), row(sha('b'), ClassAutomatic, "itd-2610301750162657"), row(sha('c'), ClassAutomatic),
		row(sha('d'), "person-approved"), row(sha('e'), "person-push")}
	hIn, hOut := handBack("iss-1", at0), handBack("iss-2", to.Add(time.Second))
	dIn := gateDecision(9, sha('f'), true, at0)
	groups, err := populations(rows,
		[]namedHandBack{{name: fileOf(t, hIn).name, h: hIn}, {name: fileOf(t, hOut).name, h: hOut}},
		[]namedDecision{{name: fileOf(t, dIn).name, d: dIn}}, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if got := keysOf(groups[GroupDrainLanded]); !slices.Equal(got, []string{sha('a')}) {
		t.Errorf("drain-landed = %v", got)
	}
	if got := keysOf(groups[GroupAutomaticMerges]); !slices.Equal(got, []string{sha('b'), sha('c')}) {
		t.Errorf("automatic-merges = %v", got)
	}
	if got := keysOf(groups[GroupDrainHandBacks]); len(got) != 1 || !strings.HasSuffix(got[0], "-iss-1") {
		t.Errorf("drain-hand-backs = %v", got)
	}
	if got := keysOf(groups[GroupFacilitatorDecisions]); len(got) != 1 || !strings.Contains(got[0], "-pr9-") {
		t.Errorf("facilitator-decisions = %v", got)
	}
	// The roles: the product thinker too on an automatic merge that names an
	// intent; the human facilitator alone on a decision.
	am := groups[GroupAutomaticMerges]
	if !slices.Equal(am[0].roles, []string{RoleFacilitator, RoleProductThinker}) || !slices.Equal(am[1].roles, []string{RoleFacilitator}) {
		t.Errorf("automatic-merges roles = %v, %v", am[0].roles, am[1].roles)
	}
	if r := groups[GroupFacilitatorDecisions][0].roles; !slices.Equal(r, []string{RoleHumanFacilitator}) {
		t.Errorf("facilitator-decisions roles = %v", r)
	}
}

// TestUnreadApprovalRefusesTheDraw: a row whose approval the reader could not
// settle and whose forge answer it could not read refuses the populations.
func TestUnreadApprovalRefusesTheDraw(t *testing.T) {
	r := row(sha('a'), ClassAutomatic)
	r.ForgeUnread = true
	if _, err := populations([]LandingRow{row(sha('b'), ClassDrain), r}, nil, nil, at0, at0); err == nil || !strings.Contains(err.Error(), "forge") {
		t.Fatalf("a row whose forge answer was not read was drawn from (err %v)", err)
	}
}

// TestDriftedRecordIsOneItem: two files of one issue and one time whose
// content drifted are one candidate, the greater name copied.
func TestDriftedRecordIsOneItem(t *testing.T) {
	a, b := handBack("iss-3", at0), handBack("iss-3", at0)
	b.Kind = "question"
	fa, fb := fileOf(t, a), fileOf(t, b)
	if fa.name == fb.name {
		t.Fatal("the two records share a name")
	}
	groups, err := populations(nil, []namedHandBack{{name: fa.name, h: a}, {name: fb.name, h: b}}, nil, at0.Add(-time.Second), at0)
	if err != nil {
		t.Fatal(err)
	}
	hbs := groups[GroupDrainHandBacks]
	if len(hbs) != 1 {
		t.Fatalf("drifted records are %d candidates; want one", len(hbs))
	}
	want := a.Kind
	if fb.name > fa.name {
		want = b.Kind
	}
	if hbs[0].handBack.Kind != want {
		t.Fatalf("the candidate copies the record of kind %s; want the greater name's %s", hbs[0].handBack.Kind, want)
	}
}

// TestItemCopiesItsStateAtTheDraw: a candidate carries the row and the record
// in typed fields alone: no reason, no why, no question, and a field that
// does not keep to its shape dropped.
func TestItemCopiesItsStateAtTheDraw(t *testing.T) {
	r := row(sha('a'), ClassDrain)
	r.Lane, r.RulePath, r.Resolves = "Some Lane words", "/home/x/rule.md", []string{"iss-12", "not an id"}
	h := handBack("iss-4", at0)
	d := Decision{Kind: KindDrainJudgement, At: at0, Issue: "iss-5", Question: "may it?", Answer: "yes", Model: "claude-opus-4-8", Rule: "adr-2609291342092738", Reason: "because it said so"}
	groups, err := populations([]LandingRow{r}, []namedHandBack{{name: fileOf(t, h).name, h: h}}, []namedDecision{{name: fileOf(t, d).name, d: d}}, at0.Add(-time.Second), at0)
	if err != nil {
		t.Fatal(err)
	}
	lr := groups[GroupDrainLanded][0].landing
	if lr.Commit != sha('a') || lr.Lane != "" || lr.RulePath != "" || !slices.Equal(lr.Resolves, []string{"iss-12"}) || lr.RuleID != "adr-2609291342092738" {
		t.Errorf("the landing row is copied as %+v", lr)
	}
	if hb := groups[GroupDrainHandBacks][0].handBack; hb.Reason != "" || hb.Rule != "" || hb.Kind != "stuck" || hb.DrainRule != "adr-2609291342092738" {
		t.Errorf("the hand-back is copied as %+v", hb)
	}
	if dc := groups[GroupFacilitatorDecisions][0].decision; dc.Reason != "" || dc.Question != "" || dc.Answer != "yes" || dc.Model != "claude-opus-4-8" {
		t.Errorf("the decision is copied as %+v", dc)
	}
}
