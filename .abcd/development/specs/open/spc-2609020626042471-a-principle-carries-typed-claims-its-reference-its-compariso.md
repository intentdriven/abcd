---
id: spc-2609020626042471
slug: a-principle-carries-typed-claims-its-reference-its-compariso
intent: itd-2609020625405170
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The knowledge record becomes a read object: A principle carries typed claims, projects its statement, and inherits only what held

## Summary

spc-2609020626042471 delivers
[itd-2609020625405170](../../intents/planned/itd-2609020625405170-a-principle-carries-typed-claims-its-reference-its-compariso.md).
An entry under `.abcd/development/principles/` may declare four typed keys in a
frontmatter block: `claim_type`, `reference`, `comparison` and `evidence`. The
record lint reports an entry carrying none of them as untyped at warn, refuses
an entry carrying some of them at blocker, and reports a typed entry whose
evidence names a scope condition that delivery falsified, narrowed or never
tested. The cold-reading assembler gains a row that projects a principle's
statement and nothing else, admitted at the three assembling positions, so the
statement is knowledge a reading may test and the citations stay behind as
genealogy; which of those positions receives the projection is a preset choice
this spec does not make. `disembark principles` carries the four keys in the
payload it writes, and the payload's schema version moves.

The decision the intent flagged for the maintainer is adopted: The family
becomes a declared record store, a record-architecture ruling on the pattern
of [adr-30](../../decisions/adrs/0030-record-information-architecture.md),
recorded as
[adr-2609021016270132](../../decisions/adrs/2609021016270132-the-principles-family-is-a-declared-record-store-whose-entri.md)
with `status: accepted`, adopted by the maintainer on 2026-09-02 at the
planning interview after being checked against the design framework and the
readings companion;
[iss-2609020626111484](../../../work/issues/resolved/iss-2609020626111484-adr-owed-the-principles-family-becomes-a-declared-record-sto.md)
is resolved by it. The ADR fixes what this spec builds: Four typed keys,
forward-only population, a principle read cold by statement and never by
citation, and the inheritance check. Population is forward-only throughout:
No existing entry is renamed, typed or moved.

This spec lands last of the Iteration 2 set. It consumes the condition reader
the sibling spec
[spc-2609020626046252](spc-2609020626046252-a-scope-condition-is-dispositioned-from-a-reading-run-keyed.md)
introduces, and it takes the include table, the exclusion floor and the eval
tables as the earlier specs leave them.

## Scope

In: The `prn` record store and its slug-keyed identity; the four keys, their
grammar, the explicit-null form and the `mechanism` alias;
the two shape rules and their severities; the include-table row at the three
assembling positions, the labelled-paragraph projection and the exclusions it
asserts; the inheritance rules over condition dispositions; the lifeboat
principles payload at schema version 2, the relation of its `prn-` ids and
`Evidence` to the record's, and the distiller contract that moves with it; the
read-block eval case, its coverage rows and the pinned counts it moves; the
charter render, the assembler version and the manifest version.

Out: The wording of any principle; automatic distillation into the record;
the frame-level revision record; a fourth claim type; backfilling an existing
entry; any edit to `.abcd/config/reading-presets.json`; the comparative
position, whose object arrives by its own channel.

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

This spec lands last, in Phase B, after the condition disposition, whose
`internal/core/condition` package it consumes as it finds it.

For this spec that rule reads: `Kinds()` gains `principle`, and the manifest's
kind vocabulary is closed (`DecodeManifest` refuses an unknown kind), so the
bundle and manifest `SchemaVersion` moves by one; `Table`, `Exclusions`,
`Kinds()` and a projection resolution change, so `AssemblerVersionCore` takes
a MINOR bump. The lifeboat's `PrinciplesSchemaVersion` moves 1 to 2, which no
sibling touches.

### The store, under adr-2609021016270132

