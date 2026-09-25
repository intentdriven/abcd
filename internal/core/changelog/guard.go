package changelog

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// SurfaceGuardStatus is the guardrail's verdict on a cut. Three states, kept
// apart because a caller that conflated them would ship the wrong release:
//
//	SurfaceGuardPassed  — the cut may proceed.
//	SurfaceGuardFailed  — the surface narrowed and nothing in the cut declares
//	                      it; Reason names every changed surface.
//	SurfaceGuardRefused — the guardrail could not compare at all; Reason says
//	                      what to do about it.
//
// A refusal is NOT a pass. Failing to distinguish them is the whole risk of the
// first cut, which is the one with no baseline to compare against.
type SurfaceGuardStatus string

// The three verdicts. The string values are what a rendered preview prints and
// what a machine-readable front door emits, so renaming a constant is safe and
// changing a value is a contract change.
const (
	SurfaceGuardPassed  SurfaceGuardStatus = "passed"
	SurfaceGuardFailed  SurfaceGuardStatus = "failed"
	SurfaceGuardRefused SurfaceGuardStatus = "refused"
)

// SurfaceGuard is the outcome of the surface-break guardrail at a release cut.
//
// Like Derivation, a refusal is modelled as a VALUE rather than an error: "this
// cut cannot be guarded" is a legitimate result a read-only preview must render,
// and errors are reserved for "the repository could not be read at all".
type SurfaceGuard struct {
	// BaseTag is the release tag the baseline snapshot was read from, empty
	// when no tag resolved.
	BaseTag string `json:"base_tag"`
	// Status is the verdict.
	Status SurfaceGuardStatus `json:"status"`
	// Breaks is every structural narrowing found, in canonical order. It is
	// populated on a PASS too when a break was declared: the release notes have
	// to describe what broke, so a declared break is reported, not discarded.
	Breaks []surface.Break `json:"breaks,omitempty"`
	// BreakingDeclared reports whether the cut ADDS a record whose impact is
	// breaking — the author's declaration that this release narrows the surface
	// on purpose. See RecordSet.DeclaresBreak for why the removed side does not
	// count.
	BreakingDeclared bool `json:"breaking_declared"`
	// Reason names what to fix; empty on a clean pass.
	Reason string `json:"reason,omitempty"`
}

// GuardSurface runs the release surface guardrail over the repository at root
// and writes nothing (spc-10 AC 4, plan outcome 5).
//
// The comparison is the BASELINE AS OF THE LAST RELEASE TAG against the current
// surface the caller hands in. Reading the baseline out of the tag is the single
// load-bearing decision in this function, and the obvious "simplification" is
// what would destroy it: the committed snapshot is kept byte-equal to the live
// command tree by a drift test, so comparing the committed FILE against the
// current tree would compare a file with itself, never report a break, and leave
// a green light where a guardrail was meant to be.
//
// current is passed in rather than built here because building it means walking
// the live cobra command tree, and internal/core must not know cobra. The front
// door (internal/surface/cli) owns the walk; this owns the judgement.
//
// The cut's own records answer "was this break declared?". Both the record set
// and the impact ordering are phase 1's (ShippedSince and RecordSet.DeclaresBreak),
// so the guardrail and the version derivation can never disagree about what
// shipped or about what breaking means. A break passes when — and only when — the
// cut ADDS a record whose impact is breaking; a superseded record leaving the
// terminal folder declares nothing about this release.
//
// It refuses, rather than passing, in three fail-closed cases:
//
//   - No release tag. There is no immutable anchor to read a baseline from.
//   - The tag's tree carries no snapshot. This is the repository's real state
//     until a release is cut that contains the baseline, and it is deliberate:
//     the first cut is the highest-risk one and must not sail through unguarded.
//     The refusal text names the clean-cutover manual roll, which is what puts a
//     baseline into a tag.
//   - current disagrees with the snapshot committed at HEAD. See
//     currentMatchesHead: the caller walks a compiled-in command tree, and a
//     binary older than the tree being released reproduces the LAST release's
//     surface, which would diff the baseline against itself.
//
// A snapshot that is present at the tag but cannot be decoded is an ERROR, not a
// refusal: a corrupt baseline read as "no surface" would make every command in
// the release look like surface that never existed.
func GuardSurface(root string, current surface.Snapshot) (SurfaceGuard, error) {
	var g SurfaceGuard

	base, hasTag, err := LatestReleaseTag(root)
	if err != nil {
		return SurfaceGuard{}, err
	}
	if !hasTag {
		return refuseGuard(g, "no release tag found — the surface baseline is read from the last release tag, "+
			"and without one no break can be detected (tag the current release first)"), nil
	}
	g.BaseTag = base.Tag()

	baseline, hasBaseline, err := surfaceSnapshotAt(root, g.BaseTag)
	if err != nil {
		return SurfaceGuard{}, err
	}
	if !hasBaseline {
		return refuseGuard(g, noBaselineReason(g.BaseTag)), nil
	}

	matches, reason, err := currentMatchesHead(root, current)
	if err != nil {
		return SurfaceGuard{}, err
	}
	if !matches {
		return refuseGuard(g, reason), nil
	}

	records, err := ShippedSince(root, g.BaseTag)
	if err != nil {
		return SurfaceGuard{}, err
	}
	g.BreakingDeclared = records.DeclaresBreak()
	g.Breaks = surface.Diff(baseline, current)

	if len(g.Breaks) == 0 || g.BreakingDeclared {
		g.Status = SurfaceGuardPassed
		return g, nil
	}
	g.Status = SurfaceGuardFailed
	g.Reason = failureReason(g.BaseTag, g.Breaks)
	return g, nil
}

