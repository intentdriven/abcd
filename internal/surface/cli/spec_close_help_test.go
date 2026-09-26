package cli

import (
	"strings"
	"testing"
)

// spec_close_help_test.go — `abcd spec close --help` names everything the close
// writes (iss-2609202015349672): beyond the spec and intent moves, the close
// that ships an intent mints an OWED fidelity-review receipt, parks an
// `abcd-review: OWED` marker in the intent's Audit Notes, and writes the review
// request under .abcd/.work.local/reviews/. The help is where the verb is met,
// so the caller learns there that the audit is owed and where its input is.
func TestSpecCloseHelpNamesTheOwedReview(t *testing.T) {
	out := string(runCLI(t, "spec", "close", "--help"))
	for _, want := range []string{
		"OWED",
		"abcd-review: OWED",
		".abcd/.work.local/reviews/",
		"abcd intent audit",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("`abcd spec close --help` does not name %q:\n%s", want, out)
		}
	}
}
