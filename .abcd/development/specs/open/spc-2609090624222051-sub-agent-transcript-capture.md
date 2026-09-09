---
id: spc-2609090624222051
slug: sub-agent-transcript-capture
intent: itd-2609090559376002
origin: researcher-authored
production_mode: hand-written
---
# A session's sub-agents are captured, ingested, and handed back as one artefact with its telemetry

## Summary

This spec delivers
[itd-2609090559376002](../../intents/planned/itd-2609090559376002-sub-agent-transcript-capture.md):
the transcript corpus stops keeping the smaller half of the work.

Four things land. **Capture** gains a `SubagentStop` hook that stages the
finished sub-agent's own transcript through the staging path
[spc-4](../closed/spc-4-start-the-transcript-clock.md) built, so redaction and
its fail-closed refusal are reused rather than reimplemented. **The record
schema** gains explicit lineage fields, so a sub-agent record says which session
spawned it, which agent spawned it, and what kind of agent it was, instead of
smuggling that into a hand-made composite `session_id`. **Ingest** brings
transcripts already on disk into the corpus, under a destination repository the
caller names, so each repository's own redaction configuration applies to its
own transcripts and never to another's. **Reconstruction** emits, per session,
one self-contained agent-readable artefact plus a machine-readable telemetry
file.

Nothing here redesigns the store. The per-repo root-SHA keying of
[adr-29](../../decisions/adrs/0029-native-transcript-corpus.md) stands, the
two-stage redaction stands, and `Capture` remains the only path that writes a
record.

## Scope

In: `internal/core/history` (the record schema, `Capture`, `Stage`, `Drain`,
`Read`, and the new `Ingest`, `Migrate` and `Reconstruct`); the `hook` sub-tree
and the `history` sub-tree in `internal/surface/cli`; `hooks/hooks.json`; the
per-repo configuration file `.abcd/config/history.json`; the plugin page
`commands/history.md`; the brief chapter
[`04-surfaces/11-history.md`](../../brief/04-surfaces/11-history.md); the
generated command reference and the committed surface snapshot.

Out: the scanner and its detectors; the store's keying and provisioning; the
harness's retention policy; a corpus-level roll-up across sessions; structured
extraction of findings from a transcript.

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

`session_id` keeps its existing meaning unchanged: the session the transcript
belongs to. On a sub-agent record it holds the full, untruncated id of the
spawning session, which is what the `SubagentStop` payload delivers.

`source_kind` is deliberately NOT extended with a sub-agent value. Kind says
where the bytes came from (`native`, `specstory-import`), not what produced
them; encoding agent-ness there would repeat the overloading this section
removes.

`recordSchemaVersion` moves to 2. A schema-1 record parses as a main-thread
record with empty lineage, so `List` and `Read` keep working over everything
already stored, and the composite records stay readable until `migrate` runs.

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
  `CaptureMeta` carries the session id, the six lineage fields and the kind. Six
  more positional string arguments on a security-critical call is a
  transposition waiting to happen.

`Read` resolves its key in three steps: an exact record filename, then an exact
`agent_id`, then a `session_id` (preferring the main-thread record, newest
first). A new `ListForSession(rootSHA, sessionID)` returns the whole set for one
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
the whole write as it does today.

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
as `hook session-end`: fail-closed, always exit 0, diagnostics on stderr,
stdout empty. It reads the hook payload from stdin and uses five of its fields:

- `agent_transcript_path`: the finished sub-agent's own transcript. This is the
  whole reason the gap can be closed without reading the harness's on-disk
  layout, and its absence is the fallback case below.
- `agent_id`, `agent_type`: the lineage the payload carries directly.
- `session_id`: the spawning session, stored untruncated.
- `cwd`: resolved to the repository's root-commit SHA through `ahoy.Detect`,
  exactly as `hook session-end` does.

It **stages, it does not capture**, for the reason staging exists: redaction runs
at roughly 0.7 s per megabyte, and `SubagentStop` fires inside a live session
where a stall is felt directly. The next `SessionStart` drains it through the
unchanged `Capture`.

`parent_agent_id` and `spawn_depth` are not in the payload. They are resolved
down a three-rung ladder, and the rung that answered is recorded:

