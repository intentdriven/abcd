package cli

import (
	"strings"
	"testing"
)

// TestTheConsentNamesTheReviewPair: a drive's start names the pair it will
// run, each class with its model and effort, and the judge with its status,
// before it asks to start (itd-2611101419006280 decision 8).
func TestTheConsentNamesTheReviewPair(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.twoProfiles(t)
	withTerminal(t)
	out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive", "--profile", "beta")
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	text := string(out)
	line := ""
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, "  review:") {
			line = l
		}
	}
	for _, want := range []string{"ruthless-reviewer and security-reviewer", "research (anthropic/claude-fable-5, effort high)",
		"default (anthropic/claude-sonnet-5-5, effort high)", "judge top (anthropic/claude-opus-5-5, effort high), waiting"} {
		if !strings.Contains(line, want) {
			t.Fatalf("the consent's review line names %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "  review:") > strings.Index(text, "Start? [y/N]") {
		t.Fatalf("the pair is named before the question:\n%s", text)
	}
}
