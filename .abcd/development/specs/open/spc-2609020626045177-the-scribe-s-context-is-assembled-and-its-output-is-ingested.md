---
id: spc-2609020626045177
slug: the-scribe-s-context-is-assembled-and-its-output-is-ingested
intent: itd-2609020625402599
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The scribe verb: an allow-list assembler, an authoring-refusing ingest, and a per-run context stamp the transcript store can check

## Summary

spc-2609020626045177 delivers
[itd-2609020625402599](../../intents/planned/itd-2609020625402599-the-scribe-s-context-is-assembled-and-its-output-is-ingested.md).
`abcd scribe assemble --run <rdg-N>` builds the scribe's context for one
ingested run from the ledger's allow list, which is derived from the ledger's
own directory constants, and from the researcher's supplied dispositions text:
The run's reading records as the store holds them, the standing dispositions,
admissions, surprises and reframes, and the supplied text. It parks the
context and a manifest of what it passed in the local tier. `abcd scribe
ingest --scribe-json <path>` validates the scribe's four outputs, writes
dispositions, admissions and surprises through the capture verbs and the
gate they share, refuses a payload that authors anything, and promotes the
manifest beside the run inside the read block. Every reading bundle and every
scribe context carries a per-run context stamp, the transcript store records
at capture which stamps a transcript carried, and `abcd history` gains a check
that names a retained transcript carrying both stamps of one run, reports
that no retained transcript carries two stamps of one run when none does, and
says when the property is unobserved.
[spc-66](../closed/spc-66-machine-assistance-in-maintaining-the-ledger-without-any-con.md)
met both of
[itd-188](../../intents/shipped/itd-188-machine-assistance-in-maintaining-the-ledger-without-any-con.md)'s
criteria by declaration; this spec meets them by mechanism.

The decision this spec builds on is adopted:
[adr-2609021016275803](../../decisions/adrs/2609021016275803-no-session-holds-both-a-reading-and-the-ledger-and-a-per-run.md)
carries `status: accepted`, adopted by the maintainer on 2026-09-02 at the
planning interview after being checked against the design framework and the
readings companion, and
[iss-2609020626114155](../../../work/issues/resolved/iss-2609020626114155-adr-and-brief-invariant-owed-no-session-holds-both-a-reading.md)
is resolved by it. Brief invariant 15 already carried the sentence "No session
holds both a reading and the ledger"; the ADR states the mechanism behind it,
the per-run context stamp and the session-separation check that reports
held-for-what-was-seen or unobserved and never clean, and the invariant is
amended to name both. The ADR also states, so nobody reads it as a departure
from the framework, that the scribe transcribes dispositions and reads the
run's reading records from the store: The ingest verb writes the reading
records first, and the scribe is handed the committed records rather than a
raw output a second time, which is what this spec's assembler does below.
`scribe` is a top-level verb, because the two contexts must never share a
front door.

## Scope

In: The `internal/core/scribe` package with its assembler, context, manifest
and ingest; the `scribe` verb and its two sub-verbs; the `sessionkind` package
and the per-run stamp on the reading bundle and the scribe context; the
transcript store's `context_stamps` metadata and the separation check; the
`history separation` sub-verb and the line on `history list`; the definition's
Inputs and Delivery sections; the plugin page, the brief chapter and its index
row, the history chapter's sub-verb row, the internals README entry and the
surface snapshot; the eval plant.

Out: Any judgement by either verb about the content transcribed; a change to
the scribe's access rule; redaction policy, which the capture validators
already apply; the ordering gate the disposition writer holds, which
spc-2609020626040342 builds and this verb inherits; the durable-tier writer,
which spc-2609020626039834 exports and this verb calls; the ADR and the
invariant amendment.

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

This spec's classes: `SchemaVersion` on the bundle and the manifest moves by
one, because the bundle gains a field; `AssemblerVersionCore` moves MINOR,
because the bundle shape is part of the contract. It lands fourth in Phase B,
after the admission and surprise verbs, whose `Admit` and `Surprise` it
writes through, and after the reframe record, whose reframes directory its
derived allow list already names.

### The context stamp, and where it lives

`internal/core/sessionkind` is a leaf package: The two kinds `Reading` and
`Scribe`, `Stamp(kind, run, digest string) string` rendering
`abcd.context-stamp/<kind>/<rdg-N>/<sha256-12>`, `Parse`, and `StampRe`, the
one expression that recognises a stamp. Both `reading` and `history` import
it; neither imports the other. A stamp is per run, so a session that reads
the docs or the source, where the kind tokens are committed, carries no
stamp, and a stamp is matched exactly, so a transcript carrying the reading
stamp of one run and the scribe stamp of another is not a violation.

