---
schema_version: 1
id: "iss-2610262040082360"
slug: "the-agent-catalog-says-the-oracle-seam"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 912 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md:207"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-3087deb760443911
---

The agent catalog says the oracle seam is never a fixed cascade; itd-6 says its RP MCP, then Codex CLI, then in-session cascade is the fixed cascade of universal-patterns § 7

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/01-agents.md:207` — "the seam is one interface with a native default and opt-in shapes, never a fixed cascade the core imposes."
- End B: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:55` — "The cascade defined by this intent (RP MCP → Codex CLI → in-session subagent) is and stays availability-driven (per `04-universal-patterns.md § 7` "fixed cascade")"

01-agents cites 04-universal-patterns § 7 for the rule that the oracle seam is one interface with a native default and opt-in shapes, never a fixed cascade the core imposes. itd-6 cites the same section as a fixed cascade and says its RP MCP, then Codex CLI, then in-session subagent cascade is and stays availability-driven. The same section cannot both rule out and establish a fixed cascade.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 912.
