package history

import (
	"context"
	"strings"
	"testing"
)

// TestRedactorRedactsEachTextAsRedactDoes: a Redactor built once gives, for
// each text, what Redact gives for it: a secret and the caller's home path
// masked, plain text kept, and a second text redacted by the same scanner.
func TestRedactorRedactsEachTextAsRedactDoes(t *testing.T) {
	repoRoot, home := setupStore(t)
	redact, err := Redactor(context.Background(), repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"token ghp_" + strings.Repeat("a", 40) + "\n",
		"see " + home + "/plan.md\n",
		"nothing to mask\n",
	} {
		got, err := redact(text, "a text")
		if err != nil {
			t.Fatal(err)
		}
		want, err := Redact(repoRoot, text, "a text")
		if err != nil {
			t.Fatal(err)
		}
		if got.Text != want.Text {
			t.Errorf("Redactor gave %q; Redact gives %q", got.Text, want.Text)
		}
		if strings.Contains(got.Text, home) || strings.Contains(got.Text, "ghp_"+strings.Repeat("a", 40)) {
			t.Errorf("the text is not masked: %q", got.Text)
		}
	}
}
