package intent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/spec"
)

const (
	draftsDir   = ".abcd/development/intents/drafts"
	plannedDir  = ".abcd/development/intents/planned"
	shippedDir  = ".abcd/development/intents/shipped"
	specsOpen   = ".abcd/development/specs/open"
	specsClosed = ".abcd/development/specs/closed"
)

// writeFile writes content to root/rel, creating parent directories.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// draftWithAC is a draft intent carrying a non-empty ## Acceptance Criteria
// section (the itd-1 gate Plan enforces).
func draftWithAC(id, slug string) string {
	return "---\n" +
		"id: " + id + "\n" +
		"slug: " + slug + "\n" +
		"spec_id: null\n" +
		"kind: null\n" +
		"---\n" +
		"# " + slug + "\n\n" +
		"## Acceptance Criteria\n\n" +
		"- **Given** a user, **when** they act, **then** it works.\n"
}

func TestLoadCorpus(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, plannedDir+"/itd-2-beta.md",
		"---\nid: itd-2\nslug: beta\nspec_id: spc-1\nkind: standalone\n---\n# beta\n")

	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Intents) != 2 {
		t.Fatalf("expected 2 intents, got %d: %+v", len(c.Intents), c.Intents)
	}
	it, ok := c.Lookup("itd-2")
	if !ok || it.Bucket != "planned" || it.SpecID != "spc-1" || it.Kind != "standalone" {
		t.Fatalf("Lookup(itd-2) = %+v, %v", it, ok)
	}
	if it, ok := c.Lookup("itd-10"); !ok || it.Bucket != "drafts" {
		t.Fatalf("Lookup(itd-10) = %+v, %v", it, ok)
	}
}

func TestLoadMissingDirIsEmpty(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load on missing intents dir must be soft: %v", err)
	}
	if len(c.Intents) != 0 {
		t.Fatalf("expected empty corpus, got %+v", c.Intents)
	}
}

func TestLoadMalformedIsHardError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-1-broken.md", "# no frontmatter, no id\n")
	if _, err := Load(root); err == nil {
		t.Fatal("Load must hard-error on an intent file with no well-formed id")
	}
}

func TestPlanHappyPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))

	res, err := Plan(root, "itd-10", "")
	if err != nil {
		t.Fatal(err)
	}
	if !nativeSpecIDRe.MatchString(res.Spec.ID) || res.Spec.Intent != "itd-10" {
		t.Fatalf("Plan spec = %+v", res.Spec)
	}
	if res.Intent.Bucket != "planned" || res.Intent.SpecID != res.Spec.ID || res.Intent.Kind != "standalone" {
		t.Fatalf("Plan intent = %+v", res.Intent)
	}

	// The draft file is gone; the planned file exists with both link sides.
	if _, err := os.Stat(filepath.Join(root, draftsDir, "itd-10-alpha.md")); !os.IsNotExist(err) {
		t.Fatal("draft file should be gone after Plan")
	}
	body, err := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatalf("planned file should exist: %v", err)
	}
	f := frontmatter.Fields(strings.Split(string(body), "\n"))
	if f["spec_id"].Value != res.Spec.ID {
		t.Fatalf("planned intent spec_id = %q, want %s\n%s", f["spec_id"].Value, res.Spec.ID, body)
	}
	if f["kind"].Value != "standalone" {
		t.Fatalf("planned intent kind = %q, want standalone\n%s", f["kind"].Value, body)
	}
	// The spec file carries the reciprocal intent link.
	sbody, err := os.ReadFile(filepath.Join(root, specsOpen, res.Spec.ID+"-alpha.md"))
	if err != nil {
		t.Fatalf("spec file should exist: %v", err)
	}
	if !strings.Contains(string(sbody), "intent: itd-10") {
		t.Fatalf("spec file missing reciprocal link:\n%s", sbody)
	}
}