`.abcd/record-lint.json` gains `"prn": ".abcd/development/principles"` in
`record_schema.record_stores`, and `recordStores` in
`internal/core/lint/schema.go` gains a `prn` entry: Noun `principle`, node type
`principle`, no buckets (flat, as the ADR store is), `README.md` skipped as in
every store. The store is **slug-keyed**, which is new: `recordStore` gains
`slugKeyed bool`, `schemaRecord` gains `slug string`, and for such a store
`handle()` renders `prn-<filename stem>` while `num` stays zero.
`checkRecordFilename` compares an `id` the file carries against `prn-<stem>`
and reports nothing when the file carries no frontmatter, because
`requiredFields` is nil here: An untyped entry is a prose file keyed by its
filename, which is what every entry is today. `checkRecordFilenameSlug` is
silent for the same reason, since no principle carries a `slug` field.

The uniqueness leg keys on the rendered handle. `checkRecordSchema` indexes
the corpus by `recordRef{prefix, num}`, which would key every principle as
`prn-0` and report thirty collisions on a clean store. The index becomes a map
keyed by `handle()`, which for a numbered store is the `prefix-N` string it
was and for the slug-keyed store is `prn-<stem>`; the high-water mark skips a
slug-keyed store, which issues no ordinals. Two files rendering one handle is
the collision the leg exists to find, and on a filename-keyed store the file
system already refuses it, so the leg is a guard against a future second
store and reports nothing today.

`LoadRecordGraph` emits a node of type `principle` so the assembler's
`rowPaths` can route the row through the graph. `recordid.CitedIDRe` and
`familyRoots` do not change: A `prn-` handle is not a citation any other
record makes, and the resolver's comment that the principles carry no
per-entry id is amended to say they carry a slug-keyed one no citation grammar
admits. `record_provenance` is forward-only and treats an unstamped record as
a state, so it needs no change either.

### The four keys and their grammar

A typed entry opens with a frontmatter block, every key on its own line, read
by `frontmatter.Fields`:

```yaml
---
id: prn-fix-the-detector
claim_type: causal
reference: "abcd lint"
comparison: "Hand-fixing instances against arming a detector over the class."
evidence: [itd-181, spc-59, cond-2608311949582375]
---
```

- `id`: `prn-` followed by the filename stem; required on a typed entry, and
  the one key `record_schema` judges.
- `claim_type`: Exactly one of `criterion`, `causal`, `context`, the design
  documents' words for the three claim kinds the gradient
  [itd-190](../../intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s-three-claim-kinds-c.md)
  records; `mechanism` is read as an alias for `causal`, below.
- `reference`: A record handle matching `recordid.CitedIDRe`, or a
  double-quoted string naming a surface (a verb, a package, a rule id).
- `comparison`: A double-quoted string, one sentence, non-empty.
- `evidence`: A flow sequence whose members are record handles (`adr-N`,
  `itd-N`, `spc-N`, `iss-N`, `rdi-N`) or scope-condition identities matching
  `condition.MarkerIDRe` (`cond-` and sixteen digits). A duplicate member is a
  finding.

**Two claim vocabularies meet here, and the documents' word governs.** The
design framework and the readings companion name the three claim kinds
`criterion`, `causal` and `context`, and the entailment contract's
`claim_type` on a reading item, `ClosedVocabularies` in
`internal/core/reading/ingest_regime.go`, already uses those words. The
principle key uses them too, so an entailment item that surfaced a causal
claim types as a `causal` principle when it is distilled, with no mapping.
The shipped intent vocabulary in `internal/core/intent/claims.go` spells the
causal kind `mechanism`
([itd-177](../../intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md),
[itd-190](../../intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s-three-claim-kinds-c.md)),
and the `## Mechanism` heading on an intent stays, since the framework names
that section as the causal claim's home; the token itself is separate work
the implementer aligns to `causal` under the maintainer's rule that the
documents' word governs, and this spec does not move it. Until it moves,
`mechanism` is accepted on read as an alias: `principle_claims` reads
`claim_type: mechanism` as `causal` and emits no finding, and every writer of
the key, the lifeboat's `disembark principles` included, writes `causal`.
Nothing refuses `mechanism` with a rename, and nothing refuses `causal`.

**The explicit null is the literal `null`, alone.** `claim_type: null` records
a key considered and declined. The lint reads a key's byte state as `claims.go`
reads a section: Absent (not in the block), nullity (`null`), empty
(`claim_type:` with nothing after it, `evidence: []`, or the other spellings
`frontmatter.IsNull` folds, `~` and the case variants), and stated. Absent and
nullity are never collapsed, and empty is a fault: A blank value is the byte
shape of a key someone forgot. One spelling of null, on the `NullityToken`
precedent, is what lets a gate tell a declined claim from a mistyped one.

