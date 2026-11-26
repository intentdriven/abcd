package intent

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/refusal"
)

// TestAnUnknownIntentCarriesItsCode: every act that looks an intent id up
// refuses an id no bucket holds with intent/unknown-intent on the error
// itself, so the front door reads the code with refusal.CodeOf and a
// reworded message keeps it (spc-2610301431108175 approach 1).
func TestAnUnknownIntentCarriesItsCode(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedUnlinked("itd-10", "alpha"))
	const unknown = "itd-999"
	g := mustGrounds(t, grounds.Pursued, "we expect an unknown id to be refused before any write")
	acts := map[string]func() error{
		"Plan":            func() error { _, err := Plan(root, unknown, PlanOptions{}); return err },
		"Link":            func() error { _, err := Link(root, unknown, "spc-1"); return err },
		"AddRelatedIssue": func() error { _, err := AddRelatedIssue(root, unknown, "iss-1"); return err },
		"Ready":           func() error { _, err := Ready(root, unknown); return err },
		"RecordGrounds":   func() error { _, err := RecordGrounds(root, unknown, g); return err },
		"Hold":            func() error { _, err := Hold(root, unknown, "the parser rewrite comes first"); return err },
		"Unhold":          func() error { _, err := Unhold(root, unknown); return err },
		"Target":          func() error { _, err := Target(root, unknown, "v1.0.0"); return err },
		"AutoPlanChecks":  func() error { _, err := AutoPlanChecks(root, unknown); return err },
		"AssemblePrepass": func() error { _, err := AssemblePrepass(root, unknown); return err },
		"ReEmitAuditWith": func() error { _, err := ReEmitAuditWith(root, unknown, AuditEmitOptions{}); return err },
		"Reclassify": func() error {
			_, err := Reclassify(root, unknown, ReclassifyRequest{Kind: KindStandalone})
			return err
		},
		"Edge": func() error {
			_, err := Edge(root, EdgeRequest{ID: unknown, BlockedBy: []string{"itd-10"}})
			return err
		},
		"Edge's target": func() error {
			_, err := Edge(root, EdgeRequest{ID: "itd-10", BlockedBy: []string{unknown}})
			return err
		},
	}
	for name, act := range acts {
		err := act()
		if got := refusal.CodeOf(err); got != refusal.IntentUnknownIntent {
			t.Errorf("%s on an unknown id: code %q (err %v), want %s", name, got, err, refusal.IntentUnknownIntent)
		}
	}
}

// TestASpecCloseCarriesItsCodes: a close of a spec no folder holds, and a
// close that would ship an intent recording no impact with none given, carry
// spec/unknown-spec and spec/missing-impact on the error itself.
func TestASpecCloseCarriesItsCodes(t *testing.T) {
	root := t.TempDir()
	if _, err := Reconcile(root, "spc-9", "", RemainderRequest{}); refusal.CodeOf(err) != refusal.SpecUnknownSpec {
		t.Errorf("a close of an unknown spec: code %q (err %v), want %s", refusal.CodeOf(err), err, refusal.SpecUnknownSpec)
	}
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	pr, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if _, err := Reconcile(root, pr.Spec.ID, "", RemainderRequest{}); refusal.CodeOf(err) != refusal.SpecMissingImpact {
		t.Errorf("a close with no impact: code %q (err %v), want %s", refusal.CodeOf(err), err, refusal.SpecMissingImpact)
	}
}
