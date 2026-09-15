---
id: spc-2609020626048722
slug: a-committed-preset-declares-the-window-it-was-calibrated-for
intent: itd-2609020625400445
origin: researcher-authored
production_mode: dictated-and-formatted
---
# One committed entry per position, in three parts: the object set, the kinds within it and the window it measures at, held by an eval that re-measures every entry on every committed change

## Summary

spc-2609020626048722 delivers
[itd-2609020625400445](../../intents/shipped/itd-2609020625400445-a-committed-preset-declares-the-window-it-was-calibrated-for.md).
The committed preset file `.abcd/config/reading-presets.json` moves to schema
version 2, whose shape is the one the design admits: One entry per position,
applied by the assembler with no operand
([adr-2609021016286571](../../decisions/adrs/2609021016286571-the-invocation-is-a-position-and-a-target-state-and-the-comm.md)),
and each entry in three committed parts. The **object set** is the framework's
own term for what a run is about: Which records and which delivered paths. The
**kinds** are the material admitted within it: Every kind the position's
definition reads, when the object set fits the reader's window, with `source`
and `test` items marked `unscanned` by itd-194 wherever an entry names them,
and the brief's six chapters admitted as brief sections by the same intent.
The **window** is the estimated-token figure the entry measured, beside the
commit it measured on, against a target of two hundred thousand estimated
tokens that is stated and never enforced. A version 1 file
goes on loading when it holds one preset. The size report
[spc-68](../closed/spc-68-an-assembly-reports-what-it-would-cost-before-a-reading-is.md)
added carries the declaration beside the measurement, and a new eval in the
cold-reading lane assembles the entry for each of the three assembling
positions by dry run and fails when a result exceeds its declaration.

For Iteration 2 the default object set is Iteration 1's shipped state, which
in this repository is the ledger: The fifteen workstream intents, their
fifteen specs, and the packages and pages they delivered that the deny list
does not exclude. The default kinds are all the kinds a position's definition
reads within that object set, on the maintainer's ruling of 2026-09-02 that a
reading of the object set includes all kinds when the object set fits the
reader's window, with the size report stating any figure over the target. The
detection and widening entries therefore name `source` and `test` beside the
record and documentation kinds, because each definition's object closes with
the shipped tree, and measure about 624,000 and about 615,000 estimated
tokens: Over the target, said so on every assembly, and inside a million-token
reader. The entailment entry names no tree kind, because its definition's
object holds no tree, and measures lower. This spec states the figures rather
than narrowing the object to make them fit. At the entailment
position the size report also states the mechanism proportion the readings
companion asks for beside the findings, which resolves iss-2609012259585189.
The maintainer also ruled on 2026-09-02 that the preset entry is the one
configuration surface for a position's object set, kinds and window, so the
change that delivers one entry per position resolves
[iss-2608311501240566](../../../work/issues/resolved/iss-2608311501240566-three-of-the-four-reading-positions-receive-a-byte-identical.md):
each position is handed its own entry, the default item set at each position
is pinned by a digest recorded before the change, and two assemblies of one
entry produce identical bundles, under the tests named below.

A change to any part of an entry is a commit to the preset file, reviewed and
inside the dirty gate, and the manifest names the entry applied and its hash.
That is how the object set grows, how a kind that proves useless (tests, say)
leaves the default entry for a position, and how a window moves: Each is a
recorded change, never a flag. There is no override at invocation and nothing
to stamp.

The assembler still enforces no budget at invocation. The entry declares, the
eval checks, and the report says what a run would cost. The selector grammar
of [spc-69](../closed/spc-69-a-reading-is-about-something-narrower-than-everything-its.md)
is not extended: An entry selects by kind, by record id and by path, and
`.abcd` stays in `denySegments`, so no path can reach a record bucket.

## Scope

In: The version 2 shape and its loader in `internal/core/reading/scope.go`,
including the kinds-admit and object-narrows rule; the declaration, the
over-target line and the entailment mechanism proportion on `SizeReport` in
`assemble.go`; the CLI and plugin renderings; the default object set and the
default kinds for the three assembling positions; the eval
`evals/coldreading_window_test.go` and the guard that proves it can fail; the
unit tests for a record selector's positive half.

Out: A budget at invocation; a tokenizer; the comparative position, whose
object arrives by its own channel
(spc-2609020626039834, the comparative channel) and is bounded by the widening
run rather than by the tree, and whose entry that spec adds in the shape this
spec fixes; the invocation itself and the loader's `PresetFor(position)`,
which the two-operand spec (spc-2609021004075744) lands first;
the eval lane becoming a required check
([iss-2608311632382737](../../../work/issues/resolved/iss-2608311632382737-the-pre-push-gate-is-blind-to-both-eval-lanes-so-the-read-bl.md),
[iss-2608311051046981](../../../work/issues/resolved/iss-2608311051046981-the-new-cold-reading-evals-ci-job-is-not-a-required-status-c.md)),
which is the gate work the enforcement claim below depends on.

