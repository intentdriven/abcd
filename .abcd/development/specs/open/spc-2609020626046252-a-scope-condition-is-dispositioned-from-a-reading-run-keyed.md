---
id: spc-2609020626046252
slug: a-scope-condition-is-dispositioned-from-a-reading-run-keyed
intent: itd-2609020625405251
origin: researcher-authored
production_mode: dictated-and-formatted
---
# A second writer into the disposition surface: `abcd intent condition` dispositions one scope condition from a reading item or a delivered intent

## Summary

spc-2609020626046252 delivers
[itd-2609020625405251](../../intents/planned/itd-2609020625405251-a-scope-condition-is-dispositioned-from-a-reading-run-keyed.md).
`abcd intent condition <itd-N> <cond-id> --disposition <value> --occasioned-by
<rdi-N|itd-N> --grounds "<why>" [--narrowing "<what now holds>"]` writes one
scope-condition disposition against a shipped intent, keyed to the identity
[spc-55](../closed/spc-55-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md)
stamped on the condition, joined to what occasioned it: A reading item, or a
delivered intent in `shipped/` whose delivery changed the condition's standing.
It is the second writer into the surface
[spc-59](../closed/spc-59-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md)
gave the fidelity verdict: The `## Audit Notes` section of the intent, one
block per write. The verdict ingest keeps writing one undated block covering
every condition; this verb writes one dated block covering one condition and
naming its occasion. A condition's standing disposition is the last block in
document order that names it, except that a later verdict block does not
override a reading-occasioned block unless its rationale names that block's
occasion, and the render says which block the standing came from.

The vocabulary the two writers share moves into a leaf package,
`internal/core/condition`, so that one enum, one marker grammar and one block
reader serve the verdict ingest, this verb, the readiness gate, and the record
lint that the sibling spec
[spc-2609020626042471](spc-2609020626042471-a-principle-carries-typed-claims-its-reference-its-compariso.md)
builds over the same dispositions. The reading-item locator moves into a
second leaf, `internal/core/readingitem`, which the later Iteration 2 specs
share.

## Scope

In: The verb, its grammar and its refusals; the leaf package and the move of
the disposition enum out of `internal/core/intent/audit.go`; the condition
block grammar and its coexistence with the verdict block; the standing reader,
its fold rule and its render; the verdict ingest's report of the
reading-occasioned blocks it leaves standing; the reading-item locator and
the occasion resolver both `capture` and `intent` call; the detection
definition's item-shape guidance on citing a condition's identity and the
verb's report of the occasion against that citation; the plugin surface
page, the sub-verbs table and the command-tree snapshot; the assembler test
that the block never reaches a bundle.

Out: The reading marking a condition itself, a registered interpretation
stated under Who marks; a disposition on a draft or
planned intent; dispatch from a reading item to the conditions it occasioned,
which lands with the record dispatcher's coverage of reading items; any change
to the verdict ingest's own block shape or to the exclusion floor.

## Approach

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

This spec lands first in Phase B, at Step 5 after the maintainer's four
readings and their dispositions; the condition dispositions a reading
occasions before then are hand-authored in the target format until the verb
lands. It moves no assembler, manifest or preset version: Nothing it adds
reaches a bundle, and `internal/core/reading` is untouched. It introduces the
two leaf packages, `internal/core/condition` and `internal/core/readingitem`,
that the four Phase B specs after it consume; the reading-occasioned origin,
which lands before it in Phase A, resolves through `capture.findReadingItem`
and inherits the leaf when this spec makes that function a wrapper over it.

### The verb and its grammar

`newIntentCommand` in `internal/surface/cli/cli.go` gains a `condition`
sub-command beside `plan`, `ready`, `link` and `audit`:

```
abcd intent condition <itd-N>                                   # read: standing per condition
abcd intent condition <itd-N> <cond-id> \
    --disposition <survived|narrowed|falsified|untested> \
    --occasioned-by <rdi-N|itd-N> --grounds "<why>" [--narrowing "<text>"] [--json]
```

