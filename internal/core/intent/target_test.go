package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// withTarget is a record carrying `target_release: <v>` as its last
// frontmatter line, the place the writer inserts a key it has not seen.
func withTarget(content, v string) string {
	return strings.Replace(content, "\n---\n", "\ntarget_release: "+v+"\n---\n", 1)
}

// targetFindings are the record-lint findings that name the target key.
func targetFindings(t *testing.T, root string) []lint.Finding {
	t.Helper()
	var out []lint.Finding
	for _, f := range recordLintFindings(t, root) {
		if strings.Contains(f.Message, "target_release") {
			out = append(out, f)
		}
	}
	return out
}

// Criterion 1: given a planned intent, when `intent target <itd-N> v0.11.0`
// runs, then the record carries `target_release: v0.11.0` — one line, nothing
// else in the file touched — the loader reads it back, and a second target
// replaces the first and says what it replaced.
func TestTargetWritesTheFieldOnAPlannedIntent(t *testing.T) {
	root := t.TempDir()
	rel := plannedDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	before := readRec(t, root, rel)

	res, err := Target(root, "itd-10", "v0.11.0")
	if err != nil {
		t.Fatal(err)
	}
	if res.IntentID != "itd-10" || res.Path != rel || res.Target != "v0.11.0" || res.Previous != "" || !res.Written {
		t.Fatalf("TargetResult = %+v", res)
	}
	after := readRec(t, root, rel)
	if after != withTarget(before, "v0.11.0") {
		t.Fatalf("the verb must add exactly one line:\n--- before\n%s\n--- after\n%s", before, after)
	}
	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if it, _ := c.Lookup("itd-10"); it.TargetRelease != "v0.11.0" {
		t.Fatalf("the loader must read the target back: %+v", it)
	}
	if fs := targetFindings(t, root); len(fs) != 0 {
		t.Fatalf("a target on a planned intent is legal: %+v", fs)
	}

	res, err = Target(root, "itd-10", "next")
	if err != nil {
		t.Fatal(err)
	}
	if res.Previous != "v0.11.0" || res.Target != "next" || !res.Written {
		t.Fatalf("a retarget must name what it replaced: %+v", res)
	}
	if got := readRec(t, root, rel); got != withTarget(before, "next") {
		t.Fatalf("a retarget rewrites the one line in place:\n%s", got)
	}

	// The same target again writes nothing and says so.
	res, err = Target(root, "itd-10", "next")
	if err != nil {
		t.Fatal(err)
	}
	if res.Written || res.Previous != "next" {
		t.Fatalf("an unchanged target must report no write: %+v", res)
	}
}

// Criterion 1, the refusals: the verb writes only onto a planned intent and
// only a legal value; everything else is refused with nothing written, and a
// draft is sent to `intent plan --target`.
func TestTargetRefusesWhatItCannotMean(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		draftsDir + "/itd-10-alpha.md":    draftWithAC("itd-10", "alpha"),
		plannedDir + "/itd-11-beta.md":    plannedLinked("itd-11", "beta", "spc-1"),
		shippedDir + "/itd-3-done.md":     "---\nid: itd-3\nslug: done\nspec_id: spc-2\nkind: standalone\nimpact: additive\n---\n# done\n",
		supersededDir + "/itd-2-old.md":   "---\nid: itd-2\nslug: old\nspec_id: null\nkind: standalone\nsuperseded_by: itd-3\n---\n# old\n",
		disciplinesDir + "/itd-1-rule.md": "---\nid: itd-1\nslug: rule\nspec_id: null\nkind: discipline\n---\n# rule\n",
		specsOpen + "/spc-1-beta.md":      specNaming("spc-1", "beta", "itd-11"),
		specsClosed + "/spc-2-done.md":    specNaming("spc-2", "done", "itd-3"),
	}
	for rel, body := range files {
		writeFile(t, root, rel, body)
	}
	snapshot := func() map[string]string {
		out := map[string]string{}
		for rel := range files {
			out[rel] = readRec(t, root, rel)
		}
		return out
	}
	before := snapshot()

	cases := []struct {
		id, version, want string
	}{
		{"itd-10", "v0.11.0", "intent plan itd-10 --target"},
		{"itd-3", "v0.11.0", "shipped"},
		{"itd-2", "v0.11.0", "superseded"},
		{"itd-1", "v0.11.0", "disciplines"},
		{"itd-11", "0.11.0", "vX.Y.Z"},
		{"itd-11", "v0.11.0-rc.1", "vX.Y.Z"},
		{"itd-11", "soon", "vX.Y.Z"},
		{"itd-99", "v0.11.0", "not found"},
		{"ITD-11", "v0.11.0", "must match"},
	}
	for _, tc := range cases {
		_, err := Target(root, tc.id, tc.version)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("Target(%s, %q) = %v, want a refusal naming %q", tc.id, tc.version, err, tc.want)
		}
	}
	after := snapshot()
	for rel, b := range before {
		if after[rel] != b {
			t.Errorf("a refused target changed %s", rel)
		}
	}
}

