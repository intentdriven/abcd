---
schema_version: 1
id: "iss-2610262036303950"
slug: "itd-104-says-an-idea-is-gated-before-it"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 653 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-104]
finding_ref: cfk-ce33ea60f3b54291
---

itd-104 says an idea is gated before it can mint any record; the ideate chapter says ideate is optional, never a gate, and required by no other verb

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes.md:17` — "abcd gates a new idea before it can mint a single record"
- End B: `.abcd/development/brief/04-surfaces/18-ideate.md:13` — "No other verb requires it, nothing warns when it is skipped"

itd-104's press release makes the gauntlet a precondition: an idea is gated before it can mint a single record. 18-ideate says ideate is optional and never a gate, that no other verb requires it and nothing warns when it is skipped. An idea cannot be both barred from minting a record until ideated and free to be captured without it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 653.
