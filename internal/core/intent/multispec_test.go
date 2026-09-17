package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// An intent owns one or more specs: closing one while another still names the
// intent closes that spec and leaves the intent planned.
func TestReconcilePartialDeliveryLeavesIntentPlanned(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, specsOpen+"/spc-2-rest.md", specNaming("spc-2", "rest", "itd-10"))

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatalf("closing one spec of a multi-spec intent must not refuse: %v", err)
	}
	if res.IntentMoved {
		t.Fatalf("intent must not move while spc-2 is open: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err != nil {
		t.Fatalf("intent must stay in planned/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, specsClosed, "spc-1-alpha.md")); err != nil {
		t.Fatalf("spc-1 must be closed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, specsOpen, "spc-2-rest.md")); err != nil {
		t.Fatalf("spc-2 must stay open: %v", err)
	}
}

// The intent ships on the close after which no open spec names it.
func TestReconcileShipsWhenLastSpecCloses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsClosed+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, specsOpen+"/spc-2-rest.md", specNaming("spc-2", "rest", "itd-10"))

	res, err := Reconcile(root, "spc-2", "", RemainderRequest{})
	if err != nil {
		t.Fatalf("closing the last open spec must ship the intent: %v", err)
	}
	if !res.IntentMoved || res.To != BucketShipped {
		t.Fatalf("intent must ship on the last close: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(root, shippedDir, "itd-10-alpha.md")); err != nil {
		t.Fatalf("intent must be in shipped/: %v", err)
	}
}

// --impact is demanded at the ship transition and refused at an earlier close,
// which ships nothing: the refusal names the spec that keeps the intent planned.
func TestReconcileRefusesImpactWhenAnotherSpecStaysOpen(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, specsOpen+"/spc-2-rest.md", specNaming("spc-2", "rest", "itd-10"))

	_, err := Reconcile(root, "spc-1", "fix", RemainderRequest{})
	if err == nil {
		t.Fatal("an --impact at a close that ships nothing must be refused")
	}
	if !strings.Contains(err.Error(), "spc-2") || !strings.Contains(err.Error(), "still open") {
		t.Fatalf("refusal must name the open spec that keeps the intent planned: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, specsOpen, "spc-1-alpha.md")); err != nil {
		t.Fatalf("nothing may move on the refusal: %v", err)
	}
}

// Ready judges the OPEN spec of a multi-spec intent, not the closed one the
// intent's spec_id happens to name.
func TestReadyJudgesTheOpenSpecOfAMultiSpecIntent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsClosed+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, specsOpen+"/spc-2-rest.md",
		"---\nid: spc-2\nslug: rest\nintent: itd-10\n---\n# rest\n\n## Summary\n\n_Draft: describe what spc-2 delivers for itd-10._\n")

	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if res.Ready {
		t.Fatalf("the open spec's body is still the minted stub: %+v", res.Checks)
	}
	var body ReadyCheck
	for _, c := range res.Checks {
		if c.Name == CheckSpecBody {
			body = c
		}
	}
	if !strings.Contains(body.Detail, "spc-2") {
		t.Fatalf("spec_body must judge the open spec spc-2: %+v", body)
	}
}

// The close that delivers part of an intent mints the remainder spec and
// attaches it to the same intent, in one operation: the honest path is one
// command, not a close plus a hand-mint nothing enforces.
func TestReconcileMintsTheRemainderSpec(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "the-rest"})
	if err != nil {
		t.Fatalf("minting the remainder must succeed: %v", err)
	}
	if res.Remainder.Intent != "itd-10" || res.Remainder.Status != spec.StatusOpen {
		t.Fatalf("the remainder must be an open spec on the same intent: %+v", res.Remainder)
	}
	if res.IntentMoved {
		t.Fatalf("the intent must stay planned while the remainder is open: %+v", res)
	}
	if len(res.OpenSpecs) != 1 || res.OpenSpecs[0] != res.Remainder.ID {
		t.Fatalf("the remainder must be the spec that holds the intent: %+v", res.OpenSpecs)
	}
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err != nil {
		t.Fatalf("intent must stay in planned/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, res.Remainder.Path)); err != nil {
		t.Fatalf("the remainder spec must exist on disk: %v", err)
	}
	// And the close that closes IT ships the intent, with the impact demanded
	// exactly there.
	last, err := Reconcile(root, res.Remainder.ID, "fix", RemainderRequest{})
	if err != nil {
		t.Fatalf("closing the remainder must ship the intent: %v", err)
	}
	if !last.IntentMoved || last.To != BucketShipped || len(last.OpenSpecs) != 0 {
		t.Fatalf("the last close must ship the intent: %+v", last)
	}
}

