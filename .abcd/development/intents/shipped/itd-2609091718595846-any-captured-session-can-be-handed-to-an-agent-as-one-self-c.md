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

> **abcd renders one whole session — the main thread and every sub-agent it spawned — as a single file an agent can be handed as context, beside a machine-readable file describing what the work cost.** A session's transcripts are of interest as a set, not one at a time, and a corpus of scattered per-agent records is not a set until something assembles it. `abcd history reconstruct <session-id>` emits one Markdown artefact and one telemetry JSON. The artefact carries every turn, names each sub-agent twice in the thread that spawned it — spawned here, joined here — and leads with a timeline table so a reader can see which delegates overlapped instead of inferring an order the document never asserted. It names its records by basename and emits no path of its own, so it reads with the store gone, while the turns it quotes keep whatever paths were spoken in them, because editing somebody's recorded words to tidy a path would falsify the record. The telemetry reports span, turns, tokens, tool calls, models and agent types, per session and per agent, and states its own gaps.
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

<!-- abcd-review: INGESTED receipt=rcp-3b513d68dbd6 -->
Fidelity review — receipt rcp-3b513d68dbd6 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:1cac37e1eae2c1aaf9bd6d68cae2bd5982b0c8510b3e65abcc50e3fd7d354b3a
Input attestations: diff:319da670..d751109c over internal/core/history/reconstruct{,_render,_test}.go and internal/surface/cli/history_reconstruct{,_test}.go (commit 9af9a30a principally, plus the anti-forgery part of d751109c)@sha256:7fae363926bc98ce56325a25fe491aef31e39411e0ad2ea2a9ac0b55f23cc4e7; run:read-only `history reconstruct` over the live store: sessions 14e2fa13 (55 records, 7.7 MB, 0.7s), 6d426540 (99 agents, 18.8 MB, 2.0s), db0f4683 (72 sub-agents), 33bd3e3c@-;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: One artefact holds the main thread and every stored sub-agent, and each placed sub-agent is marked at the exact turn of the spawning thread that launched it (SPAWNED marker), at its join turn, in a per-agent provenance block and in the head timeline table; on session 14e2fa13 all 54 sub-agents were placed and 54 SPAWNED and 54 JOINED markers appear. The appended-rather-than-nested form satisfies the criterion AS WRITTEN — the words require the work to be ATTRIBUTABLE to the spawn point, not rendered at it, and the marker names that turn precisely. Concerns: (a) attribution is not universal — on session db0f4683, 28 of 72 sub-agents had no recoverable spawn point and were segregated under '## Unattributed sub-agents'; (b) for depth>1 agents the spawn turn is in the parent agent's thread, not 'the main thread' the criterion names; (c) the attribution structure itself is forgeable — a transcript's tool_use.name, tool_result.tool_use_id or message.model is interpolated OUTSIDE any fence, and a value carrying newlines emits '## Agent `ffffffff`', '### Turn 99 — assistant' and a '[JOINED …]' marker byte-for-byte as lines of the document (reproduced on a scratch copy of HEAD).
  evidence: internal/core/history/reconstruct_render.go:282 — "fmt.Fprintf(b, "\n> **[SPAWNED** agent `%s` (%s) here — its transcript is in section \"Agent `%s`\". "+"
  evidence: internal/core/history/reconstruct.go:801 — "func locateSpawn(host, sub *thread) (int, string, string) {"
  evidence: internal/core/history/reconstruct_render.go:75 — "fmt.Fprintf(&body, "\n## Unattributed sub-agents\n\nThe %d agent(s) below belong to this "+"
  evidence: internal/core/history/reconstruct_render.go:345 — "fmt.Fprintf(b, "\n**tool call** `%s`", orDash(blk.Name))"
- ac-2 — MET: `<session>.telemetry.json` is written beside the artefact by the CLI and carries every measure the criterion names — tokens, wall_clock_seconds, turns, tool_calls keyed by tool name, models and agent_types — per session and again per agent, plus a completeness block; measured on session 14e2fa13 it reported tokens 374419512 over 1801 api_responses against 3711 usage_lines_seen, tool_calls {Bash:1613, Read:208, Agent:87, …}, models [< synthetic>, claude-opus-5, claude-sonnet-5] and 7 agent_types. Tokens are de-duplicated once per distinct message id inside a whole-thread map, not summed per line: the measured inflation the naive sum would have produced was 2.06x on 14e2fa13 and 2.00x / 1.91x / 1.78x on three further real sessions, confirming the factor varies per session and both counters are published side by side so a consumer can see the de-duplication happened.
  evidence: internal/core/history/reconstruct.go:596 — "case seenUsage[key]:"
  evidence: internal/core/history/reconstruct.go:608 — "t.tokens.APIResponses++"
  evidence: internal/core/history/reconstruct.go:282 — "Turns TurnCounts `json:"turns"`"
  evidence: internal/core/history/reconstruct.go:284 — "ToolCalls map[string]int `json:"tool_calls"`"
  evidence: internal/surface/cli/history_reconstruct.go:111 — "if err := fsutil.WriteFileAtomic(telemetryPath, tel, 0o644); err != nil {"
- ac-3 — MET_WITH_CONCERNS: The artefact inlines every turn of every stored thread, names records by BASENAME only, and leads with a header, a 'How to read this document' guide, a completeness block and a timeline, so a reader needs neither the store nor the harness to read it; the 7.7 MB artefact from session 14e2fa13 carries no store path and no store root, and it carries no generation timestamp so the same records render to identical bytes. The outcome the criterion states therefore holds. Concerns: (a) the intent's own ground for it is false of the delivered artefact — the press release says it 'carries no absolute path' and the spec repeats 'no absolute path of any kind', while the real artefact contains 14 '/Users/…' and 1488 '/private/…' occurrences inside reproduced transcript text (a defensible fidelity choice, but neither record states it, and no code comment records the reasoning either); (b) nothing detects the overclaim — TestReconstructionIsSelfContained asserts only the store root, the '.abcd/history'/'transcripts/' path shapes and record basenames; (c) the guide the self-explaining reader acts on states 'everything outside one is this document', which the metadata-forgery hole under ac-1 makes untrue.
  evidence: internal/core/history/reconstruct.go:174 — "// Record is the record's BASENAME. Never a path: the artefact and its"
  evidence: internal/core/history/reconstruct_render.go:133 — "b.WriteString("\n## How to read this document\n\n")"
  evidence: internal/core/history/reconstruct_test.go:522 — "if strings.Contains(art, ".abcd/history") || strings.Contains(art, "transcripts/") {"
  evidence: .abcd/development/specs/closed/spc-2609091722269727-any-captured-session-can-be-handed-to-an-agent-as-one-self-c.md:159 — "harness path and no absolute path of any kind, and the CLI additionally"

Gap audit:
- honoured:
  - One wired verb emits one Markdown artefact and one telemetry JSON per session, from a core that writes nothing and knows no path
    evidence: internal/surface/cli/history.go:437 — "historyCmd.AddCommand(newHistoryReconstructCommand(asJSON))"
    evidence: internal/surface/cli/history_reconstruct.go:98 — "func writeReconstruction(dir string, res history.Reconstruction) ([]string, error) {"
  - Tokens counted once per response, not once per transcript line, with both counts published so the de-duplication is visible
    evidence: internal/core/history/reconstruct.go:587 — "t.tokens.UsageLinesSeen++"
    evidence: internal/core/history/reconstruct.go:596 — "case seenUsage[key]:"
  - Concurrency is represented rather than linearised: a timeline table carries the spans, section order asserts nothing about time, and a CONCURRENCY line counts the spawning turns that ran without the delegate's result
    evidence: internal/core/history/reconstruct_render.go:193 — "| agent | type | depth | parent | spawned | started | ended | joined | turns | tokens |"
    evidence: internal/core/history/reconstruct_render.go:222 — "- CONCURRENCY: %d turn(s) of `%s` ran between the spawn and the join, and none of them had this agent's result"
  - A completeness block that states its own gaps — absent main thread, unplaceable agents, records found and not used, unparseable lines, un-de-duplicable usage, absent field names, elisions
    evidence: internal/core/history/reconstruct.go:223 — "type Completeness struct {"
    evidence: internal/core/history/reconstruct_render.go:166 — "- sub-agents with no recoverable spawn point: %d of %d"
  - An agent nothing can place is listed separately and labelled, never placed by guess; a named-but-absent parent is not silently replaced by the main thread
    evidence: internal/core/history/reconstruct.go:780 — "host = nil"
    evidence: internal/core/history/reconstruct_render.go:75 — "## Unattributed sub-agents"
  - Both size answers shipped and hold at the observed working scale: spine mode with counted gap markers and an 8 KiB per-block cap that marks and counts what it removes
    evidence: internal/core/history/reconstruct_render.go:367 — "func (r *renderer) cap(s string) string {"
    evidence: internal/surface/cli/history_reconstruct.go:35 — "const defaultMaxBlockBytes = 8 << 10"
- diverged:
  - Sub-agent sections nested at their spawn points by spawn_depth — delivered instead as a contiguous main thread with appended sections, twin inline markers and a timeline table. This is a signed-off reversal against the PLAN, argued from the corpus, and it does not diverge from ac-1's words, which ask for attributability rather than placement.
    evidence: internal/core/history/reconstruct.go:12 — "// The spec calls for each sub-agent's section to be nested at its spawn point."
    evidence: .abcd/development/specs/closed/spc-2609091722269727-any-captured-session-can-be-handed-to-an-agent-as-one-self-c.md:106 — "### The layout: appended sections, doubly marked — a reversal"
  - 'Structure that transcript text cannot forge' — containment is applied uniformly to block CONTENT but not to block METADATA. tool_use.name, tool_result.tool_use_id and message.model are interpolated outside any fence; a value carrying newlines emits the document's own '## Agent `<id>`', '### Turn < n> — …' and '[JOINED …]' lines verbatim outside a fence, which is the exact class d751109c set out to close. Verified on a scratch copy of HEAD.
    evidence: internal/core/history/reconstruct_render.go:345 — "fmt.Fprintf(b, "\n**tool call** `%s`", orDash(blk.Name))"
    evidence: internal/core/history/reconstruct_render.go:352 — "fmt.Fprintf(b, "\n**tool result** for `%s`\n\n", orDash(blk.ToolUseID))"
    evidence: internal/core/history/reconstruct_render.go:306 — "fmt.Fprintf(b, " · %s", t.model)"
  - The reader-facing guide states an unconditional rule — 'Everything inside a fence is something somebody said; everything outside one is this document' — which is not the rule that shipped: the metadata fields above are outside every fence and are somebody's bytes.
    evidence: internal/core/history/reconstruct_render.go:151 — ""**Everything inside a fence is something somebody said; everything outside one is this " +"
  - 'It names its records by basename and carries no absolute path' — records are basenames, but the artefact does carry absolute paths inside reproduced transcript text (measured: 14 '/Users/…' and 1488 '/private/…' in the 14e2fa13 artefact). Neither the intent nor the spec records the fidelity reason for keeping them.
    evidence: .abcd/development/intents/shipped/itd-2609091718595846-any-captured-session-can-be-handed-to-an-agent-as-one-self-c.md:20 — "It names its records by basename and emits no path of its own, so it reads with the store gone, while the turns it quotes keep whatever paths were spoken in them, because editing somebody's recorded words to tidy a path would falsify the record."
    evidence: internal/core/history/reconstruct_render.go:337 — "writeFenced(b, "", blk.Text)"
  - Reconstruct(rootSHA, sessionID) grew into an options struct carrying Mode and MaxBlockBytes — a documented, measured change of signature rather than a silent one.
    evidence: internal/core/history/reconstruct.go:93 — "type ReconstructOptions struct {"
- missing:
  - No detector for the metadata-forgery class: TestReconstructCannotBeForgedByTranscriptText plants a forged structure only in a text block's text, so nothing exercises tool_use.name, tool_use.id, tool_result.tool_use_id, message.model or an unknown block's type.
    evidence: internal/core/history/reconstruct_test.go:734 — ""content": []map[string]any{{"type": "text", "text": forgedStructure}},"
  - No detector for the record's stated absolute-path property: the self-containment test bounds the store root and record basenames only, so the intent's and spec's 'no absolute path of any kind' claim is asserted by prose and checked by nothing.
    evidence: internal/core/history/reconstruct_test.go:519 — "if strings.Contains(art, home) {"

Scope-condition dispositions:
- cond-2609091722267847 — survived: The telemetry carries token counters and no monetary or cost field, reconciles against nothing, and every total is paired with a completeness block that says what it was missing — descriptive of what the harness recorded, exactly as assumed.
  evidence: internal/core/history/reconstruct.go:131 — "type TokenCounts struct {"
  evidence: internal/core/history/reconstruct.go:969 — "if comp.UsageWithoutMessageID > 0 {"
- cond-2609091722264112 — survived: Usage with no message id is counted rather than dropped, tallied into completeness.usage_without_message_id, and given a note stating the totals are an upper bound to that extent; the condition's stated exception is implemented and tested, though on all four real sessions I measured the counter was zero, so only the fixture exercised it.
  evidence: internal/core/history/reconstruct.go:595 — "t.noMsgID++"
  evidence: internal/core/history/reconstruct.go:971 — "%d response(s) carried usage with no message id, so their usage could not be de-duplicated; the token totals are an upper bound to that extent"
- cond-2609091722262463 — survived: Measured at the top of the stated range: 55 records to a 7.7 MB artefact in 0.7s, and 99 agents to an 18.8 MB artefact in 2.0s — seconds rather than minutes, at roughly a hundred sub-agents.
  evidence: internal/core/history/reconstruct.go:34 — "// A single unbounded artefact is not usable for the consumer it is for."
  evidence: internal/surface/cli/history_reconstruct.go:162 — "fmt.Fprintf(w, " artefact: %s\n", humanBytes(res.ArtefactBytes))"
- cond-2609091722266189 — narrowed: The document is Markdown for a model or a person and does state the containment rule in words, but the containment it states does not cover every byte the document places outside a fence: block metadata is interpolated raw, so the stated line between assertion and quotation is drawn in a different place from where the guide says it is.
  narrowing: The containment rule holds for block CONTENT — text, thinking, tool input and tool result body, each in a dynamically sized fence — and not for block METADATA (tool_use.name, tool_use.id, tool_result.tool_use_id, message.model, an unknown block's type), which is written outside every fence and can therefore emit the document's own headings and markers.
  evidence: internal/core/history/reconstruct_render.go:381 — "func writeFenced(b *strings.Builder, lang, body string) {"
  evidence: internal/core/history/reconstruct_render.go:345 — "fmt.Fprintf(b, "\n**tool call** `%s`", orDash(blk.Name))"
- cond-2609091722264395 — narrowed: Both rungs shipped and the fallback shipped with them — an agent neither rung places is listed under '## Unattributed sub-agents', labelled and counted, never placed by guess — but the recoverability the condition assumes is materially rarer than the intent's one measured session showed.
  narrowing: Holds fully on sessions whose spawning tool call is stored or whose spawning transcript names the agent (0 of 54, 0 of 64 and 0 of 98 unplaceable on three sessions I ran); on session db0f4683 28 of 72 sub-agents had no recoverable spawn point at all, so for roughly two fifths of that session the condition's escape clause, not its assumption, is what carried the artefact.
  evidence: internal/core/history/reconstruct.go:830 — "return cand.index, b.ToolUseID, "transcript""
  evidence: internal/core/history/reconstruct_render.go:217 — "b.WriteString("- spawned at: NOT RECOVERABLE from what is stored\n")"
## Grounds

- pursued: we expect reconstruction and telemetry to need no new instrumentation because the raw transcripts already carry per-response token usage, timestamps, models, agent attribution and tool calls, and we expect a contiguous main thread with appended, doubly-marked sub-agent sections to read more truthfully than sections spliced in at their spawn points, because the sub-agents a spawning transcript can place are the asynchronous ones whose spawn and join are many turns apart; it is shown wrong if the telemetry fields vary enough across harness versions that derived measures cannot be compared, if a real session's artefact is too large to be handed to a model even in its reduced form, or if spawn and join points cannot be recovered often enough for the timeline to be worth reading
