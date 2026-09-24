//go:build smoke || coldreading

package evals

// The sentinel fixture corpus: what is planted, where, and how a variant of it
// is materialised into a repository the assembler can be run over.
//
// The corpus is inert on disk under testdata/cold-reading/ and materialised into
// a temporary directory per run, because the assembler reads a git working tree
// and refuses one whose included paths are uncommitted. Nothing here reads the
// assembler: the plants are declared, the counts are declared, and the eval is
// wrong about the assembler exactly when the assembler is wrong about the
// record.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The four reading positions, spelled as the closed tokens the verb accepts.
const (
	posWidening    = "widening"
	posEntailment  = "entailment"
	posComparative = "comparative"
	posDetection   = "detection"
)

// evalPresetName is the single committed preset the fixture repository carries,
// applied by the assembler with no operand (adr-2609021016286571). It names
// EVERY material kind at every assembling position, deliberately: three of the
// read-block's eleven carriers (main.go, fence.go, go.mod) are shipped-tree
// files, and fence.go is the sole corpus behind the body-redaction row, so an
// entry that dropped source would turn a live assertion into an undeclared gap
// without failing anything. The eval asserts a firewall, and a firewall is
// asserted over the whole corpus or not at all.
//
// It is named here rather than passed anywhere: the invocation carries no
// operand that could name it, and the constant exists so the fixture's own
// preset file and this comment cannot drift apart unnoticed.
const evalPresetName = "everything"

// assemblingPositions is every position, which is what it was before itd-199
// took comparative out of it and what adr-2609021016272867 restores.
//
// itd-199 removed comparative because the position refused: its object is the
// widening reading's pre-admission output and no channel supplied it. The
// channel exists — one derived widening run's items, two body fields each — so
// the position assembles and every assertion here runs at it again.
var assemblingPositions = []string{posWidening, posEntailment, posComparative, posDetection}

// everyPosition is the closed set, in charter order.
var everyPosition = []string{posWidening, posEntailment, posComparative, posDetection}

// fullyAsserted are the positions every bundle assertion runs at.
//
// It is all four, and the reason it must be is worth keeping: leaving
// comparative out had once left six of the ten sentinel classes unasserted
// there and left the oracle's drafts-at-comparative exclusion a row that could
// never fire, so an assembler admitting the candidate set or the local ledger
// tier at that one position was green. It was out for a second time while the
// position refused, and the refusal itself was what stood in for the
// assertions. Now that the position assembles, the assertions are back — and
// they reach further than before, because that position is the one whose object
// comes OUT of the ledger, so the classes that are warm there are warm by a
// narrower margin than anywhere else.
var fullyAsserted = assemblingPositions

// sentinelPrefix is the shape every planted token takes, so a leak names the
// warm location class that leaked rather than reading as ordinary prose.
const sentinelPrefix = "ABCD-EVAL-SENTINEL-"

// promotedFixtureItem is the reading item the promoted draft in the corpus was
// graduated from, and it is NOT a sentinel class: it is an identifier that must
// reach no bundle at any position, not a warm span with a home to count.
//
// A promoted draft carries the join in `origin` and `related_issues` — two
// frontmatter keys the intent projection does not name — and names the item once
// more under Why This Matters, which the projection does not name either. The
// seed in its Press Release, which IS projected, says "a reading item" and no
// more. That is the readings companion's rule made falsifiable: no reading sees
// another's output (companion 8.3), and a draft is admitted at entailment.
const promotedFixtureItem = "rdi-2609020000000009"

// sentinelClass is one warm location class: its token, every home the corpus
// plants it in, and the positions (if any) at which it may legitimately reach a
// reading.
type sentinelClass struct {
	// Name completes the token: sentinelPrefix + Name.
	Name string
	// Homes are the planted files. A "repo:" home is repo-relative inside the
	// materialised fixture repository; a "home:" home is relative to the
	// materialised fixture HOME.
	Homes []string
	// Count is how many times the token appears across the materialised fixture.
	// It is declared rather than counted so a corpus that lost a plant fails.
	Count int
	// ColdAt are the positions at which this token is COLD — legitimately part of
	// the reading's object, so its presence in the bundle is not a leak. Empty
	// means the token is warm at every position.
	ColdAt []string
	// Why states the warm location class in the record's own terms.
	Why string
}

// Token is the class's planted token.
func (c sentinelClass) Token() string { return sentinelPrefix + c.Name }

// coldAt reports whether the class is cold at position p.
func (c sentinelClass) coldAt(p string) bool {
	for _, q := range c.ColdAt {
		if q == p {
			return true
		}
	}
	return false
}

