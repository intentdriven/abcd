package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// draftBlockedBy is draftWithAC carrying a `blocked_by` line spelled as given.
func draftBlockedBy(id, slug, blockedBy string) string {
	return strings.Replace(draftWithAC(id, slug), "kind: null\n", "kind: null\n"+blockedBy+"\n", 1)
}

// fmOf reads a record's frontmatter fields.
func fmOf(t *testing.T, root, rel string) map[string]frontmatter.Field {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return frontmatter.Fields(strings.Split(string(body), "\n"))
}

// specCount is how many spec files the store holds, open or closed.
func specCount(t *testing.T, root string) int {
	t.Helper()
	store, err := spec.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return len(store.Specs)
}

// recordLintFindings runs the record rules a bundle touches over the fixture.
func recordLintFindings(t *testing.T, root string) []lint.Finding {
	t.Helper()
	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
			"spec_lifecycle":   {Enabled: true, Severity: "blocker", IntentsDir: "intents", SpecsDir: "specs"},
			"record_schema": {Enabled: true, Severity: "blocker", RecordStores: map[string]string{
				"itd": ".abcd/development/intents", "spc": ".abcd/development/specs", "adr": ".abcd/development/decisions/adrs",
			}},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	return findings
}

// Criterion 1: two drafts planned as one bundle share one spec naming both,
// carry kind bundle-member and the bundle's name, and reach planned/ together.
func TestPlanBundleMintsOneSharedSpecAndMovesBothMembers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))

	res, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Bundle != "alpha-beta" || len(res.Members) != 2 {
		t.Fatalf("PlanBundle = %+v", res)
	}
	if res.Spec.Intent != "itd-10" || strings.Join(res.Spec.Intents, ",") != "itd-10,itd-11" || res.Spec.Bundle != "alpha-beta" {
		t.Fatalf("shared spec = %+v", res.Spec)
	}
	if specCount(t, root) != 1 {
		t.Fatalf("a bundle mints ONE spec, found %d", specCount(t, root))
	}
	for _, m := range []struct{ id, slug string }{{"itd-10", "alpha"}, {"itd-11", "beta"}} {
		if _, err := os.Stat(filepath.Join(root, draftsDir, m.id+"-"+m.slug+".md")); !os.IsNotExist(err) {
			t.Fatalf("%s must have left drafts/", m.id)
		}
		f := fmOf(t, root, plannedDir+"/"+m.id+"-"+m.slug+".md")
		if f["kind"].Value != "bundle-member" || f["bundle"].Value != "alpha-beta" || f["spec_id"].Value != res.Spec.ID {
			t.Fatalf("%s planned frontmatter kind=%q bundle=%q spec_id=%q", m.id, f["kind"].Value, f["bundle"].Value, f["spec_id"].Value)
		}
	}
	for _, fnd := range recordLintFindings(t, root) {
		t.Errorf("a planned bundle must be record-lint clean: %s:%d [%s] %s", fnd.File, fnd.Line, fnd.RuleID, fnd.Message)
	}
}

// Criterion 1, the refusal: a member naming another in blocked_by is refused
// naming the edge, in either list spelling, and nothing moves or is minted.
func TestPlanBundleRefusesAMemberBlockedByAnother(t *testing.T) {
	for _, spelling := range []string{"blocked_by: [itd-10]", "blocked_by:\n  - itd-010"} {
		root := t.TempDir()
		writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
		writeFile(t, root, draftsDir+"/itd-11-beta.md", draftBlockedBy("itd-11", "beta", spelling))
		before, _ := os.ReadFile(filepath.Join(root, draftsDir, "itd-11-beta.md"))

		_, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta"})
		if err == nil {
			t.Fatalf("%q: a bundle containing its own blocker must be refused", spelling)
		}
		if !strings.Contains(err.Error(), "itd-11") || !strings.Contains(err.Error(), "blocked_by") || !strings.Contains(err.Error(), "itd-10") {
			t.Fatalf("%q: the refusal must name the edge: %v", spelling, err)
		}
		after, _ := os.ReadFile(filepath.Join(root, draftsDir, "itd-11-beta.md"))
		if string(before) != string(after) {
			t.Fatalf("%q: a refused bundle must leave the draft byte-identical", spelling)
		}
		if _, err := os.Stat(filepath.Join(root, draftsDir, "itd-10-alpha.md")); err != nil {
			t.Fatalf("%q: nothing may move on a refusal: %v", spelling, err)
		}
		if specCount(t, root) != 0 {
			t.Fatalf("%q: nothing may be minted on a refusal", spelling)
		}
	}
}