// TestPlanResidualPassesRecordLint proves that a drafts->planned Plan leaves a
// frontmatter/bucket state the intent_lifecycle record-lint rule accepts — the
// DoD's "make record-lint stays green" guarantee, checked through the real lint
// engine over the fixture.
func TestPlanResidualPassesRecordLint(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	if _, err := Plan(root, "itd-10", ""); err != nil {
		t.Fatal(err)
	}

	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, fnd := range findings {
		if fnd.RuleID == "intent_lifecycle" {
			t.Fatalf("planned intent violates intent_lifecycle: %s:%d %s", fnd.File, fnd.Line, fnd.Message)
		}
	}
}

func TestPlanRefusesNoAcceptanceCriteria(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\n---\n# alpha\n\nNo AC section here.\n")

	if _, err := Plan(root, "itd-10", ""); err == nil {
		t.Fatal("Plan must refuse an intent with no Acceptance Criteria")
	}
	// Nothing moved, no spec minted.
	if _, err := os.Stat(filepath.Join(root, draftsDir, "itd-10-alpha.md")); err != nil {
		t.Fatal("draft must remain in place after refusal")
	}
	if _, err := os.Stat(filepath.Join(root, specsOpen)); !os.IsNotExist(err) {
		t.Fatal("no spec should be minted on refusal")
	}
}

func TestPlanRefusesEmptyAcceptanceCriteria(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\n---\n# alpha\n\n## Acceptance Criteria\n\n## Next Section\n\nbody\n")
	if _, err := Plan(root, "itd-10", ""); err == nil {
		t.Fatal("Plan must refuse an intent whose Acceptance Criteria section is empty")
	}
}

// TestPlanRefusesBulletlessAcceptanceCriteria (iss-67 C9) proves Plan refuses an
// Acceptance Criteria section that has prose but no top-level -/* bullet. The old
// gate accepted any non-blank line, but the ingest gate counts only bullets — so
// such an intent planned, then dead-lettered every fidelity verdict forever
// (zero criteria to map). Plan and ingest must agree on "has criteria".
func TestPlanRefusesBulletlessAcceptanceCriteria(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\n---\n# alpha\n\n## Acceptance Criteria\n\nThe system should just work well.\n")
	if _, err := Plan(root, "itd-10", ""); err == nil {
		t.Fatal("Plan must refuse an Acceptance Criteria section with no top-level bullet")
	}
	if _, err := os.Stat(filepath.Join(root, draftsDir, "itd-10-alpha.md")); err != nil {
		t.Fatal("draft must remain in place after refusal")
	}
}

// TestPlanRefusesNonDraft: only a draft can be PLANNED. (A planned record has
// its own path — the stamp-only step — covered in claims_test.go; every other
// bucket is refused outright.)
func TestPlanRefusesNonDraft(t *testing.T) {
	for _, dir := range []string{shippedDir, disciplinesDir, supersededDir} {
		t.Run(dir, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, dir+"/itd-10-alpha.md",
				"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
			if _, err := Plan(root, "itd-10", ""); err == nil {
				t.Fatal("Plan must refuse an intent that is not in drafts/")
			}
		})
	}
}

// TestPlanReusesExistingSpecForIntent proves Plan is retry-safe: when a spec
// already realises the intent (e.g. a prior Plan minted it but the
// drafts->planned rename failed, leaving the intent a null-spec_id draft),
// re-running Plan reuses that spec instead of minting a duplicate.
func TestPlanReusesExistingSpecForIntent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	// Pre-create the spec for this intent; the draft is still an unlinked draft.
	sp, err := spec.Create(root, "itd-10", "alpha", "")
	if err != nil {
		t.Fatal(err)
	}

	res, err := Plan(root, "itd-10", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Spec.ID != sp.ID {
		t.Fatalf("Plan minted a new spec %q, want reuse of %q", res.Spec.ID, sp.ID)
	}
	if res.Intent.Bucket != "planned" || res.Intent.SpecID != sp.ID {
		t.Fatalf("Plan intent = %+v", res.Intent)
	}

	// Exactly one spec realises the intent (no duplicate minted).
	store, err := spec.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, s := range store.Specs {
		if s.Intent == "itd-10" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 spec for itd-10, got %d: %+v", n, store.Specs)
	}
}

