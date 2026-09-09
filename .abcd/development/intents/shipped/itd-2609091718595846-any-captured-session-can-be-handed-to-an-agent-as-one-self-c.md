---
id: itd-2609091718595846
slug: any-captured-session-can-be-handed-to-an-agent-as-one-self-c
spec_id: spc-2609091722269727
kind: standalone
suggested_kind: null
reclassification_history: []
related_adrs: [adr-29]
builds_on: [itd-2609090559376002]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Any Captured Session Can Be Handed to an Agent as One Self-Contained Artefact, With a Telemetry File Saying What It Cost

## Press Release

> **abcd renders one whole session — the main thread and every sub-agent it spawned — as a single file an agent can be handed as context, beside a machine-readable file describing what the work cost.** A session's transcripts are of interest as a set, not one at a time, and a corpus of scattered per-agent records is not a set until something assembles it. `abcd history reconstruct <session-id>` emits one Markdown artefact and one telemetry JSON. The artefact carries every turn, names each sub-agent twice in the thread that spawned it — spawned here, joined here — and leads with a timeline table so a reader can see which delegates overlapped instead of inferring an order the document never asserted. It names its records by basename and carries no absolute path, so it reads with the store gone. The telemetry reports span, turns, tokens, tool calls, models and agent types, per session and per agent, and states its own gaps.
>
> "Handing a model the session is the whole point — I want to ask what happened and have the answer be in the file, not in six places it cannot reach," said Maya, an autonomous-development practitioner. "And I want the numbers to be numbers. If the measurement cannot tell me what it was missing, I cannot compare two runs, and then it is decoration."

## Why This Matters

`itd-2609090559376002` fills the corpus and `itd-2609091718566731` recovers what was already there. Neither makes a session readable. What both leave is a per-agent archive: one record for the main thread and one for each sub-agent, each individually correct and collectively unassembled. The question a reader actually asks — what happened in this session, and which delegate did which part of it — cannot be answered by opening any one of them.

Two things follow from that, and they are different jobs sharing one pass over the same records.

**Reading.** The consumer is usually a model being handed the session as context, so the artefact is Markdown rather than a schema, self-contained rather than a set of pointers, and honest about what it could not establish. It also has to be safe: transcript text is content a session participant chose the bytes of, and a document that renders it raw can be made to grow a heading, a turn or a join marker that never happened. Fencing every block, and telling the reader that everything inside a fence is something somebody said, is what makes the structure the document's own rather than the transcript's.

**Measuring.** Every measure a study of this corpus would want is already in the raw lines — per-response token usage, timestamps, models, agent attribution, tool calls — and was being discarded along with them. Extracting it needs no new instrumentation, but it does need care: the harness writes one line per content block and repeats the same usage on every one, so summing lines counts a response once per block. On a real 55-record session, 3711 usage-bearing lines resolve to 1801 distinct responses — a naive sum would have reported that session as 2.06 times its actual cost, and no constant correction could fix it because the factor varies per session.

The measured shape of the delivered thing: that same 55-record session reconstructs to a 7.7 MB artefact plus 64 KB of telemetry in about 0.6 seconds, placing all 54 sub-agents at their spawn and join points.

## Typed Links

- **builds_on `itd-2609090559376002`** (sub-agent transcript capture): reads the lineage fields that intent put on the record — the spawning session, the agent id, the spawn depth and the spawning tool call — which are what let a set of per-agent records be assembled into one session at all.
- **refines `adr-29`** (native transcript corpus): a read-only consumer of the store that ADR established. It adds no write path and changes no record.

## What's In Scope

- **Reconstruction as an artefact:** any captured session can be emitted as one self-contained, agent-readable file containing the main thread and every sub-agent it spawned.
- **Attribution to the spawn point:** each sub-agent is tied to the point in the spawning thread that launched it and the point at which its result arrived, so a reader can tell what ran without it.
- **Concurrency represented rather than linearised:** the document states spans and spawn/join points and asserts nothing about time through section order.
- **Structure that transcript text cannot forge:** content is contained, the document's own asserted structures are enumerated for the reader, and the containment rule is stated in the artefact itself.
- **A telemetry file beside each reconstruction**, machine-readable, reporting token usage, wall-clock duration, turn and tool-call counts, models used and agent types, per session and per agent.
- **Tokens counted once per response**, not once per transcript line, with both counts published so a consumer can see the de-duplication happened.
- **A completeness block that states its own gaps** — an absent main thread, agents nothing could place, records found and not used, usage that could not be de-duplicated — because a derived measure that cannot say what it was missing must not be compared across runs.
- **Self-containment:** the artefact reads without the store and without the harness's files.

