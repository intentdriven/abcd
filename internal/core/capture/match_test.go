package capture

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/record/match"
)

// The planted double: a finding, and the same finding filed again in other
// words. The fillers are records about other things, so the corpus has a
// vocabulary to weigh terms against.
const (
	plantedFinding = "The capture ledger reader silently skips a record whose frontmatter carries " +
		"a duplicated key, so the finding disappears from every listing without a warning."
	plantedDouble = "Capture ledger reader silently skips any record whose frontmatter carries a " +
		"duplicated key: the finding disappears from every listing, and no warning is printed."
	matchFiller1 = "The site builder renders a stale anchor for a heading renamed since the last build."
	matchFiller2 = "The history store drops a transcript that exceeds its byte budget without saying so."
	// A broad intent the narrow capture below refines.
	broadTitle = "Record readers agree with the lint gate"
	broadPress = "Record readers disagree with the lint gate. The capture ledger reader skips " +
		"records with duplicated frontmatter keys, the intent loader fails closed on a missing id, " +
		"the spec store tolerates unknown properties, the site builder renders stale anchors, " +
		"the memory store truncates pages, and the history store drops transcripts over its budget."
	narrowCapture = "The capture ledger reader skips records with duplicated frontmatter keys " +
		"while the lint gate reports them."
)

func captureText(t *testing.T, repo, ir, text string, m *match.Config) CaptureResult {
	t.Helper()
	res, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: text, Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Match: m,
	})
	if err != nil {
		t.Fatalf("capture refused: %v", err)
	}
	return res
}

func plantIntent(t *testing.T, repo, bucket, id, title, press string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, ".abcd/development/intents", bucket, id+"-planted.md"),
		"---\nid: "+id+"\nslug: planted\nspec_id: null\nkind: null\n---\n\n# "+title+
			"\n\n## Press Release\n\n> "+press+"\n\n## Why This Matters\n\nUnrelated words about budgets.\n")
}

func readRecord(t *testing.T, repo, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func bundled() *match.Config { c := match.Bundled(); return &c }

// Criterion 1: a capture that overlaps an existing issue above the threshold
// is written with a typed link naming it, and the result carries the match.
func TestCaptureLinksAPlantedDouble(t *testing.T) {
	repo, ir := ledger(t)
	first := captureText(t, repo, ir, plantedFinding, nil)
	captureText(t, repo, ir, matchFiller1, nil)
	captureText(t, repo, ir, matchFiller2, nil)

	res := captureText(t, repo, ir, plantedDouble, bundled())
	if res.Match == nil || len(res.Match.Matches) == 0 {
		t.Fatalf("no match reported: %+v", res.Match)
	}
	m := res.Match.Matches[0]
	if m.ID != first.ID || m.Relation != match.Duplicates || !m.Linked {
		t.Fatalf("match = %+v, want %s linked as duplicates", m, first.ID)
	}
	content := readRecord(t, repo, res.Path)
	if !strings.Contains(content, "\nduplicates: ["+first.ID+"]\n") {
		t.Fatalf("the record carries no duplicates link naming %s:\n%s", first.ID, content)
	}
	// The link reads back through the ledger reader.
	list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	for _, iss := range list.Issues {
		if iss.ID == res.ID {
			if len(iss.Duplicates) != 1 || iss.Duplicates[0] != first.ID {
				t.Fatalf("read back duplicates = %v", iss.Duplicates)
			}
			return
		}
	}
	t.Fatalf("the linked record %s is not listed (skipped: %+v)", res.ID, list.Skipped)
}

// The candidates include every intent's title and press release, and a
// narrower capture of a broader record is written as `refines`.
func TestCaptureRefinesABroaderIntent(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, matchFiller1, nil)
	captureText(t, repo, ir, matchFiller2, nil)
	plantIntent(t, repo, "planned", "itd-9", broadTitle, broadPress)

	res := captureText(t, repo, ir, narrowCapture, bundled())
	if res.Match == nil || len(res.Match.Matches) != 1 || res.Match.Matches[0].ID != "itd-9" ||
		res.Match.Matches[0].Relation != match.Refines {
		t.Fatalf("match = %+v, want itd-9 as refines", res.Match)
	}
	if content := readRecord(t, repo, res.Path); !strings.Contains(content, "\nrefines: [itd-9]\n") {
		t.Fatalf("no refines link:\n%s", content)
	}
}

// Criterion 4: below the threshold nothing is written and the near misses
// are listed with their scores.
func TestCaptureBelowTheThresholdWritesNothingAndListsNearMisses(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	captureText(t, repo, ir, matchFiller1, nil)
	cfg := match.Bundled()
	cfg.Threshold = 0.99
	res := captureText(t, repo, ir, narrowCapture, &cfg)
	if res.Match == nil || len(res.Match.Matches) != 0 {
		t.Fatalf("matched above a 0.99 threshold: %+v", res.Match)
	}
	if len(res.Match.NearMisses) == 0 || res.Match.NearMisses[0].Score <= 0 {
		t.Fatalf("near misses = %+v, want the finding with its score", res.Match.NearMisses)
	}
	content := readRecord(t, repo, res.Path)
	if regexp.MustCompile(`(?m)^(duplicates|refines):`).MatchString(content) {
		t.Fatalf("a link was written below the threshold:\n%s", content)
	}
}