// TestPlanRefusesDraftWithSpecID rejects a draft whose frontmatter already
// carries a non-null spec_id (half-planned / lint-invalid): Plan must not mint
// another spec for it.
func TestPlanRefusesDraftWithSpecID(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	if _, err := Plan(root, "itd-10", ""); err == nil {
		t.Fatal("Plan must refuse a draft that already has a non-null spec_id")
	}
}

// TestPlanRefusesWhenPlannedTargetExists proves Plan fails closed rather than
// clobbering a same-name file already sitting in planned/.
func TestPlanRefusesWhenPlannedTargetExists(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# pre-existing\n")
	if _, err := Plan(root, "itd-10", ""); err == nil {
		t.Fatal("Plan must refuse to overwrite an existing planned target")
	}
	// The pre-existing planned file is untouched.
	body, err := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "pre-existing") {
		t.Fatalf("planned target was clobbered:\n%s", body)
	}
}

func TestPlanRejectsBadID(t *testing.T) {
	root := t.TempDir()
	if _, err := Plan(root, "itd-../../etc", ""); err == nil {
		t.Fatal("Plan must reject a traversal id")
	}
}

func TestLinkHappyPath(t *testing.T) {
	root := t.TempDir()
	// A planned intent with no spec_id yet, and a spec that already realises it.
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n")
	writeFile(t, root, specsOpen+"/spc-3-alpha.md",
		"---\nid: spc-3\nslug: alpha\nintent: itd-10\n---\n# alpha\n")

	res, err := Link(root, "itd-10", "spc-3")
	if err != nil {
		t.Fatal(err)
	}
	if res.Intent.SpecID != "spc-3" {
		t.Fatalf("Link intent spec_id = %q, want spc-3", res.Intent.SpecID)
	}
	body, err := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if f := frontmatter.Fields(strings.Split(string(body), "\n")); f["spec_id"].Value != "spc-3" {
		t.Fatalf("spec_id not written: %q\n%s", f["spec_id"].Value, body)
	}
}

func TestLinkMismatchFails(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n")
	// The spec realises a DIFFERENT intent.
	writeFile(t, root, specsOpen+"/spc-3-other.md",
		"---\nid: spc-3\nslug: other\nintent: itd-99\n---\n# other\n")
	if _, err := Link(root, "itd-10", "spc-3"); err == nil {
		t.Fatal("Link must fail closed when the spec realises a different intent")
	}
}

func TestLinkMissingSpecFails(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n")
	if _, err := Link(root, "itd-10", "spc-9"); err == nil {
		t.Fatal("Link must fail when the spec does not exist")
	}
}

func TestStatusCounts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, plannedDir+"/itd-2-beta.md",
		"---\nid: itd-2\nslug: beta\nspec_id: spc-1\nkind: standalone\n---\n# beta\n")
	writeFile(t, root, specsOpen+"/spc-1-beta.md",
		"---\nid: spc-1\nslug: beta\nintent: itd-2\n---\n# beta\n")
	writeFile(t, root, specsClosed+"/spc-2-old.md",
		"---\nid: spc-2\nslug: old\nintent: itd-7\n---\n# old\n")

	v, err := Status(root)
	if err != nil {
		t.Fatal(err)
	}
	if v.Buckets["drafts"] != 1 || v.Buckets["planned"] != 1 {
		t.Fatalf("bucket counts = %+v", v.Buckets)
	}
	if v.SpecsOpen != 1 || v.SpecsClosed != 1 {
		t.Fatalf("spec counts open=%d closed=%d", v.SpecsOpen, v.SpecsClosed)
	}
	if len(v.Linked) != 1 || v.Linked[0].Intent != "itd-2" || v.Linked[0].Spec != "spc-1" {
		t.Fatalf("linked pairs = %+v", v.Linked)
	}
}

