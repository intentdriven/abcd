package intent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/record/match"
)

const (
	existingTitle = "The quiet board says who holds the next move"
	existingPress = "The bare board names the record whose next move is waiting on a person, " +
		"says which person, and prints the command that answers it, without opening a file."
	newDraftText = "The bare board names the record whose next move waits on a person, " +
		"says which person that is, and prints the command that answers it, without opening any file."
)

func fixedCandidates(cands ...match.Candidate) func() ([]match.Candidate, error) {
	return func() ([]match.Candidate, error) {
		return append([]match.Candidate{
			{ID: "itd-21", Text: "The site builder renders a stale anchor for a heading renamed since the last build."},
			{ID: "iss-22", Text: "The history store drops a transcript that exceeds its byte budget without saying so."},
		}, cands...), nil
	}
}

// Criterion 2: a draft whose text overlaps an existing intent's title and
// press release is written with the link, and the result carries the match.
func TestCreateFromTextMatchedLinksADouble(t *testing.T) {
	root := t.TempDir()
	m := &Matcher{Threshold: match.DefaultThreshold, Candidates: fixedCandidates(
		match.Candidate{ID: "itd-20", Text: existingTitle + "\n" + existingPress})}
	c, err := CreateFromTextMatched(root, newDraftText, TextOptions{}, m)
	if err != nil {
		t.Fatalf("create refused: %v", err)
	}
	if c.Match == nil || len(c.Match.Matches) != 1 || c.Match.Matches[0].ID != "itd-20" ||
		c.Match.Matches[0].Relation != match.Duplicates {
		t.Fatalf("match = %+v, want itd-20 as duplicates", c.Match)
	}
	data, err := os.ReadFile(filepath.Join(root, c.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\nduplicates: [itd-20]\n") {
		t.Fatalf("the draft carries no link:\n%s", data)
	}
	// The linked draft is as lint-valid as an unlinked one.
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
	for _, f := range findings {
		t.Fatalf("linked draft: %s:%d %s", f.File, f.Line, f.Message)
	}
}

// Criteria 3 and 4: a candidate set that cannot be read never refuses the
// create, and below the threshold nothing is written while the near misses
// are listed.
func TestCreateFromTextMatchedNeverRefuses(t *testing.T) {
	root := t.TempDir()
	broken := &Matcher{Threshold: match.DefaultThreshold, Candidates: func() ([]match.Candidate, error) {
		return nil, errors.New("the ledger is unreadable")
	}}
	c, err := CreateFromTextMatched(root, newDraftText, TextOptions{}, broken)
	if err != nil {
		t.Fatalf("an unreadable candidate set refused the create: %v", err)
	}
	if c.Match == nil || !strings.Contains(c.Match.Skipped, "unreadable") {
		t.Fatalf("outcome = %+v, want the reason nothing was compared", c.Match)
	}

	strict := &Matcher{Threshold: 0.99, Candidates: fixedCandidates(
		match.Candidate{ID: "itd-20", Text: existingTitle + "\n" + existingPress})}
	c, err = CreateFromTextMatched(root, newDraftText+" Also a second time.", TextOptions{}, strict)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Match.Matches) != 0 || len(c.Match.NearMisses) == 0 || c.Match.NearMisses[0].ID != "itd-20" {
		t.Fatalf("outcome = %+v, want itd-20 as a near miss only", c.Match)
	}
	data, _ := os.ReadFile(filepath.Join(root, c.Path))
	if strings.Contains(string(data), "duplicates:") || strings.Contains(string(data), "refines:") {
		t.Fatalf("a link was written below the threshold:\n%s", data)
	}
}

// Without a matcher the create is CreateFromText exactly.
func TestCreateFromTextMatchedWithoutAMatcher(t *testing.T) {
	c, err := CreateFromTextMatched(t.TempDir(), newDraftText, TextOptions{}, nil)
	if err != nil || c.Match != nil {
		t.Fatalf("err %v, match %+v: want a plain create", err, c.Match)
	}
}

// MatchTexts offers each intent's H1 and press release, and leaves out a press
// release that is still a create's seed note.
func TestMatchTextsReadsTitleAndPressRelease(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, IntentsRelDir, BucketPlanned)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("itd-5-a.md", "---\nid: itd-5\nslug: a\n---\n\n# "+existingTitle+"\n\n## Press Release\n\n> "+
		existingPress+"\n\n## Why This Matters\n\nNot compared.\n")
	write("itd-6-b.md", "---\nid: itd-6\nslug: b\n---\n\n# Promoted\n\n## Press Release\n\n> _"+
		promotionSeedOpening+"iss-9. "+seedNoteTail+"_\n")
	texts, err := MatchTexts(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]MatchText{}
	for _, mt := range texts {
		got[mt.ID] = mt
	}
	if got["itd-5"].Title != existingTitle || !strings.Contains(got["itd-5"].PressRelease, "prints the command") ||
		strings.Contains(got["itd-5"].PressRelease, "Not compared") {
		t.Fatalf("itd-5 = %+v", got["itd-5"])
	}
	if got["itd-6"].Title != "Promoted" || got["itd-6"].PressRelease != "" {
		t.Fatalf("itd-6 = %+v, want its seed note left out", got["itd-6"])
	}
}