### The two shape rules and the warn/blocker split

Two rule ids, because a rule in `.abcd/record-lint.json` carries one
`severity` and no rule in `internal/core/lint` emits any other. Both live in a
new `internal/core/lint/principles.go`, dispatched from `lint.go` beside
`record_schema`, and both take `record_stores` from the `record_schema`
configuration as `record_provenance` does.

- `principle_untyped`, declared `{"enabled": true, "severity": "warn"}`: One
  finding per entry carrying none of the four keys, message `principle
  <handle> is untyped (carries none of claim_type, reference, comparison,
  evidence)`. A count, not a fault, expected to stay non-zero. A warn-level
  rule is expressed exactly as `no_brittle_line_refs` is, by the `severity`
  string; preflight reports it and does not fail on it.
- `principle_claims`, declared at `blocker`: Findings on a typed entry, one
  per defect, each naming file, line and key. The defects: A key present while
  another of the four is absent (the message names the missing key); a missing
  or wrong `id`; an empty value; a value outside its grammar above, with
  `mechanism` read as `causal` and not a defect; a duplicated `evidence` member; and a statement carrying a
  citation, that is a record handle or a markdown link inside the `**The
  rule.**` paragraph, because the projection below promises a statement free
  of genealogy and a promise the assembler cannot keep is one the lint refuses
  first. Only a typed entry is judged; an untyped entry produces no finding
  until its author types it.

`principle_claims` enters at blocker because the corpus is clean by
construction: No typed entry exists, so no pre-existing finding lands as a wall.

### The projection row: Which paragraph is the statement

A principle has one heading, its H1, and a body of labelled paragraphs:
`**The rule.**`, `**Why.**`, `**Bounds.**`, `**Promotion.**` and whatever else
the author added. The statement is the H1 title and the `**The rule.**`
paragraph, and nothing after it. `projectField` in
`internal/core/reading/project.go` resolves a field as a heading section and
otherwise as a frontmatter key; it gains a third resolution between the two, a
**labelled paragraph**: A field spelled `The rule` resolves as the first
paragraph outside a fence whose first line opens with `**The rule.**`, taken to
the next blank line, the label removed and the document's H1 title placed
above it, because a rule without its name is not readable cold. Inline links
are unwrapped to their label on the `renderedTexts` precedent: A link target is
a citation and the label is prose. An entry with no such paragraph contributes
no item.

`Table` gains one row, admitted at the three assembling positions:

| Positions | Source | Matches | Fields | Store | Kind | Admitting rule |
| --- | --- | --- | --- | --- | --- | --- |
| `widening`, `entailment`, `detection` | `.abcd/development/principles` | `.md` | `The rule` | `prn` | `principle` | The knowledge record is a read object: A principle travels as its statement, and its keys and citations are genealogy |

The row's `Positions` exclude the comparative position. At comparative the
include table is the whole account and admits the candidate channel and the
criteria discipline only, on the comparative spec's ruling, so the knowledge
record is not among the sources that position may see, and the oracle binds
`.abcd/development/principles` as an excluded family there.

`Kinds()` gains `KindPrinciple = "principle"`, so `principle` is a kind a
committed preset entry may name. **This spec makes no edit to
`.abcd/config/reading-presets.json`.** The table admits the projection at
every position but the comparative; admitting `principle` at a position is a
commit to that position's entry, and which of the three assembling positions
receives the knowledge record is a ruling this spec does not make. The
presets spec's eval measures every committed entry against the window it
declares, so an entry that adds `principle` is made under that eval and
re-measured by it. Until such an entry exists, `principle` reaches no reading:
The invocation carries nothing that could widen an entry, and the manifest
records the entry applied.

