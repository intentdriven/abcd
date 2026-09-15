---
id: spc-2609020626042168
slug: an-intent-that-a-reading-occasioned-says-so-in-its-origin-wi
intent: itd-2609020625400169
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The reading-occasioned origin: promote stamps the run and the item it graduated, and the stamp primitive refuses the kind without them

## Summary

spc-2609020626042168 delivers
[itd-2609020625400169](../../intents/shipped/itd-2609020625400169-an-intent-that-a-reading-occasioned-says-so-in-its-origin-wi.md).
Today `capture promote <rdi-N>` mints a draft whose `origin` reads
`extracted-from-record`: `promoteReadingItem` in `internal/core/capture/promote.go`
passes `provenance.KindExtractedFromRecord` into `intent.CreateDraft` exactly as
the issue route does, and `provenance.NewStamp` refuses `KindContributedByReading`
outright. Its doc comment says the kind "is minted only by the reading-ingest
verb", and its error reads, exactly, "origin contributed-by-reading is minted
only by the reading-ingest verb, which carries the run and item identifiers; no
write path in this repository can supply them". That verb has shipped
([itd-185](../../intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md))
and it mints items, not intents. The one command that moves a reading item
toward an intent is promote, so the third arrival path of
[itd-178](../../intents/shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md)
has a lint that resolves it (`checkRecordProvenance` in
`internal/core/lint/provenance.go` maps an item to its run directory) and no
writer.

This spec gives it the writer. The pair is taken from where the item was found:
`capture.findReadingItem` returns the run and the path
`readings/<rdg-N>/<rdi-N>.md` (through `readingitem.Locate` once the
condition spec lands), so the run is the directory the item sits in, which is
the same join the lint checks. The stamp primitive gains a constructor that takes the pair and refuses
it unshaped, and `NewStamp` keeps refusing the kind without one. Link mode
writes both edges and leaves the draft's `origin` untouched. The issue route is
not changed. The promoted draft's Press Release seed carries no item id, so the
join lives only in fields no reading projects.

## Scope

In: A reading-pointer constructor in `internal/core/provenance`; the parsed
origin as the draft-mint option in `internal/core/intent`; the reading route of
`capture.Promote` deriving the run from the item's store path and refusing a
record whose own `run` disagrees; the seed note of a reading-route draft; a
`promoted_from` writer for link mode; the lint checking the join from both
ends; a promoted-seed plant in the read-block eval; the two plugin pages saying
which path writes which value.

Out: The issue route's stamp; any backfill; a landing other than an intent
draft; a keyed digest that would make the stamp unforgeable, which itd-178
scopes out and this spec does not reopen.

## Approach

### The stamp primitive: `NewReadingStamp`, with `NewStamp` unchanged for the other two kinds

`provenance.NewStamp(kind Kind, mode string)` keeps its signature and keeps
refusing `KindContributedByReading`. A call that supplies no pointer is exactly
the call the intent's fourth criterion says must refuse, so that door stays
shut; only its message changes, to say the kind is minted through the
constructor below, because the sentence about the ingest verb being the minter
has been false since itd-185. Beside it:

```go
// NewReadingStamp is the ONE constructor of a contributed-by-reading stamp.
// The run and the item are the pointer the value carries; a stamp without
// them is unrepresentable, which is why NewStamp refuses the kind.
func NewReadingStamp(run, item, mode string) (Stamp, error)
```

It refuses an empty run or item, a run that fails `readingRunRe`, an item that
fails `readingItemRe`, and a mode `ModeOrDefault` refuses, and returns
`Stamp{Origin: Origin{Kind: KindContributedByReading, Run: run, Item: item}, Mode: m}`.
The rendered value round-trips through `ParseOrigin` byte for byte, which a
test pins.

