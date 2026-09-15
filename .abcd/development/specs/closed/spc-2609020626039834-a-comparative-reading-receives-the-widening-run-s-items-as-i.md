---
id: spc-2609020626039834
slug: a-comparative-reading-receives-the-widening-run-s-items-as-i
intent: itd-2609020625407419
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The comparative channel: one widening run's candidates, two fields each, characterised before anyone admits one

## Summary

spc-2609020626039834 delivers
[itd-2609020625407419](../../intents/shipped/itd-2609020625407419-a-comparative-reading-receives-the-widening-run-s-items-as-i.md).
`abcd reading assemble --position comparative --target <commit>` derives from
the record the one widening run whose candidates await characterisation, and
the include table gains one row, admitted at the comparative position alone,
that reaches that run's reading records and projects each to the two body
fields the widening position declares, keyed by the item identifier the
comparative body must cite. The invocation stays a position and a target
state: No operand names the run. At the comparative position the
include table is the whole account of what the reading sees: The candidates
pass through the table and the walk as every other row does, every other row
withdraws from the position except the criteria discipline, and the rest of the
readings store is excluded by rows derived from the ledger's own directory
constants. Nothing else from the store travels: No disposition, no admission,
no surprise, no reframe, no other run, no manifest. If any candidate already
carries a disposition or an admission the assembly refuses and names it,
because the candidate set is defined as pre-admission. If the run holds fewer
than two candidates the assembly refuses, names the fixed interpretation, and
stages a comparative run with an empty candidate set whose manifest names the
widening run; ingesting that run commits it, so the outcome of a widening run
is always the same thing: A committed comparative run whose manifest names it.
At ingest, a comparative item naming a candidate outside the recorded run or a
criterion the discipline does not declare is refused by name. The comparative
preset refusal of
[spc-69](../closed/spc-69-a-reading-is-about-something-narrower-than-everything-its.md)
is withdrawn, and the read-block eval gains the comparative rows and plants.

The decision this spec builds on is adopted:
[adr-2609021016272867](../../decisions/adrs/2609021016272867-the-comparative-reading-receives-one-widening-run-s-candidat.md)
carries `status: accepted`, adopted by the maintainer on 2026-09-02 at the
planning interview after being checked against the design framework and the
readings companion, and
[iss-2609020626100041](../../../work/issues/resolved/iss-2609020626100041-adr-owed-the-comparative-channel-admits-two-fields-of-one-wi.md)
is resolved by it. The positional exception to the prior-run exhaust is the
ADR's decision, with the withdrawal of itd-199's ac-10; the same ADR, corrected
on 2026-09-02, fixes the selection of the widening run as a derivation from
the record and adds no operand, because the design fixes the invocation at a
position and a target state
([adr-2609021016286571](../../decisions/adrs/2609021016286571-the-invocation-is-a-position-and-a-target-state-and-the-comm.md))
and brief invariant 15's operand enumeration names two. The assembler selects
the one committed widening run at the target whose items carry no disposition
and no admission; none or more than one refuses, listing the widening runs at
the target with each run's item count and disposition state. The ambiguous
case, two undispositioned widening runs after the closing run, is resolved by
dispositioning one run's items, which is the act the design sequences next.

## Scope

In: The derivation of the candidate run and its two refusals; the candidate
row of the include table, its two-field projection and its selector; the withdrawal of
every other row from the comparative position and the criteria narrowing of
the disciplines row; the ordering guard; the fewer-than-two branch and the
empty comparative run it stages; the candidate kind, the bundle and manifest
additions, and the version classes that move with them; the derived exclusion
rows the comparative position asserts; the committed preset entry and the
withdrawal of the loader's comparative refusal; the criteria discipline as a
required part of the comparative bundle; the two comparative ingest checks and
the empty-run rule; the comparative-run probe the admission gate reads;
the exported durable-tier writer; the definition's Object section; the eval
rows, plants and counts; the plugin page and the brief chapter.

Out: Admission itself and the verb that writes it, which are
[spc-2609020626040342](../open/spc-2609020626040342-an-admission-and-a-surprise-are-written-by-a-verb-and-the-or.md)'s,
including the gate in the shared disposition writer that holds the ruled
ordering; any characterisation performed by the assembler; a candidate set
drawn from more than one run; the preset file's own version, which
spc-2609020626048722 owns; the ADR and the invariant amendment, which are the
maintainer's.

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

