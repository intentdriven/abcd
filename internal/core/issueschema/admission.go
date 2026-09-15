package issueschema

// The step-2 admission records (itd-189, spc-67).
//
// Declining a proposal costs nothing epistemically; ADMITTING one is where the
// frame is engaged. So every admission into the widening reading's candidate set
// carries recorded GROUNDS, and every declined proposal carries a disposition —
// because uniform adoption of everything a reading proposes is equally consistent
// with careful judgement and with abdication, and only a record that says which
// happened can tell the two apart.
//
// Three shapes are named by the intent and only ONE of them is new here:
//
//   - The ADMISSION record (adm-N) is new, and this file declares it.
//   - The DECLINED proposal is not a new record type. It is the disposition
//     record of reading.go in its `declined` state, which the widening position
//     already reserves — a second store for a state the disposition vocabulary
//     holds would be a parallel answer to one question.
//   - The SURPRISE entry's join key is reserved on the reading-record envelope
//     (ReservedSurpriseFields, reading.go); what this file adds is its own
//     family, its own store and its own required set, which is what makes it a
//     record rather than a field on something else.
//
// This cycle ships the SCHEMAS, not the commands that write them: no reading has
// run, so there is nothing to write yet. The shapes are wired to the gate that
// reads committed records (core/lint's record_schema) rather than to a verb, so
// a hand-written admission record with a blank `grounds` is a blocker finding
// from the day this lands. What is hand-run is WHO writes the file, never
// whether anything checks it.

// The two families this design adds. Both mint through recordid.Minter.Mint like
// every other record family in this workstream (adr-45): a UTC stamp plus four
// random digits, never content-derived, so two admissions of the same proposal
// stay distinguishable.
const (
	// AdmissionFamily identifies one admission (adm-N): one proposal admitted
	// into the candidate set, with the grounds it was admitted on.
	AdmissionFamily = "adm"
	// SurpriseFamily identifies one surprise entry (srp-N): something the
	// researcher did not expect, recorded as its own act.
	SurpriseFamily = "srp"
)

// The two sibling directories these families occupy under the issue ledger root.
// Like the reading families, they are deliberately NOT in StatusDirs: neither is
// an issue, and every gate scoped to the ledger's status folders must keep
// ignoring them.
const (
	// AdmissionsDir holds one directory per run: admissions/<run-id>/adm-<N>.md.
	// An admission is bucketed by RUN, exactly as the reading store is, because it
	// is meaningful only against the run whose proposals it admits.
	AdmissionsDir = "admissions"
	// SurprisesDir is FLAT: surprises/srp-<N>.md. A surprise is keyed to whatever
	// occasioned it — a detection, an admission, a consequence — and that key is
	// carried in the record's own `occasioned_by`, never in the directory. Keying
	// the directory would tie a surprise to one occasion's family and re-open the
	// collapse this record exists to prevent: it is never a field on a
	// disposition, and it never shares a key with one.
	SurprisesDir = "surprises"
)

// AdmissionRequired is every property an admission record carries. `proposal`
// names the widening item admitted (an rdi-N); `run` is the run whose candidate
// set it joins; `grounds` is free text and non-empty, the analogue of the
// disposition_grounds a rejection already requires.
//
// It is the ONE list: core/lint's adm store declares its requiredFields from
// this value, so the gate and the schema cannot disagree about what a well-formed
// admission carries.
var AdmissionRequired = []string{"schema_version", "id", "run", "proposal", "grounds"}

// AdmissionKnown is the admission's additionalProperties:false allow-list. The
// record has no optional half: an admission is exactly the proposal, the run and
// the grounds, and a key outside that set is a field nothing reads.
var AdmissionKnown = knownSet(AdmissionRequired)

// SurpriseRequired is every property a surprise entry carries. `occasioned_by`
// names whatever occasioned it and is the record's whole join: an rdi-N
// detection, an adm-N admission, or a consequence named in prose. The surprise
// ITSELF is the record's body, where a reader can write more than a frontmatter
// value holds.
var SurpriseRequired = []string{"schema_version", "id", "occasioned_by"}

// SurpriseKnown is the surprise entry's allow-list.
var SurpriseKnown = knownSet(SurpriseRequired)

// knownSet renders a required list as an allow-list. Both families are closed
// records whose allow-list IS their required set, so deriving it is what keeps
// the two from drifting apart in the only way they could.
func knownSet(required []string) map[string]bool {
	known := make(map[string]bool, len(required))
	for _, f := range required {
		known[f] = true
	}
	return known
}
