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
---

itd-113 calls MCP abcd's second front door; 03-configuration calls an mcp/ door adr-23's third

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-113-abcd-s-core-verbs-are-reachable-over-mcp.md:16` — "abcd's second front door is open: MCP."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1275` — "(an mcp/ door is adr-23's third)"

itd-113 announces MCP as abcd's second front door. 03-configuration's layout calls an mcp/ door adr-23's third. The two documents count the front doors differently.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 836.
