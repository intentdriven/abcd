package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/provenance"
)

const (
	disciplinesDir = ".abcd/development/intents/disciplines"
	supersededDir  = ".abcd/development/intents/superseded"
)

// draftSeeded is a draft in the exact shape CreateFromText seeds: the
// Acceptance Criteria section holds only the placeholder blockquote, no bullets.
func draftSeeded(id, slug string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nspec_id: null\nkind: null\n---\n" +
		"# " + slug + "\n\n## Acceptance Criteria\n\n" +
		"> _Required: add at least one Given-When-Then bullet before planning._\n"
}

// plannedUnlinked is a planned intent whose spec_id is still null (a shape Plan
// never leaves, but the gate must report it rather than trust it).
func plannedUnlinked(id, slug string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nspec_id: null\nkind: standalone\n---\n" +
		"# " + slug + "\n\n## Scope Conditions\n\n" + NullityToken + "\n\n## Acceptance Criteria\n\n- ok\n"
}

// specStub is an open spec still carrying the minted _Draft: placeholder.
func specStub(id, slug, intentID string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nintent: " + intentID + "\n---\n" +
		"# " + slug + "\n\n## Summary\n\n" +
		"_Draft: describe what " + id + " delivers for " + intentID + " — scope, approach._\n"
}

// checkByName finds one check row; the shape contract makes absence a test bug.
func checkByName(t *testing.T, res ReadyResult, name string) ReadyCheck {
	t.Helper()
	for _, c := range res.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("check %q missing from %+v", name, res.Checks)
	return ReadyCheck{}
}

// assertShape enforces the machine-shape contract: always exactly eight rows in
// fixed order, whatever the intent's state.
func assertShape(t *testing.T, res ReadyResult) {
	t.Helper()
	want := []string{"bucket", "acceptance_criteria", "mechanism_claim", "scope_conditions", "spec_link", "spec_body", "steps", "grounds"}
	if len(res.Checks) != len(want) {
		t.Fatalf("expected %d checks, got %d: %+v", len(want), len(res.Checks), res.Checks)
	}
	for i, name := range want {
		if res.Checks[i].Name != name {
			t.Fatalf("check[%d] = %q, want %q", i, res.Checks[i].Name, name)
		}
	}
}

func TestReadyDraftSeededPlaceholder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftSeeded("itd-10", "alpha"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
	if res.Ready {
		t.Fatal("a seeded draft must not be ready")
	}
	bucket := checkByName(t, res, "bucket")
	if bucket.OK {
		t.Fatal("bucket check must fail for drafts")
	}
	if !strings.Contains(bucket.Remedy, "planning interview") {
		t.Fatalf("bucket remedy for a draft without AC must offer the interview, got %q", bucket.Remedy)
	}
	if ac := checkByName(t, res, "acceptance_criteria"); ac.OK {
		t.Fatal("acceptance_criteria must fail on the seeded placeholder (no bullets)")
	}
}

// TestReadyDraftWithRealAC is the motivating incident (itd-93's shape): seeded
// AC bullets look real, but the intent is unplanned — NOT READY, with the
// confirm-then-plan remedy.
func TestReadyDraftWithRealAC(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
	if res.Ready {
		t.Fatal("a draft must never be ready, however complete its AC look")
	}
	bucket := checkByName(t, res, "bucket")
	if bucket.OK || !strings.Contains(bucket.Remedy, "abcd intent plan itd-10") {
		t.Fatalf("bucket check = %+v, want fail with plan remedy", bucket)
	}
	if ac := checkByName(t, res, "acceptance_criteria"); !ac.OK {
		t.Fatalf("acceptance_criteria = %+v, want OK (bullets present)", ac)
	}
	if sb := checkByName(t, res, "spec_body"); sb.OK {
		t.Fatal("spec_body must not pass with no linked spec")
	}
}

