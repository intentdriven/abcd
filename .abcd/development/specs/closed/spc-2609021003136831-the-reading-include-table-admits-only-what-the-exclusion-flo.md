---
id: spc-2609021003136831
slug: the-reading-include-table-admits-only-what-the-exclusion-flo
intent: itd-194
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The include table declares what the floor parses, the floor refuses the markdown it cannot resolve, and the manifest marks every item it did not examine

## Summary

spc-2609021003136831 delivers
[itd-194](../../intents/shipped/itd-194-the-reading-include-table-admits-only-what-the-exclusion-flo.md).
The include table in `internal/core/reading/include.go` gains, on every row,
a declaration of whether the exclusion floor parses what the row admits, and
the rendered charter carries it as a column. The floor's key and heading scan
runs over exactly the rows declared parsed, which are the markdown rows, in
place of the extension test `redactExcluded` performs today, so admission and
examination are one declaration. Over a parsed document the floor refuses the
six shapes it cannot resolve, naming the document and the shape, and never
admits one unscanned. Over an unparsed row, which today is `source`, `test`
and `config`, an item travels whole and its manifest entry carries the mark
`"scan": "unscanned"`; every parsed item carries `"scan": "parsed"`, and the
manifest's key and heading exclusions are asserted for the parsed items and
for no other. The `source` and `test` kinds stay in the table, so the object
stays as the design framework and the readings companion state it: The
committed detection and widening entries name both, on the object-set ruling
of 2026-09-02, with every such item marked `unscanned`, and the entailment
entry names neither. The shipped intent projection row
withdraws from the widening position and the floor asserts that exclusion
there, which resolves iss-2609012259587904 on the maintainer's ruling of
2026-09-02. The table also admits the brief's meta, product, constraints,
surfaces, internals and delivery chapters as brief sections, one row each, at
every position the brief rows admit, and keeps the evidence chapter excluded
as verdict material with that ground stated in the exclusion row, which
resolves iss-2609021153264023 on the corrections ruling of the same day.

The narrowing costs the committed entries nothing they do not disclose. Under
the object-set entries the preset-windows spec fixes, the seven delivered
paths hand each of the detection and widening positions forty-nine source and
eighty-six test items, every one marked `unscanned`, and hand the entailment
position nothing, so no source or test item reaches a reader unmarked.
Measured in a clean clone of the design branch's head (the presets
recalibrated in the clone as the preset-windows spec records; the clone's own
commit is not on any branch, and the figures are re-measured at landing),
every item a detection entry
that opts the lint package's tree in by path hands arrives marked (21 source,
35 test and 21 configuration items at that measurement, every one of them
marked), as do the configuration files the default kinds admit under the
object set's paths. The live fixture leak at `itm-0736` is absent from every
committed entry, whose paths do not reach `internal/core/site`, and disclosed
as unscanned wherever an entry opts that path's tests in.

## Scope

In: The `Scan` vocabulary and the row declaration in `include.go`; the mark
on `ManifestItem` and its refusal in `DecodeManifest`; the row-driven scan and
the per-item mark in `collect` and `Assemble`; the six refusals in
`project.go`; the unscanned count on the size report and its rendering; the
shipped row's positions, the widening exclusion row and the regenerated
widening definition; the four brief-chapter rows and the reworded evidence
exclusion; the regenerated charter; the read-block eval's refusal
plants, its carrier, its oracle mark check and the pinned counts; the plugin
page and the readings charter prose.

Out: Widening the floor's parseable set; removing the `test` kind from the
table; recalibrating the committed presets, which the preset-windows spec
owns; the four inherited findings the intent lists; the comparative channel's
row and every later spec's move of the same constants.

## Approach

### The vocabulary: one word for the declaration and the mark

`include.go` declares `type Scan string` with two members, `ScanParsed`
(`"parsed"`) and `ScanUnscanned` (`"unscanned"`), and `Scans()` lists them
in that order. The same word is used in two places, so the table's promise and
the manifest's fact cannot be spelled differently: `Row` gains `Scan Scan`,
the declaration of whether the floor parses what the row admits, and
`ManifestItem` gains `Scan Scan` with the JSON key `scan`, the fact of whether
it did.

