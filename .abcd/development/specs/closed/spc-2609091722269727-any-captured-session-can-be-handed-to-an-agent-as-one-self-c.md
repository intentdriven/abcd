---
id: spc-2609091722269727
slug: any-captured-session-can-be-handed-to-an-agent-as-one-self-c
intent: itd-2609091718595846
origin: researcher-authored
production_mode: hand-written
---
# One session, one artefact, one telemetry file: what `history reconstruct` renders and what it refuses to assert

## Summary

This spec delivers
[itd-2609091718595846](../../intents/planned/itd-2609091718595846-any-captured-session-can-be-handed-to-an-agent-as-one-self-c.md):
a per-agent archive becomes a readable session.

`history reconstruct <session-id>` emits two files — one Markdown artefact
(`<session>.md`) and one machine-readable telemetry file
(`<session>.telemetry.json`). Core writes nothing and knows no path; the CLI
writes both into `--out`, or both to stdout.

It reads the lineage
[itd-2609090559376002](../../intents/planned/itd-2609090559376002-sub-agent-transcript-capture.md)
put on each record and adds no write path of its own, so the store of
[adr-29](../../decisions/adrs/0029-native-transcript-corpus.md) is untouched by
it. Two things in the shipped shape are deliberate reversals of what was
planned, and both are recorded below: the sub-agent sections are appended after a
contiguous main thread rather than nested at their spawn points, and token usage
is de-duplicated per response rather than summed per line.

## Scope

In: `internal/core/history` — `Reconstruct`, `ReconstructOptions`,
`Reconstruction`, `Telemetry`, `AgentTelemetry`, `TokenCounts`, `TurnCounts`,
`Completeness`, `DroppedRecord` (`reconstruct.go`) and the artefact renderer
(`reconstruct_render.go`). The `reconstruct` sub-verb in
`internal/surface/cli/history_reconstruct.go`, wired from `history.go`. The
plugin page `commands/history.md`, the brief chapter
[`04-surfaces/11-history.md`](../../brief/04-surfaces/11-history.md), the
regenerated surface snapshot and command reference.

Out: every write path into the store; the scanner and redaction (reconstruction
reads records that were already redacted on write); the record schema; capture;
ingest and migrate; any roll-up across sessions.

## Approach

### The signature carries the size problem

```go
type ReconstructMode string
const ( ModeFull ReconstructMode = "full"; ModeSpine ReconstructMode = "spine" )

type ReconstructOptions struct {
	SessionID     string
	Mode          ReconstructMode // "" → ModeFull
	MaxBlockBytes int             // 0 = unbounded; negative is an error
}

func Reconstruct(rootSHA string, opts ReconstructOptions) (Reconstruction, error)
```

The plan's `Reconstruct(rootSHA, sessionID)` did not survive contact with a real
session: a 55-record session renders to 7.7 MB, and the whole point of the
artefact is that it be handed to a model. Both remedies are caller choices, so
they are options rather than constants. `spine` mode keeps the main thread whole
and reduces each delegate to its first and last turn, with counted gap markers;
`MaxBlockBytes` (8 KiB at the front door) caps one rendered tool input or result,
marked where it happens and counted in the telemetry. There is deliberately no
per-agent mode: `abcd history show <agent-id>` already answers that.

`Reconstruction` returns the artefact bytes, both filenames, the artefact size
and the `Telemetry` structure. `Artefact` is `json:"-"`, so the `--json` envelope
carries the telemetry and the names but never the document.

### Record selection, and saying what was not used

One session's records are loaded and reduced to one thread per `(session,
agent)`: readable first, then longest body, then newest capture, then filename
descending. Every loser is reported in `Completeness.DroppedRecords` with a
reason and named in the document's own `## Completeness` section. Nothing is
dropped silently — a session whose superseded records vanished from the artefact
without a word would be a reconstruction that quietly disagreed with the store.

A session with no main-thread record still renders, labelled, with a completeness
note. Only a session with no records at all is an error.

### The layout: appended sections, doubly marked — a reversal

The plan called for `## Sub-agent <agent-id>` sections nested at their spawn
points by `spawn_depth`. **The corpus refuted it.** The sub-agents whose id a
spawning transcript records are the ASYNCHRONOUS ones, and for those the spawn
and the join are many turns apart — one measured case spawned at turn 18 and
joined at turn 167. Splicing the section in at the spawn point puts a delegate's
conclusions in front of 148 main-thread turns that ran before those conclusions
existed, which is precisely the false inference the artefact exists to prevent.

So the main thread stays contiguous and is never spliced, and each sub-agent gets
its own appended `## Agent <agent-id>` section. Attribution is carried three
ways instead of by indentation:

- **Two inline markers in the spawning thread.** `[SPAWNED agent <id> (<type>)
  here — its transcript is in section "Agent <id>". Everything below this line up
  to its JOIN marker ran without its result.]` and `[JOINED agent <id> here — its
  result reached this thread at this turn.]`, collapsed into one marker when the
  spawn and the join are the same turn (a synchronous delegate).
