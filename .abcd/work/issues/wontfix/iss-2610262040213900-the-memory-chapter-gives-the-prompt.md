---
schema_version: 1
id: "iss-2610262040213900"
slug: "the-memory-chapter-gives-the-prompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 925 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:181"
remedy: "none (filed automatically)"
finding_ref: cfk-0d833404537bfe51
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038154778. Same End B and the same fix; 07-memory.md correctly names force-refresh-every-N, which internal/core/rules/inject.go implements."
duplicates: [iss-2610262038154778]
---

The memory chapter gives the prompt router a refresh every N prompts; the verification matrix says the refresh is event-driven and nothing counts prompts.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:181` — "a `UserPromptSubmit` hook with keyword recall, signature dedup, and force-refresh-every-N"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:73` — "The refresh is event-driven rather than turn-counted"

07-memory describes itd-3's prompt router as a UserPromptSubmit hook with force-refresh-every-N, which is a turn-counted refresh, and says itd-39 keeps the same refresh discipline. The matrix's rules-loader dedup row says the refresh is event-driven rather than turn-counted: SessionStart and PreCompact clear the ledger and nothing counts prompts. One router cannot have both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 925.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038154778. Same End B and the same fix; 07-memory.md correctly names force-refresh-every-N, which internal/core/rules/inject.go implements.
