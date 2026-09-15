---
id: spc-2609090624222051
slug: sub-agent-transcript-capture
intent: itd-2609090559376002
origin: researcher-authored
production_mode: hand-written
---
# A session's sub-agents are captured through the completion event, with lineage the record carries in fields

## Summary

This spec delivers
[itd-2609090559376002](../../intents/shipped/itd-2609090559376002-sub-agent-transcript-capture.md):
the transcript corpus stops keeping the smaller half of the work.

Two things land. **Capture** gains a `SubagentStop` hook that stages the
finished sub-agent's own transcript through the staging path
[spc-4](../closed/spc-4-start-the-transcript-clock.md) built, so redaction and
its fail-closed refusal are reused rather than reimplemented. **The record
schema** gains explicit lineage fields, so a sub-agent record says which session
spawned it, which agent spawned it, what kind of agent it was and which rung
placed it, instead of smuggling that into a hand-made composite `session_id`.

Nothing here redesigns the store. The per-repo root-SHA keying of
[adr-29](../../decisions/adrs/0029-native-transcript-corpus.md) stands, the
two-stage redaction stands, and `Capture` remains the only path that writes a
record.

Two capabilities that were planned inside this spec are separately delivered
and separately specced, because the landing order delivered them as independent
steps: recovering transcripts already on disk and repairing the composite
records is
[itd-2609091718566731](../../intents/shipped/itd-2609091718566731-transcripts-already-on-disk-are-recovered-into-the-right-rep.md),
and rendering a session as one artefact with telemetry is
[itd-2609091718595846](../../intents/shipped/itd-2609091718595846-any-captured-session-can-be-handed-to-an-agent-as-one-self-c.md).
Both consume the record schema and the fail-closed `Capture` this spec settles,
so this spec is where that material stays.

## Scope

In: `internal/core/history` — the record schema and `CaptureMeta`, `Capture`,
`Stage`, `Drain`, `Read`, `ListForSession`, the staging sidecar, the staged TTL
and quarantine, and the session/gap notes in `locate.go`. The `hook` sub-tree in
`internal/surface/cli` (`hook_subagent.go`) and `hooks/hooks.json`. The plugin
page `commands/history.md`, the brief chapter
[`04-surfaces/11-history.md`](../../brief/04-surfaces/11-history.md), the
regenerated surface snapshot and the generated command reference.

Out: the scanner and its detectors; the store's keying and provisioning; the
harness's retention policy; ingest and migrate; reconstruction and telemetry;
structured extraction of findings from a transcript.

## Approach

### The lineage fields, and why the composite id goes

A sub-agent record has to answer three questions a session record never had to:
which session spawned it, which agent inside that session spawned it, and what
kind of agent it was. The workaround already in the store answers the first two
by concatenating them into `session_id`, in the shape
`<truncated-parent-prefix>--agent-<agent-id>`. That fails in both directions.
A reader holding the identifier the harness actually gives them (the full
session id) cannot find the record, because the stored value is a truncation of
it; and a reader holding the record cannot recover the full parent id from it,
because the truncation is lossy. Overloading one field with two identifiers is
the defect, so the fix is fields.

`Record` gains, all optional and all empty on a main-thread record:

- `agent_id`: the sub-agent's own identifier, as the harness reports it.
- `parent_agent_id`: the identifier of the agent that spawned it, empty when the
  main thread spawned it.
- `agent_type`: the kind of agent, as the harness reports it.
- `spawn_depth`: 0 for the main thread, 1 for a sub-agent of it, and so on.
- `spawn_tool_use_id`: the identifier of the tool call in the spawning
  transcript that launched this agent, when it can be established.
- `lineage_source`: `hook`, `ingest` or `migrated`, so an empty `agent_type` on
  a sub-agent record is distinguishable from an agent type that was never
  recoverable.
- `spawn_attribution`: which rung of the attribution ladder placed the spawn
  point — `sidecar`, `transcript` or `unattributed`. Required on a sub-agent
  record and empty on the main thread, because "spawned by the main thread" and
  "spawn point unknown" would otherwise be the same empty fields.

`session_id` keeps its existing meaning unchanged: the session the transcript
belongs to. On a sub-agent record it holds the full, untruncated id of the
spawning session, which is what the `SubagentStop` payload delivers.

`source_kind` is deliberately NOT extended with a sub-agent value. Kind says
where the bytes came from (`native`, `specstory-import`), not what produced
them; encoding agent-ness there would repeat the overloading this section
removes.

