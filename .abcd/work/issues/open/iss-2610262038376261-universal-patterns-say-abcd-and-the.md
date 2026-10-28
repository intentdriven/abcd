---
schema_version: 1
id: "iss-2610262038376261"
slug: "universal-patterns-say-abcd-and-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 807 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:73"
remedy: "none (filed automatically)"
finding_ref: cfk-8afab655f6b6daf2
---

Universal patterns say abcd and the companion harness interoperate as two MCP servers; the build sequence says abcd's MCP front door is a design target.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:73` — "when both are present, abcd and the companion harness interoperate as two independent MCP servers"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:108` — "a second thin door over the unchanged"

04-universal-patterns calls MCP the runtime contract: with both present, abcd and the companion harness interoperate as two independent MCP servers. 01-build-sequence marks the MCP front door as a design target, with `internal/surface/` holding only `cli` and `dashboard`. abcd cannot take part as an MCP server it does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 807.
