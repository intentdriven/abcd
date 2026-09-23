package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
)

// TestPromoteLinkModeWritesBothHalvesOnAnIssue — itd-4 AC3: the join is
// bidirectional, the intent's related_issues naming the issue and the issue's
// related_intents naming the intent. Link mode on the issue route used to stamp
// the issue alone, so a draft filed by hand and then linked carried no edge back
// and the drift check would report it one-sided. Both halves are written.
func TestPromoteLinkModeWritesBothHalvesOnAnIssue(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "an observation somebody already filed an intent for")
	draft, err := intent.CreateDraft(repo, intent.DraftOptions{
		Slug: "filed-by-hand", Title: "Filed by hand", SeedBody: "a hand-filed draft",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	res, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID, LinkIntent: draft.ID})
	if err != nil {
		t.Fatalf("Promote --intent: %v", err)
	}
	if !res.Linked {
		t.Fatal("link mode must report Linked=true")
	}
	if got := draftFields(t, repo, draft.Path)["related_issues"].Value; got != "["+issID+"]" {
		t.Fatalf("the linked draft's related_issues = %q, want [%s]", got, issID)
	}
	if got := strings.Join(readIssue(t, ir, issID).RelatedIntents, ","); got != draft.ID {
		t.Fatalf("the issue's related_intents = %q, want [%s]", got, draft.ID)
	}
}

// TestPromoteKeepsALooseRelatedIntentAndAppends — an issue may already name an
// intent it is merely related to (captured with one). That relation is not a
// promotion: the named intent does not name the issue back. Promote goes ahead,
// and the minted intent joins the list beside the loose one, which is kept.
func TestPromoteKeepsALooseRelatedIntentAndAppends(t *testing.T) {
	repo, ir := ledger(t)
	loose, err := intent.CreateDraft(repo, intent.DraftOptions{
		Slug: "a-loosely-related-intent", Title: "A loosely related intent", SeedBody: "filed by hand",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	cres, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "an observation near an existing intent", Severity: SeverityMinor,
		Category: "observation", Source: "user-observation", FoundDuring: "t",
		Slug: "near-an-intent", RelatedIntents: []string{loose.ID},
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: cres.ID})
	if err != nil {
		t.Fatalf("a loose related intent is not a promotion, so promote must proceed: %v", err)
	}
	if got, want := strings.Join(readIssue(t, ir, cres.ID).RelatedIntents, ","), loose.ID+","+res.IntentID; got != want {
		t.Fatalf("related_intents = %q, want %q (the loose relation kept, the promotion appended)", got, want)
	}

	// Now the issue IS promoted — the minted intent names it back — so a second
	// promote is refused and names that intent, not the loose one.
	_, err = Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: cres.ID})
	if err == nil || !strings.Contains(err.Error(), res.IntentID) {
		t.Fatalf("a second promote must be refused naming %s; got %v", res.IntentID, err)
	}
}

// TestPromoteRefusesARecordCarryingTheRetiredField — `promoted_to` is retired
// (itd-4 AC3). A ledger written by an older abcd still carries it until the
// migration runs; promote refuses such a record before anything is minted, and
// the refusal names the migration.
func TestPromoteRefusesARecordCarryingTheRetiredField(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "an old record")
	path, _, err := findIssue(ir, issID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	old := strings.Replace(string(data), "\n---\n", "\npromoted_to: itd-5\n---\n", 1)
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if err == nil {
		t.Fatal("promote over a record carrying promoted_to must be refused")
	}
	if !strings.Contains(err.Error(), "capture migrate") {
		t.Errorf("the refusal must name the migration; got %v", err)
	}
	if n := draftCount(t, repo); n != 0 {
		t.Fatalf("a refused promote minted %d draft(s)", n)
	}
	after, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != old {
		t.Fatalf("a refused promote rewrote the record:\n%s", after)
	}
}
