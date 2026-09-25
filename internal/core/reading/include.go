// Package reading assembles the input a cold reading is handed.
//
// Blindness is a property of the input, not a promise the reader makes. The
// include table below is the whole of what a reading may see: what it does not
// name is absent, including a record family invented after the table was
// written, and including this instrument's own output (itd-183, spc-61).
//
// The package is cobra-free and stdout-free like every sibling under
// internal/core (adr-23): it takes a structured request and returns a
// structured result, and the front doors format it.
package reading

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// AssemblerVersionCore is the hand-set semver of the assembly contract: the
// include table, the projection, the bundle shape and the manifest shape
// together. It moves when the contract moves, which a digest cannot detect —
// a rewritten rule text moves the rendering without changing what the
// assembler promises, and a projection change alters the promise without
// touching the table (spc-61, ruling (12); spc-68).
// It goes 1.2.0 to 1.3.0 with the local ledger tier's entry in the exclusion
// floor: the floor is asserted into every manifest and enforced by
// assertExclusions, so an added entry is an added PROMISE — a refusal a reader
// can now check and previously could not — rather than a rewording of an
// existing one (iss-2608311238236490).
// It goes 1.3.0 to 1.4.0 with the withdrawal of the scope operand: the bundle's
// `scope` block becomes `preset` and the manifest's `scope`, `scope_hash` and
// `scope_overridden` become `preset` and `preset_hash`, so the shape a reader
// and an auditor are promised has moved and nothing about the previous shape
// remains true (adr-2609021016286571, spc-2609021004075744).
// It goes 1.4.0 to 1.5.0 with itd-194: Table gains a Scan declaration per row
// and four brief-chapter rows, the shipped row withdraws from widening,
// Exclusions gains the widening-scoped shipped entry, the rendering gains a
// Floor column and the manifest item gains its scan mark. Every one of those is
// a change to what a reader and an auditor are PROMISED about what was
// examined, and the digest moves with the rendering by construction
// (adr-56 as refined 2026-09-02, spc-2609021003136831).
// It goes 1.5.0 to 1.6.0 with the comparative channel: Table gains the candidate
// row, Kinds() gains `candidate`, Exclusions gains the comparative position's
// derived ledger rows and narrows the `.abcd/work/issues` row to the three other
// positions, and the bundle item gains `candidate` and `field`. Every one of
// those moves what a reader and an auditor are PROMISED — a source admitted at
// one position, a vocabulary member, refusals a reader can now check — which is
// MINOR by this constant's own rule (adr-2609021016272867,
// spc-2609020626039834).
// It goes 1.6.0 to 1.7.0 with the reading of "at the target" DeriveCandidateRun
// states: a committed widening run at an ancestor of the target qualifies when
// only the readings store and the issue ledger changed between the two, the
// refusal listing gains the run whose object set moved and the path that moved
// it, and the manifest gains `candidate_run_target`. The selection rule and the
// manifest shape are both what a reader and an auditor are PROMISED, which is
// MINOR by this constant's own rule (adr-2609021016272867 as read there;
// iss-2609021833302981, iss-2609021857343626).
// It goes 1.7.0 to 1.8.0 with the transcript store's relocation: the exclusion
// floor's transcript-store row stops asserting an unreachability that is now
// conditional — a checkout declared in ~/.abcd/local-transcript-roots keeps its
// store inside the tree — and becomes a `directory` row naming
// `.abcd/.work.local/transcripts`. Signal `directory` is enforced by
// assertExclusions where `unreachable path` was enforced by nothing, so the
// floor gains a refusal a reader can check, which is MINOR by this constant's
// own rule (iss-95).
// It goes 1.8.0 to 1.9.0 with the reframe record: Exclusions gains the row
// naming the family at every position, and the comparative position's derived
// ledger rows gain `.abcd/work/issues/reframes` from the ledger's directory
// list. Both are refusals a reader can now check, which is MINOR by this
// constant's own rule (spc-2609020626048705).
// It goes 1.9.0 to 1.10.0 with the per-run context stamp: the bundle gains
// `context_stamp`, and the bundle shape is part of the contract a reader is
// PROMISED — it now carries a token naming its kind and its run — which is
// MINOR by this constant's own rule (adr-2609021016275803,
// spc-2609020626045177).
// It goes 1.10.0 to 1.11.0 with the knowledge record as a read object: Table
// gains the principles row at the three assembling positions, Kinds() gains
// `principle`, the projection gains its labelled-paragraph resolution, and
// Exclusions gains the four claim keys and the citation entry, which a new
// refusal enforces. A source admitted, a vocabulary member and refusals a
// reader can now check are each MINOR by this constant's own rule
// (adr-2609021016270132, spc-2609020626042471).
const AssemblerVersionCore = "1.11.0"

