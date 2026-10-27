---
schema_version: 1
id: "iss-2610262031213919"
slug: "01-platform-calls-the-markdown-plugin"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 119 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/01-platform.md:17"
remedy: "none (filed automatically)"
finding_ref: cfk-401068e24a266f5b
---

01-platform calls the markdown plugin surface a later front door and a new adapter over the core; the verification matrix treats commands/*.md shelling to the binary through the CLI as delivered wiring.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/01-platform.md:17` — "are later front doors — each is a new adapter over the unchanged core"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:25` — "A `commands/*.md` surface shells to the binary → the CLI front door parses input"

01-platform says an MCP server and a markdown plugin surface 'are later front doors — each is a new adapter over the unchanged core', the CLI being the door that ships in the MVP. The matrix's front-door wiring row, which describes behaviour the binary carries today, says 'A `commands/*.md` surface shells to the binary → the CLI front door parses input', staging only the MCP half. The plugin surface is thus both a later door over the core and a shipped layer over the CLI.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 119.
