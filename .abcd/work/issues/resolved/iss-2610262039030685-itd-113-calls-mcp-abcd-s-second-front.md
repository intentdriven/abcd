---
schema_version: 1
id: "iss-2610262039030685"
slug: "itd-113-calls-mcp-abcd-s-second-front"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 836 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-113-abcd-s-core-verbs-are-reachable-over-mcp.md:16"
remedy: "Drop the ordinal in itd-113 ('second front door' becomes 'the MCP front door') and in 03-configuration.md say an mcp door is one of the later doors adr-23 names, not 'adr-23's third'"
related_intents: [itd-113]
finding_ref: cfk-5ef4898f3187c41e
resolution: "itd-113 no longer counts the MCP door: its press release says 'abcd's MCP front door is open' and its Why says 'one of the later thin front doors', as adr-23 names it. The configuration chapter's layout already called it 'adr-23's mcp/ door' with no ordinal at the lane's base, so it was left unchanged. A reachaudit test refuses an ordinal front door on either page."
impact: internal
resolved_by:
  commit: "44cb11124ad59d2b0eb4b706695b151fdfad0c19"
---

itd-113 calls MCP abcd's second front door; 03-configuration calls an mcp/ door adr-23's third

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-113-abcd-s-core-verbs-are-reachable-over-mcp.md:16` — "abcd's second front door is open: MCP."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1275` — "(an mcp/ door is adr-23's third)"

itd-113 announces MCP as abcd's second front door. 03-configuration's layout calls an mcp/ door adr-23's third. The two documents count the front doors differently.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 836.

## Grounds

- pursued: no page counts the MCP door by position, so itd-113 and 03-configuration agree with adr-23's 'later front doors'; TestTheMCPDoorIsNotCountedByPosition, or a consistency pass that finds another ordinal front door on either page, would show it wrong