## What's Out of Scope

- **Capture and recovery.** Filling the corpus is `itd-2609090559376002`; recovering what was already on disk is `itd-2609091718566731`.
- **A corpus-level roll-up across sessions.** A stable per-session file has to exist first.
- **Structured extraction of findings.** Turning a review agent's transcript into structured findings is a separate concern from rendering the transcript.
- **Any write to the store.** Reconstruction reads records and writes only its two output files.
- **Treating telemetry as a billing record.** It describes what the harness recorded, and nothing reconciles it against a vendor's accounting.

## Mechanism

We expect reconstruction and telemetry to need no new instrumentation, because the raw transcripts already carry per-message token counts, timestamps, model identifiers, agent attribution and tool-call records, and the record's lineage fields already say which session and which agent each transcript belongs to. We expect a contiguous main thread with appended, doubly-marked sub-agent sections to read more truthfully than sections spliced in at their spawn points, because the sub-agents a spawning transcript can place are the asynchronous ones, whose spawn and join are many turns apart, so splicing would put a delegate's conclusions in front of main-thread turns that ran before those conclusions existed. This is shown wrong if the telemetry fields prove inconsistent enough across harness versions that derived measures cannot be compared, if a real session's artefact is too large to be handed to a model at all, or if the spawn and join points cannot be recovered often enough for the timeline to be worth reading.

## Scope Conditions

- Telemetry is descriptive of what the harness recorded and is not a billing record; token counts are as reported per response and may omit what the harness did not report. <!-- cond: cond-2609091722267847 -->
- Holds where a transcript line's token usage carries a response identifier. Usage without one cannot be de-duplicated, so the totals are then an upper bound to that extent, and the completeness block says by how much. <!-- cond: cond-2609091722264112 -->
- Holds for sessions at the observed working scale — up to roughly a hundred sub-agents and a few tens of megabytes of stored transcript per session — where one reconstruction is seconds rather than minutes. <!-- cond: cond-2609091722262463 -->
- Assumes the artefact's consumer is a model or a person, not a parser: the document is Markdown, and the line between what it asserts and what a participant said is drawn by containment and stated in the document, not by a schema. <!-- cond: cond-2609091722266189 -->
- Assumes a sub-agent's spawn point is recoverable from the record's stored spawning tool call, or failing that from the spawning transcript's own tool result. An agent neither can place is listed separately and labelled, never placed by guess. <!-- cond: cond-2609091722264395 -->

## Acceptance Criteria

- **Given** a captured session with sub-agents, **when** an operator reconstructs it, **then** they receive one self-contained artefact containing the main thread and every sub-agent, in which each sub-agent's work is attributable to the point in the main thread that spawned it.
- **Given** a reconstructed session, **when** the artefact is produced, **then** a machine-readable telemetry file accompanies it reporting at least token usage, wall-clock duration, turn counts, tool-call counts by tool, models used and agent types.
- **Given** a reconstructed artefact, **when** it is handed to an agent as context, **then** it is readable without access to the original store or the harness's files.

## Open Questions

- **Whether a corpus-level roll-up across sessions belongs here later.** The per-session telemetry file is the unit; whether anything aggregates it, and against what question, is not settled.
- **Whether the artefact needs a size answer beyond the two it has.** A reduced mode and a per-block cap bound the output today; whether a real session ever exceeds what a model can be handed even so is unmeasured.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-3b513d68dbd6 -->
Fidelity review OWED (receipt rcp-3b513d68dbd6).

## Grounds

- pursued: we expect reconstruction and telemetry to need no new instrumentation because the raw transcripts already carry per-response token usage, timestamps, models, agent attribution and tool calls, and we expect a contiguous main thread with appended, doubly-marked sub-agent sections to read more truthfully than sections spliced in at their spawn points, because the sub-agents a spawning transcript can place are the asynchronous ones whose spawn and join are many turns apart; it is shown wrong if the telemetry fields vary enough across harness versions that derived measures cannot be compared, if a real session's artefact is too large to be handed to a model even in its reduced form, or if spawn and join points cannot be recovered often enough for the timeline to be worth reading
