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
remedy: "Edit 04-universal-patterns.md section 3 (MCP bullet): mark abcd's MCP server as a design target and say today abcd and a peer interoperate over shared on-disk conventions only."
finding_ref: cfk-8afab655f6b6daf2
resolution: "04-universal-patterns.md section 3 now marks abcd's MCP server (bullet, opening sentence and diagram) a design target, and the MCP bullet says today abcd and a peer interoperate over shared on-disk conventions only; a lint-package test holds the bullet to that while internal/surface/mcp does not exist."
impact: internal
resolved_by:
  commit: "263a40ff66a758549264c6ee1e367fbf62a967f5"
---

Universal patterns say abcd and the companion harness interoperate as two MCP servers; the build sequence says abcd's MCP front door is a design target.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:73` — "when both are present, abcd and the companion harness interoperate as two independent MCP servers"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:108` — "a second thin door over the unchanged"

04-universal-patterns calls MCP the runtime contract: with both present, abcd and the companion harness interoperate as two independent MCP servers. 01-build-sequence marks the MCP front door as a design target, with `internal/surface/` holding only `cli` and `dashboard`. abcd cannot take part as an MCP server it does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 807.

## Grounds

- pursued: the brief agrees with the build sequence and the tree that abcd has no MCP front door yet; shown wrong if internal/surface/ gains an MCP door (the test then skips and the bullet should be revisited) or if another brief page still claims abcd runs as an MCP server today.
