package guard

import (
	"errors"
	"testing"
)

// TestRecursiveDeleteOfRootOrHomeBlocks — iss-2609282105242542. The chapter's
// headline promises the guard catches "an `rm -rf` with an unlucky glob", and
// the registry's one rm entry fired only behind a `cd` chain, so the
// catastrophic forms were an allow: `rm -rf /`, `rm -rf ~`, `rm -rf $HOME`. A
// recursive delete whose operand is the filesystem root or the home directory
// is a block wherever it stands, in every flag spelling and behind every
// launcher the registry already steps over.
func TestRecursiveDeleteOfRootOrHomeBlocks(t *testing.T) {
	const id = "rm-rf-root-or-home"
	runVerdictCases(t, []verdictCase{
		{`rm -rf /`, VerdictBlock, id},
		{`rm -rf /*`, VerdictBlock, id},
		{`rm -fr /`, VerdictBlock, id},
		{`rm -Rf /`, VerdictBlock, id},
		{`rm -r -f /`, VerdictBlock, id},
		{`rm --recursive --force /`, VerdictBlock, id},
		{`rm -r /`, VerdictBlock, id},
		{`rm -rf -- /`, VerdictBlock, id},
		{`rm -rf --no-preserve-root /`, VerdictBlock, id},
		{`rm -rf ~`, VerdictBlock, id},
		{`rm -rf ~/`, VerdictBlock, id},
		{`rm -rf ~/*`, VerdictBlock, id},
		{`rm -rf $HOME`, VerdictBlock, id},
		{`rm -rf "$HOME"`, VerdictBlock, id},
		{`rm -rf "$HOME"/*`, VerdictBlock, id},
		{`rm -rf ${HOME}`, VerdictBlock, id},
		{`rm -rf "${HOME}/"`, VerdictBlock, id},
		{`rm -rf ./build /`, VerdictBlock, id},
		{`sudo rm -rf /`, VerdictBlock, id},
		{`sudo -u root rm -rf /*`, VerdictBlock, id},
		{`env rm -rf ~`, VerdictBlock, id},
		{`command rm -rf $HOME`, VerdictBlock, id},
		{`/bin/rm -rf /`, VerdictBlock, id},
		{`sh -c 'rm -rf ~'`, VerdictBlock, id},

		// Ordinary deletes under the root or the home directory are not it.
		{`rm -rf /tmp/build`, VerdictAllow, ""},
		{`rm -rf ~/scratch/build`, VerdictAllow, ""},
		{`rm -rf "$HOME/.cache/abcd-test"`, VerdictAllow, ""},
		{`rm ~/notes.txt`, VerdictAllow, ""},
		{`rm -f /`, VerdictAllow, ""},
		{`ls -la /`, VerdictAllow, ""},
		{`printf '%s\n' "rm -rf /"`, VerdictAllow, ""},
	})
}

// TestRecursiveDeleteOfTheHomesDotfilesBlocks — iss-2609282105242542, the
// review's medium finding. `rm -rf ~/.*` names every dotfile and dot-directory
// in the home — the keys, the shell and tool settings, abcd's own store — and it
// is a shape people type, not an obfuscation: the working directory's `.*` was
// already a warn while the home's was an allow. It blocks with the root and the
// home themselves, in every spelling of the home the entry knows. A delete that
// names one dot-directory under the home is ordinary work and stays an allow.
func TestRecursiveDeleteOfTheHomesDotfilesBlocks(t *testing.T) {
	const id = "rm-rf-root-or-home"
	runVerdictCases(t, []verdictCase{
		{`rm -rf ~/.*`, VerdictBlock, id},
		{`rm -r ~/.*`, VerdictBlock, id},
		{`rm -rf $HOME/.*`, VerdictBlock, id},
		{`rm -rf "$HOME"/.*`, VerdictBlock, id},
		{`rm -rf ${HOME}/.*`, VerdictBlock, id},
		{`sudo rm -rf ~/.*`, VerdictBlock, id},

		{`rm -rf ~/.cache/x`, VerdictAllow, ""},
		{`rm -rf "$HOME/.cache/x"`, VerdictAllow, ""},
		{`rm -f ~/.*`, VerdictAllow, ""},
	})
}

