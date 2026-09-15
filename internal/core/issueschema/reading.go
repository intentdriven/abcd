package issueschema

// The reading-record and disposition-record families (itd-180, spc-58).
//
// A capture is something a person noticed. A READING RECORD is something an
// instrument returned under a recorded visible world, and the researcher's
// answer to one is a DISPOSITION — a second record, keyed to the item. The two
// are never one write, so the record can always show that a finding existed
// before it was answered.
//
// Both schemas live here, beside the issue schema, for the reason this package
// exists at all: the writer (core/capture) and the gate that judges the
// committed tree (core/lint) must agree about what a well-formed record carries,
// and two hand-kept copies drift the moment one side gains a field.

import "regexp"

// The three record families this design adds. Each mints through
// recordid.Minter.Mint, which validates any lowercase family and consults no
// maximum (adr-45): an id is a UTC stamp plus four random digits and is never
// content-derived, so two runs returning the SAME tension carry different ids for
// free — a re-raise stays distinguishable from its first appearance, and the
// recurrence signal survives.
const (
	// ReadingRunFamily identifies one reading run (rdg-N): the durable run
	// record and its manifest under .abcd/development/readings/<run-id>/.
	ReadingRunFamily = "rdg"
	// ReadingItemFamily identifies one reading item (rdi-N) — one thing the
	// instrument returned.
	ReadingItemFamily = "rdi"
	// DispositionFamily identifies one disposition (dsp-N) — one answer to one
	// item.
	DispositionFamily = "dsp"
)

// The two sibling directories the working-tier families occupy under the issue
// ledger root. They are deliberately NOT in StatusDirs: a reading item's status
// is the presence of its keyed disposition directory, which is one probe, and
// never a folder-membership question. Every gate scoped to the ledger scopes to
// StatusDirs and therefore ignores both.
const (
	// ReadingsDir holds one directory per run: readings/<run-id>/rdi-<N>.md.
	ReadingsDir = "readings"
	// DispositionsDir is keyed by the ITEM: dispositions/<item-id>/dsp-<N>.md.
	// Keying the directory on the item and the file on the disposition's own id
	// settles two requirements that pull against each other — status is one
	// directory probe, and a disposition still has an id of its own, so the only
	// exit from a `held` state (a superseding disposition that CITES the one it
	// replaces) has something to cite.
	DispositionsDir = "dispositions"
)

// RecordReadLimit is the byte cap on ONE record of these families, and it lives
// here for the same reason the property allow-list does: both readers of the
// family must apply it, and a cap the board applies loosely while the verb
// applies tightly means an oversized disposition stands on the board and is
// refused by the verb — the ledger then says two things about one file. There
// were two caps, eight megabytes apart, before this was one.
//
// A reading record and a disposition are single-screen documents. The cap is not
// there to bound legitimate content; it is there so a device, or a file swapped
// for a huge one, cannot make a read unbounded. The issue family applies the same
// cap for the same reason (GHSA-fh9j-8xmg-m33f): an issue record is a page, and
// the largest one this repository has ever carried is under one sixtieth of it.
const RecordReadLimit = 1 << 20

// ReadingPosition binds one reading position to the supply regime it implies and
// the body its items carry.
//
// Two vocabularies meet here and both are load-bearing, so they are bound in one
// place rather than left to agree by habit: Position is the value the envelope
// carries (the instrument's own name for where it read from), and Regime is the
// supply regime that position was read under, which also names the item body it
// returns. Ruling (17) is what makes the pair necessary — "detection" stays the
// name of the Step-6 instrument and of its REGISTRATIVE body — and a table that
// held only one of the two names would leave the other to drift in prose.
//
// The regime is resolvable by position ALONE, which is the property ruling (4)
// and ruling (18) rest on: it is stated in the reading's definition and no
// operator input can set it. Binding the pair here is what lets the ingest check
// that rather than take a caller's word for it.
type ReadingPosition struct {
	// Position is the envelope's `position` value.
	Position string
	// Regime is the supply regime this position implies, and the name of the
	// item body it returns.
	Regime string
	// Fields are the body properties an item at this position must carry. The
	// pattern named is NOT among them: it is an envelope field, because a
	// universal core condition must not live in a variant part (ruling (18)).
	Fields []string
}

// PositionWidening is the reading position whose items are PROPOSALS: the one
// position where acceptance is admission into a candidate set, and therefore the
// one whose answer set is wider than a disposition (spc-67). It is named because
// two packages now branch on it, and a position spelled as a literal in a branch
// is a position that can drift out of the table below without anything saying so.
const PositionWidening = "widening"

