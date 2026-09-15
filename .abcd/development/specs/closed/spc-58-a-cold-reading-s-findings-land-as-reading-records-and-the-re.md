---
id: spc-58
slug: a-cold-reading-s-findings-land-as-reading-records-and-the-re
intent: itd-180
---
# Reading records and disposition records: two acts, two writes

## Summary

spc-58 delivers [itd-180](../../intents/shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md)'s
two record types. A **reading record** is what an instrument returned under a
recorded visible world; a **disposition record** is the researcher's answer to
one such item, written separately and keyed to it. The two are never one write,
so the record can always show that a finding existed before it was answered.

Three packages move. `internal/core/issueschema` gains both schemas as data,
the position-typed body sets, and the reserved hold axes.
`internal/core/capture` refuses a malformed record of either kind and refuses to
promote an undispositioned item. `internal/core/lint` gains a report-only
outstanding report, teaches `record_schema` the new stores, and the RS gates
learn to scope to the three issue status directories rather than to the whole
ledger tree.

The policy layer is settled in `.abcd/work/DECISIONS.md` and is not this spec's
to reopen: rulings (2), (8), (9), (17) and (18), and ruling (19)'s slate,
`disposition_grounds` and position validation. This spec settles what they left
to the build.

## Scope

In: the shared status-directory list, consolidated before anything is added to
it; the reading-record envelope and its four bodies; the disposition record and
its availability rule; the reserved hold axes and the reserved surprise shape;
the on-disk layout and the three new record families; capture's refusals and the
undispositioned-promote refusal; the outstanding report; the RS-gate and
`record_schema` scoping. Out: everything under `## Out of scope`.

## Approach

### One canonical status list, before a fourth folder exists

The issue ledger's status directories are enumerated three times today:
`issueStatusDirs` in `internal/core/lint/lint.go`, and two `[]string{"open",
"resolved", "wontfix"}` literals in `internal/core/capture/alloc.go`
(`ensureLedgerDirs` and `issPresent`). A change that adds sibling directories to
that tree must not add a fourth spelling, so the consolidation lands **first**,
as its own commit.

The canonical home is `internal/core/issueschema`: already the one place both
`core/capture` and `core/lint` read the ledger's schema from, and a leaf that
imports only the standard library, so neither side gains an import. It gains
`StatusDirs`; lint's `issueStatusDirs` becomes that value, capture's two literals
iterate it, and `statusDirs` / `statusDirName` in `capture.go` are built from it
rather than restated, so the `State` projection and the directory names cannot
disagree. The shell gates cannot import Go, so
`scripts/check-issue-resolution.sh` holds the second and last spelling, and a Go
test reads its declared array and asserts equality with `issueschema.StatusDirs`.

### The layout, and why status is a directory probe

- `.abcd/work/issues/readings/<run-id>/rdi-<N>.md` : one reading record per item.
- `.abcd/work/issues/dispositions/<item-id>/dsp-<N>.md` : the dispositions of one
  item, in a directory **keyed by the item identifier**.
- `.abcd/development/readings/<run-id>/rdg-<N>.md` plus `manifest.json` : the
  durable run record and its manifest (ruling (9)).

Keying the directory on the item and the file on the disposition's own id
settles two requirements that pull against each other. The status signal is the
presence of the keyed directory, which is one probe and never a folder-membership
question. And a disposition still has an id of its own, so the only exit from a
`held` state, a superseding disposition that **cites** the one it replaces, has
something to cite: `supersedes_disposition: dsp-<N>`. The standing disposition of
an item is the one no sibling supersedes; the superseded records stay in place,
because a hold that vanished when it was answered would take its own exit
condition with it
([adr-3](../../decisions/adrs/0003-directory-as-truth-for-lifecycle.md): the
location is the state, and git is the history).

### Three families, minted, never derived

`rdg` (run), `rdi` (reading item) and `dsp` (disposition) mint through
`recordid.Minter.Mint`, which validates any lowercase family and consults no
maximum ([adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md)).
Two runs returning the same tension therefore carry different ids **for free**:
the id is a UTC stamp plus four random digits, never content-derived, so a
re-raise stays distinguishable from its first appearance and the recurrence
signal survives. No new mint is written for any of the three.

### One record type, four bodies, held as data

The envelope every reading record carries is `schema_version`, `id`, `run`,
`manifest`, `position`, `regime` and `pattern`. The pattern named sits in the
envelope, never in a body (ruling (18)): a universal core condition must not
live in a variant part.