func TestReadyPlannedNullSpecNoClaimer(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedUnlinked("itd-10", "alpha"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
	if res.Ready {
		t.Fatal("planned with no spec must not be ready")
	}
	link := checkByName(t, res, "spec_link")
	if link.OK || !strings.Contains(link.Detail, "no spec realises") {
		t.Fatalf("spec_link = %+v, want fail naming the missing spec", link)
	}
}

func TestReadyPlannedNullSpecOneSidedClaimer(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedUnlinked("itd-10", "alpha"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	link := checkByName(t, res, "spec_link")
	if link.OK || !strings.Contains(link.Remedy, "abcd intent link itd-10 spc-1") {
		t.Fatalf("spec_link = %+v, want the link remedy for a one-sided claimer", link)
	}
}

func TestReadyPlannedStubSpecBody(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specStub("spc-1", "alpha", "itd-10"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
	if res.Ready {
		t.Fatal("a stub spec body must block readiness")
	}
	if link := checkByName(t, res, "spec_link"); !link.OK {
		t.Fatalf("spec_link = %+v, want OK", link)
	}
	body := checkByName(t, res, "spec_body")
	if body.OK || !strings.Contains(body.Remedy, "write the spec body") {
		t.Fatalf("spec_body = %+v, want fail with write-the-body remedy", body)
	}
}

func TestReadyGreen(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
	if !res.Ready {
		t.Fatalf("planned+linked+written must be ready: %+v", res.Checks)
	}
	for _, c := range res.Checks {
		if !c.OK {
			t.Fatalf("check %s failed on the green path: %+v", c.Name, c)
		}
	}
	if res.Bucket != BucketPlanned || res.SpecID != "spc-1" {
		t.Fatalf("result header = %+v", res)
	}
}

func TestReadyBidirectionalDrift(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-99"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatalf("drift is a report, not a fault: %v", err)
	}
	link := checkByName(t, res, "spec_link")
	if link.OK || !strings.Contains(link.Detail, "itd-99") {
		t.Fatalf("spec_link = %+v, want fail naming the disagreeing claim", link)
	}
}

func TestReadyTerminalBuckets(t *testing.T) {
	tests := []struct {
		dir, wantDetail string
	}{
		{shippedDir, "shipped"},
		{disciplinesDir, "discipline"},
		{supersededDir, "superseded"},
	}
	for _, tt := range tests {
		t.Run(tt.wantDetail, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, tt.dir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))

			res, err := Ready(root, "itd-10")
			if err != nil {
				t.Fatal(err)
			}
			assertShape(t, res)
			if res.Ready {
				t.Fatalf("%s must not be ready", tt.wantDetail)
			}
			bucket := checkByName(t, res, "bucket")
			if bucket.OK || !strings.Contains(bucket.Detail, tt.wantDetail) {
				t.Fatalf("bucket = %+v, want fail mentioning %q", bucket, tt.wantDetail)
			}
			if bucket.Remedy != "" {
				t.Fatalf("a terminal bucket has no remedy, got %q", bucket.Remedy)
			}
		})
	}
}

