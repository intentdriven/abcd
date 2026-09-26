package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const adrsDir = ".abcd/development/decisions/adrs"

// readRec reads a fixture file.
func readRec(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(b)
}

// shippedRecord is a shipped standalone intent linked to a closed spec.
func shippedRecord(id, slug, specID string) string {
	return strings.Replace(plannedLinked(id, slug, specID), "impact: fix\n", "impact: fix\nreclassification_history: []\n", 1)
}

// Criterion 3: superseding writes the record's kind, shelf and links in one
// write — `superseded_by` on the record and `supersedes` on the successor
// together — and the result names the paths it moved.
func TestReclassifySupersededWritesBothDirectionsAndMoves(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", strings.Replace(plannedLinked("itd-10", "alpha", "spc-1"), "kind: standalone\n", "kind: standalone\nreclassification_history: []\n", 1))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, draftsDir+"/itd-20-successor.md", draftWithAC("itd-20", "successor"))

	res, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20", Reason: "absorbed by itd-20", Date: "2026-09-26"})
	if err != nil {
		t.Fatal(err)
	}
	if res.FromKind != KindStandalone || res.ToKind != KindSuperseded || res.Path != supersededDir+"/itd-10-alpha.md" {
		t.Fatalf("Reclassify = %+v", res)
	}
	if len(res.Moved) != 1 || res.Moved[0].From != plannedDir+"/itd-10-alpha.md" || res.Moved[0].To != supersededDir+"/itd-10-alpha.md" {
		t.Fatalf("the result must name the path moved: %+v", res.Moved)
	}
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); !os.IsNotExist(err) {
		t.Fatal("the record must leave planned/")
	}
	rec := readRec(t, root, supersededDir+"/itd-10-alpha.md")
	f := fmOf(t, root, supersededDir+"/itd-10-alpha.md")
	if f["superseded_by"].Value != "itd-20" || f["kind_at_supersession"].Value != KindStandalone || f["kind"].Value != KindStandalone {
		t.Fatalf("superseded frontmatter wrong:\n%s", rec)
	}
	if !strings.Contains(rec, `  - { date: 2026-09-26, from: standalone, to: superseded, reason: "absorbed by itd-20" }`) {
		t.Fatalf("the reclassification must be recorded in its history:\n%s", rec)
	}
	if !strings.Contains(rec, "# alpha\n\n> **Superseded by itd-20** on 2026-09-26: absorbed by itd-20\n") {
		t.Fatalf("the record must carry the supersession note under its title:\n%s", rec)
	}
	if got := fmOf(t, root, draftsDir+"/itd-20-successor.md")["supersedes"].Value; got != "[itd-10]" {
		t.Fatalf("the successor must name the record in supersedes, got %q", got)
	}
	for _, fnd := range recordLintFindings(t, root) {
		t.Errorf("a reclassified record must be record-lint clean: %s:%d [%s] %s", fnd.File, fnd.Line, fnd.RuleID, fnd.Message)
	}
}

// The successor may be an ADR, and an existing `supersedes` list — in either
// spelling — is appended to, never replaced.
func TestReclassifySupersededByAnADRAppendsToItsList(t *testing.T) {
	for _, tc := range []struct{ list, want string }{
		{"supersedes: [itd-3]", "supersedes: [itd-3, itd-10]"},
		{"supersedes:\n  - itd-3", "supersedes:\n  - itd-3\n  - itd-10"},
	} {
		root := t.TempDir()
		writeFile(t, root, draftsDir+"/itd-3-old.md", "---\nid: itd-3\n---\n# old\n")
		writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
		writeFile(t, root, adrsDir+"/0007-the-decision.md", "---\nid: adr-7\nslug: the-decision\nstatus: accepted\n"+tc.list+"\nsuperseded_by: null\n---\n# ADR-7: The decision\n")
		if _, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "adr-7", Reason: "decided instead", Date: "2026-09-26"}); err != nil {
			t.Fatalf("%q: %v", tc.list, err)
		}
		if adr := readRec(t, root, adrsDir+"/0007-the-decision.md"); !strings.Contains(adr, tc.want+"\n") {
			t.Fatalf("%q: the ADR's supersedes must gain the record:\n%s", tc.list, adr)
		}
		if f := fmOf(t, root, supersededDir+"/itd-10-alpha.md"); f["superseded_by"].Value != "adr-7" || f["kind_at_supersession"].Value != KindStandalone {
			t.Fatalf("%q: superseded_by must name the ADR", tc.list)
		}
	}
}

