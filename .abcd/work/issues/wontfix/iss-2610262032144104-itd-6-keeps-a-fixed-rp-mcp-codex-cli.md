---
schema_version: 1
id: "iss-2610262032144104"
slug: "itd-6-keeps-a-fixed-rp-mcp-codex-cli"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 240 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-d763b129b9df43dc
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The quoted cascade sentence sits in itd-6's original scope, which the record's own re-file note says is dropped and to be read through the re-file and Decisions; no fixed cascade exists in internal/core/oracle or the adapters."
---

itd-6 keeps a fixed RP MCP → Codex CLI → in-session cascade; the dependencies page says no such fixed cascade exists and that RepoPrompt, codex and a direct API oracle are opt-in adapters

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:55` — "The cascade defined by this intent (RP MCP → Codex CLI → in-session subagent) is and stays availability-driven"
- End B: `.abcd/development/brief/02-constraints/02-dependencies.md:34` — "RepoPrompt and codex are two such adapters; a **direct API oracle is a legitimate adapter** too. There is no fixed RP→codex→in-session cascade."

itd-6, a shipped intent, says the cascade it defines (RP MCP → Codex CLI → in-session subagent) 'is and stays availability-driven'. 02-dependencies says RepoPrompt and codex are two opt-in adapters, that a direct API oracle is a legitimate adapter, and that there is no fixed RP→codex→in-session cascade. The oracle architecture cannot both keep that cascade and have none, and a builder following itd-6's scope would build the routing that the constraints chapter says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 240.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The quoted cascade sentence sits in itd-6's original scope, which the record's own re-file note says is dropped and to be read through the re-file and Decisions; no fixed cascade exists in internal/core/oracle or the adapters.