The schema version moves forward. A schema-1 record parses as a main-thread
record with empty lineage, so `List` and `Read` keep working over everything
already stored, and the composite records stay readable until the repair verb
runs.

Three consequential changes follow:

- **The idempotency key becomes `(source_sha256, session_id, agent_id, kind)`.**
  Without `agent_id`, two sub-agents of one session that happened to produce
  byte-identical transcripts would collapse into one record, which is the
  precise failure the existing key's session component was added to prevent.
- **`recordFilename` becomes `<stamp>-<session-id>.md` for a main-thread record
  and `<stamp>-<session-id>-agent-<agent-id>.md` for a sub-agent's.** The
  filename is readable convenience only. `listRecords` parses frontmatter and
  never the filename, so nothing decodes this string back into fields.
- **`Capture` takes a metadata struct.** `Capture(repoRoot, rootSHA string, raw
  []byte, meta CaptureMeta)` replaces the five-positional signature;
  `CaptureMeta` carries the session id, the lineage fields and the kind. Six
  more positional string arguments on a security-critical call is a
  transposition waiting to happen, and a transposed session and agent id would
  silently mis-attribute a transcript rather than fail.

`Read` resolves its key in three steps: an exact record filename, then an exact
`agent_id`, then a `session_id` (preferring the main-thread record, newest
first). `ListForSession(rootSHA, sessionID)` returns the whole set for one
session, main thread first, and is what `history show` uses to list a session's
sub-agents under the record it rendered.

### The frontmatter is redacted with the body

Every new field is externally supplied: an agent type comes from the harness
payload, an adopted project name from a configuration file a contributor can
edit. `marshalRecord` writes frontmatter, and frontmatter has never been
scanned, so adding externally-supplied scalars to it would open a redaction
bypass beside a redaction gate.

The fix is not a second scan. `Capture` prepends the lineage scalars to the raw
text as ordinary lines before the existing two-stage pass runs, then splits them
back off afterwards. The scalars therefore go through exactly the same
sanitise-then-verify discipline as the body, including the caller-home backstop
and the fail-closed residual refusal, with no second code path to drift. A
scalar whose redaction changed it is stored changed; a blocking residual refuses
the whole write as it does today. A scalar carrying a line break would forge a
frontmatter field, so it is refused at the store and dropped at the hook —
losing one decorative field beats losing the transcript.

### The deferred decision record

The move from a composite identifier to explicit lineage fields refines
adr-29 and deserves an ADR. It is deliberately deferred and unminted here: the
shape above is the first thing implementation can falsify, and a decision record
minted before its own migration has run would record a decision that has never
met the store. The ADR is minted after this work merges, and until then **this
spec is the decision of record**.

One caveat for whoever mints it. The repository's root `AGENTS.md` says ADRs are
the one record family whose ids are hand-numbered and so need coordination
between checkouts; the ADR store's own charter records the ruling of 2026-09-01
that `abcd decide` mints an ADR through the same collision-proof timestamp-numeric
seam as every other family, with the pre-existing ordinals grandfathered. Those
two statements disagree, and the disagreement is not this spec's to settle.
Resolve it before minting, and correct whichever surface is stale.

### `hook subagent-stop`, and its wiring

A new operator-internal verb, `abcd hook subagent-stop`, built to the same shape
as `hook session-end`: fail-closed, diagnostics on stderr, stdout empty. One
difference is not cosmetic. `SessionEnd`'s exit code is ignored by contract;
`SubagentStop`'s is not — exit 2 is that event's BLOCKING status and would stop
the sub-agent from finishing. So every path in the verb returns nil, and no
staging failure is worth risking an agent's completion over. That is a rule about
this binary's own exits and not about the launcher's: the `hooks.json` wrapper's
`exit 1` when no binary resolves is deliberately non-zero and deliberately not 2,
because it is the one signal a user gets that their transcripts are not being
captured.

It reads the hook payload from stdin and uses five of its fields:

- `agent_transcript_path`: the finished sub-agent's own transcript. This is the
  whole reason the gap can be closed without reading the harness's on-disk
  layout, and its absence is the fallback case below.
- `agent_id`, `agent_type`: the lineage the payload carries directly.
- `session_id`: the spawning session, stored untruncated.
- `cwd`: resolved to the repository's root-commit SHA.