func TestReadyFaults(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))

	if _, err := Ready(root, "itd-../etc"); err == nil {
		t.Fatal("malformed id must be a fault")
	}
	if _, err := Ready(root, "itd-999"); err == nil {
		t.Fatal("unknown intent must be a fault")
	}

	// A symlinked intent record violates the store trust boundary.
	linkRoot := t.TempDir()
	target := filepath.Join(linkRoot, "outside.md")
	if err := os.WriteFile(target, []byte(draftWithAC("itd-7", "gamma")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(linkRoot, draftsDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(linkRoot, draftsDir, "itd-7-gamma.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Ready(linkRoot, "itd-7"); err == nil {
		t.Fatal("a symlinked intent record must be a fault")
	}
}

// TestReadySpecLinkToleratesSpecIDSpelling mirrors Reconcile's tolerance in the
// report: the spec_link check reads the same stored spec_id, so a lint-green
// slug-suffixed or zero-padded value must report the link as held, not missing.
func TestReadySpecLinkToleratesSpecIDSpelling(t *testing.T) {
	for _, specID := range []string{"spc-1-alpha", "spc-01"} {
		t.Run(specID, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", specID))
			writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

			res, err := Ready(root, "itd-10")
			if err != nil {
				t.Fatal(err)
			}
			if link := checkByName(t, res, "spec_link"); !link.OK {
				t.Fatalf("spec_link = %+v, want OK for the lint-green spec_id %q", link, specID)
			}
		})
	}
}

// TestReadyChecksOrderAndCount pins the machine-shape contract itself: the two
// claim checks report between the criterion claim and the spec link, and every
// row is present whatever the record says.
func TestReadyChecksOrderAndCount(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
}

// plannedWithClaims is a planned, linked intent whose claim sections carry the
// given bodies verbatim — the fixture the five gradient cases vary.
func plannedWithClaims(mechanism, conditions string) string {
	return "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n" +
		mechanism + conditions + "## Acceptance Criteria\n\n- ok\n" + groundsSection
}

// readyWithClaims runs the gate over a planned record carrying the given claim
// sections, against a written spec, so only the claim checks can fail.
func readyWithClaims(t *testing.T, mechanism, conditions string) ReadyResult {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedWithClaims(mechanism, conditions))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestReadyScopeConditionsAbsent is itd-177's first criterion: no conditions and
// no explicit nullity is reported, naming the missing field. The row is advisory
// (iss-2609091009111294), so the report names it and the verdict ignores it.
func TestReadyScopeConditionsAbsent(t *testing.T) {
	res := readyWithClaims(t, "", "")
	if !res.Ready {
		t.Fatal("an advisory check must not withhold readiness: an intent with no context claim must not be ready is reported, not gated")
	}
	c := checkByName(t, res, "scope_conditions")
	if c.OK || !strings.Contains(c.Detail, "## Scope Conditions") {
		t.Fatalf("scope_conditions = %+v, want fail naming the section", c)
	}
	if !strings.Contains(c.Remedy, NullityToken) {
		t.Fatalf("remedy must name the nullity token, got %q", c.Remedy)
	}
}

// TestReadyScopeConditionsEmptyFails: a heading with nothing under it is the
// gate fault the gradient distinguishes from an absent section.
func TestReadyScopeConditionsEmptyFails(t *testing.T) {
	res := readyWithClaims(t, "", "## Scope Conditions\n\n")
	if c := checkByName(t, res, "scope_conditions"); c.OK || !strings.Contains(c.Detail, "empty") {
		t.Fatalf("scope_conditions = %+v, want fail on the empty section", c)
	}
}

// TestReadyScopeConditionsNullityPasses: the recorded decline is a pass.
func TestReadyScopeConditionsNullityPasses(t *testing.T) {
	res := readyWithClaims(t, "", "## Scope Conditions\n\n"+NullityToken+"\n\n")
	c := checkByName(t, res, "scope_conditions")
	if !c.OK || !strings.Contains(c.Detail, "nullity recorded") {
		t.Fatalf("scope_conditions = %+v, want OK reporting the recorded nullity", c)
	}
	if !res.Ready {
		t.Fatalf("a recorded nullity must leave the intent ready: %+v", res.Checks)
	}
}

// TestReadyMechanismNullityPasses is itd-177's third criterion.
func TestReadyMechanismNullityPasses(t *testing.T) {
	res := readyWithClaims(t, "## Mechanism\n\n"+NullityToken+"\n\n", "## Scope Conditions\n\n"+NullityToken+"\n\n")
	c := checkByName(t, res, "mechanism_claim")
	if !c.OK || !strings.Contains(c.Detail, "declined (nullity recorded)") {
		t.Fatalf("mechanism_claim = %+v, want OK reporting the recorded nullity", c)
	}
	if !res.Ready {
		t.Fatalf("a declined mechanism must leave the intent ready: %+v", res.Checks)
	}
}

// TestReadyMechanismEmptyFails is itd-177's fourth criterion: present but empty
// exits non-zero and names the section — write the claim or the token.
func TestReadyMechanismEmptyFails(t *testing.T) {
	res := readyWithClaims(t, "## Mechanism\n\n", "## Scope Conditions\n\n"+NullityToken+"\n\n")
	if !res.Ready {
		t.Fatal("an advisory check must not withhold readiness: an empty mechanism section must not be ready is reported, not gated")
	}
	c := checkByName(t, res, "mechanism_claim")
	if c.OK || !strings.Contains(c.Detail, "## Mechanism") {
		t.Fatalf("mechanism_claim = %+v, want fail naming the section", c)
	}
	if !strings.Contains(c.Remedy, NullityToken) {
		t.Fatalf("remedy must offer the claim or the token, got %q", c.Remedy)
	}
}

// TestReadyMechanismAbsentPasses: prompted is not required — an absent section
// is a claim not carried, which the gradient never conflates with a fault.
func TestReadyMechanismAbsentPasses(t *testing.T) {
	res := readyWithClaims(t, "", "## Scope Conditions\n\n"+NullityToken+"\n\n")
	if c := checkByName(t, res, "mechanism_claim"); !c.OK {
		t.Fatalf("mechanism_claim = %+v, want OK for an absent prompted claim", c)
	}
}

// TestReadyConditionMarkerMissing is itd-177's fifth criterion, missing half.
func TestReadyConditionMarkerMissing(t *testing.T) {
	res := readyWithClaims(t, "", "## Scope Conditions\n\n"+
		"- stamped <!-- cond: cond-2608300102030405 -->\n- unstamped\n\n")
	if !res.Ready {
		t.Fatal("an advisory check must not withhold readiness: an unidentified condition must not be ready is reported, not gated")
	}
	c := checkByName(t, res, "scope_conditions")
	if c.OK || !strings.Contains(c.Detail, "2") {
		t.Fatalf("scope_conditions = %+v, want fail naming condition 2", c)
	}
	if !strings.Contains(c.Remedy, "abcd intent plan itd-10") {
		t.Fatalf("remedy must name the write-capable verb, got %q", c.Remedy)
	}
}

// TestReadyConditionMarkerDuplicated is the fifth criterion's duplicate half:
// two conditions sharing an identity is named by the repeated cond- id.
func TestReadyConditionMarkerDuplicated(t *testing.T) {
	const id = "cond-2608300102030405"
	res := readyWithClaims(t, "", "## Scope Conditions\n\n"+
		"- one <!-- cond: "+id+" -->\n- two <!-- cond: "+id+" -->\n\n")
	if !res.Ready {
		t.Fatal("an advisory check must not withhold readiness: a duplicated identity must not be ready is reported, not gated")
	}
	c := checkByName(t, res, "scope_conditions")
	if c.OK || !strings.Contains(c.Detail, id) {
		t.Fatalf("scope_conditions = %+v, want fail naming %s", c, id)
	}
}

// TestReadyScopeConditionsProseWithoutBulletsFails: a condition that is not a
// top-level bullet has nothing to carry an identity, so prose alone — the
// create-path scaffold's own prompt line included — is not a recorded claim.
func TestReadyScopeConditionsProseWithoutBulletsFails(t *testing.T) {
	res := readyWithClaims(t, "", "## Scope Conditions\n\nIt holds wherever a POSIX shell exists.\n\n")
	if c := checkByName(t, res, "scope_conditions"); c.OK || !strings.Contains(c.Detail, "bullet") {
		t.Fatalf("scope_conditions = %+v, want fail on prose without bullets", c)
	}
}

// TestReadyDisciplineExemptFromClaimChecks: a discipline record's template
// carries no claim sections, so the gradient exempts it and both checks say so.
func TestReadyDisciplineExemptFromClaimChecks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, disciplinesDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"mechanism_claim", "scope_conditions"} {
		c := checkByName(t, res, name)
		if !c.OK || !strings.Contains(c.Detail, "discipline records carry no claim sections") {
			t.Fatalf("%s = %+v, want the exemption", name, c)
		}
	}
}