The digest is the first twelve hex digits of the sha256 over the context's
own item set: For the bundle, its `items` array as serialised; for the scribe
context, its `ledger` array. `Bundle` in `internal/core/reading/manifest.go`
gains `ContextStamp string` (`context_stamp`), set by `Assemble` for the
run it mints; it is a token, not a path, so brief invariant 15 holds, and
`TestNoBundleFieldIsAScopeSelector` in `internal/core/reading/scope_test.go`
gains the member. The closed bundle allow-set in
`TestBundleGainsNoFieldFromTheReport` in `internal/core/reading/size_test.go`
gains `context_stamp`; the item allow-set there does not move for this spec.
The scribe context carries `context_stamp` in the same position.

The stamp reaches a transcript because a session reads its bundle or its
context through a tool whose result the host retains, which is the intent's
mechanism and its disclosed limit: Where a host hands content the transcript
does not retain, the check reports the property unobserved.

### `scribe assemble`

`internal/core/scribe/assemble.go` adds `Assemble(req AssembleRequest)` with
`AssembleRequest{RepoRoot, Run, DispositionsPath, OutDir, OutDirLabel string;
DryRun bool}`. `Run` is validated by `recordid.ValidReadingRunID` and must be
an ingested run, with its commit marker at
`.abcd/development/readings/<rdg-N>/run.json`, refused otherwise: The scribe
transcribes a reading the ledger already holds, so the run's items come from
the store, never from a raw reading output. The one supplied path is the
researcher's dispositions text, read behind `fsutil.ReadGuarded` at
`reading.MaxFileBytes`, in any form they wrote it.

The context is positive inclusion at directory grain. `AllowList()` is
derived from `issueschema.LedgerDirs()`, the function spc-2609020626039834
adds beside the ledger directory constants, under `capture.LedgerRelPath`:
`readings`, `dispositions`, `admissions`, `surprises`, `reframes` and the
three status directories `open`, `resolved` and `wontfix`. A family the
ledger declares later is on the list the day its constant is, and the
comparative position's exclusion rows derive from the same function, so the
two instruments describe one set. The collector walks those directories and
nothing else, refusing a symlinked directory or leaf as `capture`'s readers
do, reading each `.md` behind `fsutil.ReadGuarded` at
`issueschema.RecordReadLimit`. The shipped tree, the brief, the intents, the
specs, the decisions and the transcript store are excluded because no walk
starts in them, and `assertAllowList` is the fail-closed half: It refuses to
emit any item whose path lacks one of the allow-list prefixes, so an item
that reached the context by any future route is refused rather than
disclosed (adr-56). The transcript store sits outside the repository tree and
is unreachable by construction; the context also names no home path, held by
the same scrub `reading` applies.