`Exclusions` gains four `frontmatter key` entries, `claim_type`, `reference`,
`comparison` and `evidence`, under the rule `field projection`, and one entry
`{Rule: "the statement is knowledge and the citations are genealogy", Signal:
"citation", Detail: "record handles and links in a principle"}`.
`redactExcluded` strips the four keys from every admitted principle before
projection, and a new `verifyPrincipleItem` refuses the assembly if a
principle item carries a record handle after projection, so the manifest's
assertion is checked rather than trusted. The charter render moves and is
regenerated in the same change, which
`TestReadingsCharterCarriesTheRenderedIncludeTable` requires; the version
classes are stated under Landing order.

### The inheritance check

`principle_inheritance`, declared at `warn`, and `principle_falsified`,
declared at `blocker`, both in `principles.go`, both over the `evidence` of
typed entries. Resolution of one member:

1. A record handle is resolved against the record graph. One in the corpus
   is silent; one not in the corpus is reported by `principle_inheritance` as
   `unresolvable`, never as absent, which is the reading a principle distilled
   from a lifeboat needs.
2. A `cond-` identity is resolved by scanning every intent under
   `intents/shipped/` for its marker, `condition.MarkerRe`, the regex
   `claims.go` reads. An identity no shipped intent carries is `unresolvable`;
   one carried by more than one is reported as ambiguous.
3. For a carried identity, the standing disposition is read with
   `condition.Standing(content)` from the intent's `## Audit Notes`.
   `falsified` is a `principle_falsified` finding naming the principle and
   the identity; `narrowed` is a `principle_inheritance` finding carrying the
   narrowing verbatim; no disposition, or `untested`, is a
   `principle_inheritance` finding saying the principle rests on an untested
   condition.

The `abcd-condition` block grammar and the block reader live in
`internal/core/condition`, which spc-2609020626046252 introduces together
with the verdict-block grammar and the `Standing` fold, and spc-2609020626046252
lands before spc-2609020626042471; this spec extends nothing there and
consumes the package as it finds it. `Standing` folds by that spec's rule,
under which a later verdict block does not override a reading-occasioned
block unless its rationale names the block's occasion, so what this lint
reports is the standing the condition verb's record shows.

`principle_falsified` is the one blocker because it is the one case the
intent exists to prevent, and the remedy is local: The author restates the
evidence or the principle. The lint does not import `internal/core/intent`:
That package's own tests import `internal/core/lint`, so the reader has to be
a leaf, which is why it lives in `internal/core/condition`.

### The lifeboat payload

`lifeboat.Principle` in `internal/core/lifeboat/synthesis_types.go` gains
`ClaimType`, `Reference` and `Comparison` as `*string` fields serialised as
`claim_type`, `reference` and `comparison`, beside the existing `Evidence`.
`PrinciplesSchemaVersion` moves 1 to 2, so `synthSchemaGate` refuses a payload
at 1 with `unsupported principles schema_version 1` and a consumer of
`principles.json` sees the change by the version. `validateDelegatedPrinciples`
decodes each entry's raw object before the typed decode so absent and `null`
are told apart: Every entry carries all four keys, `null` allowed on the three
new ones, `claim_type` one of the three or `null`, a delegated `mechanism`
written back as `causal`; an entry failing that is a
`PrincipleDrop` with its reason, never fatal. Deterministic mode writes
`reference` as the ADR handle it distilled from and the other two as `null`:
It carries what it can establish and invents nothing.
`renderPrinciplesMarkdown` renders the three beside the evidence.

**Two `prn-` derivations, one grammar.** The lifeboat already mints `prn-`
ids: Deterministic mode derives `prn-<adr-id>` from the ADR it distils, in
`synthesis_principles.go`, and delegated mode takes the distiller's `id`, both
bounded by `prnIDRe`, which is `prn-` followed by kebab segments. The record's
`prn-<stem>` handle fits the same grammar, and that is the whole of the
relation: A lifeboat id names a distilled entry inside one packed payload, a
record handle names a file in this repository's store, and neither resolves
in the other's namespace. A person who writes a lifeboat principle into the
record chooses a filename and the handle follows it; the lifeboat's id is not
carried into the record, and a record handle is never a lifeboat evidence
reference, because the lifeboat's `Evidence` grammar admits no `prn-`.