This spec lands fourth in Phase A, after the preset windows, whose version 2
shape its entry conforms to, and before the reading-occasioned origin. The
admission and surprise verbs, whose gate reads the probe this spec adds, land
in Phase B; between the phases the maintainer's comparative reading runs over
the channel this spec lands, and the first admissions are hand-authored in
the target format until the verb lands.

This spec's classes: `SchemaVersion` on the bundle and the manifest moves by
one, because the closed `Kind` vocabulary gains `candidate` and both shapes
gain fields (`DecodeManifest` refuses an unknown kind, so an old binary refuses
the new manifest); `AssemblerVersionCore` moves MINOR, because `Table`,
`Exclusions`, `Kinds()` and the bundle shape all move. The comparative preset
entry conforms to the preset file's committed version at landing, so it
carries the three parts spc-2609020626048722's version 2 requires.

### The derived run

`AssembleRequest` in `internal/core/reading/assemble.go` gains nothing, and
the front door in `internal/surface/cli/reading.go` gains no flag: The
invocation is `--position` and `--target`, and `readingOperands` in
`internal/surface/cli/regime_surface_test.go` stays pinned to the two, failing
closed on any addition. At the comparative position `Assemble` derives the
candidate run from the record before anything is resolved.

`reading` gains `WideningRuns(repoRoot, target string) ([]WideningRun, error)`,
with `WideningRun{ID string; Items int; Dispositioned bool; Admitted bool}`;
it walks `.abcd/development/readings/*/run.json` behind the symlink-refusing
walk, keeps the committed runs whose `position` is `widening` and whose
`target_commit` is the resolved target, counts each run's reading records,
and asks `capture.ItemFate` for every item, which `reading` may call because
it already imports `capture` for the ingest. `DeriveCandidateRun(repoRoot,
target string) (WideningRun, error)` applies the ADR's rule over that list: A
run qualifies when it holds items and none of them carries a disposition or an
admission, and exactly one must qualify. With none, the refusal is the typed
error `NoCandidateRun{Runs []WideningRun}`; with more than one it is
`AmbiguousCandidateRun{Runs []WideningRun}`. Both carry the whole listing,
every widening run at the target with its run id, its item count and its
disposition state, rendered one run per line, so the JSON surface carries it
under `widening_runs` and the plain rendering prints it, and the operator sees
what to disposition to make the selection unambiguous; a target with no
committed widening run at all says so in the first refusal. A run whose items
already carry a fate is listed rather than hidden, because that is what the
operator needs to see to understand why it was not selected. The result and
the manifest name the run derived: `AssembleResult` gains `CandidateRun` and
the manifest carries it, below, so a reader of either knows what the assembler
selected and why. `commands/reading.md` documents the rule and the two
refusals, and says that the remedy for an ambiguous selection is to
disposition one run's items, which is the act the design places after the
comparative reading in any case. The block in `Assemble` that refuses the
comparative position before anything is resolved is removed, and
`AssemblingPositions()` returns all four positions.

### The candidate row

`Table` in `internal/core/reading/include.go` gains one row: Positions
`PositionComparative` alone, Source `.abcd/work/issues/readings`, Store `rdi`,
Fields `configuration` and `what_admits_it`, Kind `KindCandidate`, and a Rule
citing adr-2609021016272867. The row's selector is the derived run: The
assembly narrows the row's enumeration to that run's directory before the
preset entry is applied, the way a record in an entry's object set narrows a
row to one record, so the row reaches one run directory and never the
family. That directory is the leaf bucket the readings family keys on,
which is what assembler rule 1 permits an include row to name (ruling (18)),
and the row is refused at every other position by `AdmittedAt` as every row
is. Each item carries the item id as its key, and the two fields resolve
through `projectField` as frontmatter keys, which is what they are on a
reading record. The pattern, the manifest reference, the regime and every
other field of the record have no row and therefore no item: The projection is
the include table's, not a caller's memory.