1. **The harness's own sidecar**, derived from the payload's transcript path by
   substituting the extension, never by walking a directory. When it is present
   it carries the spawning agent, the spawn depth and the tool-call identifier
   that launched the agent. The read is guarded and best-effort: absence,
   unreadability, or a mismatched agent id all leave the fields empty and are
   never an error.
2. **The spawning transcript's own tool result** for asynchronously launched
   agents, which names the agent id beside the tool-call id. Resolved at
   reconstruction time, not in the hook.
3. **Unattributed.** The record stores the sub-agent with empty spawn fields,
   and reconstruction lists it in a separate, labelled section rather than
   guessing a spawn point.

Wiring: `hooks/hooks.json` gains a `SubagentStop` entry running
`"$CLAUDE_PLUGIN_ROOT/abcd" hook subagent-stop`, using the same binary
resolution preamble as the existing `SessionEnd` entry, and
`applyHookPlaneFailOpen`'s path list gains `{"hook", "subagent-stop"}` so a
usage error on this verb fails open like every other hook path. Without both,
the verb is dead scaffolding.

### Staging carries a sidecar

Staged files are currently identified by parsing their filename, which is why
the filename had to encode the session. Encoding lineage there would rebuild the
composite-identifier defect one directory earlier.

So `Stage` gains a sidecar. It writes two files, both mode 0o600:

- `<stamp>-<key>.raw`, the raw bytes, where `<key>` is the agent id when there
  is one and the session id otherwise.
- `<stamp>-<key>.stage.json`, holding a schema version, the session id, the six
  lineage fields, the source path the bytes were read from, and the stage time.

`listStaged` reads the sidecar when it is present. A `.raw` file with no sidecar
is a staged file from an older binary: its session id is parsed from the
filename exactly as today and it drains as a main-thread transcript, so an
upgrade never strands a backlog.

Stage idempotency moves from the session id to the `(session_id, agent_id)` pair,
with the existing content compare and the existing last-writer-wins replacement,
all still inside the staging lock.

### The drain's order and its budget

Two changes, both forced by the volume this intent admits.

**Order.** `listStaged` is chronological, and a session's main thread is staged
last because it ends last, so a bounded pass would drain the branches and leave
the spine. The drain therefore takes main-thread entries first, then sub-agent
entries oldest-first. If a pass is truncated, what it stored is the part that
makes the rest legible.

**Budget.** `sessionStartDrainBudget` becomes a pair: a byte bound and a count
bound, whichever is reached first. The byte bound is the one that protects the
first prompt, because redaction cost tracks bytes rather than files, and the
count bound keeps a pathological many-tiny-files case bounded too. A session
that delegates heavily can stage dozens of transcripts, and a count-only budget
of four would leave the rest unredacted at 0o700 for as many sessions as it
takes to work through them. The remainder is reported as it is today, and the
`SessionStart` notice gains a sentence naming how many staged transcripts are
still unredacted, because a growing pile of raw text is a privacy fact and not a
scheduling detail.

### `history ingest`: the destination is an operand

`Ingest` is the recovery path for transcripts that are on disk and were never
captured. Its signature makes the destination explicit and non-derivable:

```
history.Ingest(dest Destination, sources []string, opts IngestOptions) (IngestResult, error)
```

`Destination` carries the repository root and its root-commit SHA. The scanner
is constructed from `dest.RepoRoot` and from nothing else, so a repository's own
`pii.json` and `gitleaks.json` govern its own transcripts and can never be
applied to another repository's. This is the seam the intent requires, and it is
a seam rather than a default because the working directory is exactly the wrong
authority here: an operator recovering a backlog is not standing in the
repository the transcripts belong to.

`sources` are explicit file or directory paths. There is no implicit "scan the
harness's store" mode: a vendor path baked into core would be the on-disk-layout
dependency the intent's mechanism rules out. A repository that ingests regularly
declares its roots in `.abcd/config/history.json` so the operator does not
retype them, and the configuration is the only place a path lives. Directory
sources are walked for line-delimited transcripts to a bounded depth; every read
is guarded and capped exactly as the hook path's read is.