Every markdown row (the six brief rows, the glossary row, the disciplines,
shipped, drafts, planned and specs rows, and the root `.md` row) declares
`ScanParsed`. The `_test.go`, `.go` and config rows declare `ScanUnscanned`.
`TestEveryRowNamesAKnownKindAndPosition` in `include_test.go` extends to
refuse a row whose `Scan` is neither member, and a new
`TestParsedRowsAdmitOnlyMarkdown` holds the declaration to the parser: a row
declared parsed matches `.md` and nothing else, and a row declared unscanned
matches no `.md`. That test is the point at which the include table's
admission and the floor's examination are proven to describe one set,
whichever of them moves later, which is the third clause of brief invariant
16.

### The floor runs over the declaration, not over the extension

`redactExcluded` in `project.go` loses its first statement, the
`path.Ext(rel)` test that scoped the scan to markdown. `collect` in
`assemble.go` decides instead: for a row declared `ScanParsed` it calls
`redactExcluded` as today; for a row declared `ScanUnscanned` it passes the
document through untouched. `candidate` gains `scan Scan`, set from the row,
and `Assemble` copies it onto each `ManifestItem`. `refuseOwnArtefact` still
runs over every admitted file whatever its row, because the artefact tag is a
byte signature and needs no parse.

The rule text of the three unscanned rows is rewritten to state the
narrowing: "admitted where a committed preset entry names this kind, and
never examined: an item admitted here travels whole and marked `unscanned` in
the manifest, because the exclusion floor's key and heading signals are
record shapes only a markdown file carries". `Render()` gains a `Floor`
column between `Kind` and `Admitting rule`, rendering the row's `Scan`, so the
charter states the narrowing per row and `AssemblerVersion()` digests it.
`TestRenderCoversKindAndSuffix` in `size_test.go` extends to the new column,
and the charter between the markers in `.abcd/development/readings/README.md`
is regenerated in the same change, which
`TestReadingsCharterCarriesTheRenderedIncludeTable` requires. The charter's
prose paragraph that says a source or configuration file "travels whole with
no section scan run over them" is rewritten to say it travels whole and
marked, and to name the mark.

### The manifest marks each item, and asserts per item

`ManifestItem` in `manifest.go` gains `Scan Scan` with the tag `json:"scan"`,
deliberately not `omitempty`, on the same argument `Kind` carries: an item
without a mark is a defect, and a shape that can omit the field cannot tell
that defect from a well-formed item. `DecodeManifest` refuses an item whose
`scan` is absent or outside `Scans()`, naming the ordinal and the key, beside
the kind check it already makes. `SchemaVersion` moves by one, because the
manifest item's field set changes; the bundle is restamped with it under the
shared constant, and the constant's comment gains one sentence saying so.
This spec names the class of bump and no number: The merging change sets it
from the merged base, and the comparative channel moves it again when it
lands.

The exclusion assertion becomes per item by construction. `Manifest.Exclusions`
stays the floor's declaration, unchanged in shape, and its doc comment gains
the sentence that fixes its meaning: a row with a `frontmatter key` or
`heading` signal is asserted for the items marked `parsed` and for no other;
a row with a `directory`, `file` or `unreachable path` signal is asserted for
every item, because `assertExclusions` enforces those by path and a path
needs no parse. The read-block oracle reads the mark the same way, below.

### The size report counts what was not examined

`SizeReport` in `assemble.go` gains `Unscanned int` with the JSON key
`unscanned`, the count of items marked `unscanned`, filled by `sizeReport`
from the candidates. `renderSizeReport` in `internal/surface/cli/reading.go`
prints one line after the per-kind rows when the count is non-zero:
`unscanned: N item(s) travel whole, not examined by the exclusion floor`; under
a zero count the line is absent, so a cold run says nothing it does not need
to. `commands/reading.md` adds `size.unscanned` to the fields the host reports
and says what the mark means, and its assembled-input section states that a
manifest item carries `scan`. The report rides on the result and not on either
artefact, so this field moves neither version.

### The six refusals

Each shape below is a markdown document the floor cannot resolve, and each is
answered the way `unresolvableFrontmatterShape` already answers a YAML tag or
an anchor: a refusal naming the document, the line and the shape, raised from
`verifyRedaction` so the whole assembly stops and the operator is told which
document stopped it. No shape is redacted by guess, and none is admitted.

1. **A fence delimiter inside the frontmatter block**
   (iss-2608301350533102). `verifyRedaction` locates the block with
   `firstBlockRange` before any mask is computed, and `fenceMask` is computed
   over the body from the line after the block's close. A fence delimiter on
   any line inside the block is reported by `unresolvableFrontmatterShape` as
   "a fence delimiter inside the frontmatter block". The mask can no longer be
   toggled from inside the block, and the key scan is never switched off by
   it.