// plannedLinked is a planned intent already carrying both link sides (the shape
// Plan leaves): kind + spec_id set, ready to ship. It also declares an impact,
// because shipped/ requires one and `spec close` refuses to move a record that
// has neither its own judgement nor one on the flag (iss-126) — a fixture
// without one is a fixture that cannot ship, which impact_test.go tests on
// purpose and no other test here means to.
func plannedLinked(id, slug, specID string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nspec_id: " + specID + "\nkind: standalone\nimpact: fix\n---\n" +
		"# " + slug + "\n\n## Scope Conditions\n\n" + NullityToken +
		"\n\n## Acceptance Criteria\n\n- ok\n" + groundsSection + "\n## Audit Notes\n"
}

// specNaming is an open spec file whose intent: link names the given intent.
func specNaming(id, slug, intentID string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nintent: " + intentID + "\n---\n# " + slug + "\n"
}

func TestReconcileHappyPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IntentMoved || res.From != BucketPlanned || res.To != BucketShipped {
		t.Fatalf("Reconcile result = %+v", res)
	}
	if res.Intent.Bucket != BucketShipped || res.Intent.ID != "itd-10" {
		t.Fatalf("Reconcile intent = %+v", res.Intent)
	}
	if res.Spec.Status != spec.StatusClosed {
		t.Fatalf("Reconcile spec status = %q, want closed", res.Spec.Status)
	}
	// Intent moved planned -> shipped; spec moved open -> closed.
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); !os.IsNotExist(err) {
		t.Fatal("planned intent should be gone after reconcile")
	}
	if _, err := os.Stat(filepath.Join(root, shippedDir, "itd-10-alpha.md")); err != nil {
		t.Fatalf("shipped intent should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, specsOpen, "spc-1-alpha.md")); !os.IsNotExist(err) {
		t.Fatal("open spec should be gone after reconcile")
	}
	if _, err := os.Stat(filepath.Join(root, specsClosed, "spc-1-alpha.md")); err != nil {
		t.Fatalf("closed spec should exist: %v", err)
	}
	// The ship move parks an OWED fidelity-review stub in the Audit Notes.
	body, err := os.ReadFile(filepath.Join(root, shippedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "## Audit Notes") {
		t.Fatalf("Audit Notes heading should survive verbatim:\n%s", body)
	}
	if !strings.Contains(string(body), "abcd-review: OWED") {
		t.Fatalf("ship move should park an OWED stub in Audit Notes:\n%s", body)
	}
	if res.ReceiptID == "" {
		t.Fatal("ship move should emit a receipt id")
	}
}

// TestReconcileIdempotent proves a re-run on an already-shipped intent whose
// spec is already closed is a clean no-op/complete, not an error.
func TestReconcileIdempotent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatalf("second reconcile must be a clean no-op: %v", err)
	}
	if res.IntentMoved {
		t.Fatalf("second reconcile must not move the intent: %+v", res)
	}
	if res.Intent.Bucket != BucketShipped || res.Spec.Status != spec.StatusClosed {
		t.Fatalf("idempotent reconcile state = %+v", res)
	}
}

// TestReconcileClosesSpecWhenIntentAlreadyShipped covers the partial-failure
// recovery path: the intent already shipped but the spec is still open (a prior
// run moved the intent then failed before the close). Re-running just closes the
// spec.
func TestReconcileClosesSpecWhenIntentAlreadyShipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

	res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.IntentMoved {
		t.Fatalf("intent already shipped; must not move: %+v", res)
	}
	if res.Spec.Status != spec.StatusClosed {
		t.Fatalf("spec should be closed: %+v", res.Spec)
	}
}

func TestReconcileFailsNoIntentLink(t *testing.T) {
	root := t.TempDir()
	// A spec whose intent link is malformed cannot be minted by Create, so write a
	// spec whose intent names a non-existent intent to exercise the missing-intent path.
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-99"))
	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err == nil {
		t.Fatal("Reconcile must fail closed when the named intent does not exist")
	}
	// No partial move: the spec is untouched (still open).
	if _, err := os.Stat(filepath.Join(root, specsOpen, "spc-1-alpha.md")); err != nil {
		t.Fatal("spec must stay open after a fail-closed reconcile")
	}
}