It **stages, it does not capture**, for the reason staging exists: redaction runs
at roughly 0.7 s per megabyte, and `SubagentStop` fires inside a live session
where a stall is felt directly. A later drain runs it through the unchanged
`Capture`.

**Resolving the store needed a second rung the plan did not have.** A sub-agent
given its own worktree records that worktree as its `cwd`, and the harness
REMOVES the worktree when the agent stops — so the directory is frequently gone
by the time the hook runs, and the agents that affects are the isolated
implementation lanes, whose transcripts are worth the most. So: the `cwd` first,
exactly as `hook session-end` does, and failing that the spawning session. A hook
that CAN resolve a repository writes a zero-byte note naming the session under
`~/.abcd/history/<sha>/sessions/`, and a later hook holding only that session id
finds the store with one stat per repository and no transcript read. A session id
two stores claim is REFUSED rather than guessed, because a transcript filed
against the wrong repository is redacted by the wrong repository's scanner
configuration. Notes are pruned past thirty days.

A payload with no usable `agent_id` stages nothing: without an agent id the
stage has no key of its own and would collide with the spawning session's own
staged transcript, overwriting the spine to save a branch.

`parent_agent_id`, `spawn_depth` and `spawn_tool_use_id` are not in the payload.
They are resolved down a three-rung ladder, and the rung that answered is
recorded in `spawn_attribution`:

1. **The harness's own sidecar**, derived from the payload's transcript path by
   substituting the extension, never by walking a directory. When it is present
   it carries the spawning agent, the spawn depth and the tool-call identifier
   that launched the agent. The read is guarded and best-effort: absence,
   unreadability, a non-regular file, a parse failure or a missing depth all
   leave the fields empty and are never an error.
2. **The spawning transcript's own tool result** for asynchronously launched
   agents, which names the agent id beside the tool-call id. Resolved at
   reconstruction time, not in the hook — it is
   `itd-2609091718595846`'s to deliver, and it did.
3. **Unattributed.** The record stores the sub-agent with empty spawn fields and
   `spawn_attribution: unattributed`, and a reader lists it separately rather
   than guessing a spawn point.

Wiring: `hooks/hooks.json` gains a `SubagentStop` entry running
`"$CLAUDE_PLUGIN_ROOT/abcd" hook subagent-stop`, using the same binary
resolution preamble as the existing `SessionEnd` entry, and
`applyHookPlaneFailOpen`'s path list gains `{"hook", "subagent-stop"}` so a
usage error on this verb fails open like every other hook path. Without both,
the verb is dead scaffolding.

### Staging carries a sidecar

Staged files were identified by parsing their filename, which is why the
filename had to encode the session. Encoding lineage there would rebuild the
composite-identifier defect one directory earlier.

So `Stage` gains a sidecar. It writes two files, both mode 0o600:

- `<stamp>-<key>.raw`, the raw bytes, where `<key>` is the agent id when there
  is one and the session id otherwise.
- `<stamp>-<key>.stage.json`, holding a schema version, the session id, the
  lineage fields, the source path the bytes were read from, and the stage time.

`listStaged` reads the sidecar when it is present. A `.raw` file with no sidecar
is a staged file from an older binary: its session id is parsed from the
filename exactly as before and it drains as a main-thread transcript, so an
upgrade never strands a backlog. A sidecar that is PRESENT but unreadable is a
different fact and is not drained at all — falling back to the filename there
would read a sub-agent's key as a session id and file the transcript under a
session that does not exist.

Stage idempotency moves from the session id to the `(session_id, agent_id)` pair,
with the existing content compare and the existing last-writer-wins replacement,
all still inside the staging lock.

### The drain's order and its budget

**Order.** `listStaged` is chronological, and a session's main thread is staged
last because it ends last, so a bounded pass would drain the branches and leave
the spine. The drain therefore takes overdue entries first, then main-thread
entries, then sub-agent entries oldest-first. If a pass is truncated, what it
stored is the part that makes the rest legible.

**Budget.** `DrainBudget` is a pair: a byte bound and a count bound, whichever is
reached first. The byte bound is the one that protects the first prompt, because
redaction cost tracks bytes rather than files, and the count bound keeps a
pathological many-tiny-files case bounded too. A pass with nothing yet attempted
never declares itself exhausted, so a single staged transcript larger than the
byte bound cannot be skipped by every pass forever.

### Staged text does not live indefinitely

