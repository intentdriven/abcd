// Package release is the transport-agnostic composition of a release cut: the
// deterministic half of `abcd launch ship` and the whole of `abcd changelog`.
//
// It is a composition, not a new domain. The version arithmetic, the record
// set-difference, the tag anchor and the surface guardrail all live in
// internal/core/changelog; the record-tree traversal lives in internal/core/lint.
// What is here is the ORDER those parts run in for a release cut, the refusals
// that stop one, and the one result value a host needs to compose the changelog
// prose. It sits in its own package because it reads both of those packages and
// changelog cannot import lint (lint already imports changelog for the impact
// enum).
//
// Nothing here writes: the emit step is the input to a review, and the CHANGELOG
// heading is written only by the ingest step that follows the composer.
package release

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/core/surface"
)

// The record trees a cut reasons about, repo-relative. The terminal folders the
// set-difference reads are internal/core/changelog's; these two are the
// lifecycle trees the stale-intent refusal reads, and they are named here
// because this is the only consumer of them outside the lint's own config.
const (
	intentsDir = ".abcd/development/intents"
	specsDir   = ".abcd/development/specs"
)

// Entry is one record in the cut, carrying exactly what a changelog composer
// needs to write a line about it and nothing more: what it is, where it lives,
// what shipping it did, and the author's own words to draw on.
type Entry struct {
	// ID is the record id (itd-N, iss-N) — the token every generated changelog
	// line must cite, and the key the completeness bijection is computed over.
	ID string `json:"id"`
	// Path is the record's repo-relative path, so a reviewer can open it.
	Path string `json:"path"`
	// Impact is the record's declared product judgement.
	Impact changelog.Impact `json:"impact"`
	// Title names the record; never empty.
	Title string `json:"title"`
	// Summary is the record's opening paragraph — source material, not the line.
	Summary string `json:"summary"`
	// InChangelog reports whether this record must be cited in the prose. It is
	// carried per entry rather than left for the host to re-derive from Impact,
	// because the bijection and the version must agree on one definition of
	// "user-facing" and the host is not the place to keep that rule. It is
	// changelog.Record.InChangelog verbatim, which is why a record with no valid
	// impact reads true here: unknown is not internal.
	InChangelog bool `json:"in_changelog"`
	// ShippedInErr is why this record's `shipped_in:` did not parse, empty when
	// the field is absent, null, or valid. A record carrying one is IN the cut —
	// an unreadable value never removes a record — so this is the only place an
	// operator learns that a record tried to say it shipped elsewhere and failed.
	//
	// Reported rather than refused. Refusing would block a release on a typo in a
	// field whose whole purpose is to take records OUT, and the fail-safe
	// direction already holds: the record stays, so the worst outcome is a
	// redundant changelog line rather than a missing one.
	ShippedInErr string `json:"shipped_in_err,omitempty"`
}

// RefusalKind classifies why a cut cannot proceed. The string values are the
// wire format a front door emits, so renaming a constant is safe and changing a
// value is a contract change.
type RefusalKind string

// The refusal kinds. Every one of them is fail-closed: the cut stops rather than
// deriving a number or a changelog that would be wrong.
const (
	// RefusalNoReleaseTag: no immutable base to measure the cut from.
	RefusalNoReleaseTag RefusalKind = "no-release-tag"
	// RefusalReleaseInFlight: the newest CHANGELOG heading is ahead of the
	// newest tag, so a release sits between its merge and its tag.
	RefusalReleaseInFlight RefusalKind = "release-in-flight"
	// RefusalUnlabelled: a record added by the cut carries no valid impact.
	RefusalUnlabelled RefusalKind = "unlabelled-record"
	// RefusalStaleIntent: an intent in planned/ has no open spec left — every
	// spec realising it has closed and the record never moved.
	RefusalStaleIntent RefusalKind = "stale-intent"
	// RefusalSurfaceGuard: the surface guardrail failed or could not compare.
	RefusalSurfaceGuard RefusalKind = "surface-guard"
	// RefusalUnfixedFinding: a consequential finding this cycle captured is
	// still open, with no recorded decision to defer it.
	RefusalUnfixedFinding RefusalKind = "unfixed-finding"
	// RefusalDeletedFinding: a consequential record the anchor held in open/ has
	// been removed from the ledger rather than answered. It is a kind of its own
	// rather than a shape of unfixed-finding because the remedy differs — the
	// record has to come back before it can be resolved, waived or wontfixed —
	// and because a front door acting on the refusal cannot open a file that is
	// no longer there.
	RefusalDeletedFinding RefusalKind = "deleted-finding"
	// RefusalEmptyCut: nothing user-facing shipped, so there is no release.
	RefusalEmptyCut RefusalKind = "empty-cut"
)