// sentinelClasses is the plants table: one class per warm location class the
// record names, each carrying a distinct token.
//
// The counts are the anti-vacuity guard's oracle (TestEverySentinelIsPlanted).
// An absence assertion cannot see a corpus that lost its plants, so the corpus
// is asserted separately from the absence.
var sentinelClasses = []sentinelClass{
	{
		Name:  "LEDGER-FRAMING",
		Homes: []string{"repo:.abcd/.work.local/ledger/2026-08-30-declined-construals.md"},
		Count: 1,
		Why: "brief invariant 14: declined construals and every other framing trace live " +
			"on the local ledger side, and no reading consumes that side",
	},
	{
		Name: "TRANSCRIPT",
		Homes: []string{
			// The repo-side plant stands for the in-repo half of the store: a
			// checkout declared in ~/.abcd/local-transcript-roots keeps its
			// transcripts under this same .work.local tier (iss-95), which the
			// assembler denies by segment, drops by the tracked-set
			// intersection, and refuses by asserted exclusion.
			"repo:.abcd/.work.local/scratch/session-notes.md",
			"home:.abcd/transcripts/ROOT_COMMIT_SHA/records/ses-0001-a-stored-session.md",
		},
		Count: 2,
		Why: "brief invariant 15: the session-transcript store has an enumerated consumer " +
			"list, and the cold readings are denied it structurally",
	},
	{
		Name: "DECISION",
		Homes: []string{
			"repo:.abcd/work/DECISIONS.md",
			"repo:.abcd/development/decisions/adrs/0001-a-ruled-decision.md",
			"repo:.abcd/work/issues/open/iss-1-a-lapse-entry.md",
			"repo:.abcd/work/issues/resolved/iss-2-a-resolved-defect.md",
			"repo:.abcd/work/issues/wontfix/iss-3-a-declined-defect.md",
		},
		Count: 5,
		Why: "itd-183 exclusion list: decisions/, work/issues/ in every state, a wontfix " +
			"reason, and the lapse log",
	},
	{
		Name: "DELIBERATION",
		Homes: []string{
			"repo:.abcd/development/brief/03-evidence/01-open-questions.md",
			"repo:.abcd/development/intents/superseded/itd-5-a-retired-intent.md",
			"repo:.abcd/development/plans/2026-08-30-a-plan.md",
			"repo:.abcd/development/research/notes/2026-08-30-a-note.md",
			"repo:.abcd/development/roadmap/rfcs/0001-an-rfc.md",
		},
		Count: 5,
		Why: "itd-183 exclusion list: brief/03-evidence/, intents/superseded/, plans/, " +
			"research/notes/, roadmap/rfcs/ are deliberation, not committed construal",
	},
	{
		Name: "WARM-FIELD",
		Homes: []string{
			"repo:.abcd/development/brief/01-product/01-press-release.md",
			"repo:.abcd/development/intents/disciplines/itd-4-selection-criteria.md",
			"repo:.abcd/development/intents/shipped/itd-1-a-shipped-intent.md",
			"repo:.abcd/development/specs/open/spc-1-a-design-record.md",
		},
		Count: 7,
		Why: "itd-183: a heading the exclusion floor names, on a record type the include " +
			"list admits. Every one of the four excluded headings has a home on a record " +
			"type that travels WHOLE — Why This Matters in a brief chapter, Audit Notes in " +
			"a discipline, Open Questions and the scope-condition dispositions in a spec — " +
			"because on a PROJECTED record type the projection keeps the heading out " +
			"whatever the floor says, so deleting that heading's exclusion there leaks " +
			"nothing and the rule cannot be falsified. The three on the shipped intent are " +
			"the projected shape, kept so both shapes are exercised",
	},
	{
		Name: "UNPROJECTED-SECTION",
		Homes: []string{
			"repo:.abcd/development/intents/shipped/itd-1-a-shipped-intent.md",
			"repo:.abcd/development/intents/drafts/itd-2-a-draft-intent.md",
			"repo:.abcd/development/intents/planned/itd-3-a-planned-intent.md",
		},
		Count: 3,
		Why: "itd-183 assembler rule 2: the intent projection is POSITIVE at field " +
			"granularity, so a section it does not name stays behind — a section that is " +
			"neither projected nor on the exclusion floor is the only plant that can tell " +
			"a live projection from a dead one the redaction tidied up after",
	},
	{
		Name: "WARM-KEY",
		Homes: []string{
			"repo:.abcd/development/specs/open/spc-1-a-design-record.md",
			"repo:.abcd/development/intents/disciplines/itd-4-selection-criteria.md",
			"repo:.abcd/development/brief/01-product/01-press-release.md",
		},
		Count: 3,
		Why:   "itd-183 exclusion list: origin and production mode, detected by frontmatter key",
	},
	{
		Name: "EXHAUST",
		Homes: []string{
			"repo:.abcd/development/readings/rdg-2608300900000001/manifest.json",
			"repo:.abcd/work/issues/readings/rdi-1-a-prior-reading.md",
			"repo:.abcd/work/issues/dispositions/dsp-1-a-prior-disposition.md",
			// The SECOND widening run's items. The comparative exception admits
			// ONE derived run, so another run's returned text is exhaust at every
			// position including that one — which is the half of the exception a
			// path rule could not hold, because both runs live in one family
			// (adr-2609021016272867).
			"repo:.abcd/work/issues/readings/" + dispositionedWideningRun + "/rdi-201.md",
			"repo:.abcd/work/issues/readings/" + dispositionedWideningRun + "/rdi-202.md",
		},
		Count: 5,
		Why: "itd-183: manifests, reading records and dispositions are warm on the next " +
			"run, so the instrument's own output is never its input — with the one " +
			"positional exception adr-2609021016272867 states, which reaches ONE run",
	},
	{
		Name: "GROUNDS",
		Homes: []string{
			"repo:.abcd/work/issues/admissions/adm-1-admission-grounds.md",
			"repo:.abcd/work/issues/admissions/adm-2-selection-grounds.md",
			"repo:.abcd/work/issues/admissions/" + dispositionedWideningRun + "/adm-201.md",
			"repo:.abcd/work/issues/admissions/" + dispositionedWideningRun + "/adm-202.md",
		},
		Count: 4,
		Why: "itd-183 exclusion list: admission and selection grounds. The two under a run " +
			"directory are what the comparative position's derived admissions row excludes: " +
			"a candidate's fate is the researcher's judgement and never a reading's input",
	},
	{
		// The CARRIER of the comparative channel, and the only class in this table
		// that must ARRIVE somewhere. Everything else here is a leak when it
		// appears; this is a leak's inverse — a channel that stopped carrying is
		// as much a defect as one that carries too much, and an absence oracle
		// cannot see it (adr-2609021016272867; companion 7.2, R4).
		Name: "CANDIDATE",
		Homes: []string{
			"repo:.abcd/work/issues/readings/" + derivedCandidateRun + "/rdi-301.md",
			"repo:.abcd/work/issues/readings/" + derivedCandidateRun + "/rdi-302.md",
			"repo:.abcd/work/issues/readings/" + derivedCandidateRun + "/rdi-303.md",
		},
		Count:  3,
		ColdAt: []string{posComparative},
		Why: "adr-2609021016272867: at the comparative position and no other, the derived " +
			"widening run's returned configurations ARE the reading's object; at every " +
			"other position they are the instrument's own output",
	},
	{
		// The limit inside the exception. The projection is two body fields, and
		// the item's pattern is not one of them: provenance is the envelope's and
		// not the candidate's (the intent's third scope condition; divergence
		// register 23).
		Name: "ENVELOPE",
		Homes: []string{
			"repo:.abcd/work/issues/readings/" + derivedCandidateRun + "/rdi-301.md",
			"repo:.abcd/work/issues/readings/" + derivedCandidateRun + "/rdi-302.md",
			"repo:.abcd/work/issues/readings/" + derivedCandidateRun + "/rdi-303.md",
		},
		Count: 3,
		Why: "the candidate projection is two fields; the item's pattern, its manifest " +
			"reference, its regime and every other field of the record have no row and " +
			"therefore no item",
	},
	{
		// What has happened to a candidate SINCE it was returned. The candidate
		// text is cold and its fate is warm, and that separation is the whole
		// ground the exception rests on (adr-2609021016272867; companion 8.3).
		Name: "FATE",
		Homes: []string{
			"repo:.abcd/work/issues/dispositions/rdi-201/dsp-201.md",
			"repo:.abcd/work/issues/dispositions/rdi-202/dsp-202.md",
			"repo:.abcd/work/issues/surprises/srp-1-a-surprise.md",
		},
		Count: 3,
		Why: "the researcher's judgement on a candidate — a disposition, an admission, a " +
			"surprise — is warm, and a reading that saw it would be reading what it exists " +
			"to inform",
	},
	{
		Name:  "DEFINITION",
		Homes: []string{"repo:agents/cold-reading-widening.md"},
		Count: 1,
		Why:   "itd-183: the reading definitions are the instrument, not its input",
	},
	{
		Name:  "INSTRUMENT",
		Homes: []string{"repo:evals/read_block.go"},
		Count: 1,
		Why:   "itd-183: the evals that guard this assembler are the instrument",
	},
	{
		Name:  "ASSEMBLER-SOURCE",
		Homes: []string{"repo:internal/core/reading/include.go"},
		Count: 1,
		Why:   "itd-183, ruling (18): a reading never receives the include table that decides what it sees",
	},
	{
		Name: "UNMATCHED-KIND",
		Homes: []string{
			"repo:.abcd/development/brief/01-product/07-working-notes.txt",
			"repo:working-notes.txt",
		},
		Count: 2,
		Why: "itd-183 assembler rule 1: inclusion is positive at FILE grain as well as at " +
			"directory grain. A file whose extension no include row names is absent for the " +
			"same reason a family the table does not name is — and the two homes are the two " +
			"grains it has to hold at, one inside a named source and one under the root rows, " +
			"which reach every undenied path in the tree",
	},
	{
		Name:  "NESTED-SECTION",
		Homes: []string{"repo:.abcd/development/specs/open/spc-1-a-design-record.md"},
		Count: 1,
		Why: "itd-183 exclusion list: an excluded heading never travels, and what it names is " +
			"the SECTION — everything down to the next heading of its own level or shallower. " +
			"A subsection of Open Questions is Open Questions",
	},
	{
		Name:  "BLOCK-SCALAR",
		Homes: []string{"repo:.abcd/development/intents/disciplines/itd-4-selection-criteria.md"},
		Count: 1,
		Why: "itd-183 exclusion list: origin never travels, and an excluded key's VALUE is the " +
			"warm thing. A block scalar carries it on continuation lines, blank lines inside " +
			"the block included, so a floor that took the key line alone would leave the value",
	},
	{
		Name:  "DENIED-CASE",
		Homes: []string{"repo:docs/Agents/note.md"},
		Count: 1,
		Why: "itd-183 assembler rule 1: the deny is structural over every path component, so a " +
			"component spelling a denied namespace in another case is the denied namespace",
	},
	{
		Name: "DRAFT-BODY",
		Homes: []string{
			"repo:.abcd/development/intents/drafts/itd-2-a-draft-intent.md",
			"repo:.abcd/development/intents/drafts/itd-5-a-promoted-draft.md",
		},
		Count:  2,
		ColdAt: []string{posEntailment},
		Why: "itd-183's drafts asymmetry: articulation precedes selection, so entailment " +
			"reads the candidate set and the reading asked to widen it does not. The second " +
			"home is a PROMOTED draft, minted in the reading route's shape " +
			"(itd-2609020625400169): same asymmetry, and the plant sits in the seed a " +
			"promotion writes rather than in a hand-typed press release",
	},
	{
		Name:  "DRAFT-ORIGIN",
		Homes: []string{"repo:.abcd/development/intents/drafts/itd-2-a-draft-intent.md"},
		Count: 1,
		Why: "the drafts asymmetry moves the BODY, never the excluded key: origin is warm " +
			"at every position, the entailment reading included",
	},
}

