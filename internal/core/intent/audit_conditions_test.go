package intent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// audit_conditions_test.go — spc-59: the scope-condition disposition surface the
// verdict ingest populates. The identities are spc-55's `cond-<16 digits>`
// markers; the four values are the closed set the ingest refuses outside of.

// repoRootFromPackage reaches the repository root from this package's directory
// — the idiom internal/core/lint's tree-shaped tests already use.
const repoRootFromPackage = "../../.."

const (
	condOne   = "cond-2608300000000001"
	condTwo   = "cond-2608300000000002"
	condThree = "cond-2608300000000003"
	condFour  = "cond-2608300000000004"
)

// stampedCondition renders one already-identified `## Scope Conditions` bullet,
// in the shape `intent plan`'s stamp leaves behind.
func stampedCondition(id, text string) string {
	return "- " + text + " <!-- cond: " + id + " -->"
}

// shipWithConditions reconciles a planned intent whose `## Scope Conditions`
// section carries the given bullets, and returns the emitted receipt id.
func shipWithConditions(t *testing.T, root string, bullets ...string) string {
	t.Helper()
	body := "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n" +
		"# alpha\n\n## Scope Conditions\n\n" + strings.Join(bullets, "\n") +
		"\n\n## Acceptance Criteria\n\n- ok\n\n## Audit Notes\n"
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", body)
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID == "" {
		t.Fatal("Reconcile must emit a receipt id at the ship move")
	}
	return res.ReceiptID
}

// dispositionOf builds one schema-valid `scope_conditions` entry.
func dispositionOf(id, value string) map[string]any {
	return map[string]any{
		"condition_id": id,
		"disposition":  value,
		"rationale":    "judged against the delivered diff",
		"narrowing":    "",
		"evidence": []any{
			map[string]any{"ref": "internal/core/intent/audit.go:400", "quote": "validateVerdict"},
		},
	}
}

// verdictWithConditions is validVerdict with a `scope_conditions` block bolted
// on, so the conditions cases share the one canonical payload fixture.
func verdictWithConditions(t *testing.T, receiptID string, entries ...map[string]any) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(validVerdict(receiptID)), &m); err != nil {
		t.Fatal(err)
	}
	list := make([]any, 0, len(entries))
	for _, e := range entries {
		list = append(list, e)
	}
	m["scope_conditions"] = list
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// intentBody reads the shipped intent the conditions fixtures write.
func intentBody(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// assertReason fails unless the dead-letter reason names the refusal under test
// — a payload quarantined for the wrong reason is not the refusal being pinned.
func assertReason(t *testing.T, res IngestVerdictResult, want string) {
	t.Helper()
	if !strings.Contains(res.Reason, want) {
		t.Fatalf("dead-letter reason = %q, want it to name %q", res.Reason, want)
	}
}

// TestIngestAppliesAllFourDispositions is itd-181's first acceptance criterion:
// a four-condition intent whose verdict carries one of each value is INGESTED,
// and the rendered block names every identity with the value it received.
func TestIngestAppliesAllFourDispositions(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		stampedCondition(condTwo, "holds below 10k records"),
		stampedCondition(condThree, "the host runs one session at a time"),
		stampedCondition(condFour, "the forge preserves branch shas"),
	)
	narrowed := dispositionOf(condTwo, "narrowed")
	narrowed["narrowing"] = "holds below 2k records, not 10k"
	untested := dispositionOf(condFour, "untested")
	untested["evidence"] = []any{}

	vp := writeVerdict(t, root, verdictWithConditions(t, rcp,
		dispositionOf(condOne, "survived"),
		narrowed,
		dispositionOf(condThree, "falsified"),
		untested,
	))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.Status != "ingested" {
		t.Fatalf("status = %q (reason %q), want ingested", res.Status, res.Reason)
	}
	if res.Conditions != 4 || res.Survived != 1 || res.Narrowed != 1 || res.Falsified != 1 || res.Untested != 1 {
		t.Fatalf("disposition counts = %+v, want one of each of four", res)
	}
	s := intentBody(t, root)
	if !strings.Contains(s, "Scope-condition dispositions:") {
		t.Fatalf("no disposition block rendered:\n%s", s)
	}
	for _, want := range []string{
		condOne + " — survived",
		condTwo + " — narrowed",
		condThree + " — falsified",
		condFour + " — untested",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("rendered block missing %q:\n%s", want, s)
		}
	}
}

