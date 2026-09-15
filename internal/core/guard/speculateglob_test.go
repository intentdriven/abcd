package guard

import "testing"

// TestGlobbedLauncherKeepsItsTier2Warn pins that reading a payload behind a
// GUESSED interpreter name is taken in addition to Tier 2, never instead of it.
// `* -c <words>` may expand to `sh`, and it may equally expand to a program
// nothing here names; letting the guess satisfy speculate's carriesPayload gate
// dropped the unrecognised-launcher warn adr-42 decision 2 says is never
// dropped. It dropped it into silence, too: `sh -c` reads only the first operand
// as the payload, so the destructive words after it became the payload's own
// positional parameters and nothing looked at them again.
func TestGlobbedLauncherKeepsItsTier2Warn(t *testing.T) {
	for _, line := range []string{
		"* -c gh api -X DELETE repos/owner/repo",
		"s? -c gh api -X DELETE repos/owner/repo",
		"?h -c gh api -X DELETE repos/owner/repo",
		"[sb]* -c gh api -X DELETE repos/owner/repo",
		"ba?h -c gh api -X DELETE repos/owner/repo",
	} {
		d, err := Defaults().Check(line)
		if err != nil {
			t.Fatalf("Check(%q): %v", line, err)
		}
		if d.Verdict == VerdictAllow {
			t.Errorf("Check(%q) = allow — a guessed launcher name must not drop the unrecognised-launcher warn", line)
		}
	}
	// A literal interpreter name is not a guess: its payload is read, and
	// speculating on the parent as well would report the same hazard twice.
	const literal = "sh -c 'gh api -X DELETE repos/owner/repo'"
	d, err := Defaults().Check(literal)
	if err != nil {
		t.Fatalf("Check(%q): %v", literal, err)
	}
	if d.Verdict != VerdictBlock || d.EntryID != "gh-api-repo-delete" {
		t.Fatalf("Check(%q) = %q via %q, want block via gh-api-repo-delete", literal, d.Verdict, d.EntryID)
	}
}
