---
schema_version: 1
id: "iss-2610262030290887"
slug: "platform-says-the-markdown-plugin"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 6 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/01-platform.md:17"
remedy: "none (filed automatically)"
finding_ref: cfk-b61f1a105b5163b3
---

Platform says the markdown plugin surface is a later front door; the ahoy chapter describes the slash command already dispatching every sub-verb

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/01-platform.md:17` — "An **MCP server** and a **markdown plugin surface that shells to the Go binary** are later front doors"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:42` — "The slash command dispatches every sub-verb and mode but the identity check, the write verbs included, and each announces that it writes before it runs."

The platform page says the MCP server and the markdown plugin surface are later front doors, with the CLI the only one shipping in the MVP. The ahoy chapter says the slash command dispatches every sub-verb and mode except the identity check, so the plugin surface is a working front door now. The two cannot both describe the current state.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 6.