// TestReadyReportsConditionIdentities: ReadyResult carries the parsed conditions,
// which is the payload the identity criteria assert against.
func TestReadyReportsConditionIdentities(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n"+
			"# alpha\n\n## Scope Conditions\n\n"+
			"- holds on a POSIX shell <!-- cond: cond-2608300102030405 -->\n\n"+
			"## Acceptance Criteria\n\n- ok\n")
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conditions) != 1 {
		t.Fatalf("Conditions = %+v, want one", res.Conditions)
	}
	if res.Conditions[0].ID != "cond-2608300102030405" {
		t.Fatalf("condition id = %q", res.Conditions[0].ID)
	}
	if cond := checkByName(t, res, "scope_conditions"); !cond.OK {
		t.Fatalf("scope_conditions = %+v, want OK for an identified condition", cond)
	}
}

// TestReadyClaimChecksNotApplicableInTerminalBuckets is half of
// iss-2608300210588414: spc-55 rules retro-fitting claims onto shipped/ and
// superseded/ records out of scope — an absent stamp is information — so the
// gate must not print a backfill remedy at a record nobody may backfill.
func TestReadyClaimChecksNotApplicableInTerminalBuckets(t *testing.T) {
	for _, dir := range []string{shippedDir, supersededDir} {
		t.Run(dir, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, dir+"/itd-10-alpha.md",
				"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n"+
					"# alpha\n\n## Mechanism\n\n## Acceptance Criteria\n\n- ok\n")

			res, err := Ready(root, "itd-10")
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"mechanism_claim", "scope_conditions"} {
				c := checkByName(t, res, name)
				if !c.OK || !strings.Contains(c.Detail, "not applicable") {
					t.Errorf("%s = %+v, want OK reporting the check as not applicable", name, c)
				}
				if c.Remedy != "" {
					t.Errorf("%s must carry no remedy at a record nobody may backfill, got %q", name, c.Remedy)
				}
			}
		})
	}
}