// TestRecursiveDeleteOfTheWorkingDirectoryWarns — iss-2609282105242542. A
// recursive delete of everything in the directory the shell is in (`rm -rf *`,
// `rm -rf .`) deletes the repository when that directory is the repository,
// which is where an agent's shell usually is. It is graded like `git clean`, a
// warn: the delete of a build directory's contents is ordinary work too, so the
// warning names the risk and lets it run. A named subdirectory is not it.
func TestRecursiveDeleteOfTheWorkingDirectoryWarns(t *testing.T) {
	const id = "rm-rf-working-directory"
	runVerdictCases(t, []verdictCase{
		{`rm -rf *`, VerdictWarn, id},
		{`rm -fr *`, VerdictWarn, id},
		{`rm -r -f *`, VerdictWarn, id},
		{`rm --recursive --force *`, VerdictWarn, id},
		{`rm -rf .`, VerdictWarn, id},
		{`rm -rf ./`, VerdictWarn, id},
		{`rm -rf ./*`, VerdictWarn, id},
		{`rm -rf ..`, VerdictWarn, id},
		{`rm -rf ../*`, VerdictWarn, id},
		{`rm -rf .*`, VerdictWarn, id},
		{`sudo rm -rf *`, VerdictWarn, id},
		{`rm $(true) -rf *`, VerdictWarn, id},
		// The working directory by name, and the globs that still reach
		// everything in it: `rm -rf .` is refused by rm itself, while
		// `rm -rf "$PWD"` is the spelling that deletes (the review's low finding).
		{`rm -rf $PWD`, VerdictWarn, id},
		{`rm -rf "$PWD"`, VerdictWarn, id},
		{`rm -rf ${PWD}`, VerdictWarn, id},
		{`rm -rf $PWD/*`, VerdictWarn, id},
		{`rm -rf "$PWD"/*`, VerdictWarn, id},
		{`rm -rf ${PWD}/*`, VerdictWarn, id},
		{`rm -rf */`, VerdictWarn, id},
		{`rm -rf ./*/`, VerdictWarn, id},
		{`rm -rf ./.*`, VerdictWarn, id},

		{`rm -rf ./build`, VerdictAllow, ""},
		{`rm -rf build/*`, VerdictAllow, ""},
		{`rm -rf node_modules`, VerdictAllow, ""},
		{`rm -f *.o`, VerdictAllow, ""},
		{`rm *`, VerdictAllow, ""},
		{`rm -rf .git`, VerdictAllow, ""},
		{`rm -rf .venv`, VerdictAllow, ""},
		{`rm -rf "$PWD/build"`, VerdictAllow, ""},
		{`rm -rf $PWD/build/*`, VerdictAllow, ""},

		// Behind a cd chain the blocker still decides, and names both.
		{`cd scratch && rm -rf *`, VerdictBlock, "rm-rf-after-cd-chain"},
	})
}

// TestArgValuesIsValidated holds arg_values to the rule every other pattern
// field keeps: an empty value would match an empty operand nobody meant, and a
// dash-led one describes an operand nothing can be, so both are refused at load
// rather than shipped as an entry that looks armed.
func TestArgValuesIsValidated(t *testing.T) {
	for _, bad := range [][]string{{""}, {"  "}, {"-rf"}} {
		r := Defaults()
		r.Entries["x-arg-values"] = Entry{
			ID: "x-arg-values", Tier: TierWarn, Why: "x", Successor: "y",
			Pattern: Pattern{Command: "rm", ArgValues: bad},
		}
		if err := Validate(r); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("Validate accepted arg_values %q; want ErrInvalidEntry, got %v", bad, err)
		}
	}
	// A repo override replaces the list, and a clone shares no slice with it.
	base := Defaults()
	over := Registry{SchemaVersion: SchemaVersion, Entries: map[string]Entry{
		"rm-rf-root-or-home": {Pattern: Pattern{ArgValues: []string{"/srv"}}},
	}}
	merged := Merge(base, over)
	got := merged.Entries["rm-rf-root-or-home"].Pattern.ArgValues
	if len(got) != 1 || got[0] != "/srv" {
		t.Fatalf("merged arg_values = %q, want [/srv]", got)
	}
	clone := clonePattern(merged.Entries["rm-rf-root-or-home"].Pattern)
	clone.ArgValues[0] = "/changed"
	if merged.Entries["rm-rf-root-or-home"].Pattern.ArgValues[0] != "/srv" {
		t.Fatal("clonePattern shares its arg_values slice with the pattern it copied")
	}
}