// hole is one relocation the negative-control variant performs: a plant taken
// out of its canonical home and put into material the include table admits.
//
// The corpus is a baseline plus a declared set of holes rather than two full
// trees, so the two variants cannot drift apart in everything the hole does not
// touch — a second full tree would have to be edited twice on every change.
type hole struct {
	// Class is the sentinel class relocated, which is what the eval's failure
	// must name.
	Class string
	// From is the canonical home the plant is removed from, repo-relative.
	From string
	// To is the included file it lands in, repo-relative. The file's replacement
	// content lives under testdata/cold-reading/holed/ at the same path.
	To string
	// Positions are the positions at which the relocated plant is reachable, and
	// therefore the positions the negative control must report this class at. It
	// exists because the four positions no longer read one corpus: at
	// comparative the include table admits the derived widening run's candidates
	// and the criteria discipline and nothing else, so a plant relocated into a
	// brief chapter is unreachable there and a control demanding it would demand
	// that a correct withdrawal had not happened. Empty means every position.
	Positions []string
	// Why states why a correct assembler passes the relocated plant through.
	Why string
}

// reachesAt reports whether the relocated plant is reachable at position p, and
// so whether the negative control must report this class there.
func (h hole) reachesAt(p string) bool {
	if len(h.Positions) == 0 {
		return true
	}
	for _, q := range h.Positions {
		if q == p {
			return true
		}
	}
	return false
}