// TestReconcileFailsWrongBucket refuses an intent that was never planned (still
// in drafts), with no partial move.
func TestReconcileFailsWrongBucket(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err == nil {
		t.Fatal("Reconcile must refuse an intent still in drafts")
	}
	if _, err := os.Stat(filepath.Join(root, draftsDir, "itd-10-alpha.md")); err != nil {
		t.Fatal("drafts intent must not move on a fail-closed reconcile")
	}
	if _, err := os.Stat(filepath.Join(root, specsOpen, "spc-1-alpha.md")); err != nil {
		t.Fatal("spec must stay open on a fail-closed reconcile")
	}
}

// TestReconcileFailsBidirectionalDrift refuses when the intent the spec names
// points back at a different spec (a one-sided link).
func TestReconcileFailsBidirectionalDrift(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-2"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err == nil {
		t.Fatal("Reconcile must refuse when the intent's spec_id disagrees with the spec")
	}
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err != nil {
		t.Fatal("planned intent must not move on drift")
	}
}

// TestReconcileAcceptsTwoSpecsOnOneIntent replaces the ambiguity refusal this
// verb used to raise. More than one spec naming one intent is the normal state
// under adr-2609151513118583, not an undetermined link: the close proceeds and
// the question that decides the intent's move is asked instead (see
// multispec_test.go for what it decides).
func TestReconcileAcceptsTwoSpecsOnOneIntent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, specsOpen+"/spc-2-alpha.md", specNaming("spc-2", "alpha", "itd-10"))
	if _, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err != nil {
		t.Fatalf("two specs realising one intent is the normal state, not a refusal: %v", err)
	}
}

func TestReconcileRejectsBadSpecID(t *testing.T) {
	if _, err := Reconcile(t.TempDir(), "spc-../../etc", "", RemainderRequest{}); err == nil {
		t.Fatal("Reconcile must reject a traversal spec id")
	}
}

func TestReconcileFailsMissingSpec(t *testing.T) {
	if _, err := Reconcile(t.TempDir(), "spc-9", "", RemainderRequest{}); err == nil {
		t.Fatal("Reconcile must fail when the spec does not exist")
	}
}

// TestFullCycle drives the real lifecycle: draft -> Plan -> Reconcile (spec
// close), asserting the intent ends in shipped/, the spec in closed/, BOTH
// lifecycle lint rules find zero issues, and a second reconcile is idempotent.
func TestFullCycle(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))

	pr, err := Plan(root, "itd-10", "")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// draftWithAC seeds no impact, so the close carries the judgement — and the
	// second close below re-runs with an empty one, over a record that now
	// records it, which is the idempotent shape.
	rr, err := Reconcile(root, pr.Spec.ID, "fix", RemainderRequest{})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if rr.Intent.Bucket != BucketShipped || rr.Spec.Status != spec.StatusClosed {
		t.Fatalf("cycle end state = %+v", rr)
	}
	if _, err := os.Stat(filepath.Join(root, shippedDir, "itd-10-alpha.md")); err != nil {
		t.Fatalf("intent must be shipped: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, specsClosed, pr.Spec.ID+"-alpha.md")); err != nil {
		t.Fatalf("spec must be closed: %v", err)
	}

	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
			"spec_lifecycle":   {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.RuleID == "intent_lifecycle" || f.RuleID == "spec_lifecycle" {
			t.Fatalf("post-cycle lifecycle finding: %s:%d %s %s", f.File, f.Line, f.RuleID, f.Message)
		}
	}

	// Second reconcile is a clean no-op.
	if _, err := Reconcile(root, pr.Spec.ID, "", RemainderRequest{}); err != nil {
		t.Fatalf("second reconcile must be idempotent: %v", err)
	}
}