// Criterion 1, the lint half: a target on a shipped or superseded intent is
// refused by record_schema, and so is a value the verb would never write.
func TestTheRecordLintRefusesATargetOnAShippedOrSupersededIntent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-3-done.md",
		withTarget("---\nid: itd-3\nslug: done\nspec_id: spc-2\nkind: standalone\nimpact: additive\n---\n# done\n", "v0.11.0"))
	writeFile(t, root, specsClosed+"/spc-2-done.md", specNaming("spc-2", "done", "itd-3"))
	writeFile(t, root, supersededDir+"/itd-2-old.md",
		withTarget("---\nid: itd-2\nslug: old\nspec_id: null\nkind: standalone\nsuperseded_by: itd-3\n---\n# old\n", "next"))
	writeFile(t, root, plannedDir+"/itd-11-beta.md", withTarget(plannedLinked("itd-11", "beta", "spc-1"), "0.11"))
	writeFile(t, root, specsOpen+"/spc-1-beta.md", specNaming("spc-1", "beta", "itd-11"))
	writeFile(t, root, plannedDir+"/itd-12-gamma.md", withTarget(plannedLinked("itd-12", "gamma", "spc-4"), "v0.12.0"))
	writeFile(t, root, specsOpen+"/spc-4-gamma.md", specNaming("spc-4", "gamma", "itd-12"))

	got := map[string]string{}
	for _, f := range targetFindings(t, root) {
		got[filepath.Base(f.File)] = f.Message
		if f.RuleID != "record_schema" {
			t.Errorf("the target finding belongs to record_schema: %+v", f)
		}
	}
	for file, want := range map[string]string{
		"itd-3-done.md":  "shipped",
		"itd-2-old.md":   "superseded",
		"itd-11-beta.md": "vX.Y.Z",
	} {
		if !strings.Contains(got[file], want) {
			t.Errorf("%s: want a target_release finding naming %q, got %q", file, want, got[file])
		}
	}
	if msg, ok := got["itd-12-gamma.md"]; ok {
		t.Errorf("a legal target on a planned intent must pass: %s", msg)
	}
}

// `plan --target` writes the target in the same write that plans the draft,
// and a value it cannot write is refused before the spec is minted.
func TestPlanTargetStampsTheDraftAsItPlans(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	before := readRec(t, root, draftsDir+"/itd-10-alpha.md")

	if _, err := Plan(root, "itd-10", PlanOptions{Target: "soon"}); err == nil || !strings.Contains(err.Error(), "vX.Y.Z") {
		t.Fatalf("an illegal target must refuse the plan: %v", err)
	}
	if readRec(t, root, draftsDir+"/itd-10-alpha.md") != before || specCount(t, root) != 0 {
		t.Fatal("a refused target must leave the draft byte-identical with no spec minted")
	}

	res, err := Plan(root, "itd-10", PlanOptions{Target: "v0.11.0"})
	if err != nil {
		t.Fatal(err)
	}
	if res.TargetStamped != "v0.11.0" {
		t.Fatalf("the result must say the target was stamped: %+v", res)
	}
	if got := fmOf(t, root, res.Intent.Path)["target_release"].Value; got != "v0.11.0" {
		t.Fatalf("the planned record must carry the target, got %q", got)
	}
	if fs := targetFindings(t, root); len(fs) != 0 {
		t.Fatalf("the planned record is lint-clean: %+v", fs)
	}

	// On a record already planned, the target is the target verb's.
	_, err = Plan(root, "itd-10", PlanOptions{Target: "next"})
	if err == nil || !strings.Contains(err.Error(), "abcd intent target itd-10") {
		t.Fatalf("plan --target on a planned record must name the target verb: %v", err)
	}
}

