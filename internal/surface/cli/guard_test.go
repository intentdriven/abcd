package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/spf13/cobra"
)

// guardFixture is this package's name for the shared hermetic git fixture,
// gittest.Repo, plus the plugin manifests the surface walk reads. The front door
// walks the LIVE cobra tree and reads the LIVE manifests under a root, so the
// only honest exercise of it is a real repo with real manifests, a real tag, and
// a real snapshot blob in that tag's tree.
//
// It delegates to gittest.Repo rather than carrying its own copy: the fixture is
// shared with internal/core/changelog and with the ship verb's tests, and three
// private copies would drift apart.
type guardFixture struct {
	*gittest.Repo
	t    *testing.T
	root string
}

func newGuardFixture(t *testing.T) *guardFixture {
	t.Helper()
	r := newRepo(t)
	return &guardFixture{Repo: r, t: t, root: r.Root()}
}

func (f *guardFixture) git(args ...string)  { f.Git(args...) }
func (f *guardFixture) write(rel, c string) { f.Write(rel, c) }
func (f *guardFixture) commit(msg string)   { f.Commit(msg) }

func (f *guardFixture) writeManifests() {
	f.t.Helper()
	f.write(".claude-plugin/plugin.json", `{"name":"abcd","description":"fixture"}`+"\n")
	f.write(".claude-plugin/marketplace.json", `{"name":"abcd","plugins":[{"name":"abcd","source":"./"}]}`+"\n")
}

// TestGuardSurfaceEndToEnd exercises the whole wired path: the front door walks
// the live cobra tree, reads the live manifests, and hands the result to the
// core guardrail, which reads its baseline out of the release tag.
//
// The baseline planted in the tag is the live surface PLUS a command that does
// not exist ("abcd ghost"), so from the tag's point of view the release removed
// it. The snapshot committed at HEAD is the live surface — exactly what the
// drift test enforces in the real repo — which means a guardrail comparing the
// committed file against the current tree would see nothing at all. Detecting
// the removal proves the baseline came from the tag.
func TestGuardSurfaceEndToEnd(t *testing.T) {
	f := newGuardFixture(t)
	f.writeManifests()

	live, err := SurfaceSnapshot(f.root)
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	tagged := surface.NewSnapshot(
		append(append([]surface.Command{}, live.Commands...), surface.Command{Path: "abcd ghost"}),
		live.Manifest)

	writeEncoded(t, f, tagged)
	f.commit("seed the surface baseline")
	f.git("tag", "v0.4.0")

	writeEncoded(t, f, live)
	f.write(".abcd/development/intents/shipped/itd-1-thing.md", "---\nid: itd-1\nimpact: additive\n---\n# itd-1\n")
	f.commit("ship an additive intent and regenerate the snapshot")

	got, err := GuardSurface(f.root)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != changelog.SurfaceGuardFailed {
		t.Fatalf("Status = %q (reason %q), want %q", got.Status, got.Reason, changelog.SurfaceGuardFailed)
	}
	if !strings.Contains(got.Reason, "abcd ghost") {
		t.Errorf("Reason = %q, want it to name the removed command", got.Reason)
	}
	if got.BaseTag != "v0.4.0" {
		t.Errorf("BaseTag = %q, want v0.4.0", got.BaseTag)
	}
}

// TestGuardSurfaceEndToEndPassesUnchangedSurface is the other direction of the
// same wiring: a tag whose baseline IS the live surface reports a clean pass, so
// the failing case above is the guardrail firing rather than the front door
// being broken.
func TestGuardSurfaceEndToEndPassesUnchangedSurface(t *testing.T) {
	f := newGuardFixture(t)
	f.writeManifests()

	live, err := SurfaceSnapshot(f.root)
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	writeEncoded(t, f, live)
	f.commit("seed the surface baseline")
	f.git("tag", "v0.4.0")

	f.write(".abcd/development/intents/shipped/itd-1-thing.md", "---\nid: itd-1\nimpact: additive\n---\n# itd-1\n")
	f.commit("ship an additive intent")

	got, err := GuardSurface(f.root)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != changelog.SurfaceGuardPassed {
		t.Fatalf("Status = %q (reason %q), want %q", got.Status, got.Reason, changelog.SurfaceGuardPassed)
	}
	if len(got.Breaks) != 0 {
		t.Errorf("Breaks = %v, want none", got.Breaks)
	}
}