The one-operand form performs zero writes and renders every condition the
intent carries with its standing disposition and the block it came from. The
two-operand form writes. Both are front doors onto
`intent.DispositionCondition(repoRoot string, req ConditionRequest)` and
`intent.ConditionStanding(repoRoot, intentID string)` in a new
`internal/core/intent/condition.go`; the core never prints. `ConditionRequest`
carries `IntentID`, `ConditionID`, `Disposition`, `Narrowing`, `OccasionedBy`,
`Grounds` and `Date` (the CLI passes today's date in UTC; tests inject one).
The result, `ConditionResult`, carries the four fields written, the intent
path, the standing list after the write and, when the occasion is a reading
item whose citation names a different condition, the `OccasionCitation`
report stated under Who marks. `commands/intent.md` documents
both forms with the refusal list, `.abcd/development/brief/04-surfaces/05-intent.md`
gains the row `condition | — | shipped` in its sub-verbs table, and the
snapshot at `.abcd/development/release/surface.json` gains the path, which
`surface_coverage` checks in both directions.

### One vocabulary in a leaf package

`internal/core/condition` holds what both writers and every reader need:
`Enum` (the four values), `Narrowed` and `Untested` as named constants,
`MarkerRe` and `MarkerIDRe` (the `<!-- cond: cond-N -->` identity grammar
`claims.go` reads today), the two block-marker grammars below, and the reader.
`audit.go`'s `dispositionEnum`, `dispositionNarrowed` and `dispositionUntested`
become aliases of the package's values, and `claims.go`'s `condMarkerRe` reads
`condition.MarkerRe`. It is a leaf for the reason `core/grounds` and
`core/issueschema` are: `internal/core/lint` must read dispositions for the
sibling spec, `internal/core/intent`'s tests import `internal/core/lint`, and a
lint importing intent back is an import cycle. `internal/README.md` gains the
package's entry.

The `abcd-condition` block grammar and the block reader live in
`internal/core/condition`, which spc-2609020626046252 introduces together
with the verdict-block grammar and the `Standing` fold, and spc-2609020626046252
lands before spc-2609020626042471; the principles spec extends nothing there
and consumes the package as it finds it.

### The block grammar, and how the two writers coexist

The verdict ingest writes, through `renderDispositions`, a block that opens
with `<!-- abcd-review: INGESTED receipt=rcp-… -->` and carries a
`Scope-condition dispositions:` label followed by one bullet per condition.
This verb writes a block of the same bullet shape under its own marker:

```
<!-- abcd-condition: cond-2608311949582375 occasion=rdi-2609011200000001 -->
Condition disposition — 2026-09-02, occasioned by rdi-2609011200000001.
- cond-2608311949582375 — narrowed: <grounds>
  narrowing: <text>
```

The bullet is byte-for-byte the shape `renderDispositions` emits (`- <id> —
<value>: <rationale>`, then an indented `narrowing:` line where there is one),
so one reader parses both. The marker grammar is
`condition.BlockMarkerRe = <!-- abcd-condition: (cond-[0-9]{16}) occasion=((?:rdi|itd)-[0-9]+) -->`,
so the occasion is one of the two forms and nothing else. Every field the
verb writes goes through `oneLine`, the neutraliser the verdict render uses,
so a ground cannot forge either marker.

Coexistence is by boundary. `upsertReviewBlock` today ends a review block at
the next `markerRe` line, the next heading, or end of file; a condition block
following an INGESTED block would be swallowed when a re-ingest replaced it.
The boundary becomes `condition.IsBlockMarker(line)`, true for either marker
grammar, and both writers use it. The verb appends through
`appendToAuditNotes`, which drops the template placeholder and keeps a trailing
link-reference run below the new block, so the section grows the same way for
both writers. The verdict ingest's whole-block rule is untouched: It still
covers every condition exactly once, keyed to the record's identities, and it
knows nothing of the condition blocks around it.

### How standing is computed

`condition.ReadDispositions(content)` scans the `## Audit Notes` section and
returns every disposition in document order, each carrying its `ConditionID`,
`Disposition`, `Rationale`, `Narrowing`, `Source` (`verdict rcp-…`,
`condition rdi-…` or `condition itd-…`) and, for a condition block, its `Date`
and `Occasion`. `condition.Standing(content)` folds that list. For each
identity the **last entry in document order** stands, with one exception: An
entry from a verdict block does not replace a standing entry from a condition
block unless the verdict entry's rationale names that entry's occasion. A
re-audit writes a block covering every condition by construction and knows
nothing of the readings; without the exception it would erase, silently, the
one thing the join exists to keep, and an auditor who has weighed the reading
says so by naming it. Document order is the rule, not the date, because the
verdict block carries no date by design and both writers append, so position
in the section is the order of writing.

The verdict ingest reports what it leaves standing. After `IngestVerdict`
writes its block, `IngestVerdictResult` gains `ReadingOccasionedStanding`, the
list of condition-block entries the fold still reports as standing after the
write, each with its identity and occasion, and the CLI prints the list under
the ingest's summary. An auditor who meant to override one names its occasion
in the rationale and ingests again.

The render lists each condition the intent carries as
`<id> — <standing value> (from <source>)`, and `untested (no block)` for an
identity no block names; the `--json` form carries the whole history under
`dispositions` and the fold under `standing`. `ReadyResult` is unchanged; the
readiness gate keeps reporting identities, not dispositions.

### The occasion

`capture.findReadingItem` and `readingItemPaths` move into a leaf,
`internal/core/readingitem`, as `Locate(issuesRoot, item string) (run, path
string, err error)` and `Paths`, with sentinels `ErrUnknown` and
`ErrDuplicate`; beside them the leaf carries `LocateDisposition` and
`ResolveOccasion(issuesRoot, id string, families ...Family)`, the one occasion
resolver the admission, reframe and condition verbs share, each naming the
families it admits. `capture` keeps `findReadingItem` and `readingItemPaths`
as thin wrappers over the leaf, mapping `ErrUnknown` and `ErrDuplicate` onto
its own `ErrUnknownIssueID` and `ErrDuplicateIssueID`, until every caller has
moved, so its contract and `mintUnusedItemID`'s probing are unchanged and the
later specs that name the `capture` functions keep resolving. `intent` cannot
import `capture` (capture imports intent), so the locator has to live below
both.

The verb resolves `--occasioned-by` through `ResolveOccasion` with the
families `rdi` and `itd`. A `rdi-N` resolves through `Locate` under
`.abcd/work/issues`: Zero matches, more than one, a symlinked run directory,
or an id outside `recordid.ValidReadingItemID` is a refusal. An `itd-N`
resolves only in the intents store's `shipped/` bucket, which the leaf
reaches from the repository root the issues root sits under; a planned or
draft intent, or one not found, is a refusal naming the bucket or the
absence. The delivered-intent form is what re-dispositions a condition a
later delivery changed, which is how itd-199's surviving condition
cond-2608312031028702 (the comparative position refuses) is re-dispositioned
with the comparative channel's intent as its occasion once that ships. The
item's position is not read: An entailment item bears on a context claim as
readily as a detection does, and the verb takes an item at any position.

### Who marks: A registered interpretation

The reading names a tension and the researcher marks the condition. The
design framework's letter reads the other way in two places: Section 7.1 has
the reading mark each scope condition survived, narrowed, falsified or
untested, and section 14 has each scope condition dispositioned under
Iteration 2's cold column. This spec reads both as the reading naming the
condition and the researcher marking it, and registers that as an
interpretation on three grounds: The readings companion's ratified
registrative body (section 4.2) carries `tension`, `constraint_in_play` and
`why_a_tension` and no field for a mark; the companion (section 9) says
readings do not disposition their own items; and the build sheet's W8 makes
dispositions warm and never passed to a reading. A body field for a mark
would have the reading disposition its own item, which is the rule
[itd-181](../../intents/shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md)
adopted against.

The join runs through the item's citation. The item-shape guidance in
`agents/cold-reading-detection.md` gains one sentence: Where the constraint in
play is a scope condition, the item cites the condition's identity, `cond-`
and sixteen digits, in `constraint_in_play`, beside the quoted text. The
definition's `prompt_version` moves and `agents/CHANGELOG.md` records the
move under the Iteration 2 heading, which the `agent_contract` rule holds in
lockstep; the ingest verb's validation of the body is untouched, because
`constraint_in_play` is free text and the citation sits inside it. The verb
reads the citation when the occasion is a `rdi-N`: `ResolveOccasion` returns
the item's path, the verb decodes its `constraint_in_play`, and when the
field carries a `cond-` identity that is not the one being dispositioned the
result carries `OccasionCitation` naming both and the CLI prints it as a
report under the write's summary. A mismatch is reported and never refused,
because the item is the reading's word and the mark is the researcher's, and
a researcher who dispositions a condition an item did not cite may be right;
an item with no citation reports nothing. The definition's guidance binds the
reading and this report is the researcher's mirror of it, and it is the only
place the verb reads a reading item's body.

### What the verb refuses

Every refusal exits 2 with nothing written, in this order: An intent id
outside `recordid.ValidIntentID`; an intent not found; an intent whose bucket
is not `shipped/`, naming the bucket it is in; a condition id outside
`MarkerIDRe`; an identity the intent does not carry, read through
`ParseClaims`; an identity carried by more than one bullet
(`DuplicateConditionIDs`); a value outside the enum, naming the four; a
`--grounds` below the floor; `narrowed` without `--narrowing`; `--narrowing`
with any other value; an occasion that does not resolve in either form; and
an `itd-N` occasion naming the intent itself, whose own delivery is the
verdict ingest's ground and not this verb's.

`--grounds` is held to the substance floor every grounds-shaped field
shares: The text goes through `grounds.Fold` and then `grounds.ValidateText`,
so the empty text, a control character the frontmatter serialiser refuses, a
text below the unit or letter count, and a text made only of the vocabulary's
own words are refused with `ValidateText`'s own messages, the empty text as
`grounds text is empty; name the conjecture being acted on, not the route
taken`. The folded text is what the block carries. The write goes through
`writeIntentFile`, so the byte cap the other intent writers have applies here
too.

### Exclusion from readings

Nothing changes in `internal/core/reading`. The block lands under
`## Audit Notes`, which `Exclusions` names by heading and `intentProjection`
never lists, so `redactExcluded` removes it from every shipped intent before
projection. A test in the reading package assembles over a fixture intent
carrying a condition block and asserts that no bundle item contains the
`abcd-condition` marker, the occasion id or the ground, and that the manifest
asserts the `Audit Notes` exclusion. The test is a regression guard: It
cannot fail against the assembler as delivered, because the heading is
already excluded, and it exists so that a later change to the heading floor
is caught by the block it would expose.

### Dispatch, from the intent's side only

`abcd <rdi-N>` does not dispatch reading items today, and this spec does not
add it: The join is read from the intent's side by `ConditionStanding`, whose
`--json` names the occasion of every condition block, which is what the
intent's third scope condition allows. Dispatch on reading items is the
residual
[spc-67](../closed/spc-67-what-the-widening-reading-proposes-is-admitted-or-declined-o.md)
records, as spc-2609020626040342 cites it, and until it
lands the intent's side is the only side; the intent's In Scope claims
nothing of the item's dispatch.

## How the Acceptance Criteria are satisfied

- ac-1: The write form with `falsified` appends a dated block carrying the
  identity, the value, the occasion and the ground; the render shows it under
  its own marker. `TestConditionWritesADatedBlock`.
- ac-2: `narrowed` without `--narrowing` is refused with the message
  `narrowed but states no narrowing`. `TestConditionNarrowedRequiresNarrowing`.
- ac-3: `survived` with `--narrowing` is refused: `only a narrowed condition
  carries one`. `TestConditionNarrowingOnlyOnNarrowed`.
- ac-4: A value outside the four is refused and the message lists them.
  `TestConditionRefusesOutOfEnum`.
- ac-5: An occasion that does not resolve is refused in both forms: A `rdi-N`
  no run holds, through `readingitem.Locate`, and an `itd-N` that is absent
  or not in `shipped/`, through `ResolveOccasion`. An item that resolves and
  cites a different condition is not a refusal; it is reported, beside.
  `TestConditionOccasionMustResolve`, `TestConditionOccasionIntentMustBeShipped`,
  `TestConditionReportsOccasionCitationMismatch`.
- ac-6: A planned intent is refused and the message names `planned`.
  `TestConditionRefusesUnshippedBucket`.
- ac-7: After a verdict block and a condition block naming the same identity,
  `ReadDispositions` returns both and `Standing` reports the later. The
  inverse order is covered beside it: A verdict block written after a
  condition block leaves the condition block standing and is reported,
  unless its rationale names the occasion.
  `TestStandingIsTheLatestBlock`, `TestVerdictDoesNotOverrideAReadingOccasionedBlock`,
  `TestVerdictNamingTheOccasionOverrides`, `TestReviewBlockBoundaryStopsAtConditionMarker`.
- ac-8: An assembly over the fixture intent carries no block and the manifest
  asserts the exclusion. A regression guard, as stated above.
  `TestConditionBlockNeverReachesTheBundle`.

## Tests

Each watched fail before the change and pass after; each refusal proved by
presenting the forbidden input and asserting nothing was written.

- `internal/core/condition`: `TestReadDispositionsParsesVerdictBlock`,
  `TestReadDispositionsParsesConditionBlock`, `TestStandingIsTheLatestBlock`,
  `TestVerdictDoesNotOverrideAReadingOccasionedBlock`,
  `TestVerdictNamingTheOccasionOverrides`, `TestIsBlockMarkerAcceptsBothGrammars`,
  `TestBlockMarkerAdmitsBothOccasionForms`, `TestEnumIsTheFourValues`.
- `internal/core/intent`: The refusal tests above,
  `TestConditionGroundsBelowTheFloorRefuse`, `TestConditionRefusesSelfOccasion`,
  `TestConditionWritesADatedBlock`, `TestConditionGroundsAreNeutralised` (a
  ground carrying `<!-- abcd-review: INGESTED receipt=rcp-000000000000 -->`
  forges nothing), `TestConditionRefusesDuplicateIdentity`,
  `TestReviewBlockBoundaryStopsAtConditionMarker`,
  `TestVerdictIngestUnchangedBesideConditionBlocks`,
  `TestIngestReportsReadingOccasionedStanding`,
  `TestConditionReportsOccasionCitationMismatch`,
  `TestConditionOccasionWithoutCitationReportsNothing`, and the existing
  `audit_test.go` suite unchanged after the enum moves.
- `internal/core/readingitem` and `internal/core/capture`:
  `TestLocateFindsOneItemAcrossRuns`, `TestLocateRefusesSymlinkedRun`,
  `TestResolveOccasionAcceptsShippedIntent`,
  `TestResolveOccasionRefusesPlannedIntent`, `TestCaptureSentinelsWrapLocator`.
- `internal/surface/cli`: `TestIntentConditionReadFormWritesNothing`,
  `TestIntentConditionJSONCarriesStanding`, and the surface snapshot test.
- `internal/core/reading`: `TestConditionBlockNeverReachesTheBundle`, the
  regression guard.
- `internal/core/lint`: The sub-verbs row and the snapshot in lockstep, and
  the `agent_contract` rule holding the detection definition's moved
  `prompt_version` and its changelog entry in lockstep.

## Out of scope

- The reading marking conditions itself, on the rule
  [itd-181](../../intents/shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md)
  adopted: The reading names tensions and the researcher marks. That is a
  registered interpretation of the design framework's sections 7.1 and 14,
  stated under Who marks with its grounds.
- A disposition on a draft or planned intent.
- Dispatching `abcd <rdi-N>` to the conditions it occasioned.
- A superseding disposition record of the kind
  [itd-180](../../intents/shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md)
  gives reading items: A condition's history is the sequence of blocks, and
  standing is a fold over it, never a separate record.
- Any move of the assembler, manifest or preset versions; nothing this spec
  adds reaches a bundle.