itd-180 offers a discriminated union and names a fallback. **The fallback is
chosen**: one record type, an untyped body, and a per-position required-field
set held as data in `issueschema.ReadingBodyFields`. Go has no discriminated
union, and this package's whole idiom is already "the schema is a value both the
writer and the gate read"; four Go types would be four places for the schema to
drift. The four sets are registrative (`tension`, `constraint_in_play`,
`why_a_tension`), generative (`configuration`, `what_admits_it`), explicative
(`claim_surfaced`, `claim_type`, `what_implies_it`) and evaluative
(`candidate_id`, `criterion`, `characterisation`).

### The disposition vocabulary, and one count the record must settle

The disposition carries `id`, `item`, `state`, `disposition_grounds`, an
`exit_condition` when the state is `held`, an optional `supersedes_disposition`,
and an optional `recurs` list of prior item ids. The `recurs` citation is the
recorded form of the researcher's warm recognition of a persistence: it lives
entirely on the ledger side, is never a fourth state, and no mechanical join
produces it.

Four states ship: `accepted` (all positions; at the widening position acceptance
IS admission), `rejected` (explicative, evaluative and registrative only),
`declined` (widening only), `held` (exit condition required). itd-180 and ruling
(19) both say *five states* and both enumerate four. The schema is data, so a
fifth is one line the day it is named, and naming it is a vocabulary judgement
belonging to the researcher rather than to this build: the shipped enum is the
four named states, and the discrepancy is carried, not resolved here.
`disposition_grounds` is free text, required on every state except `held`, and
what it must contain varies by state, enforced by lint rather than by four
fields. Nothing meaning "already covered" exists at any position: an
undispositioned item is reported as outstanding, never named as a state.

### The refusing gate

Capture refuses, rather than accepting and flagging, in every case below. Each
refusal names the rule it enforces:

- an unknown property, or a body field absent for the record's own position;
- a `disposition_grounds` that is empty, on every state except `held`;
- a `held` disposition whose `exit_condition` is empty;
- a state unavailable at the item's position. The disposition reads
  `position` from the keyed reading record, so an orphan disposition (no such
  item) is refused by the same path;
- a populated `hold_frame_location` or `hold_moscow`. The two-axis hold field is
  reserved and dormant: the value grammars are stated (frame-location is free
  text naming the frame element; MoSCoW is `must` / `should` / `could` / `wont`)
  and a populated value is refused until activation is ruled, so the reservation
  is a behaviour rather than a comment.

`capture promote` gains the same posture. It accepts an `rdi-N` alongside an
`iss-N`; for an `rdi-N` it reads the item's STANDING disposition before anything
is minted and refuses every state but `accepted`, naming the rule the refusal
enforces. An absent disposition directory is the first of those refusals and the
one itd-180 states — it collapses the answer and the action into one act, so
nothing can show the finding was weighed before it was acted on — and the same
reasoning carries the rest: a `rejected` or `declined` item would put a refusal
and the admission it refused in one ledger, and a `held` one would settle by
action exactly what the hold left open. Where the answer needs to change, a
superseding disposition is what changes it. The standing state is read again
inside the write's own lock, so a disposition landing between the two cannot
leave a refusal beside a stamp. On success it stamps `promoted_to` on the reading
record and `promoted_from` in the minted draft, the forward-and-back join
itd-180's routing rule requires. Circumventing it is a lapse-log entry, not a
gate.

### The outstanding report is a report

`internal/core/lint` gains `reading_outstanding`: for each run directory, every
`rdi-*.md` with no keyed disposition is reported, and every open `held`
disposition is rendered with its exit condition. Its severity is pinned to
`info` **in code**, not read from `RuleConfig` : a rule whose severity a config
could raise to blocker is a gate waiting to happen, and a reading must never
block an unrelated push. It surfaces on `abcd lint`'s findings and on the bare
`abcd capture` status board beside the skipped roster.

### The gates learn the tree's new shape

`record_schema` reports an undeclared directory under a store root, so the two
new folders would each be a blocker on the day they appear. Three changes, all
of them general rather than special cases:

1. A directory that is itself a configured record-store root is not an
   undeclared bucket. The set is derived from `cfg.RecordStores`, so the config
   stays the single declaration of what a store is.
2. `recordStore` gains `bucketRe`. A store whose buckets are **minted** rather
   than enumerated declares them by grammar, which is what a run-id and an
   item-id directory both are.
3. Three store entries join `recordStores`, each bucketed by grammar: `rdi` at
   `.abcd/work/issues/readings` and `rdg` at `.abcd/development/readings` (buckets
   `^rdg-[0-9]+$`), and `dsp` at `.abcd/work/issues/dispositions` (buckets
   `^rdi-[0-9]+$`). `manifest.json` is not markdown, so the existing scan skips
   it untouched.