// Criterion 3, the refusal: a shipped intent never becomes a discipline; the
// refusal names the remedy, and nothing is written.
func TestReclassifyRefusesAShippedIntentBecomingADiscipline(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-10-alpha.md", shippedRecord("itd-10", "alpha", "spc-1"))
	before := readRec(t, root, shippedDir+"/itd-10-alpha.md")
	_, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindDiscipline, Reason: "it is a rule"})
	if err == nil || !strings.Contains(err.Error(), "file a discipline that supersedes it") {
		t.Fatalf("the refusal must name the remedy: %v", err)
	}
	if readRec(t, root, shippedDir+"/itd-10-alpha.md") != before {
		t.Fatal("a refused reclassify must write nothing")
	}
	// Nor does a shipped intent change kind to another delivery shape.
	if _, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindBundleMember, Bundle: "x"}); err == nil {
		t.Fatal("a shipped intent never changes kind")
	}
}

// A kind change on a draft or planned record keeps its shelf, writes the kind
// (and the bundle, or clears it), and records the change.
func TestReclassifyChangesKindInPlace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", strings.Replace(draftWithAC("itd-10", "alpha"), "kind: null\n", "kind: standalone\nreclassification_history: []\n", 1))
	writeFile(t, root, plannedDir+"/itd-12-gamma.md", strings.Replace(plannedLinked("itd-12", "gamma", "spc-9"), "kind: standalone\n", "kind: bundle-member\nbundle: shared\n", 1))
	writeFile(t, root, specsOpen+"/spc-9-gamma.md", specNaming("spc-9", "gamma", "itd-12"))

	res, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindBundleMember, Bundle: "shared", Reason: "delivered with itd-12", Date: "2026-09-26"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Moved) != 0 || res.Path != draftsDir+"/itd-10-alpha.md" {
		t.Fatalf("a kind change keeps the shelf: %+v", res)
	}
	f := fmOf(t, root, draftsDir+"/itd-10-alpha.md")
	if f["kind"].Value != KindBundleMember || f["bundle"].Value != "shared" {
		t.Fatalf("kind/bundle not written: %+v", f)
	}
	if !strings.Contains(readRec(t, root, draftsDir+"/itd-10-alpha.md"), `from: standalone, to: bundle-member, reason: "delivered with itd-12"`) {
		t.Fatal("the kind change must be recorded")
	}

	// Back to standalone clears the bundle.
	if _, err := Reclassify(root, "itd-12", ReclassifyRequest{Kind: KindStandalone, Date: "2026-09-26"}); err != nil {
		t.Fatal(err)
	}
	g := fmOf(t, root, plannedDir+"/itd-12-gamma.md")
	if g["kind"].Value != KindStandalone || g["bundle"].Value != "null" {
		t.Fatalf("standalone must clear the bundle: kind=%q bundle=%q", g["kind"].Value, g["bundle"].Value)
	}
}

// A bundle no other record names is not joined by a reclassify: that would be
// a bundle of one nobody planned.
func TestReclassifyRefusesJoiningABundleNobodyNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	before := readRec(t, root, draftsDir+"/itd-10-alpha.md")
	if _, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindBundleMember, Bundle: "lonely"}); err == nil {
		t.Fatal("joining a bundle no other record names must be refused")
	}
	if _, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindBundleMember}); err == nil {
		t.Fatal("--kind bundle-member without --bundle must be refused")
	}
	if readRec(t, root, draftsDir+"/itd-10-alpha.md") != before {
		t.Fatal("a refused reclassify must write nothing")
	}
}

