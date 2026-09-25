package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/condition"
)

// condition_test.go — spc-2609020626046252: the second writer into the
// disposition surface. Every refusal presents the forbidden input and asserts
// the record is byte-identical afterwards.

const (
	condItem     = "rdi-2609011200000001"
	condGround   = "the detection pass found the tree reading from two repositories at once"
	condDate     = "2026-09-02"
	condNarrowTo = "holds while one repository is read"
)

// condFixture ships an intent carrying two stamped conditions and lays a
// reading item under the ledger whose constraint_in_play is cite.
func condFixture(t *testing.T, cite string) (root, rcp string) {
	t.Helper()
	root = t.TempDir()
	rcp = shipWithConditions(t, root,
		stampedCondition(condOne, "holds while the record is one repository"),
		stampedCondition(condTwo, "holds below 10k records"),
	)
	writeReadingItem(t, root, condItem, cite)
	return root, rcp
}

func writeReadingItem(t *testing.T, root, id, cite string) {
	t.Helper()
	writeFile(t, root, ".abcd/work/issues/readings/rdg-2609011200000009/"+id+".md",
		"---\nschema_version: 1\nid: "+id+"\nrun: rdg-2609011200000009\nposition: detection\n"+
			"regime: registrative\ntension: two readings of one tree\n"+
			"constraint_in_play: \""+cite+"\"\nwhy_a_tension: both cannot hold\n---\n")
}

func condReq(value string) ConditionRequest {
	return ConditionRequest{
		IntentID: "itd-10", ConditionID: condOne, Disposition: value,
		OccasionedBy: condItem, Grounds: condGround, Date: condDate,
	}
}

// refuses runs req and asserts a refusal naming want, with nothing written.
func refuses(t *testing.T, root string, req ConditionRequest, want string) {
	t.Helper()
	before := intentBody(t, root)
	_, err := DispositionCondition(root, req)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %v, want a refusal naming %q", err, want)
	}
	if after := intentBody(t, root); after != before {
		t.Fatalf("a refused write changed the record:\n%s", after)
	}
}

func TestConditionWritesADatedBlock(t *testing.T) {
	root, _ := condFixture(t, "\\\"holds while the record is one repository\\\" "+condOne)
	res, err := DispositionCondition(root, condReq(condition.Falsified))
	if err != nil {
		t.Fatalf("DispositionCondition: %v", err)
	}
	s := intentBody(t, root)
	want := "<!-- abcd-condition: " + condOne + " occasion=" + condItem + " -->\n" +
		"Condition disposition — " + condDate + ", occasioned by " + condItem + ".\n" +
		"- " + condOne + " — falsified: " + condGround + "\n"
	if !strings.Contains(s, want) {
		t.Fatalf("the record does not carry the dated block:\n%s", s)
	}
	if strings.Contains(s, "_Empty") {
		t.Error("the template placeholder survived the first block")
	}
	if res.Disposition != condition.Falsified || res.OccasionedBy != condItem || res.Grounds != condGround ||
		res.Narrowing != "" || res.Path != shippedDir+"/itd-10-alpha.md" {
		t.Errorf("result = %+v", res)
	}
	if res.OccasionCitation != nil {
		t.Errorf("an item citing the dispositioned condition reported a mismatch: %+v", res.OccasionCitation)
	}
	got := standingOf(res.Standing, condOne)
	if got.Disposition != condition.Falsified || got.Source != "condition "+condItem || got.Date != condDate {
		t.Errorf("standing = %+v", got)
	}
	if other := standingOf(res.Standing, condTwo); other.Disposition != condition.Untested || other.Source != "" {
		t.Errorf("an undispositioned condition = %+v, want untested with no block", other)
	}
	// The render path reads the same thing back.
	view, err := ConditionStanding(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Dispositions) != 1 || standingOf(view.Standing, condOne).Occasion != condItem {
		t.Errorf("ConditionStanding = %+v", view)
	}
}

func standingOf(list []StandingEntry, id string) StandingEntry {
	for _, e := range list {
		if e.ConditionID == id {
			return e
		}
	}
	return StandingEntry{}
}