// TestReadyScaffoldPromptIsNotAClaim is the other half: the create-path prompt
// is bytes this package wrote asking for a claim, never a claim somebody made,
// and the gate must not report it as one.
func TestReadyScaffoldPromptIsNotAClaim(t *testing.T) {
	root := t.TempDir()
	seeded := seedDraft("itd-10", DraftOptions{Slug: "alpha", Title: "alpha", SeedBody: "why it matters"}, researcherStamp(t))
	// Plan the criteria only, so the two claim sections stay as seeded.
	seeded = strings.Replace(seeded,
		"> _Required (the itd-1 discipline)", "- **Given** x, **when** y, **then** z.\n\n> _was: ", 1)
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		strings.Replace(seeded, "spec_id: null", "spec_id: spc-1", 1))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	mech := checkByName(t, res, "mechanism_claim")
	if !mech.OK {
		t.Fatalf("an unanswered prompt is not a fault (mechanism is nullable): %+v", mech)
	}
	if strings.Contains(mech.Detail, "mechanism claim stated") {
		t.Fatalf("the scaffold prompt must not read as a stated claim: %+v", mech)
	}
	if !strings.Contains(mech.Detail, "unanswered") {
		t.Fatalf("mechanism_claim = %+v, want the detail to name the unanswered prompt", mech)
	}
	cond := checkByName(t, res, "scope_conditions")
	if cond.OK || !strings.Contains(cond.Detail, "unanswered") {
		t.Fatalf("scope_conditions = %+v, want fail naming the unanswered prompt", cond)
	}
}

// TestReadyConditionCarryingTwoMarkers: a bullet holding two identities is
// named by the gate, because nothing downstream can choose between them.
func TestReadyConditionCarryingTwoMarkers(t *testing.T) {
	res := readyWithClaims(t, "", "## Scope Conditions\n\n"+
		"- one condition <!-- cond: cond-2608300102030405 --> <!-- cond: cond-2608300102030406 -->\n\n")
	if !res.Ready {
		t.Fatal("an advisory check must not withhold readiness: a bullet with two identities must not be ready is reported, not gated")
	}
	c := checkByName(t, res, "scope_conditions")
	if c.OK || !strings.Contains(c.Detail, "1") || !strings.Contains(c.Detail, "more than one identity") {
		t.Fatalf("scope_conditions = %+v, want fail naming the ambiguous bullet", c)
	}
	if c.Remedy == "" {
		t.Fatal("the fault must carry a remedy")
	}
}

