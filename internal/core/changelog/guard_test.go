package changelog

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// The fixture constructors keep the taxonomy table readable.
func cmdOf(path string, flags ...surface.Flag) surface.Command {
	return surface.Command{Path: path, Flags: flags}
}
func optFlag(name string) surface.Flag { return surface.Flag{Name: name, Type: "bool"} }
func reqFlag(name string) surface.Flag {
	return surface.Flag{Name: name, Type: "bool", Required: true}
}
func entry(key string) surface.ManifestEntry {
	return surface.ManifestEntry{File: ".claude-plugin/plugin.json", Key: key}
}

// writeSurface writes an encoded snapshot into the fixture's working tree at the
// one canonical path, so a fixture exercises exactly the location the generator
// writes and the guardrail reads.
func writeSurface(r *fixtureRepo, snap surface.Snapshot) {
	r.t.Helper()
	data, err := surface.Encode(snap)
	if err != nil {
		r.t.Fatalf("Encode: %v", err)
	}
	r.write(surface.SnapshotPath, string(data))
}

// guardRepo builds the ONLY history shape that makes the guardrail meaningful:
// baseline is committed and TAGGED as the last release, and then the working
// tree's snapshot is replaced by current and committed on top.
//
// That second step is deliberate and is the fixture's whole point. The committed
// snapshot is kept equal to the live tree by the drift test, so a guardrail that
// compared "the committed file" against "the current tree" would be comparing a
// file with itself and could never fire. Every fixture built here reproduces
// that: HEAD's committed snapshot IS current. A break is therefore only visible
// to a guardrail that reads its baseline out of the TAG.
//
// impacts are the records the cut ships, one shipped intent each — the input the
// guardrail asks "is a break declared?" of.
func guardRepo(t *testing.T, baseline, current surface.Snapshot, impacts ...string) *fixtureRepo {
	t.Helper()
	r := newFixtureRepo(t)
	writeSurface(r, baseline)
	r.commit("seed the surface baseline")
	r.git("tag", "v0.4.0")

	writeSurface(r, current)
	r.commit("regenerate the surface snapshot")
	for i, impact := range impacts {
		id := "itd-" + string(rune('1'+i))
		r.record(".abcd/development/intents/shipped/"+id+"-thing.md", id, impact)
		r.commit("ship " + id)
	}
	return r
}

