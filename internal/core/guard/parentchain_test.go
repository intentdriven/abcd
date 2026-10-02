package guard

import "testing"

// TestAParentChainReadsAsOneParent — iss-2610021542183618. A recursive delete
// of `..` warns on rm-rf-working-directory, but a chain of parents (`../..`,
// `../../..`) was an allow: the entry's arg_values name `..`, `../` and `../*`
// and no deeper chain, so `rm -rf ../..` passed while it deletes everything
// `rm -rf ..` deletes and more. A target whose leading run of `..` segments
// is two or more is also read as that run written once, beside the readings
// already compared, so a chain reaches the verdict its one-parent form does.
func TestAParentChainReadsAsOneParent(t *testing.T) {
	for in, want := range map[string]string{
		"../..":       "..",
		"../../":      "../",
		"../../..":    "..",
		"../../../":   "../",
		"../../*":     "../*",
		"../../../*":  "../*",
		"../../build": "../build",
		"..":          "",
		"../":         "",
		"../*":        "",
		"../..x":      "",
		"..x/..":      "",
		".../..":      "",
		"./../..":     "",
		"":            "",
	} {
		if got := parentRun(in); got != want {
			t.Errorf("parentRun(%q) = %q, want %q", in, got, want)
		}
	}

	const cwd = "rm-rf-working-directory"
	for _, cmd := range []string{
		// The chain itself, at two and three parents and deeper.
		`rm -rf ../..`, `rm -rf ../../..`, `rm -rf ../../../..`,
		// A trailing separator, a trailing `.`, and redundant separators.
		`rm -rf ../../`, `rm -rf ../../.`, `rm -rf ../../../`, `rm -rf ../../../.`,
		`rm -rf ../..//`, `rm -rf ../.././`, `rm -rf .././..`,
		// Quoted.
		`rm -rf "../.."`, `rm -rf '../..'`, `rm -rf "../../."`, `rm -rf "../../"`, `rm -rf '../../..'`,
		// Every recursive flag spelling the entry names.
		`rm -r ../..`, `rm -R ../..`, `rm --recursive ../..`, `rm -fr ../..`,
		`rm -r -f ../..`, `rm --recursive --force ../..`, `rm -rf -- ../..`,
		// Through a wrapper and a shell.
		`sudo rm -rf ../..`, `env rm -rf ../..`, `sh -c 'rm -rf ../..'`,
		// A start that folds to a chain.
		`rm -rf ./../..`, `rm -rf $PWD/../..`, `rm -rf ${PWD}/../..`, `rm -rf "$PWD"/../..`,
		`rm -rf x/../../..`, `rm -rf ../../x/../..`,
		// The chain's own entries, as `../*` is.
		`rm -rf ../../*`, `rm -rf ../../../*`, `rm -rf ../../**`, `rm -rf $PWD/../../*`,
		// Beside another operand.
		`rm -rf ./build ../..`,
	} {
		if d := verdictOf(t, cmd); d.Verdict != VerdictWarn || d.EntryID != cwd {
			t.Errorf("Check(%q) = %q via %q, want warn via %q", cmd, d.Verdict, d.EntryID, cwd)
		}
	}

	// A named directory under the chain is not the chain, a segment that only
	// begins with `..` is a name, and without the recursive flag rm deletes no
	// directory: each is read as it was before.
	for _, cmd := range []string{
		`rm -rf ../../build`, `rm -rf ../..x`, `rm -rf ..x/..`, `rm -rf .../..`,
		`rm -rf ../../x/.`, `rm -f ../..`, `rm ../..`, `ls -la ../..`,
	} {
		if d := verdictOf(t, cmd); d.EntryID == cwd {
			t.Errorf("Check(%q) = %q via %q, want no working-directory verdict", cmd, d.Verdict, d.EntryID)
		}
	}
}

// TestTheTrailingSlashGlobOfEachNamedDirectoryWarns — iss-2610021547096336.
// `*/` and `./*/` warn on rm-rf-working-directory, since the glob names every
// directory of the working directory, but the same glob of the parent and of
// `$PWD` was an allow: the entry named its `/*` form and not its `/*/` form.
// The parent's glob names the working directory itself among the rest.
func TestTheTrailingSlashGlobOfEachNamedDirectoryWarns(t *testing.T) {
	const cwd = "rm-rf-working-directory"
	for _, cmd := range []string{
		`rm -rf ../*/`, `rm -r ../*/`, `rm -rf "$PWD"/*/`, `rm -rf $PWD/*/`, `rm -rf ${PWD}/*/`,
		`rm -rf ../../*/`, `rm -rf ./../*/`, `rm -rf ../**/`,
	} {
		if d := verdictOf(t, cmd); d.Verdict != VerdictWarn || d.EntryID != cwd {
			t.Errorf("Check(%q) = %q via %q, want warn via %q", cmd, d.Verdict, d.EntryID, cwd)
		}
	}
	for _, cmd := range []string{`rm -rf ../build/*/`, `rm -rf $PWD/build/*/`, `rm -f ../*/`} {
		if d := verdictOf(t, cmd); d.EntryID == cwd {
			t.Errorf("Check(%q) = %q via %q, want no working-directory verdict", cmd, d.Verdict, d.EntryID)
		}
	}
}
