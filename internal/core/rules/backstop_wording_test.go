package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackstopSendsOnlyWhatThePromptMatches pins what the force_refresh_every_n
// backstop does (iss-2611060809419595): on the backstop prompt it clears the
// dedup ledgers, so a domain the prompt matches is sent again, and a domain the
// prompt does not match is not sent at all — there is no always-relevant domain
// the counter brings back by itself.
func TestBackstopSendsOnlyWhatThePromptMatches(t *testing.T) {
	rs := Defaults()
	st := SessionState{}
	for i := 0; i < 2; i++ {
		st = Inject(rs, "commit", st, 3, BracketUnknown).State
	}
	res := Inject(rs, "commit", st, 3, BracketUnknown)
	if !strings.Contains(res.Text, "## COMMITTING") {
		t.Fatalf("the backstop prompt did not send the matched domain again:\n%s", res.Text)
	}
	if strings.Contains(res.Text, "## ROADMAP") {
		t.Fatalf("the backstop prompt sent a domain the prompt does not match:\n%s", res.Text)
	}

	st = SessionState{Count: 2, Ledger: map[string]string{"COMMITTING": "x"}}
	if res := Inject(rs, "paint a landscape in oils", st, 3, BracketUnknown); res.Text != "" {
		t.Fatalf("a backstop prompt matching nothing sent rules:\n%s", res.Text)
	}
}

// TestBackstopIsNotDescribedAsAlwaysRelevant holds the comments and the
// configuration chapter to what the backstop does: they once said the counter
// only re-injects always-relevant domains, and Match injects nothing a prompt
// does not name or hit (iss-2611060809419595).
func TestBackstopIsNotDescribedAsAlwaysRelevant(t *testing.T) {
	files := []string{
		"inject.go",
		filepath.Join("..", "..", "..", ".abcd", "development", "brief", "05-internals", "03-configuration.md"),
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "always-relevant") {
			t.Errorf("%s still says the backstop re-injects always-relevant domains", f)
		}
	}
}