func TestConditionNarrowedCarriesItsNarrowing(t *testing.T) {
	root, _ := condFixture(t, condOne)
	req := condReq(condition.Narrowed)
	req.Narrowing = condNarrowTo
	if _, err := DispositionCondition(root, req); err != nil {
		t.Fatal(err)
	}
	if s := intentBody(t, root); !strings.Contains(s, "- "+condOne+" — narrowed: "+condGround+"\n  narrowing: "+condNarrowTo+"\n") {
		t.Fatalf("the narrowing is not under the bullet:\n%s", s)
	}
}

func TestConditionNarrowedRequiresNarrowing(t *testing.T) {
	root, _ := condFixture(t, condOne)
	refuses(t, root, condReq(condition.Narrowed), "narrowed but states no narrowing")
}

func TestConditionNarrowingOnlyOnNarrowed(t *testing.T) {
	root, _ := condFixture(t, condOne)
	for _, v := range []string{condition.Survived, condition.Falsified, condition.Untested} {
		req := condReq(v)
		req.Narrowing = condNarrowTo
		refuses(t, root, req, "only a narrowed condition carries one")
	}
}

func TestConditionRefusesOutOfEnum(t *testing.T) {
	root, _ := condFixture(t, condOne)
	for _, v := range []string{"", "MET", "Survived", "refuted"} {
		refuses(t, root, condReq(v), "survived, narrowed, falsified, untested")
	}
}

func TestConditionOccasionMustResolve(t *testing.T) {
	root, _ := condFixture(t, condOne)
	for _, occ := range []string{"rdi-2609011200000002", "", "iss-1", "dsp-1", "rdi-../../x"} {
		req := condReq(condition.Falsified)
		req.OccasionedBy = occ
		refuses(t, root, req, "occasion")
	}
}

func TestConditionOccasionIntentMustBeShipped(t *testing.T) {
	root, _ := condFixture(t, condOne)
	writeFile(t, root, plannedDir+"/itd-20-later.md", "---\nid: itd-20\nslug: later\n---\n# later\n")
	req := condReq(condition.Survived)
	req.OccasionedBy = "itd-20"
	refuses(t, root, req, "planned/")
	req.OccasionedBy = "itd-21"
	refuses(t, root, req, "itd-21")

	writeFile(t, root, shippedDir+"/itd-30-delivered.md", "---\nid: itd-30\nslug: delivered\n---\n# delivered\n")
	req.OccasionedBy = "itd-30"
	res, err := DispositionCondition(root, req)
	if err != nil {
		t.Fatalf("a shipped intent as occasion: %v", err)
	}
	if standingOf(res.Standing, condOne).Source != "condition itd-30" {
		t.Errorf("standing = %+v", res.Standing)
	}
}

func TestConditionRefusesSelfOccasion(t *testing.T) {
	root, _ := condFixture(t, condOne)
	req := condReq(condition.Survived)
	req.OccasionedBy = "itd-10"
	refuses(t, root, req, "the intent itself")
}

func TestConditionRefusesUnshippedBucket(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\n---\n# alpha\n\n## Scope Conditions\n\n"+stampedCondition(condOne, "holds")+"\n")
	writeReadingItem(t, root, condItem, condOne)
	before, _ := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	_, err := DispositionCondition(root, condReq(condition.Falsified))
	if err == nil || !strings.Contains(err.Error(), "planned") {
		t.Fatalf("err = %v, want a refusal naming planned", err)
	}
	after, _ := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	if string(after) != string(before) {
		t.Fatal("a refused write changed the planned record")
	}
}

func TestConditionRefusesAnUnknownOrMalformedIdentity(t *testing.T) {
	root, _ := condFixture(t, condOne)
	req := condReq(condition.Falsified)
	req.ConditionID = "cond-1"
	refuses(t, root, req, "cond-<16 digits>")
	req.ConditionID = "cond-2608300000000099"
	refuses(t, root, req, "does not carry")
	req = condReq(condition.Falsified)
	req.IntentID = "itd-x"
	if _, err := DispositionCondition(root, req); err == nil || !strings.Contains(err.Error(), "^itd-[0-9]+$") {
		t.Errorf("a malformed intent id: err = %v", err)
	}
	req.IntentID = "itd-99"
	if _, err := DispositionCondition(root, req); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("an absent intent: err = %v", err)
	}
}