2. **A delimited block displaced from line 0** (iss-2608301237456350). A new
   `displacedFrontmatter(lines)` reports a line opening with three dashes
   that is preceded only by blank or whitespace-only lines, a byte-order mark
   or an HTML comment, as "a frontmatter block displaced from line 0 by N
   line(s)". A delimiter after real prose is a thematic break to every reader
   and opens nothing, so the false-refusal class the line-0 rule closed stays
   closed; what is refused is exactly the document this binary reads as prose
   and the reader of the bundle reads as frontmatter.
3. **A compact mapping nested in a block sequence**
   (iss-2608301237450573, first half). `unresolvableFrontmatterShape` gains
   `nestedMappingRe`, a block-sequence entry whose item opens a mapping
   (`- key:` at any indent, bare or quoted), reported as "a mapping nested in
   a block sequence". A sequence of scalars, which committed records carry,
   is not that shape and is untouched.
4. **An explicit key in a flow mapping** (iss-2608301251398360).
   `unresolvableFrontmatterShape` gains `flowExplicitKeyRe`, a `?` following
   `{` or `,` in a flow context, reported as "an explicit key in a flow
   mapping". Shapes 3 and 4 are refused whatever the key is named, which is
   the one fix the two records ask for: The floor stops recognising a nested
   key by its spelling and refuses the nesting.
5. **An attribute value opening on the line after its equals sign**
   (iss-2608301350534164). `maskMarkupData` gains a second return naming the
   shape when the blank skip after `=` reaches a newline before the opening
   quote, and `verifyRedaction` refuses it as "an attribute value that opens
   on the line after its equals sign". The markup mask never declines
   silently again on that shape; the HTML-whitespace skip the record proposes
   is not taken, because a resolved mask on that shape is comprehension the
   ruling declined.
6. **A raw heading opener with no bound** (iss-2608301421380392). The
   blank-line alternative of `rawHeadingBoundRe` becomes `\n[ \t\r]*\n`, so a
   CRLF blank line bounds an element as an LF one does; and
   `rawHeadingTitleEnds` reports an opener that reaches the end of the
   document with no hard or soft bound, which `verifyRedaction` refuses as "a
   raw heading element that is never closed". The title is never read over
   the remainder, which is the shape that admitted the heading. The quadratic
   cost of that read and the linearity test's name stay with
   iss-2608301421382564, as the intent's out-of-scope list says; the refusal
   removes the shape's admission and claims nothing about the scan's cost.

None of the six shapes appears in a committed record the include table
admits, checked over the record at the tree named above, so the refusals
refuse nothing on this corpus today. The escaped-key refusal's message,
`opensTag`'s second definition and `rawHTMLHeading`'s doc comment are
untouched, as the intent inherits them.

### The widening row

The shipped intents row's `Positions` becomes
`[]Position{PositionEntailment, PositionComparative, PositionDetection}`,
and `Exclusions` gains
`{Rule: "the widening object as the design documents state it", Signal: "directory", Detail: ".abcd/development/intents/shipped", Positions: []Position{PositionWidening}}`
beside the drafts and planned rows, so the manifest asserts the exclusion at
that position and `assertExclusions` enforces it. The row's rule text cites
the 2026-09-02 ruling and iss-2609012259587904. The comparative position stays
on the row here because the comparative channel withdraws every other row from
that position when it lands, and this spec does not pre-empt it.

`agents/cold-reading-widening.md` loses the shipped line from its "Repository
sources the assembler admits at this position" list, which
`TestDefinitionHoldsItsFiveParts` in `definitions_test.go` holds against
`admittedSources`; its `prompt_version` moves PATCH and `agents/CHANGELOG.md`
gains the line. `TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem`
in `include_test.go` gains a shipped path, refused at widening and admitted at
entailment and detection, and `TestExclusionFloorNamesEveryRecordedExclusion`
gains the row. `AssemblerVersionCore` moves MINOR: `Table`, `Exclusions`, the
rendering and the manifest shape all move, and the digest moves with the
rendering by construction.

### The brief rows