// TestReadyReportsStructuralConditionFaults: every fault the stamp refuses on
// must also be a fault the gate names. Otherwise the gate says "run plan" and
// plan says no, which is the dead end iss-2608300210588874 already closed once.
func TestReadyReportsStructuralConditionFaults(t *testing.T) {
	tests := []struct {
		name, conditions, wantDetail string
	}{
		{"fenced section", "## Scope Conditions\n\n- holds on POSIX\n\n```\n- example\n```\n\n", "fenced"},
		{"malformed marker", "## Scope Conditions\n\n- holds on POSIX <!-- cond: cond-123 -->\n\n", "malformed identity marker"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := readyWithClaims(t, "", tt.conditions)
			if !res.Ready {
				t.Fatal("an advisory check must not withhold readiness: a structural fault must not be ready is reported, not gated")
			}
			c := checkByName(t, res, "scope_conditions")
			if c.OK || !strings.Contains(c.Detail, tt.wantDetail) {
				t.Fatalf("scope_conditions = %+v, want fail naming %q", c, tt.wantDetail)
			}
			if c.Remedy == "" {
				t.Fatal("a named fault needs a remedy the reader can act on")
			}
		})
	}
}

// TestReadyReportsADuplicatedSectionHeading: two headings make "the section"
// ambiguous; the gate says so rather than silently reading the first.
func TestReadyReportsADuplicatedSectionHeading(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Scope Conditions\n\n- holds on POSIX <!-- cond: cond-2608300102030405 -->\n\n"+
			"## Scope Conditions\n\n- and again <!-- cond: cond-2608300102030406 -->\n\n"+
			"## Acceptance Criteria\n\n- ok\n")
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	c := checkByName(t, res, "scope_conditions")
	if c.OK || !strings.Contains(c.Detail, "more than one") {
		t.Fatalf("scope_conditions = %+v, want fail naming the duplicated heading", c)
	}
}

// TestReadyNamesACommentedSection: the stamp refuses a section carrying a
// comment span, so the gate has to name it — a refusal the gate cannot describe
// is a dead end.
func TestReadyNamesACommentedSection(t *testing.T) {
	res := readyWithClaims(t, "", "## Scope Conditions\n\n- holds on POSIX\n\n<!--\n- parked\n-->\n\n")
	if !res.Ready {
		t.Fatal("an advisory check must not withhold readiness: a section carrying a comment span must not be ready is reported, not gated")
	}
	c := checkByName(t, res, "scope_conditions")
	if c.OK || !strings.Contains(c.Detail, "comment") {
		t.Fatalf("scope_conditions = %+v, want fail naming the comment", c)
	}
	if c.Remedy == "" {
		t.Fatal("the fault needs a remedy the reader can act on")
	}
}

// TestReadyNamesADuplicateHiddenBehindANullity is iss-2608300259321329: the
// structural faults were judged only after the claim-state switch, so a first
// section reading `None stated.` returned OK and the second heading — carrying
// real, unidentified bullets — was never reached. The gate passed while the
// stamp refused.
func TestReadyNamesADuplicateHiddenBehindANullity(t *testing.T) {
	res := readyWithClaims(t, "", "## Scope Conditions\n\n"+NullityToken+"\n\n"+
		"## Scope Conditions\n\n- holds only where a POSIX shell exists\n\n")
	if !res.Ready {
		t.Fatal("an advisory check must not withhold readiness: a nullity in the first section must not hide a second one is reported, not gated")
	}
	c := checkByName(t, res, "scope_conditions")
	if c.OK || !strings.Contains(c.Detail, "more than one") {
		t.Fatalf("scope_conditions = %+v, want fail naming the duplicated heading", c)
	}
}