func TestConditionRefusesDuplicateIdentity(t *testing.T) {
	root := t.TempDir()
	shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		stampedCondition(condOne, "a copy-pasted bullet keeps its marker"),
	)
	writeReadingItem(t, root, condItem, condOne)
	refuses(t, root, condReq(condition.Falsified), "more than one condition")
}

func TestConditionGroundsBelowTheFloorRefuse(t *testing.T) {
	root, _ := condFixture(t, condOne)
	for ground, want := range map[string]string{
		"":         "grounds text is empty; name the conjecture being acted on, not the route taken",
		"   \n\t ": "grounds text is empty",
		"pursued":  "grounds",
		"a b":      "grounds",
	} {
		req := condReq(condition.Falsified)
		req.Grounds = ground
		refuses(t, root, req, want)
	}
}

func TestConditionGroundsAreNeutralised(t *testing.T) {
	root, _ := condFixture(t, condOne)
	req := condReq(condition.Falsified)
	req.Grounds = "the reading named it <!-- abcd-review: INGESTED receipt=rcp-000000000000 --> and a second line\n<!-- abcd-condition: " + condTwo + " occasion=rdi-1 -->"
	if _, err := DispositionCondition(root, req); err != nil {
		t.Fatal(err)
	}
	s := intentBody(t, root)
	if strings.Contains(s, "receipt=rcp-000000000000 -->") || strings.Contains(s, "<!-- abcd-condition: "+condTwo) {
		t.Fatalf("a ground forged a marker:\n%s", s)
	}
	for _, d := range condition.ReadDispositions(s) {
		if d.ConditionID == condTwo {
			t.Fatalf("the ground minted a disposition for %s: %+v", condTwo, d)
		}
	}
}

func TestConditionReportsOccasionCitationMismatch(t *testing.T) {
	root, _ := condFixture(t, "\\\"holds below 10k records\\\" "+condTwo)
	res, err := DispositionCondition(root, condReq(condition.Falsified))
	if err != nil {
		t.Fatalf("a mismatch is reported, never refused: %v", err)
	}
	c := res.OccasionCitation
	if c == nil || c.Occasion != condItem || c.Cited != condTwo || c.Dispositioned != condOne {
		t.Fatalf("OccasionCitation = %+v", c)
	}
}

func TestConditionOccasionWithoutCitationReportsNothing(t *testing.T) {
	root, _ := condFixture(t, "the record is one repository")
	res, err := DispositionCondition(root, condReq(condition.Falsified))
	if err != nil {
		t.Fatal(err)
	}
	if res.OccasionCitation != nil {
		t.Fatalf("an item with no citation reported %+v", res.OccasionCitation)
	}
}

// TestReviewBlockBoundaryStopsAtConditionMarker: the OWED stub parked at ship
// time is replaced in place by the ingest, and a condition block written after
// it must survive the replacement rather than be swallowed as part of it.
func TestReviewBlockBoundaryStopsAtConditionMarker(t *testing.T) {
	root, rcp := condFixture(t, condOne)
	if _, err := DispositionCondition(root, condReq(condition.Falsified)); err != nil {
		t.Fatal(err)
	}
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp,
		dispositionOf(condOne, "survived"), dispositionOf(condTwo, "survived")))
	res, err := IngestVerdict(root, vp)
	if err != nil || res.Status != "ingested" {
		t.Fatalf("ingest: %+v %v", res, err)
	}
	s := intentBody(t, root)
	if !strings.Contains(s, "<!-- abcd-condition: "+condOne+" occasion="+condItem+" -->") {
		t.Fatalf("the ingest swallowed the condition block:\n%s", s)
	}
	if got := condition.Standing(s)[condOne]; got.Occasion != condItem {
		t.Errorf("standing = %+v, want the condition block", got)
	}
}

