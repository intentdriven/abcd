---
schema_version: 1
id: "iss-2610262031127666"
slug: "the-platform-page-says-an-mcp-server-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 100 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/01-platform.md:17"
remedy: "none (filed automatically)"
finding_ref: cfk-e253661dc49290e9
---

The platform page says an MCP server is a later front door, but universal patterns says abcd interoperates with the companion harness as an MCP server

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/01-platform.md:17` — "An **MCP server** and a **markdown plugin surface that shells to the Go binary** are later front doors"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:73` — "when both are present, abcd and the companion harness interoperate as two independent MCP servers, each a front door over its own core (adr-23)"

The platform page says abcd's MCP server is a later front door, not one that exists now. Universal patterns states, in the present tense and with no staged marker, that MCP is the runtime contract. It says that when both tools are present, abcd and the companion harness interoperate as two independent MCP servers, each a front door over its own core. abcd cannot interoperate as an MCP server while that server is still a later front door.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 100.