// An --impact at a close that mints a remainder is refused: that close ships
// nothing, so the judgement would be written against a record staying planned.
func TestReconcileRefusesImpactWithARemainder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	if _, err := Reconcile(root, "spc-1", "fix", RemainderRequest{Slug: "the-rest"}); err == nil {
		t.Fatal("--impact with a remainder must be refused")
	}
	// Nothing was minted and nothing moved.
	entries, err := os.ReadDir(filepath.Join(root, specsOpen))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("the refusal must mint nothing: %d specs in open/", len(entries))
	}
}

// TestPartialDeliveryResidualPassesRecordLint proves the state a partial close
// leaves — intent in planned/, one spec closed, one spec open, both naming it —
// is a state the record lint accepts, through the real lint engine over the
// fixture the verbs themselves produced. It is the guarantee that matters most
// here: the rule would be worthless if taking the honest path turned the gate
// red (adr-2609151513118583).
func TestPartialDeliveryResidualPassesRecordLint(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	pr, err := Plan(root, "itd-10", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Reconcile(root, pr.Spec.ID, "", RemainderRequest{Slug: "the-rest"}); err != nil {
		t.Fatal(err)
	}

	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
			"spec_lifecycle":   {Enabled: true, Severity: "blocker", IntentsDir: "intents", SpecsDir: "specs"},
			"spec_id_unique":   {Enabled: true, Severity: "blocker", IntentsDir: "intents", SpecsDir: "specs"},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, fnd := range findings {
		t.Errorf("a mid-delivery corpus must be lint-clean: %s:%d [%s] %s", fnd.File, fnd.Line, fnd.RuleID, fnd.Message)
	}
}

// openSpecFilesWithSlug lists the open spec files whose name ends in the given
// slug — the way a test counts how many remainders a close actually minted.
func openSpecFilesWithSlug(t *testing.T, root, slug string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, specsOpen, "spc-*-"+slug+".md"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

// F2: the store's answer to "which specs realise this intent" must be the
// lint's answer. A back-link written zero-padded (itd-007) names itd-7, and a
// literal compare instead ships the intent while the remainder is still open
// and leaves that remainder permanently unclosable.
func TestReconcileResolvesTheIntentBackLinkCanonically(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-7-alpha.md", plannedLinked("itd-7", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-7"))
	writeFile(t, root, specsOpen+"/spc-2-rest.md", specNaming("spc-2", "rest", "itd-007"))

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatalf("closing spc-1: %v", err)
	}
	if res.IntentMoved {
		t.Fatalf("itd-7 must stay planned while spc-2 (intent: itd-007) is open: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-7-alpha.md")); err != nil {
		t.Fatalf("intent must stay in planned/: %v", err)
	}

	res2, err := Reconcile(root, "spc-2", "", RemainderRequest{})
	if err != nil {
		t.Fatalf("closing spc-2 must resolve itd-007 to itd-7, not refuse: %v", err)
	}
	if !res2.IntentMoved || res2.To != BucketShipped {
		t.Fatalf("the close after which no open spec names itd-7 must ship it: %+v", res2)
	}
}

// F2: `intent link` must accept a back-link spelling the record lint calls
// green, or the spec is unlinkable for a padding difference alone.
func TestLinkResolvesACanonicallyEqualBackLink(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-7-alpha.md", plannedLinked("itd-7", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-2-rest.md", specNaming("spc-2", "rest", "itd-007"))

	if _, err := Link(root, "itd-7", "spc-2"); err != nil {
		t.Fatalf("Link must accept a canonically equal back-link: %v", err)
	}
}

// F3: a failure AFTER the mint strands the remainder. The retry is the same
// command, so it must REUSE the stranded remainder rather than mint a second.
func TestReconcileReusesAStrandedRemainderOnRetry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	// A directory at the close destination trips spec.Close's clobber guard,
	// which runs AFTER the mint — the exact window the retry has to survive.
	if err := os.MkdirAll(filepath.Join(root, specsClosed, "spc-1-alpha.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "rest"}); err == nil {
		t.Fatal("the close must fail on the clobber guard")
	}
	if got := openSpecFilesWithSlug(t, root, "rest"); len(got) != 1 {
		t.Fatalf("the failed close must leave exactly one remainder: %v", got)
	}

	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "rest"}); err == nil {
		t.Fatal("the retry fails at the same clobber guard")
	}
	if got := openSpecFilesWithSlug(t, root, "rest"); len(got) != 1 {
		t.Fatalf("the retry must reuse the stranded remainder, not mint another: %v", got)
	}
}