`IngestResult` reports four populations: `captured`, `skipped` (each with a
reason), `orphans`, and `failed`. Nothing is silent.

### Resolving the owning repository

Per transcript, in order:

1. Collect the distinct `cwd` values recorded on the transcript's own lines,
   reading a bounded prefix plus the final line rather than the whole file.
2. Resolve each through `ahoy.Detect` to a root-commit SHA. A session run in a
   worktree resolves to the repository the worktree derives from, because they
   share a root commit, which is the scope condition the intent already records.
3. One root SHA, equal to `dest.RootSHA`: ingest it.
4. One root SHA, different: skip with reason `owned-elsewhere`, reporting the
   root SHA. A SHA names a repository without naming it.
5. Two or more distinct root SHAs: skip with reason `ambiguous-owner`. A
   transcript is never split between stores.
6. No resolvable `cwd`, or a `cwd` that no longer exists on disk: an orphan,
   handled below.

The harness's project-directory name is never decoded. It is not reversible to a
filesystem path (a directory named for a repository and a path with a separator
in the same position produce the same mangled name), so the recorded `cwd` is
the only sound signal, and it is the one this uses.

### Orphans are ignored, reported, and adopted only by name

The default for a transcript whose repository cannot be found on disk is to
**ignore it and say so**. It is listed in `orphans` with its project directory
name as given and its recorded working directory home-redacted, and nothing is
written.

Adoption is opt-in and per repository, declared in `.abcd/config/history.json`:

```json
{
  "schema_version": 1,
  "ingest_roots": [],
  "adopt_projects": [],
  "on_orphan": "ignore"
}
```

`adopt_projects` lists the project directory names this repository claims. A
transcript under a claimed name is ingested into this repository's store, under
this repository's redaction configuration, and its record is stamped
`lineage_source: ingest` together with `adopted_project`, so the adoption is
recorded on the artefact rather than only in a run's output.

`on_orphan` accepts `ignore` (the default) or `prompt`. Core never prompts:
under `prompt` it still ingests nothing and returns the orphan list, and the CLI
front door is what asks the operator and re-invokes ingest with the chosen names
in `opts.Adopt`. The transport-agnostic boundary is not negotiable, and an
interactive question is a transport concern.

Configuration is loaded by `history.LoadConfig(repoRoot)` under the same
discipline the scanner's per-repo configuration already uses: a size cap, a
symlink refusal, and containment inside the repository.

### `history migrate`: the composite records

The records already written under composite identifiers are the migration's
input. The count is per-machine local data rather than a fixture; the store this
was designed against held 176 of them.

`Migrate(rootSHA string, apply bool)` walks the store for a `session_id` matching
`^<prefix>--agent-<agent>$` and, for each:

- takes `agent_id` from the suffix;
- **repairs the truncated parent id from the record's own body**, which still
  carries the full session identifier on its transcript lines, and requires the
  recovered value to begin with the stored prefix. A body that disagrees, or one
  where no session identifier can be found, leaves the record untouched and is
  reported. The truncation is lossy, so the prefix is a check and never a source;
- leaves `agent_type`, `parent_agent_id` and `spawn_depth` empty, since neither
  the record nor the store ever held them, and stamps
  `lineage_source: migrated` so the emptiness reads as unrecoverable rather than
  as a main-thread record;
- rewrites the frontmatter atomically and does not touch the body.

Three properties matter. `source_sha256` is computed over the raw source and is
not recomputed, so a migrated record still dedups against a re-capture of the
same bytes. The filename is left alone: a rename would break any path a reader
already holds and buys nothing, because listing reads frontmatter. And the verb
**reports by default and writes only under `--apply`**, because the store holds
the only copy of these records. Re-running it is a no-op: a record that already
carries `agent_id` is skipped.

### `history reconstruct`: one artefact and one telemetry file

`Reconstruct(rootSHA, sessionID string) (Reconstruction, error)` returns the
artefact bytes and the telemetry structure. Core writes nothing; the CLI writes
the two files into `--out` (default the current directory) or to stdout for `-`.

The artefact is **Markdown**, named `<session-id>.md`. The requirement is that
it be handed to a model as context and read without the store or the harness's
files, and Markdown is what a model reads without a schema. Its shape:

