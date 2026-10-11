---
schema_version: 1
id: "iss-2609240227150859"
slug: "itd-6-implementation-status-describes-an"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A, planning briefs"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-6-rp-mcp-only-integration.md"
resolution: "itd-6's Status paragraph, Resolved (post-spc-5) section and Implementation status say no part of the RP MCP route is built: RPUnavailable, MCPBridge and oracle.py belong to an earlier Python lineage's spc-5 and ADR-02/03, the settled answers stand as design input only, the Codex fall-through defers to the re-filed Decisions, and spc-2609211950427074 carries all four re-filed criteria. No gate would have caught it: no lint compares an intent's implementation claims with the tree."
impact: internal
resolved_by:
  commit: "3468a4bae"
---

itd-6's Implementation status and its post-spc-5 Status paragraph say spc-5 built a typed RPUnavailable error in internal/core and a concrete MCPBridge (the ADR-02 spawn implementation and the ADR-03 host-reuse path), and its Resolved sections route failures through oracle.py. None of it is in the binary: grep for MCPBridge, RPUnavailable and RepoPrompt over internal/ and cmd/ finds only the scanner's RepoPrompt sessionKey pattern (internal/adapter/scanner/patterns.go) and a guard corpus line, and go.mod carries no MCP dependency. The sections describe an earlier Python lineage, so a planner or an implementer reading the planned record is told a foundation exists that does not; the re-filed scope (spc-2609211950427074) needs that foundation built from nothing.

## Grounds

- pursued: every implementation claim left in itd-6 is checkable against the tree (grep MCPBridge, RPUnavailable over internal/ and cmd/, go.mod's requires); a sentence still telling a planner a bridge or typed error exists would show it wrong