- **A per-agent provenance block** naming the type, spawn depth, spawner, spawn
  turn and tool call, join turn, span, turns, tokens and source record — plus a
  `CONCURRENCY:` line stating how many turns of the spawning thread ran between
  the spawn and the join without this agent's result.
- **`## Agent timeline`**, a table at the head of the document with one row per
  thread carrying spawn, start, end, join, turns and tokens.

Nesting survives as DATA (`spawn_depth`, the parent, the spawning thread), not as
layout. Concurrency is read off the timeline; **section order asserts nothing
about time**, and the document says so in words.

Placement is a two-rung ladder. Rung 1 is the record's own stored spawning tool
call, matched against the host thread's tool-use blocks (`placed_by: record`).
Rung 2 is the host transcript's own tool result naming the agent id, resolved
back to the call that issued it (`placed_by: transcript`) — the spec's deferred
"resolved at reconstruction time" rung, delivered. The host is the parent agent's
thread when that parent is in the set, the main thread when no parent is named,
and NOTHING when a parent is named but absent: placing such an agent on the main
thread would assert a spawn that did not happen.

An agent neither rung places goes under `## Unattributed sub-agents`, last and
labelled, never interleaved. Membership is decided by "no recoverable spawn
point", which is not the same fact as the record's own `spawn_attribution`; the
telemetry counts the two separately (`agents_unattributed` versus
`agents_without_spawn_point`).

### Structure the transcript cannot forge

A text block is the one kind of content a session participant chose the bytes of.
Rendered raw it can emit `## Agent <id>`, `### Turn 99 — assistant`, or a
`[JOINED …]` marker byte-for-byte, fabricating an agent that never ran or a
result that never arrived. Redaction has nothing to say about this: the text is
not secret, it is structural.

The answer is containment, applied uniformly. **Every** block — text included —
goes through one fenced writer that measures the longest backtick run in the
content and opens with one more, so content carrying its own fences cannot close
the block containing it. One escaping mechanism everywhere beats a per-type
judgement about which content is dangerous, because the judgement is what drifts.

The second half is declarative. The artefact carries a `## How to read this
document` section that enumerates the structures the document itself asserts and
states the invariant: everything inside a fence is something somebody said,
everything outside one is this document. A forged heading inside a fence is
quoted content asserting nothing, however exactly it matches.
`TestReconstructCannotBeForgedByTranscriptText` plants forged headings, a forged
turn heading, a forged join marker and a fence of its own, and asserts none of
them appears as a line OUTSIDE a fence — while also asserting the content is
still reproduced, because this is containment and not redaction.

### Self-containment and determinism

Records are named by basename everywhere; the artefact carries no store path, no
harness path and no absolute path of any kind, and the CLI additionally
home-redacts every path it prints or marshals. The artefact carries **no
generation timestamp** — that lives on the telemetry alone — so the same records
reconstruct to identical bytes. `TestReconstructionIsSelfContained` and
`TestReconstructIsDeterministic` are the two detectors.

### Telemetry, and the counting bug it exists to avoid

`<session>.telemetry.json`, schema 1, carrying `session_id`, `root_commit`,
`mode`, `generated_at`, the span and `wall_clock_seconds`, `turns`, `tokens`,
`tool_calls` (a count per tool name), `models`, `agent_types`, one `agents` entry
per thread (the main thread included) with its own copy of every measure, and a
`completeness` block.

**Tokens are counted once per distinct response id, never once per transcript
line.** The harness writes one line per content block and repeats the response's
usage on every one of them, so a naive sum multiplies a response by its block
count — and the factor varies per session, so no constant could correct it. On a
real 55-record session, 3711 usage-bearing lines resolve to 1801 distinct
responses: an inflation of 2.06x (`iss-2609090723027424`). Turns de-duplicate the
same way, so a multi-line assistant response is one turn, and tool calls
de-duplicate on the tool-use block's own id.

Both counts are published side by side — `api_responses` and `usage_lines_seen` —
so a consumer can SEE that the de-duplication happened rather than take it on
trust. Usage carrying no message id is counted anyway (dropping it would
understate), tallied into `completeness.usage_without_message_id`, and given a
note saying the totals are an upper bound to that extent. That is the one
condition under which they can still be inflated, and it says so.

`completeness` is not decoration. The intent's scope condition says telemetry
describes what the harness recorded and is not a billing record, and a derived
measure that cannot say what it was missing invites exactly the cross-version
comparison the condition warns against. It reports whether the main thread was
present, records found against records used and why each loser was dropped, the
agent counts that failed each attribution question, unparseable lines,
un-de-duplicable usage, the field NAMES that were absent from the source lines
(distinguishing "zero happened" from "this harness does not record it"), and the
elision and omission counts.

One asymmetry worth stating: telemetry is computed BEFORE elision and spine
reduction, so the token and turn counts describe the whole transcript while
`elided_blocks`, `elided_bytes` and `omitted_turns` describe the artefact.

### Surfaces

