package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
)

// testRepoRoot locates the repo root from this package directory
// (internal/surface/cli), so the snapshot tests read the real manifests and the
// real committed baseline.
func testRepoRoot() string { return filepath.Join("..", "..", "..") }

func findCommand(snap surface.Snapshot, path string) (surface.Command, bool) {
	for _, c := range snap.Commands {
		if c.Path == path {
			return c, true
		}
	}
	return surface.Command{}, false
}

func findFlag(cmd surface.Command, name string) (surface.Flag, bool) {
	for _, f := range cmd.Flags {
		if f.Name == name {
			return f, true
		}
	}
	return surface.Flag{}, false
}

// TestSurfaceSnapshotIncludesHiddenCommands is the reason this walk exists rather
// than reusing the Markdown reference walker: the operator-internal `hook`
// subtree is hidden from the docs page but IS public surface for compatibility
// purposes, because harness wiring invokes it by name. A snapshot that skipped it
// would let a hook removal ship as a non-breaking release.
func TestSurfaceSnapshotIncludesHiddenCommands(t *testing.T) {
	snap, err := SurfaceSnapshot(testRepoRoot())
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}

	// Hidden is recorded as each command DECLARES it, which is why the subtree
	// below a hidden parent reports Hidden=false: cobra hides descendants by
	// never rendering the parent, not by marking them. Presence is what the
	// guardrail diffs, so both must be present either way.
	tests := []struct {
		path       string
		wantHidden bool
	}{
		{"abcd", false},
		{"abcd hook", true},
		{"abcd hook prompt-router", false},
	}
	for _, tc := range tests {
		cmd, ok := findCommand(snap, tc.path)
		if !ok {
			t.Fatalf("%q missing from the snapshot; the walk is skipping hidden commands", tc.path)
		}
		if cmd.Hidden != tc.wantHidden {
			t.Fatalf("%q hidden = %v, want %v", tc.path, cmd.Hidden, tc.wantHidden)
		}
	}
}

// TestSurfaceSnapshotRecordsFlagDetail checks the structured per-flag data the
// Markdown reference cannot carry: a persistent flag is recorded once on the
// command that declares it (not repeated on every descendant that inherits it),
// and its type travels with it.
func TestSurfaceSnapshotRecordsFlagDetail(t *testing.T) {
	snap, err := SurfaceSnapshot(testRepoRoot())
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}

	root, ok := findCommand(snap, "abcd")
	if !ok {
		t.Fatalf("root command missing")
	}
	jsonFlag, ok := findFlag(root, "json")
	if !ok {
		t.Fatalf("--json missing from the root command's flags: %+v", root.Flags)
	}
	if jsonFlag.Type != "bool" {
		t.Fatalf("--json type = %q, want %q", jsonFlag.Type, "bool")
	}
	if jsonFlag.Required {
		t.Fatalf("--json recorded as required")
	}

	update, ok := findCommand(snap, "abcd update")
	if !ok {
		t.Fatalf("`abcd update` missing")
	}
	if _, inherited := findFlag(update, "json"); inherited {
		t.Fatalf("inherited persistent flag recorded on a subcommand; it must be recorded only where it is declared")
	}
}

// TestCommandSurfaceExtractsRequiredness tests requiredness against a SYNTHETIC
// tree, because nothing in the live tree marks a flag required — asserting only
// against the live tree would be asserting that an always-false field is false.
// A flag becoming required is a break, so the extraction must work the day it
// first happens rather than the day someone notices it never did.
func TestCommandSurfaceExtractsRequiredness(t *testing.T) {
	root := &cobra.Command{Use: "synth"}
	child := &cobra.Command{Use: "child"}
	child.Flags().String("must", "", "a required flag")
	child.Flags().StringP("optional", "o", "", "an optional flag")
	child.Flags().Bool("secret", false, "a hidden flag")
	if err := child.MarkFlagRequired("must"); err != nil {
		t.Fatalf("MarkFlagRequired: %v", err)
	}
	if err := child.Flags().MarkHidden("secret"); err != nil {
		t.Fatalf("MarkHidden: %v", err)
	}
	root.AddCommand(child)

	snap := surface.NewSnapshot(commandSurface(root), nil)
	cmd, ok := findCommand(snap, "synth child")
	if !ok {
		t.Fatalf("synthetic child missing from %+v", snap.Commands)
	}

	tests := []struct {
		flag  string
		want  surface.Flag
		label string
	}{
		{"must", surface.Flag{Name: "must", Shorthand: "", Type: "string", Required: true, Hidden: false}, "required flag"},
		{"optional", surface.Flag{Name: "optional", Shorthand: "o", Type: "string", Required: false, Hidden: false}, "optional flag with a shorthand"},
		{"secret", surface.Flag{Name: "secret", Shorthand: "", Type: "bool", Required: false, Hidden: true}, "hidden flag"},
	}
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			got, ok := findFlag(cmd, tc.flag)
			if !ok {
				t.Fatalf("--%s missing from %+v", tc.flag, cmd.Flags)
			}
			if got != tc.want {
				t.Fatalf("--%s = %+v, want %+v", tc.flag, got, tc.want)
			}
		})
	}
}