// researcherStamp is the default disclosure pair a seeded draft carries, built
// through the one constructor rather than assembled by hand.
func researcherStamp(t *testing.T) provenance.Stamp {
	t.Helper()
	s, err := provenance.NewStamp(provenance.KindResearcherAuthored, "")
	if err != nil {
		t.Fatalf("NewStamp: %v", err)
	}
	return s
}

// groundsSection is the recorded-grounds section a green fixture carries, so a
// test about the claim checks is not incidentally a test about the grounds one.
const groundsSection = "\n## Grounds\n\n- pursued: we expect the recorded conjecture to outlive the session that had it\n"

// TestReadyGroundsAbsentFails: with the recording path landed and the planned/
// bucket populated, the check refuses. The remedy names the flag and the closed
// vocabulary, so the report says exactly how to answer it.
func TestReadyGroundsAbsentFails(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedUnlinked("itd-10", "alpha"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
	g := checkByName(t, res, CheckGrounds)
	if g.OK {
		t.Fatalf("grounds check = %+v, want a refusal on a record carrying none", g)
	}
	if !strings.Contains(g.Remedy, "--grounds") || !strings.Contains(g.Remedy, "pursued") {
		t.Fatalf("grounds remedy = %q, want the flag and the vocabulary", g.Remedy)
	}
	if res.Ready {
		t.Fatal("an intent with no recorded grounds must not be ready")
	}
}

// TestReadyGroundsPresentPasses and TestReadyGroundsReportsEntries are one
// assertion: a record carrying a well-formed entry passes, and the report says
// how many are recorded.
func TestReadyGroundsPresentPasses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		plannedUnlinked("itd-10", "alpha")+
			"\n## Grounds\n\n- pursued: we expect a stamped identity to survive rewording\n")

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
	g := checkByName(t, res, CheckGrounds)
	if !g.OK {
		t.Fatalf("grounds check = %+v, want OK on a record carrying an entry", g)
	}
	if !strings.Contains(g.Detail, "1 recorded ground") {
		t.Fatalf("grounds detail = %q, want the entry count", g.Detail)
	}
}

// TestReadyGroundsIgnoresMalformedEntry: prose under the heading is prose. Only
// a well-formed entry counts, and the gate never puts a verdict on a sentence
// somebody wrote for a human.
func TestReadyGroundsIgnoresMalformedEntry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		plannedUnlinked("itd-10", "alpha")+
			"\n## Grounds\n\nSome prose about why this matters.\n\n- planned: not a vocabulary value\n")

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if g := checkByName(t, res, CheckGrounds); strings.Contains(g.Detail, "1 recorded ground") {
		t.Fatalf("grounds detail = %q, want no entry counted", g.Detail)
	}
}

// TestReadyGroundsExemptInTerminalBuckets: population is forward-only, so a
// shipped or superseded record and a discipline are reported as not applicable
// rather than told to record work nobody may do.
func TestReadyGroundsExemptInTerminalBuckets(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-10-alpha.md", plannedUnlinked("itd-10", "alpha"))
	writeFile(t, root, disciplinesDir+"/itd-11-beta.md", plannedUnlinked("itd-11", "beta"))

	for _, id := range []string{"itd-10", "itd-11"} {
		res, err := Ready(root, id)
		if err != nil {
			t.Fatal(err)
		}
		g := checkByName(t, res, CheckGrounds)
		if !g.OK || !strings.Contains(g.Detail, "not applicable") {
			t.Fatalf("%s grounds check = %+v, want the not-applicable exemption", id, g)
		}
	}
}