// TestGuardSurfaceEndToEndRefusesWithoutBaseline pins the fail-closed refusal
// through the front door — the repository's real state today, because v0.3.0
// predates the snapshot. The message has to be actionable, so it names the tag,
// the missing artefact, and the manual roll that resolves it.
func TestGuardSurfaceEndToEndRefusesWithoutBaseline(t *testing.T) {
	f := newGuardFixture(t)
	f.writeManifests()
	f.commit("a release that predates the surface baseline")
	f.git("tag", "v0.3.0")

	live, err := SurfaceSnapshot(f.root)
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	writeEncoded(t, f, live)
	f.commit("seed the surface baseline after the release")

	got, err := GuardSurface(f.root)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != changelog.SurfaceGuardRefused {
		t.Fatalf("Status = %q, want %q: the first cut must not sail through unguarded", got.Status, changelog.SurfaceGuardRefused)
	}
	for _, want := range []string{"v0.3.0", SurfaceSnapshotPath, "manual roll"} {
		if !strings.Contains(got.Reason, want) {
			t.Errorf("Reason = %q, want it to mention %q", got.Reason, want)
		}
	}
}

// TestGuardSurfaceEndToEndRefusesStaleBinary pins the fail-closed rule that
// covers this front door's one structural weakness: the current surface is walked
// from the command tree COMPILED INTO the running binary, while every other input
// is read out of the repository at repoRoot. An installed release binary
// therefore reports the surface of the release it was built from, and a command
// removed since that build would be missing from both sides of the diff — a real
// break shipping as additive, invisibly.
//
// The fixture is exactly that shape: HEAD's committed snapshot has a command
// dropped, so it disagrees with the tree this binary walks. There is no way to
// tell "stale binary" from "stale snapshot" apart from here, so the guardrail
// refuses and the message names both remedies.
func TestGuardSurfaceEndToEndRefusesStaleBinary(t *testing.T) {
	f := newGuardFixture(t)
	f.writeManifests()

	live, err := SurfaceSnapshot(f.root)
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	if len(live.Commands) < 2 {
		t.Fatalf("live surface has %d commands, want a tree to drop one from", len(live.Commands))
	}
	writeEncoded(t, f, live)
	f.commit("seed the surface baseline")
	f.git("tag", "v0.4.0")

	// The tree being released removed a command and regenerated the snapshot;
	// the binary doing the walking predates that removal.
	writeEncoded(t, f, surface.NewSnapshot(live.Commands[1:], live.Manifest))
	f.write(".abcd/development/intents/shipped/itd-1-thing.md", "---\nid: itd-1\nimpact: additive\n---\n# itd-1\n")
	f.commit("remove a command and regenerate the snapshot")

	got, err := GuardSurface(f.root)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != changelog.SurfaceGuardRefused {
		t.Fatalf("Status = %q (reason %q), want %q: a binary that disagrees with the tree cannot guard it",
			got.Status, got.Reason, changelog.SurfaceGuardRefused)
	}
	for _, want := range []string{SurfaceSnapshotPath, "rebuild", "regenerate"} {
		if !strings.Contains(got.Reason, want) {
			t.Errorf("Reason = %q, want it to mention %q", got.Reason, want)
		}
	}
}

// TestHelpTextIsNotSurface pins the taxonomy row the snapshot answers by not
// modelling it: changed help, description, and summary text are not breaks. Two
// trees identical but for their prose must produce identical surface and diff
// clean — which is why the snapshot carries no prose at all.
func TestHelpTextIsNotSurface(t *testing.T) {
	build := func(short, long, example string) surface.Snapshot {
		root := &cobra.Command{Use: "abcd", Short: short, Long: long}
		child := &cobra.Command{Use: "plan", Short: short, Long: long, Example: example}
		child.Flags().String("since", "", short)
		root.AddCommand(child)
		return surface.NewSnapshot(commandSurface(root), nil)
	}

	before := build("does a thing", "the long version", "abcd plan")
	after := build("REWORDED entirely", "a completely different long description", "abcd plan --since v1")

	if breaks := surface.Diff(before, after); len(breaks) != 0 {
		t.Errorf("Diff = %v, want none: reworded help is not a compatibility break", breaks)
	}
}

func writeEncoded(t *testing.T, f *guardFixture, snap surface.Snapshot) {
	t.Helper()
	data, err := surface.Encode(snap)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	f.write(SurfaceSnapshotPath, string(data))
}