The design documents name "brief current text" as a reading's object, and
the table admits two chapters of it today. On the corrections ruling of
2026-09-02, brief current text is the whole brief bar the evidence chapter
and bar the glossary, which is its own row. `Table` therefore gains four rows
of kind `KindBriefSection`, at the positions the product and constraints rows
carry, one per chapter: `.abcd/development/brief/04-surfaces`,
`.abcd/development/brief/05-internals` and
`.abcd/development/brief/06-delivery` with the `.md` match, and the meta
chapter, which is the one file `00-meta.md` at the brief's root, by a row
whose source is the brief directory and whose `Match` is that exact basename,
so the row reaches that file and nothing beside it. The six chapters are
named individually because assembler rule 1 forbids naming `brief/` whole:
The directory contains the glossary, which is a record family with its own
row. Each row declares `ScanParsed`, and each rule text says which chapter it
admits and that the chapter is brief current text.

The evidence chapter stays excluded, and the ground moves into the table.
The `Exclusions` row for `.abcd/development/brief/03-evidence` keeps its
`directory` signal and its every-position scope, and its rule text becomes
"verdict material: A prior verdict is revision history, the ground the Audit
Notes exclusion rests on", so the charter states why the one chapter the
brief rows do not reach is left out. The manifest asserts the exclusion as it
asserts every directory row, and `assertExclusions` enforces it by path. The
oracle's `admittedRecordPaths` gains the four chapter paths, and its
evidence-chapter entry keeps its place with the ground reworded. This
resolves
[iss-2609021153264023](../../../work/issues/resolved/iss-2609021153264023-the-assembler-admits-only-the-brief-s-product-and-constraint.md);
the preset-windows spec re-measures every entry the rows enlarge at landing.

### The read-block eval and the oracle

The oracle stays transcribed and imports nothing from the assembler; every
edit below is by hand, and the pinned counts in `requireOracleTables` in
`evals/coldreading_oracle_test.go` move deliberately, each stated as the delta
this spec makes, since the merging change sets each literal from the merged
base.

- `refusals` in `evals/coldreading_fixture_test.go` +6. Six plants
  are added, one per shape, each a markdown file under an admitted row of the
  baseline fixture carrying a `refusedPrefix` token, its `Names` naming the
  path, the shape's wording and the excluded thing it hides, and its
  `Falsifier` naming the check that refuses it. `TestTheAssemblerRefusesAnUnredactableShape`
  in `evals/coldreading_test.go` runs them as it runs the two today, at
  every assembling position, and reads back what escaped when one does not
  refuse.
- `carriers` +2: `UNSCANNED`, a `_test.go` fixture under the
  baseline root carrying record-shaped text with a literal `## Audit Notes`
  section, in the shape of `internal/core/site/fixture_test.go`. It is
  assembled under a fixture preset whose detection entry opts `test` in,
  committed in the fixture so the dirty gate admits it, and is required to
  arrive whole and to carry `"scan": "unscanned"` in the manifest. And
  `CHAPTER`, the carrier token planted in a markdown file under each of the
  fixture's `04-surfaces`, `05-internals` and `06-delivery` chapters and in
  the fixture's `00-meta.md`, each required to arrive as `brief-section` at
  every position the brief rows admit.
- `checkScanMarks` is a new oracle assertion: every manifest item carries a
  `scan`; an item whose path ends in `.md` is `parsed` and any other is
  `unscanned`, transcribed from the table's declaration rather than read from
  it. `checkFieldAbsence` reads the manifest's marks and holds the excluded
  key and heading rows against the items marked `parsed` alone, so the
  carrier above does not report as a leak and an unparsed item that lost its
  mark does. That is the oracle making the assertion per item, as the
  manifest does.
- `excludedFamilies` +1: `.abcd/development/intents/shipped` at
  the widening position; the evidence-chapter entry keeps its place with its
  source reworded to the verdict-material ground. `admittedRecordPaths` +4,
  gaining the four chapter paths, its shipped row gaining
  `Positions` entailment, comparative and detection.
  `TestPriorRunExhaustNeverReaches` is untouched. A new
  `TestWideningNeverSeesTheShippedIntents` asserts family absence at widening
  over the fixture's shipped intent and its presence at the other two.