// Refusal is one reason a cut cannot proceed. It always names something
// specific — a record, a version, a command — because "refused" on its own tells
// an operator nothing about what to fix.
type Refusal struct {
	Kind   RefusalKind `json:"kind"`
	Reason string      `json:"reason"`
	// Records lists the blocking record ids when the refusal is about records,
	// so a front door can act on the refusal without parsing its prose.
	Records []string `json:"records,omitempty"`
}

// Cut is the emit step's result: the deterministic release cut.
//
// It is the input to the changelog composer and, after it, to the ingest step
// that writes the dated heading — so its shape is a contract between three
// stages. A refusal is carried as a VALUE, in the same shape as
// changelog.Derivation and launch.RetentionPlan: "this cut cannot proceed" is a
// legitimate result a read-only preview must render, and errors are reserved for
// "the repository could not be read at all".
type Cut struct {
	// Ready reports that the cut may proceed to the composer. It is exactly
	// "no refusals", so a caller can never read a ready cut off a refused one.
	Ready bool `json:"ready"`
	// BaseTag is the release tag the cut is measured from, "" when none resolved.
	BaseTag string `json:"base_tag"`
	// NextTag is the derived version as a git tag, "" when nothing is released.
	NextTag string `json:"next_tag"`
	// Bumped reports whether the cut moves the version at all.
	Bumped bool `json:"bumped"`
	// Impact is the strongest impact in the cut — the judgement that decided
	// the version.
	Impact changelog.Impact `json:"impact"`
	// DecidedBy names the records carrying that deciding impact, so the version
	// can be traced to the record that caused it rather than asserted.
	DecidedBy []string `json:"decided_by"`
	// Added and Removed are the cut, split by direction. A record that LEFT a
	// terminal folder is a user-visible change too, so it travels rather than
	// being dropped.
	Added   []Entry `json:"added"`
	Removed []Entry `json:"removed"`
	// Guard is the surface-break guardrail's verdict on this cut.
	Guard changelog.SurfaceGuard `json:"guard"`
	// Findings is the unfixed-findings guardrail's verdict: what this cycle
	// captured and has not answered. It travels on a PASSING cut too, because a
	// waived finding is only consciously deferred if the release report says
	// what was deferred and why.
	Findings changelog.FindingGuard `json:"findings"`
	// Refusals is every reason the cut cannot proceed, in the order they are
	// checked. All of them are reported, not just the first: an operator fixing
	// a release should see the whole list in one pass.
	Refusals []Refusal `json:"refusals,omitempty"`
}

// Emit computes the deterministic release cut for the repository at root and
// writes nothing.
//
// current is the caller's view of the public command surface. It is passed in
// rather than built here for the reason changelog.GuardSurface states: building
// it means walking the cobra tree, which internal/core must not do. The front
// door owns the walk; this owns the judgement.
//
// The order is deliberate. The derivation runs first, and if it refuses this
// returns THAT refusal alone: its three refusals (no tag, release in flight, an
// unlabelled record) all mean the anchor or the record set cannot be trusted, and
// every later check reads the same inputs — so continuing would restate one fault
// as three and send the operator hunting for bugs that are not there. Once the
// derivation holds, the remaining checks all run and accumulate, because they are
// independent and an operator fixing a release should see the whole list at once.
func Emit(root string, current surface.Snapshot) (Cut, error) {
	derivation, err := changelog.Derive(root)
	if err != nil {
		return Cut{}, err
	}

	cut := Cut{
		BaseTag:   derivation.BaseTag,
		NextTag:   derivation.NextTag,
		Bumped:    derivation.Bumped,
		Impact:    derivation.Bump,
		DecidedBy: decidedBy(derivation),
		Added:     entriesOf(derivation.Records.Added),
		Removed:   entriesOf(derivation.Records.Removed),
	}
	if derivation.Refused {
		cut.Refusals = []Refusal{derivationRefusal(derivation)}
		return sealed(cut), nil
	}

	guard, err := changelog.GuardSurface(root, current)
	if err != nil {
		return Cut{}, err
	}
	cut.Guard = guard

	stale, err := staleIntents(root)
	if err != nil {
		return Cut{}, err
	}
	if len(stale) > 0 {
		cut.Refusals = append(cut.Refusals, staleRefusal(stale))
	}
	if guard.Status != changelog.SurfaceGuardPassed {
		cut.Refusals = append(cut.Refusals, Refusal{
			Kind:   RefusalSurfaceGuard,
			Reason: guard.Reason,
		})
	}

	findings, err := changelog.GuardFindings(root, derivation.BaseTag)
	if err != nil {
		return Cut{}, err
	}
	cut.Findings = findings
	if len(findings.Unfixed) > 0 {
		cut.Refusals = append(cut.Refusals, unfixedRefusal(findings))
	}
	if len(findings.Deleted) > 0 {
		cut.Refusals = append(cut.Refusals, deletedRefusal(findings))
	}
	// The guard's own verdict is the backstop: a failure it reports through
	// neither list would otherwise pass silently, which is the fail-open shape
	// this whole gate exists to close.
	if findings.Status != changelog.FindingGuardPassed && len(findings.Unfixed) == 0 && len(findings.Deleted) == 0 {
		cut.Refusals = append(cut.Refusals, Refusal{Kind: RefusalUnfixedFinding, Reason: findings.Reason})
	}
	if !derivation.Bumped {
		cut.Refusals = append(cut.Refusals, Refusal{
			Kind: RefusalEmptyCut,
			Reason: fmt.Sprintf("nothing user-facing shipped since %s — every record in the cut is internal "+
				"(or the cut is empty), so there is no version to derive and no release to make", cut.BaseTag),
		})
	}
	return sealed(cut), nil
}

