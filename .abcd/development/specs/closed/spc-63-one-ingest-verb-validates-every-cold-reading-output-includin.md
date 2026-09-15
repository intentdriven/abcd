---
id: spc-63
slug: one-ingest-verb-validates-every-cold-reading-output-includin
intent: itd-185
---

# The cold-reading output contract: a strict per-position schema, a supply-regime gate, and a staged ingest that leaves evidence or nothing

## Bundle

itd-185,
[itd-183](../../intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md)
and
[itd-184](../../intents/shipped/itd-184-four-cold-reading-definitions-one-blindness-core-each-positi.md)
are one design under one bundle kind, and the ceremony cannot give them one
spec: a spec's `intent:` is a single id, captured as iss-2608300108376943.

| Spec | Component it owns |
| --- | --- |
| [spc-61](../closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) | The input assembler, the include table, the pathless bundle, the manifest, and the bundle's shared decisions |
| [spc-62](../closed/spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md) | The four reading definitions under `agents/` and the blindness-core byte-identity test |
| spc-63 (this record) | The output contract, the supply-regime gate, and the ingest sub-verb |

The package name, the verb tree, the run-identifier form and the artefact
layout are the bundle's shared decisions, stated once in
[spc-61 § The package, the verb tree and the artefact layout](../closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md#the-package-the-verb-tree-and-the-artefact-layout-shared-bundle-decisions).
This spec uses them and does not restate them.

## Summary

spc-63 delivers itd-185's contract: a reading that quietly exceeds its licence
is refused at ingest, and the refusal names the offending field or item. The
verb is `abcd reading ingest --output-json <path>` in the same
`internal/core/reading` package as the assembler, built on the
output-contract idiom the repository already carries (agent emits JSON, a
deterministic verb validates it, the verb writes the record):
`intent audit ingest --verdict-json`
([`internal/core/intent/audit.go`](../../../../internal/core/intent/audit.go),
whose `validateVerdict` already runs `DisallowUnknownFields`) and
`launch ship --changelog-json`.

Three properties distinguish this contract from a structural schema check.
Ids are **minted by the verb**, never self-supplied. The **supply regime** is
resolved from the definition through the run's position, so no operator input
can set it. And **provenance is enforced for every regime**: an item with an
empty pattern field is refused, which the definitions instruct and nothing
else checks.

The policy is settled by rulings (4), (5), (8), (12) and (18) of 2026-08-28
and is not this spec's to reopen. This spec settles the payload schema, the
reserved-name tables, the signature registry, the staging protocol and the
refusal-record shape.

## Scope

**In.** The output payload schema and its per-position bodies; the regime gate
and its signature registry; id minting; the staged write protocol and its
orphan sweep; the manifest-reference check; refusal records; the
`abcd reading ingest` sub-verb on both planes.

**Out.** The definitions that state each regime (spc-62); the assembler and
the manifest (spc-61); the reading-record and disposition record *schemas*,
which belong to
[itd-180](../../intents/shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md)
and spc-58 (this verb validates against them and writes them, it does not
define them); admission, which is
[itd-189](../../intents/shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md)'s.

## Approach

### The payload

One JSON document per run, read behind `fsutil.ReadGuarded` with a byte cap,
decoded with `DisallowUnknownFields` at every level, so every violation names
a field rather than guessing at one:

```
_type          = "abcd.reading.output/1"
run_id         = rdg-<...>          (must match a parked assemble run)
position       = widening | entailment | comparative | detection
regime         = generative | explicative | evaluative | registrative
manifest_sha256
instrument     = {model, definition_sha256, assembler_version}
items[]        = flat objects: pattern, plus the position's body fields
```

**Ids are the verb's to mint.** The payload carries no item identifier; a
reading holds no mint. On acceptance each item is minted
`rdi-<yymmddHHMMSS><rrrr>` through `recordid.Minter.Mint("rdi")`
([adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md)),
run-scoped and never content-derived, so a re-raise of the same finding in a
later run is distinguishable from its first appearance. The mint itself is
`capture.IngestReading`'s, under the ledger lock, because that is where the
collision probe can see the tree it is about to write into.

**Provenance is one envelope field.** itd-180 calls it the pattern named and
itd-185 calls it the pattern-basis field; they are one requirement (ruling
(18)), and its wire name is `pattern` — the reading record's own envelope field
name, and the key the four definitions instruct. Empty or absent refuses the
item, at every regime without exception.

**The item is FLAT, and its body field names are the record schema's.** An item
is `pattern` plus exactly the fields the position declares, in one object; there
is no nested `body`, and the names come from `issueschema.ReadingBodyFields`
rather than from a table restated here. Both facts are a correction to this
spec's first draft, which predated the four definitions: those definitions —
shipped, and what the producer actually reads — instruct a flat item and those
field names, and the reading record family already uses them. A verb built to
the earlier table would have refused every output the shipped instrument can
produce, and would have needed a rename between the payload and the record for
no gain (iss-2608311123267602).

### The bodies, per position

Item bodies are position-typed, and the schema for the wrong position's body
is not merely unusual but undecodable. JSON keys are US English, as code-side
text is throughout the repository:

| Regime | Body fields |
| --- | --- |
| `generative` | `configuration`, `what_admits_it` |
| `explicative` | `claim_surfaced`, `claim_type` (`criterion` / `causal` / `context`), `what_implies_it` |
| `evaluative` | `candidate_id`, `criterion`, `characterisation` |
| `registrative` | `tension`, `constraint_in_play`, `why_a_tension` |

The table is `issueschema.ReadingBodyFields` read out, not a second copy: the
verb derives the closed key set from it, `TestDefinitionHoldsItsFiveParts`
already holds the four definitions to it, and the record writer writes from it.
One vocabulary reaches from the instruction through the payload to the record.

### The regime gate

**The regime's source of truth is the definition.** The verb resolves the run's
position to `agents/cold-reading-<position>.md`, reads its `regime:`
frontmatter key (spc-62), and compares. A payload whose self-declared regime
disagrees is refused at list level. There is no `--regime` flag and no
configuration key: the value cannot be reached from operator input.

`issueschema.ReadingPositions` also binds a position to a regime, and the two
are reconciled rather than left to agree by habit. The ingest verb reads ONLY
the definition, through `reading.LoadDefinition`; the schema table is the record
family's vocabulary (which regimes exist, and which body a position returns),
and `TestRegimeValuesAreTheFourAndDistinct` in `definitions_test.go` already
holds every shipped definition to it. So the definition is the source of truth
at runtime, the table is the source of truth for the record schema, and a drift
between them is a test failure rather than a silent disagreement.

Two enforcement layers sit behind that:

- **Reserved names.** Strict decoding already refuses an unknown field, but a
  generic "unknown field" message is a poor account of a licence breach. Each
  position therefore declares a reserved-name table (`evaluative`: `rank`,
  `score`, `recommended`, `order`; `registrative`: `resolution`, `fix`,
  `remedy`; `explicative`: `disposition`, `status`), and a payload naming one
  is refused with the field named and the licence stated. Arrangement order is
  never refused: items arrive in document order by mandate.
- **Where a reserved name is matched.** Against the item's own KEYS, and the
  keys of any nested object the contract does not define — never against the
  words inside a value (ruling of 2026-09-01 on iss-2608311518056854). A reading
  that REPORTS what the passed material disposed, ranked, recommended or fixed
  lands at every regime; only a decision carried as a field of the reading's own
  output refuses. `generative` declares no reserved name, because its licence is
  the widest and the constraint on it falls at admission.
- **No semantic signature.** The gate once carried a registry of named prose
  detectors (`RG-EVAL-ORDERING`, `RG-EVAL-RECOMMENDATION`, `RG-REG-FIXPROPOSAL`,
  `RG-EXPL-DISPOSITION`) over an item's text. It could not tell a reading that
  proposes from one reporting somebody else proposing, and measured over
  thirty-four realistic outputs it caught fourteen, every one for quoting the
  document it read. The registry is withdrawn; recording the hit rather than
  refusing it was considered and rejected, because a gate that reads prose is
  still reading prose.

**Key matching runs over a folded copy, and so does the provenance rule.** A
reserved name respelled in code points that render the same is the same key: a
fullwidth `ｄｉｓｐｏｓｉｔｉｏｎ`, an `ﬁ` ligature in `fix`, a zero-width space
inside `score`. `strings.TrimSpace` likewise does not treat a zero-width rune as
blank, so a `pattern` of one U+200B satisfied ac-11 at all four regimes and the
record then asserted a provenance it does not carry.

`foldForMatching` closes three classes and the stored text is untouched: every
Unicode space folds to ASCII; every invisible rune is dropped, across all three
categories that hold one — `Cf`, `Other_Default_Ignorable_Code_Point` (U+034F is
a MARK, so guarding `Cf` alone left it open) and `Variation_Selector`; U+2800
BRAILLE PATTERN BLANK folds to a space, being the one rune found that renders as
nothing and is a GRAPHIC character, so no category above reports it; and NFKC
folds the compatibility forms (the `fi` ligature, the fullwidth letters). Both
the reserved-name rule over keys and the blankness rules read through it, and
neither reaches a value's prose.

**What stays open, stated plainly because this section previously read as a
closure:** a script-CONFUSABLE substitution in a KEY. A Cyrillic that is not the
Latin one, and no normalisation equates them, so a reserved name spelled in a
confusable script is refused as an unknown field rather than as a named licence
breach. Closing it needs a confusables table, a new dependency and a maintainer's
decision. itd-185's disclosed residue names this class alongside the one the
withdrawal of the registry opened: a reading that proposes in PROSE, carrying no
field for it, raises nothing.

The line between defect and residue is one test: a reserved name with a byte
substituted is an evasion of the gate, and a proposal that carries no field at
all is a limit of it. The invisible and compatibility classes fell on the defect
side and are closed (iss-2608311306535485, iss-2608311351290623); the confusable
class falls on the defect side too and is open, which is why it is disclosed
rather than filed under the residue.

**The gate refuses only a real decision field.** The registry of prose
detectors was degraded from enforce to flag on 2026-08-31 and withdrawn
altogether on 2026-09-01, and the second ruling supersedes the first: a hit that
is recorded rather than refused is still a hit on a reading for quoting the
document it read. What remains is one structural rule over the item's key set,
and `internal/core/reading/ingest_corpus_test.go` holds it to the thirty-four
realistic outputs the noise was measured on — every honest report lands with
nothing raised against it, and every reserved name carried as a key refuses,
naming itself and the licence.

### Manifest reference

`manifest_sha256` is the content hash of the assembler's manifest, the one
unforgeable reference. Ingest resolves the parked run at
`.abcd/.work.local/scratch/reading-runs/<run_id>/manifest.json`, hashes it,
and refuses the run when nothing resolves or the hashes disagree. Only after
acceptance is the manifest promoted to
`.abcd/development/readings/<run-id>/manifest.json`.

**Instrument identity** (ruling (12)) is `model` plus `definition_sha256`
plus `assembler_version`, all three required and all three carried into run
metadata, so two runs claiming the same instrument are provably the same. The
definition hash is recomputed here from the definition file and compared with
the payload's claim; the assembler version is compared with the manifest's.

### Staged writes, run metadata last

The only durable thing a run writes before its payload validates is its own
`refusal.json`, and only once its identity is proven.

1. Validate: schema, regime, provenance, manifest reference, instrument. A
   list-level refusal from the point the identity is proven writes
   `refusal.json` and stops.
1b. Sweep any orphaned stage, rolling its reading records out of the committed
   ledger. This rides with the COMMIT and not with a refusal: the rollback is a
   destructive act in the committed tier, and a run on its way to a refusal must
   not destroy another run's records (iss-2608311517509690). A refusal does roll
   back its OWN run's earlier crashed attempt, because a refused run leaves no
   reading records. An orphan of another run therefore outlives an invocation
   that does not validate, and the bare verb reports it as `orphaned_ingests`.
2. Stage a write-aside marker into
   `.abcd/.work.local/scratch/reading-ingest/<run-id>/stage.json`.
3. Write the reading records into the reading-record family (spc-58's
   `.abcd/work/issues/readings/<run-id>/`) through `capture.IngestReading`, and
   stamp the ids that landed back onto the marker.
4. Promote the manifest, then write `.abcd/development/readings/<run-id>/run.json`
   **last**: the run metadata is the commit marker, so a run without one never
   happened. The stage is cleared once the marker is down.

The stage holds a MARKER rather than the rendered records, which is a departure
from this spec's first draft and is taken for a reason. `capture.IngestReading`
mints every item id under the ledger lock, probing the tree it is about to write
into, precisely so two ingests landing in one second cannot draw one id; staging
the rendered records into a second issues root would move that probe off the
real ledger and reopen what the lock was taken for. One record writer, one mint,
one lock — and the stage does the job ac-2 actually asks of it.

**Two locks, always in one order.** The stage flock covers the sweep through the
commit marker; the sweep's unlink of committed reading records additionally takes
core/capture's LEDGER lock through `capture.WithLedgerLock`, so a concurrent
disposition or promote waits rather than reading a record as it disappears. The
ledger lock is taken around the unlink alone and never around
`capture.IngestReading`, which re-takes it internally — an flock is not
reentrant. The order is always stage-then-ledger, so no cycle exists.

**One ingest at a time in one checkout**, from the sweep through the commit
marker, behind an flock on the stage root (`fsutil.WithFileLock`). The sweep
DELETES committed reading records, and its only test for an orphan is a stage
with no commit marker beside it — which is exactly what a live ingest looks like
between its ledger write and its marker. Without the lock a second invocation
rolls the first one back mid-flight, and the first then writes a run record
naming records that no longer exist and exits 0. capture's ledger lock cannot
serve: `IngestReading` re-takes it internally, and the sweep sits outside it.

**Every path inside the repository is resolved through an `os.Root` opened at
the repository root.** The run-id grammar makes the run id a safe path
COMPONENT and says nothing about the components above it, and this verb writes
and DELETES under two directories a hostile clone can commit a symlink at. The
orphan sweep runs before the payload is even read, so no valid payload is needed
to reach it. A directory the sweep walks or removes from is additionally refused
when it IS a symlink, which is the stance `capture` takes on the same ledger
directory.

**A rerun is refused.** The run id is payload-chosen, so a second ingest of one
run would otherwise land a second batch beside the first and rewrite the run
metadata to name only the second — leaving the first batch unreachable from any
run record and beyond every later sweep, because the rollback bails whenever a
commit marker exists. An id that already carries a commit marker or a refusal
record is refused before either can be overwritten.

An orphaned stage found on a later invocation is reported by name and cleared,
and the run is ROLLED BACK: a run whose commit marker never landed never
happened, so its reading records are removed (bounded by the `rdi-N.md`
filename grammar, so a file a person put in the directory survives), and its
own directory with them. A stage beside a run whose marker DID land is a
leftover from a crash after the marker; that run is complete and only the stage
goes. A crash mid-ingest therefore leaves evidence, and the next invocation
leaves neither half a run nor a stale one.

### Refusal granularity and refusal records

An item-level violation refuses that item and lands the rest, naming the
refused item's ordinal and the rule. Beyond the schema and the regime, an item
is refused when a closed body vocabulary is broken (`claim_type`), and when the
record it would become would exceed `issueschema.RecordReadLimit` — an oversize
record is durable and, because every reader of the family applies that limit,
permanently unanswerable.

**The record's size is DECIDED where the exact byte count exists**, in
`capture.IngestReading` on the assembled content: values already redacted,
already escaped, and the string that reaches the disk. Two attempts to decide it
from the payload failed the same way — each modelled one lengthening step and
missed the next, and a record written past `issueschema.RecordReadLimit` is
durable and unreadable by every reader of the family, so the item can never be
dispositioned. `recordBytes` remains upstream as a cheap early FILTER, which is
what gives an obviously oversize item an item-level refusal that lands the rest
of the run; it is not the decision, and it does not have to be exact.

Three caps bound what a payload can put into a message or a durable record, in
the three dimensions it chooses: the length of one value, the number of field
NAMES one refusal quotes, and the number of REFUSALS a run carries. A per-value
cap bounds nothing when the payload chooses how many values there are. The
refusal count is reported separately, so bounding the list hides nothing.

Item text is passed to the record writer through `termsafe.EncodeHiddenRunes`,
after the checks rather than before: the record is a committed markdown file a
reviewer reads in a terminal, and the writer's own scalar guard refuses runes
below 0x20 and nothing above. Encoding first would defeat the checks — a pattern
of one tab encodes to a non-blank string, and the provenance rule would pass it.
Every payload-derived value that reaches a message or a durable record goes
through `termsafe.CleanProseLine`, the repository's canonical untrusted-prose
cleaner, under a per-value cap and a cap on how many payload-chosen NAMES one
refusal quotes. A list-level violation (bad `_type`,
regime mismatch, unresolvable manifest, missing instrument field) refuses the
whole run. **A refusal writes a durable refusal record once the run's identity is proven**
— that is, once the run id resolves to a parked manifest whose content hash
matches: `.abcd/development/readings/<run-id>/refusal.json`, carrying the run
metadata and the whole named reason and no items. A refusal reached BEFORE that
point writes nothing anywhere, because there is no proven run to record against,
and ac-1 requires exactly that. The reason is carried WHOLE: every
payload-derived substring inside it is cleaned where it is interpolated, so a
second cap over the composed sentence would only truncate the repository's own
prose. The front door renders the result on a recorded refusal before it exits
2, so the record path is reachable rather than named only in the error text.

## Acceptance criteria mapping

The criteria were split on 2026-08-31, before this spec was built, so that no
criterion conjoins a structural half a gate holds with a semantic half bounded
by a registry. The numbering below is the positional authority ac-1..ac-13.

| itd-185 criterion | How spc-63 satisfies it | Test |
| --- | --- | --- |
| ac-1 — malformed output refused, nothing durable anywhere | Validation is step 1 of four; staging is local-tier; the durable write happens only after the whole payload validates | `TestMalformedPayloadWritesNothing`, `TestNoDurableWriteBeforeValidation`, `TestUnknownFieldRefusedAtEveryLevel` |
| ac-2 — a fault between staging and the commit marker leaves no half-run, and the orphan is named and cleared | Run metadata is written last as the commit marker; an orphaned stage found on a later invocation is reported by name, the run is rolled back, and the stage is cleared | `TestRunMetadataLandsLast` (both fault windows), `TestOrphanedStageIsReportedAndCleared`, `TestOrphanSweepLeavesACommittedRunAlone` |
| ac-3 — the manifest reference resolves, and a mismatch refuses the run | `manifest_sha256` is checked against the parked manifest's own content hash before anything is written | `TestManifestReferenceMustResolve`, `TestManifestHashMismatchRefusesRun` |
| ac-4 — a registrative reserved name refuses, naming ordinal, field and licence | The `registrative` reserved-name table: `resolution`, `fix`, `remedy` | `TestRegistrativeResolutionFieldRefused` |
| ac-5 — a registrative body reporting a fix lands; only a `fix`, `remedy` or `resolution` KEY refuses | The reserved name is matched against the item's own keys, never against a value's prose | `TestEveryHonestReportLandsWithNothingRaisedAgainstIt`, `TestEveryRealDecisionFieldRefuses`, `TestTheRulingsExamplesPassAsValuesAndRefuseAsKeys` |
| ac-6 — an evaluative reserved name refuses, naming the field | The `evaluative` reserved-name table: `rank`, `score`, `recommended`, `order` | `TestEvaluativeRankScoreRecommendedRefused` |
| ac-7 — arrangement order alone is accepted | Arrangement order is never inspected: items arrive in document order by mandate | `TestEvaluativeDocumentOrderIsNeverRefused` |
| ac-8 — an explicative disposition-bearing field refuses, naming the field | `disposition` and `status` are reserved on the explicative body, and the item's key set is closed against the position's own body fields, so any field outside it is refused by name | `TestExplicativeDispositionRefused`, `TestWrongPositionBodyIsUndecodable` |
| ac-9 — an explicative body reporting a disposition lands; only a `disposition` or `status` KEY refuses | The reserved name is matched against the item's own keys, never against a value's prose | `TestEveryHonestReportLandsWithNothingRaisedAgainstIt`, `TestEveryRealDecisionFieldRefuses`, `TestTheRulingsExamplesPassAsValuesAndRefuseAsKeys` |
| ac-10 — a list-level refusal writes a refusal record and no items | The refusal path writes `refusal.json` carrying run metadata and the named reason; the stage is never taken | `TestListLevelRefusalWritesRefusalRecordOnly` |
| ac-11 — an empty or absent `pattern` refuses the item at every regime | Provenance is one envelope field, checked before the body, at all four regimes without exception | `TestEmptyPatternNamedRefusesItemAtEveryRegime` (all four positions, three forms each) |
| ac-12 — a self-declared regime disagreeing with the definition refuses the run | The regime's source of truth is the definition, resolved through the run's position; the payload's claim is compared, never trusted | `TestSelfDeclaredRegimeMismatchRefusesRun`, `TestRegimeComesFromTheDefinitionNotThePayload`, and `TestADriftedDefinitionRefusesTheRunRatherThanChangingTheLicence` for the adversarial half the criterion does not state — the DEFINITION lying rather than the payload |
| ac-13 — item ids are minted by the verb, and a supplied id is an unknown field | `recordid.Minter.Mint("rdi")` through `capture.IngestReading` on acceptance; the payload schema carries no item identifier at all | `TestItemIDsAreMintedByTheVerb` |

ac-5 and ac-9 were written against a signature registry that no longer exists;
they are rewritten to the structural rule, and itd-185 discloses what that
leaves open — a proposal made in prose, carrying no field, raises nothing. Their
structural halves, ac-4, ac-6 and ac-8, are unbounded in the other direction:
the field is present or it is not.

Two behaviours this spec delivers carry no criterion of their own, and are
recorded here rather than left to be discovered at the audit: item-level
refusal granularity, which lands the surviving items when one item is refused
(`TestItemLevelViolationLandsTheRest`), and the generative position's absence of
a reserved-name row, its constraint falling at admission instead
(`TestGenerativeHasNoRegimeRefusal`).

ac-13 is checked against the mint's SHAPE rather than through an injected clock
and entropy, which is a departure from this spec's first draft. The mint is
`capture.IngestReading`'s, taken under that package's ledger lock so the
collision probe sees the tree it is about to write into, and its seams are not
reachable from this package. What the case establishes is what the criterion
asks: every id on an accepted run came from a mint and none from the payload,
and a payload supplying its own is refused by name.


## Tests

Each case is written to fail before the change and pass after, in
`internal/core/reading/` unless named otherwise.

- `ingest_schema_test.go`: `TestMalformedPayloadWritesNothing`,
  `TestUnknownFieldRefusedAtEveryLevel`,
  `TestWrongPositionBodyIsUndecodable`,
  `TestNoDurableWriteBeforeValidation`,
  `TestMissingBodyFieldRefusesTheItem`,
  `TestAnOversizeItemIsRefusedRatherThanWrittenUnreadable`,
  `TestEscapingCannotPushARecordPastTheLimit`,
  `TestTheRefusalListIsBoundedInCount`,
  `TestAClosedBodyVocabularyIsEnforced`,
  `TestARefusalNeverEchoesRawPayloadBytes`,
  `TestTheCommittedRecordCarriesNoHiddenRunes`,
  `TestTheRefusalRecordCarriesTheWholeReason`,
  `TestPatternFieldIsTheRecordEnvelopeField`.
- `ingest_regime_test.go`: `TestRegimeComesFromTheDefinitionNotThePayload`,
  `TestADriftedDefinitionRefusesTheRunRatherThanChangingTheLicence`,
  `TestSelfDeclaredRegimeMismatchRefusesRun`,
  `TestEvaluativeRankScoreRecommendedRefused`,
  `TestEvaluativeDocumentOrderIsNeverRefused`,
  `TestRegistrativeResolutionFieldRefused`,
  `TestExplicativeDispositionRefused`,
  `TestGenerativeHasNoRegimeRefusal`,
  `TestInnocentProseCarryingFoldedRunesStillLands`,
  `TestTheProvenanceFieldIsJudgedByTheSameRuleAsAnyOther`,
  `TestReservedNamesNeverCollideWithABodyField`.
- `ingest_corpus_test.go`: the thirty-four-case realistic corpus and the
  decision-field half — `TestEveryHonestReportLandsWithNothingRaisedAgainstIt`,
  `TestEveryRealDecisionFieldRefuses`,
  `TestTheRulingsExamplesPassAsValuesAndRefuseAsKeys`.
- `ingest_provenance_test.go`: `TestEmptyPatternNamedRefusesItemAtEveryRegime`.
- `ingest_fixture_test.go`: the shared fixture. Every refusal case is built by
  mutating a payload it produced, and asserts the adjacent LEGAL item in the
  same payload landed — so no case could pass against a verb that refused
  everything.
- `ingest_stage_test.go`: `TestRunMetadataLandsLast`,
  `TestOrphanedStageIsReportedAndCleared`,
  `TestOrphanSweepLeavesACommittedRunAlone`,
  `TestItemLevelViolationLandsTheRest`,
  `TestListLevelRefusalWritesRefusalRecordOnly`,
  `TestRunIDNeverBuildsAPathBeforeItIsChecked`,
  `TestARerunOfACommittedRunIsRefused`,
  `TestTheStageLockIsHeldAcrossTheSweepAndTheWrite`.
- `ingest_regime_test.go` also carries
  `TestTheRegimeGateIsNotEvadedByInvisibleRunes`, whose last case is the one
  that keeps the fix honest: a non-breaking space in innocent prose is still
  ACCEPTED, so an evasion has not been traded for a false refusal.
- `ingest_containment_test.go`: the adversarial-repository cases —
  `TestASymlinkedReadingsTreeCannotRedirectTheDurableWrite`,
  `TestASymlinkedLedgerRunCannotRedirectTheRollback`,
  `TestASymlinkedDurableRunCannotRedirectTheRollback`,
  `TestASymlinkedStageRootCannotRedirectTheSweep`,
  `TestASymlinkedParkedRunCannotRedirectTheManifestRead`. Each plants the
  symlink a hostile clone commits, runs the verb, and asserts the directory
  outside the repository is untouched.
- `ingest_identity_test.go`: `TestManifestReferenceMustResolve`,
  `TestManifestHashMismatchRefusesRun`,
  `TestInstrumentIdentityRequiresAllThreeParts`,
  `TestItemIDsAreMintedByTheVerb` (the mint's shape; a payload-supplied id is
  an unknown field).
- `internal/surface/cli/reading_surface_test.go`:
  `TestIngestRequiresOutputJSON`, `TestIngestReachesBothPlanes`,
  `TestIngestRendersARefusalRecord` and `TestIngestExecutesEndToEnd`, which assembles a run through the sibling verb
  and ingests its output, so the wiring claim is a run rather than a tree walk.
  The no-operator-surface guard on the regime is spc-62's
  `TestNoOperatorSurfaceSetsARegime`, in its own file.

Three further cases were added because a MUTATION run found their guards
vacuous — each was neutralised in a copy of the tree and every test stayed
green. `TestMissingBodyFieldRefusesTheItem` covers an item partial at its own
position, which the foreign-body case never reached because the unknown-key
check fires first. `TestARefusalNeverEchoesRawPayloadBytes` binds the
sanitiser and the length cap on every payload value that reaches a message or
the durable refusal record. And `TestOrphanSweepLeavesACommittedRunAlone`
binds the rollback's commit-marker guard.

Six further cases hold properties this spec states in prose and would
otherwise only assert there. `TestPatternFieldIsTheRecordEnvelopeField` holds
the payload key to the record's own envelope field.
`TestReservedNamesNeverCollideWithABodyField` keeps a reserved name from being
a body field, which would refuse every legal output at that position.
`TestOrphanSweepLeavesACommittedRunAlone` holds the rollback off a run that
committed. `TestRunIDNeverBuildsAPathBeforeItIsChecked` is the trust boundary at the
input end: the run id is the one payload value a path is built from, and a
traversal id is refused before any file is opened.
`TestARefusalNeverEchoesRawPayloadBytes` is the trust boundary at the OUTPUT
end: a refusal quotes model-produced text to a terminal and writes it into a
durable record, so an escape sequence must not rewrite the message that reports
it and an oversized field must not drown it.

## Out of scope

- **Running a reading.** The instrument ships unrun for the whole cycle: this
  verb is exercised against fixture payloads only, and no reading is
  commissioned by this delivery.
- Whether the gate lints cleanly against a REAL reading. The corpus it is held
  to is constructed, and no reading has been run through this verb.
- Teaching the issue-resolution gates about the new reading-record folders,
  and the reading-record schema itself: spc-58's.
- The standing tension with the repository's widen-options promotion clause
  ("calibrated before it gates") is recorded, not resolved here; the ruled
  design governs the instrument meanwhile.