## Approach

### The preset file at schema version 2

```json
{
  "schema_version": 2,
  "positions": {
    "detection": {
      "comment": "The object set is Iteration 1's shipped state; the kinds are every kind the detection definition reads, source and test included and marked unscanned; the window is what this entry measured, over the 200,000 target and inside a million-token reader. Changing any part is a commit. ...",
      "object": {
        "records": ["itd-177", "itd-178", "...", "itd-199", "spc-55", "...", "spc-69"],
        "paths": ["internal/core/capture", "...", "commands/intent.md"]
      },
      "kinds": ["brief-section", "glossary-term", "discipline", "spec", "intent-projection", "doc", "config", "source", "test"],
      "window": {
        "tokens_est": 630000,
        "measured_tokens_est": 624000,
        "measured_bytes": 2402000,
        "measured_at": "<sha>"
      }
    }
  }
}
```

The `measured_*` values shown are the estimates of the measurement section
below, rounded; the implementer replaces them with the figures the landing
commit measures.

The named `presets` map, its `cold` and `warm` keys and `extends` are gone:
There is one entry per position and nothing to choose between at the
invocation. `PresetFile` becomes `SchemaVersion int` and
`Positions map[string]PositionEntry`; `PositionEntry` is `Object ObjectSet`,
`Kinds []Kind`, `Window *Window` (`json:"window,omitempty"`) and
`Comment string` (`json:"comment,omitempty"`); `ObjectSet` is
`Records []string` and `Paths []string`. `Window` is `TokensEst int`,
`MeasuredTokensEst int`, `MeasuredBytes int`, `MeasuredAt string`.
`tokens_est` is the declaration, on the size report's own basis (bytes divided
by 3.85, per spc-68). The three `measured_*` keys are disclosure and nothing
gates on them beyond shape: `measured_at` must match the assembler's
`targetRe`, and its reachability is deliberately not checked, because a
squash or rebase merge rewrites a branch sha out of existence and a
disclosure that fails the build after one would teach people to omit it.
`comment` is free text nothing reads except a reviewer; the loader declares
it so the strict decoder admits it.

Every load-time refusal of spc-69 survives on the new shape: An uncommitted
edit, a symlink, a path outside the tree, a denied segment, a duplicate key
and an unknown kind each refuse by name.

### The three parts, and how they select

The kinds admit and the object set narrows. An entry's `kinds` list names
every include-table kind the position may be handed, and nothing outside it
travels. Within that, the object set narrows two ways. A record row of the
include table (the shipped, drafts, planned, specs and disciplines rows) is
narrowed to the object set's `records` when the object set names any record
under that row's source, and is admitted whole when it names none: The brief,
glossary and discipline kinds a definition names as constraint sources are
therefore handed whole at the three assembling positions, while the `spec`
and `intent-projection` kinds are handed for the object set's records alone.
A tree row (the `doc`, `config`, `source` and `test` rows at the repository
root) is narrowed to the files at or beneath the object set's `paths`, and an
entry with no path hands nothing from the tree whatever kinds it lists. This
replaces spc-69's union, under which a path selected every kind beneath it:
A path in the object set now selects only the kinds the entry admits, which is
what lets the object set stay one fact in every entry while each position's
kinds follow its own definition. The detection and widening entries name
`source` and `test` over the seven delivered paths; the entailment entry names
the same paths and no tree kind, so it hands nothing from them. The rule lives in
`PositionEntry.selects`, beside the record and path predicates
`pathNamesRecord` and `underPath` it already has.

`PresetFor(position)`, which the two-operand spec introduces, returns the
entry for the position and refuses when the file holds none. `PresetWindow(pf,
position) *Window` returns that entry's declaration. The `Preset` block the
manifest carries under that spec, the entry applied as selectors and its hash,
gains nothing here: The window rides on the size report, not on the manifest,
so the manifest's shape and hash do not move.

The hash is also what pins the closing run to the opening run. The design
framework's section 13 requires the closing run to be over "the same object
set", and the object set is the entry's records and paths, which a commit to
the file can move between the two runs. So the closing run at each position
uses the entry at the same hash the opening run's manifest records under
`preset_hash`, and where a prior run at the same position recorded a
different hash the closing run record's `bounds` list, defined below, states
the mismatch as a bound on the comparison,
on ruling M5 that a reading which departs from the one the documents state is
"stated as a bound rather than passed off", and never silently. Nothing at the
invocation can choose the entry, so the pin is a check on the record and not an
operand: An operator who moves the entry between the runs has made a recorded
change, and the closing run's record says the comparison crosses it. The pin is
registered as an interpretation, because neither document says what holds the
object set still between two runs.