// TestGroundsExemptionIsNotTheClaimExemption is the third nit of
// iss-2608301657350399: groundsCheck reused claimCheckExemption, so the grounds
// row of a shipped record reported "a shipped record's CLAIMS are never
// backfilled" — a true sentence about a check that is not the one reporting it,
// the same detail-string class as the resolved iss-2608300210588414.
//
// The assertion is that the two rows do not report the SAME STRING, not that the
// grounds row avoids some particular word. One string doing two jobs is the
// defect itself, and identity is what no rewording of either sentence can
// satisfy while the reuse is still there.
func TestGroundsExemptionIsNotTheClaimExemption(t *testing.T) {
	for _, bucket := range []string{BucketShipped, BucketSuperseded, BucketDisciplines} {
		it := Intent{ID: "itd-1", Bucket: bucket}
		got := groundsCheck(it, "")
		if !got.OK {
			t.Fatalf("%s: the grounds row is exempt, so it must pass: %+v", bucket, got)
		}
		claim, exempt := claimCheckExemption(it)
		if !exempt {
			t.Fatalf("%s: the claim check is expected to be exempt here too", bucket)
		}
		if got.Detail == claim {
			t.Errorf("%s: the grounds row reports the CLAIM check's exemption verbatim: %q", bucket, got.Detail)
		}
		// And it says what it is about, so the fix is a different sentence rather
		// than an empty one.
		if !strings.Contains(got.Detail, "grounds") && !strings.Contains(got.Detail, "conjecture") {
			t.Errorf("%s: the grounds row names neither grounds nor the conjecture: %q", bucket, got.Detail)
		}
	}
}

// TestReadyAdvisoryChecksNeverGate is iss-2609091009111294: the two claim checks
// and the grounds check are evaluated and reported, remedy included, and none of
// them withholds readiness. The four structural checks keep gating. A record
// that fails all three advisory rows at once and nothing else is READY.
func TestReadyAdvisoryChecksNeverGate(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
			"## Mechanism\n\n"+ // present and empty: the gate fault the gradient names
			"## Acceptance Criteria\n\n- ok\n") // no scope conditions, no grounds
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	assertShape(t, res)
	if !res.Ready {
		t.Fatalf("three failing advisory rows must leave the intent ready: %+v", res.Checks)
	}
	advisory := map[string]bool{CheckMechanismClaim: true, CheckScopeConditions: true, CheckGrounds: true}
	for _, c := range res.Checks {
		switch {
		case c.Name == CheckSteps:
			// Advisory, and passing: the spec lists no steps, so it is one step.
			if !c.OK || !c.Advisory {
				t.Fatalf("%s = %+v, want a passing advisory row", c.Name, c)
			}
		case advisory[c.Name]:
			if c.OK || !c.Advisory || c.Remedy == "" {
				t.Fatalf("%s = %+v, want a failing advisory row carrying its remedy", c.Name, c)
			}
		default:
			if !c.OK || c.Advisory {
				t.Fatalf("%s = %+v, want a passing structural row", c.Name, c)
			}
		}
	}
}

// The steps row reports the linked spec's `## Steps` shape — a list, or none
// and so one step — and is advisory: a malformed section is named with its
// remedy and never withholds readiness (itd-2609212103565953, spec scope 1).
func TestReadyReportsTheStepsShape(t *testing.T) {
	cases := []struct {
		name, steps string
		ok          bool
		detail      string
	}{
		{"none", "", true, "built as one step"},
		{"listed", "\n## Steps\n\n1. The parser\n   - landed: #1\n2. The loop\n", true, "2 step(s) listed, 1 landed"},
		{"malformed", "\n## Steps\n\nFirst the parser, then the loop.\n", false, "not a numbered step"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
			writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10")+tc.steps)
			res, err := Ready(root, "itd-10")
			if err != nil {
				t.Fatal(err)
			}
			assertShape(t, res)
			c := checkByName(t, res, CheckSteps)
			if c.OK != tc.ok || !c.Advisory || !strings.Contains(c.Detail, tc.detail) {
				t.Fatalf("steps = %+v, want ok=%v advisory with detail containing %q", c, tc.ok, tc.detail)
			}
			if !tc.ok && !strings.Contains(c.Remedy, "1. <title>") {
				t.Fatalf("a malformed section's remedy must show the list shape: %+v", c)
			}
			if !res.Ready {
				t.Fatalf("the steps row never withholds readiness: %+v", res.Checks)
			}
		})
	}
}