// failureReason names every changed surface, because "a break was detected"
// sends the operator hunting through a 500-line snapshot for it. Each break is
// listed on its own line so a long list stays readable in a terminal, and the
// remedy states both ways out: declare it, or restore the surface.
func failureReason(baseTag string, breaks []surface.Break) string {
	lines := make([]string, 0, len(breaks))
	for _, b := range breaks {
		lines = append(lines, "  - "+b.String())
	}
	return fmt.Sprintf("the public surface narrowed since %s and no record in the cut declares it:\n%s\n"+
		"either ship a record with `impact: breaking` in this cut, or restore the surface", baseTag, strings.Join(lines, "\n"))
}

// currentMatchesHead checks the caller's current surface against the snapshot
// committed at HEAD, and is what closes the stale-binary hole.
//
// current is walked from a COMPILED-IN command tree: the front door reflects over
// the cobra tree of the process that is running. An installed release binary
// therefore reports the surface of the release it was built from, not of the tree
// being released — so a command deleted since that build is missing from the
// baseline's counterpart as well, the diff comes back empty, and a real break
// ships labelled additive. Nothing else in the guardrail can see this: every
// other input (the baseline, the records) is read from git and would look
// perfectly consistent.
//
// HEAD's committed snapshot is the tree's own statement of its surface, kept
// honest by the drift test, so equality with it is exactly the proof that the
// walking binary IS the tree being released. A mismatch is a refusal rather than
// a failure: nothing has been shown about compatibility, only that the guardrail
// cannot trust its own input.
//
// HEAD is read rather than the working tree for the reason ShippedSince reads it:
// a release is cut from a commit, and a dirty or half-staged tree must not be
// able to change what a release gate concludes.
func currentMatchesHead(root string, current surface.Snapshot) (bool, string, error) {
	head, found, err := surfaceSnapshotAt(root, "HEAD")
	if err != nil {
		return false, "", err
	}
	if !found {
		return false, fmt.Sprintf("no surface snapshot committed at HEAD — %s is absent from the tree being "+
			"released, so the surface this binary reports cannot be checked against it (%s)",
			surface.SnapshotPath, regenerateRemedy), nil
	}

	wantBytes, err := surface.Encode(head)
	if err != nil {
		return false, "", err
	}
	gotBytes, err := surface.Encode(current)
	if err != nil {
		return false, "", err
	}
	if !bytes.Equal(wantBytes, gotBytes) {
		reason := fmt.Sprintf("the surface this binary reports does not match %s at HEAD, so the guardrail "+
			"would compare the wrong tree and cannot answer whether the release breaks anything. Either the "+
			"binary predates the tree being released — rebuild it from this tree — or the snapshot is stale (%s).",
			surface.SnapshotPath, regenerateRemedy)
		// A regroup changes no invocation, so nothing but this comparison sees
		// it; naming each moved verb is what makes the refusal actionable
		// (itd-146 criterion 4).
		if moved := surface.PlacementChanges(head, current); len(moved) > 0 {
			reason += "\nhelp placement differs from HEAD's snapshot:\n  - " + strings.Join(moved, "\n  - ")
		}
		return false, reason, nil
	}
	return true, "", nil
}

