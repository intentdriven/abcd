//go:build smoke || coldreading

package evals

// The coverage matrix: for every rule the assembler's contract carries, the
// mutation that removes it and the plant that dies when it does.
//
// It exists because the previous two review rounds found the same shape twice —
// an assertion with nothing underneath it — and the answer to that is not
// another floor. A floor protects an assertion from vacuity in the one dimension
// somebody thought of; a plant that dies when a rule is removed protects it in
// the dimension that matters, which is the rule's own. So this table is the
// eval's real claim, and everything above it is machinery for making the table
// true.
//
// Two things make it more than a comment (itd-195). Every row names sentinel
// classes and refusal plants that must exist and must be planted, and every
// declared class and refusal must be named by at least one row — so a plant no
// rule depends on, and a rule naming a plant that has gone, both fail here. And
// a row that no mutation can falsify carries its reason in Gap rather than being
// quietly omitted: a declared gap is worth more than a floor that turns out to
// be one dimension short.
//
// Its own disclosed limit is what a fidelity audit then found through: the rows
// are checked against the plants, never against the assembler, so a rule the
// matrix simply does not NAME is invisible here and the declared-gap discipline
// never engages on it. Six were absent that way, and two of them were the ones
// that mattered — the match list, whose mutation leaves the fixture manifests
// byte-identical, and the redaction verifier, whose call could be deleted
// outright with the lane green because a leak cannot reach a rule enforced by
// refusing. Both are rows below now, with corpus behind them. The limit stands:
// closing it needs a human reading the include table against this list.
//
// What it does NOT claim: that each Falsifier has been executed on every run.
// The ones marked in the record as watched red have been; the rest are the
// mutation each row is written against, stated so the next reader can run it.
//
// # What this matrix enumerates, and what it does not
//
// It enumerates the assembler's READ-BLOCK: the include table's rows and their
// projections, the exclusion floor, the structural deny, and the mechanisms that
// decide what reaches the assembled input. That is spc-64's subject — what a
// reading sees — and the boundary is deliberate rather than a stopping point
// nobody chose.
//
// It deliberately does NOT enumerate the assembler's refusals about the
// INVOCATION and the TREE, which decide whether a run happens at all rather than
// what it passes: refuseSelfAdmittingOutDir, refuseDirtyIncludedPaths,
// resolveTarget and the closed position set, MaxFileBytes, the absent
// record-lint.json refusal, requireConfiguredStores, the walk-row
// must-be-a-directory refusal, and the symlinked-leaf rule. Each is covered by
// internal/core/reading's own tests, and none of them can put warm content into
// a bundle — the failure this eval exists to catch.
//
// Two read-block properties are also out: that a bundle item carries no
// repository path (brief invariant 15) and that the manifest's own shape holds.
// Both are properties of the ARTEFACT rather than of what was selected into it,
// spc-64 names three assertions and neither is among them, and both are held by
// TestBundleCarriesNoRepositoryPath in the assembler's package. They are named
// here so their absence reads as a decision rather than an oversight.

import (
	"sort"
	"strings"
	"testing"
)

// How the eval notices a rule's removal.
const (
	// caughtLeak: a planted token reaches the bundle and assertion 1 names it.
	caughtLeak = "sentinel absence names the leaked class"
	// caughtFamily: an item's manifest path lands in a refused family, or under
	// no individually named include, and assertion 3 names it.
	caughtFamily = "family absence names the path"
	// caughtCarrier: a plant-bearing file, or its content, stops arriving and the
	// carrier floor names the assertions that would have gone vacuous.
	caughtCarrier = "the carrier floor names the disarmed assertion"
	// caughtControl: the negative control stops reporting one of its two classes.
	caughtControl = "the holed control reports the wrong class set"
	// caughtKind: the manifest attests an item's material class, and the
	// hand-transcribed kind oracle disagrees with it.
	caughtKind = "the kind oracle names the item whose material class is mis-attested"
	// caughtRefusal: the assembler refuses the run and the verb exits non-zero,
	// which the eval reports as a failed assembly rather than as a leak.
	caughtRefusal = "the assembly is refused and the verb exits non-zero"
	// caughtUnrefused: the inverse, and the only mechanism that reaches the
	// floor's fail-closed half. A shape the redactor cannot delete a span for
	// STOPS being refused, so the refusal corpus reports an assembly that
	// succeeded — with the warm token in the bundle it reads back.
	caughtUnrefused = "the refusal corpus reports an assembly that was not refused"
	// caughtScan: the manifest's per-item scan mark disagrees with the
	// hand-transcribed expectation, so an assembly that stopped disclosing what
	// the exclusion floor did not examine is named. It reaches a claim no leak
	// can: an item marked `parsed` that no scan ran over leaks nothing on this
	// corpus and still leaves the manifest's exclusion assertion resting on an
	// examination that did not happen (itd-194).
	caughtScan = "the scan-mark oracle names the item whose examination is mis-attested"
)