// TestGuardSurfaceTaxonomy walks every row of the break taxonomy (spc-10 AC 4,
// plan outcome 5) end to end through real git objects, in BOTH directions: what
// must fail the cut, and what must pass it. Each row gets its own fixture repo
// whose tagged snapshot genuinely differs from the current tree.
func TestGuardSurfaceTaxonomy(t *testing.T) {
	tests := []struct {
		name       string
		baseline   surface.Snapshot
		current    surface.Snapshot
		impacts    []string
		wantStatus SurfaceGuardStatus
		wantNamed  string
	}{
		{
			name:       "removed command fails and names the command",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardFailed,
			wantNamed:  "abcd ghost",
		},
		{
			name:       "renamed command fails and names the old path",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd spirit")}, nil),
			impacts:    []string{"fix"},
			wantStatus: SurfaceGuardFailed,
			wantNamed:  "abcd ghost",
		},
		{
			name:       "removed flag fails and names the flag",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd", optFlag("json"), optFlag("quiet"))}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd", optFlag("json"))}, nil),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardFailed,
			wantNamed:  "abcd --quiet",
		},
		{
			name:       "renamed flag fails and names the old flag",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd", optFlag("json"))}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd", optFlag("jsonl"))}, nil),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardFailed,
			wantNamed:  "abcd --json",
		},
		{
			name:       "optional flag becoming required fails",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd launch", optFlag("dry-run"))}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd launch", reqFlag("dry-run"))}, nil),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardFailed,
			wantNamed:  "abcd launch --dry-run",
		},
		{
			name:       "absent flag arriving required on an existing command fails",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd launch")}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd launch", reqFlag("token"))}, nil),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardFailed,
			wantNamed:  "abcd launch --token",
		},
		{
			name:       "removed manifest entry fails and names file and key",
			baseline:   surface.NewSnapshot(nil, []surface.ManifestEntry{entry("author.name"), entry("description")}),
			current:    surface.NewSnapshot(nil, []surface.ManifestEntry{entry("author.name")}),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardFailed,
			wantNamed:  ".claude-plugin/plugin.json:description",
		},
		{
			name:       "new command passes",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd brand-new")}, nil),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardPassed,
		},
		{
			name:       "new optional flag passes",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd", optFlag("json"))}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd", optFlag("json"), optFlag("verbose"))}, nil),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardPassed,
		},
		{
			name:       "new command carrying a required flag passes",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ship", reqFlag("changelog-json"))}, nil),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardPassed,
		},
		{
			name: "reordering passes",
			baseline: surface.NewSnapshot(
				[]surface.Command{cmdOf("abcd ghost", optFlag("quiet"), optFlag("json")), cmdOf("abcd")},
				[]surface.ManifestEntry{entry("description"), entry("author.name")}),
			current: surface.NewSnapshot(
				[]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost", optFlag("json"), optFlag("quiet"))},
				[]surface.ManifestEntry{entry("author.name"), entry("description")}),
			impacts:    []string{"additive"},
			wantStatus: SurfaceGuardPassed,
		},
		{
			name:       "removed command with a breaking record in the cut passes",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil),
			impacts:    []string{"breaking"},
			wantStatus: SurfaceGuardPassed,
		},
		{
			name:       "newly-required flag with a breaking record in the cut passes",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd launch", optFlag("dry-run"))}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd launch", reqFlag("dry-run"))}, nil),
			impacts:    []string{"breaking"},
			wantStatus: SurfaceGuardPassed,
		},
		{
			name:       "removed manifest entry with a breaking record in the cut passes",
			baseline:   surface.NewSnapshot(nil, []surface.ManifestEntry{entry("author.name"), entry("description")}),
			current:    surface.NewSnapshot(nil, []surface.ManifestEntry{entry("author.name")}),
			impacts:    []string{"breaking"},
			wantStatus: SurfaceGuardPassed,
		},
		{
			name:       "a breaking record among several declares the break",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil),
			impacts:    []string{"fix", "breaking", "internal"},
			wantStatus: SurfaceGuardPassed,
		},
		{
			name:       "a break with no records at all in the cut fails",
			baseline:   surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil),
			current:    surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil),
			wantStatus: SurfaceGuardFailed,
			wantNamed:  "abcd ghost",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := guardRepo(t, tc.baseline, tc.current, tc.impacts...)

			got, err := GuardSurface(r.root, tc.current)
			if err != nil {
				t.Fatalf("GuardSurface: %v", err)
			}
			if got.Status != tc.wantStatus {
				t.Fatalf("Status = %q (reason %q), want %q", got.Status, got.Reason, tc.wantStatus)
			}
			if got.BaseTag != "v0.4.0" {
				t.Errorf("BaseTag = %q, want v0.4.0", got.BaseTag)
			}
			if tc.wantNamed == "" {
				return
			}
			if !strings.Contains(got.Reason, tc.wantNamed) {
				t.Errorf("Reason = %q, want it to name the changed surface %q", got.Reason, tc.wantNamed)
			}
		})
	}
}

// TestGuardSurfaceReadsBaselineFromTagNotWorkingTree is the anti-simplification
// test: it fails the moment the guardrail is "simplified" to compare the
// committed snapshot against the current tree.
//
// The drift test keeps the committed snapshot equal to the live tree, so that
// comparison is a file against itself and can never report a break. The fixture
// asserts the trap condition explicitly — HEAD's committed bytes ARE the current
// snapshot — and then asserts the break is still detected, which is only
// possible if the baseline came out of the tag.
func TestGuardSurfaceReadsBaselineFromTagNotWorkingTree(t *testing.T) {
	baseline := surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil)
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	r := guardRepo(t, baseline, current, "additive")

	wantCommitted, err := surface.Encode(current)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	committed := r.git("show", "HEAD:"+surface.SnapshotPath)
	if strings.TrimSpace(string(wantCommitted)) != committed {
		t.Fatalf("fixture precondition broken: HEAD's snapshot must equal the current tree, "+
			"otherwise this test does not exercise the trap it exists for\ncommitted: %s", committed)
	}

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardFailed {
		t.Fatalf("Status = %q, want %q: the baseline must be read from %s, not from the committed file "+
			"(which the drift test keeps equal to the current tree)", got.Status, SurfaceGuardFailed, got.BaseTag)
	}
	if !strings.Contains(got.Reason, "abcd ghost") {
		t.Errorf("Reason = %q, want it to name abcd ghost", got.Reason)
	}
}

