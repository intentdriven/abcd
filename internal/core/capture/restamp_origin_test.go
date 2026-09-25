package capture

import (
	"os"
	"strings"
	"testing"
)

// rewriteDisclosure rewrites a record's origin line to origin, and drops the
// production_mode line when dropMode is set.
func rewriteDisclosure(t *testing.T, issuesRoot, id, origin string, dropMode bool) {
	t.Helper()
	src, _, err := findIssue(issuesRoot, id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "origin:") && origin != "":
			line = "origin: " + origin
		case strings.HasPrefix(line, "production_mode:") && dropMode:
			continue
		}
		kept = append(kept, line)
	}
	if err := os.WriteFile(src, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRestampRefusesAnOriginOutsideTheVocabulary is iss-2608300941548519's
// first half: the restamp gate tested that an origin was PRESENT, so a record
// carrying an out-of-vocabulary origin accepted a restamp and was written as a
// pair no command produces. The gate now asks the origin parser, which is what
// "the pair is written together or not at all" means literally.
func TestRestampRefusesAnOriginOutsideTheVocabulary(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "odd"})
	if err != nil {
		t.Fatal(err)
	}
	rewriteDisclosure(t, ir, res.ID, "invented-by-hand", false)
	before := readRaw(t, ir, res.ID)
	_, err = Resolve(ResolveRequest{RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed",
		Impact: "fix", ProductionMode: "scribe-transcribed"})
	if err == nil || !strings.Contains(err.Error(), "invented-by-hand") {
		t.Fatalf("a restamp over an out-of-vocabulary origin must be refused naming it, got %v", err)
	}
	if readRaw(t, ir, res.ID) != before {
		t.Fatal("a refused restamp rewrote the record")
	}
}

// TestRestampOfALoneOriginCompletesThePair pins the second half as the stated
// behaviour: a record carrying a valid origin and no production_mode (itself a
// record_provenance blocker) is repaired into a clean pair by a restamp, which
// is a legal write — the pair a command writes.
func TestRestampOfALoneOriginCompletesThePair(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "lone"})
	if err != nil {
		t.Fatal(err)
	}
	rewriteDisclosure(t, ir, res.ID, "", true)
	if _, err := Resolve(ResolveRequest{RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed",
		Impact: "fix", ProductionMode: "scribe-transcribed"}); err != nil {
		t.Fatal(err)
	}
	fm := readLedgerFrontmatter(t, ir, res.ID)
	if fm["origin"] != "researcher-authored" || fm["production_mode"] != "scribe-transcribed" {
		t.Fatalf("the restamp did not complete the pair: %v", fm)
	}
}