Because the row is a table row, the rest follows without a second mechanism.
`Admits` answers true for the derived run's records at the comparative
position and false everywhere else; `Render()` carries the row, so the charter
names the channel and `AssemblerVersion` digests it; the comparative
definition's source list is regenerated from the table, which
`TestDefinitionHoldsItsFiveParts` in `internal/core/reading/definitions_test.go`
holds through `statedSources` against `admittedSources`. `Kinds()` gains
`KindCandidate` (`"candidate"`), and the kind is refused as a preset kind by
`validatePresets`, naming it: An entry selects repository material, and a
candidate is selected by the derived run, never by an entry.

The run must have committed: `Assemble` probes
`.abcd/development/readings/<rdg-N>/run.json` and refuses a run without its
commit marker, because a run without one never happened and its records are
the next sweep's to roll back. Every record the row enumerates is validated
through `validateReadingStrict`, and a record whose `position` is not
`widening` refuses the assembly by name: A run at any other position is not a
candidate set.

`internal/core/reading/candidates.go` turns each enumerated record into two
`candidate` items, one per projected field, ordered by id then by field so two
assemblies of one run produce a byte-identical bundle.

### Every other row withdraws

At the comparative position the include table is the whole account, and the
readings companion admits no source but the candidates and the criteria. Every
row that still carries the comparative position loses it: The seven brief and
glossary rows (itd-194's six chapter rows and the glossary row), the
shipped-intents row, which after itd-194 carries entailment, comparative and
detection, the specs row and the four shipped-tree rows. The disciplines row keeps it, and at the comparative
position the assembler narrows that row's enumeration to `CriteriaDiscipline`
(`"itd-191"`) before the preset entry is applied, so no entry can widen the
comparative material past the criteria. The consequence is deliberate: An
entry naming `source` or a path at the comparative position selects nothing
outside the criteria discipline, and the entry's meaning at this position is
exactly what the committed entry below names.
`TestThreePositionsCarryDistinctItemSets` extends to four positions on that
basis.

### The ordering guard

`capture` gains `ItemFate(repoRoot, run, item string) (ItemFate, error)`, the
one admitted-proposal probe in this binary: It returns the item's standing
disposition ids (through `standingDispositions` and
`issueschema.StandingDispositionIDs`) and the ids of every admission under
`admissions/<run>/` whose `proposal` names the item, keyed on the (run,
proposal) pair exactly as `admittedProposals` in
`internal/core/lint/readingoutstanding.go` keys it. The judgement, given a
disposition set and an admission set, lives in `issueschema` beside the
standing-disposition judgement; the walk lives in `capture` once, and
spc-2609020626040342's `Admit` calls the same function rather than probing the
store again. `Assemble` asks it for every candidate before anything is minted
or written. Any standing disposition, in any state, refuses and names the item
and the disposition; any admission refuses and names the item and the
admission; a contested or cyclic disposition set refuses and names the item,
because a candidate whose fate cannot be read is not demonstrably uncommitted.
The refusal states the rule: The candidate set is defined as pre-admission, and
characterisation precedes admission.

The ruled ordering itself is not held here. spc-2609020626040342 puts one gate
in the shared disposition writer, `writeDispositionLocked`, so that at the
widening position no disposition in any state can be written until a committed
comparative run names the item's run. This guard is therefore unreachable
through any verb and stays as a guard against hand-written records: The
assembler cannot know a record was placed by hand, so it refuses what it finds.
There is no deadlock between the two, because the writer's gate opens on the
comparative run this assembly produces, and the closing run at position 4 is
repeatable for the same reason: Dispositions cannot exist before a comparative
run, and a second comparative run over the same widening run before any
disposition is a second run. A comparative reading commissioned after
dispositions is not a repeat of this position; recurrence is warm work.

### The fixed interpretation and the empty comparative run

Fewer than two candidates refuses the assembly, and the refusal names the
interpretation: The comparative reading has nothing to compare and is not
exercised. Unless the invocation is a dry run, the assembly still stages a run
directory, on the same rules as any other, whose bundle carries no candidate
item and whose manifest carries `candidate_run: <rdg-N>`, `candidates` at the
count of items the derived run holds, which is one for a one-item run, and
`exercised: false`, and the result reports `NotExercised` with the same count.
The count is never written as zero for a run that holds an item: `candidates`
is defined as the derived run's item count, and `exercised` is the field that
says the position was not exercised. That run
is committed by `abcd reading ingest` with an empty item list, which is the
clean-run idiom ruling (3) already fixed: A run recorded with an empty item
set. `checkEnvelope` in `internal/core/reading/ingest.go` refuses an empty item
set today, and on correction (4) of the 2026-09-02 ruling that refusal goes:
The framework's section 13 records a run that returns no items as a run with
an empty item set at every position, so the ingest commits such an output as
a run record with an empty item list at widening, entailment, comparative and
detection alike, and never refuses it. Refusal is reserved for a malformed
payload, which `checkEnvelope` goes on refusing by name (a missing field, an
unknown key, a regime the position does not declare, an item that fails
`checkItem`). The not-exercised comparative run is one instance of the
general rule and not a carve-out: Its manifest carries `exercised: false`,
and an empty item set at any other position is the clean run the framework's
contingency describes. `TestIngestCommitsAnEmptyRunAtEveryPosition` in
`ingest_regime_test.go` commits an empty output at each of the four positions
and reads the run record back with an empty item list, and
`TestIngestStillRefusesAMalformedEmptyPayload` proves the refusal survives
for a payload that is empty and malformed. This resolves
[iss-2609021153269181](../../../work/issues/resolved/iss-2609021153269181-the-ingest-verb-refuses-an-output-carrying-no-items-where-th.md).
The outcome of a widening run is therefore one thing whether the position ran
or was not exercised: A committed comparative run whose manifest names that
widening run. The durable tier stays write-once at
emission; no other run's directory is ever amended, and a second assembly over
the same one-item run is a second run.

`ReadingsRecordDir` moves to `issueschema` in the same change and
`reading.ReadingsRecordDir` becomes an alias of it, so the two packages name
one directory. `issueschema` declares `RunHead{RunID, Position, CandidateRun}`,
the strictly decoded subset of the run record that answers "which widening run
did this comparative run characterise". `capture` gains
`ComparativeRunFor(repoRoot, run string) (string, error)`, which walks
`.abcd/development/readings/*/run.json` behind the symlink-refusing walk that
package already carries, reads each behind `fsutil.ReadGuarded`, and returns
the lowest comparative run id whose `candidate_run` names the run, or nothing.
That is the one probe the disposition writer's gate reads, and `capture` can
call it because it does not import `reading`.

### The exported durable-tier writer

`reading` exports `WriteRunArtefact(repoRoot, runID, name string, v any)
(string, error)`, the one way a package outside `reading` writes beside a
committed run. It opens the containment root as `Ingest` does, requires the
run's `run.json` to exist, refuses a name that is one of the run's own three
files or that is not a plain `.json` basename, refuses an existing file
because the durable tier is write-once at emission, and writes through
`writeJSONIn`. spc-2609020626045177's `scribe ingest` promotes its manifest
through it; nothing in this spec calls it, and it exists here because the root
and the write are this package's.

### The bundle, the kind, the entry, and the criteria

`BundleItem` in `internal/core/reading/manifest.go` gains `Candidate string`
and `Field string`, both `omitempty`, both set only for a candidate item: The
reading is told which `rdi-N` each text belongs to and whether it is the
configuration or what admits it, and nothing else. Neither is a repository
path, so brief invariant 15 holds, and `TestNoBundleFieldIsAScopeSelector` in
`internal/core/reading/scope_test.go` gains the two members. At every other
position both are empty, and a test pins that. The closed bundle-item allow-set
in `TestBundleGainsNoFieldFromTheReport` in `internal/core/reading/size_test.go`
gains `candidate` and `field`; the bundle's own allow-set there does not move
for this spec.

`validatePresets` in `internal/core/reading/scope.go` stops refusing a
comparative entry, and `.abcd/config/reading-presets.json` gains the entry
for `comparative` in the three-part shape spc-2609020626048722 fixes: An
object set of `records: ["itd-191"]` and no path, kinds `["discipline"]`, and
a `window` measured in a clean clone. spc-2609020626048722's window eval
iterates the three assembling positions by name and states that the
comparative entry is exempt, because its object is bounded by the widening
run rather than by the tree; this spec's own eval covers the comparative
position: `TestComparativeEntryFitsItsDeclaredWindow` in
`evals/coldreading_test.go` assembles the comparative position over the
fixture widening run by dry run and compares the measured figure against the
committed entry's declaration, failing on a breach with the same three facts
the window eval names. The entry keeps its meaning: It selects the repository material
passed beside the candidates, which at this position is the discipline and
nothing else, and a change to it is a commit.

The criteria are never supplied at invocation and a comparative bundle without
them characterises against nothing, so `reading` declares
`CriteriaDiscipline = "itd-191"` and `Assemble` refuses a comparative assembly
whose selected items carry no item whose path names that record. The declared
criteria are read at assembly by `declaredCriteria` in
`internal/core/reading/criteria.go`: The bullets of the discipline's
`## The rule` section, each name being the text before the first em dash. A
test pins the committed
[itd-191](../../intents/disciplines/itd-191-the-selection-criteria-are-a-declared-recorded-discipline-a.md)
to its six names, so an amendment to the slate is a recorded change here too.

The comparative definition's Object section in `agents/cold-reading-comparative.md`
names both the derived candidate run and the discipline and loses its two
does-not-assemble paragraphs; the precedence sentence stays, and its source
list is regenerated from the table.
`TestComparativeObjectIsTheWideningPreAdmissionOutput` in
`internal/core/reading/definitions_test.go` moves with it.

### The manifest and the exclusions it asserts

`Manifest` gains `CandidateRun` (the run derived, with the JSON key
`candidate_run`), `Candidates` (the count of items the derived run holds,
which is the count of candidates the bundle carries when the position is
exercised and stays the run's count when it is not), `Exercised` (with the
JSON key `exercised`: True when the run holds two or more candidates and the
bundle carries them, false on the staged empty run; written whenever
`candidate_run` is, so a false value is a statement and not an absence),
`CandidateFields` (the two projected field names) and `Criteria` (the parsed
names), the last two `omitempty`, and `ManifestItem` gains `Candidate` beside
the `Field` it already carries. The
manifest therefore records what the assembler selected and what it weighed,
and a reader can check the selection against the record: The run named is the
one committed widening run at the manifest's target whose items carried no
fate when the assembly ran. `RunRecord` in
`internal/core/reading/ingest.go` carries `CandidateRun`, `Candidates` and
`Exercised` forward, so a committed comparative run says which widening run it
characterised, how many candidates that run held, and whether it was
exercised.

The exclusion floor's row for `.abcd/work/issues` narrows to the three other
positions. At the comparative position the directory rows are derived, never
listed: `issueschema` gains `LedgerDirs()`, returning every ledger directory
constant it declares (`StatusDirs`, `ReadingsDir`, `DispositionsDir`,
`AdmissionsDir`, `SurprisesDir`, and spc-2609020626048705's `ReframesDir` once
it lands), and `Exclusions` gains one comparative-only directory row per entry
except `ReadingsDir`, whose row carries the signal `readings store` and the
detail "every run other than the candidate run, and every field of the named
run's items other than configuration and what_admits_it". A family the ledger
adds later is excluded here the day its constant is declared, and the scribe's
allow list in spc-2609020626045177 derives from the same function, so the two
instruments describe one set. `assertExclusions` enforces the directory rows
as it does today; `assertCandidateProjection` is the fail-closed half of the
signal row, refusing to emit any candidate item whose path is not under the
derived run's directory or whose field is not one of the two. An assertion the
manifest declares and the assembler cannot violate is what adr-56 asks of an
exclusion control. `.abcd/development/readings` stays excluded at every
position: A run's own manifest and `run.json` never travel.

### Ingest at the comparative position

`validateItems` in `internal/core/reading/ingest_regime.go` takes the
manifest: Its signature becomes `validateItems(out Output, m Manifest, def
Definition)`, because the two checks below are against the manifest and the
function is handed none today. Both run after `checkItem`, at the comparative
position only. `candidate_id` must equal the `Candidate` of some manifest
item, refused as `unknown-candidate` naming the item's ordinal and the id.
`criterion` must equal, after `foldForMatching`, one of `manifest.Criteria`,
refused as `undeclared-criterion` naming the ordinal and the criterion. The
reserved names and the ordering signatures of the evaluative regime are
untouched. The supply-regime check as this tree stands is the reserved-name
refusals, enforced, and the four prose detectors, flagging; the case report
describes whichever state the opening target carries, on correction (6) of
the 2026-09-02 ruling.

On the commit path nothing outside the run's own directory is written: The run
record carries `candidate_run`, and the admission gate finds it through
`ComparativeRunFor`. No file in the widening run's directory is touched.

### The read-block eval

The fixture under `evals/testdata/cold-reading/baseline/` gains a committed
widening run of three items at the fixture's target, a second widening run at
the same target whose items all carry dispositions, admissions and a surprise
(so the first run is the one the assembler derives), and a variant carrying a
disposition on one candidate of the derived run. Three sentinel classes are added:
`CANDIDATE`, the carrier, planted in the `configuration` of the derived run's
items and required to arrive at the comparative position; `ENVELOPE`, planted
in every candidate's `pattern` and required never to arrive; and `FATE`,
planted in the second run's dispositions and surprise (the leak check) and in
the variant's disposition on a candidate (the refusal check). `EXHAUST` gains
the second run's items as homes. The oracle stays transcribed: No file under
`evals/` imports the assembler, so `excludedFamilies` is edited by hand to
mirror the derived rows, and spc-2609020626048705 adds its row when it lands.

The pinned counts move as follows, each stated as the delta this spec makes,
since itd-194 and the preset windows move some of the same tables first and
the merging change sets each literal from the merged base. `sentinelClasses`
+3. `excludedKeys` unchanged: The envelope's keys are excluded by the
candidate row's projection and caught by plant, not by the key table.
`excludedFamilies` +6: The `.abcd/work/issues` row gains the three other
positions, and six comparative-only rows are added for `open`, `resolved`,
`wontfix`, `dispositions`, `admissions` and `surprises`. `admittedRecordPaths`
+1: The fixture's candidate run directory at the comparative position.
`coverage` +9, with these rows:

- The candidate run's items are admitted at the comparative position,
  projected to configuration and what admits it. Falsifier: Delete the
  candidate row from `Table`. Caught by the carrier floor; class `CANDIDATE`.
- The candidate row is admitted at no other position. Falsifier: Add the three
  other positions to the row. Caught by family absence; class `CANDIDATE`.
- The candidate projection drops the envelope. Falsifier: Empty the row's
  `Fields`. Caught by sentinel absence; class `ENVELOPE`.
- No run other than the derived candidate run reaches the comparative
  reading. Falsifier: Drop the run narrowing from the candidate selector.
  Caught by sentinel absence; class `EXHAUST`.
- Dispositions never reach the comparative reading. Falsifier: Delete the
  derived dispositions row and add an include row for it. Caught by sentinel
  absence; class `FATE`.
- Admissions never reach the comparative reading. Falsifier: Delete the
  derived admissions row and add an include row for it. Caught by sentinel
  absence; class `GROUNDS`.
- Surprises never reach the comparative reading. Falsifier: Delete the derived
  surprises row and add an include row for it. Caught by sentinel absence;
  class `FATE`.
- The status directories never reach the comparative reading. Falsifier:
  Delete the three derived status rows and add an include row under
  `.abcd/work/issues`. Caught by sentinel absence; class `DECISION`.
- A candidate carrying a standing disposition or an admission refuses the
  assembly. Falsifier: Delete the `ItemFate` call. Caught by refusal; class
  `FATE`.

`declaredGaps` unchanged: Every row above is caught. `evals/coldreading_test.go`
gains the comparative cases named below; `TestComparativeRefusesToAssemble` is
retired with the refusal it guarded, and `TestEverySentinelIsPlanted` counts
the three new classes.

## How the Acceptance Criteria are satisfied

- **ac-1**: `DeriveCandidateRun` selects the one committed widening run at
  the target whose items carry no disposition and no admission; with none it
  refuses as `NoCandidateRun` and with more than one as
  `AmbiguousCandidateRun`, each listing the widening runs at the target with
  their item counts and disposition state, and the manifest records the run
  derived. Tests: `TestComparativeDerivesTheOneUndispositionedWideningRun`
  (two runs planted, one with a dispositioned item; the other is selected and
  the manifest names it), `TestTwoUndispositionedWideningRunsRefuseNamingThem`
  (both listed with counts and state), `TestNoQualifyingWideningRunRefuses`
  (one run planted, every item dispositioned; listed, and the refusal says why
  it did not qualify), and `TestNoWideningRunAtTheTargetRefusesSayingSo`.
- **ac-2**: The candidate row enumerates the derived run and projects two fields;
  `candidates.go` emits two items per candidate; the bundle carries
  `Candidate` and `Field` and the discipline item, and no other row is
  admitted. Tests: `TestComparativeBundleCarriesTwoFieldsPerCandidate`,
  `TestComparativeBundleCarriesTheCriteriaDiscipline`,
  `TestComparativeAdmitsNoOtherRow`.
- **ac-3**: `ItemFate` reports a standing disposition; `Assemble` refuses
  naming the item. Test: `TestComparativeRefusesADispositionedCandidate`.
- **ac-4**: `ItemFate` reports an admission keyed on (run, proposal);
  `Assemble` refuses naming the item. Test:
  `TestComparativeRefusesAnAdmittedCandidate`.
- **ac-5**: The fewer-than-two branch refuses, names the interpretation, and
  stages a comparative run with an empty candidate set whose manifest names
  the widening run; `reading ingest` commits it with an empty item list.
  Tests: `TestOneCandidateStagesAnEmptyComparativeRun`,
  `TestIngestCommitsAnEmptyComparativeRun`, and the general rule it is one
  instance of, `TestIngestCommitsAnEmptyRunAtEveryPosition`.
- **ac-6**: The candidate row reaches one run directory and two fields; the
  derived exclusion rows assert the rest; `assertCandidateProjection` refuses
  any breach. Tests: `TestComparativeManifestAssertsTheDerivedExclusions`,
  `TestCandidateProjectionRefusesAForeignRun`.
- **ac-7**: The `unknown-candidate` check against the manifest's candidate set.
  Test: `TestIngestRefusesACandidateOutsideTheRun`.
- **ac-8**: The `undeclared-criterion` check against `manifest.Criteria`. Test:
  `TestIngestRefusesAnUndeclaredCriterion`.
- **ac-9**: The eval plants the `FATE` sentinel on a candidate's disposition
  and the `ENVELOPE` sentinel on every candidate's pattern; the assembly must
  refuse, and if it ever assembles the sentinel-absence oracle reports the leak
  by class and position. Test: `TestComparativeChannelCatchesAPlantedFate`.

## Tests

Watched fail before, pass after; each refusal proved by a mutation that removes
it.

- `internal/core/reading/candidates_test.go`:
  `TestComparativeDerivesTheOneUndispositionedWideningRun`,
  `TestTwoUndispositionedWideningRunsRefuseNamingThem`,
  `TestNoQualifyingWideningRunRefuses`,
  `TestNoWideningRunAtTheTargetRefusesSayingSo`,
  `TestAWideningRunAtAnotherTargetIsNotDerived`,
  `TestComparativeBundleCarriesTwoFieldsPerCandidate`,
  `TestComparativeBundleCarriesTheCriteriaDiscipline`,
  `TestComparativeAdmitsNoOtherRow`,
  `TestComparativeRefusesWithoutTheCriteriaDiscipline`,
  `TestComparativeRefusesADispositionedCandidate`,
  `TestComparativeRefusesAnAdmittedCandidate`,
  `TestComparativeRefusesAnUncommittedRun`,
  `TestComparativeRefusesANonWideningRun`,
  `TestOneCandidateStagesAnEmptyComparativeRun`,
  `TestAnEmptyComparativeRunIsNotStagedOnADryRun`,
  `TestComparativeManifestAssertsTheDerivedExclusions`,
  `TestCandidateProjectionRefusesAForeignRun`,
  `TestCandidateFieldsAreEmptyAtEveryOtherPosition`,
  `TestTwoComparativeAssembliesAreByteIdentical`.
- `internal/core/reading/include_test.go`: `TestCandidateRowIsComparativeOnly`,
  `TestEveryOtherRowWithdrawsFromComparative`,
  `TestComparativeExclusionRowsAreDerivedFromLedgerDirs`,
  `TestRenderCarriesTheCandidateRow`.
- `internal/core/reading/criteria_test.go`:
  `TestDeclaredCriteriaReadsTheCommittedDiscipline` (six names),
  `TestDeclaredCriteriaRefusesAnEmptySlate`.
- `internal/core/reading/scope_test.go`: `TestComparativeRefuses` is replaced by
  `TestComparativePresetIsAdmitted`;
  `TestThreePositionsCarryDistinctItemSets` extends to four positions;
  `TestNoBundleFieldIsAScopeSelector` covers `Candidate` and `Field`;
  `TestCandidateIsNotAPresetKind`.
- `internal/core/reading/size_test.go`: `TestBundleGainsNoFieldFromTheReport`
  with the item allow-set extended.
- `internal/core/reading/ingest_regime_test.go`:
  `TestIngestRefusesACandidateOutsideTheRun`,
  `TestIngestRefusesAnUndeclaredCriterion`,
  `TestIngestCommitsAnEmptyComparativeRun`,
  `TestIngestCommitsAnEmptyRunAtEveryPosition`,
  `TestIngestStillRefusesAMalformedEmptyPayload`.
- `internal/core/reading/ingest_test.go`: `TestWriteRunArtefactIsWriteOnce`,
  `TestWriteRunArtefactRefusesAnUncommittedRun`,
  `TestWriteRunArtefactRefusesTheRunsOwnFiles`.
- `internal/core/capture/reading_test.go`: `TestItemFateReadsBothStores`,
  `TestComparativeRunForNamesTheLowestMatch`,
  `TestComparativeRunForIsEmptyBeforeAnyRun`.
- `internal/core/issueschema/disposition_test.go`: `TestItemFateJudgement`;
  `internal/core/issueschema/statusdirs_test.go`: `TestLedgerDirsNamesEveryConstant`.
- `internal/surface/cli/regime_surface_test.go`: `readingOperands` stays at
  `position`, `target`, and the pin fails closed if this spec's landing adds
  one; `TestComparativeRefusalRendersTheWideningRuns` asserts the plain and
  JSON renderings of both refusals carry the listing.
- `evals/coldreading_test.go`:
  `TestComparativeChannelCarriesCandidatesAndNothingElse` (dispositions,
  admissions, a surprise and a second run planted, the second run's items all
  dispositioned so the first is the one derived; the bundle carries that run's
  two candidate fields and the discipline, the carrier arrives, no other
  sentinel does, and the manifest asserts the derived exclusions),
  `TestComparativeChannelCatchesAPlantedFate` (ac-9),
  `TestComparativeEntryFitsItsDeclaredWindow` (the fixture widening run
  assembled at the comparative position, measured against the committed
  entry's declaration).
  `TestComparativeRefusesToAssemble` is retired with the refusal it guarded;
  `TestEverySentinelIsPlanted` and `TestEveryAssemblerRuleHasAFalsifier`
  carry the counts above.

## Out of scope

- The verb that admits a candidate, and the gate in the shared disposition
  writer that holds the ruled ordering, which spc-2609020626040342 builds.
- A candidate set from more than one run, or from a run at any other position;
  and any operand that names the run, which adr-2609021016286571 closes.
- The pattern named by a widening item. It stays behind on the ground the
  intent's scope condition states; widening the projection is a manifest change
  and a recorded one.
- The preset file's version and the window eval, which spc-2609020626048722
  owns; this spec adds one entry in the committed shape.
- The ADR and the amendment to brief invariant 15, both adopted on
  2026-09-02 (adr-2609021016272867, `status: accepted`); this spec builds on them and does
  not restate them.
- The re-disposition of itd-199's surviving condition cond-2608312031028702,
  which this spec's delivery moves. spc-2609020626046252's verb records it with
  the shipped intent as the occasion, and that verb lands first in Phase B, so
  the re-disposition waits for it: Between this spec's landing and the
  condition verb's, the condition stands as itd-199's verdict left it.