The context is `abcd.scribe.context/1`: `_type`, `schema_version`,
`context_stamp`, `run`, `ledger` (one `{path, text}` per record, sorted by
path, the named run's reading records among them), and `supplied`
(`dispositions`, verbatim). The scribe's material may name ledger paths,
because the scribe files by them; it is the reading bundle that may not. It
lands at `.abcd/.work.local/scratch/scribe-runs/<rdg-N>/context.json` or
under `--out`, on `reading assemble`'s empty-directory and dry-run rules.

The manifest is `abcd.scribe.manifest/1`: `_type`, `schema_version`,
`context_stamp`, `run`, `definition_sha256` (the hash of `agents/scribe.md`,
read as `LoadDefinition` reads a reading definition), `context_sha256`,
`supplied` (`dispositions_sha256`), `items` (one `{path, bytes, sha256}` per
ledger record), `allow_list`, and `exclusions` (the shipped tree, the durable
record outside the ledger, the local tier, and the transcript store, each
with its signal). At assemble it is parked beside the context as
`scribe-manifest.json`, in the local tier, and nothing in the durable tier is
touched: A scribe session that is assembled and never ingested leaves no
trace beside the run.

### `scribe ingest`

`internal/core/scribe/ingest.go` adds `Ingest(req IngestRequest)` with
`IngestRequest{RepoRoot, ScribeJSONPath, ContextPath string}`. The payload is
`abcd.scribe.output/1`, decoded strictly:

```json
{
  "_type": "abcd.scribe.output/1",
  "run": "rdg-N",
  "context_sha256": "…",
  "dispositions": [{"item": "rdi-N", "state": "accepted", "grounds": "…",
                    "exit_condition": "", "supersedes": "", "recurs": []}],
  "admissions": [{"item": "rdi-N", "grounds": "…"}],
  "surprises": [{"occasioned_by": "…", "text": "…"}],
  "fidelity_flags": [{"first": "…", "second": "…"}],
  "outstanding": ["rdi-N"],
  "refusals": [{"subject": "…", "reason": "…"}]
}
```

The run's identity is proven as `reading.Ingest` proves one: The parked
manifest is read from the local tier, and the context at `ContextPath`
(default, the local-tier path above) must hash to the manifest's
`context_sha256`. Until that holds nothing is written.

The authoring refusal has three parts, each naming the field and the item. A
key outside the closed shapes above, `resolution`, `pattern`, `position` or
any other, is refused by strict decoding with the key named. A disposition
whose `item` does not occur in `supplied.dispositions` is a disposition the
researcher did not supply. A disposition `grounds`, an admission `grounds`,
an `exit_condition` or a surprise `text` that does not occur verbatim, after
whitespace folding, in `supplied.dispositions` is a ground the researcher did
not write. The scribe reformats; it never adds a word, and the check is that
every word it carries was already there. Every item in `outstanding` must be
an item of the run with no entry in `dispositions`, and the result reports
the list; an item of the run that appears nowhere in the payload is refused,
because silence is not one of the scribe's options.

Writes go through the verbs' own functions, in payload order:
`capture.Disposition`, then `capture.Admit`, then `capture.Surprise` (the
last two delivered by
[spc-2609020626040342](spc-2609020626040342-an-admission-and-a-surprise-are-written-by-a-verb-and-the-or.md)),
each under the ledger lock it takes for itself, each inheriting the redaction
and the refusals it already applies. The verb adds no validation path of its
own. Two inherited refusals are named here because a scribe payload meets
them whole. First, the ordering gate in the shared disposition writer: At the
widening position no disposition in any state is written until a committed
comparative run names the item's run, so a payload over a widening run nobody
has characterised refuses at its first disposition, before anything lands,
and the result names the comparative run the ledger is waiting for. Second,
`Admit`'s admission-alone branch requires the admission's ground to equal the
standing disposition's, so a payload carrying a disposition and an admission
for one item on two grounds refuses at the admission and names both texts;
spc-2609020626040342's one-ground rule therefore holds on this path without a
second check. A refusal from any write stops the ingest and the result names
what landed before it, so a rerun can see the partial state rather than mint
it twice. Fidelity flags and refusals are carried into the result unresolved
and never into a record, which is what the definition promises.

Once every write has landed, the parked manifest is promoted to
`.abcd/development/readings/<rdg-N>/scribe-manifest.json` through
`reading.WriteRunArtefact`, the one exported durable-tier writer, which
spc-2609020626039834 adds: It opens the containment root `reading.Ingest`
uses, requires the run's `run.json`, and refuses an existing file, because the
durable tier is write-once at emission. Promotion comes last so that a refused
ingest leaves the manifest parked and a rerun re-proves the same context. That
directory is denied to every assembly by the exclusion floor's
`.abcd/development/readings` row, so the next reading cannot see it.

### The transcript store's half

`history.Capture` in `internal/core/history/history.go` scans the raw
transcript with `sessionkind.StampRe` before redaction (a stamp carries no
secret) and records every distinct stamp it found in the frontmatter as
`context_stamps`, a comma-joined list; `Record` gains `ContextStamps
[]string`, and `marshalRecord` and `parseRecord` in
`internal/core/history/store.go` carry it, the field being optional on read so
that a record captured before this change still parses and
`recordSchemaVersion` does not move. The stamp is therefore metadata, and the
check reads metadata only, which is the consumer brief invariant 15 already
enumerates as "session-separation evidence (metadata only, never bodies)".

`internal/core/history/separation.go` adds
`SessionSeparation(rootSHA string) (SeparationReport, error)` over `List`:
`{Transcripts, Stamped int; Runs []string; Violations []Violation{SessionID,
File, Run string}; Unobserved bool; Reason string}`. A violation is a
transcript carrying both the reading stamp and the scribe stamp of one run,
named by session id, filename and run. `Runs` lists every run any stamp
named, so a clean report says what it examined. `Unobserved` is true, with
the reason stated, when the store holds no transcript or none carrying any
stamp: A check that ran and saw nothing and a check that could see nothing
must not produce the same artefact (adr-56).

`abcd history separation` is a read-only sub-verb rendering the report: The
violations by name, or "no retained transcript carries two stamps of one run"
with the runs it saw, or the unobserved reason. `history list`'s text render
ends with the same one line; the list's JSON stays an array, so a consumer of
it is untouched. The smoke lane's `TestReadOnlyVerbsRun` in
`evals/smoke_test.go` gains `history separation`.

### The surfaces and the definition

`internal/surface/cli/scribe.go` registers `scribe` with `assemble` and
`ingest`, on `reading.go`'s shape: Closed operands, exit 2 on every refusal,
the operator's own spelling of `--out` echoed back, and a result rendered on
the refusal path whenever it has something to disclose. `commands/scribe.md`
is the plugin page and states the host obligation: The scribe session is
granted the context and nothing else, and it is not the reading session.
`.abcd/development/brief/04-surfaces/24-scribe.md` is the chapter, with its
sub-verb table; the surfaces `README.md` gains row 24 and `scribe` in its
`<!-- index: commands -->` region; `11-history.md` gains the `separation`
row; `.abcd/development/release/surface.json` is regenerated.
`internal/README.md` gains entries for `core/sessionkind/` and
`core/scribe/`, on the `core/grounds/` entry's shape: What the leaf holds and
why it is not inside either package that imports it.

`agents/scribe.md` moves to `prompt_version: 0.2.0`, with its entry under the
shared dated heading in `agents/CHANGELOG.md` that every prompt bump landing
in this iteration uses. Its Inputs list is regenerated from `AllowList()`, so
it will name the admissions, surprises and reframes stores beside the five
directories it names today: A completion of the ledger enumeration rather
than a change to the rule, which stays ledger content only, and
`TestScribeInputsAreLedgerOnly` in `internal/core/lint/scribecontract_test.go`
keeps holding it while a new test there holds the list to the function. Its
Delivery section names the two verbs and the payload above in place of "There
is no ingest verb", and the scribe protocol in
`.abcd/development/brief/05-internals/01-agents.md` is edited in two places:
Rule 3, which names `reading ingest` and `capture disposition` as the paths a
transcribed record reaches the tree through, names `scribe ingest` beside
them, and the sentence "Mechanical assembly belongs to the ingest verb" names
`scribe assemble`.

### The eval plant

The read-block fixture gains
`.abcd/development/readings/rdg-2608300900000001/scribe-manifest.json`
carrying the `EXHAUST` sentinel, and the class's count and homes move with it
in `evals/coldreading_fixture_test.go`. `TestPriorRunExhaustNeverReaches`
then covers ac-8 at every assembling position with no new assertion. This is
a regression guard rather than a falsifier: A `.json` under
`.abcd/development/readings` matches no include row and sits under a denied
segment, so the plant cannot leak today; it is planted so that a row or a
segment change that would let it leak is caught by class and position.

## How the Acceptance Criteria are satisfied

- **ac-1**: The collector walks the derived allow list, the named run's
  reading records among it, and carries the supplied dispositions text; the
  manifest lists every path passed; `assertAllowList` refuses any other path.
  Tests: `TestScribeContextIsLedgerAndSuppliedTextOnly`,
  `TestScribeContextCarriesTheRunsRecordsFromTheStore`,
  `TestScribeManifestNamesEveryPathPassed`, `TestAssertAllowListFailsClosed`.
- **ac-2**: A supplied disposition is written through `capture.Disposition`,
  and the ledger differs afterwards by that one file. Test:
  `TestScribeIngestWritesASuppliedDisposition`.
- **ac-3**: An unknown key is refused by name at decode; a ground absent from
  the supplied text is refused naming `grounds` and the item, on the
  disposition and on the admission alike. Tests:
  `TestScribeIngestRefusesAnAuthoredField`,
  `TestScribeIngestRefusesAnUnsuppliedGround`,
  `TestScribeIngestRefusesAnUnsuppliedAdmissionGround`.
- **ac-4**: An item listed in `outstanding` is reported and no record is
  written for it. Test: `TestScribeIngestReportsOutstandingAndWritesNothing`.
- **ac-5**: A transcript captured with both stamps of one run carries them in
  `context_stamps`, and `SessionSeparation` names it with the run. Test:
  `TestSeparationNamesATranscriptCarryingBothStampsOfOneRun`.
- **ac-6**: Two transcripts, one stamp each, produce no violation, and the
  report says that no retained transcript carries two stamps of one run and
  lists the run it saw. Test: `TestSeparationReportsNoTranscriptCarryingTwoStamps`.
- **ac-7**: An empty store reports `Unobserved` true with the reason. Test:
  `TestSeparationReportsAnEmptyStoreAsUnobserved`.
- **ac-8**: The planted `scribe-manifest.json` reaches no bundle. Test:
  `TestPriorRunExhaustNeverReaches` with the extended plant, kept as a
  regression guard.

## Tests

Watched fail before, pass after; each refusal proved by a mutation that removes
it.

- `internal/core/scribe/assemble_test.go`:
  `TestScribeContextIsLedgerAndSuppliedTextOnly`,
  `TestScribeContextCarriesTheRunsRecordsFromTheStore`,
  `TestScribeManifestNamesEveryPathPassed`, `TestAssertAllowListFailsClosed`
  (an item under `.abcd/development` injected through the seam is refused),
  `TestAllowListIsDerivedFromLedgerDirs`,
  `TestScribeAssembleRefusesAnUncommittedRun`,
  `TestScribeContextCarriesThePerRunStamp`,
  `TestScribeAssembleWritesNothingBesideTheRun`,
  `TestScribeContextCarriesNoHomePath`.
- `internal/core/scribe/ingest_test.go`:
  `TestScribeIngestWritesASuppliedDisposition`,
  `TestScribeIngestRefusesAnAuthoredField`,
  `TestScribeIngestRefusesAnUnsuppliedGround`,
  `TestScribeIngestRefusesAnUnsuppliedAdmissionGround`,
  `TestScribeIngestRefusesAnUnsuppliedDisposition`,
  `TestScribeIngestReportsOutstandingAndWritesNothing`,
  `TestScribeIngestRefusesASilentItem`,
  `TestScribeIngestProvesTheContextHash`,
  `TestScribeIngestCarriesFidelityFlagsIntoNoRecord`,
  `TestScribeIngestNamesWhatLandedBeforeARefusal`,
  `TestScribeIngestRefusesBeforeTheComparativeRun` (the inherited gate, with
  nothing landed),
  `TestScribeIngestPromotesTheManifestLast` (a refused ingest leaves it
  parked; a completed one lands it write-once).
- `internal/core/sessionkind/sessionkind_test.go`:
  `TestStampsArePerRunAndMatchedExactly`, `TestStampReRecognisesOnlyAStamp`.
- `internal/core/reading/manifest_test.go`:
  `TestBundleCarriesTheReadingStampOfItsRun`, and
  `TestNoBundleFieldIsAScopeSelector` covering `ContextStamp`;
  `internal/core/reading/size_test.go`: `TestBundleGainsNoFieldFromTheReport`
  with the bundle allow-set extended.
- `internal/core/history/separation_test.go`:
  `TestCaptureRecordsContextStamps`,
  `TestARecordWithoutContextStampsStillParses`,
  `TestSeparationNamesATranscriptCarryingBothStampsOfOneRun`,
  `TestSeparationIgnoresTwoStampsOfTwoRuns`,
  `TestSeparationReportsNoTranscriptCarryingTwoStamps`,
  `TestSeparationReportsAnEmptyStoreAsUnobserved`,
  `TestSeparationReadsMetadataOnly` (the bodies are unreadable under the test
  and the report is unchanged).
- `internal/surface/cli/scribe_surface_test.go`: `TestScribeOperandsArePinned`
  (on the `readingOperands` idiom), `TestScribeAssembleEchoesTheOperatorsOut`,
  `TestScribeIngestRendersOnRefusal`.
- `internal/core/lint/scribecontract_test.go`: The existing cases over the
  moved definition, `TestScribeInputsMatchTheLedgerDirs` and
  `TestScribeDeliveryNamesTheVerb`.
- `evals/coldreading_fixture_test.go` and `evals/smoke_test.go`: the `EXHAUST`
  plant and the `history separation` read-only case.

## Out of scope

- Any judgement by the verbs about the content transcribed: A state, a
  ground, an exit condition or a resolution the material does not carry is
  refused, never supplied.
- The scribe's access rule, which stays the definition's; the verb enforces
  it and derives its enumeration of the ledger from the schema's constants.
- Reading transcript bodies from the check. The stamp is metadata at capture,
  and the check stays inside the consumer invariant 15 enumerates.
- The ordering gate and the one-ground rule, which spc-2609020626040342 builds
  in the functions this verb writes through.
- The durable-tier writer, which spc-2609020626039834 exports.
- The ADR and the invariant amendment, both adopted on 2026-09-02 (adr-2609021016275803,
  `status: accepted`); this spec builds on them and does not restate them.