- A header block: schema version, session id, root commit, span, record count,
  agent count, and a completeness block saying what was missing.
- `## Main thread`, rendered turn by turn.
- At each spawn point, an inline marker naming the agent type, the agent id and
  the time, immediately before the nested section it introduces.
- `## Sub-agent <agent-id>` sections, nested by `spawn_depth`, each stating who
  spawned it, at which point, and over what span. A sub-agent that spawned
  sub-agents nests again, to whatever depth the records carry.
- `## Unattributed sub-agents`, last, for anything the attribution ladder could
  not place. Labelled, never interleaved.

Concurrency is represented rather than linearised, which answers the intent's
open question. Sub-agents are ordered by their spawn point in the spawning
transcript, not by their own timestamps; several spawned in one turn are
rendered in the order their spawn points appear in that turn, and the section
says in words that they overlapped, with each carrying its own start and end.
Nothing in the artefact implies that a later section began after an earlier one
ended.

The artefact is self-contained by construction and by test: it carries no store
path, no harness path and no absolute path of any kind.

The telemetry file is `<session-id>.telemetry.json`:

- `schema_version`, `session_id`, `root_commit`
- `started_at`, `ended_at`, `wall_clock_seconds`
- `turns`: user, assistant, total
- `tokens`: input, output, cache creation input, cache read input
- `tool_calls`: a count per tool name
- `models`: the distinct model identifiers seen
- `agent_types`: the distinct agent types seen
- `agents`: one entry per agent (the main thread included) carrying its id,
  parent, type, spawn depth, model, span, turns, tokens and tool-call counts
- `completeness`: records present, sub-agents unattributed, and the named fields
  that were absent from the source lines

`completeness` is not decoration. The intent's scope condition says telemetry
describes what the harness recorded and is not a billing record, and a derived
measure that cannot say what it was missing invites exactly the comparison
across harness versions the condition warns against.

### Failure modes

- **The event fires before the transcript is flushed.** The likeliest way this
  design is shown wrong, and the first thing implementation measures. Two
  mitigations, both cheap. At stage time, a transcript whose final line is not
  complete JSON is re-read after a short bounded wait, up to a small fixed
  number of attempts. At drain time, the source path recorded in the staging
  sidecar is re-read when it still exists, and its bytes replace the staged copy
  only when they are strictly longer AND the staged bytes are a prefix of them,
  so a recycled path can never substitute a different transcript. Truncation
  that survives both is a shorter record, not a corrupt one, and the residual
  rate is what the measurement reports.
- **A session with many sub-agents.** Staging is one write per completion, so
  the hook's cost is independent of the count. The drain is where the count
  lands, which the byte-and-count budget and the drain ordering above address.
  The failure this leaves is a backlog of unredacted staged text, which is
  reported by `history staged` and named in the `SessionStart` notice.
- **The payload is absent on an older harness.** No `agent_transcript_path`
  means the hook warns on stderr, stages nothing and exits 0. Silence is what
  this intent exists to end, so the hook also drops a marker beside the staging
  directory the first time it sees a payload without the field, and
  `history staged` reports it: this harness version does not deliver sub-agent
  transcripts, so no sub-agent capture is happening here.
- **An unreadable transcript.** The existing guarded read already refuses a
  symlink, a FIFO, a device node, a non-regular file and an over-cap file, and
  refuses an over-cap file whole rather than truncating it. Reported on stderr,
  nothing staged, exit 0.
- **A degraded scanner.** Unchanged and untouched. Sub-agent capture runs
  through the same `Capture`, so the refusal on a degraded scanner or a surviving
  blocking span applies by construction; the tests assert it on the sub-agent
  path specifically rather than inferring it.

### Surfaces

Three new user-facing sub-verbs (`ingest`, `reconstruct`, `migrate`) and one new
operator-internal hook verb. Each needs, in the same change:

- The `history` sub-tree moved out of `internal/surface/cli/cli.go` into
  `internal/surface/cli/history.go` as a pure move with no behaviour change,
  before the new verbs are added, so they land in a file that is about one thing.
- Sections in `commands/history.md`, which is the plugin surface for a user verb.
  The hook verb's plugin surface is its `hooks/hooks.json` entry, as it is for
  `hook session-end`.