// F4: --remainder on a SHIPPED intent is refused before any mint. Accepting it
// mints on every invocation and produces the shipped-intent-with-an-open-spec
// shape invariant 17 forbids.
func TestReconcileRefusesARemainderOnAShippedIntent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	_, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "rest"})
	if err == nil {
		t.Fatal("--remainder on a shipped intent must be refused")
	}
	if !strings.Contains(err.Error(), "shipped") {
		t.Fatalf("the refusal must say why: %v", err)
	}
	if got := openSpecFilesWithSlug(t, root, "rest"); len(got) != 0 {
		t.Fatalf("nothing may be minted on the refusal: %v", got)
	}
	if _, err := os.Stat(filepath.Join(root, specsOpen, "spc-1-alpha.md")); err != nil {
		t.Fatalf("nothing may move on the refusal: %v", err)
	}
}

// F4: --remainder on an ALREADY-CLOSED spec is refused before any mint. The
// close is complete; a re-run is not a fresh delivery boundary.
func TestReconcileRefusesARemainderOnAClosedSpec(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsClosed+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	_, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "rest"})
	if err == nil {
		t.Fatal("--remainder on an already-closed spec must be refused")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Fatalf("the refusal must say why: %v", err)
	}
	if got := openSpecFilesWithSlug(t, root, "rest"); len(got) != 0 {
		t.Fatalf("nothing may be minted on the refusal: %v", got)
	}
}

// F5: the fidelity-audit request scopes to every spec the intent owned, not to
// the intent's scalar spec_id alone (adr-2609151513118583's audit consequence).
func TestAuditRequestNamesEverySpecThatRealisedTheIntent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsClosed+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, specsOpen+"/spc-2-rest.md", specNaming("spc-2", "rest", "itd-10"))

	res, err := Reconcile(root, "spc-2", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptID == "" {
		t.Fatalf("the shipping close must emit a receipt: %+v", res)
	}
	req, err := os.ReadFile(filepath.Join(root, ".abcd/.work.local/reviews", res.ReceiptID+".request.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(req)
	for _, want := range []string{"spc-1", "spc-2"} {
		if !strings.Contains(body, want) {
			t.Fatalf("the request must name %s among the specs that realised the intent:\n%s", want, body)
		}
	}
	if strings.Contains(body, "- spec: spc-1\n") {
		t.Fatalf("the request still scopes to the scalar spec_id alone:\n%s", body)
	}
}

// F5: the receipt stays keyed per intent — the audit is owed once, whatever the
// spec set — and a close against an already-shipped intent reports that the
// review is ALREADY owed rather than announcing a fresh one.
func TestReconcileReportsAnAlreadyOwedReviewAsSuch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	first, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if first.ReceiptStatus != "owed" {
		t.Fatalf("the close that ships owes the review: %q", first.ReceiptStatus)
	}
	again, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if again.ReceiptID != first.ReceiptID {
		t.Fatalf("the receipt is keyed per intent: %q vs %q", again.ReceiptID, first.ReceiptID)
	}
	if again.ReceiptStatus != "already_owed" {
		t.Fatalf("a re-run must not announce a fresh OWED review: %q", again.ReceiptStatus)
	}
}

// F3 (render half): the result distinguishes the invocation that MINTED the
// remainder from the retry that merely found it, so the surface cannot report a
// record it did not write as one it just wrote.
func TestReconcileSaysWhichInvocationMintedTheRemainder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	blocker := filepath.Join(root, specsClosed, "spc-1-alpha.md")
	if err := os.MkdirAll(blocker, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "rest"}); err == nil {
		t.Fatal("the close must fail on the clobber guard")
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{Slug: "rest"})
	if err != nil {
		t.Fatalf("the retry must complete: %v", err)
	}
	if res.Remainder.ID == "" {
		t.Fatalf("the retry must report the remainder it reused: %+v", res)
	}
	if res.RemainderMinted {
		t.Fatalf("this invocation minted nothing — it reused %s: %+v", res.Remainder.ID, res)
	}

	// The minting invocation says so.
	fresh := t.TempDir()
	writeFile(t, fresh, plannedDir+"/itd-11-beta.md", plannedLinked("itd-11", "beta", "spc-5"))
	writeFile(t, fresh, specsOpen+"/spc-5-beta.md", specNaming("spc-5", "beta", "itd-11"))
	first, err := Reconcile(fresh, "spc-5", "", RemainderRequest{Slug: "rest"})
	if err != nil {
		t.Fatal(err)
	}
	if !first.RemainderMinted {
		t.Fatalf("the close that wrote the remainder must say so: %+v", first)
	}
}