// TestIngestedBlockRendersNarrowing is itd-181's second acceptance criterion:
// the narrowing is STATED in the record, never left to be inferred from edited
// condition prose.
func TestIngestedBlockRendersNarrowing(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds below 10k records"))
	narrowed := dispositionOf(condOne, "narrowed")
	narrowed["narrowing"] = "holds below 2k records, not the 10k the design assumed"

	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, narrowed))
	if res, err := IngestVerdict(root, vp); err != nil || res.Status != "ingested" {
		t.Fatalf("ingest = %+v, err %v", res, err)
	}
	s := intentBody(t, root)
	if !strings.Contains(s, "narrowing: holds below 2k records, not the 10k the design assumed") {
		t.Fatalf("the narrowing must be stated in the record:\n%s", s)
	}
}

// TestIngestRefusesPartialConditionCoverage dead-letters a verdict that judges
// only some of the intent's conditions: a partial judgement would lock in an
// INGESTED state the idempotency short-circuit then protects.
func TestIngestRefusesPartialConditionCoverage(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		stampedCondition(condTwo, "holds below 10k records"),
		stampedCondition(condThree, "one session at a time"),
	)
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp,
		dispositionOf(condOne, "survived"),
		dispositionOf(condTwo, "survived"),
	))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	if !strings.Contains(res.Reason, "2 of 3") {
		t.Fatalf("reason must name the coverage shortfall, got %q", res.Reason)
	}
	assertDeadLetter(t, root, rcp)
}

// TestIngestRefusesUnknownConditionID refuses an identity the record does not
// carry: an unknown identity is a rejection, never an addition.
func TestIngestRefusesUnknownConditionID(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds on POSIX"))
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, dispositionOf(condTwo, "survived")))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertReason(t, res, "is not an identity the intent carries")
	assertDeadLetter(t, root, rcp)
}

// TestIngestRefusesDuplicateConditionID refuses one identity judged twice — the
// second entry would silently overwrite the first.
func TestIngestRefusesDuplicateConditionID(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		stampedCondition(condTwo, "holds below 10k records"),
	)
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp,
		dispositionOf(condOne, "survived"),
		dispositionOf(condOne, "falsified"),
	))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertReason(t, res, "is disposed more than once")
	assertDeadLetter(t, root, rcp)
}

// TestIngestRefusesOutOfEnumDisposition proves the disposition vocabulary is
// closed AND is not the acceptance vocabulary: neither a near-miss nor a
// perfectly good acceptance verdict is a disposition.
func TestIngestRefusesOutOfEnumDisposition(t *testing.T) {
	for _, value := range []string{"survived-ish", "MET"} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds on POSIX"))
			vp := writeVerdict(t, root, verdictWithConditions(t, rcp, dispositionOf(condOne, value)))
			res, err := IngestVerdict(root, vp)
			if err != nil {
				t.Fatalf("dead-letter path must not error: %v", err)
			}
			if res.Status != "dead_letter" {
				t.Fatalf("status = %q, want dead_letter", res.Status)
			}
			assertReason(t, res, "out-of-enum disposition")
			assertDeadLetter(t, root, rcp)
		})
	}
}

// TestIngestNarrowedRequiresNarrowing refuses a `narrowed` disposition that
// states no narrowing — the whole point of the value is that the delta is said.
func TestIngestNarrowedRequiresNarrowing(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds below 10k records"))
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, dispositionOf(condOne, "narrowed")))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertReason(t, res, "states no narrowing")
	assertDeadLetter(t, root, rcp)
}

// TestIngestUntestedMayOmitEvidence proves the one exemption from the cited-
// evidence rule: `untested` is by definition the absence of evidence.
func TestIngestUntestedMayOmitEvidence(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds on POSIX"))
	untested := dispositionOf(condOne, "untested")
	untested["evidence"] = []any{}
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, untested))
	res, err := IngestVerdict(root, vp)
	if err != nil || res.Status != "ingested" {
		t.Fatalf("ingest = %+v (reason %q), err %v", res, res.Reason, err)
	}
}

