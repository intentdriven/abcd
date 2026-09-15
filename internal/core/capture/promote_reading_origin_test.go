package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/provenance"
)

// readingFixtureRun is the run every reading fixture in this package ingests
// into, so a test asserting the origin pair can name the run half of it.
const readingFixtureRun = "rdg-2608300000000001"

// draftFields reads a minted or linked draft's frontmatter through the shared
// same-line scanner — the reader every record consumer uses, so a key it cannot
// see is a key that was not really written.
func draftFields(t *testing.T, repo, rel string) map[string]frontmatter.Field {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return frontmatter.Fields(strings.Split(string(data), "\n"))
}

// provenanceLintFindings runs the shipped record-provenance rule over a fixture
// repository: the intent store and the ledger's readings store, armed alone, so
// a finding can only come from the rule under test.
func provenanceLintFindings(t *testing.T, repo string) []lint.Finding {
	t.Helper()
	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"record_provenance": {
				Enabled:  true,
				Severity: "blocker",
				RecordStores: map[string]string{
					"itd": intent.IntentsRelDir,
					"rdi": LedgerRelPath + "/readings",
				},
			},
		},
	}
	findings, err := lint.Lint(cfg, repo)
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	var out []lint.Finding
	for _, f := range findings {
		if f.RuleID == "record_provenance" {
			out = append(out, f)
		}
	}
	return out
}

// TestPromoteReadingItemStampsContributedByReading — framework 11.3 (linkage),
// and criteria 1 and 4 of itd-2609020625400169.
//
// Promotion is the one command that moves a reading item toward an intent, so it
// is the one command that can stamp where the intent came from without being
// told. The pair it writes is the pair it READ — the run is the directory the
// item sits in — which is the resolution half of the fourth criterion, and the
// shipped provenance lint is run over the result so "the lint resolves it" is
// exercised rather than inferred.
//
// The fixture item sits at the detection position: at widening the `accepted`
// disposition this route requires waits on the comparative run, and this test
// asserts nothing either way about that gate.
func TestPromoteReadingItemStampsContributedByReading(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)

	res, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item})
	if err != nil {
		t.Fatalf("Promote(%s): %v", item, err)
	}
	fields := draftFields(t, repo, res.IntentPath)
	want := "contributed-by-reading " + readingFixtureRun + "/" + item
	if got := fields[provenance.KeyOrigin].Value; got != want {
		t.Fatalf("promoted draft origin = %q, want %q", got, want)
	}
	if got := fields[provenance.KeyProductionMode].Value; got != string(provenance.DefaultMode) {
		t.Errorf("promoted draft production_mode = %q, want the default %q", got, provenance.DefaultMode)
	}
	if got := fields["promoted_from"].Value; got != item {
		t.Errorf("promoted draft promoted_from = %q, want %q", got, item)
	}

	// The value the lint reads must resolve: the run is the directory the item
	// sits in, which is exactly the join checkRecordProvenance performs.
	if fs := provenanceLintFindings(t, repo); len(fs) != 0 {
		t.Fatalf("the promoted draft is not clean under the provenance lint: %+v", fs)
	}
}

// TestPromoteReadingItemRefusesARunMismatch — criterion 4: the pair the mint
// stamps is the pair the store holds, so a record whose own `run` disagrees with
// the directory it sits in is a ledger fault rather than a choice between two
// answers. It refuses before anything is minted.
func TestPromoteReadingItemRefusesARunMismatch(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)
	path, err := findReadingItem(ir, item)
	if err != nil {
		t.Fatalf("findReadingItem: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const otherRun = "rdg-2608300000000009"
	rewritten := strings.Replace(string(content), `run: "`+readingFixtureRun+`"`, `run: "`+otherRun+`"`, 1)
	if rewritten == string(content) {
		t.Fatalf("the fixture record carries no `run` line naming %s to rewrite:\n%s", readingFixtureRun, content)
	}
	if err := os.WriteFile(path, []byte(rewritten), 0o644); err != nil {
		t.Fatal(err)
	}
	before := draftCount(t, repo)

	_, err = Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item})
	if err == nil {
		t.Fatal("a reading record whose run disagrees with its directory must be refused")
	}
	if !strings.Contains(err.Error(), readingFixtureRun) || !strings.Contains(err.Error(), otherRun) {
		t.Errorf("the refusal must name both runs; got %v", err)
	}
	if after := draftCount(t, repo); after != before {
		t.Fatalf("a refused promote minted a draft (%d -> %d); nothing is minted before the pair agrees", before, after)
	}
}