// TestGuardSurfaceIgnoresBreakingOnTheRemovedSide pins that only what the cut
// ADDS can declare a break.
//
// The fixture is ordinary maintenance, not an attack: the tag's tree carries a
// shipped intent labelled `breaking` (the historical back-fill puts such labels
// on old intents), and this cut supersedes it — the record leaves shipped/ — while
// removing a command and shipping nothing but an `additive` intent. Judging the
// declaration over the whole cut would let the LAST release's label wave through
// a narrowing that nothing in THIS release declares, silently and with the break
// named in the result.
func TestGuardSurfaceIgnoresBreakingOnTheRemovedSide(t *testing.T) {
	baseline := surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil)
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	superseded := ".abcd/development/intents/shipped/itd-9-superseded.md"

	r := newFixtureRepo(t)
	writeSurface(r, baseline)
	r.record(superseded, "itd-9", "breaking")
	r.commit("seed the surface baseline and a shipped breaking intent")
	r.git("tag", "v0.4.0")

	writeSurface(r, current)
	r.remove(superseded)
	r.record(".abcd/development/intents/shipped/itd-10-thing.md", "itd-10", "additive")
	r.commit("supersede itd-9 and ship itd-10")

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.BreakingDeclared {
		t.Errorf("BreakingDeclared = true, want false: the only breaking record is LEAVING the release")
	}
	if got.Status != SurfaceGuardFailed {
		t.Fatalf("Status = %q (reason %q), want %q: a superseded record's label must not declare this cut's break",
			got.Status, got.Reason, SurfaceGuardFailed)
	}
	if !strings.Contains(got.Reason, "abcd ghost") {
		t.Errorf("Reason = %q, want it to name the undeclared removal", got.Reason)
	}
}

// TestGuardSurfaceRefusesWhenCurrentDiffersFromHead pins the stale-binary
// fail-closed rule. The caller's `current` is walked from the command tree
// compiled into the RUNNING binary; when that binary was built before the tree
// being released, it reproduces the last release's surface — so the guardrail
// would compare the baseline against itself and wave through every removal made
// since. The fixture is exactly that: HEAD commits the narrowed surface while the
// caller hands in the baseline's.
func TestGuardSurfaceRefusesWhenCurrentDiffersFromHead(t *testing.T) {
	baseline := surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil)
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	r := guardRepo(t, baseline, current, "additive")

	// A binary built before `abcd ghost` was removed still walks the baseline.
	got, err := GuardSurface(r.root, baseline)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardRefused {
		t.Fatalf("Status = %q (reason %q), want %q: a surface that disagrees with HEAD cannot be guarded",
			got.Status, got.Reason, SurfaceGuardRefused)
	}
	if !strings.Contains(got.Reason, surface.SnapshotPath) {
		t.Errorf("Reason = %q, want it to name the snapshot path", got.Reason)
	}
	if len(got.Breaks) != 0 {
		t.Errorf("Breaks = %v, want none: nothing trustworthy was compared", got.Breaks)
	}
}