// Any refusal leaves nothing moved: one member that cannot be planned stops
// the whole bundle, before the mint.
func TestPlanBundleRefusalMovesNothing(t *testing.T) {
	cases := map[string]func(root string){
		"member without criteria": func(root string) {
			writeFile(t, root, draftsDir+"/itd-11-beta.md", "---\nid: itd-11\nslug: beta\nspec_id: null\nkind: null\n---\n# beta\n")
		},
		"member already planned": func(root string) {
			writeFile(t, root, plannedDir+"/itd-11-beta.md", plannedLinked("itd-11", "beta", "spc-9"))
			writeFile(t, root, specsOpen+"/spc-9-beta.md", specNaming("spc-9", "beta", "itd-11"))
		},
		"member held": func(root string) {
			writeFile(t, root, draftsDir+"/itd-11-beta.md", strings.Replace(draftWithAC("itd-11", "beta"), "kind: null\n", "kind: null\nheld: \"waiting\"\n", 1))
		},
		"member with a planned twin at the destination": func(root string) {
			writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
			writeFile(t, root, plannedDir+"/itd-11-beta.md", "---\nid: itd-11\nslug: beta\nspec_id: null\nkind: standalone\n---\n# pre-existing\n")
		},
		"member in another bundle": func(root string) {
			writeFile(t, root, draftsDir+"/itd-11-beta.md", strings.Replace(draftWithAC("itd-11", "beta"), "kind: null\n", "kind: bundle-member\nbundle: other\n", 1))
		},
	}
	for name, plant := range cases {
		root := t.TempDir()
		writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
		plant(root)
		specsBefore := specCount(t, root)
		before, _ := os.ReadFile(filepath.Join(root, draftsDir, "itd-10-alpha.md"))
		if _, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta"}); err == nil {
			t.Errorf("%s: PlanBundle must refuse", name)
			continue
		}
		after, err := os.ReadFile(filepath.Join(root, draftsDir, "itd-10-alpha.md"))
		if err != nil || string(before) != string(after) {
			t.Errorf("%s: the plannable member must stay in drafts/ byte-identical (%v)", name, err)
		}
		if specCount(t, root) != specsBefore {
			t.Errorf("%s: a refused bundle must mint nothing", name)
		}
	}
}

// The bundle is named by the person planning it, the name is a slug, and a
// name another record already carries names another bundle.
func TestPlanBundleRefusesAMissingOrTakenName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
	writeFile(t, root, plannedDir+"/itd-12-gamma.md", strings.Replace(plannedLinked("itd-12", "gamma", "spc-9"), "kind: standalone\n", "kind: bundle-member\nbundle: taken\n", 1))
	for _, name := range []string{"", "Not A Slug", "taken"} {
		if _, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: name}); err == nil {
			t.Errorf("bundle name %q must be refused", name)
		}
	}
	if _, err := PlanBundle(root, []string{"itd-10"}, BundleOptions{Bundle: "solo"}); err == nil {
		t.Error("a bundle of one member must be refused")
	}
	if _, err := PlanBundle(root, []string{"itd-10", "itd-010"}, BundleOptions{Bundle: "twice"}); err == nil {
		t.Error("a member named twice must be refused")
	}
	if _, err := os.Stat(filepath.Join(root, draftsDir, "itd-10-alpha.md")); err != nil {
		t.Fatalf("nothing may move: %v", err)
	}
}