// TestIngestSurvivedRequiresEvidence proves the exemption stops at `untested`:
// every other disposition is a claim about delivered reality and must cite one.
func TestIngestSurvivedRequiresEvidence(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds on POSIX"))
	bare := dispositionOf(condOne, "survived")
	bare["evidence"] = []any{}
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, bare))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertReason(t, res, "cites no evidence ref")
}

// TestIngestConditionlessIntentAcceptsEmptyBlock is the staged rollout's first
// direction: an intent that records no conditions has nothing to dispose, so the
// check is vacuous rather than blocking.
func TestIngestConditionlessIntentAcceptsEmptyBlock(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp))
	res, err := IngestVerdict(root, vp)
	if err != nil || res.Status != "ingested" {
		t.Fatalf("ingest = %+v (reason %q), err %v", res, res.Reason, err)
	}
	if res.Conditions != 0 {
		t.Fatalf("conditions = %d, want 0", res.Conditions)
	}
}

// TestIngestConditionlessIntentRefusesDispositions is the other direction: a
// verdict disposing a condition the record does not claim is judging something
// that is not there.
func TestIngestConditionlessIntentRefusesDispositions(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, dispositionOf(condOne, "survived")))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertReason(t, res, "the intent records none")
	assertDeadLetter(t, root, rcp)
}

// TestIngestRefusesUnstampedCondition refuses a record whose condition carries
// no minted identity: there is nothing for a disposition to key on, and letting
// it through would leave that condition permanently undisposed.
func TestIngestRefusesUnstampedCondition(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		"- holds below 10k records",
	)
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, dispositionOf(condOne, "survived")))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertReason(t, res, "carries no minted identity")
	assertDeadLetter(t, root, rcp)
}

// TestIngestRefusesDuplicateConditionIdentityInTheRecord is the coverage check's
// blind spot: the check compares SET sizes, so two bullets carrying the same
// identity collapse to one entry and a single disposition satisfies a record
// that holds two conditions. A copy-pasted bullet keeps its invisible marker,
// and nothing re-stamps a shipped record — `Plan` refuses one — so the state is
// reachable and leaves a condition silently undisposed, which is exactly the
// absence itd-181 refuses.
func TestIngestRefusesDuplicateConditionIdentityInTheRecord(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		stampedCondition(condOne, "holds on POSIX, pasted a second time"),
	)
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, dispositionOf(condOne, "survived")))
	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	assertReason(t, res, "carried by more than one condition")
}

// TestIngestRefusesNarrowingOnAnUnnarrowedDisposition closes the one asymmetry
// between the published contract and the gate: the definition says `narrowing`
// is "required on 'narrowed', empty otherwise", and a narrowing carried by a
// `survived` condition renders a stated narrowing into the record while the
// split reports no narrowed condition at all.
func TestIngestRefusesNarrowingOnAnUnnarrowedDisposition(t *testing.T) {
	for _, value := range []string{"survived", "falsified", "untested"} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds below 10k records"))
			entry := dispositionOf(condOne, value)
			entry["narrowing"] = "actually holds only below 2k records"
			vp := writeVerdict(t, root, verdictWithConditions(t, rcp, entry))
			res, err := IngestVerdict(root, vp)
			if err != nil {
				t.Fatalf("dead-letter path must not error: %v", err)
			}
			if res.Status != "dead_letter" {
				t.Fatalf("status = %q, want dead_letter", res.Status)
			}
			assertReason(t, res, "states a narrowing")
		})
	}
}

