---
schema_version: 1
id: "iss-2610262035171024"
slug: "itd-6-s-scope-makes-repoprompt-over-mcp"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 561 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-6a8be1b3c848b43f
---

itd-6's scope makes RepoPrompt over MCP the preferred oracle transport ahead of a Codex CLI and the in-session agent; the status-board chapter says every delegated step runs through the harness unless a routing table is accepted

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:41` — "RP MCP (preferred) → Codex CLI subprocess (alternative for non-Mac / no-RP users — RP is macOS-only) → in-session subagent (final fallback)"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:283` — "With no table accepted the lines are absent and every delegated step runs through the harness at `host-decides`."

itd-6 keeps a three-transport chain with RepoPrompt over MCP preferred, a Codex CLI subprocess second and the in-session subagent as the last fallback. The status-board chapter says that with no routing table accepted, every delegated step runs through the harness at host-decides. One document puts an external route first by default and the other makes the harness the default.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 561.
