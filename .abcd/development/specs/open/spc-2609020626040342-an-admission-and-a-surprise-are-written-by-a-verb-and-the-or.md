---
id: spc-2609020626040342
slug: an-admission-and-a-surprise-are-written-by-a-verb-and-the-or
intent: itd-2609020625400194
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The admission and surprise verbs: one act, two records, and the ruled order held as a refusal

## Summary

spc-2609020626040342 delivers
[itd-2609020625400194](../../intents/planned/itd-2609020625400194-an-admission-and-a-surprise-are-written-by-a-verb-and-the-or.md).
`abcd capture admit <rdi-N> --grounds "<text>"` records an admission as one
act under the ledger lock: The item's `accepted` disposition carrying the
grounds, and the admission record joining it to its run's candidate set. Where
an `accepted` disposition already stands, the admission record is written
alone, and only on the same ground. It refuses an item that is not a widening
item, an item already admitted, a standing disposition in any other state, and
a ground below the substance floor. The ruled order, characterise first and
admit second, is held as one gate in the shared disposition writer: At the
widening position no disposition in any state is written until a committed
comparative run names the item's run, so `capture disposition`, `capture
admit` and the scribe's ingest all refuse the same way and name what they are
waiting for. `abcd capture surprise --occasioned-by <id> "<text>"` writes one
surprise entry as its own record and refuses an occasion that does not resolve
to a reading item, an admission or a disposition. `abcd adm-N` and `abcd
srp-N` dispatch, and the outstanding report gains a per-run summary so the
admitted-against-declined count is a query.

The flagged decision is built under its flagged reading: Admission is the
`accepted` disposition plus the admission record, written together, which
refines
[itd-180](../../intents/shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md)
and closes the state
[spc-67](../closed/spc-67-what-the-widening-reading-proposes-is-admitted-or-declined-o.md)
left, where an `accepted` item was still reported unadmitted.

## Scope

In: The two verbs, their core functions in `internal/core/capture`, their
refusals, and their flags; the shared disposition writer, the ordering gate
in it and the probe the gate reads; the one-lock write of two records, and the
mint of `disposition` moved inside the lock; the dispatcher's three new
families and the plugin page's dispatch sentence; the closed form of a
surprise's occasion, in the schema and in the lint join, and the retired
reservation on the reading envelope; the per-run summary in the outstanding
report and on the capture board; the plugin page and the brief chapter.

Out: The scanner fix for absence as a class
([iss-2608301808198621](../../../work/issues/resolved/iss-2608301808198621-isabsentvalue-decides-on-literal-strings-rather-than-the-yam.md),
[iss-2608301744268001](../../../work/issues/resolved/iss-2608301744268001-a-trailing-comment-on-a-frontmatter-key-defeats-every-blank.md));
the disposition vocabulary; the comparative channel and the committed
comparative run it produces, which
[spc-2609020626039834](../closed/spc-2609020626039834-a-comparative-reading-receives-the-widening-run-s-items-as-i.md)
defines and this gate only reads; the reading-item locator leaf, which
spc-2609020626046252 introduces and this spec calls; enforcing that a session
ended.

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

This spec moves no assembler, bundle or preset version: It touches neither the
include table nor the bundle shape. It lands second in Phase B, after the
condition disposition, whose `readingitem` leaf its wrappers resolve through,
and after the comparative channel in Phase A, because the gate reads that
channel's probe; it lands before the scribe because the scribe's ingest
writes through the functions it adds. Between Phase A and this spec the
maintainer's widening reading has run and the comparative reading has
characterised its candidates, so the first admissions are hand-authored in
the target format, with the gate this spec builds applied by hand to them,
until the verb lands.

### One act, two records, one lock

`internal/core/capture/admit.go` adds `Admit(req AdmitRequest) (AdmitResult,
error)` with `AdmitRequest{RepoRoot, IssuesRoot, Item, Grounds string}` and
`AdmitResult{Admission, Disposition, Item, Run, Path, DispositionPath string;
Redacted int; Degraded string}`. It follows `Disposition` and
`promoteReadingItem` in `internal/core/capture/reading.go` and `promote.go`:
`resolveRoots`, `recordid.ValidReadingItemID`, `mutationPreamble`, a
pre-flight read of the item through `findReadingItem`, `readRecordGuarded` and
`validateReadingStrict`, then everything that decides anything under
`withLedgerLock`, re-read there rather than trusted from the pre-flight.
`findReadingItem` and `readingItemPaths` are, from spc-2609020626046252
onward, thin wrappers over `readingitem.Locate` and `readingitem.Paths` in the
`internal/core/readingitem` leaf, kept in `capture` until every caller moves;
this spec calls the wrappers and adds no caller of its own to move later.

Under the lock, in order: The item's `position` must be `widening`, refused
otherwise by name, because admission is that position's warm act alone.
`ItemFate(repoRoot, run, item)`, the one admitted-proposal probe, which
spc-2609020626039834 adds to `capture` with its judgement in `issueschema`, is
asked once: An admission under `admissions/<run>/` whose `proposal` names the
item, keyed on the (run, proposal) pair exactly as `admittedProposals` in
`internal/core/lint/readingoutstanding.go` keys it, refuses naming it. The
same probe carries the standing dispositions: None means both records are
written; exactly one in the `accepted` state means the admission alone, and
on that branch the supplied ground, after `grounds.Fold`, must equal the
standing disposition's `disposition_grounds`, refused otherwise naming both
texts, so the one-ground rule below holds whichever verb reaches this branch;
any other state refuses naming the disposition and its state; a contested or
cyclic set refuses as `Disposition` already does. Then the shared writer's
gate below. Only then are the ids minted, through `minter.Mint` for
`issueschema.DispositionFamily` and `issueschema.AdmissionFamily`, each
checked by `refuseExistingRecord`, so the mint sees the tree it writes into.

The disposition is written first, the admission second, and a failure on the
second removes the first, on the both-or-neither shape `writePair` in
`internal/core/reading/assemble.go` carries. The admission is assembled by
`admissionFields` as `schema_version`, `id`, `run`, `proposal`, `grounds`, in
that order, validated by `validateAdmissionStrict` against
`issueschema.AdmissionRequired` and `AdmissionKnown`, serialised by
`buildIssueText`, and written to `admissions/<run>/adm-N.md` after
`ensureFamilyDir`. `recordid` gains `ValidAdmissionID` and `ValidSurpriseID`
beside the four grammars it holds, so no path is built from an id nothing has
matched.

### The shared writer and the ordering gate

The disposition write is the body `Disposition` already runs under its lock,
factored into `writeDispositionLocked` so that every disposition in this
binary passes one validator path (`dispositionFields`,
`validateDispositionStrict` against the item's position) and one gate.
`Disposition` calls the same function, and its mint moves inside the lock:
Today `Disposition` mints its `dsp-N` before `withLedgerLock`, and after this
change it mints under the lock as `mintUnusedItemID` in `IngestReading`
already does. That is a behaviour change to `capture disposition`, not only to
`admit`, and `TestDispositionMintsUnderTheLock` records it.

The gate is keyed on the item's position alone. At the widening position,
`writeDispositionLocked` refuses a disposition in any state the position
admits, `accepted`, `declined` or `held`, until a committed comparative run
names the item's run, and names what it is waiting for: A committed
comparative run whose manifest names `rdg-N`. The probe is
`capture.ComparativeRunFor(repoRoot, run)`, which spc-2609020626039834 adds:
It walks `.abcd/development/readings/*/run.json` and returns the comparative
run whose `candidate_run` names the widening run, whether that run
characterised two or more candidates or was committed with an empty item set
because the position was not exercised. Nothing else satisfies the gate, and
no mutable file anywhere records the outcome. Because the gate sits in the
writer, `capture disposition`, `capture admit` and spc-2609020626045177's
`scribe ingest` all hold the ruled order without each carrying a check, and
the assembler's own refusal of a dispositioned candidate becomes unreachable
through any verb: It stays as a guard against a hand-placed record, which the
scanner fix named in the residue is what catches as a class. There is no
deadlock, because the comparative run the gate waits for is exactly the run
the assembler produces over the undispositioned set.

`held` is available at the widening position, and the ground is stated here
because the readings companion leaves it open: Its section 4.3 sends the
question to section 11, and section 11 does not settle it. The ground is that
a proposed configuration can be real and not yet articulable, with an
epistemic exit, exactly as a tension can: The researcher can see that the
construal admits it without yet being able to say whether the record should,
and `held` with its `exit_condition` records that state as the disposition
vocabulary already does for a detection tension. Routing such a candidate
through the selection vocabulary's `deferred` instead would move it onto the
selection surface before the comparative reading has characterised it, and
the companion's section 8.3 orders the two the other way, characterisation
first and admission or decline with grounds after. So `held` is the widening
position's third state and is written through the same gate, after the
comparative run and never before. The choice is registered as an
interpretation of a section the document leaves open, not as a reading of it.

The scope condition holds by construction: The gate reads the comparative run
the channel commits, and the channel lands in Phase A before this verb, so by
the time the gate exists every widening run it is asked about either has a
committed comparative run naming it or has none, and the refusal is not a
stub but the honest state of a run nobody has characterised.

### The grounds

`--grounds` on `admit` is free text without a `<token>:` prefix. The
admission's `grounds` is free text by spc-67's schema and the disposition's
`disposition_grounds` is free text by spc-58's, and `AdmissionKnown` is a
closed set with no room for a token field, so the grammar `requireGrounds`
applies to the triage routes does not fit here. What is shared is the floor:
The text is redacted through `redactLedgerText`, folded by `grounds.Fold`, and
held to `grounds.ValidateText`, which refuses the empty, whitespace,
control-character, too-short and vocabulary-only texts. The same folded text
lands in both records, and the admission-alone branch requires it to equal the
standing disposition's, so the disposition and the admission cannot state two
reasons for one act by any path. The refusal writes nothing and says so.

### The surprise verb

`internal/core/capture/surprise.go` adds `Surprise(req SurpriseRequest)` with
`SurpriseRequest{RepoRoot, IssuesRoot, OccasionedBy, Text string}` and
`SurpriseResult{ID, OccasionedBy, Path string; Redacted int; Degraded string}`.
The occasion is resolved before anything is minted through
`readingitem.ResolveOccasion(issuesRoot, id, families)` with the families
`rdi`, `adm` and `dsp`, the one occasion resolver spc-2609020626046252
introduces and spc-2609020626048705 also calls; an id outside those families,
or one that resolves to nothing, refuses naming it. The text is redacted
through `redactLedgerText`, folded by `grounds.Fold` and held to
`grounds.ValidateText`, the same floor every grounds-shaped field in this
workstream applies. The record carries `schema_version`, `id` and
`occasioned_by` in its frontmatter and the text as its body, because spc-67
places the surprise itself in the body, and is written to `surprises/srp-N.md`
under the ledger lock after `refuseExistingRecord`. No disposition file is
opened for writing on this path, which is what "never a field on a
disposition" means in code.

The occasion is a closed form. The doc comment on `issueschema.SurpriseRequired`
admits "a consequence named in prose" today, and the `srp` join in
`internal/core/lint/schema.go` declares no `sameBucketAs`, so a prose value
passes the record gate. This spec closes it: `occasioned_by` is an `rdi-N`,
`adm-N` or `dsp-N` handle and nothing else, `issueschema` says so, and the
join gains a resolution over that family set on the pattern `sameBucketAs`
uses for one family, so a hand-written prose occasion is a lint finding. The
reserved `occasioned_by` on the reading envelope, `ReservedSurpriseFields` in
`internal/core/issueschema/reading.go`, is retired in the same change: The
join lives on the surprise record and nowhere else, and the key on a reading
record is refused as any unknown key is.

### Dispatch

`record.IDRe` in `internal/core/record/record.go` matches `iss`, `itd`, `spc`
and `adr` today and nothing else, so `abcd rdi-N` and `abcd dsp-N` are refused
as unknown commands; spc-67 records that residual. `IDRe` becomes
`^(iss|itd|spc|adr|adm|srp|rfm)-[0-9]+$` in this change, `rfm` included
because spc-2609020626048705 lands after this spec and inherits the edit
rather than making a second one; its `Describe` case is that spec's.
`Describe` gains the `adm` and `srp` cases with the family names `admission`
and `surprise`, and the status column reads `admitted` and `recorded`,
because neither family has a folder to name. An admission's links are `run`,
`proposal`, the proposal's path and the standing disposition's id; a
surprise's are `occasioned_by` and its path. Neither emits a next move.
`commands/abcd.md`'s dispatch section quotes the positional grammar and its
description names the four families; both are edited to the seven. Widening
the dispatcher to `rdi`, `dsp` and `rdg` stays out of scope and is named in
the residue below.

### The outstanding report and the board

`OutstandingReadings` in `internal/core/lint/readingoutstanding.go` already
answers the intent's ac-7 in two legs: An undispositioned widening item with
no admission lands in `Undispositioned`, and an item with a standing
disposition other than `declined` or `held` and no admission lands in
`Unadmitted`. What is missing is the count as a query. The walk gains
`WideningRuns []WideningRun` with `Run`, `Items`, `Admitted`, `Declined` and
`Outstanding []string`, one per run holding a widening item, and
`checkReadingOutstanding` renders one `info` finding per run. The bare
`capture` board in `internal/surface/cli/cli.go` carries the same report
through `lint.ReadReadingOutstanding`, so `abcd capture --json` answers the
count with no second implementation.

### The front doors

`internal/surface/cli/cli.go` registers `admit <rdi-N> --grounds "<text>"`
and `surprise --occasioned-by <id> <text>` under `capture`, on the disposition
sub-verb's shape: No cobra-required flag, the core refusing an empty value,
`--json` inherited, the render sanitised through `termsafe.Sanitize`, and
`redacted` reported whenever it is non-zero. `commands/capture.md` documents
both verbs and the ordering gate and drops its "written by hand" paragraph;
`.abcd/development/brief/04-surfaces/06-capture.md` gains the two sub-verb
rows; `.abcd/development/release/surface.json` is regenerated so
`surface_coverage` holds in both directions.

## How the Acceptance Criteria are satisfied

- **ac-1**: With no disposition and a comparative run committed over the
  item's run, `Admit` writes `dsp-N` in `accepted` and `adm-N` under one lock;
  the second call finds the admission by (run, proposal) and refuses. Test:
  `TestAdmitWritesBothRecordsAndRefusesTwice`.
- **ac-2**: A standing `accepted` disposition takes the admission-only branch,
  and the disposition file's bytes are unchanged. Test:
  `TestAdmitWritesTheAdmissionAloneOverAStandingAcceptance`.
- **ac-3**: With no comparative run naming the item's run,
  `writeDispositionLocked` refuses, naming the run and the committed
  comparative run it waits for, whichever verb reached it. Tests:
  `TestAdmitRefusesBeforeTheComparativeRun`,
  `TestDispositionRefusesBeforeTheComparativeRun`.
- **ac-4**: A standing `declined` disposition refuses naming its id and state;
  `held` refuses the same way. Test: `TestAdmitRefusesAStandingNonAcceptance`.
- **ac-5**: `grounds.ValidateText` over the folded text refuses blank,
  whitespace and degenerate grounds before any mint; the ledger is
  byte-identical afterwards. Test: `TestAdmitRefusesADegenerateGround`.
- **ac-6**: `Surprise` resolves the occasion, writes `surprises/srp-N.md`, and
  opens no disposition for writing; the test hashes the dispositions tree
  before and after. Test: `TestSurpriseIsItsOwnRecord`.
- **ac-7**: The `WideningRuns` summary over a run with one admission, one
  decline, one `held` item and one untouched item names the fourth alone. Test:
  `TestWideningRunSummaryNamesTheOutstandingItem`.
- **ac-8**: `IDRe` and `Describe` accept `adm-N` and `srp-N` and report the
  record and its joins. Tests: `TestDescribeAdmission`,
  `TestDescribeSurprise`.

## Tests

Watched fail before, pass after; each refusal proved by a mutation that removes
it.

- `internal/core/capture/admit_test.go`:
  `TestAdmitWritesBothRecordsAndRefusesTwice`,
  `TestAdmitWritesTheAdmissionAloneOverAStandingAcceptance`,
  `TestAdmissionAloneRequiresTheStandingGround`,
  `TestAdmitRefusesBeforeTheComparativeRun`,
  `TestAdmitProceedsOnAnEmptyComparativeRun`,
  `TestAdmitRefusesAStandingNonAcceptance`,
  `TestAdmitRefusesANonWideningItem`, `TestAdmitRefusesADegenerateGround`,
  `TestAdmitRefusesAContestedItem`,
  `TestAdmitRemovesTheDispositionWhenTheAdmissionWriteFails` (through a
  write hook on the admission write, on the `readingWriteHook` idiom),
  `TestAdmitMintsUnderTheLock` (on the `lockrace_test.go` idiom),
  `TestAdmissionAndDispositionCarryOneGround`.
- `internal/core/capture/surprise_test.go`: `TestSurpriseIsItsOwnRecord`,
  `TestSurpriseRefusesAnUnresolvedOccasion` (an unknown id, a malformed id,
  an id of a fourth family, and each of the three families resolving),
  `TestSurpriseRefusesADegenerateText`, `TestSurpriseRedactsItsBody`.
- `internal/core/capture/reading_test.go`:
  `TestDispositionAndAdmitShareOneWritePath` (the factored function is the one
  `Disposition` calls, proved by the write hook firing on both),
  `TestDispositionRefusesBeforeTheComparativeRun`,
  `TestDispositionMintsUnderTheLock`,
  `TestTheGateIsKeyedOnTheWideningPositionAlone` (an entailment item is
  dispositioned with no comparative run anywhere).
- `internal/core/issueschema/reading_test.go`:
  `TestOccasionedByIsNoLongerReservedOnTheEnvelope`.
- `internal/core/lint/schema_test.go`: `TestSurpriseOccasionMustResolve` (a
  prose value, an unknown handle, and each family resolving).
- `internal/core/recordid/valid_test.go`: `TestAdmissionAndSurpriseIDGrammars`.
- `internal/core/record/record_test.go`: `TestDescribeAdmission`,
  `TestDescribeSurprise`, `TestIDReAdmitsTheThreeNewFamilies`,
  `TestIDReStillRefusesTheReadingFamilies`.
- `internal/core/lint/reading_outstanding_test.go`:
  `TestWideningRunSummaryNamesTheOutstandingItem`,
  `TestWideningRunSummaryStandsDownOnAnUnreadableRun`,
  `TestWideningRunSummaryIsInfoNotBlocker`.
- `internal/surface/cli/cli_test.go`: `TestCaptureAdmitRendersAndRedacts`,
  `TestCaptureSurpriseRequiresAnOccasion`, and the capture board case gaining
  `widening_runs`.
- Evals: No cold-reading eval case of this spec's own. Neither verb touches
  the assembler or the read block; the admission records already sit in the
  eval's `GROUNDS` sentinel class, and the surprise plant arrives with
  spc-2609020626039834's fixture. The smoke lane's command-tree walk covers
  the two sub-verbs' help and flag parsing with no edit.

## Out of scope

- The scanner fix for absence as a class, which is two recorded issues and
  ships in its own change; the verb closes the gap for the records it writes,
  and a hand-written blank stays the gate's to catch.
- A fifth disposition state, or any change to which states the widening
  position admits.
- Dispatching `abcd <id>` on `rdi-N`, `dsp-N` or `rdg-N`, which stays the
  residual spc-67 named, and the `rfm` case of `Describe`, which is
  spc-2609020626048705's.
- The comparative channel and the committed comparative run its assembly and
  ingest produce, which the gate reads and never writes.
- The `internal/core/readingitem` leaf and the move of `capture`'s locators
  into it, which spc-2609020626046252 owns.