// AssemblerVersion is the core semver with the rendered include table's digest
// as semver build metadata. The digest is computed, not declared, so a table
// change moves the stamped version whether or not anyone notices: a manifest
// cannot name a version that does not describe the table it was built from.
//
// This is structural where the previous gate was advisory. That gate compared
// the rendering's digest to a standalone literal and never read the version at
// all, so changing the table and restating the literal was green with the
// version unmoved (iss-2608311949385350) — an attestation asserting more than
// its examination establishes, which brief invariant 16 forbids.
//
// The digest is carried WHOLE, not truncated. A short digest would have been
// easier to read and would not have supported the sentence above: Row.Rule is
// free prose of unbounded length inside the digested input, so a truncation is
// a collision an author can grind rather than one they would have to be unlucky
// to hit. The claim this function makes is absolute, so the evidence behind it
// is too — brief invariant 16 is the rule that an attestation states no more
// than its examination establishes, and this function IS an attestation.
func AssemblerVersion() string {
	sum := sha256.Sum256([]byte(Render()))
	return AssemblerVersionCore + "+" + hex.EncodeToString(sum[:])
}

// Position is the reading position an assembly is invoked at. The set is
// closed: an unknown token is refused by name, never defaulted.
type Position string

const (
	// PositionWidening reads for configurations the record has not considered.
	// It is the one position that must not see the candidate set it is asked to
	// widen, which is why its rows exclude the draft and planned intents.
	PositionWidening Position = "widening"
	// PositionEntailment reads for what the record already commits to. It sees
	// the draft and planned intents because articulation precedes selection.
	PositionEntailment Position = "entailment"
	// PositionComparative reads two or more configurations against the
	// selection-criteria discipline.
	PositionComparative Position = "comparative"
	// PositionDetection registers detections against the object.
	PositionDetection Position = "detection"
)

// Positions lists every position, in the order the charter renders them.
func Positions() []Position {
	return []Position{PositionWidening, PositionEntailment, PositionComparative, PositionDetection}
}

// AssemblingPositions lists the positions an assembly can run at.
//
// It is now every position. It was Positions() minus comparative, whose object
// is the widening reading's pre-admission output: itd-199 refused that position
// rather than serving it a corpus that was not what it is about, and scoped the
// channel out as a separate intent. adr-2609021016272867 is that channel — one
// derived widening run's candidates, two body fields each, at the comparative
// position and no other — so the refusal is withdrawn and the two lists are one.
//
// The function stays, rather than every caller moving to Positions(). The
// distinction it names is a real one and could return: a position with a
// definition, a regime and no assembly is a shape this design has already had
// once, and a caller asking "which positions assemble" should keep asking that
// question rather than assuming the answer.
func AssemblingPositions() []Position {
	return Positions()
}

// ParsePosition resolves a token to a position, refusing anything else by name.
func ParsePosition(s string) (Position, error) {
	for _, p := range Positions() {
		if string(p) == s {
			return p, nil
		}
	}
	names := make([]string, 0, len(Positions()))
	for _, p := range Positions() {
		names = append(names, string(p))
	}
	return "", fmt.Errorf("unknown reading position %q; the set is closed: %s",
		s, strings.Join(names, ", "))
}

// Kind is a bundle item's material class. It names WHAT a passed item is and
// never WHERE it came from: the vocabulary is closed, and no member of it
// carries a location. The path mapping lives in the manifest alone, so an
// auditor can resolve an item to its source and the reading cannot
// (brief invariant 15).
type Kind string

const (
	KindBriefSection     Kind = "brief-section"
	KindGlossaryTerm     Kind = "glossary-term"
	KindIntentProjection Kind = "intent-projection"
	KindDiscipline       Kind = "discipline"
	KindSpec             Kind = "spec"
	KindSource           Kind = "source"
	KindTest             Kind = "test"
	KindDoc              Kind = "doc"
	KindConfig           Kind = "config"
	// KindCandidate is one projected field of one item of the derived widening
	// run: the configuration, or what admits it. It is the comparative
	// position's whole object beside the criteria discipline, and it is the ONE
	// member of this vocabulary a committed preset entry may not name — an entry
	// selects repository material, and a candidate is selected by the derived
	// run (adr-2609021016272867; validatePresets refuses it by name).
	KindCandidate Kind = "candidate"
	// KindPrinciple is one principle of the knowledge record, projected to its
	// statement: the H1 title and the `**The rule.**` paragraph, links unwrapped
	// to their labels. Its four claim keys and every citation stay behind as
	// genealogy (adr-2609021016270132, spc-2609020626042471).
	KindPrinciple Kind = "principle"
)

// Kinds lists the closed material-class vocabulary.
func Kinds() []Kind {
	return []Kind{KindBriefSection, KindGlossaryTerm, KindIntentProjection,
		KindDiscipline, KindSpec, KindSource, KindTest, KindDoc, KindConfig,
		KindCandidate, KindPrinciple}
}

// Scan says whether the exclusion floor examined an item, and it is ONE word
// used in two places: on a Row it is the table's promise that the floor parses
// what the row admits, and on a ManifestItem it is the fact of whether the floor
// did. Spelling the promise and the fact differently is how the two came to
// disagree in the first place — the include table admitted by container shape
// and the floor examined by file extension, and nothing downstream was told
// which items the examination had reached (adr-56; brief invariant 16).
type Scan string

