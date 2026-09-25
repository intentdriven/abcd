package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// linksResolveFindings runs record-lint's links_resolve rule over the durable
// record, the root the committed record-lint.json declares, and returns every
// finding it raises.
func linksResolveFindings(t *testing.T, root string) []lint.Finding {
	t.Helper()
	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"links_resolve": {Enabled: true, Severity: "blocker"},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	var out []lint.Finding
	for _, f := range findings {
		if f.RuleID == "links_resolve" {
			out = append(out, f)
		}
	}
	return out
}

func readRel(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// relinkFixture lays out the three link classes that broke by hand in the ship
// ceremony (iss-2608311127491949, iss-2609091732329046): a spec that is ALREADY
// closed linking the spec about to close through ../open/, an ADR and a plan
// linking the intent's planned/ path, and a draft linking both — plus the
// closing spec's own links, which were written from open/ and name a still-open
// sibling bare. Closing spc-2 ships itd-10, so both records move.
func relinkFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md",
		strings.Replace(plannedLinked("itd-10", "alpha", "spc-1"), "## Audit Notes\n",
			"The work is specified in [spc-2](../../specs/open/spc-2-rest.md).\n\n## Audit Notes\n", 1))
	writeFile(t, root, plannedDir+"/itd-11-other.md", plannedLinked("itd-11", "other", "spc-3"))
	writeFile(t, root, specsClosed+"/spc-1-alpha.md",
		specNaming("spc-1", "alpha", "itd-10")+"\nThe rest is [spc-2](../open/spc-2-rest.md).\n")
	writeFile(t, root, specsOpen+"/spc-2-rest.md",
		specNaming("spc-2", "rest", "itd-10")+
			"\nRealises [itd-10](../../intents/planned/itd-10-alpha.md#acceptance-criteria).\n"+
			"Follows [spc-1](../closed/spc-1-alpha.md) and sits beside [spc-3](spc-3-other.md).\n"+
			"A link that never resolved stays as written: [gone](spc-9-gone.md).\n")
	writeFile(t, root, specsOpen+"/spc-3-other.md",
		specNaming("spc-3", "other", "itd-11")+"\nSibling of [spc-2](spc-2-rest.md).\n")
	writeFile(t, root, ".abcd/development/decisions/adrs/0001-choice.md",
		"# choice\n\nSee [itd-10](../../intents/planned/itd-10-alpha.md) and [the spec](../../specs/open/spc-2-rest.md).\n"+
			"Unmoved: [itd-11](../../intents/planned/itd-11-other.md).\n")
	writeFile(t, root, ".abcd/development/plans/2026-01-01-plan.md",
		"# plan\n\n- [itd-10](../intents/planned/itd-10-alpha.md)\n\n[ref]: ../intents/planned/itd-10-alpha.md\n")
	writeFile(t, root, draftsDir+"/itd-12-draft.md",
		"---\nid: itd-12\nslug: draft\n---\n# draft\n\nBuilds on [itd-10](../planned/itd-10-alpha.md) via [spc-2](../../specs/open/spc-2-rest.md).\n")
	writeFile(t, root, ".abcd/work/CONTEXT.md",
		"# context\n\nShipping [itd-10](../development/intents/planned/itd-10-alpha.md).\n")
	return root
}