// coverageRow is one rule of the assembler's contract.
type coverageRow struct {
	// Rule is the rule in the record's own terms.
	Rule string
	// Falsifier is the mutation to the assembler that removes the rule.
	Falsifier string
	// Caught is how this eval notices. Empty exactly when Gap is set.
	Caught string
	// Classes are the sentinel classes that die or leak. It may be empty for a
	// rule this eval catches by path rather than by plant.
	Classes []string
	// Refusals are the refusal-corpus entries that stop being refused. It is the
	// same discipline as Classes, over the corpus that reaches the exclusion
	// floor's fail-closed half: a refusal plant no rule depends on, and a rule
	// naming a refusal plant that has gone, both fail the check below.
	Refusals []string
	// Gap is why no mutation falsifies this rule against this corpus. Set exactly
	// when Caught is empty. A row carrying one is declared unfalsifiable
	// coverage, not an open hole to be discovered later.
	Gap string
}

// coverage is the matrix. One row per rule, positive and negative alike.
var coverage = []coverageRow{
	// ---- the include table's positive rows ----
	{
		Rule:      "brief/01-product is admitted whole at every position",
		Falsifier: "delete the 01-product row from Table",
		Caught:    caughtCarrier,
		Classes:   []string{"WARM-KEY", "WARM-FIELD"},
	},
	{
		Rule:      "brief/02-constraints is admitted whole at every position",
		Falsifier: "delete the 02-constraints row from Table",
		Caught:    caughtCarrier,
	},
	{
		Rule:      "brief/glossary is admitted whole at every position",
		Falsifier: "delete the glossary row from Table",
		Caught:    caughtCarrier,
	},
	{
		Rule:      "intents/disciplines is admitted whole at every position",
		Falsifier: "delete the disciplines row from Table",
		Caught:    caughtCarrier,
		Classes:   []string{"WARM-KEY", "WARM-FIELD"},
	},
	{
		Rule:      "intents/shipped is admitted at entailment and detection",
		Falsifier: "delete the shipped row from Table",
		Caught:    caughtCarrier,
		Classes:   []string{"WARM-FIELD", "UNPROJECTED-SECTION"},
	},
	{
		Rule: "the shipped intent's projection is POSITIVE at field granularity: a section " +
			"the field list does not name stays behind",
		Falsifier: "delete `Fields: intentProjection` from the shipped row",
		Caught:    caughtLeak,
		Classes:   []string{"UNPROJECTED-SECTION"},
	},
	{
		Rule:      "specs are admitted whole at every position",
		Falsifier: "delete the specs row from Table",
		Caught:    caughtCarrier,
		Classes:   []string{"WARM-KEY", "WARM-FIELD"},
	},
	{
		Rule:      "intents/drafts are admitted at entailment",
		Falsifier: "delete the drafts row from Table",
		Caught:    caughtCarrier,
		Classes:   []string{"DRAFT-ORIGIN", "UNPROJECTED-SECTION"},
	},
	{
		Rule:      "the draft projection is positive at field granularity",
		Falsifier: "delete `Fields: intentProjection` from the drafts row",
		Caught:    caughtLeak,
		Classes:   []string{"UNPROJECTED-SECTION"},
	},
	{
		Rule:      "intents/planned are admitted at entailment",
		Falsifier: "delete the planned row from Table",
		Caught:    caughtCarrier,
		Classes:   []string{"UNPROJECTED-SECTION"},
	},
	{
		Rule:      "the planned projection is positive at field granularity",
		Falsifier: "delete `Fields: intentProjection` from the planned row",
		Caught:    caughtLeak,
		Classes:   []string{"UNPROJECTED-SECTION"},
	},
	{
		Rule:      "the shipped tree's delivered documentation and root prose are admitted",
		Falsifier: "delete the root `.md` row from Table",
		Caught:    caughtControl,
		Classes:   []string{"TRANSCRIPT"},
	},
	{
		Rule:      "the shipped tree's source is admitted",
		Falsifier: "delete the root `.go` row from Table",
		Caught:    caughtCarrier,
	},
	{
		Rule:      "the shipped tree's configuration and build files are admitted",
		Falsifier: "delete the root config row from Table",
		Caught:    caughtCarrier,
	},

	{
		Rule: "inclusion is positive at FILE grain too: a file whose extension no row's " +
			"Match list names is absent, inside a named Source and under the root rows alike",
		Falsifier: "make Row.matches return true unconditionally",
		Caught:    caughtLeak,
		Classes:   []string{"UNMATCHED-KIND"},
		// The stated falsifier fires on EITHER home. The two exist because the
		// rule holds at two grains and a row is a claim about the rule, not about
		// the cheapest way to trip it: a row naming a Source reaches only inside
		// that directory, while the three root rows reach every undenied path in
		// the tree, so a narrower mutation — dropping the match list from the root
		// rows alone — is falsified by the root home and by nothing else.
		//
		// What made the rule invisible without either home is that against a
		// corpus where every file already carries a named extension the mutation
		// leaves the manifests byte-identical, which is how it stayed unnamed here
		// through three review rounds.
	},

	{
		Rule: "the shipped tree's TESTS are labelled apart from its source, by a basename " +
			"SUFFIX rather than by an extension, and the suffix row is ordered above the " +
			"source row so it owns the paths it reaches",
		Falsifier: "delete the `_test.go` MatchSuffix row from Table (or order it below the `.go` row)",
		Caught:    caughtKind,
		// Neither a leak nor a carrier can reach this rule, which is why it went
		// unnamed: `path.Ext("main_test.go")` is ".go", so the source row admits
		// the file either way. The mutation changes no path in any fixture
		// manifest and no byte of any bundle item's text — only the class the
		// manifest attests for it — so the only thing that can catch it is
		// something that READS that attestation.
	},
	{
		Rule: "a manifest item's `kind` names the material class the record gives its path, " +
			"and the bundle item beside it carries the same class",
		Falsifier: "emit a fixed Kind for every ManifestItem (or swap two rows' Kind in Table)",
		Caught:    caughtKind,
		// itd-198 added the per-item kind so a size report is checkable against
		// the manifest rather than asserted beside it. Nothing read it: brief
		// invariant 16 says an attestation states no more than its examination
		// establishes, and until this row an unexamined class was exactly that.
		// The bundle half is separate because the two artefacts travel apart —
		// the manifest to the auditor, the bundle to the reader — so a class
		// correct in one and wrong in the other is a reading told its material
		// is something the auditor's copy denies.
	},
	{
		Rule: "the intent projection is the FIVE contracted fields: press release, " +
			"acceptance criteria, scope conditions, mechanism and spec_id",
		Falsifier: "narrow intentProjection to []string{\"Press Release\"}",
		Caught:    caughtCarrier,
		Classes:   []string{"WARM-FIELD", "UNPROJECTED-SECTION", "DRAFT-ORIGIN"},
	},

	// ---- the per-position asymmetry ----
	{
		Rule: "a reading's object excludes what it exists to change: the widening, " +
			"comparative and detection readings do not see the drafts",
		Falsifier: "widen the drafts row's Positions to allPositions AND delete the " +
			"position-scoped drafts row from Exclusions",
		Caught:  caughtLeak,
		Classes: []string{"DRAFT-BODY"},
	},
	{
		Rule: "the same asymmetry for the planned intents",
		Falsifier: "widen the planned row's Positions to allPositions AND delete the " +
			"position-scoped planned row from Exclusions",
		Caught: caughtFamily,
	},

	// ---- the exclusion floor: keys ----
	{
		Rule:      "the `origin` frontmatter key never travels",
		Falsifier: "delete the origin row from Exclusions",
		Caught:    caughtLeak,
		Classes:   []string{"WARM-KEY", "DRAFT-ORIGIN"},
	},
	{
		Rule:      "the production-mode frontmatter key never travels",
		Falsifier: "delete the production_mode row from Exclusions",
		Caught:    caughtLeak,
		Classes:   []string{"WARM-KEY"},
	},

	{
		Rule: "an excluded key's VALUE goes with it: a block scalar's continuation lines, " +
			"and the blank lines inside the block, are part of the value and are dropped too",
		Falsifier: "delete redactExcluded's indented-continuation drop",
		Caught:    caughtLeak,
		Classes:   []string{"WARM-KEY", "BLOCK-SCALAR"},
		// Two plants inside one block, either side of a blank line, so the two
		// halves of the rule are separable: the continuation drop as a whole, and
		// the narrower reading that stops at the first blank line. Under the
		// stated falsifier both leak; under a drop that ends at the first blank
		// line the second alone does.
	},

	// ---- the exclusion floor: headings ----
	{
		Rule:      "an Audit Notes heading never travels",
		Falsifier: "delete the Audit Notes row from Exclusions",
		Caught:    caughtLeak,
		Classes:   []string{"WARM-FIELD"},
	},
	{
		Rule:      "a scope-condition disposition never travels",
		Falsifier: "delete the Scope Condition Dispositions row from Exclusions",
		Caught:    caughtLeak,
		Classes:   []string{"WARM-FIELD"},
	},
	{
		Rule:      "an Open Questions heading never travels",
		Falsifier: "delete the Open Questions row from Exclusions",
		Caught:    caughtLeak,
		Classes:   []string{"WARM-FIELD"},
	},
	{
		Rule:      "a Why This Matters heading never travels",
		Falsifier: "delete the Why This Matters row from Exclusions",
		Caught:    caughtLeak,
		Classes:   []string{"WARM-FIELD"},
	},

	{
		Rule: "an excluded heading owns its whole SECTION: everything down to the next " +
			"heading of its own level or shallower, subsections included",
		Falsifier: "make sectionSpan end at the next heading of ANY level",
		Caught:    caughtLeak,
		Classes:   []string{"NESTED-SECTION"},
	},
	{
		Rule: "the key-and-heading floor is FAIL-CLOSED: a file still carrying an excluded " +
			"shape after redaction refuses the run rather than travelling under a manifest " +
			"asserting the shape was refused",
		Falsifier: "delete the verifyRedaction call from redactExcluded",
		Caught:    caughtUnrefused,
		Refusals:  []string{"setext", "rendered"},
		// This half cannot be falsified by a leak, and that is why it went
		// unnamed: removing a refusal admits nothing new against a corpus with
		// nothing to refuse, so the call could be deleted outright with this lane
		// green. It needs a corpus that HAS something to refuse — a file admitted
		// whole carrying an excluded heading the section scan does not report —
		// and the refusal corpus is that, held in its own variant because a
		// baseline carrying one cannot be assembled at all.
	},
	{
		Rule: "a heading is the excluded heading however it is SPELLED: two titles that " +
			"render as the same heading are the same heading",
		Falsifier: "make sameRendering return false",
		Caught:    caughtUnrefused,
		Refusals:  []string{"rendered"},
		// The narrower of the two falsifiers over the same plant, which is what
		// tells this rule apart from the fail-closed rule above: deleting the
		// verifyRedaction call unrefuses both refusal plants, while this unrefuses
		// only the emphasised one.
	},

	// ---- the exclusion floor: families ----
	//
	// Each of these needs a TWO-part falsifier, and that is a property of the
	// assembler rather than a shortcut here: exclusion by absence from the
	// positive walk and the fail-closed assertExclusions gate are two independent
	// mechanisms, so removing either alone changes no output.
	{
		Rule:      "the brief's evidence chapter is verdict material and never travels",
		Falsifier: "add an include row for brief/03-evidence and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DELIBERATION"},
	},
	{
		Rule:      "the decisions family never travels",
		Falsifier: "add an include row for development/decisions and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DELIBERATION"},
	},
	{
		Rule:      "roadmap/rfcs never travel",
		Falsifier: "add an include row for roadmap/rfcs and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DELIBERATION"},
	},
	{
		Rule:      "superseded intents never travel",
		Falsifier: "add an include row for intents/superseded and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DELIBERATION"},
	},
	{
		Rule:      "plans never travel",
		Falsifier: "add an include row for development/plans and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DELIBERATION"},
	},
	{
		Rule:      "research notes never travel",
		Falsifier: "add an include row for research/notes and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DELIBERATION"},
	},
	{
		Rule: "the issue ledger never travels in any state, reading records, dispositions, " +
			"admission and selection grounds, reframe records and the lapse log included",
		Falsifier: "add an include row under work/issues and delete the work/issues Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DECISION", "EXHAUST", "GROUNDS", "LEDGER-REFRAME"},
	},
	{
		Rule:      "the shared decision log never travels",
		Falsifier: "add an include row for work/DECISIONS.md and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DECISION"},
	},
	{
		Rule:      "the readings family — the instrument's own output — never travels",
		Falsifier: "add an include row for development/readings and delete its Exclusions row",
		Caught:    caughtRefusal,
		Classes:   []string{"EXHAUST"},
	},
	{
		Rule:      "the local ledger tier never travels: framing traces and declined construals",
		Falsifier: "add an include row for .work.local/ledger",
		Caught:    caughtLeak,
		Classes:   []string{"LEDGER-FRAMING"},
	},
	{
		Rule:      "the local scratch tier never travels: transcript-class text inside the repository",
		Falsifier: "add an include row for .work.local/scratch",
		Caught:    caughtLeak,
		Classes:   []string{"TRANSCRIPT"},
	},
	{
		Rule:      "the reading definitions are the instrument and never travel",
		Falsifier: "add an include row for agents and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"DEFINITION"},
	},
	{
		Rule:      "the evals that guard this assembler never travel",
		Falsifier: "add an include row for evals and delete its Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"INSTRUMENT"},
	},
	{
		Rule:      "a reading never receives the include table that decides what it sees",
		Falsifier: "empty denyPrefixes and delete the internal/core/reading Exclusions row",
		Caught:    caughtLeak,
		Classes:   []string{"ASSEMBLER-SOURCE"},
	},
	{
		Rule:      "the session-transcript store never travels, wherever it sits",
		Falsifier: "none: the assembler holds no code path that walks HOME, so there is nothing to remove",
		Gap: "unfalsifiable by construction for the default store, and deliberately so. The " +
			"plant in the fixture HOME is what makes the class REACHABLE — it is keyed on " +
			"the fixture's own root-commit sha, exactly where the store would be — so the " +
			"day a walk over HOME is added, this row becomes falsifiable with no change to " +
			"the corpus. The in-repo half is a different matter and IS falsifiable: a " +
			"checkout declared in ~/.abcd/local-transcript-roots keeps its store under " +
			".abcd/.work.local/, so the repo-side plant in this class is caught by the " +
			"`.abcd` deny and the local-tier exclusion, which the structural rows below " +
			"falsify directly (iss-95)",
		Classes: []string{"TRANSCRIPT"},
	},

	// ---- structural mechanisms ----
	{
		Rule:      "the `.abcd` namespace is denied structurally, from each row's own Source downward",
		Falsifier: "drop \".abcd\" from denySegments",
		Caught:    caughtRefusal,
	},
	{
		Rule: "the body redaction is scoped to markdown and runs on nothing else, because a " +
			"markdown section scan over a source file refuses a raw fence at the left margin",
		Falsifier: "delete redactExcluded's non-markdown early return",
		Caught:    caughtRefusal,
		// The oracle and the assembler DISAGREE about this scope, and the assembler
		// is right. Its key exclusion binds to markdown alone; the item-text scan in
		// checkFieldAbsence binds to every item whatever its kind. itd-183 names the
		// signal as a FRONTMATTER key, which is a record shape, so a top-level
		// `origin:` in a YAML or TOML config file is not the excluded thing and
		// redacting it would be wrong. The oracle is deliberately the broader of the
		// two — the safe direction for a transcribed oracle, and the reason this row
		// is a refusal rather than a leak: a corpus file that made the difference
		// visible would have this eval report a violation over correct behaviour. No
		// tracked non-markdown file the root rows admit carries either key today.
	},
	{
		Rule:      "the `agents` namespace is denied structurally, from each row's own Source downward",
		Falsifier: "drop \"agents\" from denySegments",
		Caught:    caughtRefusal,
	},
	{
		Rule:      "the structural deny binds each path component CASE-INSENSITIVELY",
		Falsifier: "make segmentDenied compare with == rather than strings.EqualFold",
		Caught:    caughtLeak,
		Classes:   []string{"DENIED-CASE"},
		// The plant is at `docs/Agents/`, not at the root: the four denied
		// segments all exist in the corpus in lower case, and a same-name
		// differently-cased sibling beside one of them is not a tree a
		// case-insensitive filesystem can hold. A component two levels down
		// exercises the same predicate, because the deny binds every component
		// rather than the first.
	},
	{
		Rule:      "the `evals` namespace is denied structurally, from each row's own Source downward",
		Falsifier: "drop \"evals\" from denySegments",
		Caught:    caughtRefusal,
	},
	{
		Rule:      "where two rows reach one path, the FIRST row owns the projection applied to it",
		Falsifier: "make the claimed-path check keep the last row rather than the first",
		Gap: "no fixture path is admitted by two rows, so there is no tie for the rule to " +
			"break. The record rows name paths under .abcd, which every root row denies " +
			"structurally, so constructing one needs a record store outside .abcd — a " +
			"repository shape this corpus does not have and the record does not use",
	},
	{
		Rule:      "the `.git` namespace is denied structurally",
		Falsifier: "drop \".git\" from denySegments",
		Gap: "the walk is intersected with the tracked set and git reports none of its own " +
			"object database as tracked, so nothing under .git can be admitted however the " +
			"deny is spelled; the segment is belt to that braces and its removal leaks nothing",
	},
	{
		Rule:      "the record graph is what enumerates the record rows",
		Falsifier: "point storeNodeType at a node type the graph does not report",
		Caught:    caughtCarrier,
		Classes:   []string{"WARM-KEY", "WARM-FIELD", "UNPROJECTED-SECTION", "DRAFT-ORIGIN"},
	},
	{
		Rule:      "the bundle carries the text of what it passed, not merely a manifest naming it",
		Falsifier: "emit an empty Text for every BundleItem",
		Caught:    caughtCarrier,
		Classes:   []string{"WARM-KEY", "WARM-FIELD", "UNPROJECTED-SECTION", "DRAFT-ORIGIN"},
	},
	{
		Rule:      "assertExclusions refuses an item under a path-shaped exclusion the manifest asserts",
		Falsifier: "delete the assertExclusions call",
		Gap: "unfalsifiable in isolation: inclusion is positive, so removing the fail-closed " +
			"half admits nothing new and the output does not change. It is the second half " +
			"of every two-part family falsifier above, where its removal is exactly what " +
			"lets the leak through",
	},
	{
		Rule:      "refuseOwnArtefact refuses an admitted file carrying this assembler's own artefact tag",
		Falsifier: "delete the refuseOwnArtefact call",
		Gap: "unfalsifiable in isolation for the same reason: the fixture's prior manifest is " +
			"already denied by path, so nothing reaches the scan. It is what turns the " +
			"readings-family falsifier above into a refusal rather than a leak",
	},
	{
		Rule:      "the walk is intersected with the tracked set, so an untracked file never travels",
		Falsifier: "drop the trackedSet intersection",
		Gap: "the corpus commits every plant, so there is no untracked file to admit. The " +
			"complementary claim IS asserted: the anti-vacuity guard fails if a declared " +
			"plant is untracked, because the assembler would then never have seen it",
	},
	{
		Rule:      "an excluded frontmatter key is refused at any indentation, not only at column 0",
		Falsifier: "anchor excludedKeyLine at column 0",
		Gap: "unfalsifiable through this eval: the assembler FAILS CLOSED on an indented " +
			"excluded key that survives redaction, so a corpus carrying one makes the " +
			"assembly exit non-zero and the eval's own depth-agnostic branch is never " +
			"reached. The branch is kept because spc-64 asks for it and it costs nothing " +
			"to be right if that refusal ever softens, not because it has caught anything",
	},
	// ---- itd-194: the six shapes the floor refuses rather than admits ----
	//
	// Every one of these is reached ONLY by the refusal corpus. A refusal
	// removed against a corpus with nothing to refuse changes no output, so the
	// falsifier here is the plant ceasing to be refused — at which point the
	// same binary exits 0 with the warm token in the bundle.
	{
		Rule: "a fence delimiter inside the frontmatter block is refused, and the fence mask " +
			"cannot be toggled from inside the block",
		Falsifier: "compute the fence mask over the whole document again, " +
			"instead of over the body from the line after the block closes",
		Caught:   caughtUnrefused,
		Refusals: []string{"fence-in-frontmatter"},
	},
	{
		Rule: "a delimited block preceded only by blank lines, whitespace or an HTML comment " +
			"is refused as displaced from line 0",
		Falsifier: "delete the displacedFrontmatter call from verifyRedaction",
		Caught:    caughtUnrefused,
		Refusals:  []string{"displaced-block"},
	},
	{
		Rule:      "a compact mapping nested in a block sequence is refused, whatever the key is named",
		Falsifier: "delete nestedMappingRe's case from unresolvableFrontmatterShape",
		Caught:    caughtUnrefused,
		Refusals:  []string{"nested-mapping"},
	},
	{
		Rule:      "an explicit key in a flow mapping is refused, whatever the key is named",
		Falsifier: "delete flowExplicitKeyRe's case from unresolvableFrontmatterShape",
		Caught:    caughtUnrefused,
		Refusals:  []string{"flow-explicit-key"},
	},
	{
		Rule:      "an attribute value that opens on the line after its equals sign is refused",
		Falsifier: "drop maskMarkupData's shape return, or stop raising it from verifyRedaction",
		Caught:    caughtUnrefused,
		Refusals:  []string{"attribute-newline"},
	},
	{
		Rule: "a raw heading opener that reaches the end of the document with no bound is " +
			"refused, and a CRLF blank line bounds an element as an LF one does",
		Falsifier: "delete the unboundedRawHeading call from verifyRedaction",
		Caught:    caughtUnrefused,
		Refusals:  []string{"unbounded-raw-heading"},
	},

	// ---- itd-194: the widening object, and what the manifest says it examined ----
	{
		Rule:      "the widening reading does not see the shipped intents",
		Falsifier: "widen the shipped row's Positions to allPositions and delete the widening-scoped shipped row from Exclusions",
		Caught:    caughtFamily,
	},
	{
		Rule:      "an item from an unscanned row carries the mark and a parsed item does not",
		Falsifier: "stamp `ScanParsed` on every candidate",
		Caught:    caughtScan,
	},

	// ---- itd-194: brief current text ----
	//
	// One row per chapter, because a carrier pins one path and each chapter's
	// include row has to be falsifiable on its own.
	{
		Rule:      "brief/00-meta.md is admitted as a brief section at every position the brief rows admit",
		Falsifier: "delete the row",
		Caught:    caughtCarrier,
	},
	{
		Rule:      "brief/04-surfaces is admitted as brief sections at every position the brief rows admit",
		Falsifier: "delete the row",
		Caught:    caughtCarrier,
	},
	{
		Rule:      "brief/05-internals is admitted as brief sections at every position the brief rows admit",
		Falsifier: "delete the row",
		Caught:    caughtCarrier,
	},
	{
		Rule:      "brief/06-delivery is admitted as brief sections at every position the brief rows admit",
		Falsifier: "delete the row",
		Caught:    caughtCarrier,
	},

	// The comparative channel (adr-2609021016272867; spc-2609020626039834). It
	// is the one place the assembler reaches INTO the ledger, so every row below
	// is a limit on that reach rather than a licence for it.
	{
		Rule: "the derived candidate run's items are admitted at the comparative position, " +
			"projected to the configuration and what admits it",
		Falsifier: "delete the candidate row from Table",
		Caught:    caughtCarrier,
		Classes:   []string{"CANDIDATE"},
	},
	{
		Rule:      "the candidate row is admitted at no other position",
		Falsifier: "add the three other positions to the candidate row",
		Caught:    caughtFamily,
		Classes:   []string{"CANDIDATE"},
	},
	{
		Rule:      "the candidate projection drops the item's envelope",
		Falsifier: "empty the candidate row's Fields, so the whole record travels",
		Caught:    caughtLeak,
		Classes:   []string{"ENVELOPE"},
	},
	{
		Rule:      "no run other than the derived candidate run reaches the comparative reading",
		Falsifier: "drop the run narrowing from narrowRow, so the row reaches the family",
		Caught:    caughtLeak,
		Classes:   []string{"EXHAUST"},
	},
	{
		Rule:      "dispositions never reach the comparative reading",
		Falsifier: "delete the derived dispositions row and add an include row for it",
		Caught:    caughtLeak,
		Classes:   []string{"FATE"},
	},
	{
		Rule:      "admissions never reach the comparative reading",
		Falsifier: "delete the derived admissions row and add an include row for it",
		Caught:    caughtLeak,
		Classes:   []string{"GROUNDS"},
	},
	{
		Rule:      "surprises never reach the comparative reading",
		Falsifier: "delete the derived surprises row and add an include row for it",
		Caught:    caughtLeak,
		Classes:   []string{"FATE"},
	},
	{
		Rule:      "reframe records never reach the comparative reading (spc-2609020626048705)",
		Falsifier: "delete the derived reframes row and add an include row for it",
		Caught:    caughtLeak,
		Classes:   []string{"LEDGER-REFRAME"},
	},
	{
		Rule:      "the status directories never reach the comparative reading",
		Falsifier: "delete the three derived status rows and add an include row under .abcd/work/issues",
		Caught:    caughtLeak,
		Classes:   []string{"DECISION"},
	},
	{
		Rule: "a candidate carrying a standing disposition or an admission refuses the assembly, " +
			"because the candidate set is defined as pre-admission",
		Falsifier: "delete the capture.ItemFate call from WideningRuns",
		Caught:    caughtRefusal,
		Classes:   []string{"FATE"},
	},
}

