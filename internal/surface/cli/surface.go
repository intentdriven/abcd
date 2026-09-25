package cli

//go:generate go run ../../../cmd/abcd-gen-surface

import (
	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SurfaceSnapshotPath is the committed compatibility snapshot, relative to the
// repo root. The generator writes it, the drift test (surface_test.go) diffs the
// freshly-walked surface against it, and the release guardrail reads the
// baseline copy of it out of the last release tag — so all three agree on one
// location, declared once beside the type in internal/core/surface. It sits
// under the development record rather than under docs/ because it is a
// release-gate input, not something a reader of the documentation consumes.
const SurfaceSnapshotPath = surface.SnapshotPath

// SurfaceSnapshot builds the current compatibility surface: every command in the
// tree with its flags, plus the declared entries of the two plugin manifests
// under repoRoot.
//
// It walks the shared NewRootCommand() tree — the one canonical root command,
// the same one the CLI executes — rather than the Markdown reference walker.
// GenerateReference emits prose, returns early on hidden commands, and carries no
// structured requiredness or manifest data; reusing it would bake those blind
// spots into a compatibility gate. What is shared is the tree, which is the part
// that must not diverge.
//
// The tree is built fresh here and never executed. Cobra lazily attaches its
// default `help` and `completion` machinery during execution, so snapshotting an
// executed tree would record surface that a freshly-built one does not have, and
// the answer would depend on what else ran first in the process.
func SurfaceSnapshot(repoRoot string) (surface.Snapshot, error) {
	entries, err := surface.ManifestEntries(repoRoot)
	if err != nil {
		return surface.Snapshot{}, err
	}
	return surface.NewSnapshot(commandSurface(NewRootCommand()), entries), nil
}

// GenerateSurface renders the current surface as the committed artefact's bytes.
// Both the generator and the drift test call it, so the file that is written and
// the file that is checked are produced by one code path and can never disagree
// on formatting.
func GenerateSurface(repoRoot string) ([]byte, error) {
	snap, err := SurfaceSnapshot(repoRoot)
	if err != nil {
		return nil, err
	}
	return surface.Encode(snap)
}

// GuardSurface runs the release surface guardrail for the repository at
// repoRoot: it builds the CURRENT compatibility surface from the live command
// tree and the live manifests, and hands it to the core guardrail, which reads
// the baseline out of the last release tag and answers whether a narrowing was
// declared.
//
// This is the split the architecture requires. Building the current surface
// means walking cobra, which internal/core may not do; judging a break is domain
// logic, which must not depend on a transport. The front door therefore owns the
// walk and core owns the verdict, and the dependency points one way only.
//
// It is the entry point the ship flow calls. There is deliberately no `launch
// ship` verb yet — the write path is a later phase — so today this is reached
// from tests and from whatever front door composes the cut next; the guardrail
// itself is complete and does not change when that verb arrives.
func GuardSurface(repoRoot string) (changelog.SurfaceGuard, error) {
	current, err := SurfaceSnapshot(repoRoot)
	if err != nil {
		return changelog.SurfaceGuard{}, err
	}
	return changelog.GuardSurface(repoRoot, current)
}

// commandSurface flattens a command tree into snapshot entries, depth-first from
// cmd.
//
// Hidden commands are recorded, not skipped: the operator-internal `hook` subtree
// is invoked by name from harness wiring, so removing it breaks installations
// even though the documentation never mentions it. Ordering is left to
// surface.NewSnapshot, which sorts by command path — relying on cobra's traversal
// order would make the artefact depend on registration order.
func commandSurface(cmd *cobra.Command) []surface.Command {
	out := []surface.Command{{
		Path:   cmd.CommandPath(),
		Hidden: cmd.Hidden,
		// The help placement (itd-146): recorded so a regroup is visible in
		// the committed tree even though it changes no invocation.
		Group: helpGroup(cmd),
		Block: helpBlock(cmd),
		Flags: commandFlags(cmd),
	}}
	for _, child := range cmd.Commands() {
		out = append(out, commandSurface(child)...)
	}
	return out
}

// commandFlags reads the flags a command declares itself — its own flags plus the
// persistent flags it defines, which is exactly what cobra's LocalFlags excludes
// inherited persistent flags from. A persistent flag is therefore recorded once,
// where it is declared, so removing `--json` from the root reads as one break
// rather than one per command in the tree.
func commandFlags(cmd *cobra.Command) []surface.Flag {
	var out []surface.Flag
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		out = append(out, surface.Flag{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Type:      f.Value.Type(),
			Required:  flagRequired(f),
			Hidden:    f.Hidden,
		})
	})
	return out
}

// flagRequired reports whether cobra considers the flag mandatory. Cobra records
// requiredness as an annotation set by MarkFlagRequired rather than as a field,
// and it reads that annotation as "present and its first value is true"; this
// mirrors that reading exactly, so the snapshot's answer is the one the runtime
// would give.
func flagRequired(f *pflag.Flag) bool {
	values, found := f.Annotations[cobra.BashCompOneRequiredFlag]
	return found && len(values) > 0 && values[0] == "true"
}