// Closing a spec moves the spec (open/ -> closed/) and, on the last close, its
// intent (planned/ -> shipped/). The close leaves a tree record-lint accepts:
// every link that named either record's old path, from any folder, is repointed
// in the same operation — the already-closed sibling spec included.
func TestReconcileRepointsLinksToTheMovedRecords(t *testing.T) {
	root := relinkFixture(t)
	// The fixture's one deliberate dead link is the baseline: it is not the
	// close's to repair, and it proves the rewrite leaves a link alone when its
	// target never resolved.
	if got := linksResolveFindings(t, root); len(got) != 1 {
		t.Fatalf("fixture baseline: want exactly the one deliberate dead link, got %+v", got)
	}

	res, err := Reconcile(root, "spc-2", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.RelinkError != "" {
		t.Fatalf("relink error: %s", res.RelinkError)
	}
	// The close reports what it rewrote: one entry per link, the already-closed
	// sibling's among them.
	if len(res.Relinked) != 13 {
		t.Errorf("want 13 rewrites reported, got %d: %+v", len(res.Relinked), res.Relinked)
	}
	sawSibling := false
	for _, rw := range res.Relinked {
		if rw.File == specsClosed+"/spc-1-alpha.md" && rw.From == "../open/spc-2-rest.md" && rw.To == "spc-2-rest.md" {
			sawSibling = true
		}
	}
	if !sawSibling {
		t.Errorf("the closed sibling's rewrite is not reported: %+v", res.Relinked)
	}

	got := linksResolveFindings(t, root)
	if len(got) != 1 || !strings.Contains(got[0].Message, "spc-9-gone.md") {
		t.Fatalf("after spec close only the pre-existing dead link may remain; links_resolve found %+v", got)
	}

	for rel, want := range map[string][]string{
		specsClosed + "/spc-1-alpha.md":                   {"[spc-2](spc-2-rest.md)"},
		specsClosed + "/spc-2-rest.md":                    {"(../../intents/shipped/itd-10-alpha.md#acceptance-criteria)", "(spc-1-alpha.md)", "(../open/spc-3-other.md)", "(spc-9-gone.md)"},
		specsOpen + "/spc-3-other.md":                     {"(../closed/spc-2-rest.md)"},
		shippedDir + "/itd-10-alpha.md":                   {"(../../specs/closed/spc-2-rest.md)"},
		".abcd/development/decisions/adrs/0001-choice.md": {"(../../intents/shipped/itd-10-alpha.md)", "(../../specs/closed/spc-2-rest.md)", "(../../intents/planned/itd-11-other.md)"},
		".abcd/development/plans/2026-01-01-plan.md":      {"(../intents/shipped/itd-10-alpha.md)", "[ref]: ../intents/shipped/itd-10-alpha.md"},
		draftsDir + "/itd-12-draft.md":                    {"(../shipped/itd-10-alpha.md)", "(../../specs/closed/spc-2-rest.md)"},
		".abcd/work/CONTEXT.md":                           {"(../development/intents/shipped/itd-10-alpha.md)"},
	} {
		body := readRel(t, root, rel)
		for _, w := range want {
			if !strings.Contains(body, w) {
				t.Errorf("%s: want %q in\n%s", rel, w, body)
			}
		}
	}
}

// A re-run of a completed close is a clean completion, and it repoints what a
// failed earlier attempt left behind: the moves are derived from where the
// records are now.
func TestReconcileRerunRepointsWhatAnEarlierCloseLeft(t *testing.T) {
	root := relinkFixture(t)
	if _, err := Reconcile(root, "spc-2", "", RemainderRequest{}); err != nil {
		t.Fatal(err)
	}
	// An edit after the close reintroduces a link to the old path.
	writeFile(t, root, ".abcd/development/plans/late.md", "[late](../specs/open/spc-2-rest.md)\n")
	res, err := Reconcile(root, "spc-2", "", RemainderRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Relinked) != 1 || res.Relinked[0].To != "../specs/closed/spc-2-rest.md" {
		t.Fatalf("the re-run must repoint the stale link: %+v", res.Relinked)
	}
}

// Planning moves the draft drafts/ -> planned/, and every link that named the
// draft's path follows it (iss-2609250846525896).
func TestPlanRepointsLinksToTheDraft(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, ".abcd/development/decisions/adrs/0001-choice.md",
		"# choice\n\nSee [itd-10](../../intents/drafts/itd-10-alpha.md).\n")
	writeFile(t, root, draftsDir+"/itd-11-beta.md",
		"---\nid: itd-11\nslug: beta\n---\n# beta\n\nAfter [itd-10](itd-10-alpha.md).\n")

	res, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := linksResolveFindings(t, root); len(got) != 0 {
		t.Fatalf("after plan, links_resolve found %+v", got)
	}
	if !strings.Contains(readRel(t, root, ".abcd/development/decisions/adrs/0001-choice.md"), "(../../intents/planned/itd-10-alpha.md)") ||
		!strings.Contains(readRel(t, root, draftsDir+"/itd-11-beta.md"), "(../planned/itd-10-alpha.md)") {
		t.Fatal("links to the draft were not repointed at planned/")
	}
	if len(res.Relinked) != 2 || res.RelinkError != "" {
		t.Fatalf("plan must report the two rewrites: %+v %q", res.Relinked, res.RelinkError)
	}
}