- `coverage` in `evals/coldreading_coverage_test.go` +12. The row
  "intents/shipped is admitted at every position" is reworded to entailment
  and detection, and the row "the brief's evidence chapter is deliberation
  and never travels" is reworded to "the brief's evidence chapter is verdict
  material and never travels", its falsifier unchanged. Twelve rows are
  added: one per refusal plant, caught by refusal; "the widening reading does
  not see the shipped intents", falsifier "widen the shipped row's Positions
  to allPositions and delete the widening-scoped shipped row from
  Exclusions", caught by family absence; "an item from an unscanned row
  carries the mark and a parsed item does not", falsifier "stamp
  `ScanParsed` on every candidate", caught by the scan-mark check; and one
  row per brief chapter admitted here, "brief/00-meta.md is admitted as a
  brief section at every position the brief rows admit" and its three
  siblings, falsifier "delete the row", caught by the carrier floor, class
  `CHAPTER`. `declaredGaps` and `sentinelClasses` are unchanged: No
  sentinel class is added, because a refusal plant and a carrier are not
  sentinels.
- `TestEverySentinelIsPlanted` and `TestEveryAssemblerRuleHasAFalsifier`
  carry the counts above.

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

This spec lands second in Phase A, after the two-operand invocation and
before the preset windows, because the preset-windows eval measures every
committed entry against a table whose widening row set, whose brief rows and
whose manifest shape this spec fixes, and because the comparative channel
moves `SchemaVersion` and `AssemblerVersionCore` again and withdraws the
shipped row and the brief rows from the comparative position; both are
cleaner as a later move over this one than as a merge against it. This spec
names the class of each bump and no number: `SchemaVersion` by one,
`AssemblerVersionCore` MINOR, the widening agent's `prompt_version` PATCH.

## How the Acceptance Criteria are satisfied

- **ac-1 (an unresolvable markdown document is refused, naming document and
  shape).** `verifyRedaction` raises each refusal from
  `unresolvableFrontmatterShape`, `displacedFrontmatter`, `maskMarkupData`'s
  shape return and `rawHeadingTitleEnds`, and `collect` stops the assembly
  on the error, so no item of the document reaches the bundle.
  `TestAnUnresolvableDocumentIsRefusedByName` in `project_test.go` asserts
  the message names the path and the shape and that `Assemble` returns no
  result.
- **ac-2 (the six shapes).** One unit test per shape in `project_test.go`:
  `TestAFenceInsideTheFrontmatterRefuses`,
  `TestADisplacedFrontmatterBlockRefuses` (blank line, whitespace, HTML
  comment; a delimiter after prose does not),
  `TestANestedMappingInASequenceRefuses` (a sequence of scalars does not),
  `TestAFlowExplicitKeyRefuses`,
  `TestAnAttributeValueOnTheNextLineRefuses`,
  `TestAnUnboundedRawHeadingRefusesAndACRLFBlankLineBounds`; and the six
  refusal plants in the eval, run at every assembling position.
- **ac-3 (opted-in items travel whole and marked; no parsed item marked).**
  `collect` passes an unscanned row's document through and stamps the
  candidate; `Assemble` copies the mark.
  `TestOptedInSourceAndTestsTravelWholeMarkedUnscanned` in `size_test.go`
  assembles a fixture under an entry that opts `test` in and one that opts
  `source` in and compares each item's text to the file's bytes and its mark
  to `unscanned`;
  `TestNoParsedItemCarriesTheUnscannedMark` asserts every `.md` item is
  `parsed`. `TestKindSplitDoesNotMoveAdmission` and
  `TestTestKindIsReachableAtEveryPosition` stay green, because the rows
  still admit.
- **ac-4 (the assertion is per item; a markless item is refused).** The
  `Exclusions` doc comment fixes the meaning; `DecodeManifest` refuses.
  `TestDecodeManifestRefusesAnItemWithoutAScanMark` and
  `TestDecodeManifestRefusesAnUnknownScanMark` in `size_test.go`, beside the
  kind refusal; the oracle's `checkScanMarks` and the per-item
  `checkFieldAbsence` in the eval.
- **ac-5 (the fixture leak is absent under every committed entry and marked
  when tests are opted in).** `TestOnlyTheTreePositionsNameSourceOrTest` in
  `scope_test.go` loads the committed file, asserts the detection and
  widening entries name `source` and `test`, the entailment entry names
  neither, and no entry's path reaches `internal/core/site`, then assembles
  the detection and widening entries by dry run and asserts every `source`
  and `test` item in each manifest carries the `unscanned` mark;
  `TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset` in
  `evals/coldreading_test.go` clones `HEAD` detached, assembles the committed
  entry at each of the three assembling positions, asserts no manifest item
  names `internal/core/site/fixture_test.go`, then commits in the clone a
  detection entry that opts `test` in under `internal/core/site`, assembles
  at detection, and asserts the item is present and marked `unscanned`.
