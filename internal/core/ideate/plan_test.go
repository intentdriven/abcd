package ideate

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decisions"
	"github.com/intentdriven/abcd/internal/core/dryrun"
)

// plan_test.go pins the check/write split of `ideate record`
// (spc-2610301431131636, approach 1): PlanRecord runs every check Record runs
// and writes nothing, and Record is that plan applied, so the two cannot
// disagree about a verdict.

// TestPlanRecordRefusesAnUnresolvableCitationBeforeAnyWrite: a grill hit
// citing a record that is not there is refused by the plan, with the same
// typed refusal Record gives, and nothing reaches the disk.
func TestPlanRecordRefusesAnUnresolvableCitationBeforeAnyWrite(t *testing.T) {
	root := seedRepo(t)
	p := validPayload()
	p["legs"].([]any)[1].(map[string]any)["hits"] = []any{
		map[string]any{"record": "itd-9999", "relation": "covered"},
	}
	_, err := PlanRecord(root, "the-ideate-gate", publicStore, encode(t, p), at)
	var cite *CitationError
	if !errors.As(err, &cite) || strings.Join(cite.Unresolved, ",") != "itd-9999" {
		t.Fatalf("PlanRecord error = %v, want a *CitationError naming itd-9999", err)
	}
	_, realErr := Record(root, "the-ideate-gate", publicStore, encode(t, p), at)
	if realErr == nil || realErr.Error() != err.Error() {
		t.Fatalf("Record refuses as %v; the plan refused as %v", realErr, err)
	}
	assertNothingWritten(t, root)
	assertNoDecision(t, root)
}

// TestPlanRecordPlansThePublicWritesAndWritesNothing: the plan names the
// record (with the bytes Record lays) and the decision pointing at it (with
// the id the decision writer mints at the write), and the disk is untouched.
func TestPlanRecordPlansThePublicWritesAndWritesNothing(t *testing.T) {
	root := seedRepo(t)
	plan, err := PlanRecord(root, "the-ideate-gate", publicStore, encode(t, validPayload()), at)
	if err != nil {
		t.Fatalf("PlanRecord: %v", err)
	}
	assertNothingWritten(t, root)
	assertNoDecision(t, root)
	if len(plan.Plan.Writes) != 2 {
		t.Fatalf("the plan names %d writes, want the record and its pointer: %+v", len(plan.Plan.Writes), plan.Plan.Writes)
	}
	rec, ptr := plan.Plan.Writes[0], plan.Plan.Writes[1]
	if rec.Path != plan.Result.Path || rec.Action != dryrun.Create || rec.Bytes == 0 {
		t.Errorf("the record write is %+v, want a create of %s", rec, plan.Result.Path)
	}
	wantPtr := decisions.RelDir + "/" + decisions.Family + "-" + dryrun.MintedID + "-ideate-the-ideate-gate"
	if !strings.HasPrefix(ptr.Path, wantPtr) || ptr.Pending == "" {
		t.Errorf("the pointer write is %+v, want %s… with its pending id", ptr, wantPtr)
	}

	res, err := plan.Apply()
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rec.Path)))
	if err != nil || string(got) != string(rec.Content) {
		t.Fatalf("Apply wrote %q (%v), not the planned bytes", got, err)
	}
	parts := strings.Split(ptr.Path, dryrun.MintedID)
	for i := range parts {
		parts[i] = regexp.QuoteMeta(parts[i])
	}
	if !regexp.MustCompile("^" + strings.Join(parts, "[0-9]{16}") + "$").MatchString(res.DecisionPath) {
		t.Errorf("the pointer landed at %s, not where the plan named (%s)", res.DecisionPath, ptr.Path)
	}
	d, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(res.DecisionPath)))
	if err != nil || len(d) != ptr.Bytes {
		t.Errorf("the pointer is %d bytes (%v), the plan said %d", len(d), err, ptr.Bytes)
	}
}

// TestPlanRecordRefusesARecordedSlug: a verdict already recorded under the
// slug and date is refused by the plan in Record's own words.
func TestPlanRecordRefusesARecordedSlug(t *testing.T) {
	root := seedRepo(t)
	if _, err := Record(root, "the-ideate-gate", publicStore, encode(t, validPayload()), at); err != nil {
		t.Fatal(err)
	}
	_, err := PlanRecord(root, "the-ideate-gate", publicStore, encode(t, validPayload()), at)
	var exists *ExistsError
	if !errors.As(err, &exists) {
		t.Fatalf("PlanRecord error = %v, want an *ExistsError", err)
	}
}

func assertNoDecision(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(decisions.RelDir)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("the plan recorded a decision: %s", e.Name())
	}
}
