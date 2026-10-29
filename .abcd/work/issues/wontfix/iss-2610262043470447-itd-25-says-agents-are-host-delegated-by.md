---
schema_version: 1
id: "iss-2610262043470447"
slug: "itd-25-says-agents-are-host-delegated-by"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1162 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:43"
remedy: "none (filed automatically)"
related_intents: [itd-25, itd-6]
finding_ref: cfk-be4985647b154de9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038572033. Same draft edit as iss-2610262038572033, set against a different statement of the same current state."
duplicates: [iss-2610262038572033]
---

itd-25 says agents are host-delegated by default with RepoPrompt opt-in; itd-6 keeps RP MCP as the preferred first backend in the oracle chain

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:43` — "Oracle backend chain (same as other agents): host-delegated by default, with the opt-in RepoPrompt adapter and Codex CLI as configured alternatives"
- End B: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:41` — "RP MCP (preferred) → Codex CLI subprocess (alternative for non-Mac / no-RP users — RP is macOS-only) → in-session subagent (final fallback)"

itd-25 says the oracle chain that agents share is host-delegated by default, with RepoPrompt and Codex CLI as opt-in alternatives. itd-6's scope keeps a three-transport chain where RP MCP is preferred and the in-session subagent is only the last fallback. The two records disagree on which backend an agent reaches first.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1162.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038572033. Same draft edit as iss-2610262038572033, set against a different statement of the same current state.