// TestPromoteReadingItemLinkWritesBothEdgesAndLeavesOriginAlone — criterion 2,
// and framework 7.1: an origin is stamped at mint and never rewritten. Link mode
// writes the back-edge on the draft and the forward stamp on the item, and
// touches neither disclosure key — so a hand-filed draft linked to a reading item
// stays researcher-authored and says so.
func TestPromoteReadingItemLinkWritesBothEdgesAndLeavesOriginAlone(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)
	draft, err := intent.CreateDraft(repo, intent.DraftOptions{
		Slug: "a-hand-filed-draft", Title: "A hand filed draft", SeedBody: "filed by hand",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	before := draftFields(t, repo, draft.Path)

	res, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item, LinkIntent: draft.ID})
	if err != nil {
		t.Fatalf("Promote --intent: %v", err)
	}
	if !res.Linked || res.IntentID != draft.ID {
		t.Fatalf("link mode must link the named draft, got %+v", res)
	}
	if res.BackEdgeKept != "" {
		t.Errorf("a draft with a free back-edge must report none kept, got %q", res.BackEdgeKept)
	}

	after := draftFields(t, repo, draft.Path)
	if got := after["promoted_from"].Value; got != item {
		t.Fatalf("the linked draft's promoted_from = %q, want %q", got, item)
	}
	if got := readingPromotedTo(t, ir, item); got != draft.ID {
		t.Fatalf("the item's promoted_to = %q, want %q", got, draft.ID)
	}
	// The two disclosure lines are byte-identical before and after.
	for _, key := range []string{provenance.KeyOrigin, provenance.KeyProductionMode} {
		if before[key].Value != after[key].Value {
			t.Errorf("%s moved from %q to %q; an origin is stamped at mint and never rewritten",
				key, before[key].Value, after[key].Value)
		}
	}
	if got := after[provenance.KeyOrigin].Value; got != string(provenance.KindResearcherAuthored) {
		t.Errorf("the linked draft's origin = %q, want it unmoved at %q", got, provenance.KindResearcherAuthored)
	}
}

// TestPromoteReadingItemLinkKeepsAnExistingBackEdge — the first scope condition:
// an intent occasioned by several items is promoted from ONE. A draft already
// naming another source keeps it, the second item's forward stamp is still
// written, and the operator is told which record stayed.
func TestPromoteReadingItemLinkKeepsAnExistingBackEdge(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)
	const firstItem = "rdi-2608300000000042"
	draft, err := intent.CreateDraft(repo, intent.DraftOptions{
		Slug: "occasioned-by-two-items", Title: "Occasioned by two items",
		SeedBody: "graduated from the first item", PromotedFrom: firstItem,
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	res, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item, LinkIntent: draft.ID})
	if err != nil {
		t.Fatalf("a taken back-edge is not a refusal on the reading route: %v", err)
	}
	if res.BackEdgeKept != firstItem {
		t.Errorf("the result reports back_edge kept %q, want %q", res.BackEdgeKept, firstItem)
	}
	fields := draftFields(t, repo, draft.Path)
	if got := fields["promoted_from"].Value; got != firstItem {
		t.Errorf("the draft's one back-edge moved to %q; it must stay at %q", got, firstItem)
	}
	if got := readingPromotedTo(t, ir, item); got != draft.ID {
		t.Errorf("the second item's promoted_to = %q, want %q; it still points forward", got, draft.ID)
	}
}

// TestPromoteReadingItemLinkCompletesOnRerunAfterAStampFailure: the draft write
// runs before the ledger-locked stamp, so a failure between the two leaves a
// draft naming the item and an item not yet naming the draft. Re-running the
// same command completes it — the draft write is idempotent and the stamp is the
// step that was missing.
func TestPromoteReadingItemLinkCompletesOnRerunAfterAStampFailure(t *testing.T) {
	repo, ir, item := dispositionedReadingFixture(t)
	draft, err := intent.CreateDraft(repo, intent.DraftOptions{
		Slug: "a-draft-to-complete", Title: "A draft to complete", SeedBody: "filed by hand",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	orig := stampWriteHook
	stampWriteHook = func(string, []byte) error { return errors.New("forced stamp failure") }
	t.Cleanup(func() { stampWriteHook = orig })

	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item, LinkIntent: draft.ID}); err == nil {
		t.Fatal("the forced stamp failure must surface")
	}
	if got := draftFields(t, repo, draft.Path)["promoted_from"].Value; got != item {
		t.Fatalf("the draft write runs before the stamp, so the back-edge must be present; got %q", got)
	}
	if got := readingPromotedTo(t, ir, item); got != "" {
		t.Fatalf("the item was stamped despite the forced failure: %q", got)
	}

	stampWriteHook = orig
	res, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item, LinkIntent: draft.ID})
	if err != nil {
		t.Fatalf("the re-run must complete the join: %v", err)
	}
	if res.BackEdgeKept != "" {
		t.Errorf("the re-run's back-edge names the same item, which is a no-op, not a kept edge: %q", res.BackEdgeKept)
	}
	if got := readingPromotedTo(t, ir, item); got != draft.ID {
		t.Fatalf("the item's promoted_to = %q, want %q", got, draft.ID)
	}
}

// readingPromotedTo reads one reading item's forward stamp.
func readingPromotedTo(t *testing.T, ir, item string) string {
	t.Helper()
	path, err := findReadingItem(ir, item)
	if err != nil {
		t.Fatalf("findReadingItem: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm, _, err := parseFrontmatterAndBody(string(content))
	if err != nil {
		t.Fatalf("parse reading record: %v", err)
	}
	return asString(fm["promoted_to"])
}