// TestGuardSurfaceRefusesWithoutBaseline pins the fail-closed no-baseline rule
// (plan outcome 5): a tag with no snapshot in its tree cannot be diffed, and a
// cut that cannot be diffed refuses instead of passing. That refusal is the
// repo's real state until a release is cut that CONTAINS the baseline, so the
// message has to say what puts it there.
func TestGuardSurfaceRefusesWithoutBaseline(t *testing.T) {
	r := newFixtureRepo(t)
	r.write("README.md", "# fixture\n")
	r.commit("a release that predates the surface baseline")
	r.git("tag", "v0.3.0")
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	writeSurface(r, current)
	r.commit("seed the surface baseline after the release")

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardRefused {
		t.Fatalf("Status = %q, want %q: a cut with no baseline must refuse, never silently pass",
			got.Status, SurfaceGuardRefused)
	}
	for _, want := range []string{"v0.3.0", surface.SnapshotPath, "refus"} {
		if !strings.Contains(got.Reason, want) {
			t.Errorf("Reason = %q, want it to mention %q", got.Reason, want)
		}
	}
	if len(got.Breaks) != 0 {
		t.Errorf("Breaks = %v, want none: nothing was compared", got.Breaks)
	}
}

// TestGuardSurfaceRefusesWithoutTag pins the other missing anchor: with no
// release tag there is no baseline to read at all, so the guardrail refuses for
// the same reason derivation does rather than reporting a clean surface.
func TestGuardSurfaceRefusesWithoutTag(t *testing.T) {
	r := newFixtureRepo(t)
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	writeSurface(r, current)
	r.commit("seed the surface baseline")

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardRefused {
		t.Fatalf("Status = %q, want %q", got.Status, SurfaceGuardRefused)
	}
	if !strings.Contains(got.Reason, "release tag") {
		t.Errorf("Reason = %q, want it to name the missing release tag", got.Reason)
	}
}

// TestGuardSurfaceReportsEveryBreak pins that one failing cut names every
// changed surface, not just the first. An operator who has to fix, re-run, and
// discover the next break one at a time will stop reading the message.
func TestGuardSurfaceReportsEveryBreak(t *testing.T) {
	baseline := surface.NewSnapshot(
		[]surface.Command{cmdOf("abcd", optFlag("json")), cmdOf("abcd ghost")},
		[]surface.ManifestEntry{entry("author.name")})
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	r := guardRepo(t, baseline, current, "additive")

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardFailed {
		t.Fatalf("Status = %q, want %q", got.Status, SurfaceGuardFailed)
	}
	if len(got.Breaks) != 3 {
		t.Fatalf("Breaks = %v, want three (a flag, a command, a manifest entry)", got.Breaks)
	}
	for _, want := range []string{"abcd --json", "abcd ghost", ".claude-plugin/plugin.json:author.name"} {
		if !strings.Contains(got.Reason, want) {
			t.Errorf("Reason = %q, want it to name %q", got.Reason, want)
		}
	}
}

// TestGuardSurfaceKeepsBreaksWhenDeclared pins that a declared break is still
// REPORTED, not discarded: the release notes have to describe what broke, so a
// passing guard still hands its caller the surfaces that changed.
func TestGuardSurfaceKeepsBreaksWhenDeclared(t *testing.T) {
	baseline := surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil)
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	r := guardRepo(t, baseline, current, "breaking")

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardPassed || !got.BreakingDeclared {
		t.Fatalf("Status = %q, BreakingDeclared = %v, want passed and declared", got.Status, got.BreakingDeclared)
	}
	if len(got.Breaks) != 1 || got.Breaks[0].Surface != "abcd ghost" {
		t.Errorf("Breaks = %v, want the declared break to still be reported", got.Breaks)
	}
	if got.Reason != "" {
		t.Errorf("Reason = %q, want empty on a passing guard", got.Reason)
	}
}

// TestGuardSurfaceRejectsUnreadableBaseline pins the difference between a
// refusal and an error. An ABSENT baseline is a state the operator resolves by
// cutting a release; a baseline that is present but cannot be decoded means the
// tag's artefact is corrupt, and treating it as "no surface" would make every
// command in the release look like surface that never existed.
func TestGuardSurfaceRejectsUnreadableBaseline(t *testing.T) {
	r := newFixtureRepo(t)
	r.write(surface.SnapshotPath, "{ not json\n")
	r.commit("a corrupt baseline")
	r.git("tag", "v0.4.0")
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	writeSurface(r, current)
	r.commit("regenerate the surface snapshot")

	if _, err := GuardSurface(r.root, current); err == nil {
		t.Fatal("GuardSurface = nil error, want a corrupt baseline to be an error, not a pass")
	}
}

