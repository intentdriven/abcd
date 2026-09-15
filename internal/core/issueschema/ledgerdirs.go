package issueschema

// The ledger's own directory list, and the two shapes the comparative channel
// reads off a committed run (spc-2609020626039834; adr-2609021016272867).
//
// LedgerDirs exists because two instruments have to describe ONE set. The
// cold-reading assembler asserts, at the comparative position, that every ledger
// family except the candidate run stayed behind; the scribe's allow list names
// the families it may transcribe from. Hand-keeping two lists is how a family
// added later ends up excluded by one instrument and admitted by the other, and
// the assembler's assertion would then be an attestation exceeding its
// examination — which brief invariant 16 forbids. So the list is derived from
// the constants this package already declares, and a family is covered the day
// its constant is.

import "sort"

// LedgerDirs is every directory the issue ledger's record families occupy under
// the ledger root, in the order a reader meets them: the three status
// directories, then the sibling families that are deliberately not among them.
//
// It is DERIVED from the constants above it rather than restated, so a family
// this package gains is in the list from the day its constant is declared.
// spc-2609020626048705's reframes directory joins it when that spec lands, by
// declaring its constant and adding it here in the same change.
func LedgerDirs() []string {
	out := make([]string, 0, len(StatusDirs)+4)
	out = append(out, StatusDirs...)
	return append(out, ReadingsDir, DispositionsDir, AdmissionsDir, SurprisesDir)
}

// ReadingsRecordDir is the DURABLE home of a run's own artefacts — the promoted
// manifest, the run metadata and a refusal — as distinct from ReadingsDir, which
// is the working-tier bucket holding the run's reading records.
//
// It lives here rather than in core/reading because two packages name it:
// core/reading writes it, and core/capture reads it to answer "which comparative
// run characterised this widening run" (ComparativeRunFor). `reading` keeps an
// alias so its own callers read unchanged, and the two packages name one
// directory rather than two strings that agree today (spc-2609020626039834).
const ReadingsRecordDir = ".abcd/development/readings"

// RunRecordFileName is the run-metadata file inside a durable run directory. It
// is the COMMIT MARKER: a run without one never happened, and its reading
// records are the next ingest sweep's to roll back.
const RunRecordFileName = "run.json"

// RunHead is the strictly decoded subset of a committed run record that answers
// one question: which widening run did this comparative run characterise?
//
// It is a SUBSET on purpose. The full run record is core/reading's shape and
// carries the instrument, the refusals and the bounds; a reader asking only
// about the candidate join has no business decoding those, and a decoder that
// had to would break every time the record grew a field.
type RunHead struct {
	RunID string `json:"run_id"`
	// Position is the run's reading position, as the run record states it.
	Position string `json:"position"`
	// CandidateRun is the widening run a comparative run characterised, empty at
	// every other position.
	CandidateRun string `json:"candidate_run"`
}

// ItemFate is what the ledger says has happened to one reading item since it was
// returned: the dispositions standing over it and the admissions naming it.
//
// It is the probe the comparative assembly reads before it hands a candidate to
// a reading. The candidate set the design fixes is PRE-ADMISSION — the widening
// reading's returned configurations before anyone answered them — so an item
// carrying either is not a candidate, and the assembly refuses rather than
// handing a reading the researcher's judgement dressed as cold text
// (adr-2609021016272867; companion 8.3).
type ItemFate struct {
	// Dispositions are the standing disposition ids, sorted.
	Dispositions []string
	// Admissions are the ids of the admissions naming this item as their
	// proposal, sorted.
	Admissions []string
	// Cyclic reports a supersession cycle: records are present and none stands,
	// so the item's fate cannot be READ. That is not the same fact as "no
	// disposition", and treating it as one would hand a reading a candidate
	// whose answer nobody can find (the same distinction core/capture's
	// standingDispositions already refuses to collapse).
	Cyclic bool
}

// Free reports whether the item carries no fate at all: no standing
// disposition, no admission, and a disposition set that could be read.
func (f ItemFate) Free() bool {
	return len(f.Dispositions) == 0 && len(f.Admissions) == 0 && !f.Cyclic
}

// Contested reports more than one standing disposition — a ledger fault the
// write path refuses and only a hand edit can produce. A contested item's fate
// is as unreadable as a cyclic one for this purpose: no reader may pick a winner
// from it, so it is not demonstrably uncommitted.
func (f ItemFate) Contested() bool { return len(f.Dispositions) > 1 }

// JudgeItemFate is the JUDGEMENT half, given the disposition records of one item
// and the admission ids naming it. The WALKS live in core/capture, which holds
// the ledger's filesystem; the decision lives here beside
// StandingDispositionIDs for the reason that file states — two readers of one
// question are tolerable, two answers are not.
func JudgeItemFate(records []DispositionRecord, admissions []string) ItemFate {
	standing := StandingDispositionIDs(records)
	fate := ItemFate{Dispositions: standing, Admissions: sortedCopy(admissions)}
	// Records present and nothing standing is a supersession cycle: every answer
	// retired by another, so an item plainly carrying answers reads as carrying
	// none.
	if len(records) > 0 && len(standing) == 0 {
		fate.Cyclic = true
	}
	return fate
}

// sortedCopy returns a sorted copy, so a judgement over a directory listing does
// not depend on the order the filesystem happened to return.
func sortedCopy(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}