- **ac-6 (the table states the narrowing).** `Row.Scan`, the `Floor` column
  in `Render()`, the rewritten rule text and the regenerated charter.
  `TestParsedRowsAdmitOnlyMarkdown`, `TestRenderCoversKindAndSuffix`
  extended, `TestReadingsCharterCarriesTheRenderedIncludeTable`.
- **ac-7 (widening receives no shipped intent; the manifest asserts it;
  entailment and detection still do).** The row's positions, the exclusion
  row and the regenerated definition.
  `TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem` extended,
  `TestExclusionFloorNamesEveryRecordedExclusion`,
  `TestDefinitionHoldsItsFiveParts`, and `TestWideningNeverSeesTheShippedIntents`
  in the eval.
- **ac-8 (the six brief chapters are admitted as brief sections; the
  evidence chapter is excluded as verdict material).** The four new rows and
  the reworded exclusion row. `TestBriefChaptersAreAdmittedAsBriefSections`
  in `include_test.go` asserts a file under each of the six chapters is
  admitted as `brief-section` at every position the product row admits, and
  that the meta row reaches `00-meta.md` and no other file;
  `TestTheEvidenceChapterIsExcludedAsVerdictMaterial` asserts a file under
  `03-evidence` is refused at every position and that the exclusion row's
  rule names verdict material; the `CHAPTER` carrier and the four coverage
  rows in the eval; `TestReadingsCharterCarriesTheRenderedIncludeTable` over
  the regenerated charter.

## Tests

Watched fail before, pass after; each refusal proved by a mutation that
removes it. `internal/core/reading`: the six shape tests and
`TestAnUnresolvableDocumentIsRefusedByName` in `project_test.go`;
`TestParsedRowsAdmitOnlyMarkdown`, `TestEveryRowNamesAKnownKindAndPosition`
(extended), `TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem`
(extended), `TestExclusionFloorNamesEveryRecordedExclusion` (extended),
`TestBriefChaptersAreAdmittedAsBriefSections`,
`TestTheEvidenceChapterIsExcludedAsVerdictMaterial` and
`TestATableChangeMovesTheStampedVersion` (existing) in `include_test.go`;
`TestOptedInSourceAndTestsTravelWholeMarkedUnscanned`,
`TestNoParsedItemCarriesTheUnscannedMark`,
`TestDecodeManifestRefusesAnItemWithoutAScanMark`,
`TestDecodeManifestRefusesAnUnknownScanMark`,
`TestSizeReportCountsUnscannedItems`, `TestRenderCoversKindAndSuffix`
(extended) and `TestBundleGainsNoFieldFromTheReport` (the manifest item
allow-set gains `scan`; the bundle's does not move) in `size_test.go`;
`TestOnlyTheTreePositionsNameSourceOrTest` in `scope_test.go`;
`TestDefinitionHoldsItsFiveParts` in `definitions_test.go`.
`internal/surface/cli`: `TestRenderSizeReportNamesUnscannedItems` (present
above zero, absent at zero). `evals`: the eight refusal plants under
`TestTheAssemblerRefusesAnUnredactableShape`, the `UNSCANNED` and `CHAPTER`
carriers,
`checkScanMarks`, `TestWideningNeverSeesTheShippedIntents`,
`TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset`, and the pinned counts
under `requireOracleTables` and `TestEveryAssemblerRuleHasAFalsifier`. Both
eval lanes are run explicitly and once under `TMPDIR=/tmp`.

## Out of scope

- Widening the floor's parseable set, declined 2026-08-30 and confirmed
  2026-09-02; and removing the `test` kind from the table, declined because
  it contradicts both design documents.
- Recalibrating the committed presets; the preset-windows spec owns the file
  and records which entries name the two tree kinds.
- The four inherited findings: the unbounded opener's cost and the linearity
  test's name (iss-2608301421382564), `opensTag`'s second definition
  (iss-2608301251394412), `rawHTMLHeading`'s doc comment
  (iss-2608301237450573, second half) and the escaped-key refusal's message
  (iss-2608301421381157).
- Withdrawing the shipped row from the comparative position, and the
  comparative channel's own rows, plants and moves of `SchemaVersion` and
  `AssemblerVersionCore`.
- A reading over the assembled output. No reading runs before this intent
  ships, per the intent's last scope condition, and every criterion is judged
  against the assembler's output and its manifest.