// TestGuardSurfaceNamesARegroupedVerb is itd-146 criterion 4 at the release
// gate. A verb moved to another help group without the snapshot being
// regenerated leaves the binary's surface disagreeing with HEAD's, which the gate
// already refuses; what it did not do was say WHERE. The refusal names the verb
// and both placements, so the operator does not diff two long files by hand.
func TestGuardSurfaceNamesARegroupedVerb(t *testing.T) {
	committed := surface.NewSnapshot([]surface.Command{
		cmdOf("abcd"), {Path: "abcd capture", Group: "records", Block: "people"},
	}, nil)
	r := guardRepo(t, committed, committed, "additive")

	moved := surface.NewSnapshot([]surface.Command{
		cmdOf("abcd"), {Path: "abcd capture", Group: "checks", Block: "people"},
	}, nil)
	got, err := GuardSurface(r.root, moved)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardRefused {
		t.Fatalf("Status = %q (reason %q), want %q", got.Status, got.Reason, SurfaceGuardRefused)
	}
	if !strings.Contains(got.Reason, "abcd capture: group records → checks") {
		t.Errorf("Reason = %q, want it to name the regrouped verb and both groups", got.Reason)
	}
}

// TestGuardSurfaceNamesARewordedSentence is the sentence's half of the same
// refusal (itd-2609212113220149): a rewording changes no invocation, so nothing
// but the stale-surface comparison sees it, and the refusal names each command
// whose sentence differs from HEAD's snapshot the way it names a moved verb.
func TestGuardSurfaceNamesARewordedSentence(t *testing.T) {
	committed := surface.NewSnapshot([]surface.Command{
		cmdOf("abcd"), {Path: "abcd capture", Sentence: "Capture an issue: Writes a record; refuses a lone word."},
	}, nil)
	r := guardRepo(t, committed, committed, "additive")

	reworded := surface.NewSnapshot([]surface.Command{
		cmdOf("abcd"), {Path: "abcd capture", Sentence: "File an issue: Writes a record; refuses a lone word."},
	}, nil)
	got, err := GuardSurface(r.root, reworded)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardRefused {
		t.Fatalf("Status = %q (reason %q), want %q", got.Status, got.Reason, SurfaceGuardRefused)
	}
	if !strings.Contains(got.Reason, "sentence differs from HEAD's snapshot:\n  - abcd capture") {
		t.Errorf("Reason = %q, want it to name the command whose sentence was reworded", got.Reason)
	}
	if strings.Contains(got.Reason, "abcd:") {
		t.Errorf("Reason = %q, names a command whose sentence did not change", got.Reason)
	}
}

// TestGuardSurfaceReadsAVersionOneBaseline pins the schema bump's compatibility
// half end to end: the last release tag carries a version-1 snapshot (no
// placement fields), HEAD carries the current version, and the cut is guarded
// rather than failing to decode its own baseline.
func TestGuardSurfaceReadsAVersionOneBaseline(t *testing.T) {
	r := newFixtureRepo(t)
	r.write(surface.SnapshotPath, `{"schema_version":1,"commands":[{"path":"abcd","hidden":false,"flags":[]}],"manifest":[]}`+"\n")
	r.commit("a version-1 baseline, as every release before itd-146 carries")
	r.git("tag", "v0.4.0")
	current := surface.NewSnapshot([]surface.Command{
		cmdOf("abcd"), {Path: "abcd capture", Group: "records", Block: "people"},
	}, nil)
	writeSurface(r, current)
	r.commit("regenerate the surface snapshot at version 2")

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v, want a version-1 baseline to stay readable", err)
	}
	if got.Status != SurfaceGuardPassed {
		t.Fatalf("Status = %q (reason %q), want %q", got.Status, got.Reason, SurfaceGuardPassed)
	}
}