// holes is the negative control: two plants relocated into positively included
// material. A correct assembler passes both through, so the eval must report
// exactly these two violations — which is the permanent proof that the
// assertion can fail.
// It is derived from nothing: emptying it would make the negative control pass
// with no control in it, which is why TestReadBlockCatchesAHoledFirewall refuses
// an empty table outright.
var holes = []hole{
	{
		Class:     "LEDGER-FRAMING",
		From:      ".abcd/.work.local/ledger/2026-08-30-declined-construals.md",
		To:        ".abcd/development/brief/01-product/06-framing.md",
		Positions: []string{posWidening, posEntailment, posDetection},
		Why:       "the 01-product chapters are admitted wholesale at every position that reads the brief",
	},
	{
		Class:     "TRANSCRIPT",
		From:      ".abcd/.work.local/scratch/session-notes.md",
		To:        "docs/reference/thing.md",
		Positions: []string{posWidening, posEntailment, posDetection},
		Why:       "the shipped tree's delivered documentation is admitted wholesale where the tree is read",
	},
	{
		// The comparative position's own control, and the corpus's only hole into
		// the LEDGER. Without it that position had no negative control at all —
		// its two admitted sources are unlike every other position's, so a plant
		// relocated into a brief chapter proves nothing there — and a firewall
		// with no control behind it is the shape this whole eval exists to
		// prevent (adr-2609021016272867).
		Class:     "FATE",
		From:      ".abcd/work/issues/dispositions/rdi-201/dsp-201.md",
		To:        ".abcd/work/issues/readings/" + derivedCandidateRun + "/rdi-301.md",
		Positions: []string{posComparative},
		Why: "the derived widening run's items are admitted at the comparative position and " +
			"projected to two body fields, so a fate relocated into one of those fields travels",
	},
}

// refusedPrefix is the shape a refusal plant's token takes. It is deliberately
// NOT sentinelPrefix: a sentinel class is counted against the baseline corpus by
// the anti-vacuity guard, and this material is never in the baseline — a corpus
// carrying it cannot be assembled at all, which is the behaviour under test.
const refusedPrefix = "ABCD-EVAL-REFUSED-"

// refusal is one shape the key-and-heading floor cannot REDACT and therefore
// refuses: a file the include table admits WHOLE, carrying an excluded heading
// in a form the section scan does not report, so the redactor has no span to
// delete and the fail-closed half is the only thing standing between the warm
// text and the bundle.
//
// It is what makes that half falsifiable. The floor's redacting half is
// falsified by a leak; its refusing half cannot be, because removing a refusal
// admits nothing new against a corpus with nothing to refuse. So the corpus
// grows a shape that MUST be refused, and the falsifier is the refusal going
// away — at which point the same binary exits 0 with the token in the bundle.
type refusal struct {
	// Name is the variant directory under testdata/cold-reading/refused/.
	Name string
	// Path is the repo-relative file replaced. Its replacement content lives at
	// refused/<Name>/<Path>.
	Path string
	// Token is the warm token the refusal keeps out of the bundle.
	Token string
	// Names are fragments the refusal message must carry, so a refusal for some
	// unrelated reason cannot be read as this one.
	Names []string
	// Falsifier is the assembler mutation that removes the refusal.
	Falsifier string
	// Why states what the redactor cannot do about the shape.
	Why string
}

// refusals is the refusal corpus: one entry per unredactable shape.
var refusals = []refusal{
	{
		Name:  "setext",
		Path:  ".abcd/development/brief/02-constraints/03-invariants.md",
		Token: refusedPrefix + "SETEXT",
		Names: []string{
			".abcd/development/brief/02-constraints/03-invariants.md",
			"underlines the excluded heading",
			"Audit Notes",
		},
		Falsifier: "delete the verifyRedaction call from redactExcluded",
		Why: "a setext underline turns the line above it into a heading, and the section scan " +
			"does not model setext at all, so the redactor sees prose where every renderer and " +
			"every reader sees the excluded heading",
	},
	{
		Name:  "rendered",
		Path:  ".abcd/development/brief/glossary/core/construal.md",
		Token: refusedPrefix + "RENDERED",
		Names: []string{
			".abcd/development/brief/glossary/core/construal.md",
			"still carries the excluded heading",
			"Audit Notes",
		},
		Falsifier: "make sameRendering return false",
		Why: "the redactor's own span lookup compares the normalised title exactly, which the " +
			"emphasis marks defeat; the equality that recognises `## **Audit Notes**` as the " +
			"excluded heading exists on the refusal path alone",
	},
	// The six shapes itd-194 adds. Each is a markdown document the include
	// table admits and the exclusion floor cannot resolve, and each is refused
	// rather than admitted unscanned — the mechanism that closes the
	// container-shape class nine rounds of added patterns did not (adr-56 rule
	// 1; brief invariant 16).
	//
	// Their Names carry the path and the shape's wording and NOT the excluded
	// thing the shape hides. That is a property of the refusals themselves: a
	// frontmatter shape this package cannot resolve is refused without the key
	// being named, because naming it would mean resolving it, which is the
	// guess these refusals exist not to make. What the shape hides is stated in
	// Why instead.
	{
		Name:  "fence-in-frontmatter",
		Path:  ".abcd/development/specs/open/spc-1-a-design-record.md",
		Token: refusedPrefix + "FENCE-IN-FRONTMATTER",
		Names: []string{
			".abcd/development/specs/open/spc-1-a-design-record.md",
			"a fence delimiter inside the frontmatter block",
		},
		Falsifier: "compute the fence mask over the whole document again, " +
			"instead of over the body from the line after the block closes",
		Why: "a fence delimiter inside the frontmatter toggled the fence mask, and the excluded-key " +
			"scan skips fenced lines — so the delimiter switched off the very refusal that exists " +
			"to catch a key the field reader cannot see, and `origin` travelled",
	},
	{
		Name:  "displaced-block",
		Path:  ".abcd/development/brief/01-product/06-framing.md",
		Token: refusedPrefix + "DISPLACED-BLOCK",
		Names: []string{
			".abcd/development/brief/01-product/06-framing.md",
			"displaced from line 0",
		},
		Falsifier: "delete the displacedFrontmatter call from verifyRedaction",
		Why: "the frontmatter block is recognised at line 0 alone, so a block preceded only by " +
			"blank lines, whitespace or an HTML comment is prose to this binary and frontmatter " +
			"to every reader of the bundle — and `origin` inside it travelled",
	},
	{
		Name:  "nested-mapping",
		Path:  ".abcd/development/intents/disciplines/itd-4-selection-criteria.md",
		Token: refusedPrefix + "NESTED-MAPPING",
		Names: []string{
			".abcd/development/intents/disciplines/itd-4-selection-criteria.md",
			"a mapping nested in a block sequence",
		},
		Falsifier: "delete nestedMappingRe's case from unresolvableFrontmatterShape",
		Why: "an excluded key nested inside a block-sequence entry is invisible to a field reader " +
			"that reports one value per top-level key, so there was no span to redact and the key " +
			"travelled; the floor refuses the nesting rather than learning the key's spelling",
	},
	{
		Name:  "flow-explicit-key",
		Path:  ".abcd/development/brief/02-constraints/03-invariants.md",
		Token: refusedPrefix + "FLOW-EXPLICIT-KEY",
		Names: []string{
			".abcd/development/brief/02-constraints/03-invariants.md",
			"an explicit key in a flow mapping",
		},
		Falsifier: "delete flowExplicitKeyRe's case from unresolvableFrontmatterShape",
		Why: "the flow scan reads a key that follows a brace or a comma directly, and YAML's " +
			"explicit-key indicator is not that shape, so the key behind it travelled; this and " +
			"the nested mapping are the one fix the two records ask for rather than two",
	},
	{
		Name:  "attribute-newline",
		Path:  ".abcd/development/brief/glossary/core/construal.md",
		Token: refusedPrefix + "ATTRIBUTE-NEWLINE",
		Names: []string{
			".abcd/development/brief/glossary/core/construal.md",
			"an attribute value that opens on the line after its equals sign",
		},
		Falsifier: "drop maskMarkupData's shape return, or stop raising it from verifyRedaction",
		Why: "the mask's blank skip after `=` is space and tab, so a value whose opening quote " +
			"sits on the next line is never found and the `>` inside it is read as the end of the " +
			"opening tag — the heading is judged as something else and the section under it travels",
	},
	{
		Name:  "unbounded-raw-heading",
		Path:  "docs/reference/thing.md",
		Token: refusedPrefix + "UNBOUNDED-RAW-HEADING",
		Names: []string{
			"docs/reference/thing.md",
			"a raw heading element that is never closed",
		},
		Falsifier: "delete the unboundedRawHeading call from verifyRedaction",
		Why: "an opener with neither a hard nor a soft bound has its title read over the whole " +
			"remainder of the document, which is how the heading sitting under it was admitted; " +
			"the refusal removes the shape and claims nothing about the scan's cost",
	},
}

