package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// freeTextParents are the parents whose positional is prose by design: `abcd
// capture "…"` files an issue and `abcd intent "…"` files a draft. They are
// cobra.ArbitraryArgs on purpose and guard the door in RunE instead — a
// did-you-mean for a near-miss sub-verb, and a flat refusal of any lone bare
// word (iss-2608221328552172).
//
// They are NO LONGER exempt from the sweep below. The exemption existed because
// a lone unknown token was swallowed as create text and minted a durable record
// at exit 0 — the very defect that issue reports — so the list that was meant to
// describe a design choice was in fact holding the sweep off the two verbs where
// an unwanted write costs the most. The list survives only to pin the other half:
// prose still files.
var freeTextParents = []string{"capture", "intent"}

// TestEveryParentRefusesAnUnknownSubverb is the iss-266 sweep, stated
// structurally. Two independent cobra facts have to hold for a parent to refuse a
// mistyped sub-verb, and missing EITHER one silently exits 0:
//
//   - The parent must be Runnable. A parent with no RunE returns flag.ErrHelp
//     before ValidateArgs ever runs, so cobra prints help and exits 0 without
//     looking at the args at all.
//   - The parent must declare an Args validator. With Args nil, cobra's
//     legacyArgs falls through to ArbitraryArgs for a non-root parent, so a
//     Runnable parent still accepts the stray token and exits 0.
//
// The property is asserted over the whole live tree rather than a hand-kept list,
// so a parent added later cannot reintroduce the hole by missing either half.
func TestEveryParentRefusesAnUnknownSubverb(t *testing.T) {
	var notRunnable, noArgsValidator []string
	for _, p := range parents(t, NewRootCommand()) {
		name := strings.Join(p.path, " ")
		if p.cmd.RunE == nil && p.cmd.Run == nil {
			notRunnable = append(notRunnable, name)
		}
		if p.cmd.Args == nil {
			noArgsValidator = append(noArgsValidator, name)
		}
	}
	if len(notRunnable) > 0 {
		t.Errorf("parents with no RunE print help and exit 0 on an unknown sub-verb (iss-266): %v\n"+
			"give each a `RunE: helpRunE`", notRunnable)
	}
	if len(noArgsValidator) > 0 {
		t.Errorf("parents with no Args validator accept an unknown sub-verb and exit 0 (iss-266): %v\n"+
			"give each a `cobra.NoArgs` (or an explicit validator that refuses a stray token)", noArgsValidator)
	}
}

// TestParentsRefuseAnUnknownSubverbAtExitTwo is the behavioural half. The
// structural check above guarantees the Args validator RUNS; this proves it
// actually refuses, and refuses with the exit code the CHANGELOG promises — a
// parent that regressed to exit 1 would be a different, worse contract. The list
// is DERIVED from the live tree, so a parent added later is exercised without
// anyone remembering to add it. Runs in a scratch cwd so a regression that does
// reach a write path cannot dirty the repository under test.
//
// `capture` and `intent` are in the sweep: a lone bare token is refused there too
// (iss-2608221328552172), so this is the tree-wide detector for that defect —
// nothing verb-specific has to be remembered for it to stay closed.
//
// hookPlaneParents are excluded and covered by
// TestHookPlaneParentsFailOpenOnUnknownSubverb instead, under a STRONGER
// constraint rather than a weaker one: they must refuse (non-zero, loud) AND must
// not use exit 2, because on the hook plane 2 is the host's blocking status
// rather than a usage code (iss-267). Both sets refuse; they disagree only on
// which non-zero code says so.
func TestParentsRefuseAnUnknownSubverbAtExitTwo(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, p := range parents(t, NewRootCommand()) {
		name := strings.Join(p.path, " ")
		if slices.Contains(hookPlaneParents, name) {
			continue
		}
		t.Run(strings.Join(p.path, "_"), func(t *testing.T) {
			args := append(append([]string{}, p.path...), "zzz-not-a-subverb")
			var out, errOut strings.Builder
			if code := Run(args, &out, &errOut); code != 2 {
				t.Fatalf("abcd %s exited %d, want 2 (usage refusal).\nstdout:\n%s\nstderr:\n%s",
					strings.Join(args, " "), code, out.String(), errOut.String())
			}
		})
	}
}

// TestFreeTextParentsStillFileProse is the counterweight to putting `capture` and
// `intent` into the sweep above. The sweep only proves they refuse; a guard that
// refused everything would pass it and break the product. This proves the create
// path still works, through the same front door (`Run`, quoted prose as one
// argument) the sweep uses to prove the refusal — so the pair together state the
// whole contract: a lone bare word never writes, prose always does.
func TestFreeTextParentsStillFileProse(t *testing.T) {
	byName := map[string]*cobra.Command{}
	for _, p := range parents(t, NewRootCommand()) {
		byName[strings.Join(p.path, " ")] = p.cmd
	}
	for _, name := range freeTextParents {
		if _, ok := byName[name]; !ok {
			t.Errorf("freeTextParents names %q, which is not a parent in the command tree", name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			_ = captureLedgerRepo(t)
			var out, errOut strings.Builder
			if code := Run([]string{name, "widen the public api for downstream callers"}, &out, &errOut); code != 0 {
				t.Fatalf("abcd %s \"<prose>\" exited %d, want 0 (a genuine title must still file).\nstdout:\n%s\nstderr:\n%s",
					name, code, out.String(), errOut.String())
			}
		})
	}
}

type parentCmd struct {
	path []string
	cmd  *cobra.Command
}

// parents walks the command tree and returns every command that has
// sub-commands, with the argv path that reaches it. cobra's own generated `help`
// and `completion` commands are skipped: they are not part of abcd's surface.
func parents(t *testing.T, root *cobra.Command) []parentCmd {
	t.Helper()
	var found []parentCmd
	var walk func(cmd *cobra.Command, prefix []string)
	walk = func(cmd *cobra.Command, prefix []string) {
		for _, sub := range cmd.Commands() {
			if sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			path := append(append([]string{}, prefix...), sub.Name())
			if len(sub.Commands()) > 0 {
				found = append(found, parentCmd{path: path, cmd: sub})
				walk(sub, path)
			}
		}
	}
	walk(root, nil)
	if len(found) == 0 {
		t.Fatal("walked the command tree and found no parents — the walk is broken")
	}
	return found
}