// ReadingPositions is the closed set of positions, each with its body.
//
// itd-180 offers a discriminated union and names a fallback; the FALLBACK is
// what ships. Go has no discriminated union, and this package's whole idiom is
// already "the schema is a value both the writer and the gate read" — four Go
// types would be four places for one schema to drift. So: one record type, an
// untyped body, and the per-position required-field set held here as data.
var ReadingPositions = []ReadingPosition{
	{Position: PositionWidening, Regime: "generative", Fields: []string{"configuration", "what_admits_it"}},
	{Position: "entailment", Regime: "explicative", Fields: []string{"claim_surfaced", "claim_type", "what_implies_it"}},
	{Position: "comparative", Regime: "evaluative", Fields: []string{"candidate_id", "criterion", "characterisation"}},
	{Position: "detection", Regime: "registrative", Fields: []string{"tension", "constraint_in_play", "why_a_tension"}},
}

// Positions is the position values alone, in ReadingPositions order — for a
// membership check and for the message that quotes the legal set.
var Positions = positionValues()

// ReadingBodyFields is the per-position required body set, derived from
// ReadingPositions so the table and the lookup cannot disagree.
var ReadingBodyFields = bodyFieldsByPosition()

// ReadingRequired is the reading record's envelope: every property an item
// carries whatever position returned it. `pattern` is here rather than in a body
// for ruling (18)'s reason — it is a universal core condition.
var ReadingRequired = []string{
	"schema_version", "id", "run", "manifest", "position", "regime", "pattern",
}

// ReadingKnown is the reading record's additionalProperties:false allow-list:
// the envelope, every body field of every position, and the two optional
// properties below. A key outside it is refused, exactly as it is on an issue.
var ReadingKnown = readingKnown()

// ReservedSurpriseFields are reserved and DORMANT (spc-58, out of scope: "the
// surprise entry, reserved here and populated in Iteration 2"). The reading's
// output, the researcher's disposition, and the surprise that occasions
// abduction are three acts and three records; this reserves the third's join
// key in the family now. A populated value is REFUSED until the shape is ruled,
// so the reservation is a behaviour rather than a comment.
var ReservedSurpriseFields = []string{"occasioned_by"}

// DispositionRequired is what every disposition carries whatever its state.
// `disposition_grounds` is NOT here: it is required on every state except
// `held`, which is a per-state rule rather than a schema-wide one.
var DispositionRequired = []string{"schema_version", "id", "item", "state"}

// DispositionKnown is the disposition's allow-list.
var DispositionKnown = dispositionKnown()

// The four disposition states that ship. itd-180 and ruling (19) both say FIVE
// states and both enumerate four; the schema is data, so a fifth is one line the
// day it is named, and naming it is a vocabulary judgement belonging to the
// researcher rather than to this build. The discrepancy is carried here, not
// resolved.
//
// Nothing meaning "already covered" exists at any position: an undispositioned
// item is REPORTED as outstanding, never named as a state.
const (
	// DispositionAccepted is available at every position. At the widening
	// position acceptance IS admission — a state encoding the position would
	// duplicate the envelope.
	DispositionAccepted = "accepted"
	// DispositionRejected asserts a purpose the closing run tests, so it is
	// never available at the widening position.
	DispositionRejected = "rejected"
	// DispositionDeclined is the widening position's only: the proposal was
	// admissible and the researcher chose otherwise, asserting nothing testable.
	// Forcing that into `rejected` would manufacture a principle never at stake.
	DispositionDeclined = "declined"
	// DispositionHeld is directional, and requires an epistemic exit condition.
	// A hold exits only through a superseding disposition that cites it — never
	// by expiry, and never silently.
	DispositionHeld = "held"
)

// DispositionStates is the shipped state vocabulary.
var DispositionStates = []string{
	DispositionAccepted, DispositionRejected, DispositionDeclined, DispositionHeld,
}