// apply replaces the refusal's file in a materialised tree.
func (r refusal) apply(t *testing.T, root string) {
	t.Helper()
	copyFile(t,
		filepath.Join(refusedDir, r.Name, filepath.FromSlash(r.Path)),
		filepath.Join(root, filepath.FromSlash(r.Path)))
}

// carrier is one plant-bearing file the include table admits, and the positions
// it must actually reach the assembly at.
//
// The anti-vacuity guard below the corpus proves the plants are on disk and
// tracked. This proves their CONTENT is in the assembly, which is a different
// claim and the one every absence assertion here rests on. Two failures it has
// to see, and both have been watched: an assembler that stopped enumerating a
// whole class of source drops the files carrying WARM-FIELD, two of the three
// WARM-KEY plants and every candidate plant, and leaves a bundle that is still
// comfortably non-empty; and an assembler that emitted an EMPTY text for each
// item leaves the manifest describing exactly the same paths while the bundle
// carries nothing to leak. A floor of "any item" cannot see the first. A floor
// of "this path is in the manifest" cannot see the second.
//
// The list is transcribed from the same source as the include list above —
// itd-183's positive includes — never from the assembler.
type carrier struct {
	// Path is the repo-relative file, which must appear among the manifest's
	// item paths.
	Path string
	// Positions are the positions it must reach. Empty means every position.
	Positions []string
	// Markers are COLD strings drawn from the carrier's own travelling text: not
	// sentinels, so they belong in the bundle, and distinctive, so finding one in
	// the bundle's bytes means this file's content arrived rather than its name.
	//
	// Requiring a marker rather than the manifest path is the whole point. A
	// manifest names what an assembly SAYS it passed; an assembly that emitted an
	// empty text for every projected item would satisfy a path check exactly while
	// every absence assertion over those items asserted nothing at all.
	//
	// A marker pins ONLY the field it was drawn from, so a projected carrier draws
	// one from each field it pins, and the set of them is drawn from DISTINCT
	// fields. Every marker taken from the same section is the mistake that made
	// this floor land short twice: with all of them inside `## Press Release`,
	// narrowing the projection to that one field drops four of the five contracted
	// fields and every marker still arrives.
	Markers []string
	// Scan is the mark the manifest must carry for this carrier's items:
	// "parsed" for a document the exclusion floor examined, "unscanned" for one
	// it did not. Empty asserts nothing, which is what every carrier planted
	// before itd-194 does.
	//
	// It is what makes the disclosure falsifiable from the oracle's side. A
	// carrier that arrives whole proves the item travelled; only the mark says
	// whether an examination stood behind the manifest's exclusion assertion
	// over it, and a run that stamped every item `parsed` would otherwise be
	// green with the assertion resting on a scan that never ran.
	Scan string
	// Fields are the projected fields the manifest must record for this path.
	//
	// It exists because one of the five contracted fields cannot be pinned by a
	// marker AT ALL, and no number of markers could have found that. `spec_id`
	// projects to the bare string `spc-1`, which also travels inside the whole
	// spec file, so bytes.Contains is satisfied whether or not the projection
	// emitted it. The manifest's `field` column names what was projected, which is
	// the only place that distinction is visible — so the two checks are not
	// belt-and-braces, they reach different halves of the contract.
	Fields []string
	// Classes are the sentinel classes the file carries, so a missing carrier
	// names the assertions it silently disarmed. It may be empty for a carrier
	// whose job is to pin an include row rather than a plant.
	Classes []string
	// Why states what the file is doing in the assembly.
	Why string
}