// TestSetFrontmatterFieldsToleratesDelimiterTrailingSpace (B10) proves the writer
// agrees with frontmatter.Fields on delimiter tolerance: a closing delimiter that
// carries a trailing space ("--- ") — a shape the reader accepts — must still be
// found as the close, so absent keys are inserted INSIDE the frontmatter block and
// never into the body (where a "---" horizontal rule would otherwise be mistaken
// for the close and a body line spelled like a key silently rewritten).
func TestSetFrontmatterFieldsToleratesDelimiterTrailingSpace(t *testing.T) {
	content := strings.Join([]string{
		"--- ", // opening delimiter with a trailing space
		"id: itd-9",
		"--- ",   // closing delimiter with a trailing space (reader-valid)
		"# Body", // body
		"prose above and below a horizontal rule",
		"---", // a markdown thematic break in the body
		"kind: not-a-field",
	}, "\n")

	out, err := setFrontmatterFields(content, map[string]string{
		"kind":    "standalone",
		"spec_id": "spc-1",
	})
	if err != nil {
		t.Fatalf("setFrontmatterFields: %v", err)
	}

	// The reader must now see the inserted keys as real frontmatter fields.
	fields := frontmatter.Fields(strings.Split(out, "\n"))
	if got := fields["kind"]; got.Value != "standalone" {
		t.Fatalf("kind must land in the frontmatter block, got %+v\n---\n%s", got, out)
	}
	if got := fields["spec_id"]; got.Value != "spc-1" {
		t.Fatalf("spec_id must land in the frontmatter block, got %+v\n---\n%s", got, out)
	}
	// The keys must be inserted before the real (trailing-space) closing delimiter,
	// not after the body's horizontal rule.
	kindIdx := strings.Index(out, "kind: standalone")
	closeIdx := strings.Index(out, "--- \n# Body")
	if kindIdx < 0 || closeIdx < 0 || kindIdx > closeIdx {
		t.Fatalf("inserted keys must precede the closing delimiter, not enter the body\n---\n%s", out)
	}
}

// TestReconcileToleratesSpecIDSpelling proves the lifecycle verb yields to the
// record lint: record-lint compares a spec_id by NUMBER, so a slug suffix or a
// zero-padded id is lint-green — and a lint-green record must never be refused
// by the verb that ships it.
func TestReconcileToleratesSpecIDSpelling(t *testing.T) {
	for _, specID := range []string{"spc-1-alpha", "spc-01"} {
		t.Run(specID, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", specID))
			writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))

			res, err := Reconcile(root, "spc-1", "", RemainderRequest{})
			if err != nil {
				t.Fatalf("Reconcile must accept the lint-green spec_id %q: %v", specID, err)
			}
			if !res.IntentMoved || res.To != BucketShipped {
				t.Fatalf("Reconcile result = %+v", res)
			}
			// The audit emit reads the same stored spec_id, so it must tolerate it too.
			if res.AuditEmitError != "" {
				t.Fatalf("audit emit must accept spec_id %q: %s", specID, res.AuditEmitError)
			}
		})
	}
}

// TestLinkResolvesSpecByNumber proves the strict ^spc-[0-9]+$ ARGUMENT grammar
// still stands while the store lookup behind it compares by number, so a
// zero-padded spec record is linkable by its canonical id.
func TestLinkResolvesSpecByNumber(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n")
	writeFile(t, root, specsOpen+"/spc-01-alpha.md", specNaming("spc-01", "alpha", "itd-10"))

	res, err := Link(root, "itd-10", "spc-1")
	if err != nil {
		t.Fatalf("Link must resolve a zero-padded spec record: %v", err)
	}
	if res.Spec.ID != "spc-01" {
		t.Fatalf("Link spec = %+v, want the spc-01 record", res.Spec)
	}
	if _, err := Link(root, "itd-10", "spc-1-alpha"); err == nil {
		t.Fatal("Link must keep the strict ^spc-[0-9]+$ grammar for its argument")
	}
}