`abcd history reconstruct <session-id>`, exactly one argument. `--out` (default
`.`, `-` for stdout), `--mode` (`full` | `spine`, default `full`),
`--max-block-bytes` (default 8192, 0 disables). Both files are written
atomically into an `--out` that must already exist — the directory is never
created — together or not at all. On stdout the artefact passes through the
terminal-safety filter first, because it is stored transcript text, and the
telemetry follows it in a fenced JSON block so a reader piping to a file still
gets both. The `--json` envelope carries the `Reconstruction` plus either the
written paths or the artefact string, never both.

The verb carries its section in `commands/history.md`, its row in the brief's
history chapter sub-verb table, and a regenerated surface snapshot and command
reference.

## How the Acceptance Criteria are satisfied

- **ac-1 (one artefact; each sub-agent attributable to its spawn point).** One
  Markdown document holding the main thread and every stored sub-agent, with the
  spawn/join markers, the per-agent provenance block and the timeline table
  carrying the attribution, and the unplaceable segregated. Tests:
  `TestReconstructMarksEverySubagentAtItsSpawnAndJoinPoint`,
  `TestReconstructPlacesAnAgentFromTheSpawningTranscriptAlone` (the second
  attribution rung on its own), `TestReconstructSegregatesUnattributedSubagents`,
  `TestReconstructChoosesOneRecordPerAgentAndSaysWhich`,
  `TestReconstructRendersASessionWithNoMainThread`,
  `TestReconstructRefusesWhatItCannotAnswer`. On the real 55-record session, all
  54 sub-agents were placed.
- **ac-2 (a machine-readable telemetry file with the named measures).**
  The `Telemetry` structure and its per-agent breakdown. Tests:
  `TestTelemetryReportsEveryRequiredMeasure` asserts each named field is present
  and non-trivial over a fixture session;
  `TestTelemetryReportsItsOwnIncompleteness` asserts the completeness block names
  a field the fixture deliberately omits;
  `TestTelemetryCountsOneUsagePerResponse` and
  `TestTelemetryReportsUsageItCouldNotDeduplicate` hold the counting rule and its
  stated exception.
- **ac-3 (readable without the store or the harness's files).**
  `TestReconstructionIsSelfContained` asserts the artefact contains no absolute
  path, no store root and no reference resolvable only against the harness;
  `TestReconstructIsDeterministic` bounds it from the other side by proving the
  document carries nothing about the run that produced it.

## Tests

`internal/core/history`: `TestReconstructMarksEverySubagentAtItsSpawnAndJoinPoint`,
`TestReconstructPlacesAnAgentFromTheSpawningTranscriptAlone`,
`TestReconstructSegregatesUnattributedSubagents`,
`TestReconstructRendersASessionWithNoMainThread`,
`TestReconstructChoosesOneRecordPerAgentAndSaysWhich`,
`TestReconstructRefusesWhatItCannotAnswer`,
`TestReconstructionIsSelfContained`, `TestReconstructIsDeterministic`,
`TestReconstructCannotBeForgedByTranscriptText`,
`TestReconstructCapsABlockAndSaysSo`,
`TestSpineModeKeepsTheThreadAndSummarisesTheDelegates`,
`TestTelemetryReportsEveryRequiredMeasure`,
`TestTelemetryReportsItsOwnIncompleteness`,
`TestTelemetryCountsOneUsagePerResponse`,
`TestTelemetryReportsUsageItCouldNotDeduplicate`.

`internal/surface/cli`: `TestReconstructWritesBothFiles`,
`TestReconstructToStdoutWritesNoFiles`,
`TestReconstructJSONEnvelopeCarriesTheTelemetry`,
`TestReconstructRefusesAnAbsentOutDirectory`,
`TestReconstructRefusesAnUnknownSession`.

Gates: `surface_coverage` over the new sub-verb, the regenerated snapshot and
command reference, and `make record-lint` over this spec and its intent.

Measured run: a real 55-record session reconstructs to a 7.7 MB artefact plus
64 KB of telemetry in about 0.6 seconds, placing all 54 sub-agents.

## Uncertainties

- **Whether the full artefact fits the context it is read into.** 7.7 MB for a
  55-record session does not, for most readers. `spine` mode and the per-block
  cap are the two answers shipped, and neither has been evaluated against a real
  question put to a real model.
- **How much lineage survives without the harness's per-agent sidecar.** The
  first attribution rung depends on it. The second rung recovers a spawn point
  without it, but only for agents the spawning transcript names, so what degrades
  is the precision of the timeline rather than its existence — and the rate has
  not been measured.
- **Cross-version comparability of the telemetry.** The completeness block is
  what makes a comparison arguable rather than reckless, but nothing has yet
  compared two harness versions to find out whether the fields hold still.

## Out of scope

Every write path into the store. The record schema and the lineage fields, which
are itd-2609090559376002's. Capture, ingest and migrate. A corpus-level
telemetry roll-up across sessions, which needs a stable per-session file first —
this is it, and the roll-up is not. Structured extraction of findings from a
transcript. Any reconciliation of the token counts against a vendor's
accounting.