// carriers pins the include list at content level: every row it names must
// arrive at each position, carrying its own text.
//
// Six rows are plant-bearing, so a lost carrier disarms a named assertion. Four
// carry no plant and exist only to make their include row falsifiable — the
// carrier floor is a PRESENCE assertion, not a leak assertion, so it reaches a
// row whose removal leaks nothing, which is a reach an absence assertion alone
// does not have.
var carriers = []carrier{
	{
		Path:    ".abcd/development/brief/01-product/01-press-release.md",
		Markers: []string{"The fixture product, stated as it presently stands."},
		Classes: []string{"WARM-KEY", "WARM-FIELD"},
		Why:     "a brief chapter admitted wholesale, carrying the production-mode key",
	},
	{
		Path:    ".abcd/development/brief/02-constraints/03-invariants.md",
		Markers: []string{"The core is transport agnostic."},
		Why:     "the constraints chapter, which carries no plant; the carrier is what makes its row falsifiable",
	},
	{
		Path:    ".abcd/development/brief/glossary/core/construal.md",
		Markers: []string{"What the situation is treated as."},
		Why:     "the glossary, which carries no plant; the carrier is what makes its row falsifiable",
	},
	{
		Path:    ".abcd/development/intents/disciplines/itd-4-selection-criteria.md",
		Markers: []string{"Six criteria, recorded as a standing commitment."},
		Classes: []string{"WARM-KEY", "WARM-FIELD"},
		Why:     "a discipline admitted whole, carrying the origin key",
	},
	{
		Path:    ".abcd/development/specs/open/spc-1-a-design-record.md",
		Markers: []string{"The mechanics the capability was built against."},
		Classes: []string{"WARM-KEY", "WARM-FIELD"},
		Why:     "a spec admitted whole, carrying the origin key and two excluded headings",
	},
	{
		// The only fixture record carrying all five contracted fields, so it is the
		// one that can pin the whole projection: four markers from four distinct
		// sections, and spec_id off the manifest because no marker can reach it.
		// Not at widening since itd-194: neither design document lists the
		// shipped intents in that position's object, so the row withdrew from it.
		Path:      ".abcd/development/intents/shipped/itd-1-a-shipped-intent.md",
		Positions: []string{posEntailment, posDetection},
		Markers: []string{
			"The promise, as it was made.",
			"- Given a fixture state, when the assembly runs, then the read-block holds.",
			"- Holds while the record is one repository.",
			"A positive include table over a hand-transcribed exclusion floor.",
		},
		Fields: []string{
			"Press Release", "Acceptance Criteria", "Scope Conditions", "Mechanism", "spec_id",
		},
		Classes: []string{"WARM-FIELD", "UNPROJECTED-SECTION"},
		Why:     "a shipped intent projected to its claim record, pinned at all five contracted fields",
	},
	{
		Path:      ".abcd/development/intents/drafts/itd-2-a-draft-intent.md",
		Positions: []string{posEntailment},
		Markers: []string{
			"The draft's press release, which the entailment reading reads.",
			"The draft's mechanism claim, which the entailment reading reads.",
		},
		Fields:  []string{"Press Release", "Mechanism"},
		Classes: []string{"DRAFT-ORIGIN", "UNPROJECTED-SECTION"},
		Why:     "the candidate set the entailment reading reads, carrying the origin key",
	},
	{
		Path:      ".abcd/development/intents/planned/itd-3-a-planned-intent.md",
		Positions: []string{posEntailment},
		Markers: []string{
			"A planned promise the entailment reading reads.",
			"The planned mechanism claim, which the entailment reading reads.",
		},
		Fields:  []string{"Press Release", "Mechanism", "spec_id"},
		Classes: []string{"UNPROJECTED-SECTION"},
		Why:     "the planned half of the candidate set, projected the same way",
	},
	// The four chapters itd-194 adds to brief current text, which both design
	// documents name as a reading's object (framework 7.2, companion 5.2) and
	// which the table admitted two of. One carrier per chapter, because a
	// carrier pins ONE path and each chapter's row has to be falsifiable on its
	// own: deleting the 05-internals row must fail, and a carrier under
	// 04-surfaces cannot see that.
	{
		Path:    ".abcd/development/brief/00-meta.md",
		Markers: []string{"ABCD-EVAL-CHAPTER-META travels as brief current text."},
		Scan:    "parsed",
		Why: "the brief's meta chapter, reached by the one row whose source is the brief " +
			"directory and whose match is that exact basename",
	},
	{
		Path:    ".abcd/development/brief/04-surfaces/01-reading.md",
		Markers: []string{"ABCD-EVAL-CHAPTER-SURFACES travels as brief current text."},
		Scan:    "parsed",
		Why:     "the brief's surfaces chapter, admitted as brief current text",
	},
	{
		Path:    ".abcd/development/brief/05-internals/01-packages.md",
		Markers: []string{"ABCD-EVAL-CHAPTER-INTERNALS travels as brief current text."},
		Scan:    "parsed",
		Why:     "the brief's internals chapter, admitted as brief current text",
	},
	{
		Path:    ".abcd/development/brief/06-delivery/01-shipping.md",
		Markers: []string{"ABCD-EVAL-CHAPTER-DELIVERY travels as brief current text."},
		Scan:    "parsed",
		Why:     "the brief's delivery chapter, admitted as brief current text",
	},
	{
		// The comparative channel's carrier. It is the one carrier whose path is
		// inside the LEDGER, and the only one admitted at a single position: at
		// comparative the derived widening run's items are the reading's object,
		// and at every other position they are the instrument's own output
		// (adr-2609021016272867).
		//
		// It carries the projection as well as the presence, through Fields: the
		// two body fields the widening position declares, and no third. A row
		// whose Fields were emptied would still pass a marker scan — the
		// configuration's text travels either way once the whole record does —
		// so the manifest's `field` column is what tells a live projection from a
		// dead one, exactly as it does on the shipped intent above.
		Path:      ".abcd/work/issues/readings/" + derivedCandidateRun + "/rdi-301.md",
		Positions: []string{posComparative},
		Markers:   []string{sentinelPrefix + "CANDIDATE"},
		Fields:    []string{"configuration", "what_admits_it"},
		Scan:      "parsed",
		Classes:   []string{"CANDIDATE"},
		Why: "one item of the derived widening run, projected to the two body fields a " +
			"comparative reading receives; the carrier is what makes the channel's own row " +
			"falsifiable, because deleting it leaks nothing and an absence oracle cannot see it",
	},
	{
		Path:      ".abcd/development/intents/disciplines/itd-191-the-selection-criteria.md",
		Positions: []string{posComparative},
		Markers:   []string{"Plausibility — the conjecture could work by a mechanism we can state."},
		Scan:      "parsed",
		Why: "the criteria discipline, which the assembler narrows the disciplines row to at " +
			"this position; a comparative reading with no criteria characterises against nothing",
	},
	{
		// The live leak's own shape: a Go test file carrying record-shaped
		// markdown with a literal `## Audit Notes` section. itd-194 does not
		// stop it travelling — both design documents name the shipped tree's
		// code and tests as a reading's object — it makes the manifest say, per
		// item, that no examination stood behind the exclusion assertion over
		// it (iss-2608301450065320; adr-56 as refined 2026-09-02).
		Path: "sitefixture_test.go",
		Markers: []string{
			"ABCD-EVAL-UNSCANNED-CARRIER travels whole and marked, never examined.",
		},
		Scan: "unscanned",
		Why: "a Go test file carrying a record-shaped page with an excluded heading in it; it " +
			"is what makes the unscanned mark falsifiable, because an assembler that stamped " +
			"every item `parsed` would leave this item's exclusion assertion resting on a scan " +
			"that never ran",
	},
	{
		Path:    "main.go",
		Markers: []string{"func main() {}"},
		Why:     "the shipped tree's source, which carries no plant; the carrier is what makes its row falsifiable",
	},
	{
		Path:    "fence.go",
		Markers: []string{"This block opens a fence at the left margin and never closes it."},
		Why: "a source file carrying an unterminated fence at the left margin; it is what " +
			"makes the body redaction's markdown-only scope falsifiable",
	},
	{
		Path:    "go.mod",
		Markers: []string{"module example.invalid/coldreadingfixture"},
		Why:     "the shipped tree's configuration, which carries no plant; the carrier is what makes its row falsifiable",
	},
	{
		Path:    "main_test.go",
		Markers: []string{"the fixture's own test file is corpus for the assembler and is never built"},
		Why: "the shipped tree's tests, admitted by the include table's basename-SUFFIX row " +
			"rather than by an extension; it carries no plant, and it is what makes that row " +
			"falsifiable at all — deleting the suffix row leaves the file admitted by the `.go` " +
			"row and changes nothing but the material class the manifest attests",
	},
}