The fourth criterion has two halves, and the package holds one of them. The
primitive is a leaf with no filesystem, so it can judge shape and nothing
else: No value can be built without a well-formed pair, and its doc comment
states that division rather than implying more. Resolution to the readings
store is the promote path's, which reads the store before it mints, as the
next section states; the intent's fourth criterion is worded to that split.
The comment on `KindContributedByReading` is rewritten to name promote as the
minter.

### The draft-mint option carries the parsed origin

`intent.DraftOptions.Origin` changes type from `provenance.Kind` to
`provenance.Origin`, so a caller naming the reading kind must hand over the run
and the item in the same field. `CreateDraft` in `internal/core/intent/create.go`
branches on it:

- Kind empty: `KindResearcherAuthored`, as now.
- Kind reading: `provenance.NewReadingStamp(o.Run, o.Item, opts.ProductionMode)`,
  and additionally `opts.PromotedFrom` must equal `o.Item`, or the mint refuses
  ("a reading origin names item X but the back-edge names Y; the two are one
  join"). The two fields are redundant by design, and a draft carrying them in
  disagreement is a state no command produced.
- Otherwise: `provenance.NewStamp(o.Kind, opts.ProductionMode)`, as now.

`seedDraft` is untouched in what it writes to the frontmatter: It writes
`stamp.OriginValue()`, and the reading value renders as
`contributed-by-reading rdg-N/rdi-N`, a plain scalar with no `: ` inside it,
which is what keeps it readable to `frontmatter.Fields`. The two call sites
move with the type: `CreateFromText` sets nothing, as now, and the issue route
of `capture.Promote` passes
`provenance.Origin{Kind: provenance.KindExtractedFromRecord}`.

### The seed carries no item id, and why the title may

`seedNote` today renders the Press Release placeholder as
`_Seeded by promotion from <promoted_from>. Expand into the full press-release narrative before planning._`,
and the Press Release is the first field `intentProjection` projects; a draft
is admitted at the entailment position. A promoted reading draft would
therefore hand the entailment reading an `rdi-N`, a prior reading's output
named inside the object of the next one, which the readings companion forbids
("no reading sees another's output"). The seed of a reading-route draft
becomes `_Seeded by promotion from a reading item. Expand into the full press-release narrative before planning._`:
same opening, same tail, so `IsSeedNote` goes on matching both forms by prefix
and suffix, and the issue-route seed keeps its `iss-N`, since an issue is not
a reading's output. The join to the item is carried where no projection
reaches: `origin` and `promoted_from` are frontmatter keys the projection does
not name (the read-block eval's `DRAFT-ORIGIN` class holds `origin` warm at
every position), and the "Graduated from" line that names the item sits in
Why This Matters, which is not projected either.

The draft's title stays the item's `pattern`. That is admissible where the id
is not: A pattern is the name of an established practice the reading pointed
at, a name that stands on its own in the literature or the record and would be
the title a person typed; it carries no finding, no tension and no identifier,
and a reader handed it learns what the draft is about, not what an instrument
returned. The reading record stays the single source of the instrument's own
text.

The read-block eval gains a promoted-seed case in its `DRAFT-BODY` class: A
second draft fixture under `testdata/cold-reading/baseline/.abcd/development/intents/drafts/`,
minted in the reading-route shape (the reading-form seed in its Press Release
carrying the class token, `origin: contributed-by-reading` naming a fixture
run and item, `promoted_from` naming the item, the "Graduated from" line in
Why This Matters). The class's `Count` moves to 2 and `sentinelClasses` does
not, since no class is added; the baseline run proves the token cold at
entailment alone, and a companion assertion in the same test requires the
fixture item's id in no bundle at any position, which is the sentence above
made falsifiable.

### The promote path for a reading item derives the pair from the store

In `promoteReadingItem`, the run comes from
`capture.findReadingItem(issuesRoot, req.ID)`, which returns the run and the
path together. This spec lands before the condition spec (see the landing
order), so it calls the `capture` function as it stands, and when that spec
makes the function a thin wrapper over `readingitem.Locate` this path
resolves through the leaf without an edit. `readingItemPaths` admits only a
run directory whose name passes `recordid.ValidReadingRunID`, so the derived
run is well formed by the rule the store enforces, and the pair resolves by construction: It was read off
`readings/<run>/<item>.md`, which is the store read that delivers the
resolution half of the fourth criterion. Belt and braces: The record's own
`run` field, which `issueschema.ReadingRequired` makes mandatory, must equal
the directory name, and a disagreement refuses before anything is minted
("rdi-N sits in run rdg-A but its record names rdg-B; nothing minted"). The
mint then passes
`Origin: provenance.Origin{Kind: provenance.KindContributedByReading, Run: run, Item: req.ID}`
with `PromotedFrom: req.ID`. The mint-first, stamp-second ordering, the
under-lock re-read of the standing disposition, and the orphan remedy are all
unchanged.

The standing disposition the re-read requires is `accepted`, and at the
widening position an `accepted` disposition can be written only once a
committed comparative run names the item's widening run: The shared
disposition writer refuses every disposition write at that position until
then. Promoting a widening item is therefore transitively gated on the
comparative run. This spec does not implement or test that gate, which is the
comparative channel's and the admission spec's; it inherits it through the
disposition it reads, and the first criterion is read under that condition.

### Link mode writes both edges and leaves the origin alone

Today `--intent <itd-N>` on either route stamps only the source record's
`promoted_to`; the existing draft gains nothing. The intent requires the
reading route to write `promoted_from` on the draft as well. A new
`intent.SetPromotedFrom(repoRoot, intentID, source string) (Intent, error)` is
added in `internal/core/intent/lifecycle.go` beside `Link`, on the same
`readRepoFile`, `setFrontmatterFields`, `writeIntentFile` idiom. It refuses an
intent not found in any bucket and a source failing `promotedFromRe`; a draft
already naming this source is a no-op that reports the record unchanged. It
writes `promoted_from` and nothing else. It never reads or rewrites `origin` or
`production_mode`, which is what "origin unchanged" rests on: An origin is
stamped at mint and never rewritten, so a hand-filed draft linked to a reading
item stays `researcher-authored` and says so.

A draft whose `promoted_from` already names a different record is the case the
intent's first scope condition describes: An intent occasioned by several
items is promoted from one, and the others are joined by their own
`promoted_to`. `SetPromotedFrom` returns a typed `ErrBackEdgeTaken` naming the
record already there; the reading route does not treat it as a refusal. It
skips the back-edge, goes on to stamp the item's `promoted_to`, and reports
`back_edge: kept <existing>` in the result and in both renderings, so the
operator sees that the draft's one back-edge stayed where it was and the item
still points forward.

Ordering on the reading route: The draft write runs after the pre-flight and
before the ledger-locked item stamp. A failure between the two leaves a draft
naming the item and an item not yet naming the draft; re-running the same
command completes it, because the draft write is idempotent and the stamp is
the step that was missing. The issue route's link mode is not changed, per the
intent's scope.

### The lint checks the join from both ends

`provenanceFindings` in `internal/core/lint/provenance.go` gains one same-record
check: A `contributed-by-reading` origin on a record whose `promoted_from` is
absent, or names an item other than the origin's, is a finding, on the same
footing as `extracted-from-record` with no back-edge. `checkRecordProvenance`
gains a second map from the scan, item to `promoted_to`, and reports an intent
whose origin names item X while X's `promoted_to` names a different record. The
reverse direction is deliberately not a finding: An item whose `promoted_to`
names a `researcher-authored` draft is link mode working as designed, and so
is an item whose `promoted_to` names a draft whose `promoted_from` names
another item, which is the several-items case above. Every message keeps
`handEditResidual`.

### The plugin pages say which path writes which value

`commands/capture.md`'s disclosure section currently says the reading value is
one "which only the reading-ingest verb mints"; that sentence becomes "which
`capture promote <rdi-N>` mints when it derives a draft from an accepted reading
item, naming the item's run and id", and the promote section's paragraph on
reading items gains the origin the draft carries and the kept-back-edge report.
`commands/intent.md`'s disclosure section, which today names only two values,
gains the third with the same attribution. Neither page gains a flag: The
value is derived from which command ran, as before.

### Landing order

The Iteration 2 set lands in two phases around the maintainer's readings, on
the corrections ruling of 2026-09-02. Phase A, before any reading runs: The
two-operand invocation (spc-2609021004075744), itd-194
(spc-2609021003136831), the preset windows (spc-2609020626048722), the
comparative channel (spc-2609020626039834) and the reading-occasioned origin
(spc-2609020626042168), in that order. The maintainer then runs the four
readings in the cycle's order, widening, entailment, comparative and
detection, and dispositions their outputs. Phase B, built at Step 5 after
those dispositions: The condition disposition (spc-2609020626046252), the
admission and surprise verbs (spc-2609020626040342), the reframe record
(spc-2609020626048705), the scribe verbs (spc-2609020626045177) and the
principles read object (spc-2609020626042471), in that order. No spec names a
target version number: Each names the class of bump it makes, and the merging
change sets each constant from the merged base and updates every pinned count
in the same diff.

This spec lands last in Phase A, on the corrections ruling: The framework's
section 6.3 has the origin key written only by commands, and no promotion of
an accepted reading item may precede its writer, so the writer is in place
before the first reading runs. Because it lands before the condition
disposition, it calls `capture.findReadingItem` as that function stands and
inherits the leaf without an edit when the condition spec makes the function
a thin wrapper over `readingitem.Locate`. The gate it inherits on a widening
item's promotion is the comparative channel's, which lands before it, and the
admission verb's, which lands in Phase B; between the phases an `accepted`
disposition on a widening item is hand-authored in the target format after
the comparative run, and the promote path reads it as it reads a verb-written
one. This spec moves neither `SchemaVersion` nor `AssemblerVersionCore`: It
changes no artefact field, no kind and no include row, and the eval plant is
a fixture, not a table row.

## How the Acceptance Criteria are satisfied

- **ac-1 (mint stamps the pair).** `promoteReadingItem` derives the run from
  the item's store path and mints through `NewReadingStamp`, so the draft's
  frontmatter reads `origin: contributed-by-reading <rdg-N>/<rdi-N>`. Proved by
  `TestPromoteReadingItemStampsContributedByReading`, which reads the line back
  and then runs the record-provenance rule over the fixture and requires zero
  findings, so "the provenance lint resolves it" is exercised rather than
  inferred. The fixture item is at a position other than widening, since at
  widening the `accepted` disposition the route requires waits on the
  comparative run.
- **ac-2 (link mode writes both edges, origin unchanged).** `SetPromotedFrom`
  writes the back-edge, the ledger-locked stamp writes `promoted_to`, and
  neither touches the disclosure pair. Proved by
  `TestPromoteReadingItemLinkWritesBothEdgesAndLeavesOriginAlone`, which asserts
  the two disclosure lines are byte-identical before and after.
- **ac-3 (issue route unchanged).** The issue route passes
  `Origin{Kind: KindExtractedFromRecord}` and nothing else moves. Proved by the
  existing `TestPromoteStampsExtractedFromRecord`, re-run unchanged.
- **ac-4 (the primitive refuses the kind without a well-formed pair; resolution is the promote path's).**
  `NewStamp` keeps refusing the kind, and `NewReadingStamp` refuses an empty or
  unshaped run or item; that is the shape half, proved by
  `TestNewStampRefusesTheReadingKindWithoutAPointer` and
  `TestNewReadingStampRefusesAnUnshapedPair`. The resolution half is the
  store read in `promoteReadingItem`, proved by
  `TestPromoteReadingItemStampsContributedByReading` (the pair it stamps is
  the one it read) and `TestPromoteReadingItemRefusesARunMismatch`.

## Tests

Every test is watched fail before the change and pass after.

`internal/core/provenance`:

- `TestNewReadingStampCarriesTheJoin`: A well-formed pair yields a stamp whose
  `OriginValue()` is `contributed-by-reading rdg-N/rdi-N`.
- `TestReadingStampRoundTripsThroughParseOrigin`: Render then parse gives the
  same `Origin`.
- `TestNewReadingStampRefusesAnUnshapedPair`: Empty run, empty item, an `iss-N`
  where an item is wanted, an item where a run is wanted, and a pair with
  surrounding whitespace each refuse; nothing is defaulted.
- `TestNewStampRefusesTheReadingKindWithoutAPointer`: The existing refusal,
  kept, and its message now names `NewReadingStamp`.

`internal/core/intent`:

- `TestCreateDraftStampsAReadingOrigin`: The rendered frontmatter carries the
  pair and the production mode together.
- `TestCreateDraftRefusesAReadingOriginDisagreeingWithTheBackEdge`: An origin
  naming `rdi-1` with `PromotedFrom: rdi-2` mints nothing.
- `TestReadingRouteSeedNamesNoItem`: The Press Release of a draft minted with
  a reading origin contains no `rdi-`, `IsSeedNote` still matches it, and the
  issue-route seed is unchanged.
- `TestSetPromotedFromWritesOnlyTheBackEdge`: Every other frontmatter line is
  byte-identical after the write.
- `TestSetPromotedFromReportsATakenBackEdgeAndIsIdempotentOnTheSame`: A
  different source returns `ErrBackEdgeTaken` naming the record already there
  and writes nothing; the same source is a no-op.

`internal/core/capture`:

- `TestPromoteReadingItemStampsContributedByReading` (ac-1, ac-4).
- `TestPromoteReadingItemRefusesARunMismatch` (ac-4): A record whose `run`
  field names a run other than its directory mints nothing.
- `TestPromoteReadingItemLinkWritesBothEdgesAndLeavesOriginAlone` (ac-2).
- `TestPromoteReadingItemLinkKeepsAnExistingBackEdge`: A draft already promoted
  from another item keeps that back-edge, the second item's `promoted_to` is
  stamped, and the result reports the kept record.
- `TestPromoteReadingItemLinkCompletesOnRerunAfterAStampFailure`: Under
  `stampWriteHook` the first run fails after the draft write; the second run
  finds the back-edge already present and completes the stamp.
- `TestPromoteStampsExtractedFromRecord` (ac-3, existing).

`internal/core/lint`:

- `TestRecordProvenanceRequiresTheBackEdgeBesideAReadingOrigin`.
- `TestRecordProvenanceChecksTheForwardEdge`: An item whose `promoted_to` names
  a draft other than the one whose origin names it is reported once, on the
  draft.

`internal/surface/cli`:

- `TestCapturePromoteReadingItemStampsTheOriginPair`: The JSON `intent_path`
  names a file whose origin line carries the pair, so the front door is shown
  executing the core change rather than assumed to.
- `TestCapturePromoteReadingItemLinkReportsAKeptBackEdge`: Both renderings
  carry the kept-back-edge report.

`evals`: The read-block lane over the promoted-seed plant, baseline and holed,
run explicitly.

## Out of scope

- Backfilling any record: Population is forward-only, and no draft exists
  today whose origin this would have changed.
- A reading item promoted to anything other than an intent draft. A discipline,
  an ADR or a brief passage carries its join by its own means.
- The issue route's link mode writing `promoted_from` on the draft. The intent
  holds the issue path unchanged, and extending it is its own change.
- The gate on dispositions at the widening position. It lives in the shared
  disposition writer under the comparative channel and admission specs; this
  spec inherits it.
- A mechanism that could tell a hand-typed legal value from a command's write.
  The lint's residual is disclosed in every message it emits, as itd-178 ruled.