**The payload's `evidence` is the lifeboat's `Evidence`.** The fourth key is
the field the type already carries, under its existing grammar: Every member
resolves to a packed record id, a live graveyard finding id or a packed
lifeboat path, cite-or-be-dropped through `buildPrincipleEvidenceSet` and
`filterSynthEvidence`. The record grammar maps onto it as packed ids only,
per the intent's second scope condition: A record handle (`adr-N`, `itd-N`,
`spc-N`, `iss-N`) is the same token in both; `rdi-N` and a `cond-` identity
have no packed form and are written to `Evidence` by neither mode, because
the lifeboat packs no readings store and no condition, and a principle
distilled from a lifeboat inherits a condition only when its author states
one on the record. In the other direction, a record transcribed from a
payload carries the packed ids, which resolve only in the source repository
and which `principle_inheritance` reports as `unresolvable`, never as absent.

`agents/principle-distiller.md` moves to `prompt_version` 0.2.0 with the keys
in its field rules and its example, and `agents/CHANGELOG.md` records the
bump; the `agent_contract` rule holds the two in lockstep. The changelog
entry lands under the one dated heading the Iteration 2 cut shares, beside
the scribe's entry from spc-2609020626045177, as one heading with one agent
sub-heading each, on the file's existing form.

### The read-block eval case, its coverage rows and the pinned counts

`evals/coldreading_fixture_test.go` gains a sentinel class `PRINCIPLE-CITATION`
planted in a new fixture entry
`evals/testdata/cold-reading/baseline/.abcd/development/principles/a-typed-principle.md`
three times: In `evidence`, in the `**Why.**` paragraph as a record handle, and
as a link target in the `**The rule.**` paragraph. The class is warm at every
position, so `TestReadBlockBaselineIsClean` fails if the token reaches any
bundle and `TestEverySentinelIsPlanted` fails if a plant is lost. The oracle in
`evals/coldreading_oracle_test.go` gains the four keys in `excludedKeys`; its
import guard keeps it independent of the include table.

The third home is stated for what it is. A link target in the statement
cannot leak while the unwrapper stands, since the projection drops the target
and keeps the label, so against the assembler as delivered that plant proves
nothing about the citation rule; it falsifies the unwrapping alone. What ac-7
exercises is the holed-firewall relocation: `TestReadBlockCatchesAHoledFirewall`
gains the case that relocates the class into the `**The rule.**` paragraph as
a bare token, which the projection carries into the bundle, and the control
must report the class. The refusal of a real record handle after projection
is `verifyPrincipleItem`'s, tested in the reading package.

On the reframe spec's pattern, the rows this spec adds and the pinned counts
it moves are listed here, each as the delta this spec makes; the merging
change sets each literal from the merged base, since the comparative,
reframe and scribe specs move some of the same tables first.

- `sentinelClasses`: +1, `PRINCIPLE-CITATION`.
- `carriers`: +1, the fixture principle's `**The rule.**` text as a cold
  marker at the three assembling positions, so the positive row below is
  falsifiable.
- `excludedKeys`: +4, the four keys, each citing this spec.
- `excludedFamilies`: +1, `.abcd/development/principles` bound at the
  comparative position only, citing this spec; its exercise rides on the
  comparative spec's fixture run, which lands first.
- `admittedRecordPaths`: +1, `.abcd/development/principles` at the three
  assembling positions.
- `coverage`: +5 rows, `declaredGaps` unchanged. (1) Principles are admitted
  as their statement at the three assembling positions; falsifier, delete the
  principle row from `Table`; caught by the carrier floor. (2) The four
  principle keys never travel; falsifier, delete the four key rows from
  `Exclusions`; caught as a leak of `PRINCIPLE-CITATION`. (3) A principle's
  citations never travel; falsifier, project the whole file on the principle
  row; caught as a leak of `PRINCIPLE-CITATION`. (4) A link in the statement
  travels as its label; falsifier, stop unwrapping links in the
  labelled-paragraph resolution; caught as a leak of `PRINCIPLE-CITATION`.
  (5) The principle row is not admitted at comparative; falsifier, add the
  comparative position to the row; caught by family absence.

`TestEveryAssemblerRuleHasAFalsifier` and the declared table sizes move
together in the one change.

## How the Acceptance Criteria are satisfied