const (
	// ScanParsed: the floor read this item's frontmatter keys and headings, so
	// the manifest's key and heading exclusions are asserted for it.
	ScanParsed Scan = "parsed"
	// ScanUnscanned: the floor did not examine this item. It travels whole, and
	// the manifest says so per item rather than leaving a reader to infer from a
	// clean run that a scan ran (the 2026-09-02 refinement of adr-56).
	ScanUnscanned Scan = "unscanned"
)

// Scans lists the closed scan vocabulary, in the order the charter renders it.
func Scans() []Scan {
	return []Scan{ScanParsed, ScanUnscanned}
}

// Row is one include-table row: which positions admit this source, what is
// matched inside it, which fields are projected out of each matched file, the
// material class the items carry, and the rule that admits the row.
//
// A row's Source is the ONLY directory it reaches. The structural deny is
// measured from the Source downward, which is exactly why naming a record
// family's leaf bucket individually is legal while naming a directory that
// CONTAINS a family is not (assembler rule 1, ruling (18)).
type Row struct {
	// Positions are the reading positions this row is admitted at.
	Positions []Position
	// Source is the repo-relative directory the row reaches, "." for the
	// repository root.
	Source string
	// Match selects files inside Source: an entry beginning with "." is a file
	// extension, any other entry is an exact basename. An empty Match admits
	// every file, which no row uses — inclusion is positive at every grain.
	Match []string
	// MatchSuffix selects files inside Source by basename suffix, matched
	// case-sensitively. It is a separate field rather than a third convention
	// inside Match so the form is named by where it sits rather than inferred
	// from a string's first character: no disambiguation rule against the two
	// Match forms is needed, and none exists to get wrong. The case rule is
	// deliberate and differs from Match's extension form, which folds case:
	// the Go toolchain builds a test only from a lowercase _test.go, so folding
	// here would label material a test that Go does not build as one. The
	// suffix is not the WHOLE of Go's rule — a file named exactly _test.go is
	// ignored by the toolchain for its leading underscore, and files under
	// testdata/ are never built either, yet both are labelled test here. Those
	// are labelling differences on material no build compiles, never admission
	// differences. A file matched by either field is admitted; the two are
	// ORed (spc-68).
	MatchSuffix []string
	// Fields are the named fields projected out of each matched file, in the
	// order they are emitted. A field is resolved as a heading section where the
	// file carries that heading, otherwise as a frontmatter key. An empty Fields
	// passes the whole file as one item.
	Fields []string
	// Store and Bucket, when set, route the row's enumeration through the
	// record graph (internal/core/lint's LoadRecordGraph, the one parser of the
	// record's shape in this binary) rather than through a file walk. Bucket is
	// the lifecycle directory, empty for every bucket of the store.
	Store  string
	Bucket string
	// Kind is the material class every item from this row carries.
	Kind Kind
	// Scan declares whether the exclusion floor parses what this row admits. It
	// is the table's own statement of the narrowing it performs, so admission
	// and examination are ONE declaration rather than a row on one side and an
	// extension test inside the floor on the other — the undeclared scope
	// decision underneath the whole container-shape class
	// (iss-2608301450065320, adr-56 rule 3). collect reads it: a ScanParsed row
	// goes through redactExcluded, a ScanUnscanned row's document travels
	// untouched and every item it yields is marked unscanned in the manifest.
	Scan Scan
	// Rule is the rule that admits the row, quoted in the charter.
	Rule string
}

// AdmittedAt reports whether the row is admitted at p.
func (r Row) AdmittedAt(p Position) bool {
	for _, q := range r.Positions {
		if q == p {
			return true
		}
	}
	return false
}

// intentProjection is the field set a shipped, planned or draft intent travels
// as. It is positive at field granularity: an intent's Audit Notes, its Open
// Questions and its scope-condition dispositions carry no row here, so they
// cannot travel however the record's shape changes around them.
//
// The list is a CONTRACT, not a census, and the two differ per POSITION today.
// `Scope Conditions` and `Mechanism` are spc-55's headings: no shipped intent
// carries them, so the shipped rows yield three fields, while two drafts already
// do, so the entailment position — the only one that reads drafts — projects all
// five. A field the file does not carry simply contributes no item, which is
// what lets one projection describe a record whose sections the record is still
// growing, and what keeps this list from needing an edit on the day the rest of
// the corpus catches up.
var intentProjection = []string{
	"Press Release",
	"Acceptance Criteria",
	"Scope Conditions",
	"Mechanism",
	"spec_id",
}

// allPositions is the shared floor's position set.
var allPositions = []Position{PositionWidening, PositionEntailment, PositionComparative, PositionDetection}

// coldPositions is every position EXCEPT comparative, and it is the position set
// of every include row but two.
//
// At the comparative position the include table is the whole account of what the
// reading sees, and the readings companion (section 7.2, ratified position R3)
// admits no source there but the candidates and the declared criteria. So every
// row reaching ordinary repository material withdraws from that position: the six
// brief chapters and the glossary, the shipped intents, the specs, and the four
// shipped-tree rows. What remains is the disciplines row — narrowed at assembly
// to CriteriaDiscipline, so no committed entry can widen it — and the candidate
// row (spc-2609020626039834, "Every other row withdraws"; divergence register 22).
//
// The consequence is deliberate rather than incidental: a committed entry naming
// a path or a record at the comparative position selects nothing outside the
// criteria discipline, because there is nothing else there for it to reach.
var coldPositions = []Position{PositionWidening, PositionEntailment, PositionDetection}

