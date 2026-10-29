---
schema_version: 1
id: "iss-2610262043485833"
slug: "itd-19-sets-the-oracle-backend-from-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1163 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-19-stage-aware-behaviour.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-19, itd-6]
finding_ref: cfk-7ddc6f90a3aef2f1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037211310. Same draft edit as iss-2610262037211310, set against a different statement of the same current state."
duplicates: [iss-2610262037211310]
---

itd-19 sets the oracle backend from the project's stage; itd-6 has ahoy lock it based on whether RP is reachable

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-19-stage-aware-behaviour.md:31` — "oracle=in-session (fast iteration)"
- End B: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:43` — "If reachable: lock `oracle.backend = "rp"`. If not: lock `oracle.backend = "codex"` (if Codex CLI present)"

itd-19's stage profiles have ahoy set oracle=in-session for an Autonomous project, and rp or rp+codex for other stages. itd-6 has ahoy test whether RP is reachable and lock oracle.backend to rp when RP answers, falling back to codex or in-session only when it does not. For an Autonomous project where RP is reachable, one record writes in-session and the other locks rp.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1163.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037211310. Same draft edit as iss-2610262037211310, set against a different statement of the same current state.
