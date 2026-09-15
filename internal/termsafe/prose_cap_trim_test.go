package termsafe

import "testing"

// TestCleanProseCapLeavesNoTrailingSpaceAfterADanglingEscape pins the cap
// loop's contract: dropping a dangling backslash a cut exposed must not leave
// the whitespace that preceded it, or the result is neither trimmed nor
// idempotent (a second cleaning would trim it).
func TestCleanProseCapLeavesNoTrailingSpaceAfterADanglingEscape(t *testing.T) {
	for _, in := range []string{"/ `é/", "-?<s\x1baé][\n \\\x1b-/]`"} {
		for capBytes := 1; capBytes <= len(in)+1; capBytes++ {
			got := CleanProse(in, capBytes)
			if got != CleanProse(got, capBytes) {
				t.Errorf("CleanProse(%q, %d) = %q is not idempotent", in, capBytes, got)
			}
			if n := len(got); n > 0 && (got[n-1] == ' ' || got[n-1] == '\t') {
				t.Errorf("CleanProse(%q, %d) = %q ends in whitespace", in, capBytes, got)
			}
		}
	}
}
