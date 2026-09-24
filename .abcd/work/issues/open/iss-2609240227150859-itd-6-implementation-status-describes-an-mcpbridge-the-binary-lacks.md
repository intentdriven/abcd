---
schema_version: 1
id: "iss-2609240227150859"
slug: "itd-6-implementation-status-describes-an-mcpbridge-the-binary-lacks"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A, planning briefs"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-6-rp-mcp-only-integration.md"
---

itd-6's Implementation status and its post-spc-5 Status paragraph say spc-5 built a typed RPUnavailable error in internal/core and a concrete MCPBridge (the ADR-02 spawn implementation and the ADR-03 host-reuse path), and its Resolved sections route failures through oracle.py. None of it is in the binary: grep for MCPBridge, RPUnavailable and RepoPrompt over internal/ and cmd/ finds only the scanner's RepoPrompt sessionKey pattern (internal/adapter/scanner/patterns.go) and a guard corpus line, and go.mod carries no MCP dependency. The sections describe an earlier Python lineage, so a planner or an implementer reading the planned record is told a foundation exists that does not; the re-filed scope (spc-2609211950427074) needs that foundation built from nothing.