// TestSetPromotedFromWritesOnlyTheBackEdge — framework 7.1: `origin` is stamped
// at mint and never rewritten, so linking an existing draft to a reading item
// writes the back-edge and touches nothing else. A hand-filed draft linked to a
// reading item stays researcher-authored and says so.
func TestSetPromotedFromWritesOnlyTheBackEdge(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n"+
			"origin: researcher-authored\nproduction_mode: hand-written\n---\n# alpha\n")
	before, err := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := SetPromotedFrom(root, "itd-10", "rdi-17"); err != nil {
		t.Fatalf("SetPromotedFrom: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, plannedDir, "itd-10-alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	fields := frontmatter.Fields(strings.Split(string(after), "\n"))
	if got := fields["promoted_from"].Value; got != "rdi-17" {
		t.Fatalf("promoted_from = %q, want rdi-17\n%s", got, after)
	}
	// Every other frontmatter line is byte-identical: the disclosure pair above
	// all, which is what "the origin is unchanged" rests on.
	wantLines := frontmatterLines(t, string(before))
	gotLines := frontmatterLines(t, string(after))
	for _, line := range wantLines {
		if !containsLine(gotLines, line) {
			t.Errorf("SetPromotedFrom rewrote the frontmatter line %q", line)
		}
	}
	for _, line := range gotLines {
		if containsLine(wantLines, line) || line == "promoted_from: rdi-17" {
			continue
		}
		t.Errorf("SetPromotedFrom wrote an unexpected frontmatter line %q", line)
	}

	// An unknown intent and a source outside the two graduating families are
	// refused, and nothing is written.
	if _, err := SetPromotedFrom(root, "itd-99", "rdi-17"); err == nil {
		t.Error("SetPromotedFrom on an intent in no bucket must be refused")
	}
	if _, err := SetPromotedFrom(root, "itd-10", "adr-4"); err == nil {
		t.Error("SetPromotedFrom with a source outside ^(iss|rdi)-[0-9]+$ must be refused")
	}
}

// TestSetPromotedFromReportsATakenBackEdgeAndIsIdempotentOnTheSame — the first
// scope condition of itd-2609020625400169: an intent occasioned by several items
// is promoted from ONE, so a draft already naming another source keeps it and
// the caller is told, rather than the back-edge being silently overwritten.
func TestSetPromotedFromReportsATakenBackEdgeAndIsIdempotentOnTheSame(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-11-beta.md",
		"---\nid: itd-11\nslug: beta\nspec_id: null\nkind: null\npromoted_from: rdi-17\n"+
			"origin: contributed-by-reading rdg-3/rdi-17\nproduction_mode: hand-written\n---\n# beta\n")
	before, err := os.ReadFile(filepath.Join(root, draftsDir, "itd-11-beta.md"))
	if err != nil {
		t.Fatal(err)
	}

	// A different source: refused as a typed error naming the record already
	// there, and nothing is written.
	_, err = SetPromotedFrom(root, "itd-11", "rdi-18")
	if !errors.Is(err, ErrBackEdgeTaken) {
		t.Fatalf("SetPromotedFrom over a taken back-edge err = %v, want ErrBackEdgeTaken", err)
	}
	if !strings.Contains(err.Error(), "rdi-17") {
		t.Errorf("the refusal must name the record already there; got %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, draftsDir, "itd-11-beta.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("a refused SetPromotedFrom rewrote the record:\n%s", after)
	}

	// The SAME source is a no-op that reports the record unchanged.
	if _, err := SetPromotedFrom(root, "itd-11", "rdi-17"); err != nil {
		t.Fatalf("SetPromotedFrom with the source already there must be a no-op: %v", err)
	}
	again, err := os.ReadFile(filepath.Join(root, draftsDir, "itd-11-beta.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(before) {
		t.Errorf("an idempotent SetPromotedFrom rewrote the record:\n%s", again)
	}
}

// frontmatterLines returns the record's frontmatter block, line by line.
func frontmatterLines(t *testing.T, doc string) []string {
	t.Helper()
	_, rest, ok := strings.Cut(doc, "---\n")
	if !ok {
		t.Fatalf("the record carries no frontmatter block:\n%s", doc)
	}
	block, _, ok := strings.Cut(rest, "\n---")
	if !ok {
		t.Fatalf("the record's frontmatter block is unterminated:\n%s", doc)
	}
	return strings.Split(block, "\n")
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}
