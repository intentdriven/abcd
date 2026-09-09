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

# Every Sub-Agent a Session Spawns Leaves the Same Redacted Record Its Main Thread Does

## Press Release

> **abcd captures the reasoning of every sub-agent a session spawns, not just the session's main thread.** Today a session's own transcript is captured and redacted on write, but the sub-agents it spawns write their transcripts elsewhere and nothing collects them. What survives of a delegated task is the prompt that launched it and the summary it returned, while every command it ran, every file it edited and every judgement it made is dropped. Delegated work is the bulk of the work — better than three quarters of the recorded volume — so the store has been keeping the smaller half. This intent routes the sub-agent into the corpus that already exists, with the lineage that says which session and which agent produced it, at any nesting depth, under the same fail-closed redaction the session transcript already gets.
>
> "I delegate the reviews, the sweeps and most of the implementation to sub-agents, so that is where the reasoning I would want to audit actually lives," said Maya, an autonomous-development practitioner. "When I went back to reconstruct why a change was made, the store had the two lines where I handed the task off and the summary that came back, and nothing in between. The part I most needed to read was the part that was never kept."

## Why This Matters

abcd's transcript corpus exists so that work can be reconstructed after the fact. That promise is answered for the main thread and silently unanswered for everything delegated out of it:

- **The session transcript IS captured:** the session-end hook receives the path to it and stages it for redaction and storage.
- **Sub-agent transcripts are NOT:** they are written per sub-agent, and no hook abcd registers ever names them, so nothing enumerates or reads them.

The asymmetry is not marginal. On the measured corpus, sub-agent transcripts held roughly three quarters of all recorded bytes — 968 files and 673 MB against 68 session files and 206 MB — outweighing session transcripts by more than three to one. What the parent retains for each delegated task is the launch prompt and the returned report: in one measured case, two lines standing in for several hundred, discarding every tool call the agent made along the way.

This is a symmetry failure rather than a missing feature. The corpus already exists, already redacts on write, and already works for one caller class. The sub-agent caller was simply never routed into it. The harness offers a completion event for sub-agents whose payload names the finished sub-agent's own transcript, so the gap can be closed by feeding the existing store, with no new capture mechanism and no dependence on the harness's undocumented on-disk layout.

Two consequences make this urgent rather than tidy. Uncaptured transcripts age out under the harness's retention sweep, so the loss is permanent and ongoing. And the workaround already in use — inventing a composite identifier by hand — produces records that cannot be found from the identifier a reader actually holds: 176 of the store's 267 records carried one, and neither direction of the lookup worked.

Capturing the material is the load-bearing half. What is done with it afterwards — recovering the history that was never captured, and handing a whole session back as one readable artefact — rests on this and is separately delivered.

## Typed Links

- **refines `itd-59`** (autonomous-worker transcript capture): the adjacent half of the same corpus. itd-59 covers the worker on abcd's own run seam; this covers sub-agents spawned inside an interactive session.
- **refines `adr-29`** (native transcript corpus): feeds the store that ADR established rather than redesigning it.
- **corrects a premise in `itd-59`**, flagged for human confirmation and not auto-classified: itd-59's "What's Out of Scope" records interactive-session capture as already solved. That holds only of the main thread, and planning itd-59 on the unqualified claim would rebuild the same blind spot.
- **built on by `itd-2609091718566731`** (recovering transcripts already on disk) and **`itd-2609091718595846`** (reconstruction and telemetry): both consume the lineage this intent puts on a record, and neither is deliverable without it.

## What's In Scope

- **The capture gap:** every sub-agent transcript lands in the corpus under the repository it belongs to, with the same redaction and the same fail-closed refusal on a degraded scanner as a session transcript.
- **All sub-agent completions** at any nesting depth, including sub-agents spawned by other sub-agents.
- **Provenance enough to read the result back:** a stored sub-agent transcript can be traced from the identifier of the session that spawned it, and says what kind of agent produced it and which rung of the attribution ladder placed it.
- **Idempotence:** the same sub-agent transcript presented twice is one record, and two sub-agents whose transcripts happen to be byte-identical are still two.
- **A completion that yields nothing is reported**, on stderr and on disk, and never stalls or fails the session it fired inside.
- **Feed, don't fork:** sub-agent capture reuses the existing staging and redaction path rather than adding a second mechanism beside it.

## What's Out of Scope

- **Recovering history that already exists.** Transcripts on disk that were never captured, and records already filed under a composite identifier, are `itd-2609091718566731`.
- **Reconstruction and telemetry.** Emitting a session as one artefact with a machine-readable measurement file is `itd-2609091718595846`.
- **Redesigning the corpus.** The store's per-repo keying and its single-owner provisioning stay as they are.
- **The harness's own retention policy.** How long the harness keeps its transcripts is configuration outside abcd's control.
- **Structured extraction of findings.** Turning a review agent's transcript into structured findings is a separate concern from keeping the transcript.

## Mechanism