// TestLiveTreeMarksNoFlagRequired records the state the requiredness field is a
// tripwire for. Nothing marks a flag required today; when something does, this
// test fails and the author is pointed at the fact that the change is a break
// needing a `breaking` record in the cut.
func TestLiveTreeMarksNoFlagRequired(t *testing.T) {
	snap, err := SurfaceSnapshot(testRepoRoot())
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	for _, cmd := range snap.Commands {
		for _, f := range cmd.Flags {
			if f.Required {
				t.Fatalf("`%s --%s` is now required: a previously optional flag becoming required is a "+
					"surface break and needs a `breaking` record in the release cut", cmd.Path, f.Name)
			}
		}
	}
}

// TestGenerateSurfaceIsDeterministic is the phase's stop condition made
// executable: a guardrail built on a wobbling snapshot is worse than none,
// because it either cries wolf every release or is switched off. Many runs in one
// process catch ordering that depends on map iteration, on cobra's registration
// order, or on any per-run state the walk accumulates.
func TestGenerateSurfaceIsDeterministic(t *testing.T) {
	root := testRepoRoot()
	first, err := GenerateSurface(root)
	if err != nil {
		t.Fatalf("GenerateSurface: %v", err)
	}
	const runs = 50
	for i := 1; i < runs; i++ {
		again, err := GenerateSurface(root)
		if err != nil {
			t.Fatalf("GenerateSurface run %d: %v", i, err)
		}
		if string(again) != string(first) {
			t.Fatalf("GenerateSurface is not deterministic: run %d differs from run 0 at %s",
				i, firstDiff(string(first), string(again)))
		}
	}
}

// TestSurfaceSnapshotMatchesCommittedBaseline is the drift gate for the committed
// snapshot, mirroring the CLI reference's gate. It regenerates the surface from
// the live command tree and the live manifests and diffs it against the committed
// artefact, so a command, flag, or manifest entry can never change without the
// baseline moving with it — which is what makes the release guardrail's diff
// meaningful.
func TestSurfaceSnapshotMatchesCommittedBaseline(t *testing.T) {
	root := testRepoRoot()
	baseline := filepath.Join(root, filepath.FromSlash(SurfaceSnapshotPath))
	committed, err := os.ReadFile(baseline)
	if err != nil {
		t.Fatalf("cannot read committed surface snapshot %s: %v\n"+
			"regenerate it with `go generate ./internal/surface/cli`", SurfaceSnapshotPath, err)
	}

	want, err := GenerateSurface(root)
	if err != nil {
		t.Fatalf("GenerateSurface: %v", err)
	}
	if string(committed) != string(want) {
		// A regroup changes no invocation, so name each moved verb rather than
		// leave it to a line and column (itd-146 criterion 4).
		var moved string
		if base, err := surface.Decode(committed); err == nil {
			if now, err := surface.Decode(want); err == nil {
				if lines := surface.PlacementChanges(base, now); len(lines) > 0 {
					moved = "\nhelp placement moved without regenerating:\n  - " + strings.Join(lines, "\n  - ")
				}
			}
		}
		t.Fatalf("%s is stale: the committed surface no longer matches the command tree and manifests.\n"+
			"Regenerate it with `go generate ./internal/surface/cli` and commit the result.\n"+
			"first difference at %s%s", SurfaceSnapshotPath, firstDiff(string(committed), string(want)), moved)
	}
}

// TestCommittedBaselineDecodes proves the committed artefact is readable by the
// decoder the release guardrail uses, not merely byte-equal to what the generator
// produced.
func TestCommittedBaselineDecodes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(SurfaceSnapshotPath)))
	if err != nil {
		t.Fatalf("reading %s: %v", SurfaceSnapshotPath, err)
	}
	snap, err := surface.Decode(data)
	if err != nil {
		t.Fatalf("Decode(%s): %v", SurfaceSnapshotPath, err)
	}
	if len(snap.Commands) == 0 || len(snap.Manifest) == 0 {
		t.Fatalf("committed baseline decodes to an empty surface: %d commands, %d manifest entries",
			len(snap.Commands), len(snap.Manifest))
	}
}