// Every refusal writes nothing, on the record and on the successor alike.
func TestReclassifySupersededRefusals(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, draftsDir+"/itd-20-successor.md", draftWithAC("itd-20", "successor"))
	writeFile(t, root, supersededDir+"/itd-30-gone.md", "---\nid: itd-30\nslug: gone\nkind: standalone\nspec_id: null\nsuperseded_by: itd-20\nkind_at_supersession: standalone\n---\n# gone\n")
	writeFile(t, root, draftsDir+"/itd-40-held.md", strings.Replace(draftWithAC("itd-40", "held"), "kind: null\n", "kind: null\nheld: \"waiting\"\n", 1))
	rec, succ := readRec(t, root, plannedDir+"/itd-10-alpha.md"), readRec(t, root, draftsDir+"/itd-20-successor.md")

	for name, req := range map[string]struct {
		id  string
		req ReclassifyRequest
	}{
		"no successor":            {"itd-10", ReclassifyRequest{Kind: KindSuperseded, Reason: "r"}},
		"no reason":               {"itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20"}},
		"missing successor":       {"itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-99", Reason: "r"}},
		"missing ADR successor":   {"itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "adr-99", Reason: "r"}},
		"itself":                  {"itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-10", Reason: "r"}},
		"superseded successor":    {"itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-30", Reason: "r"}},
		"already superseded":      {"itd-30", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20", Reason: "r"}},
		"held record":             {"itd-40", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20", Reason: "r"}},
		"multi-line reason":       {"itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20", Reason: "a\nb"}},
		"unknown kind":            {"itd-10", ReclassifyRequest{Kind: "epic"}},
		"--by on a kind change":   {"itd-10", ReclassifyRequest{Kind: KindStandalone, By: "itd-20"}},
		"a draft to a discipline": {"itd-20", ReclassifyRequest{Kind: KindDiscipline, Reason: "r"}},
	} {
		if _, err := Reclassify(root, req.id, req.req); err == nil {
			t.Errorf("%s: Reclassify must refuse", name)
		}
	}
	if readRec(t, root, plannedDir+"/itd-10-alpha.md") != rec || readRec(t, root, draftsDir+"/itd-20-successor.md") != succ {
		t.Fatal("a refused reclassify must leave the record and the successor byte-identical")
	}
}

// Criterion 4: superseding one member of a bundle of two leaves the other a
// bundle-member whose record says the bundle now has one member; the retired
// member keeps the bundle it was in as bundle_at_supersession; the lint accepts
// both; and the shared spec's close ships the survivor alone.
func TestReclassifySupersedingABundleMemberLeavesASurvivorThatSaysSo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
	writeFile(t, root, draftsDir+"/itd-20-successor.md", draftWithAC("itd-20", "successor"))
	planned, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta", Impact: "additive"})
	if err != nil {
		t.Fatal(err)
	}

	res, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20", Reason: "absorbed", Date: "2026-09-26"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Survivor != "itd-11" {
		t.Fatalf("the result must name the survivor: %+v", res)
	}
	gone := fmOf(t, root, supersededDir+"/itd-10-alpha.md")
	if gone["kind_at_supersession"].Value != KindBundleMember || gone["bundle"].Value != "null" || gone["bundle_at_supersession"].Value != "alpha-beta" {
		t.Fatalf("the retired member must record the bundle it left: %+v", gone)
	}
	surv := fmOf(t, root, plannedDir+"/itd-11-beta.md")
	if surv["kind"].Value != KindBundleMember || surv["bundle"].Value != "alpha-beta" {
		t.Fatalf("the survivor stays a bundle-member of its bundle: %+v", surv)
	}
	if !strings.Contains(readRec(t, root, plannedDir+"/itd-11-beta.md"), "bundle alpha-beta now has one member") {
		t.Fatal("the survivor's record must state the bundle now has one member")
	}
	for _, fnd := range recordLintFindings(t, root) {
		t.Errorf("the survivor and the retired member must be record-lint clean: %s:%d [%s] %s", fnd.File, fnd.Line, fnd.RuleID, fnd.Message)
	}

	closed, err := Reconcile(root, planned.Spec.ID, "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(closed.Members) != 1 || closed.Members[0].Intent.ID != "itd-11" || len(closed.Skipped) != 1 || closed.Skipped[0] != "itd-10" {
		t.Fatalf("the close must ship the survivor and pass over the superseded member: %+v", closed)
	}
	if _, err := os.Stat(filepath.Join(root, shippedDir, "itd-11-beta.md")); err != nil {
		t.Fatalf("the survivor must ship: %v", err)
	}
}
