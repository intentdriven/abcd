package repolint_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
	"github.com/intentdriven/abcd/internal/core/repolint"
)

// TestPrivacyReportsTheAugmentersFinding: a repository that opted into a
// scanner augmenter (gitleaks) has what it flags in a tracked file reported by
// privacy-hygiene, at the line, without the value (iss-2608291814575788).
func TestPrivacyReportsTheAugmentersFinding(t *testing.T) {
	augmenttest.Install(t, augmenttest.Fake())
	res := newFixtureRepo(t).conforming().
		file("docs/notes.md", "first line\nthe config holds "+augmenttest.Value+" in prose\n").
		commit().run()
	for _, f := range res.Findings {
		if f.RuleID != "privacy-hygiene" || f.File != "docs/notes.md" {
			continue
		}
		if f.Line != 2 || f.Severity != repolint.SeverityError {
			t.Errorf("finding misplaced or advisory: %+v", f)
		}
		if strings.Contains(f.Message+f.Fix, augmenttest.Value) {
			t.Errorf("the finding echoes the value: %+v", f)
		}
		return
	}
	t.Fatalf("privacy-hygiene did not report the augmented finding: %+v", res.Findings)
}

// TestPrivacyFailsOnTheAugmenterGap: a configured augmenter that is not
// installed is an error finding, as launch refuses on it: the lint is a gate,
// and "conforms" over a scan the repository asked to be deeper is the
// didn't-scan-reported-clean shape.
func TestPrivacyFailsOnTheAugmenterGap(t *testing.T) {
	augmenttest.Install(t, augmenttest.NotFound())
	res := newFixtureRepo(t).conforming().commit().run()
	for _, f := range res.Findings {
		if f.RuleID == "privacy-hygiene" && strings.Contains(f.Message, "fake augmenter not on PATH") {
			if f.Severity != repolint.SeverityError {
				t.Errorf("the gap is advisory: %+v", f)
			}
			return
		}
	}
	t.Fatalf("privacy-hygiene did not report the augmenter gap: %+v", res.Findings)
}