// PrincipleSource is the knowledge record: one principle per file, each a
// statement and the genealogy it was distilled from.
const PrincipleSource = ".abcd/development/principles"

// PrincipleField is the one field a principle travels as: its statement, the
// labelled paragraph `**The rule.**` with the H1 title above it (projectField's
// labelled-paragraph resolution). Everything after it — the reasons, the
// bounds, the promotion rung — and every frontmatter key stays behind.
const PrincipleField = "The rule"

// CandidateSource is the ledger directory the candidate row reaches: the working
// tier's readings store, one directory per run. It is the leaf bucket the
// readings family keys on, which is what assembler rule 1 permits an include row
// to name (ruling (18)).
const CandidateSource = capture.LedgerRelPath + "/" + issueschema.ReadingsDir

// CandidateFields are the two widening body fields a candidate travels as: the
// configuration the widening reading returned, and what admits it. The item's
// pattern, its manifest reference, its regime and every other field of the record
// have no row here and therefore no item — provenance is the ENVELOPE's and not
// the candidate's, and widening the projection by one field would be a manifest
// change and a recorded one (the intent's third scope condition; divergence
// register 23).
var CandidateFields = []string{"configuration", "what_admits_it"}

// Table is the include table: the single source of truth for what a cold
// reading may see. It is rendered into the readings charter under a test
// asserting the two agree, on the idiom internal/core/lifeboat/mapping.go
// carries for the brief.
//
// Order is significant only for rendering and for tie-breaking a path two rows
// admit: the first row that reaches a path owns the projection applied to it.
var Table = []Row{
	{
		Positions: coldPositions,
		Source:    ".abcd/development/brief/01-product",
		Match:     []string{".md"},
		Kind:      KindBriefSection,
		Scan:      ScanParsed,
		Rule: "adr-55: the construal as it presently stands is committed record, " +
			"admissible to every reader including a cold reading",
	},
	{
		Positions: coldPositions,
		Source:    ".abcd/development/brief/02-constraints",
		Match:     []string{".md"},
		Kind:      KindBriefSection,
		Scan:      ScanParsed,
		Rule: "The constraints chapter states the platform, the dependency stance, " +
			"the invariants and the naming a reading reads against",
	},
	{
		Positions: coldPositions,
		Source:    ".abcd/development/brief/04-surfaces",
		Match:     []string{".md"},
		Kind:      KindBriefSection,
		Scan:      ScanParsed,
		Rule: "The surfaces chapter is brief current text, which both design documents " +
			"name as a reading's object",
	},
	{
		Positions: coldPositions,
		Source:    ".abcd/development/brief/05-internals",
		Match:     []string{".md"},
		Kind:      KindBriefSection,
		Scan:      ScanParsed,
		Rule: "The internals chapter is brief current text, which both design documents " +
			"name as a reading's object",
	},
	{
		Positions: coldPositions,
		Source:    ".abcd/development/brief/06-delivery",
		Match:     []string{".md"},
		Kind:      KindBriefSection,
		Scan:      ScanParsed,
		Rule: "The delivery chapter is brief current text, which both design documents " +
			"name as a reading's object",
	},
	{
		Positions: coldPositions,
		Source:    ".abcd/development/brief/glossary",
		Match:     []string{".md"},
		Kind:      KindGlossaryTerm,
		Scan:      ScanParsed,
		Rule: "adr-55: the glossary's committed terms are committed record; " +
			"superseded terms and the reasoning that settled them are not",
	},
	{
		// The meta chapter is one file at the brief's root, so this row's source
		// is the brief directory and its Match is that exact basename: the row
		// reaches 00-meta.md and nothing beside it. The six chapters are named
		// individually rather than by naming `brief/` whole because assembler
		// rule 1 forbids naming a directory that contains a record family, and
		// this one contains the glossary — which keeps its own row above.
		Positions: coldPositions,
		Source:    ".abcd/development/brief",
		Match:     []string{"00-meta.md"},
		Kind:      KindBriefSection,
		Scan:      ScanParsed,
		Rule: "The meta chapter is brief current text, which both design documents " +
			"name as a reading's object; it is one file at the brief's root",
	},
	{
		// The one repository row that keeps the comparative position. At that
		// position the assembler narrows this row's enumeration to
		// CriteriaDiscipline before the committed entry is applied, so the
		// declared criteria travel beside the candidates and nothing else does
		// (spc-2609020626039834; itd-191).
		Positions: allPositions,
		Source:    ".abcd/development/intents/disciplines",
		Match:     []string{".md"},
		Store:     "itd",
		Bucket:    "disciplines",
		Kind:      KindDiscipline,
		Scan:      ScanParsed,
		Rule: "A discipline is a standing commitment the record already holds, " +
			"named individually inside the intent family",
	},
	{
		// Not at widening. The design framework's widening object and the
		// readings companion's section 5.2 both state that object without the
		// shipped intents, and the maintainer ruled on 2026-09-02 that the row
		// follows the documents (iss-2609012259587904). The withdrawal is
		// asserted in the exclusion floor below, so a reader checks it rather
		// than inferring it from a row's silence.
		Positions: []Position{PositionEntailment, PositionDetection},
		Source:    ".abcd/development/intents/shipped",
		Match:     []string{".md"},
		Store:     "itd",
		Bucket:    "shipped",
		Fields:    intentProjection,
		Kind:      KindIntentProjection,
		Scan:      ScanParsed,
		Rule: "Assembler rule 2: a shipped intent travels as its claim record, " +
			"so the Audit Notes and dispositions it also carries stay behind; not at " +
			"widening, whose object neither design document states with them in it " +
			"(ruled 2026-09-02, iss-2609012259587904)",
	},
	{
		Positions: coldPositions,
		Source:    ".abcd/development/specs",
		Match:     []string{".md"},
		Store:     "spc",
		Kind:      KindSpec,
		Scan:      ScanParsed,
		Rule:      "The design record a capability was built against",
	},
	{
		Positions: []Position{PositionEntailment},
		Source:    ".abcd/development/intents/drafts",
		Match:     []string{".md"},
		Store:     "itd",
		Bucket:    "drafts",
		Fields:    intentProjection,
		Kind:      KindIntentProjection,
		Scan:      ScanParsed,
		Rule: "Assembler rule 2: articulation precedes selection, so entailment sees " +
			"the candidate set and the reading asked to widen it does not",
	},
	{
		Positions: []Position{PositionEntailment},
		Source:    ".abcd/development/intents/planned",
		Match:     []string{".md"},
		Store:     "itd",
		Bucket:    "planned",
		Fields:    intentProjection,
		Kind:      KindIntentProjection,
		Scan:      ScanParsed,
		Rule: "Assembler rule 2: articulation precedes selection, so entailment sees " +
			"the candidate set and the reading asked to widen it does not",
	},
	{
		// The knowledge record as a read object (adr-2609021016270132). A
		// principle's statement is what a project carries forward and sits on
		// the cold side; its claim keys and its citations back to the records it
		// was distilled from are derivational and stay behind, which is a
		// projection rule rather than a path rule — so the row names the family
		// and projects one field, and the floor below asserts the rest.
		//
		// Not at comparative: there the include table is the whole account and
		// admits the candidates and the criteria alone (companion 7.2, R3).
		// Which of the three assembling positions RECEIVES the knowledge record
		// is a committed preset entry's choice, measured by the presets' eval;
		// the table only admits it.
		Positions: coldPositions,
		Source:    PrincipleSource,
		Match:     []string{".md"},
		Store:     "prn",
		Fields:    []string{PrincipleField},
		Kind:      KindPrinciple,
		Scan:      ScanParsed,
		Rule: "The knowledge record is a read object: a principle travels as its statement, " +
			"and its keys and citations are genealogy (adr-2609021016270132)",
	},
	{
		// Ordered above the .go row deliberately: path.Ext("foo_test.go") is
		// ".go", so the source row would otherwise claim every test file, and
		// the first row that reaches a path owns it. Both rows admit — this
		// row narrows nothing and widens nothing, it only labels.
		Positions:   coldPositions,
		Source:      ".",
		MatchSuffix: []string{"_test.go"},
		Kind:        KindTest,
		Scan:        ScanUnscanned,
		Rule: "Admitted where a committed preset entry names this kind, and never examined: " +
			"an item admitted here travels whole and marked `unscanned` in the manifest, " +
			"because the exclusion floor's key and heading signals are record shapes only a " +
			"markdown file carries",
	},
	{
		Positions: coldPositions,
		Source:    ".",
		Match:     []string{".go"},
		Kind:      KindSource,
		Scan:      ScanUnscanned,
		Rule: "Admitted where a committed preset entry names this kind, and never examined: " +
			"an item admitted here travels whole and marked `unscanned` in the manifest, " +
			"because the exclusion floor's key and heading signals are record shapes only a " +
			"markdown file carries",
	},
	{
		Positions: coldPositions,
		Source:    ".",
		Match:     []string{".md"},
		Kind:      KindDoc,
		Scan:      ScanParsed,
		Rule: "Assembler rule 1: the shipped tree is the delivered documentation and the " +
			"root prose, with the record denied structurally",
	},
	{
		Positions: coldPositions,
		Source:    ".",
		Match:     []string{".json", ".yml", ".yaml", ".toml", ".mod", ".sum", "Makefile"},
		Kind:      KindConfig,
		Scan:      ScanUnscanned,
		Rule: "Admitted where a committed preset entry names this kind, and never examined: " +
			"an item admitted here travels whole and marked `unscanned` in the manifest, " +
			"because the exclusion floor's key and heading signals are record shapes only a " +
			"markdown file carries",
	},
	{
		// The candidate channel, and the one positional exception to the
		// prior-run exhaust (adr-2609021016272867).
		//
		// It is a TABLE ROW rather than a second mechanism, and the rest follows
		// from that without anything else being written: Admits answers for the
		// derived run's records at this position and nowhere else, Render carries
		// the row so the charter names the channel and AssemblerVersion digests
		// it, the dirty gate refuses an uncommitted candidate because the path is
		// admitted, and the comparative definition's source list is regenerated
		// from the table like every other position's.
		//
		// The row reaches the FAMILY; the assembly narrows it to one run by
		// setting Bucket to the derived run id before the walk, the way a record
		// in an entry's object set narrows a record row. The rdi store's bucket
		// IS the run directory, so the narrowing needs no second selector.
		Positions: []Position{PositionComparative},
		Source:    CandidateSource,
		Match:     []string{".md"},
		Store:     issueschema.ReadingItemFamily,
		Fields:    CandidateFields,
		Kind:      KindCandidate,
		Scan:      ScanParsed,
		Rule: "adr-2609021016272867: at the comparative position, and at no other, two body " +
			"fields of one DERIVED widening run's items are admitted as that reading's " +
			"candidate set — the candidate text is cold, and its fate is warm. Everything " +
			"else in the readings store stays excluded: the run's manifest, the items' " +
			"envelopes and patterns, every disposition, admission and surprise, and every " +
			"other run",
	},
}