// stake says what a missing carrier costs: the assertions it would have
// disarmed, or — for a carrier that pins an include row rather than a plant —
// that its row can no longer be falsified.
func (c carrier) stake() string {
	if len(c.Classes) == 0 {
		return "no plant depends on it, so nothing would leak, and its include row would " +
			"become unfalsifiable without a test noticing"
	}
	tokens := make([]string, 0, len(c.Classes))
	for _, name := range c.Classes {
		tokens = append(tokens, sentinelPrefix+name)
	}
	return "the " + strings.Join(tokens, " and ") + " assertion(s) over it would pass with nothing to catch"
}

// reachesAt reports whether the carrier must reach the assembly at position p.
//
// An empty Positions means every position that reads REPOSITORY MATERIAL, which
// is the three positions other than comparative. That default moved with the
// comparative channel and the move is a narrowing, not a loosening: at the
// comparative position the include table is the whole account and no source is
// admitted but the derived run's candidates and the criteria discipline
// (companion 7.2, R3; adr-2609021016272867), so a carrier asserted there would
// be asserting that a row which correctly withdrew is still arriving.
//
// The two carriers that DO reach comparative name it explicitly, so the channel
// is asserted rather than defaulted into.
func (c carrier) reachesAt(p string) bool {
	if len(c.Positions) == 0 {
		return p != posComparative
	}
	for _, q := range c.Positions {
		if q == p {
			return true
		}
	}
	return false
}

// The fixture corpus on disk.
const (
	corpusDir      = "testdata/cold-reading"
	baselineDir    = corpusDir + "/baseline"
	holedDir       = corpusDir + "/holed"
	refusedDir     = corpusDir + "/refused"
	fixtureHomeDir = corpusDir + "/home"
	// rootSHAPlaceholder is the directory name the fixture home carries where
	// the transcript store keys on the repository's root-commit sha. The sha
	// exists only once the fixture is committed, so materialisation substitutes
	// it.
	rootSHAPlaceholder = "ROOT_COMMIT_SHA"
)

// The two variants of the corpus.
const (
	variantBaseline = "baseline"
	variantHoled    = "holed"
)

// fixture is one materialised corpus: a committed repository, the HOME its run
// sees, and the root-commit sha the transcript store is keyed on.
type fixture struct {
	Root    string
	Home    string
	RootSHA string
	Variant string
}

// treeEdit is one further edit applied to a materialised tree BEFORE it is
// committed, so whatever it writes is tracked and the assembler can see it.
type treeEdit func(t *testing.T, root string)

// materialise copies the corpus into a temporary directory, applies the
// variant's holes and any further edits, commits it under a fixture identity in
// a reserved example domain, and plants the transcript store in a temporary HOME
// keyed on the resulting root-commit sha.
func materialise(t *testing.T, variant string, edits ...treeEdit) fixture {
	t.Helper()
	requireOracleTables(t)
	base := t.TempDir()
	f := fixture{
		Root:    filepath.Join(base, "repo"),
		Home:    filepath.Join(base, "home"),
		Variant: variant,
	}
	copyTree(t, baselineDir, f.Root)
	if variant == variantHoled {
		for _, h := range holes {
			if err := os.Remove(filepath.Join(f.Root, filepath.FromSlash(h.From))); err != nil {
				t.Fatalf("holed variant: removing %s: %v", h.From, err)
			}
			copyFile(t,
				filepath.Join(holedDir, filepath.FromSlash(h.To)),
				filepath.Join(f.Root, filepath.FromSlash(h.To)))
		}
	}
	for _, edit := range edits {
		edit(t, f.Root)
	}
	requireFixturePreset(t, f.Root)
	gitCommitFixture(t, f.Root)
	stampFixtureRuns(t, f.Root)
	f.RootSHA = rootCommit(t, f.Root)
	copyTree(t, fixtureHomeDir, f.Home)
	renamePlaceholder(t, f.Home, f.RootSHA)
	return f
}

// The two widening runs the corpus carries, and the item ids they hold. The
// comparative position derives the FIRST — its items carry no disposition and no
// admission — and the second exists so the derivation has something to reject
// and the exhaust has somewhere to live (adr-2609021016272867).
const (
	derivedCandidateRun      = "rdg-2608300900000003"
	dispositionedWideningRun = "rdg-2608300900000002"
)

