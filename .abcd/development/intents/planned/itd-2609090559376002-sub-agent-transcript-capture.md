---
id: itd-2609090559376002
slug: sub-agent-transcript-capture
spec_id: spc-2609090624222051
kind: standalone
suggested_kind: null
reclassification_history: []
related_adrs: [adr-29]
builds_on: []
severity: major
impact: additive
promoted_from: iss-2609081917287384
origin: extracted-from-record
production_mode: hand-written
---

# Any Session Can Be Handed to an Agent as One Complete, Measured Record of What It Did

## Press Release

> **abcd captures the reasoning of every sub-agent a session spawns, not just the session's main thread, and can hand any session back as a single self-contained artefact an agent can read, with a telemetry file describing what the work cost.** Today a session's own transcript is captured and redacted on write, but the sub-agents it spawns write their transcripts elsewhere and nothing collects them. What survives of a delegated task is the prompt that launched it and the summary it returned, while every command it ran, every file it edited and every judgement it made is dropped. Delegated work is the bulk of the work, better than three quarters of the recorded volume, so the store has been keeping the smaller half. This intent collects the missing half, backfills the history that already exists, and makes any session reconstructable as one artefact that another agent can be given as context.
>
> "I delegate the reviews, the sweeps and most of the implementation to sub-agents, so that is where the reasoning I would want to audit actually lives," said Maya, an autonomous-development practitioner. "When I went back to reconstruct why a change was made, the store had the two lines where I handed the task off and the summary that came back, and nothing in between. The part I most needed to read was the part that was never kept."

## Why This Matters

abcd's transcript corpus exists so that work can be reconstructed after the fact. That promise is answered for the main thread and silently unanswered for everything delegated out of it:

- **The session transcript IS captured:** the session-end hook receives the path to it and stages it for redaction and storage.
- **Sub-agent transcripts are NOT:** they are written per sub-agent, and no hook abcd registers ever names them, so nothing enumerates or reads them.

The asymmetry is not marginal. On the measured corpus, sub-agent transcripts held roughly three quarters of all recorded bytes, outweighing session transcripts by more than three to one. What the parent retains for each delegated task is the launch prompt and the returned report: in one measured case, two lines standing in for several hundred, discarding every tool call the agent made along the way.

This is a symmetry failure rather than a missing feature. The corpus already exists, already redacts on write, and already works for one caller class. The sub-agent caller was simply never routed into it. The harness offers a completion event for sub-agents whose payload names the finished sub-agent's own transcript, so the gap can be closed by feeding the existing store, with no new capture mechanism and no dependence on the harness's undocumented on-disk layout.

Two consequences make this urgent rather than tidy. Uncaptured transcripts age out under the harness's retention sweep, so the loss is permanent and ongoing. And the workaround already in use, inventing a composite identifier by hand, produces records that cannot be found from the identifier a reader actually holds.

Capturing the material is only half of the value. A session's transcripts are of interest as a set rather than one at a time, both for reading back what happened and as a corpus for studying how humans and agents actually work together. Every measure that study needs is already present in the raw transcripts and is discarded along with them.

## Typed Links

- **refines `itd-59`** (autonomous-worker transcript capture): the adjacent half of the same corpus. itd-59 covers the worker on abcd's own run seam; this covers sub-agents spawned inside an interactive session.
- **refines `adr-29`** (native transcript corpus): feeds the store that ADR established rather than redesigning it.
- **corrects a premise in `itd-59`**, flagged for human confirmation and not auto-classified: itd-59's "What's Out of Scope" records interactive-session capture as already solved. That holds only of the main thread, and planning itd-59 on the unqualified claim would rebuild the same blind spot.

## What's In Scope

- **The capture gap:** every sub-agent transcript lands in the corpus under the repository it belongs to, with the same redaction and the same fail-closed refusal on a degraded scanner as a session transcript.
- **All sub-agent completions** at any nesting depth, including sub-agents spawned by other sub-agents.
- **Provenance enough to read the result back:** a stored sub-agent transcript can be traced to the session that spawned it and to the kind of agent it was.
- **Ingest of history that already exists:** transcripts still on disk but never captured can be brought into the corpus for the repository they belong to, under that repository's own redaction configuration and never another's.
- **Reconstruction as an artefact:** any session can be emitted as one self-contained, agent-readable file containing the main thread and every sub-agent it spawned, suitable for handing to a model as context.
- **A telemetry file beside each reconstruction**, describing token usage, wall-clock duration, turn and tool-call counts, models used and agent types, so the corpus can be studied rather than only read.
- **Feed, don't fork:** capture and ingest reuse the existing staging and redaction path rather than adding a second mechanism beside it.

## What's Out of Scope

- **Redesigning the corpus.** The store's per-repo keying and its single-owner provisioning stay as they are.
- **The harness's own retention policy.** How long the harness keeps its transcripts is configuration outside abcd's control.
- **Structured extraction of findings.** Turning a review agent's transcript into structured findings is a separate concern from keeping the transcript.
- **Adopting orphaned transcripts by default.** Where a transcript's repository no longer exists, the default is to ignore it; adopting it is opt-in per repository, described below.