// Exclusions is the exclusion floor: every field, heading and directory the
// assembler refuses, each with the signal by which a reader detects it. It is
// asserted into every manifest so a reader can check the exclusions rather than
// trust a disclosure.
//
// The floor is a DECLARATION a reader checks, and the assembler additionally
// refuses to emit an item under any path-shaped entry, so the two cannot part
// company (see assertExclusions).
//
// What each entry asserts, and over WHICH items, is fixed here rather than left
// to a reader: an entry whose Signal is `frontmatter key` or `heading` is
// asserted for the items the manifest marks `parsed` and for no other, because
// those two signals are read by a scan and a scan that did not run establishes
// nothing; an entry whose Signal is `directory`, `file` or `unreachable path`
// is asserted for EVERY item, because assertExclusions enforces those by path
// and a path needs no parse. That is brief invariant 16 made a property of the
// artefact rather than of the run: a scan that ran and a scan that never ran no
// longer produce the same attestation (adr-56 as refined 2026-09-02).
var Exclusions = []Exclusion{
	{Rule: "field projection", Signal: "frontmatter key", Detail: "origin"},
	{Rule: "field projection", Signal: "frontmatter key", Detail: "production_mode"},
	// A principle's four claim keys (adr-2609021016270132): what kind of claim
	// it makes, what it is about, what comparison produced it, and the records
	// and conditions it rests on. They are genealogy, and a principle travels as
	// its statement alone.
	{Rule: "field projection", Signal: "frontmatter key", Detail: "claim_type"},
	{Rule: "field projection", Signal: "frontmatter key", Detail: "reference"},
	{Rule: "field projection", Signal: "frontmatter key", Detail: "comparison"},
	{Rule: "field projection", Signal: "frontmatter key", Detail: "evidence"},
	{Rule: "field projection", Signal: "heading", Detail: "Audit Notes"},
	{Rule: "field projection", Signal: "heading", Detail: "Open Questions"},
	{Rule: "field projection", Signal: "heading", Detail: "Why This Matters"},
	{Rule: "a reading's object excludes what it exists to change", Signal: "heading", Detail: "Scope Condition Dispositions"},
	// The one brief chapter the brief rows do not reach, and the ground is the
	// framework's own: 7.1 excludes a record's Audit Notes because a prior
	// verdict is revision history, and the evidence chapter is that material at
	// chapter grain. Stating the ground here rather than in the floor's code
	// means the charter says why the chapter is left out (iss-2609021153264023).
	{Rule: "verdict material: a prior verdict is revision history, the ground the Audit Notes exclusion rests on",
		Signal: "directory", Detail: ".abcd/development/brief/03-evidence"},
	{Rule: "absent from the positive walk", Signal: "directory", Detail: ".abcd/development/decisions"},
	{Rule: "absent from the positive walk", Signal: "directory", Detail: ".abcd/development/roadmap/rfcs"},
	{Rule: "absent from the positive walk", Signal: "directory", Detail: ".abcd/development/intents/superseded"},
	{Rule: "absent from the positive walk", Signal: "directory", Detail: ".abcd/development/plans"},
	{Rule: "absent from the positive walk", Signal: "directory", Detail: ".abcd/development/research/notes"},
	// Not at comparative. The candidate row reaches one leaf bucket inside this
	// directory, so an exclusion binding here would contradict the row that
	// admits it. What replaces it at that position is NARROWER and asserted per
	// family rather than at the container: comparativeExclusions below names
	// every ledger family individually, so the withdrawal buys a reader more
	// than it costs them (adr-2609021016272867; spc-2609020626039834).
	{
		Rule:      "no include names a directory containing a record family",
		Signal:    "directory",
		Detail:    capture.LedgerRelPath,
		Positions: coldPositions,
	},
	{Rule: "absent from the positive walk", Signal: "file", Detail: ".abcd/work/DECISIONS.md"},
	// A principle's citations. The projection keeps the statement and unwraps a
	// link to its label, and verifyPrincipleItem refuses the assembly if a
	// record handle survives into a principle item, so this is an assertion the
	// assembler checks rather than a disclosure a reader trusts.
	{Rule: "the statement is knowledge and the citations are genealogy", Signal: "citation",
		Detail: "record handles and links in a principle"},
	// The local ledger tier. It was excluded from the first day and asserted by
	// nothing: absent from the positive walk, denied by the `.abcd` segment, and
	// named in no row here — so every manifest was silent about the one tier
	// brief invariant 14 exists to keep out, and assertExclusions had nothing to
	// enforce there. A reader checking the asserted exclusions could not tell
	// that the framing traces and declined construals had been refused, which is
	// brief invariant 16 in its less-than direction: an attestation never states
	// less than the examination behind it establishes (iss-2608311238236490).
	{Rule: "no reading consumes the local ledger side, unconditionally and under no flag (brief invariant 14)",
		Signal: "directory", Detail: ".abcd/.work.local"},
	{Rule: "absent from the positive walk", Signal: "record type in a denied path", Detail: "the lapse log"},
	// The reframe record (spc-2609020626048705): a pointer to a reframe whose
	// content stays local, warm at every position. It lives under the ledger, so
	// the container row above and the derived per-family row at comparative
	// already refuse it by path; this row is the declaration a reader checks,
	// naming the family rather than leaving it to be inferred from a directory.
	{Rule: "absent from the positive walk", Signal: "record type in a denied path", Detail: "the reframe record"},
	{Rule: "absent from the positive walk", Signal: "record type in a denied path", Detail: "admission and selection grounds"},
	{Rule: "the instrument's own output is never its input", Signal: "directory", Detail: ".abcd/development/readings"},
	{Rule: "the instrument's own output is never its input", Signal: "directory", Detail: "agents"},
	{Rule: "the instrument's own output is never its input", Signal: "directory", Detail: "evals"},
	{Rule: "the instrument's own output is never its input", Signal: "directory", Detail: "internal/core/reading"},
	// The session-transcript store. The claim used to rest on unreachability —
	// the store sits under HOME and the assembler walks no HOME path — and that
	// still holds for the default location, ~/.abcd/transcripts/. It is no
	// longer the WHOLE ground: a checkout declared in
	// ~/.abcd/local-transcript-roots keeps its store at
	// .abcd/.work.local/transcripts/ inside the tree (iss-95). The exclusion is
	// unaffected, because that path is covered by the `.abcd/.work.local`
	// directory row above — denied by the `.abcd` segment, dropped by the
	// tracked-set intersection since the local tier is gitignored, and refused
	// by assertExclusions on the prefix. The row says that instead of asserting
	// an unreachability that is now conditional: brief invariant 16 in its
	// more-than direction — an attestation never states more than the
	// examination behind it establishes.
	{Rule: "the store is out of tree by default, and inside it only under .abcd/.work.local, which is excluded above",
		Signal: "directory", Detail: ".abcd/.work.local/transcripts"},
	{
		Rule:      "a reading's object excludes what it exists to change",
		Signal:    "directory",
		Detail:    ".abcd/development/intents/drafts",
		Positions: []Position{PositionWidening, PositionComparative, PositionDetection},
	},
	{
		Rule:      "a reading's object excludes what it exists to change",
		Signal:    "directory",
		Detail:    ".abcd/development/intents/planned",
		Positions: []Position{PositionWidening, PositionComparative, PositionDetection},
	},
	{
		Rule:      "the widening object as the design documents state it",
		Signal:    "directory",
		Detail:    ".abcd/development/intents/shipped",
		Positions: []Position{PositionWidening},
	},
	// The knowledge record's one withdrawal. At comparative the include table
	// is the whole account and admits the candidates and the criteria alone, so
	// the principles row withdraws there; the floor asserts it, so a reader
	// checks the withdrawal rather than inferring it from a row's silence, and
	// assertExclusions enforces it by path (spc-2609020626042471).
	{
		Rule:      "the comparative reading receives the candidates and the criteria alone",
		Signal:    "directory",
		Detail:    PrincipleSource,
		Positions: []Position{PositionComparative},
	},
}