// The compared fields are configuration: with issue bodies left out, the
// planted double is not a candidate.
func TestCaptureComparesOnlyTheConfiguredFields(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	cfg := match.Bundled()
	cfg.Fields = []string{match.FieldIntentTitle, match.FieldIntentPressRelease}
	res := captureText(t, repo, ir, plantedDouble, &cfg)
	if res.Match == nil || len(res.Match.Matches)+len(res.Match.NearMisses) != 0 {
		t.Fatalf("an issue was compared though issue.body is not a configured field: %+v", res.Match)
	}
}

// Criterion 3: a match that cannot run never refuses the capture; the record
// is filed and the result says why nothing was compared. A short text is
// filed without matching and says so.
func TestCaptureIsNeverRefusedByTheMatch(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	// An intent store no reader can load: a record whose id is malformed.
	writeFile(t, filepath.Join(repo, ".abcd/development/intents/drafts/itd-1-broken.md"), "---\nid: nonsense\n---\n")
	res := captureText(t, repo, ir, plantedDouble, bundled())
	if res.Match == nil || res.Match.Skipped == "" || len(res.Match.Matches) != 0 {
		t.Fatalf("an unreadable record set did not report a skipped match: %+v", res.Match)
	}
	if _, err := os.Stat(filepath.Join(repo, res.Path)); err != nil {
		t.Fatalf("the capture was not written: %v", err)
	}

	short := captureText(t, repo, ir, "Typo in the ledger README.", bundled())
	if short.Match == nil || !strings.Contains(short.Match.Skipped, "minimum") {
		t.Fatalf("a short capture did not say it was filed without matching: %+v", short.Match)
	}
}

// Criterion 3, second half: removing the link leaves an ordinary record, one
// every reader takes exactly as it takes a record that never had one.
func TestRemovingTheLinkLeavesAnOrdinaryRecord(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	captureText(t, repo, ir, matchFiller1, nil)
	res := captureText(t, repo, ir, plantedDouble, bundled())
	p := filepath.Join(repo, res.Path)
	content := readRecord(t, repo, res.Path)
	stripped := regexp.MustCompile(`(?m)^duplicates: .*\n`).ReplaceAllString(content, "")
	if stripped == content {
		t.Fatalf("no link to remove:\n%s", content)
	}
	if err := os.WriteFile(p, []byte(stripped), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Skipped) != 0 {
		t.Fatalf("the unlinked record is skipped: %+v", list.Skipped)
	}
	for _, iss := range list.Issues {
		if iss.ID == res.ID && len(iss.Duplicates)+len(iss.Refines) == 0 {
			return
		}
	}
	t.Fatalf("%s not read back as an ordinary record", res.ID)
}

// Without a Match the capture is exactly the one it always was.
func TestCaptureWithoutAMatchReportsNone(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, plantedFinding, nil)
	res := captureText(t, repo, ir, plantedDouble, nil)
	if res.Match != nil {
		t.Fatalf("an unrequested match ran: %+v", res.Match)
	}
	if strings.Contains(readRecord(t, repo, res.Path), "duplicates:") {
		t.Fatal("an unrequested match wrote a link")
	}
}

// A link names a record by its id: a value of any other shape is refused by
// the reader, like every other id list.
func TestALinkValueIsARecordID(t *testing.T) {
	fm := map[string]any{
		"schema_version": 1, "id": "iss-1", "slug": "a", "severity": "minor",
		"category": "bug", "source": "user-observation", "found_during": "t",
	}
	for _, key := range []string{"duplicates", "refines"} {
		for _, v := range []string{"iss-2", "itd-3"} {
			fm[key] = []string{v}
			if err := validateStrict(fm); err != nil {
				t.Fatalf("%s: [%s] refused: %v", key, v, err)
			}
		}
		fm[key] = []string{"spc-4"}
		if err := validateStrict(fm); err == nil {
			t.Fatalf("%s: [spc-4] admitted", key)
		}
		delete(fm, key)
	}
}

// Criterion 2 end to end: the quoted-text intent create, with the ledger's
// candidate set, links a draft that doubles an existing intent's title and
// press release.
func TestIntentCreateMatchesTheRecordThroughTheLedger(t *testing.T) {
	repo, ir := ledger(t)
	captureText(t, repo, ir, matchFiller1, nil)
	captureText(t, repo, ir, matchFiller2, nil)
	plantIntent(t, repo, "shipped", "itd-9", broadTitle, broadPress)
	cfg := match.Bundled()
	m := &intent.Matcher{Threshold: cfg.Threshold, Candidates: func() ([]match.Candidate, error) {
		return MatchCandidates(repo, cfg)
	}}
	c, err := intent.CreateFromTextMatched(repo, broadPress, intent.TextOptions{Title: broadTitle}, m)
	if err != nil {
		t.Fatal(err)
	}
	if c.Match == nil || len(c.Match.Matches) == 0 || c.Match.Matches[0].ID != "itd-9" ||
		c.Match.Matches[0].Relation != match.Duplicates {
		t.Fatalf("match = %+v, want itd-9 as duplicates", c.Match)
	}
	if content := readRecord(t, repo, c.Path); !strings.Contains(content, "\nduplicates: [itd-9]\n") {
		t.Fatalf("no link on the draft:\n%s", content)
	}
}
