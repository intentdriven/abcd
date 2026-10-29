---
schema_version: 1
id: "iss-2610262031126042"
slug: "the-platform-page-says-the-markdown"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 99 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/01-platform.md:17"
remedy: "Edit 01-platform.md (Front doors paragraph, line 17): the markdown plugin surface ships beside the CLI; only the MCP server remains a later front door (adr-23's third)."
finding_ref: cfk-58227ef3ef5e32ff
---

The platform page says the markdown plugin surface is a later front door, but the configuration chapter describes it as part of the shipped plugin shape

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/01-platform.md:17` — "An **MCP server** and a **markdown plugin surface that shells to the Go binary** are later front doors"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1247` — "A Go binary plus the markdown plugin surface that shells to it"

The platform page sets the CLI against what comes after it. The CLI ships in the MVP, while an MCP server and the markdown plugin surface that shells to the binary are later front doors. The configuration chapter's plugin shape is a Go binary plus the markdown plugin surface that shells to it, with the flat `commands/<verb>.md` pages in the tree. In that chapter the MCP door alone is still to come, as adr-23's third. The plugin surface cannot be both a later front door and present in the shipped layout.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 99.