// Criterion 2: closing a bundle's shared spec ships every member with that
// bundle together, each under the same impact rule.
func TestReconcileShipsEveryBundleMemberTogether(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
	planned, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta"})
	if err != nil {
		t.Fatal(err)
	}

	res, err := Reconcile(root, planned.Spec.ID, "additive", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IntentMoved || len(res.Members) != 2 {
		t.Fatalf("the close must ship both members: %+v", res)
	}
	for _, m := range []string{"itd-10-alpha.md", "itd-11-beta.md"} {
		f := fmOf(t, root, shippedDir+"/"+m)
		if f["impact"].Value != "additive" {
			t.Fatalf("%s must ship with the impact stamped, got %q", m, f["impact"].Value)
		}
	}
	if _, err := os.Stat(filepath.Join(root, specsClosed, filepath.Base(planned.Spec.Path))); err != nil {
		t.Fatalf("the shared spec must be closed: %v", err)
	}
	for _, fnd := range recordLintFindings(t, root) {
		t.Errorf("a shipped bundle must be record-lint clean: %s:%d [%s] %s", fnd.File, fnd.Line, fnd.RuleID, fnd.Message)
	}
	// Idempotent: the re-run completes without moving anything again.
	again, err := Reconcile(root, planned.Spec.ID, "", RemainderRequest{})
	if err != nil || again.IntentMoved {
		t.Fatalf("a re-run of a finished bundle close must be a clean no-op: %+v %v", again, err)
	}
}

// Criterion 2, all together or not at all: a member the impact rule refuses
// stops the close before either member moves or the spec closes.
func TestReconcileBundleRefusalMovesNoMember(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", strings.Replace(draftWithAC("itd-11", "beta"), "kind: null\n", "kind: null\nimpact: fix\n", 1))
	planned, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Reconcile(root, planned.Spec.ID, "additive", RemainderRequest{}); err == nil {
		t.Fatal("a member whose recorded impact disagrees with --impact must refuse the whole close")
	}
	for _, m := range []string{"itd-10-alpha.md", "itd-11-beta.md"} {
		if _, err := os.Stat(filepath.Join(root, plannedDir, m)); err != nil {
			t.Fatalf("%s must stay planned: %v", m, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, planned.Spec.Path)); err != nil {
		t.Fatalf("the shared spec must stay open: %v", err)
	}
}

// The readiness gate reads a bundle member's link through the shared spec's
// member list: the second member is linked, not in disagreement.
func TestReadySeesTheSecondBundleMemberAsLinked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
	if _, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta"}); err != nil {
		t.Fatal(err)
	}
	res, err := Ready(root, "itd-11")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range res.Checks {
		if c.Name == CheckSpecLink && !c.OK {
			t.Fatalf("the second member's link must hold through the shared spec: %s", c.Detail)
		}
	}
}

// --remainder on a bundle's shared spec is refused before anything is minted
// or moved: a remainder is one intent's follow-on, and a bundle ships whole.
func TestReconcileBundleRefusesARemainder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
	planned, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta", Impact: "additive"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Reconcile(root, planned.Spec.ID, "", RemainderRequest{Slug: "the-rest"})
	if err == nil || !strings.Contains(err.Error(), "--remainder") {
		t.Fatalf("--remainder on a bundle spec must be refused naming the flag: %v", err)
	}
	if n := specCount(t, root); n != 1 {
		t.Errorf("a refused remainder must mint nothing: %d specs", n)
	}
	for _, rel := range []string{"itd-10-alpha.md", "itd-11-beta.md"} {
		if _, err := os.Stat(filepath.Join(root, plannedDir, rel)); err != nil {
			t.Errorf("%s must stay planned: %v", rel, err)
		}
	}
}

// A member back in drafts/ when its bundle's spec closes is refused, naming it
// and its bucket, before any member moves.
func TestReconcileBundleRefusesADraftMember(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
	planned, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta", Impact: "additive"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, plannedDir, "itd-11-beta.md"), filepath.Join(root, draftsDir, "itd-11-beta.md")); err != nil {
		t.Fatal(err)
	}
	_, err = Reconcile(root, planned.Spec.ID, "", RemainderRequest{})
	if err == nil || !strings.Contains(err.Error(), "itd-11") || !strings.Contains(err.Error(), "drafts") {
		t.Fatalf("a draft member must refuse the close, naming it and its bucket: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err != nil {
		t.Errorf("the planned member must not ship: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, planned.Spec.Path)); err != nil {
		t.Errorf("the shared spec must stay open: %v", err)
	}
}