- Rows in the brief's history chapter sub-verb table, which `surface_coverage`
  checks in both directions and refuses on.
- A regenerated surface snapshot (`cmd/abcd-gen-surface`) and a regenerated
  command reference (`cmd/abcd-gen-cli-ref`).

### Landing order

Each step leaves the tree green, and each has its own tests.

1. **The record schema.** Schema version 2, the six fields, `CaptureMeta`, the
   widened idempotency key, the frontmatter-through-the-redaction-pass change,
   `Read`'s three-step resolution and `ListForSession`. Existing callers updated;
   no new verb yet.
2. **Staging's sidecar.** `.stage.json`, the `(session, agent)` idempotency key,
   the legacy sidecar-less path, the drain's main-thread-first ordering and the
   byte-and-count budget.
3. **The hook.** `abcd hook subagent-stop`, its `hooks.json` entry, its
   fail-open path registration, the harness-sidecar rung and the missing-payload
   marker. This is the point at which the corpus starts accruing, so it lands
   before the recovery verbs.
4. **The measurement.** An instrumented run over real sessions counting how many
   staged sub-agent transcripts arrive truncated. If the rate is not near zero,
   revisit step 3's mitigations before continuing; the number goes into the
   intent's grounds either way.
5. **`history migrate`.** Report mode, then `--apply`, run against the real store
   once the tests pass.
6. **`history ingest`.** The destination seam, the owning-repo resolution, the
   configuration file, and the orphan policy with its front-door prompt.
7. **`history reconstruct`.** The artefact, then the telemetry file, then the
   attribution ladder's second rung.
8. **The surfaces.** The CLI file move happens at the top of step 6; the plugin
   page, the brief rows, the surface snapshot and the command reference are
   regenerated once, here, and the docs-currency and surface-coverage gates are
   what prove it.

## How the Acceptance Criteria are satisfied

The intent's criteria in order.