// TestEveryAssemblerRuleHasAFalsifier keeps the coverage matrix honest against
// the rest of this package.
//
// It is the check that turns the matrix from a comment into a claim: a rule
// naming a plant that no longer exists fails here, a plant that no rule depends
// on fails here, and a row that is neither falsifiable nor declared unfalsifiable
// fails here. What it does not and cannot check is that the matrix names every
// rule the assembler HAS — that is a human reading of the include table against
// this list, and it is why the matrix carries each row's falsifier in full.
func TestEveryAssemblerRuleHasAFalsifier(t *testing.T) {
	requireOracleTables(t)

	// The count pin catches a DELETED row. It does not catch a row SUBSTITUTED for
	// a duplicate of another: swapping the structural-deny row for a second copy
	// of the glossary row keeps the count, keeps the gap count, orphans no class,
	// and drops the most load-bearing rule in the assembler's contract in silence.
	// Distinct rule text closes the by-duplication half of that. The other half —
	// a row rewritten to a rule the assembler does not have — is the declared
	// limit above, and it needs a human reading the include table against this
	// list.
	byRule := map[string]bool{}
	for _, row := range coverage {
		if byRule[row.Rule] {
			t.Errorf("two coverage rows carry the rule %q; a duplicated row keeps the count "+
				"while the rule it replaced leaves the matrix in silence", row.Rule)
		}
		byRule[row.Rule] = true
	}

	declared := map[string]bool{}
	for _, c := range sentinelClasses {
		declared[c.Name] = true
	}
	declaredRefusals := map[string]bool{}
	for _, r := range refusals {
		declaredRefusals[r.Name] = true
	}
	named := map[string]bool{}
	namedRefusals := map[string]bool{}
	gaps := 0

	for _, row := range coverage {
		switch {
		case row.Caught == "" && row.Gap == "":
			t.Errorf("the coverage row %q is neither falsifiable nor declared unfalsifiable; "+
				"a rule with no falsifier and no stated reason is the hole this matrix exists "+
				"to make visible", row.Rule)
		case row.Caught != "" && row.Gap != "":
			t.Errorf("the coverage row %q is both caught and declared a gap; it is one or the "+
				"other, and a row that hedges tells a reader nothing", row.Rule)
		}
		if row.Gap != "" {
			gaps++
		}
		if row.Falsifier == "" {
			t.Errorf("the coverage row %q names no mutation, so nobody can re-run it", row.Rule)
		}
		for _, name := range row.Classes {
			if !declared[name] {
				t.Errorf("the coverage row %q names the sentinel class %q, which no longer "+
					"exists in the plants table", row.Rule, name)
			}
			named[name] = true
		}
		for _, name := range row.Refusals {
			if !declaredRefusals[name] {
				t.Errorf("the coverage row %q names the refusal plant %q, which no longer "+
					"exists in the refusals table", row.Rule, name)
			}
			namedRefusals[name] = true
		}
	}

	// A plant no rule depends on is a plant that proves nothing. It is either a
	// rule this matrix has not written down, or a plant that should go.
	var orphans []string
	for _, c := range sentinelClasses {
		if !named[c.Name] {
			orphans = append(orphans, c.Token())
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Errorf("%d planted class(es) are named by no coverage row, so nothing says which "+
			"assembler rule they falsify: %s", len(orphans), strings.Join(orphans, ", "))
	}

	// The same for the refusal corpus, which is a second body of planted material
	// and decays the same way.
	var strayRefusals []string
	for _, r := range refusals {
		if !namedRefusals[r.Name] {
			strayRefusals = append(strayRefusals, r.Name)
		}
	}
	sort.Strings(strayRefusals)
	if len(strayRefusals) > 0 {
		t.Errorf("%d refusal plant(s) are named by no coverage row, so nothing says which "+
			"assembler rule they falsify: %s", len(strayRefusals), strings.Join(strayRefusals, ", "))
	}

	// The gap count is declared, so a row silently becoming unfalsifiable — the
	// exact way this eval would decay — has to be an explicit edit.
	const declaredGaps = 7
	if gaps != declaredGaps {
		t.Errorf("the matrix declares %d unfalsifiable row(s) and holds %d; a rule sliding "+
			"into or out of unfalsifiable coverage is the change this eval most needs said "+
			"out loud", declaredGaps, gaps)
	}
}