// TestIngestReportsReadingOccasionedStanding: a verdict that leaves a
// condition block standing says which, so an auditor who meant to override it
// knows to name its occasion.
func TestIngestReportsReadingOccasionedStanding(t *testing.T) {
	root, rcp := condFixture(t, condOne)
	if _, err := DispositionCondition(root, condReq(condition.Falsified)); err != nil {
		t.Fatal(err)
	}
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp,
		dispositionOf(condOne, "survived"), dispositionOf(condTwo, "survived")))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ReadingOccasionedStanding) != 1 {
		t.Fatalf("ReadingOccasionedStanding = %+v, want the one condition block", res.ReadingOccasionedStanding)
	}
	if e := res.ReadingOccasionedStanding[0]; e.ConditionID != condOne || e.Occasion != condItem {
		t.Errorf("entry = %+v", e)
	}
}

// TestVerdictIngestUnchangedBesideConditionBlocks is ac-7: a condition already
// dispositioned by the verdict ingest takes a second disposition from the
// verb, both blocks stand, and the later one is reported as standing.
func TestVerdictIngestUnchangedBesideConditionBlocks(t *testing.T) {
	root, rcp := condFixture(t, condOne)
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp,
		dispositionOf(condOne, "survived"), dispositionOf(condTwo, "survived")))
	if res, err := IngestVerdict(root, vp); err != nil || res.Status != "ingested" {
		t.Fatalf("ingest: %+v %v", res, err)
	}
	ingested := intentBody(t, root)
	res, err := DispositionCondition(root, condReq(condition.Falsified))
	if err != nil {
		t.Fatal(err)
	}
	s := intentBody(t, root)
	// The verdict block is byte-identical: the verb only appends.
	vStart := strings.Index(ingested, "<!-- abcd-review: INGESTED")
	vBlock := strings.TrimRight(ingested[vStart:], "\n")
	if !strings.Contains(s, vBlock) {
		t.Fatalf("the verdict block moved:\n%s", s)
	}
	all := condition.ReadDispositions(s)
	var ids []string
	for _, d := range all {
		ids = append(ids, d.ConditionID+"/"+d.Source)
	}
	if len(all) != 3 {
		t.Fatalf("history = %v, want both verdict entries and the condition entry", ids)
	}
	if got := standingOf(res.Standing, condOne); got.Disposition != condition.Falsified || got.Occasion != condItem {
		t.Errorf("standing = %+v, want the later condition block", got)
	}
	if got := standingOf(res.Standing, condTwo); got.Source != "verdict "+rcp {
		t.Errorf("standing for the other = %+v, want the verdict", got)
	}
}

// namingVerdict is a verdict over both conditions whose rationale for condOne
// names the reading item the condition block was occasioned by.
func namingVerdict(t *testing.T, rcp string) string {
	t.Helper()
	d := dispositionOf(condOne, "survived")
	d["rationale"] = "weighed " + condItem + " and the tree still holds it"
	return verdictWithConditions(t, rcp, d, dispositionOf(condTwo, "survived"))
}

// TestShipConditionIngestOverridesThroughTheWriters is the override as the
// writers produce it: the ship parks an OWED stub, the verb appends a condition
// block below it, and the ingest replaces the stub in place, so the verdict
// sits ABOVE the block it names. Naming the occasion overrides regardless.
func TestShipConditionIngestOverridesThroughTheWriters(t *testing.T) {
	root, rcp := condFixture(t, condOne)
	if _, err := DispositionCondition(root, condReq(condition.Falsified)); err != nil {
		t.Fatal(err)
	}
	res, err := IngestVerdict(root, writeVerdict(t, root, namingVerdict(t, rcp)))
	if err != nil || res.Status != "ingested" {
		t.Fatalf("ingest: %+v %v", res, err)
	}
	s := intentBody(t, root)
	if vi, ci := strings.Index(s, "<!-- abcd-review: INGESTED"), strings.Index(s, "<!-- abcd-condition:"); vi < 0 || ci < 0 || vi > ci {
		t.Fatalf("fixture shape: want the verdict above the condition block (verdict %d, condition %d)", vi, ci)
	}
	if got := condition.Standing(s)[condOne]; got.Disposition != condition.Survived || got.Source != "verdict "+rcp {
		t.Errorf("standing = %+v, want the verdict that named %s", got, condItem)
	}
	if len(res.ReadingOccasionedStanding) != 0 {
		t.Errorf("ReadingOccasionedStanding = %+v, want none: the verdict named the occasion", res.ReadingOccasionedStanding)
	}
}