Staging is the one place abcd holds unredacted transcript text on purpose. The
code claimed it lived "only until the next session starts", and the drain ran
from a hook of the repository the file belongs to — so a repository nobody
opened again kept its raw transcripts for as long as the disk lasted
(`iss-2609090722466403`). Four such files, thirteen megabytes, the oldest a
fortnight old. An intent that admits one staged file per sub-agent completion
makes that ordinary rather than pathological, so four mechanisms bound it:

- **A drain runs while a session is LIVE**, not only at its start: the
  prompt-router hook drains one entry and at most half a megabyte per prompt, so
  a session that spawns sub-agents redacts its own branches as it goes.
  Everything it says goes to stderr, never to the hook's stdout.
- **`StagedTTL` (seven days) is the maximum staged age.** Past it an entry is
  OVERDUE: it sorts to the front of every drain and is named in every notice. Age
  buys priority and volume and nothing else — an overdue transcript is never
  deleted, never redacted down, never degraded. Losing the only copy is worse
  than keeping it, which is the premise staging is built on.
- **Session start reports EVERY repository in the store**, not just the one the
  operator is standing in, and `abcd history staged --all-repos` is the
  read-only verb behind the same survey. A per-repo listing is blind to exactly
  the pile that grows: the one nobody opens. The survey carries counts, sizes and
  repository names — never another repository's session ids.
- **A transcript the fail-closed scanner will never pass is QUARANTINED**, not
  retried forever. A residual refusal is a property of the transcript's own
  bytes, so every future drain reaches the same refusal; such an entry moves to
  `quarantine/` with a written reason and leaves the queue. `DrainFailure`
  therefore distinguishes retryable from permanent, which reporting them alike
  did not (`iss-2609090722466403`). `abcd history discard --yes` is the only
  thing that removes one.

### Failure modes

- **The event fires before the transcript is flushed.** The likeliest way this
  design is shown wrong, and it is **still UNVERIFIED** — the documentation
  suggests it may, the binary does not settle it, and the rate has not been
  measured over a corpus. Two mitigations, both cheap and both self-counting. At
  stage time, `TranscriptSettled` tests whether the final non-blank line is
  complete JSON; a transcript that is not is re-read up to four times at 25 ms
  intervals, and a copy that never settles is staged anyway with a stderr line
  saying it may be short. The wait lives in the surface and deliberately NOT
  inside `Stage`, whose critical section is under a lock tuned for a single
  `SessionEnd`: a wait in there would turn a burst of simultaneous completions
  into a queue. At drain time, the source path recorded in the staging sidecar is
  re-read when it still exists, and its bytes replace the staged copy only when
  they are strictly longer AND the staged bytes are a byte-prefix of them, so a
  recycled path can never substitute a different transcript. `DrainResult.Extended`
  counts the transcripts that were caught short and completed, which is the
  measurement that will eventually settle the question.
- **A session with many sub-agents.** Staging is one write per completion, so
  the hook's cost is independent of the count. The drain is where the count
  lands, which the byte-and-count budget and the drain ordering address. The
  failure this leaves is a backlog of unredacted staged text, which the four
  mechanisms above bound and report.
- **The payload is absent on an older harness.** No `agent_transcript_path`
  means the hook warns on stderr, stages nothing and exits 0. Silence is what
  this intent exists to end, so it also writes a marker beside the staging
  directory recording the first and last sighting and a count, and
  `history staged` reports it: this harness version does not deliver sub-agent
  transcripts, so no sub-agent capture is happening here. Absence of sub-agent
  records then has an explanation on disk instead of looking like an absence of
  sub-agents.
- **An unreadable transcript.** The existing guarded read already refuses a
  symlink, a FIFO, a device node, a non-regular file and an over-cap file, and
  refuses an over-cap file whole rather than truncating it. Reported on stderr,
  nothing staged, exit 0.
- **A degraded scanner.** Unchanged and untouched. Sub-agent capture runs
  through the same `Capture`, so the refusal on a degraded scanner or a surviving
  blocking span applies by construction. The blocking-span half is asserted on the
  sub-agent path; the DEGRADED-SCANNER half is not asserted anywhere, on any path,
  and holds by construction alone. The guard predates this work, so this delivery
  inherits it rather than establishing it, and an unarmed guard is exactly the
  shape this repository refuses to trust elsewhere.

### Surfaces

