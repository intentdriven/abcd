package scanner

import (
	"regexp"
	"strings"
	"testing"
)

// quantifiedBoundaryPattern is a custom pattern a repository's pii.json can
// merge: it compiles on its own, as the merge requires, but its leading `\b`
// is QUANTIFIED, so the boundary-free body probeParts strips it to (`+ACME…`)
// does not compile, and neither does the alternation junctionProbe joins it
// into (iss-191).
func quantifiedBoundaryPattern() Pattern {
	return Pattern{Name: "custom_acme", Kind: "custom:acme", Re: regexp.MustCompile(`\b+ACME[0-9]{8}`), Severity: SeverityHardFail}
}

// The combined junction probe could fall back to a regexp matching at EVERY
// offset when the alternation would not compile, which turned the candidate
// walk behind each match into a per-byte one with a whole-match validation at
// each byte: roughly 500x slower on one 200KB match. It stays a candidate
// generator that never under-produces, and no longer yields an offset no
// pattern can start a token at.
func TestJunctionProbeFallbackYieldsOnlyRealCandidates(t *testing.T) {
	patterns := append(DefaultPatterns(), quantifiedBoundaryPattern())
	js := junctionProbe(patterns)

	if loc := js.FindStringIndex(strings.Repeat(" ", 64)); loc != nil {
		t.Fatalf("the generator yielded %v in a run of spaces, where no pattern can start a token", loc)
	}
	// Never blinder: every pattern's own token is still a candidate, the
	// uncompilable one's included, at the offset it starts.
	for _, c := range []struct{ line, token string }{
		{"  ACME12345678", "ACME12345678"},
		// Abutting a word character, where the pattern's own boundary does not
		// hold: the junction case the generator exists for (iss-185).
		{"  xACME12345678", "ACME12345678"},
		{"  ghp_" + strings.Repeat("a", 36), "ghp_"},
	} {
		loc := js.FindStringIndex(c.line)
		if loc == nil || loc[0] != strings.Index(c.line, c.token) {
			t.Errorf("the generator over %q = %v, want a candidate at %d", c.line, loc, strings.Index(c.line, c.token))
		}
	}
}

// The cost the record measured: the junction search behind one long open-ended
// match. With the pattern that breaks the alternation in the set, the whole-match
// validations behind the match are a small multiple of those the default set
// costs, not one per byte of the backtrack window.
func TestJunctionSearchCostDoesNotCliffOnAnUncompilableAlternation(t *testing.T) {
	line := "ghp_" + strings.Repeat("a", 200000)
	work := func(patterns []Pattern) int {
		ghp := -1
		for i, p := range patterns {
			if p.Name == "github_pat" {
				ghp = i
			}
		}
		if ghp < 0 {
			t.Fatal("the default set has no github_pat pattern")
		}
		m := patMatch{patIdx: ghp, start: 0, end: len(line)}
		c := &countingMatcher{re: adjacencyProbe(patterns[ghp].Re)}
		budget := gallopBudget(line)
		stolenJunctions(c, newJunctionSet(patterns).behind(patterns[ghp]), line, m, &budget)
		return c.total()
	}
	base := work(DefaultPatterns())
	got := work(append(DefaultPatterns(), quantifiedBoundaryPattern()))
	t.Logf("whole-match validation bytes: default set %d, with the uncompilable alternation %d", base, got)
	if got > 4*base+4*len(line) {
		t.Fatalf("the junction search validated %d bytes behind a %d-byte match, against %d for the default set; "+
			"the fallback walks every byte of the window", got, len(line), base)
	}
}

// End to end: a custom token a greedy secret swallowed is recovered with the
// uncompilable alternation in the set, as the every-offset fallback recovered
// it — the replacement is faster and exactly as far from blind.
func TestJunctionFallbackStillRecoversASwallowedCustomToken(t *testing.T) {
	patterns := append(DefaultPatterns(), quantifiedBoundaryPattern())
	line := "token=ghp_" + strings.Repeat("a", 36) + "ACME12345678 end"
	var found bool
	for _, f := range ScanText(line, Identity{}, patterns, nil, "f") {
		found = found || f.Kind == "custom:acme"
	}
	if !found {
		t.Fatalf("the custom token the greedy PAT swallowed was not recovered: %+v", ScanText(line, Identity{}, patterns, nil, "f"))
	}
}