We expect routing sub-agent capture through the harness's sub-agent completion event to close the gap without new machinery, because that event's payload names the finished sub-agent's own transcript path directly, so the existing stage-then-redact path can consume it unchanged and nothing needs to read the harness's undocumented directory layout. We expect the lineage a reader needs to fit in explicit record fields rather than a composite identifier, because the payload carries the spawning session untruncated and the agent's own id separately, and a field cannot be lossy in the way a concatenation is. This is shown wrong if the completion event fires before the sub-agent transcript is flushed and readable, if a session spawning many sub-agents degrades under per-completion staging, or if the payload is absent on a harness version abcd claims to support.

## Scope Conditions

- Holds for harness versions whose sub-agent completion event carries the finished sub-agent's transcript path; a version without it falls back to no capture rather than to guessing at the on-disk layout, and records that it did so. <!-- cond: cond-2609090624228012 -->
- Holds at the observed working scale of a few hundred sub-agents per repository per month, with individual transcripts up to a few megabytes and sessions up to roughly a hundred sub-agents. <!-- cond: cond-2609090624226418 -->
- Assumes sub-agent transcripts share the line-delimited shape the session transcript already uses, so one reader serves both. <!-- cond: cond-2609090624223171 -->
- Assumes a transcript's owning repository is determined by the working directory the payload reports, or failing that by the store that has already seen the spawning session — never by decoding the harness's project directory name, which is not reversible. <!-- cond: cond-2609090624228630 -->
- Assumes a session run in a worktree belongs to the store of the repository that worktree derives from, and that the worktree may already be gone by the time the completion event fires. <!-- cond: cond-2609090624222424 -->
- **Whether the completion event fires before the harness has flushed the sub-agent's transcript is UNVERIFIED.** The mitigations shipped — a bounded stage-time wait on an incomplete final line, and a drain-time re-read that replaces the staged bytes only when the source strictly extends them — bound the damage and count their own effect, but they do not settle the question, and the residual rate has not yet been measured over a corpus. <!-- cond: cond-2609091722261577 -->

## Acceptance Criteria

- **Given** a session that spawns a sub-agent, **when** that sub-agent finishes, **then** its transcript is stored in the corpus for the session's repository, redacted on write, and listed by `abcd history`.
- **Given** a sub-agent that itself spawns a sub-agent, **when** both finish, **then** both transcripts are stored and each is attributable to the session that spawned it.
- **Given** a stored sub-agent transcript, **when** an operator looks it up from the identifier of the session that spawned it, **then** they can reach it and can tell what kind of agent produced it.
- **Given** a repository whose secret scanner is degraded, **when** a sub-agent transcript would be captured, **then** the capture refuses rather than storing under weakened redaction, matching the session-transcript path.
- **Given** the same sub-agent transcript presented twice, **when** capture runs again, **then** the second capture is a no-op rather than a duplicate record.
- **Given** a sub-agent completion the harness reports without a readable transcript, **when** capture runs, **then** the miss is reported rather than failing silently or aborting the session.

## Open Questions

- **The flush race is still open.** The mitigations are in and self-counting (`DrainResult.Extended` is the count of transcripts caught short and completed), but the rate over a real corpus has not been measured, so the scope condition above stands as written rather than resolved.
- **The decision record refining adr-29 is deferred and unminted.** The move from a composite identifier to explicit lineage fields deserves an ADR; until it is minted, `spc-2609090624222051` is the decision of record. Two committed surfaces disagree about how an ADR id is allocated (the root router says hand-numbered and coordinated; the ADR store's charter records the 2026-09-01 ruling that `abcd decide` mints through the same timestamp-numeric seam as every other family), and that disagreement is not this intent's to settle — but it blocks the minting.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-e6f812226e15 -->
Fidelity review OWED (receipt rcp-e6f812226e15).

## Grounds

- pursued: we expect the harness's sub-agent completion event to carry the finished sub-agent's own transcript path, so capture reuses the existing stage-then-redact path unchanged and never reads the harness's undocumented directory layout, and we expect reconstruction and telemetry to need no new instrumentation because the raw transcripts already carry per-message token counts, timestamps, models, agent attribution and tool calls; it is shown wrong if the event fires before the sub-agent transcript is readable, if per-completion staging degrades a session that spawns many sub-agents, if the payload is absent on a supported harness version, or if the telemetry fields vary enough across versions that derived measures cannot be compared
- pursued: narrowed to capture alone after the intent was split in three — we expect the harness's sub-agent completion event to carry the finished sub-agent's own transcript path, so capture reuses the existing stage-then-redact path unchanged and never reads the harness's undocumented directory layout, and we expect explicit lineage fields on the record to carry what a composite session id could not, because the payload delivers the spawning session untruncated and the agent id separately; it is shown wrong if the event fires before the sub-agent transcript is readable at a rate the two mitigations cannot absorb, if per-completion staging degrades a session that spawns many sub-agents, or if the payload is absent on a supported harness version