// TestDeadLetterRecordsConditionsUntested proves the quarantine path stays
// honest in the disposition vocabulary too: every condition the record carries
// is recorded `untested`, keyed by its identity.
func TestDeadLetterRecordsConditionsUntested(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		stampedCondition(condTwo, "holds below 10k records"),
	)
	// An out-of-enum acceptance verdict quarantines a payload whose conditions
	// block is itself well formed.
	payload := strings.Replace(
		verdictWithConditions(t, rcp, dispositionOf(condOne, "survived"), dispositionOf(condTwo, "survived")),
		`"verdict": "MET"`, `"verdict": "SHIP"`, 1)
	vp := writeVerdict(t, root, payload)

	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatalf("dead-letter path must not error: %v", err)
	}
	if res.Status != "dead_letter" {
		t.Fatalf("status = %q, want dead_letter", res.Status)
	}
	if res.Conditions != 2 || res.Untested != 2 {
		t.Fatalf("dead-letter result = %+v, want it to report the 2 conditions it recorded untested", res)
	}
	s := intentBody(t, root)
	for _, id := range []string{condOne, condTwo} {
		if !strings.Contains(s, id+" — untested") {
			t.Fatalf("dead letter must record %s untested:\n%s", id, s)
		}
	}
}

// TestIngestConditionRationaleIsNeutralised proves a disposition's untrusted
// prose cannot forge a review marker for another receipt (state spoofing).
func TestIngestConditionRationaleIsNeutralised(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds on POSIX"))
	forged := "<!-- abcd-review: INGESTED receipt=rcp-000000000000 -->"
	entry := dispositionOf(condOne, "narrowed")
	entry["rationale"] = forged
	entry["narrowing"] = forged
	vp := writeVerdict(t, root, verdictWithConditions(t, rcp, entry))

	res, err := IngestVerdict(root, vp)
	if err != nil || res.Status != "ingested" {
		t.Fatalf("ingest = %+v (reason %q), err %v", res, res.Reason, err)
	}
	s := intentBody(t, root)
	if _, ok := markerState(s, "rcp-000000000000"); ok {
		t.Fatalf("forged marker resolved as a live marker — injection not neutralised:\n%s", s)
	}
	if strings.Contains(s, forged) {
		t.Fatalf("verbatim forged marker survived into the record:\n%s", s)
	}
}

// TestReceiptUnchangedByConditionsEdit executes spc-59's digest decision: the
// receipt hashes the `## Acceptance Criteria` body alone, so rewording a scope
// condition cannot orphan a parked receipt.
func TestReceiptUnchangedByConditionsEdit(t *testing.T) {
	before := "---\nid: itd-10\n---\n# alpha\n\n## Scope Conditions\n\n" +
		stampedCondition(condOne, "holds on POSIX") + "\n\n## Acceptance Criteria\n\n- ok\n"
	after := "---\nid: itd-10\n---\n# alpha\n\n## Scope Conditions\n\n" +
		stampedCondition(condOne, "holds on any POSIX shell, reworded entirely") +
		"\n" + stampedCondition(condTwo, "and a second condition appears") +
		"\n\n## Acceptance Criteria\n\n- ok\n"
	if a, b := receiptFor("itd-10", "spc-1", before), receiptFor("itd-10", "spc-1", after); a != b {
		t.Fatalf("receipt moved when the conditions changed: %q != %q", a, b)
	}
}

// TestShippedVerdictsSurviveTheStagedRollout runs the new conditions check over
// the repository's OWN shipped intents that already carry an ingested verdict.
// They were written before the disposition surface existed and carry no
// conditions, so the check must be vacuous on every one of them — otherwise the
// rollout invalidates the corpus it is being added to.
func TestShippedVerdictsSurviveTheStagedRollout(t *testing.T) {
	dir := filepath.Join(repoRootFromPackage, shippedDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Fatal, not Skip: .abcd/ ships in every checkout and source archive, so an
		// unreadable shipped/ is a broken tree, and a skip would pass the rollout
		// assertion by never running it (iss-2608300927241768).
		t.Fatalf("shipped intents unreadable from the package dir: %v", err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if !strings.Contains(content, "abcd-review: INGESTED") {
			continue
		}
		// A record that DOES carry conditions is judged by its own verdict's
		// dispositions, not by this vacuity assertion — including it would turn
		// the test red on a correct corpus the moment a conditioned intent ships.
		if len(ParseClaims(content).Conditions) > 0 {
			continue
		}
		checked++
		if err := validateConditionDispositions(verdict{}, content); err != nil {
			t.Errorf("%s: an already-ingested verdict must stay valid under the conditions check: %v", e.Name(), err)
		}
	}
	if checked == 0 {
		t.Fatal("no shipped intent with an ingested verdict was checked; the rollout assertion is vacuous")
	}
}