## Mechanism

We expect routing sub-agent capture through the harness's sub-agent completion event to close the gap without new machinery, because that event's payload names the finished sub-agent's own transcript path directly, so the existing stage-then-redact path can consume it unchanged and nothing needs to read the harness's undocumented directory layout. We expect reconstruction and telemetry to need no new instrumentation, because the raw transcripts already carry per-message token counts, timestamps, model identifiers, agent attribution and tool-call records. This is shown wrong if the completion event fires before the sub-agent transcript is flushed and readable, if a session spawning many sub-agents degrades under per-completion staging, if the payload is absent on a harness version abcd claims to support, or if the telemetry fields prove inconsistent enough across harness versions that derived measures cannot be compared.

## Scope Conditions

- Holds for harness versions whose sub-agent completion event carries the finished sub-agent's transcript path; a version without it falls back to no capture rather than to guessing at the on-disk layout. <!-- cond: cond-2609090624228012 -->
- Holds at the observed working scale of a few hundred sub-agents per repository per month, with individual transcripts up to a few megabytes and sessions up to roughly a hundred sub-agents. <!-- cond: cond-2609090624226418 -->
- Assumes sub-agent transcripts share the line-delimited shape the session transcript already uses, so one reader serves both. <!-- cond: cond-2609090624223171 -->
- Assumes a transcript's owning repository is determined by the working directory recorded inside the transcript, never by decoding the harness's project directory name, which is not reversible. <!-- cond: cond-2609090624228630 -->
- Assumes a session run in a worktree belongs to the store of the repository that worktree derives from. <!-- cond: cond-2609090624222424 -->
- Telemetry is descriptive of what the harness recorded and is not a billing record; token counts are as reported per message and may omit what the harness did not report. <!-- cond: cond-2609090624224793 -->

## Acceptance Criteria

- **Given** a session that spawns a sub-agent, **when** that sub-agent finishes, **then** its transcript is stored in the corpus for the session's repository, redacted on write, and listed by `abcd history`.
- **Given** a sub-agent that itself spawns a sub-agent, **when** both finish, **then** both transcripts are stored and each is attributable to the session that spawned it.
- **Given** a stored sub-agent transcript, **when** an operator looks it up from the identifier of the session that spawned it, **then** they can reach it and can tell what kind of agent produced it.
- **Given** a repository whose secret scanner is degraded, **when** a sub-agent transcript would be captured, **then** the capture refuses rather than storing under weakened redaction, matching the session-transcript path.
- **Given** the same sub-agent transcript presented twice, **when** capture runs again, **then** the second capture is a no-op rather than a duplicate record.
- **Given** a sub-agent completion the harness reports without a readable transcript, **when** capture runs, **then** the miss is reported rather than failing silently or aborting the session.
- **Given** transcripts on disk that were never captured, **when** an operator ingests them for a repository, **then** they are redacted under that repository's own configuration and stored in that repository's corpus, and ingesting the same material twice adds nothing.
- **Given** a transcript whose repository cannot be identified, **when** ingest runs without configuration naming a destination for it, **then** it is skipped and reported rather than filed anywhere by guess.
- **Given** a repository configured to adopt a named orphaned project, **when** ingest runs, **then** that project's transcripts are stored in that repository's corpus and the adoption is recorded.
- **Given** a captured session with sub-agents, **when** an operator reconstructs it, **then** they receive one self-contained artefact containing the main thread and every sub-agent, in which each sub-agent's work is attributable to the point in the main thread that spawned it.
- **Given** a reconstructed session, **when** the artefact is produced, **then** a machine-readable telemetry file accompanies it reporting at least token usage, wall-clock duration, turn counts, tool-call counts by tool, models used and agent types.
- **Given** a reconstructed artefact, **when** it is handed to an agent as context, **then** it is readable without access to the original store or the harness's files.

## Open Questions

- Does the store's record schema gain explicit fields for the spawning session and the agent kind, or does the existing identifier carry them in composite form? Routed to an ADR refining adr-29. The third acceptance criterion depends on the answer but does not dictate it.
- Are the records already stored by hand under composite identifiers migrated onto whatever that ADR decides, or left in place and marked?
- Should capture happen at each sub-agent's completion, or be deferred to session end so one drain handles the whole session?
- How are sub-agents that ran concurrently ordered within a reconstruction, given that a strict linearisation misrepresents them?
- Does the telemetry file describe one session, or does a corpus-level roll-up across sessions belong here too?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect the harness's sub-agent completion event to carry the finished sub-agent's own transcript path, so capture reuses the existing stage-then-redact path unchanged and never reads the harness's undocumented directory layout, and we expect reconstruction and telemetry to need no new instrumentation because the raw transcripts already carry per-message token counts, timestamps, models, agent attribution and tool calls; it is shown wrong if the event fires before the sub-agent transcript is readable, if per-completion staging degrades a session that spawns many sub-agents, if the payload is absent on a supported harness version, or if the telemetry fields vary enough across versions that derived measures cannot be compared