// sealed finalises a cut: Ready is exactly "no refusals", and a refused cut
// carries NO derived version.
//
// Clearing the version is the fail-closed half, and it matters most in the case
// that looks harmless. A stale-intent or guard refusal leaves a perfectly
// well-formed number behind, and a number rendered next to a refusal is the one
// an operator copies into a heading by hand — which is exactly the manual step
// derived releases abolish. The impact and the deciding records are kept: they
// describe the records, which is still true, and they are what the operator needs
// to see to fix the cut.
func sealed(cut Cut) Cut {
	cut.Ready = len(cut.Refusals) == 0
	if !cut.Ready {
		cut.NextTag = ""
		cut.Bumped = false
	}
	return cut
}

// derivationRefusal translates the derivation's own refusal into this package's
// vocabulary. The kinds are mapped explicitly rather than passed through, so the
// two enums can evolve apart and an unmapped kind is a visible fault rather than
// a silent blank.
func derivationRefusal(d changelog.Derivation) Refusal {
	ref := Refusal{Reason: d.RefusalReason}
	switch d.RefusalKind {
	case changelog.RefusalNoReleaseTag:
		ref.Kind = RefusalNoReleaseTag
	case changelog.RefusalReleaseInFlight:
		ref.Kind = RefusalReleaseInFlight
	case changelog.RefusalUnlabelledRecord:
		ref.Kind = RefusalUnlabelled
		for _, rec := range d.Records.UnlabelledAdded() {
			ref.Records = append(ref.Records, rec.ID)
		}
	default:
		ref.Kind = RefusalKind(d.RefusalKind)
	}
	return ref
}

// unfixedRefusal carries the guardrail's verdict into the cut's vocabulary,
// naming the blocking records in Records so a front door can act on the refusal
// without parsing its prose — the same shape the unlabelled-record refusal takes.
func unfixedRefusal(g changelog.FindingGuard) Refusal {
	ref := Refusal{Kind: RefusalUnfixedFinding, Reason: g.UnfixedReason()}
	for _, f := range g.Unfixed {
		ref.Records = append(ref.Records, f.ID)
	}
	return ref
}

// deletedRefusal is unfixedRefusal's twin for the records the cut removed from
// the ledger instead of answering (iss-2609091143455568). It names them in
// Records for the same reason — a front door acts on the ids, not on the prose —
// and the id is all it can name: the path it carries is where the record USED to
// be, at the anchor.
func deletedRefusal(g changelog.FindingGuard) Refusal {
	ref := Refusal{Kind: RefusalDeletedFinding, Reason: g.DeletedReason()}
	for _, f := range g.Deleted {
		ref.Records = append(ref.Records, f.ID)
	}
	return ref
}

// staleIntent is one intent whose record contradicts its specs' lifecycle: it
// sits in planned/ while every spec realising it has closed.
type staleIntent struct {
	intentID string
	path     string
	// specIDs are the closed specs that realise it — all of them, because under
	// the 1:n rule the last close is the one that should have moved the record,
	// and naming only the first would hide which deliveries are already in.
	specIDs []string
}

