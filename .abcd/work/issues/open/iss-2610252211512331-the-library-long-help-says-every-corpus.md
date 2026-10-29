---
schema_version: 1
id: "iss-2610252211512331"
slug: "the-library-long-help-says-every-corpus"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/source.go"
remedy: "Name the exception in the library long help: 'Every corpus verb but init exits 3 when there is no corpus; sync-banlist --refresh notes it and exits 0.'"
---

The library long help says 'Every corpus verb but init exits 3 when there is no corpus.' (internal/surface/cli/source.go:98-99, docs/reference/cli/commands.md:3708-3709). But library sync-banlist --refresh with no corpus prints a one-line notice and exits 0 (source.go:557-561), as its own help and flag text say (commands.md:3906-3914).