### Loading: two versions, one strict

`PresetSchemaVersion` becomes 2 and a `presetSchemaVersions` set holds 1 and
2. `LoadPresets` accepts either, then applies the version-2 rule: Every
position entry must carry `window` with `tokens_est` greater than zero,
`measured_tokens_est` and `measured_bytes` at least zero, and a well-formed
`measured_at`; a missing window refuses with the position ("the entry for
entailment declares no window; at schema_version 2 every position states the
window it was calibrated for"). A version 1 file, in spc-69's named shape, is
read into the same `PositionEntry` set when it holds exactly one preset: Its
`kinds`, `records` and `paths` become that preset's entries, with the records
and paths read as the object set, no window is declared, and the size report
says so. A version 1 file holding more than one preset refuses, naming them,
because nothing at the invocation can choose between them and the design
admits no operand that could. A version 1 file declaring a window is refused
as an unknown field by the decoder it already has, so the two shapes cannot
be mixed.

### The size report carries the declaration

`SizeReport` in `assemble.go` gains `Window *Window` (`json:"window,omitempty"`)
and `ExceedsWindow bool`, filled from `PresetWindow` in `Assemble`. The report
lives on `AssembleResult`, not on either artefact, so this part moves
neither `SchemaVersion` nor `AssemblerVersionCore`. `renderSizeReport` in
`internal/surface/cli/reading.go` gains one line after the total:
`window:        630,000 tokens declared (measured ~624,000 at <sha>); this run is within it`,
or `window:        none declared (preset schema version 1)`; a run over the
declaration says `EXCEEDS` on the same line. `commands/reading.md` adds
`size.window` and `size.exceeds_window` to the fields the host reports.

### The default object set, and why

The framework's section 13 fixes the object set of Iteration 2's opening
readings as Iteration 1's shipped state, and the design fixes the object set
as a committed fact the definition and the record supply rather than
something typed. In this repository that state is:

- The fifteen shipped intents of the cold-reading workstream: `itd-177` to
  `itd-189`, `itd-198` and `itd-199`.
- Their fifteen specs: `spc-55` to `spc-69`. A record selector matches the
  record's filename by id whatever bucket it sits in, so a listed record
  follows its file between buckets without an edit.
- The packages and pages those intents delivered that the deny list does not
  exclude: `internal/core/capture`, `internal/core/issueschema`,
  `internal/core/intent`, `internal/core/provenance`, `internal/core/grounds`
  and `internal/core/lint`; `internal/surface/cli/reading.go`; and
  `commands/reading.md`, `commands/capture.md` and `commands/intent.md`. Each
  exists at the tree this spec was written against and none lies under a
  denied segment or prefix. Two delivered paths are excluded structurally and
  are therefore not named: `agents/scribe.md`, because `agents` is a denied
  segment and the exclusion floor's row "the instrument's own output is never
  its input" binds it at every position, and the assembler's own package
  `internal/core/reading`, which `denyPrefixes` holds for the same reason.

The ten planned intents of this workstream, the eight of Iteration 2 together
with itd-194 and the two-operand intent, are outside the object set's record
list: They are this iteration's own work, not Iteration 1's shipped state, and
the closing run the interpretations schedule is over the same object set as
the opening readings. They are not outside every reading, though, and the
difference is stated here so nobody reads the list as the whole account. Under
the rule above, a record row is admitted whole when the object set names none
of its records, and the object set names no draft and no planned intent; so at
the entailment position, whose include table admits the drafts and planned
rows, the ten planned intents are admitted whole and do reach the entailment
reading, inside that position's admission and outside the object set's record
list. That is what the readings companion's section 6.2 asks, since it admits
drafts and planned intents to the entailment reading. At the widening and
detection positions the drafts and planned rows are not admitted, so the ten
reach neither. Adding them to the record list, or anything else, is a commit
to the file that the eval re-measures.

Which of the object set each position receives is the include table's
decision, not the entry's. At the widening position the shipped-intents row
excludes the position (itd-194, on the 2026-09-02 ruling) and the widening
entry's kinds name `spec` and not `intent-projection`, so the object set's
fifteen specs reach it, no intent does, and its constraint sources travel
whole. At the entailment and detection positions the records travel as their
projections. The object set's paths hold, at the tree named below, no
documentation and no configuration file inside the six packages or beside the
CLI file: Their content is Go source and Go tests, which the detection and
widening entries name and the entailment entry does not, so the seven paths
hand each of the detection and widening positions forty-nine source and
eighty-six test items, every one marked `unscanned` in the manifest under
itd-194, and hand the entailment position nothing. The three command pages
travel as `doc` at detection and widening. That is the ruling's consequence
stated plainly: The detection definition's object, the shipped tree read
against the claim record, is handed whole; the widening definition's object,
which the readings companion's section 5.2 closes with "the shipped tree where
one exists, scoped per 4.5", is handed the same tree; and the entailment
definition's object, which holds no tree, is handed the record.

The list is maintained by hand and held by review: An intent added to the
workstream is added here in the same change, and each entry's `comment` says
so. The eval is silent on that rule. It measures what the entry selects, and
`TestShippedObjectRecordsAllResolve` refuses a listed id that names no record,
so a mistyped id cannot select nothing under cover of the kinds; but nothing
checks that the list covers the workstream, and a workstream intent left off
the list is a gap review has to see. The alternative, deriving the list from a
named root's `builds_on` closure, is not taken: The closure is not a
workstream (itd-177 builds on records outside it), and a derived list is one
nobody reads.

### The default kinds

Each default entry names every include-table kind its position's definition
reads, because the maintainer ruled on 2026-09-02 that a reading of the
object set includes all kinds when the object set fits the reader's window:

- **Widening** names `brief-section`, `glossary-term`, `discipline`, `spec`,
  `doc`, `config`, `source` and `test`: Every kind the readings companion's
  section 5.2 lists for the position, "Brief current text including the
  construal section; `brief/glossary/`; `intents/disciplines/` including the
  selection-criteria discipline; `specs/`; the shipped tree where one exists,
  scoped per 4.5", and every kind the definition's own Object section reads.
  The design framework's section 14 says the same of the Iteration 2 widening
  at Step 2, "drawing on the brief, construal section, glossary and shipped
  tree". The object set narrows the `spec` kind to spc-55 to spc-69 and the
  tree kinds to the seven delivered paths and the three command pages. No
  intent kind stands in the entry, because neither document lists the intents
  in the widening object and itd-194 withdraws the shipped row from the
  position. Every `source`, `test` and `config` item travels whole and marked
  `unscanned`, as at detection.
- **Entailment** names `brief-section`, `glossary-term`, `discipline`, `spec`
  and `intent-projection`. Its object is the claim record read against the
  constraint sources, which holds no tree, so no `doc`, `config`, `source` or
  `test` stands in the entry.
- **Detection** names `brief-section`, `glossary-term`, `discipline`, `spec`,
  `intent-projection`, `doc`, `config`, `source` and `test`. Its object is
  the shipped tree read against the claim record, and the tree is code,
  tests, documentation and configuration alike. The glossary stands in the
  entry because the design framework's section 9 lists it among what the
  input assembler reads for the position, "brief current text including the
  construal section, glossary, shipped intents projected to press release,
  criteria, scope conditions and causal claim", and its section 7.2 classes
  `brief/glossary/` as committed vocabulary read cold; the definition's own
  Object section names it too. Every `source`, `test` and `config` item
  travels whole and marked `unscanned` in the manifest once itd-194 lands,
  because the floor parses markdown alone, and the size report's unscanned
  count says how many such items a run carries.

One bound is stated here in advance, as the readings companion's section 5.6
asks. That section fixes the widening reading's stated bound as "the construal
is one or two sentences and the glossary three to six terms", and its section
5.2 names `brief/glossary/` whole. The committed glossary is twenty-four term
files across four contexts (ten core, three distribution, two interview and
nine ledger), and the widening entry hands it whole, because the glossary row
of the include table admits the directory and no entry narrows a constraint
source. So the widening reading receives twenty-four terms against a stated
bound of three to six, and the run record's `bounds` list, defined below,
states the bound as exceeded, on ruling M5 that a reading which departs from
the one the documents state is "stated as a bound rather than passed off".
The alternative, narrowing the glossary the widening receives to the ledger
context's nine terms, would still exceed six and would hand the position less
than section 5.2 names, so it is not taken. The statement is the whole of what
the bound requires; nothing is refused for it. The glossary figure in the
table below, 13,057 estimated tokens over twenty items, was measured before
the ledger context existed, so the three measurements and the declared
widening and detection windows rounded from them omit its nine terms; every
figure is re-measured at landing and the declaration moves with the
measurement.

A kind that proves useless to a position (tests, say) leaves its entry by a
commit, which is how the record learns; a larger object set in future narrows
kinds the same way. Two leaner entries are one commit away at either tree
position and are costed below. The choice is recorded in each entry's
`comment`.

### The measured figures, and how they were measured

Every figure below is an estimate on the size report's basis. The tree
measured is the one this spec was corrected against, `4195bcbb`, which carries
the eight planned Iteration 2 intents and their specs; the intent's figures at
`8f68ffb3` are the v0.7.0 tree. Measurement was by dry-run assembly in a clean
clone of the design branch's head, with the preset file under test committed
in the clone so the dirty gate passed by construction rather than by
skipping, under the version 1 shape and its named entries, because the binary
in the clone predates the version 2 loader; the clone's own commit is not on
any branch, and every figure is re-measured at landing. Two measurements were
taken there. The first, under the version
1 union, took the constraint-source kinds named above, the `spec` kind whole
(all seventy-seven specs) and the shipped intents by id, with the eight
planned intents beside them at entailment. The second, under the object-set
semantics this spec fixes (kinds admit, the object set narrows), took the
tree kinds over the seven delivered paths and the three command pages. Where
a figure below is for a kind the object set narrows and the union measured
whole, it is an estimate derived from the union figure and is labelled so:
The fifteen workstream specs at about 37,000 estimated tokens, narrowed from
the 191,348 the seventy-seven measured, and the fifteen intent projections at
about 9,000. The implementer re-measures on the commit the change lands on
and records those figures as `measured_*`; the declaration is the figure
measured, rounded up to the next ten thousand, and it moves with the
measurement.

| position | est. tokens | declared | by kind (est. tokens, items) |
|---|---:|---:|---|
| widening | about 615,000 | 620,000 | brief-section 24,627 (12); glossary-term 13,057 (20); discipline 29,312 (14); spec about 37,000 (15, narrowed from 191,348 over 77); doc 14,062 (3); config 0; source 224,093 (49); test 273,212 (86) |
| entailment | about 121,000 | 130,000 | brief-section 24,627 (12); glossary-term 13,057 (20); discipline 29,312 (14); spec about 37,000 (15, narrowed from 191,348 over 77); intent-projection 16,586 (98 items over 23 listed ids, of which the object set names 15) |
| detection | about 624,000 | 630,000 | brief-section 24,627 (12); glossary-term 13,057 (20); discipline 29,312 (14); spec about 37,000 (15); intent-projection about 9,000 (15); doc 14,062 (3); config 0; source 224,093 (49); test 273,212 (86) |

- **Widening**: Every kind the definition reads, over the object set. The
  three constraint-source kinds whole were measured at 66,997 tokens (46
  items, 257,939 bytes); the fifteen specs add about 37,000 by the estimate
  above, the three command pages 14,062, the forty-nine source items 224,093
  and the eighty-six test items 273,212, and the seven delivered paths hold
  no configuration file at this tree. About 615,000 estimated tokens in all,
  declared 620,000. This exceeds the two-hundred-thousand target by a factor
  of three, and this spec says so plainly: The size report states it on
  every assembly at this position, the entry's comment states it, and the
  figure fits a reader with a window of a million tokens. Every source and
  test item arrives marked `unscanned`.
- **Entailment**: The constraint sources whole, with `spec` and
  `intent-projection` narrowed to the object set. The union measurement was
  274,932 tokens, of which the seventy-seven specs were 191,348; replacing
  that contribution by the fifteen-spec estimate gives about 121,000, and the
  projection figure is over twenty-three listed ids where the object set
  names fifteen, so the landing measurement is expected at or below it.
  Declared 130,000, from the estimate. Under the target.
- **Detection**: Every kind the definition reads, over the object set. About
  624,000 estimated tokens in all: The record and documentation kinds, the
  glossary's 13,057 among them, come to about 127,000, the forty-nine source
  items add 224,093 and the eighty-six test items 273,212, and the seven
  delivered paths hold no configuration file at this tree, so `config`
  contributes nothing. Declared 630,000. This
  exceeds the two-hundred-thousand target by a factor of three, and this spec
  says so plainly: The size report states it on every assembly at this
  position, the entry's comment states it, and the figure fits a reader with
  a window of a million tokens. Every source and test item arrives marked
  `unscanned`.
- **Every entry above that names `brief-section`** gains the brief's meta,
  surfaces, internals and delivery chapters beside the product and
  constraints chapters, because itd-194 admits the whole brief bar the
  evidence chapter and the glossary as brief sections and lands before this
  spec. The `brief-section` figures in the table were measured over the two
  chapters and are re-measured at landing, as every figure is, by the window
  eval; the evidence chapter stays excluded, so nothing here names it.
- **Two leaner detection entries are one commit away**, each a recorded
  change to the entry's `kinds` and nothing else: Without `test`, about
  351,000 estimated tokens; without `source` and `test`, about 127,000, which
  is under the target and hands the position the design record, the shipped
  intents and the three command pages. The same two commits are open to the
  widening entry, at about 342,000 and about 118,000. None is the default,
  because the ruling includes every kind while the object set fits the
  reader; any becomes the default by the commit that records why a kind
  proved useless.

The declaration is what the entry measured, rounded up to the next ten
thousand, so the eval speaks as soon as the tree grows past the rounding; that
is the point of it. A declaration is a disclosure of what an entry was
measured at, not a claim about a reader, and each entry's comment says whether
it fits the two-hundred-thousand-token target and, where it does not, the
reader it was measured for. The divisor's disclosed spread (record
and doc material over-stated by 6 to 7 per cent, test material under-stated by
8 per cent) is a property of the estimate and applies to the declaration and
the measurement alike, so it does not move the comparison. Every figure keeps
its estimate label.