- ac-1: A typed entry missing one key is a `principle_claims` blocker naming
  the file and the missing key. `TestPrincipleClaimsNamesTheMissingKey`.
- ac-2: An entry with no keys is one `principle_untyped` warn finding and no
  other finding from any principle rule. `TestUntypedPrincipleIsAWarnAndNothingElse`.
- ac-3: Evidence naming a condition whose standing disposition is `falsified`
  is a `principle_falsified` blocker naming the principle and the identity.
  `TestFalsifiedConditionIsReported`.
- ac-4: Evidence naming a `narrowed` condition is a `principle_inheritance`
  finding carrying the narrowing verbatim. `TestNarrowedConditionCarriesTheNarrowing`.
- ac-5: Evidence naming a condition with no disposition in its intent's Audit
  Notes is a `principle_inheritance` finding saying untested.
  `TestUndispositionedConditionIsUntested`.
- ac-6: An assembly at a position whose preset admits `principle` lists each
  principle in the manifest as a projected item with field `The rule`, and the
  manifest's exclusions carry the four keys and the citation entry. The test
  assembles under a fixture preset naming `principle`, since the committed
  presets carry no such entry until one is made under the presets spec's
  eval, and again by the kind token.
  `TestPrincipleProjectsItsStatementOnly`, `TestManifestAssertsPrincipleExclusions`.
- ac-7: The eval class above. The link-target plant falsifies the unwrapping
  only; the criterion is met by the holed-firewall relocation of the class
  into the rule paragraph as a bare token, which the control must report.
  `TestReadBlockCatchesAHoledFirewall` gains the case.
- ac-8: Every principle `disembark principles` writes carries the four keys,
  in both modes, with `evidence` under the lifeboat's own grammar.
  `TestDeterministicPrinciplesCarryTheFourKeys`,
  `TestDelegatedPrincipleWithoutAKeyIsDropped`.

## Tests

Each watched fail before the change and pass after.

- `internal/core/lint`: The five named above, `TestPrincipleStoreIsSlugKeyed`,
  `TestSlugKeyedStoreCollidesOnTheRenderedHandle`,
  `TestUntypedPrincipleHasNoSchemaFinding`, `TestPrincipleClaimsRefusesEmptyValue`,
  `TestPrincipleClaimsRefusesFourthClaimType`,
  `TestPrincipleClaimsReadsMechanismAsCausal`, `TestPrincipleStatementMayNotCite`,
  `TestUnresolvableEvidenceIsReportedNotAbsent`, `TestWarnRuleDoesNotFailPreflight`,
  and `agentcontract_test.go` holding the distiller's documented payload in
  lockstep with `lifeboat.Principle`.
- `internal/core/reading`: `TestLabelledParagraphResolves`,
  `TestLabelledParagraphCarriesTheTitle`, `TestLinksUnwrapInTheStatement`,
  `TestPrincipleItemCarryingAHandleRefuses`, `TestPrincipleIsAScopeToken`,
  `TestPrincipleRowExcludesComparative`, `TestManifestAtTheOldSchemaVersionIsRefused`,
  and the charter-render and assembler-version pins updated.
- `internal/core/lifeboat`: The two named above, `TestSchemaVersionOneIsRefused`,
  `TestNullKeyIsCarriedNotDropped`, `TestPrinciplesMarkdownRendersTheKeys`,
  `TestEvidenceCarriesPackedIDsOnly`, `TestDelegatedMechanismIsWrittenAsCausal`.
- `evals`: The `PRINCIPLE-CITATION` class in both eval lanes with the coverage
  rows and count pins above, run explicitly under `make evals-cold-reading`.

## Out of scope

- Changing what any principle says, or moving a principle to the disciplines
  bucket; the promotion ladder in the
  [principles README](../../principles/README.md) is untouched.
- Distilling into `.abcd/development/principles/` from a lifeboat: The verb
  writes `principles.json`, and the record is written by its author.
- A fourth claim type, a numeric identity for the family, or a citation
  grammar that admits `prn-` handles from other records.
- Any edit to `.abcd/config/reading-presets.json`, and the ruling on which
  position reads the knowledge record; both belong to the presets and their
  eval.
- The comparative position, whose object arrives by the comparative channel.
- The frame-level revision record.