`scripts/check-issue-resolution.sh` replaces its single `$ISSUES_DIR` pathspec
with the three status-directory pathspecs. RS001 already matched only the
terminal folders; RS002 and RS003 did not, and would have read a
`resolved_by.commit`-shaped line out of a reading or a disposition.

## Acceptance criteria mapping

| itd-180 criterion | How spc-58 satisfies it | Test |
|---|---|---|
| Validated output ingested: one reading record per item, each run-scoped | The ingest writes one `rdi-N.md` per item under `readings/<run-id>/`, each id minted by `recordid.Minter.Mint("rdi")` and each envelope carrying `run` | `TestIngestWritesOneRecordPerItem` |
| Two runs, same tension, different identifiers | Ids are timestamp-plus-entropy and never content-derived (adr-45), so identity is the write, not the text | `TestTwoRunsSameTensionMintDistinctIDs` |
| Empty `disposition_grounds`, or a hold with an empty exit condition: refuse | The refusing gate above, in capture's strict validation | `TestDispositionRefusesEmptyGrounds`, `TestHeldDispositionRefusesEmptyExitCondition` |
| A state unavailable at the item's position: refuse and name the rule | The disposition reads `position` off the keyed reading record and checks the availability table; the error quotes the rule | `TestDispositionRefusesStateUnavailableAtPosition` |
| Every reading record either carries a disposition or is reported as outstanding | The `reading_outstanding` report, pinned at `info` | `TestOutstandingReportNamesUndispositionedItems`, `TestOutstandingReportSeverityIsInfoNotBlocker` |

## Tests

Every case below is watched to fail before its change lands.

- `internal/core/issueschema/statusdirs_test.go` :
  `TestStatusDirsAreTheOneCanonicalList` asserts capture's and lint's lists are
  the shared value and that the new folders are absent from it.
- `internal/core/lint/preflightgates_test.go` :
  `TestIssueResolutionGateScopesToStatusDirs` reads the shell array out of
  `scripts/check-issue-resolution.sh` and asserts equality with
  `issueschema.StatusDirs`.
- `internal/core/capture/reading_test.go` : `TestReadingRecordRefusesUnknownProperty`,
  `TestReadingRecordRequiresPositionBodyFields` (one sub-case per position),
  `TestDispositionRefusesEmptyGrounds`, `TestHeldDispositionRefusesEmptyExitCondition`,
  `TestDispositionRefusesStateUnavailableAtPosition`, `TestDispositionRefusesOrphanItem`,
  `TestPopulatedHoldAxisRefused`.
- `internal/core/capture/ingest_reading_test.go` : `TestIngestWritesOneRecordPerItem`,
  `TestTwoRunsSameTensionMintDistinctIDs`, `TestSecondDispositionForOneItemRequiresSupersedes`.
- `internal/core/capture/promote_test.go` : `TestPromoteRefusesUndispositionedReadingItem`,
  `TestPromoteStampsReadingItemPromotedTo`.
- `internal/core/lint/reading_outstanding_test.go` : `TestOutstandingReportNamesUndispositionedItems`,
  `TestOutstandingReportSeverityIsInfoNotBlocker`, `TestOpenHoldRendersItsExitCondition`.
- `internal/core/lint/schema_test.go` : `TestNestedStoreRootIsNotAnUndeclaredBucket`,
  `TestReadingRunBucketDeclaredByGrammar`, `TestManifestJSONIsSkippedByTheRecordScan`.
- `scripts/check-issue-resolution-cases.sh` gains `reading-record-ignored` and
  `disposition-ignored`, each a fixture repository the gate must pass.

## Out of scope

- The output contract that "validated" refers to. It is owned by the
  cold-reading output-contract intent; this spec consumes it and adds no second
  validation path.
- The assembler and its include table, and passing a disposition to a reading.
  Dispositions are warm by definition: the exclusion is the assembler's to make
  and the read-block eval's to assert.
- Populating the surprise entry, reserved here (`occasioned_by`, keyed to
  whatever occasioned it) and populated in Iteration 2. The admission-side
  records are spc-67's.
- Activating the two-axis hold field. Reserved, grammar stated, populated value
  refused.
- Whether `held` is available at the widening position. Deferred by the
  facilitator on 2026-08-30 with a revisit point at the first widening run's
  dispositions: the availability table ships with that row unfilled, so the
  refusal for that one combination is not armed.
- Dispatching `abcd <id>` on the new families. `recordid.CitedIDRe` covers the
  four id-bearing families only, and widening it ripples across every citation
  surface.
- Reusing `open` / `resolved` / `wontfix` for reading items, which itd-180
  rejects on the merits.
