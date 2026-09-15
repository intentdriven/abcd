package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// lintIntentImpact runs only the intent_impact_valid record-lint blocker over the
// intent tree under root and returns its findings.
func lintIntentImpact(t *testing.T, root string) []lint.Finding {
	t.Helper()
	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"intent_impact_valid": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
		},
	}
	fs, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	return fs
}

// TestCreateFromTextStampsImpact is the iss-117 intent half: the seed-draft path
// must be able to stamp a valid impact so the record survives — unchanged by the
// impact-preserving planned->shipped move — into shipped/, where
// intent_impact_valid REQUIRES one. A seed path that cannot set impact produces an
// intent the tool can never ship past its own blocker.
func TestCreateFromTextStampsImpact(t *testing.T) {
	root := t.TempDir()
	it, err := CreateFromText(root, "a user-facing improvement worth shipping", "additive", "")
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	// The seeded draft carries the impact bare (impact: additive), matching the
	// machine-read enum the shipped-intent gate compares byte-for-byte.
	raw, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\nimpact: additive\n") {
		t.Fatalf("seeded draft must carry bare impact, got:\n%s", raw)
	}

	// Simulate the planned->shipped move (moveIntentToBucket preserves frontmatter):
	// relocate the file into shipped/ and assert intent_impact_valid is satisfied.
	shipped := filepath.Join(root, shippedDir, filepath.Base(it.Path))
	if err := os.MkdirAll(filepath.Dir(shipped), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, it.Path), shipped); err != nil {
		t.Fatal(err)
	}
	if fs := lintIntentImpact(t, root); len(fs) != 0 {
		t.Fatalf("shipped record must satisfy intent_impact_valid, got %d finding(s): %+v", len(fs), fs)
	}
}

// TestCreateFromTextRejectsBadImpact proves the seed boundary fails closed on an
// impact the intent gate would reject: a misspelling, and — because an intent is
// press-release-first — the otherwise-legal `internal`.
func TestCreateFromTextRejectsBadImpact(t *testing.T) {
	root := t.TempDir()
	for _, bad := range []string{"additiv", "Additive", "internal", `"fix"`} {
		if _, err := CreateFromText(root, "some intent text", bad, ""); err == nil {
			t.Fatalf("CreateFromText with impact %q must be refused", bad)
		}
	}
	// No drafts file appeared for any refused create.
	if entries, _ := os.ReadDir(filepath.Join(root, draftsDir)); len(entries) != 0 {
		t.Fatalf("a refused create wrote %d files, want 0", len(entries))
	}
}

// TestCreateFromTextImpactOptional keeps the no-impact path intact: an empty
// impact seeds a draft with no impact line (drafts are not gated), exactly as
// before, so the existing capture->intent promotion flow is unchanged.
func TestCreateFromTextImpactOptional(t *testing.T) {
	root := t.TempDir()
	it, err := CreateFromText(root, "seeded without a judgement yet", "", "")
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "impact:") {
		t.Fatalf("an unset impact must write no impact line, got:\n%s", raw)
	}
}

// TestReconcileNeverShipsAnIntentThatTripsItsOwnBlocker is iss-126: `spec close`
// is the one verb that moves an intent into shipped/, and shipped/ is the one
// bucket intent_impact_valid requires an impact in. An intent seeded without a
// judgement is legal in drafts/ and legal in planned/ — CreateFromText makes
// impact optional on purpose — so a record can travel seed -> plan -> close
// through nothing but abcd's own verbs and land in the bucket where abcd's own
// record-lint then refuses it. Two outcomes are acceptable and no third is: the
// close refuses before it moves anything, or it ships a record the blocker
// passes. Shipping a record that trips the blocker is not.
func TestReconcileNeverShipsAnIntentThatTripsItsOwnBlocker(t *testing.T) {
	root := t.TempDir()
	seedShippableIntent(t, root, "")
	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		// Refusing is the other acceptable outcome — but it must be a clean
		// refusal, with the intent left where it was for a human to judge.
		if _, statErr := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); statErr != nil {
			t.Fatalf("Reconcile refused (%v) but did not leave the intent in planned/: %v", err, statErr)
		}
		if !strings.Contains(err.Error(), "impact") {
			t.Fatalf("Reconcile refused for something other than the missing impact: %v", err)
		}
		return
	}
	if fs := lintIntentImpact(t, root); len(fs) != 0 {
		t.Fatalf("Reconcile shipped %s into %s, where intent_impact_valid then refuses it: %d finding(s): %+v",
			res.Intent.ID, res.Intent.Bucket, len(fs), fs)
	}
}