- **ac-1 (a sub-agent's transcript is stored, redacted, and listed).** The
  `SubagentStop` hook stages it; the next drain captures it through the unchanged
  fail-closed `Capture`; `List` returns it. Test:
  `TestSubagentStopStagesAndDrainStores` drives the verb with a payload and
  asserts a record with the lineage fields set.
- **ac-2 (nesting, each attributable to its spawner).** `parent_agent_id` and
  `spawn_depth` come from the attribution ladder and are stored per record. Test:
  `TestNestedSubagentsAreBothStoredAndAttributed` stages a depth-1 and a depth-2
  agent and asserts the parent chain resolves.
- **ac-3 (found from the spawning session's identifier, with its kind).**
  `session_id` holds the untruncated spawning session, so `ListForSession`
  returns the main thread and every sub-agent, each carrying `agent_type`. Test:
  `TestListForSessionReturnsMainThreadAndEverySubagent`.
- **ac-4 (a degraded scanner refuses).** Unchanged `Capture`. Test:
  `TestSubagentCaptureRefusesDegradedScanner` runs the sub-agent path with a
  broken per-repo configuration and asserts no record was written.
- **ac-5 (the same transcript twice is a no-op).** The widened idempotency key.
  Tests: `TestSubagentCaptureIdempotentOnSourceSHA` and
  `TestTwoSubagentsWithIdenticalBytesBothStore`, the second bounding the first so
  the key cannot be over-tightened back into the collapsing bug.
- **ac-6 (a completion without a readable transcript is reported, not silent and
  not fatal).** The hook's warn-and-exit-0 paths plus the missing-payload marker.
  Test: `TestSubagentStopNeverBlocksTheHost`, a table over an absent path, a
  FIFO, a symlink, an over-cap file, a malformed payload and a payload with no
  `agent_transcript_path`, each asserting exit 0, zero records, and a non-empty
  stderr reason.
- **ac-7 (ingest under a repository's own configuration, twice adds nothing).**
  `Ingest`'s explicit destination and the unchanged idempotency. Tests:
  `TestIngestUsesTheDestinationReposRedactionConfig` (two fixture repositories
  with different rules, asserting the destination's applied and the source's
  not) and `TestIngestTwiceIsANoOp`.
- **ac-8 (an unidentifiable repository is skipped and reported).** The resolution
  ladder's steps 4, 5 and 6. Tests: `TestIngestSkipsATranscriptOwnedElsewhere`,
  `TestIngestSkipsAnAmbiguousOwner`, `TestIngestReportsAnOrphanAndWritesNothing`.
- **ac-9 (a configured adoption stores and records it).** `adopt_projects` plus
  the `adopted_project` stamp. Test: `TestIngestAdoptsAConfiguredOrphanProject`
  asserts the record exists in the destination store and carries the stamp;
  `TestIngestDoesNotAdoptWithoutConfiguration` is its negative control.
- **ac-10 (one artefact, each sub-agent attributable to its spawn point).**
  `Reconstruct`'s nested rendering and its unattributed section. Tests:
  `TestReconstructNestsEverySubagentAtItsSpawnPoint` and
  `TestReconstructSegregatesUnattributedSubagents`.
- **ac-11 (a machine-readable telemetry file with the named measures).** The
  telemetry structure. Test: `TestTelemetryReportsEveryRequiredMeasure` asserts
  each named field is present and non-trivial over a fixture session, and
  `TestTelemetryReportsItsOwnIncompleteness` asserts the completeness block names
  a field the fixture deliberately omits.
- **ac-12 (readable without the store or the harness's files).** Test:
  `TestReconstructionIsSelfContained` asserts the artefact contains no absolute
  path, no store root and no reference resolvable only against the harness.

## Tests

`internal/core/history`: the schema round-trip over both versions
(`TestSchemaOneRecordReadsAsMainThread`), the frontmatter redaction
(`TestLineageFieldsAreRedactedWithTheBody`,
`TestBlockingSpanInAgentTypeRefusesTheWrite`), the idempotency pair above,
`Read`'s resolution order (`TestReadResolvesFilenameThenAgentThenSession`),
staging's sidecar and its legacy path (`TestStagedFileWithoutSidecarStillDrains`),
the drain's ordering and budget (`TestDrainTakesMainThreadFirst`,
`TestDrainStopsOnWhicheverBudgetBindsFirst`), migration
(`TestMigrateRepairsTheTruncatedParentFromTheBody`,
`TestMigrateLeavesARecordWhoseBodyDisagrees`, `TestMigrateIsIdempotent`,
`TestMigrateWritesNothingWithoutApply`), ingest's five resolution outcomes,
configuration loading (`TestHistoryConfigRefusesASymlink`,
`TestHistoryConfigDefaultsToIgnoringOrphans`), and reconstruction with its
telemetry.

`internal/surface/cli`: the hook table above, `TestSubagentStopWritesNothingToStdout`,
`TestHookPlaneFailsOpenOnEveryUsageError` (existing, extended by the manifest to
cover the new path), the three new verbs' rendering and `--json` envelopes, and
`TestIngestPromptNeverReachesCore` asserting that core returns orphans and
ingests none of them whatever `on_orphan` says.

Gates: `surface_coverage` over the new sub-verbs, the regenerated snapshot and
command reference, and `make record-lint` over this spec and the intent.

## Uncertainties

Three, stated rather than smoothed over.

- **Whether `SubagentStop` fires after the sub-agent's transcript is flushed** is
  not established. It is the intent's own falsifier, step 4 of the landing order
  measures it, and the mitigations are designed on the assumption that it
  sometimes does not.
- **How much lineage survives without the harness's sidecar** is unmeasured. The
  payload carries the agent id and the agent type, so ac-3 holds regardless; what
  degrades without it is the spawn point, and therefore the precision of ac-10's
  nesting rather than its existence.
- **The ADR ordinal question flagged above.** Two committed surfaces disagree
  about how an ADR id is allocated. That does not block this spec, but it blocks
  minting the ADR this spec defers.

## Out of scope

Redesigning the store's keying or provisioning. The harness's retention policy.
A corpus-level telemetry roll-up across sessions, which the intent raises as an
open question and which needs a stable per-session file first. Structured
extraction of findings from a transcript. Any change to the scanner's detectors
or to the two-stage redaction discipline itself. Renaming the records that
migration touches, and any change to `source_kind`.
