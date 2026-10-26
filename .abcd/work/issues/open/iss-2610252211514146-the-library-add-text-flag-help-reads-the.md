---
schema_version: 1
id: "iss-2610252211514146"
slug: "the-library-add-text-flag-help-reads-the"
severity: "nitpick"
category: "ux"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/source.go"
remedy: "Reword the flag help to 'a file holding the extracted text of a non-text document' (and the long help's 'any other needs --text' to match)."
---

The library add --text flag help reads 'the extracted text of a non-text document' (internal/surface/cli/source.go:320, docs/reference/cli/commands.md:3756), which reads as if the value is the text itself. source.Add reads the value as a path and refuses anything that is not a readable regular file with 'the extracted text is not a readable regular file' (internal/core/source/add.go:301-306).