One operator-internal hook verb, whose plugin surface is its `hooks.json` entry,
as it is for `hook session-end`; plus the `history staged` and `history discard`
changes above. Each needs, in the same change: sections in
`commands/history.md`; rows in the brief's history chapter sub-verb table, which
`surface_coverage` checks in both directions and refuses on; a regenerated
surface snapshot (`cmd/abcd-gen-surface`) and command reference
(`cmd/abcd-gen-cli-ref`).

### Landing order

Each step left the tree green, and each had its own tests.

1. **The record schema.** The lineage fields, `CaptureMeta`, the widened
   idempotency key, the frontmatter-through-the-redaction-pass change, `Read`'s
   three-step resolution and `ListForSession`. Existing callers updated; no new
   verb yet.
2. **Staging's sidecar and the hook.** `.stage.json`, the `(session, agent)`
   idempotency key, the legacy sidecar-less path, the drain's ordering and the
   byte-and-count budget; then `abcd hook subagent-stop`, its `hooks.json` entry,
   its fail-open path registration, the session-note fallback, the harness-sidecar
   rung and the missing-payload marker. This is the point at which the corpus
   starts accruing.
3. **The staged-lifetime bound.** The live drain, `StagedTTL` and the overdue
   ordering, the all-repositories survey, and quarantine.
4. **The measurement**, still outstanding: an instrumented run over real sessions
   counting how many staged sub-agent transcripts arrive truncated.
   `DrainResult.Extended` is the counter; the number goes into the intent's
   grounds when it exists.

## How the Acceptance Criteria are satisfied