// staleIntents is outcome 11's fail-closed check: every intent still sitting in
// planned/ that has no OPEN spec left.
//
// Why it exists: derivation reads shipped/, but a feature's code merges before
// its intent record moves. An intent left in planned/ after its last spec closed
// is invisible to the tree-diff, so the cut silently UNDER-BUMPS — it ships a
// user-facing feature and derives a version that says nothing shipped. The last
// close is what auto-moves the intent to shipped/ (itd-80, as amended by
// adr-2609151513118583), so the mismatch is a record defect the operator can fix
// in one move, and the ship refuses until they do.
//
// The question is "any open spec left?", never "has its spec closed?". An intent
// owns one or more specs (invariant 17), so a planned intent with one closed
// spec and one open spec is the correct steady state of a partial delivery;
// asking the old question would refuse every release taken while one was in
// flight. An intent no spec claims at all is not stale either — it is
// unscheduled, and there is nothing whose closing should have moved it.
//
// It reads the WORKING TREE, not the tagged history, deliberately: the question
// is "what must you fix before cutting?", and the answer has to be about the
// files the operator can edit right now.
func staleIntents(root string) ([]staleIntent, error) {
	idx, err := lint.ScanSpecLinks(root, intentsDir, specsDir, lint.Config{})
	if err != nil {
		return nil, err
	}
	var out []staleIntent
	for _, intent := range idx.Intents {
		if intent.Bucket != "planned" {
			continue
		}
		// The specs realising this intent: its own back-linked set, plus the spec
		// its spec_id names when that resolves — a one-sided link (the spec names
		// another intent, or none) is still a closed spec this record was waiting
		// on, and dropping it would fail the check open.
		realising := idx.SpecsForIntent(intent.ID)
		closedIDs := make([]string, 0, len(realising)+1)
		anyOpen := false
		for _, s := range realising {
			if s.Bucket == "closed" {
				closedIDs = append(closedIDs, s.ID)
				continue
			}
			anyOpen = true
		}
		if bucket, found := idx.SpecBucket(intent.SpecID); found && !containsSpecNum(realising, intent.SpecID) {
			if bucket == "closed" {
				closedIDs = append(closedIDs, intent.SpecID)
			} else {
				anyOpen = true
			}
		}
		if anyOpen || len(closedIDs) == 0 {
			continue
		}
		out = append(out, staleIntent{intentID: intent.ID, path: filepath.ToSlash(intent.Path), specIDs: closedIDs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out, nil
}

// containsSpecNum reports whether the set already holds the spec the given
// reference names, through the record's one canonical spec comparison
// (spec.SameNum: spc-9, spc-9-thing and spc-009 are one spec).
func containsSpecNum(specs []lint.SpecLink, ref string) bool {
	for _, s := range specs {
		if spec.SameNum(s.ID, ref) {
			return true
		}
	}
	return false
}

// staleRefusal names every blocking intent and the move that clears it. The
// remedy is spelled out because the operator's instinct — edit the version by
// hand — is exactly the thing derived releases exist to abolish.
func staleRefusal(stale []staleIntent) Refusal {
	lines := make([]string, 0, len(stale))
	ids := make([]string, 0, len(stale))
	for _, s := range stale {
		noun := "its spec %s has closed"
		if len(s.specIDs) > 1 {
			noun = "its specs %s have all closed"
		}
		lines = append(lines, fmt.Sprintf("  - %s (%s) — "+noun, s.intentID, s.path, strings.Join(s.specIDs, ", ")))
		ids = append(ids, s.intentID)
	}
	return Refusal{
		Kind:    RefusalStaleIntent,
		Records: ids,
		Reason: "an intent whose every spec has closed is still in planned/, so its feature is invisible to the cut " +
			"and the release would under-bump:\n" + strings.Join(lines, "\n") +
			"\nmove each record to shipped/ (its last spec closing is what moves it) and cut again",
	}
}

// decidedBy names the records carrying the cut's deciding impact — the evidence
// for the derived version. Every record at the maximum is listed, not just the
// first: they are jointly the reason, and naming one would imply the others did
// not matter.
func decidedBy(d changelog.Derivation) []string {
	if !d.Bumped {
		return nil
	}
	var out []string
	for _, rec := range d.Records.All() {
		if rec.Impact == d.Bump {
			out = append(out, rec.ID)
		}
	}
	return out
}

// entriesOf projects records into the composer's view of them.
func entriesOf(records []changelog.Record) []Entry {
	out := make([]Entry, 0, len(records))
	for _, rec := range records {
		out = append(out, Entry{
			ID:           rec.ID,
			Path:         rec.Path,
			Impact:       rec.Impact,
			Title:        rec.Title,
			Summary:      rec.Summary,
			InChangelog:  rec.InChangelog(),
			ShippedInErr: rec.ShippedInErr,
		})
	}
	return out
}