### The mechanism proportion at the entailment position

The readings companion bounds the entailment reading by how many intents carry
a mechanism claim and asks that the proportion be reported beside the
findings. The maintainer ruled on 2026-09-02 that the workstream's own
fifteen shipped intents keep their absent mechanism claims and their
`None stated.` conditions as the Iteration 1 baseline, never backfilled, and
that the yield bound is reported by the size report instead, which is what
[iss-2609012259585189](../../../work/issues/resolved/iss-2609012259585189-nothing-reports-the-proportion-of-intents-that-carry-a-mecha.md)
asks and what this spec resolves.

`SizeReport` gains `Mechanism *MechanismReport` (`json:"mechanism,omitempty"`),
set by `Assemble` at the entailment position only and nil elsewhere.
`MechanismReport` is `Intents int` (the projected intent files in the
assembly), `Stated int` (files that contributed a `Mechanism` item whose
trimmed text is anything other than the nullity), `NoneStated int` (files
whose `Mechanism` item's trimmed text is exactly `None stated.`) and
`Absent int` (files that contributed no `Mechanism` item). The count is per
file, derived from the candidates by path and field, so it is checkable
against the manifest, which already records which fields each file
contributed; nothing is added to the manifest's shape, because itd-194 and the
comparative channel each move it and a third mover for a figure the manifest
already implies would be noise. `renderSizeReport` prints one line at the
entailment position, `mechanism: N of M projected intents carry a mechanism
claim; K state none; J carry neither`, and no line elsewhere;
`commands/reading.md` adds `size.mechanism` to the fields the host reports at
that position. Over the default object set the fifteen shipped workstream
intents report zero stated, zero none-stated and fifteen absent.

### The stated bounds on the run record

The intent's two calibration conditions promise that the run record states
the glossary bound as exceeded and states a preset-hash mismatch between the
opening and the closing run. Nothing writes either statement today, so this
spec defines the field. `RunRecord` in `internal/core/reading/ingest.go`
gains `Bounds []string` (`json:"bounds"`), written by `reading ingest`
from the manifest of the run it commits and never from the operator: The
glossary-bound statement, naming the count of `glossary-term` items the
manifest lists against the companion's bound of six, when the entry applied
handed more than six glossary terms; and the preset-hash statement, naming
both hashes, when a prior committed run at the same position exists whose
manifest records a different `preset_hash`. A run with neither carries an
empty list, never an absent key, so a reader can tell a run that stated no
bound from one written before the field existed. The case report repeats the
list beside the run's findings, so the departure is read where the findings
are. The run record carries `SchemaVersion`, and its field set changes, so
that constant moves by one here; `AssemblerVersionCore` does not, because
the bundle and the manifest are untouched.
`TestRunRecordCarriesTheStatedBounds` ingests a fixture run whose manifest
lists seven glossary items after a prior run at the same position under
another hash and reads both statements back, then ingests a run with six
items and no prior run and reads back an empty list.

### The eval, and what keeps it from passing vacuously

`evals/coldreading_window_test.go`, under `//go:build smoke || coldreading`,
joins the lane with no Makefile or workflow edit, exactly as the harness
README says a later eval does. `TestEveryCommittedEntryFitsItsDeclaredWindow`:

1. Resolves the module root (`..`) and its `HEAD` sha, clones it into a
   temporary directory and checks out that sha detached, so the measured tree
   is the checkout's commit with no uncommitted edit in it; the dirty gate and
   the tracked-file check are then satisfied by construction rather than
   skipped. `HOME` is redirected to an empty temporary directory, as every eval
   in the lane redirects it.
2. Parses the clone's `.abcd/config/reading-presets.json` with the eval's own
   minimal struct (version, positions, `window.tokens_est`). The oracle rule
   holds: `bannedImports` already refuses `internal/core/reading`, so the
   eval's account of the declaration is independent of the assembler's.
3. For each of the three assembling positions, named in the eval as
   `widening`, `entailment` and `detection`, runs
   `abcd reading assemble --position <p> --target HEAD --dry-run --json`
   through `runIn`, requires the binary's `size.window.tokens_est` to equal
   the eval's own parse, and collects a breach when `size.tokens_est` exceeds
   it. The comparative position is exempt from this eval by name: Its object
   is bounded by the widening run it is handed, not by the tree, so a window
   over the tree measures nothing about it, and the comparative channel's own
   eval covers it.
4. Fails naming, for each breach, the position, the measured figure (tokens
   and bytes) and the declaration.

Non-vacuity: The test requires the number of positions measured to equal the
number of entries the file declares at the three named positions, and refuses
zero. The negative control, `TestTheWindowCheckReportsABreach`, lowers one
declaration in the clone to one below the figure just measured, commits it
there under the fixture identity so the dirty gate admits it, re-runs, and
requires exactly one breach naming all three facts; `windowBreaches` is the
one function both tests call.

The enforcement this delivers is what the intent now claims: Drift past a
window is caught by the cold-reading eval lane, which is not yet a required
check. The eval runs on every committed change in the eval lane; it does not
run in `make preflight`, and a pull request can merge without it until the two
issues named under Scope are closed. That is the gate work this depends on for
the claim to become "caught by the build".

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

This spec lands third in Phase A. The two-operand spec lands before it
because `PresetFor(position)` and the manifest's `preset` block are what this
spec's entries are applied through; itd-194 lands before it because its
widening row, its brief-chapter rows and its manifest mark are what the
window eval measures against. It lands before the comparative channel because
it owns preset schema version 2 and the window eval, which the comparative
entry conforms to, and before the reading-occasioned origin, which lands in
Phase A before the first promotion of an accepted reading item, since the
origin key is written only by commands. It moves `SchemaVersion` by one, for
the `bounds` field the run record gains, and not `AssemblerVersionCore`: The
preset file and the size report are outside both artefacts, and the bundle
and the manifest do not change.

### The target is stated, not enforced

Two hundred thousand estimated tokens is the target an entry aims at; the
maintainer ruled on 2026-09-02 that any figure inside the reader's window is
acceptable and that a figure over the target is stated to the operator. The
size report therefore carries one over-target line and no refusal, and the
declared window stays what the entry measures.

## How the Acceptance Criteria are satisfied

- **ac-1 (version 2 without a window refuses).** The version-2 rule in
  `LoadPresets`; `TestPresetV2RefusesAPositionWithoutAWindow` asserts the
  message names the position.
- **ac-2 (version 1 loads, report says none).** `presetSchemaVersions` admits
  1; a one-preset file is read into the entry set with no window; the report
  renders "none declared". `TestPresetV1LoadsAndReportsNoWindow`;
  `TestPresetV1WithTwoPresetsRefusesNamingThem`.
- **ac-3 (committed entries pass).** `TestEveryCommittedEntryFitsItsDeclaredWindow`
  over the committed file, each of the three assembling positions at or below
  the declaration its entry carries.
- **ac-4 (an undersized declaration fails, naming the facts).**
  `TestTheWindowCheckReportsABreach`.
- **ac-5 (an entry whose object set names one shipped intent carries it
  alone).** The record narrowing and `pathNamesRecord`;
  `TestAnObjectSetNamingOneRecordCarriesThatRecordAlone`, a unit test in
  `internal/core/reading` over a fixture holding two shipped intents under an
  entry whose object set names one, asserts every projected item names the
  listed intent, the projected fields present are exactly that intent's, and
  the manifest lists exactly those items.
- **ac-6 (a populated object set selects only listed records).**
  `TestAnObjectSetSelectsOnlyListedRecords` over a version-2 fixture entry
  with `records: [itd-A]` and `kinds: [intent-projection, glossary-term]` at
  entailment: Every item whose path names a record names `itd-A`, and the
  glossary items are present whole. `TestShippedObjectRecordsAllResolve`
  holds the committed list to ids that exist, and
  `TestAPathSelectsOnlyTheEntrysKinds` holds the kinds-admit rule over a
  fixture package that carries a `.go` and a `.md` file under an entry naming
  the path and `doc` alone.
- **ac-7 (each declaration carries its figure and commit).**
  `TestShippedEntriesDeclareMeasuredFigures` loads the committed file and
  requires every window to carry `measured_tokens_est`, `measured_bytes` and a
  well-formed `measured_at`.
- **ac-8 (over-target line).** `renderSizeReport` compares the total estimate
  with `TargetTokens = 200000`, a constant beside the divisor in
  `assemble.go`, and appends one line, `over target: <estimate> estimated
  tokens against a target of 200,000; the reader's window decides whether
  this is acceptable`, when the total exceeds it; under the target the line is
  absent. The target is a statement to the operator, never a refusal, and the
  JSON result carries `over_target: true` beside the figures. Over the
  committed detection and widening entries the line is present on every
  assembly, since they measure about 624,000 and about 615,000; over the
  entailment entry it is absent.
  `TestSizeReportNamesAnOverTargetTotal` renders a result above
  and one below the constant and asserts the line's presence and absence.
- **ac-9 (the entailment mechanism proportion).** `Assemble` fills
  `Size.Mechanism` at the entailment position from the candidates by path and
  field, and `renderSizeReport` prints the one line.
  `TestEntailmentSizeReportStatesTheMechanismProportion` assembles a fixture
  holding three shipped intents, one with a stated mechanism, one with
  `None stated.` and one with no section, and asserts the three counts and the
  total; `TestMechanismProportionIsAbsentAtOtherPositions` asserts the field is
  nil at widening and detection; `TestRenderSizeReportStatesTheMechanismLine`
  in `internal/surface/cli` asserts the line's presence at entailment and its
  absence elsewhere.

## Tests

Watched fail before, pass after. `internal/core/reading`:
`TestPresetV2RefusesAPositionWithoutAWindow`, `TestPresetV1LoadsAndReportsNoWindow`,
`TestPresetV1WithTwoPresetsRefusesNamingThem`, `TestPresetV1RefusesAWindow`,
`TestSizeReportCarriesTheDeclarationAndTheVerdict` (mutation: A declaration one
below the measured figure flips `ExceedsWindow`),
`TestAnObjectSetNamingOneRecordCarriesThatRecordAlone`,
`TestAnObjectSetSelectsOnlyListedRecords`, `TestAPathSelectsOnlyTheEntrysKinds`,
`TestAConstraintSourceKindIsHandedWhole`, `TestShippedObjectRecordsAllResolve`,
`TestShippedEntriesDeclareMeasuredFigures`,
`TestEntailmentSizeReportStatesTheMechanismProportion`,
`TestMechanismProportionIsAbsentAtOtherPositions`,
`TestOnlyTheTreePositionsNameSourceOrTest` (the committed detection and
widening entries name both tree kinds, every `source` and `test` item either
hands carries the `unscanned` mark, and the entailment entry names neither,
held against the ruling),
`TestRunRecordCarriesTheStatedBounds`,
`TestDefaultItemSetsMatchTheRecordedDigests` (a digest of each position's
default item set, recorded before the change lands and moved only in the
same diff as a change to that position's entry, so the item set a position
is handed moves by a commit and by nothing else, and the three digests
differ, which is what resolves iss-2608311501240566),
`TestAnEntrysItemSetHoldsTheExclusionFloor` (an assembly under each
committed entry over the read-block fixture, with every warm class the eval
plants present in the fixture, and each class absent from the bundle),
`TestTwoAssembliesOfOneEntryAreByteIdentical` (two assemblies of one entry
at one commit compared bundle for bundle and manifest for manifest);
the existing `TestThreePositionsCarryDistinctItemSets` and the three
committed-file tests of spc-69 stay green over the new file.
`internal/surface/cli`: `TestRenderSizeReportSaysWhetherAWindowIsDeclared`,
`TestRenderSizeReportStatesTheMechanismLine`.
`evals`: The two tests above. Both eval lanes are run explicitly and once under
`TMPDIR=/tmp`; the lane runs on every committed change and is not in
`make preflight`.

## Out of scope

- A budget enforced at invocation, and any refusal for size.
- A tokenizer; the basis stays bytes divided by 3.85 and says so.
- The comparative position, whose object arrives by its own channel and is
  bounded by the widening run; its window eval is the comparative channel's,
  and its entry is that spec's, in the shape fixed here.
- Removing a kind from a default entry. The detection and widening entries
  name every kind by ruling; a kind that proves useless leaves by a commit
  that records why,
  and the `unscanned` mark a source or test item carries is itd-194's.
- Backfilling a mechanism claim on any shipped intent; the proportion is
  reported as it stands.
- Checking that the object set covers the workstream; the eval is silent on it
  and review holds it.
- The eval lane becoming a required status check, with the two issues named
  above.
- Selecting a record bucket from an entry. `.abcd` is a denied segment, so a
  path clause cannot name `specs/closed`; a bucket selector would be new
  grammar, which spc-69 closed and the two-operand spec keeps closed.