// comparativeExclusions is what replaces the container row at the comparative
// position: one entry per ledger family, DERIVED from the ledger's own directory
// list rather than listed here.
//
// The derivation is the point. A family the ledger adds later is excluded at
// this position the day its constant is declared, and the scribe's allow list in
// spc-2609020626045177 derives from the same function — so the two instruments
// describe one set rather than two lists that agree today. Hand-listing them
// here would be the shape brief invariant 16 forbids: an attestation whose
// coverage nobody can check against the thing it claims to cover.
//
// The readings family gets a SIGNAL row rather than a directory row, because a
// directory row would be false: one run's items do travel. What it asserts is
// the narrower thing the row actually promises, and assertCandidateProjection is
// its fail-closed half — a directory row is enforced by path, and this one is
// enforced by refusing to emit a candidate outside the derived run and the two
// fields (adr-56: an exclusion control asserts only what it can prove).
func comparativeExclusions() []Exclusion {
	out := make([]Exclusion, 0, len(issueschema.LedgerDirs()))
	for _, dir := range issueschema.LedgerDirs() {
		if dir == issueschema.ReadingsDir {
			out = append(out, Exclusion{
				Rule: "the instrument's own output is never its input, but for the one derived " +
					"widening run adr-2609021016272867 admits",
				Signal: "readings store",
				Detail: "every run other than the candidate run, and every field of the named " +
					"run's items other than " + strings.Join(CandidateFields, " and "),
				Positions: []Position{PositionComparative},
			})
			continue
		}
		out = append(out, Exclusion{
			Rule: "the comparative reading receives candidates and never their fate: the " +
				"ledger's other families are excluded family by family, derived from the " +
				"ledger's own directory list",
			Signal:    "directory",
			Detail:    capture.LedgerRelPath + "/" + dir,
			Positions: []Position{PositionComparative},
		})
	}
	return out
}

