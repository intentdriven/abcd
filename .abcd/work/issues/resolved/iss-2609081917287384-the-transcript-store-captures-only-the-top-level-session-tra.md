---
schema_version: 1
id: "iss-2609081917287384"
slug: "the-transcript-store-captures-only-the-top-level-session-tra"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "sub-agent transcript capture audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
promoted_to: itd-2609090559376002
resolution: "Sub-agent transcripts are captured through the harness's completion event into the same redact-on-write store the main thread already used, with lineage carried in explicit record fields rather than an overloaded identifier. The history already on disk is recovered into the repository that owns it, under that repository's own redaction configuration, and a session can be emitted as one self-contained artefact with a telemetry file. Applied to this machine: 176 records repaired out of their composite identifiers, 1066 transcripts ingested, the store grown from 267 records to 1104, and 13 transcripts refused by fail-closed redaction over network addresses it could not redact. The work shipped as three intents rather than the one this issue was promoted into."
impact: additive
resolved_by:
  intent: "itd-2609090559376002"
  spec: "spc-2609090624222051"
---

The transcript store captures only the top-level session transcript; every sub-agent transcript is missed, which is 77 percent of recorded work by volume. The SessionEnd hook reads the single transcript_path handed to it by the host and never enumerates anything, and hooks.json registers no sub-agent event. Sub-agent transcripts are written by the harness to a sibling directory per session rather than inlined into the parent, so no parent transcript contains them: across all 68 session transcripts on this machine the sidechain marker appears zero times, while 968 sub-agent transcript files hold 673 MB against the parents' 206 MB. What the parent retains per sub-agent is only the launch prompt and the returned report; in one sampled case that is two lines standing in for 399, losing every tool call the agent made. The harness ships a SubagentStop hook event whose payload carries agent_transcript_path, agent_id and agent_type alongside the parent session_id, so the capture path can be closed without depending on the undocumented on-disk layout. Storing the result needs a lineage decision the Record schema cannot currently express: it has no parent, agent or type field, and source_kind is closed to native and specstory-import, so the only representable form is overloading session_id with a composite. This store already holds 105 such hand-made composite records, and because Read matches session_id by exact string they are unreachable from a show of their parent session and their agent type is discarded. The loss was time-bounded until now: the harness deletes transcripts on a rolling retention sweep, so material aged out before anything captured it.

## Grounds

- pursued: we expect the SubagentStop payload's agent_transcript_path to let sub-agent transcripts reuse the existing Stage/Drain redact-on-write path unchanged, closing the gap with no new capture mechanism and no dependence on the undocumented on-disk layout; it is shown wrong if the hook fires before the sub-agent transcript is flushed and readable, if a session spawning many sub-agents degrades under per-completion staging, or if the payload proves absent on any supported harness version
- pursued: we expect the completion event's own payload to be sufficient for capture, so no part of this depends on reading the harness's undocumented directory layout, and we expect the store to be the right home because reconstruction and telemetry needed no schema change beyond lineage; it is shown wrong if the flush race proves to have a material residual rate, which is armed and counted but still unmeasured, or if a harness version ships without the payload field the capture path rests on
