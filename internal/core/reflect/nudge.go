package reflect

import "fmt"

// Nudge is the one line a release cut prints when it is written (criterion 8,
// itd-24 decision 1): a retrospective for the release is owed, and the command
// that writes it. It is said once, at the cut, and gates nothing; the
// retrospective's absence is never announced again.
func Nudge(tag string) string {
	return fmt.Sprintf("A retrospective for %s is owed: run /abcd:reflect %s when you are ready.", tag, tag)
}