func init() {
	Exclusions = append(Exclusions, comparativeExclusions()...)
}

// Exclusion is one refused source and the signal that detects it.
type Exclusion struct {
	// Rule is the assembler rule the exclusion instances.
	Rule string `json:"rule"`
	// Signal is how a reader detects it: a frontmatter key, a heading, a
	// directory absent from the positive walk, or a store outside the tree.
	Signal string `json:"signal"`
	// Detail names the excluded thing.
	Detail string `json:"detail"`
	// Positions are the positions the exclusion binds at. An empty Positions
	// binds at every position.
	Positions []Position `json:"-"`
}

// ExclusionsFor returns the exclusions binding at p, in table order.
func ExclusionsFor(p Position) []Exclusion {
	out := []Exclusion{}
	for _, e := range Exclusions {
		if len(e.Positions) == 0 {
			out = append(out, e)
			continue
		}
		for _, q := range e.Positions {
			if q == p {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// Marker delimiters for the rendered table in the readings charter.
const (
	MarkerBegin = "<!-- BEGIN GENERATED: reading-include-table -->"
	MarkerEnd   = "<!-- END GENERATED: reading-include-table -->"
)

// CharterPath is the readings family's charter, which renders the table.
const CharterPath = ".abcd/development/readings/README.md"

// Render renders the include table and the exclusion floor as the markdown the
// charter carries between the markers.
func Render() string {
	var b strings.Builder
	b.WriteString("### Include table\n\n")
	b.WriteString("| Positions | Source | Matches | Suffixes | Fields | Store | Bucket | Kind | Floor | Admitting rule |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, row := range Table {
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s | %s | %s | %s | `%s` | `%s` | %s |\n",
			sortedPositions(row.Positions),
			row.Source,
			matchList(row),
			suffixList(row.MatchSuffix),
			fieldList(row.Fields),
			routeField(row.Store),
			routeField(row.Bucket),
			row.Kind,
			row.Scan,
			row.Rule)
	}
	b.WriteString("\n### Exclusion floor\n\n")
	b.WriteString("| Detail | Signal | Rule | Positions |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, e := range Exclusions {
		positions := "every position"
		if len(e.Positions) > 0 {
			positions = sortedPositions(e.Positions)
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", e.Detail, e.Signal, e.Rule, positions)
	}
	return b.String()
}

// matchList renders a row's Matches column. An empty Match means "every file"
// only when the row selects by nothing else; on a row that selects by suffix it
// means the Match form contributes nothing, and rendering that as "every file"
// would state the opposite of what the row admits. The rendering IS the
// contract the assembler version digests, so the two cases are distinguished
// here rather than left to a reader (spc-68).
func matchList(row Row) string {
	if len(row.Match) == 0 && len(row.MatchSuffix) == 0 {
		return "every file"
	}
	if len(row.Match) == 0 {
		return "none"
	}
	return codeList(row.Match)
}

// routeField renders a row's Store or Bucket column.
//
// Both are rendered, and the reasoning for leaving them out did not survive
// examination: rowPaths filters candidates on Bucket and selects a node type by
// Store, so they decide what a reading receives as surely as Match does. They
// look inert today only because every row's Source already bounds it to one
// bucket — a coincidence of the current record layout, not a property, and
// nothing enforces it. Rendering them costs two columns and removes the need
// for anyone to re-derive that argument and get it wrong.
func routeField(s string) string {
	if s == "" {
		return "every"
	}
	return "`" + s + "`"
}

// suffixList renders a row's Suffixes column.
func suffixList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return codeList(items)
}

// codeList renders a non-empty token list as backticked tokens. It has no
// empty-list branch: every caller decides for itself what an empty list MEANS
// in its own column, and the branch that used to answer "every file" here was
// unreachable once matchList took that decision over.
func codeList(items []string) string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, "`"+it+"`")
	}
	return strings.Join(out, ", ")
}

// fieldList renders a projection's field list.
func fieldList(items []string) string {
	if len(items) == 0 {
		return "the whole file"
	}
	return codeList(items)
}

// sortedPositions renders a row's positions in canonical order.
func sortedPositions(ps []Position) string {
	order := map[Position]int{}
	for i, p := range Positions() {
		order[p] = i
	}
	cp := append([]Position{}, ps...)
	sort.Slice(cp, func(i, j int) bool { return order[cp[i]] < order[cp[j]] })
	names := make([]string, 0, len(cp))
	for _, p := range cp {
		names = append(names, string(p))
	}
	return strings.Join(names, ", ")
}
