package guard

import "testing"

// TestTrailingDotAfterAParentFolds — iss-2609302306019245. foldParents folds a
// trailing `.` after a `..`, as its comment says, but cleanSeparators returned
// a path early unless it held `//`, `/./` or `/..`, so `../.` never reached the
// fold and `rm -rf ../.` was allowed while `./../.` warned. The trailing `.`
// names the directory the `..` reaches, so each form reads as that directory
// does: `../.` and `./../.` as `..`, and `../../.` as `../..`.
func TestTrailingDotAfterAParentFolds(t *testing.T) {
	for in, want := range map[string]string{
		"../.":    "..",
		"./../.":  "..",
		"../../.": "../..",
		"../":     "../",
		"./.":     "./.",
		"x/.":     "x/.",
	} {
		if got := cleanSeparators(in); got != want {
			t.Errorf("cleanSeparators(%q) = %q, want %q", in, got, want)
		}
	}

	const cwd = "rm-rf-working-directory"
	for _, cmd := range []string{
		`rm -rf ../.`, `rm -r ../.`, `rm --recursive --force ../.`, `rm -rf "../."`,
		`rm -rf ./../.`, `rm -rf $PWD/../.`, `rm -rf x/../../.`,
	} {
		if d := verdictOf(t, cmd); d.Verdict != VerdictWarn || d.EntryID != cwd {
			t.Errorf("Check(%q) = %q via %q, want warn via %q", cmd, d.Verdict, d.EntryID, cwd)
		}
	}
	// `../../.` is the directory `../..` names, so it reads as `../..` does,
	// whatever the registry says of that.
	plain, dotted := verdictOf(t, `rm -rf ../..`), verdictOf(t, `rm -rf ../../.`)
	if plain.Verdict != dotted.Verdict || plain.EntryID != dotted.EntryID {
		t.Errorf("Check(`rm -rf ../../.`) = %q via %q, want what `rm -rf ../..` gets: %q via %q",
			dotted.Verdict, dotted.EntryID, plain.Verdict, plain.EntryID)
	}
	// A trailing `.` that follows no `..` is not a parent, and is compared as
	// written.
	if d := verdictOf(t, `rm -rf x/.`); d.EntryID == cwd {
		t.Errorf("Check(`rm -rf x/.`) = %q via %q, want no working-directory verdict", d.Verdict, d.EntryID)
	}
}