var derivedCandidateItems = []string{"rdi-301", "rdi-302", "rdi-303"}

// stampFixtureRuns writes each planted widening run's COMMIT MARKER, naming the
// commit the fixture's records landed in.
//
// It runs after the commit, and the file is left untracked, because a run record
// names the commit its reading actually read: written before, it would name the
// commit before its own records existed. Nothing is lost by leaving it
// untracked — the durable readings family is admitted by no include row and
// excluded at every position, so it is outside both the walk and the dirty gate,
// and the candidate records the assembly reads are committed like every other
// admitted path.
//
// The type tag and the field names are TRANSCRIBED, like every other fact this
// oracle holds: no file under evals/ imports the assembler.
func stampFixtureRuns(t *testing.T, root string) {
	t.Helper()
	head := gitFixture(t, root, "rev-parse", "HEAD")
	for _, run := range []string{derivedCandidateRun, dispositionedWideningRun} {
		dir := filepath.Join(root, ".abcd", "development", "readings", run)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("stamping run %s: %v", run, err)
		}
		body := fmt.Sprintf(
			`{"_type":"abcd.reading.run/1","run_id":%q,"position":"widening","target_commit":%q}`+"\n",
			run, head)
		if err := os.WriteFile(filepath.Join(dir, "run.json"), []byte(body), 0o644); err != nil {
			t.Fatalf("stamping run %s: %v", run, err)
		}
	}
}

// requireFixturePreset holds the materialised corpus to the shape the
// invocation now depends on: ONE committed preset, named as declared above, with
// an entry at every assembling position.
//
// The eval invokes the assembler with a position and a target and nothing else,
// so what it is handed is decided entirely by this file. A fixture that lost the
// file, gained a second preset, or dropped a position would turn a firewall
// assertion into a refusal or a narrower corpus — either of which reads as green
// once the run that failed is the one that never assembled. It is read as raw
// JSON rather than through the assembler's own loader, like everything else this
// oracle checks: a fixture validated by the code under test confirms that code.
func requireFixturePreset(t *testing.T, root string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, ".abcd", "config", "reading-presets.json"))
	if err != nil {
		t.Fatalf("the fixture carries no committed preset file, so no position can assemble: %v", err)
	}
	var pf struct {
		Presets map[string]struct {
			Positions map[string]struct {
				Kinds []string `json:"kinds"`
			} `json:"positions"`
		} `json:"presets"`
	}
	if err := json.Unmarshal(raw, &pf); err != nil {
		t.Fatalf("the fixture's preset file does not decode: %v", err)
	}
	if len(pf.Presets) != 1 {
		t.Fatalf("the fixture's preset file holds %d presets; the invocation names none, so one "+
			"entry per position means one preset", len(pf.Presets))
	}
	entry, ok := pf.Presets[evalPresetName]
	if !ok {
		t.Fatalf("the fixture's preset is not named %q, which is the name this eval is written "+
			"against", evalPresetName)
	}
	for _, p := range assemblingPositions {
		if len(entry.Positions[p].Kinds) == 0 {
			t.Fatalf("the fixture's preset names no kind at %s, so that position assembles "+
				"nothing and every absence assertion there is vacuous", p)
		}
	}
}

// gitInit and the one commit. The identity is invented and its domain is
// reserved for examples, so nothing about the machine the eval runs on reaches
// the fixture's history.
func gitCommitFixture(t *testing.T, root string) {
	t.Helper()
	ident := []string{"-c", "user.name=abcd cold-reading fixture", "-c", "user.email=fixture@example.invalid"}
	steps := [][]string{
		{"init", "-q", "-b", "main"},
		append(append([]string{}, ident...), "add", "-A"),
		append(append([]string{}, ident...), "commit", "-q", "-m", "the cold-reading fixture corpus"),
	}
	for _, args := range steps {
		gitFixture(t, root, args...)
	}
}

// gitFixture runs one git command in the fixture repository, failing the test
// on error.
//
// Every command runs under gittest.Env, the repository's shared hermetic-git
// environment (iss-28): the ambient GIT_DIR/GIT_WORK_TREE and the machine's
// global git configuration are stripped, so nothing about the machine running
// the eval can reach the fixture and no fixture command can be redirected onto
// the real repository. That helper reaches internal/gitutil and nothing else, so
// it costs the oracle none of its independence from the assembler.
func gitFixture(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in the fixture: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// rootCommit returns the fixture's root-commit sha, which is what the
// session-transcript store is keyed on.
func rootCommit(t *testing.T, root string) string {
	t.Helper()
	sha := gitFixture(t, root, "rev-list", "--max-parents=0", "HEAD")
	if sha == "" {
		t.Fatal("the fixture repository reports no root commit")
	}
	return sha
}

// trackedFiles returns the fixture repository's tracked paths. The assembler
// intersects its walk with this set, so a plant git refused to track is a plant
// the assembler could never have leaked — a corpus that silently tests nothing.
func trackedFiles(t *testing.T, root string) map[string]bool {
	t.Helper()
	out := gitFixture(t, root, "ls-files", "-z")
	set := map[string]bool{}
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			set[p] = true
		}
	}
	return set
}

// renamePlaceholder swaps the transcript store's placeholder directory for the
// fixture's own root-commit sha.
func renamePlaceholder(t *testing.T, home, sha string) {
	t.Helper()
	parent := filepath.Join(home, ".abcd", "transcripts")
	from := filepath.Join(parent, rootSHAPlaceholder)
	if err := os.Rename(from, filepath.Join(parent, sha)); err != nil {
		t.Fatalf("keying the fixture transcript store on the root commit: %v", err)
	}
}

// copyTree copies a fixture directory wholesale. It is a walk rather than a
// shell copy so the eval behaves the same on every machine that runs it.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, path)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if merr := os.MkdirAll(filepath.Dir(target), 0o755); merr != nil {
			return merr
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("materialising %s: %v", src, err)
	}
}

// copyFile writes one fixture file over its destination, creating parents.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading the fixture file %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("creating the parent of %s: %v", dst, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", dst, err)
	}
}
