---
schema_version: 1
id: "iss-2610262040062985"
slug: "the-readme-s-hook-row-says-session-end"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 909 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:276"
remedy: "none (filed automatically)"
related_intents: [itd-89]
finding_ref: cfk-3511312ae310f5f9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The resolved issue states the split explicitly and the code (hook session-end, history chapter) matches the register."
---

The README's hook row says session-end only stages the transcript and session-start files it away; itd-89 says session-end calls history.Capture, which redacts and files it

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:276` — "`session-end` stages the session's own transcript"
- End B: `.abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:36` — "takes `session_id` and `transcript_path`, and calls `history.Capture` (which already redacts)"

The operator-internal verbs table says `session-end` stages the session's own transcript and `session-start` files both staged transcripts away. itd-89 says the session-end verb takes the transcript path and calls history.Capture, which redacts the transcript, so the session is stored as it ends. The two put the redact-and-file step on different hook events.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 909.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The resolved issue states the split explicitly and the code (hook session-end, history chapter) matches the register.