// A target is a claim about an unshipped record, so every move that takes a
// record out of planned/ drops it: the close that ships it, the bundle close,
// and the supersession. Otherwise the record lint would refuse the very tree
// the verb produced.
func TestLeavingPlannedDropsTheTarget(t *testing.T) {
	t.Run("spec close", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, plannedDir+"/itd-10-alpha.md", withTarget(plannedLinked("itd-10", "alpha", "spc-1"), "v0.11.0"))
		writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
		want := plannedLinked("itd-10", "alpha", "spc-1")
		wantFM := want[:strings.Index(want, "\n---\n")]
		if _, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err != nil {
			t.Fatal(err)
		}
		// The close appends the review marker to the body, so the frontmatter is
		// what is compared: the planned record's, less the target line.
		if got := readRec(t, root, shippedDir+"/itd-10-alpha.md"); !strings.HasPrefix(got, wantFM+"\n---\n") {
			t.Fatalf("the shipped record's frontmatter must be the planned one without its target line:\n%s", got)
		}
		if fs := targetFindings(t, root); len(fs) != 0 {
			t.Fatalf("the close must leave no target finding: %+v", fs)
		}
	})
	t.Run("bundle close", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
		writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
		planned, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta", Impact: "additive"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Target(root, "itd-11", "next"); err != nil {
			t.Fatal(err)
		}
		if _, err := Reconcile(root, planned.Spec.ID, "", RemainderRequest{}); err != nil {
			t.Fatal(err)
		}
		if _, ok := fmOf(t, root, shippedDir+"/itd-11-beta.md")["target_release"]; ok {
			t.Fatal("the bundle close must drop the member's target")
		}
	})
	t.Run("supersession", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, plannedDir+"/itd-10-alpha.md",
			withTarget(strings.Replace(plannedLinked("itd-10", "alpha", "spc-1"), "kind: standalone\n", "kind: standalone\nreclassification_history: []\n", 1), "v0.11.0"))
		writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
		writeFile(t, root, draftsDir+"/itd-20-successor.md", draftWithAC("itd-20", "successor"))
		if _, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20", Reason: "absorbed by itd-20", Date: "2026-09-29"}); err != nil {
			t.Fatal(err)
		}
		if _, ok := fmOf(t, root, supersededDir+"/itd-10-alpha.md")["target_release"]; ok {
			t.Fatal("the supersession must drop the target")
		}
	})
}

// Targets lists the planned intents that carry a target — the list the
// preview and the cut report — sorted by id, flagging an illegal value
// rather than dropping it. A draft carrying one by hand is not listed: it is
// not planned, so nothing is due from it.
func TestTargetsListsPlannedIntentsThatCarryOne(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-12-gamma.md", withTarget(plannedLinked("itd-12", "gamma", "spc-4"), "next"))
	writeFile(t, root, plannedDir+"/itd-9-nine.md", withTarget(plannedLinked("itd-9", "nine", "spc-5"), "v0.11.0"))
	writeFile(t, root, plannedDir+"/itd-11-beta.md", withTarget(plannedLinked("itd-11", "beta", "spc-1"), "0.11"))
	writeFile(t, root, plannedDir+"/itd-13-plain.md", plannedLinked("itd-13", "plain", "spc-6"))
	writeFile(t, root, draftsDir+"/itd-14-draft.md", withTarget(draftWithAC("itd-14", "draft"), "next"))

	got, err := Targets(root)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, g := range got {
		ids = append(ids, g.ID+"="+g.Target)
	}
	if strings.Join(ids, ",") != "itd-9=v0.11.0,itd-11=0.11,itd-12=next" {
		t.Fatalf("Targets = %v", ids)
	}
	if got[1].Invalid == "" || got[0].Invalid != "" || got[2].Invalid != "" {
		t.Fatalf("only the illegal value is flagged: %+v", got)
	}
	if got[0].Path != plannedDir+"/itd-9-nine.md" {
		t.Fatalf("the path is repo-relative: %q", got[0].Path)
	}

	empty := t.TempDir()
	if err := os.MkdirAll(filepath.Join(empty, plannedDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := Targets(empty); err != nil || len(got) != 0 {
		t.Fatalf("no target, empty list: %v %v", got, err)
	}
}

// A target value in a shape no verb writes is never half-overwritten: the
// target verb and `plan --target` both refuse it with nothing written, and
// the listing carries it raw, flagged, rather than dropping it.
func TestAMalformedTargetIsRefusedNotOverwritten(t *testing.T) {
	root := t.TempDir()
	draft := strings.Replace(draftWithAC("itd-10", "alpha"), "kind: null\n", "kind: null\ntarget_release: [v0.11.0]\n", 1)
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draft)
	planned := strings.Replace(plannedLinked("itd-11", "beta", "spc-1"), "kind: standalone\n", "kind: standalone\ntarget_release: [next]\n", 1)
	writeFile(t, root, plannedDir+"/itd-11-beta.md", planned)
	writeFile(t, root, specsOpen+"/spc-1-beta.md", specNaming("spc-1", "beta", "itd-11"))

	if _, err := Plan(root, "itd-10", PlanOptions{Target: "v0.11.0"}); err == nil || !strings.Contains(err.Error(), "shape no verb writes") {
		t.Fatalf("plan --target over a malformed value must refuse: %v", err)
	}
	if readRec(t, root, draftsDir+"/itd-10-alpha.md") != draft || specCount(t, root) != 1 {
		t.Fatal("the refused plan must leave the draft byte-identical with no spec minted")
	}
	if _, err := Target(root, "itd-11", "next"); err == nil || !strings.Contains(err.Error(), "shape no verb writes") {
		t.Fatalf("target over a malformed value must refuse: %v", err)
	}
	if readRec(t, root, plannedDir+"/itd-11-beta.md") != planned {
		t.Fatal("the refused target wrote the record")
	}
	got, err := Targets(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Target != "[next]" || got[0].Invalid == "" {
		t.Fatalf("the listing must carry the malformed value raw and flagged: %+v", got)
	}
}