// regenerateRemedy names the generator once, so both halves of the stale-surface
// refusal tell the operator the same thing to run.
const regenerateRemedy = "regenerate and commit it with `go generate ./internal/surface/cli`"

// noBaselineReason is the fail-closed refusal for a tag with no snapshot in its
// tree. It says what the state IS and what ends it, because this is not a
// transient error an operator can retry away: it holds for every cut until a
// release ships that carries the baseline, and an operator who reads it as a bug
// will go looking for one that is not there.
func noBaselineReason(baseTag string) string {
	return fmt.Sprintf("no surface baseline in %s — %s is absent from that release's tree, so a break cannot be "+
		"detected and the cut refuses rather than passing unguarded. The baseline was seeded after %s, so every cut "+
		"refuses until a release is cut that CONTAINS it; the clean-cutover manual roll is what puts it into a tag.",
		baseTag, surface.SnapshotPath, baseTag)
}

// refuseGuard stamps a refusal, clearing anything that could read as a verdict
// on the surface. Whatever was resolved before the refusal (the anchor tag) is
// kept, because it is what the operator needs to act on the message.
func refuseGuard(g SurfaceGuard, reason string) SurfaceGuard {
	g.Status = SurfaceGuardRefused
	g.Reason = reason
	g.Breaks = nil
	g.BreakingDeclared = false
	return g
}

// maxSnapshotBytes caps the guarded blob read, in the same order and for the
// same reason as maxRecordBytes: the snapshot is a few hundred lines of JSON, so
// a blob that is not one must not stream unbounded input into a release gate.
const maxSnapshotBytes = 4 << 20

// surfaceSnapshotAt reads the committed surface snapshot out of ref's tree.
//
// Presence is probed with ls-tree rather than inferred from `cat-file` failing,
// because that command fails identically for "the path is not in this tree" and
// for "this ref does not exist" — and collapsing those would turn an unreadable
// repository into a silent "no baseline" refusal. A missing path returns
// found=false with no error, which the caller turns into the fail-closed
// refusal; anything git could not answer at all is returned as an error.
func surfaceSnapshotAt(root string, ref string) (surface.Snapshot, bool, error) {
	listed, err := gitutil.Run(root, "ls-tree", "-z", "--name-only", ref, "--", surface.SnapshotPath)
	if err != nil {
		return surface.Snapshot{}, false, fmt.Errorf("looking for %s at %s: %w", surface.SnapshotPath, ref, err)
	}
	if strings.Trim(listed, "\x00") == "" {
		return surface.Snapshot{}, false, nil
	}

	blob, err := gitutil.RunLimited(root, maxSnapshotBytes, "cat-file", "blob", ref+":"+surface.SnapshotPath)
	if err != nil {
		return surface.Snapshot{}, false, fmt.Errorf("reading %s at %s: %w", surface.SnapshotPath, ref, err)
	}
	snap, err := surface.Decode([]byte(blob))
	if err != nil {
		return surface.Snapshot{}, false, fmt.Errorf("decoding %s at %s: %w", surface.SnapshotPath, ref, err)
	}
	return snap, true, nil
}
