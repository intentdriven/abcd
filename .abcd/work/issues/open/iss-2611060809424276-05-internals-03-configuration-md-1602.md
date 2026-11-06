---
schema_version: 1
id: "iss-2611060809424276"
slug: "05-internals-03-configuration-md-1602"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add memory pages, the `## AWAY` block and the `systemMessage` form to the stdout row and to line 1629."
---

`05-internals/03-configuration.md:1602` says the prompt hook's stdout is the domains block, a `## MEMORY` block of notes, then the status block and nothing else, zero bytes with nothing to show, and that the prompt owing the session its name is the one JSON exception. Three parts differ. The `## MEMORY` block also carries memory pages, as `abcd hook --help` says. A `## AWAY` block follows the status block while the away record holds (`internal/surface/cli/cli.go:2006`). And while that record holds, or once it lapsed with something unread, stdout is one JSON object with `systemMessage` on prompts that owe no name and match nothing (`internal/surface/cli/hook_away.go:177-203`). Line 1629 repeats the same contents for the envelope's `text`.
