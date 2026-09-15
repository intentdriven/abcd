---
schema_version: 1
id: "iss-2608290819228175"
slug: "a-decision-recorded-in-an-intent-an-adr-or-the-decision-log"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "intent-implementation-run"
found_at: "internal/core/history"
promoted_to: itd-171
---

A decision recorded in an intent, an ADR or the decision log should point at the section of the redacted transcript where it was actually made, so the reasoning behind a record is recoverable rather than lost with the session that produced it. This is the Naur point the lifeboat work already rests on: the record is the floor of recoverable theory, not the theory itself, and today nothing leads from a decision back to the conversation that reached it. The store is closer to this than it first appears. A transcript record already carries session_id, a source_sha256 content hash, and captured_at as an RFC3339 nanosecond timestamp, so the SESSION is timestamped and content-addressed already. What is missing is granularity inside one transcript: there is no per-turn or per-section anchor, so a link can name the session but not the passage. Two constraints shape the design and both rule out the obvious approach. A line number is the wrong anchor, and the repository already refuses that shape elsewhere through the no-brittle-line-refs rule, which tells an author to use a section anchor instead. And redaction rewrites spans in place, so any byte offset or line number computed against the raw transcript is invalid against the stored redacted one: an anchor has to be derived from the redacted artefact, or be content-addressed rather than positional. The plausible shape is therefore a typed cross-reference carrying session_id plus source_sha256 plus a stable intra-transcript anchor, minted when the decision is recorded rather than reconstructed later, with per-turn timestamps or ids added to the stored transcript so such an anchor exists to name. Worth deciding whether the anchor is a turn id, a content hash of the passage, or a heading, and whether the link is stored on the decision record, in the transcript, or both.