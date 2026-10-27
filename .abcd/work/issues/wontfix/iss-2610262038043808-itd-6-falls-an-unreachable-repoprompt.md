---
schema_version: 1
id: "iss-2610262038043808"
slug: "itd-6-falls-an-unreachable-repoprompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 769 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:42"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-51f55dfcbe265c84
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The Codex-then-subagent cascade is dropped by itd-6 itself; its Acceptance Criteria state the host fallback that 34-build and the loop code (fallback receipt, ran host) implement."
---

itd-6 falls an unreachable RepoPrompt through Codex and then an in-session subagent; 34-build falls a review the rp route does not return straight back to the host

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:42` — "if RP MCP is unreachable, abcd falls through to Codex if configured, then in-session subagent. Three-step cascade."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:566` — "the call records one fallback receipt (the role, `asked: rp`, the reason, abcd's detail, `ran: host`) in the state and the record"

itd-6 states a three-step cascade when RP MCP is unreachable: Codex if configured, then an in-session subagent. 34-build states that a review the `rp` route does not return records one fallback receipt with `ran: host` and hands the host the same brief and return path, with no Codex step between. The two cannot both describe what happens when RepoPrompt cannot be reached.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 769.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The Codex-then-subagent cascade is dropped by itd-6 itself; its Acceptance Criteria state the host fallback that 34-build and the loop code (fallback receipt, ran host) implement.
