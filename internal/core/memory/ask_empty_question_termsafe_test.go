package memory

import (
	"strings"
	"testing"
)

// TestAskEmptyStoreSanitisesQuestion is the GHSA-4fmm-95pf-32c6 detector. On
// an empty store Ask took the no-matches branch and handed the raw argv
// question to RenderNoMatches, while RenderCitedMatches on the matched branch
// sanitised it; both branches also put the raw question into AskResult.Question,
// the --json field. An ESC, a C1 control, a bidi override or a zero-width rune
// in the question therefore reached the terminal raw from exactly the branch a
// first-time user hits. The question is sanitised once in Ask, and the one
// value feeds both renders and the JSON field. Attack runes are built
// numerically so this file carries none of them.
func TestAskEmptyStoreSanitisesQuestion(t *testing.T) {
	attacks := map[string]rune{
		"ESC":          0x1b,
		"C1 CSI":       0x9b,
		"RLO override": 0x202e,
		"zero-width":   0x200b,
	}
	question := "rotate " + string(rune(0x1b)) + "[31mRED" + string(rune(0x1b)) + "[0m " + string(rune(0x202e)) + " tokens"
	for _, r := range attacks {
		question += string(r)
	}

	res, err := Ask(AskRequest{RepoRoot: t.TempDir(), Question: question})
	if err != nil {
		t.Fatalf("ask on an empty store: %v", err)
	}
	if len(res.Matches) != 0 {
		t.Fatalf("expected the empty-store branch, got %d matches", len(res.Matches))
	}
	for name, r := range attacks {
		if strings.ContainsRune(res.Answer, r) {
			t.Errorf("GHSA-4fmm: RenderNoMatches answer carries a raw %s (U+%04X) from the question:\n%q", name, r, res.Answer)
		}
		if strings.ContainsRune(res.Question, r) {
			t.Errorf("GHSA-4fmm: AskResult.Question (the --json field) carries a raw %s (U+%04X): %q", name, r, res.Question)
		}
	}
	if !strings.Contains(res.Answer, "rotate") || !strings.Contains(res.Question, "rotate") {
		t.Fatalf("sanitising must not drop the legitimate question text: answer=%q question=%q", res.Answer, res.Question)
	}
}

// TestRenderNoMatchesSanitisesDirectly pins the exported render on its own, so
// a caller that reaches it without going through Ask is covered the way
// RenderCitedMatches already is.
func TestRenderNoMatchesSanitisesDirectly(t *testing.T) {
	out := RenderNoMatches("q" + string(rune(0x1b)) + string(rune(0x202e)))
	if strings.ContainsRune(out, 0x1b) || strings.ContainsRune(out, 0x202e) {
		t.Fatalf("RenderNoMatches echoes attack runes raw: %q", out)
	}
}