// seedShippableIntent lays a planned intent linked to an open spec, carrying the
// given impact — or, for an empty impact, no impact line at all. The record is
// written literally rather than through plannedLinked, because the whole point
// of these cases is which judgement the frontmatter does and does not hold, and
// a fixture that borrowed one from elsewhere could stop exercising the case it
// names without failing.
func seedShippableIntent(t *testing.T, root, impact string) {
	t.Helper()
	fm := "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n"
	if impact != "" {
		fm += "impact: " + impact + "\n"
	}
	body := fm + "---\n# alpha\n\n## Scope Conditions\n\n" + NullityToken +
		"\n\n## Acceptance Criteria\n\n- ok\n" + groundsSection + "\n## Audit Notes\n"
	if impact == "" && strings.Contains(body, "impact") {
		t.Fatalf("the impactless fixture grew an impact field:\n%s", body)
	}
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", body)
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
}

// shippedIntentBody reads the record after a close, from shipped/.
func shippedIntentBody(t *testing.T, root string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestReconcileStampsTheImpactItIsGiven is the other half of iss-126: the
// refusal above is only tolerable because the ship verb can also supply the
// judgement it demands. `spec close --impact fix` over a record that declares
// none stamps it and ships a record intent_impact_valid passes — the end-to-end
// property the issue asks for, that a record produced entirely by abcd's own
// verbs survives abcd's own record-lint.
func TestReconcileStampsTheImpactItIsGiven(t *testing.T) {
	root := t.TempDir()
	seedShippableIntent(t, root, "")

	res, err := Reconcile(root, "spc-1", "fix", RemainderRequest{})
	if err != nil {
		t.Fatalf("Reconcile with an impact: %v", err)
	}
	if !res.IntentMoved || res.To != BucketShipped {
		t.Fatalf("Reconcile result = %+v", res)
	}
	if body := shippedIntentBody(t, root); !strings.Contains(body, "\nimpact: fix\n") {
		t.Fatalf("the shipped record does not carry the stamped impact:\n%s", body)
	}
	if fs := lintIntentImpact(t, root); len(fs) != 0 {
		t.Fatalf("the stamped record still trips intent_impact_valid: %+v", fs)
	}
}

// TestReconcileRefusesAnImpactItsOwnGateWouldReject: the stamp is validated at
// the same bar the seed path applies, so `spec close` cannot write a value
// intent_impact_valid would then refuse. `internal` is the interesting one — it
// is legal on an issue and a category error on an intent — and it must be
// refused with NO move, not stamped and shipped.
func TestReconcileRefusesAnImpactItsOwnGateWouldReject(t *testing.T) {
	for _, bad := range []string{"internal", "braking", "Fix"} {
		t.Run(bad, func(t *testing.T) {
			root := t.TempDir()
			seedShippableIntent(t, root, "")
			if _, err := Reconcile(root, "spc-1", bad, RemainderRequest{}); err == nil {
				t.Fatalf("Reconcile accepted --impact %q", bad)
			}
			if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err != nil {
				t.Fatalf("a refused impact still moved the intent out of planned/: %v", err)
			}
		})
	}
}

// TestReconcileWillNotReviseARecordedImpact: closing a spec is not the place to
// change a judgement already written down. A --impact that disagrees with the
// record is refused rather than silently overwriting it, and a --impact that
// agrees is a no-op.
func TestReconcileWillNotReviseARecordedImpact(t *testing.T) {
	root := t.TempDir()
	seedShippableIntent(t, root, "additive")
	if _, err := Reconcile(root, "spc-1", "breaking", RemainderRequest{}); err == nil {
		t.Fatal("Reconcile overwrote a recorded impact from the flag")
	}
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err != nil {
		t.Fatalf("the refused revision still moved the intent: %v", err)
	}

	if _, err := Reconcile(root, "spc-1", "additive", RemainderRequest{}); err != nil {
		t.Fatalf("an --impact agreeing with the record must be accepted: %v", err)
	}
	if body := shippedIntentBody(t, root); !strings.Contains(body, "\nimpact: additive\n") {
		t.Fatalf("the recorded impact did not survive the close:\n%s", body)
	}
}

// TestReconcileRefusesToShipAnIllegalRecordedImpact: a record carrying a value
// the gate rejects is refused at the close, where the author is standing, not
// carried into shipped/ for record-lint to find later as archaeology.
func TestReconcileRefusesToShipAnIllegalRecordedImpact(t *testing.T) {
	for _, bad := range []string{"internal", "braking"} {
		t.Run(bad, func(t *testing.T) {
			root := t.TempDir()
			seedShippableIntent(t, root, bad)
			if _, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err == nil {
				t.Fatalf("Reconcile shipped a record recording impact %q", bad)
			}
			if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err != nil {
				t.Fatalf("the refused close still moved the intent: %v", err)
			}
		})
	}
}
