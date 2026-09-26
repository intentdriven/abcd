package guard

import (
	"strings"
	"testing"
)

// TestFixedOutputAfterAnIFSAssignmentIsRefused — review7-guard finding 2
// (iss-2609260115387303). An unquoted fixed output is read as the words the
// default IFS splits it into. A command line that assigns IFS in a command of
// its own changes that split for every expansion after it, so the words bash
// runs are not the ones the guard read: `IFS=x; $(cat <<'F' … F)` with a body
// spelling a hazard joined by x reached ALLOW as one word. Modelling IFS costs
// more than refusing it, so an unquoted fixed output on a command line where
// any other command names IFS is refused under its own id. A prefix assignment
// (`IFS=x $(…)`) does not reach the expansion of its own command in bash, and
// still reads as the default split. Each shape was run under bash 3.2 and 5.3
// with a neutral word.
func TestFixedOutputAfterAnIFSAssignmentIsRefused(t *testing.T) {
	const joined = "gitxpushx--forcexoriginxmain"
	doc := func(body string) string { return "$(cat <<'F'\n" + body + "\nF\n)" }
	runVerdictCases(t, []verdictCase{
		{"IFS=x; " + doc(joined), VerdictBlock, ifsSplitEntryID},
		{"IFS=x\n" + doc(joined), VerdictBlock, ifsSplitEntryID},
		{"export IFS=x; " + doc(joined), VerdictBlock, ifsSplitEntryID},
		{"IFS=x; `cat <<'F'\n" + joined + "\nF\n`", VerdictBlock, ifsSplitEntryID},
		{"IFS=x; echo " + doc("a b"), VerdictBlock, ifsSplitEntryID},
		{"sh -c \"IFS=x; \\" + doc(joined) + "\"", VerdictBlock, ifsSplitEntryID},
		{"IFS=x; eval \"$(cat <<'G'\n" + doc(joined) + "\nG\n)\"", VerdictBlock, ifsSplitEntryID},

		// A prefix assignment does not reach its own command's expansion.
		{"IFS=x " + doc(joined), VerdictAllow, ""},
		// A quoted output is not split, and a line with no fixed output is
		// not read differently for naming IFS.
		{"IFS=x; echo \"" + doc(joined) + "\"", VerdictAllow, ""},
		{"IFS=, read -r a b <<< \"1,2\"; echo \"$a\"", VerdictAllow, ""},
	})
	if _, isEntry := Defaults().Entries[ifsSplitEntryID]; isEntry {
		t.Errorf("the reserved id %q must never be a registry entry", ifsSplitEntryID)
	}
	if !containsString(reservedEntryIDs, ifsSplitEntryID) {
		t.Errorf("reservedEntryIDs = %v, want %q listed", reservedEntryIDs, ifsSplitEntryID)
	}
}

// TestIFSSplitCheckStaysLinear pins the cost of the IFS check: every word of
// the command line is scanned at most once, and only when an unquoted fixed
// output is on it.
func TestIFSSplitCheckStaysLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"a fixed output among many commands": func(n int) string {
			return "echo $(cat <<'F'\na b\nF\n)\n" + strings.Repeat("echo abc def ghi jkl\n", n)
		},
		"many fixed outputs, IFS named last": func(n int) string {
			return strings.Repeat("echo $(cat <<'F'\na b\nF\n)\n", n) + "IFS=x"
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<9, "each word is scanned for IFS at most once")
		})
	}
}