// DispositionAvailability is the per-position availability of each state: the
// coupling ruling (19) put on the schema, which the disposition validates
// against by reading `position` off the keyed reading record.
//
// A pair the design has NOT ruled is ABSENT from the inner map, and absence is
// three-valued on purpose: `held` at the widening position was deferred by the
// facilitator on 2026-08-30 with a revisit point at the first widening run's
// dispositions, so that one row ships unfilled and its refusal is not armed. An
// unfilled pair is permitted quietly rather than refused, because refusing a
// combination nobody has ruled would decide the deferral by implementation.
var DispositionAvailability = map[string]map[string]bool{
	"widening": {
		DispositionAccepted: true,
		DispositionRejected: false,
		DispositionDeclined: true,
		// DispositionHeld: deferred, deliberately absent.
	},
	"entailment": {
		DispositionAccepted: true,
		DispositionRejected: true,
		DispositionDeclined: false,
		DispositionHeld:     true,
	},
	"comparative": {
		DispositionAccepted: true,
		DispositionRejected: true,
		DispositionDeclined: false,
		DispositionHeld:     true,
	},
	"detection": {
		DispositionAccepted: true,
		DispositionRejected: true,
		DispositionDeclined: false,
		DispositionHeld:     true,
	},
}

// DispositionStateAvailable reports whether state is available at position, and
// whether that pair has been RULED at all. A caller refuses only on
// (available=false, ruled=true): an unruled pair is the deferral above, and
// deciding it here would settle by implementation what the facilitator deferred.
func DispositionStateAvailable(position, state string) (available, ruled bool) {
	row, ok := DispositionAvailability[position]
	if !ok {
		return false, false
	}
	available, ruled = row[state]
	return available, ruled
}

// dispositionIDRe is the disposition id grammar, used by the standing-answer
// reader below and by every writer that mints one.
var dispositionIDRe = regexp.MustCompile(`^` + DispositionFamily + `-[0-9]+$`)

// ReservedHoldFields are the two-axis hold field: reserved, grammar stated, and
// DORMANT. frame-location is free text naming the frame element; MoSCoW is one
// of HoldMoscowValues. Reserving costs nothing and retrofitting is expensive, so
// the fields are in the schema now — but a populated value is refused until
// activation is ruled, which makes the reservation a behaviour rather than a
// comment.
var ReservedHoldFields = []string{"hold_frame_location", "hold_moscow"}

// HoldMoscowValues is the stated (dormant) grammar of the MoSCoW axis.
var HoldMoscowValues = []string{"must", "should", "could", "wont"}

// ReadingRegime is the supply regime a position implies ("registrative" for the
// detection position, and so on) — also the design record's name for the body
// that position returns, which is why a refusal quotes it: a message then names
// the shape the reader will look up rather than only the position that produced
// it. An unknown position yields "", which the caller reports as an unknown
// position rather than as a regime mismatch.
func ReadingRegime(position string) string {
	for _, p := range ReadingPositions {
		if p.Position == position {
			return p.Regime
		}
	}
	return ""
}

func positionValues() []string {
	out := make([]string, 0, len(ReadingPositions))
	for _, p := range ReadingPositions {
		out = append(out, p.Position)
	}
	return out
}

func bodyFieldsByPosition() map[string][]string {
	out := make(map[string][]string, len(ReadingPositions))
	for _, p := range ReadingPositions {
		out[p.Position] = p.Fields
	}
	return out
}

func readingKnown() map[string]bool {
	known := map[string]bool{
		// promoted_to is the forward half of the routing join: an accepted item's
		// action is a SEPARATE admission, and the item id stamped forward here (with
		// promoted_from in the draft) is what joins the two.
		"promoted_to": true,
	}
	for _, k := range ReadingRequired {
		known[k] = true
	}
	for _, p := range ReadingPositions {
		for _, f := range p.Fields {
			known[f] = true
		}
	}
	for _, f := range ReservedSurpriseFields {
		known[f] = true
	}
	return known
}

func dispositionKnown() map[string]bool {
	known := map[string]bool{
		// disposition_grounds is free text, required on every state except held;
		// WHAT it must contain varies by state, enforced by lint rather than by
		// four fields.
		"disposition_grounds": true,
		// exit_condition is required on held and meaningless elsewhere.
		"exit_condition": true,
		// supersedes_disposition names the disposition this one replaces — the
		// only exit from a hold, and what makes the standing disposition of an
		// item the one no sibling supersedes.
		"supersedes_disposition": true,
		// recurs cites prior item ids. It is the recorded form of the researcher's
		// WARM recognition of a persistence: no mechanical join produces it, it
		// lives entirely on the ledger side, and it is never a fifth state.
		"recurs": true,
	}
	for _, k := range DispositionRequired {
		known[k] = true
	}
	for _, f := range ReservedHoldFields {
		known[f] = true
	}
	return known
}