- **ac-1 (a sub-agent's transcript is stored, redacted, and listed).** The
  `SubagentStop` hook stages it; the next drain captures it through the unchanged
  fail-closed `Capture`; `List` returns it. Tests:
  `TestHookSubagentStopStagesLineage`,
  `TestHookSubagentStopStagesRatherThanCaptures` and
  `TestSubagentStopThenSessionStartStoresTheRecord`, the last driving the whole
  path end to end.
- **ac-2 (nesting, each attributable to its spawner).** `parent_agent_id` and
  `spawn_depth` come from the attribution ladder and are stored per record.
  Tests: `TestHookSubagentStopReadsTheHarnessSidecar`,
  `TestLineageRoundTripsThroughTheRecord`,
  `TestUnknownSpawnIsDistinguishableFromNoParent` and
  `TestUnreadableSidecarIsReportedNotMisattributed`.
- **ac-3 (found from the spawning session's identifier, with its kind).**
  `session_id` holds the untruncated spawning session, so `ListForSession`
  returns the main thread and every sub-agent, each carrying `agent_type`. Tests:
  `TestListForSessionReturnsMainThreadAndEverySubagent`,
  `TestReadResolvesFilenameThenAgentThenSession`,
  `TestSubagentRecordFilenameNamesTheAgent`.
- **ac-4 (a degraded scanner refuses).** Unchanged `Capture`. The tests below
  cover redaction and a surviving blocking span on the sub-agent path; NONE of
  them degrades a scanner, so that refusal is unasserted:
  `TestLineageFieldsAreRedactedWithTheBody`,
  `TestBlockingSpanInAgentTypeRefusesTheWrite`,
  `TestCaptureRejectsAMalformedLineageScalar`,
  `TestQuarantineHoldsUnredactedTextAtOwnerOnlyModes`.
- **ac-5 (the same transcript twice is a no-op).** The widened idempotency key.
  Tests: `TestSubagentCaptureIdempotentOnSourceSHA` and
  `TestTwoSubagentsWithIdenticalBytesBothStore`, the second bounding the first so
  the key cannot be over-tightened back into the collapsing bug;
  `TestStageIdempotencyIsPerSessionAndAgent` on the staging half.
- **ac-6 (a completion without a readable transcript is reported, not silent and
  not fatal).** The hook's warn-and-exit-0 paths plus the missing-payload marker.
  Tests: `TestHookSubagentStopAlwaysExitsZero` (a table over an absent path, an
  irregular file, an over-cap file, a malformed payload and a payload with no
  transcript path, each asserting exit 0, zero records and a non-empty stderr
  reason), `TestHookSubagentStopMarksAMissingPayloadField`,
  `TestSubagentGapMarker`, `TestSubagentStopNeverBootstraps` (pinning the
  launcher's exit as neither 2 nor 0 nor 127).

## Tests

`internal/core/history`: the schema round-trip over both versions
(`TestSchemaOneRecordReadsAsMainThread`, `TestLineageRoundTripsThroughTheRecord`),
the frontmatter redaction (`TestLineageFieldsAreRedactedWithTheBody`,
`TestBlockingSpanInAgentTypeRefusesTheWrite`,
`TestCaptureRejectsAMalformedLineageScalar`), the idempotency pair above plus
`TestDivergentTranscriptsForOneAgentBothStore`,
`TestASecondStopSupersedesTheFirstRecord` and
`TestAShorterRearrivalDoesNotWriteASecondRecord`; `Read`'s resolution order;
staging's sidecar and its legacy path (`TestStageWritesLineageSidecar`,
`TestListStagedReadsSidecarLineage`,
`TestLegacyStagedFileWithoutSidecarStillDrains`,
`TestCorruptSidecarIsNotTreatedAsPermanent`,
`TestConcurrentSubAgentStagesAllLand`,
`TestStageConcurrentSameSessionYieldsOneCopy`); the drain's ordering, budget and
re-read (`TestDrainTakesMainThreadFirst`, `TestDrainTakesOverdueEntriesFirst`,
`TestDrainByteBudgetBoundsThePass`, `TestDrainByteBudgetAlwaysMakesProgress`,
`TestDrainBudgetLeavesRemainderLoudly`, `TestDrainRereadsALongerSource`,
`TestDrainIgnoresADivergentSource`, `TestDrainCarriesSidecarLineageIntoTheRecord`,
`TestDrainKeepsStagedOnCaptureFailure`,
`TestDrainLeavesAReStagedCopyForTheNextPass`); the staged lifetime
(`TestStagedEntryPastTheLimitReportsOverdue`,
`TestOverdueEntryIsNeverDeletedByAge`,
`TestRetryableFailureStaysStagedAndRetryable`,
`TestDeterministicRefusalIsQuarantinedNotRetriedForever`,
`TestSurveyBacklogSeesEveryRepositoryInTheStore`,
`TestSurveyBacklogSkipsRepositoriesHoldingNothing`,
`TestDiscardRemovesOneTranscriptAndItsMetadata`,
`TestDiscardRefusesAnythingButABareStagedFilename`); the store resolution and the
gap marker (`TestSessionRepoRoundTrip`, `TestSessionRepoRefusesAnAmbiguousSession`,
`TestSessionRepoUnknownSessionIsAnError`,
`TestSessionRepoRefusesADirectoryReference`, `TestSessionNotesArePruned`,
`TestSubagentGapMarker`); and the flush-race predicate (`TestTranscriptSettled`).

`internal/surface/cli`: the hook table above, plus
`TestHookSubagentStopResolvesTheRepoThroughTheSession`,
`TestHookSubagentStopWaitsForAnUnsettledTranscript`,
`TestHookSubagentStopConcurrentCompletionsAllStage`,
`TestHookPlaneFailsOpenOnEveryUsageError` and `TestHooksManifestNamesLiveSubverbs`
(extended by the manifest to cover the new path),
`TestPromptRouterDrainsWhileTheSessionIsLive`,
`TestPromptRouterDrainsEvenWhenTheRulesLoaderFails`,
`TestSessionStartReportsAnotherRepositorysBacklog`,
`TestHistoryStagedAllReposSurveysTheWholeStore`,
`TestHistoryDiscardRefusesWithoutConfirmation`.

Gates: `surface_coverage` over the new sub-verbs, the regenerated snapshot and
command reference, and `make record-lint` over this spec and the intent.

## Uncertainties

- **Whether the completion event fires after the sub-agent's transcript is
  flushed is STILL NOT ESTABLISHED.** It is the intent's own falsifier. The
  mitigations shipped and count their own effect, but the residual rate over a
  real corpus has not been measured, and the intent's scope condition says so
  rather than claiming it settled.
- **How much lineage survives without the harness's sidecar** is unmeasured. The
  payload carries the agent id and the agent type, so ac-3 holds regardless; what
  degrades without it is the spawn point.
- **The ADR ordinal question flagged above.** Two committed surfaces disagree
  about how an ADR id is allocated. That does not block this spec, but it blocks
  minting the ADR this spec defers.

## Out of scope

Redesigning the store's keying or provisioning. The harness's retention policy.
Recovering transcripts already on disk and repairing the composite records
(`itd-2609091718566731`). Reconstruction and its telemetry
(`itd-2609091718595846`). Structured extraction of findings from a transcript.
Any change to the scanner's detectors or to the two-stage redaction discipline
itself. Any change to `source_kind`.
