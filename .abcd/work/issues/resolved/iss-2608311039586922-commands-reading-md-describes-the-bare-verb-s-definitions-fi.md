---
schema_version: 1
id: "iss-2608311039586922"
slug: "commands-reading-md-describes-the-bare-verb-s-definitions-fi"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "commands/reading.md now describes the bare verb's 'definitions' field as what the definition locator RESOLVED, and says that a definition silent about its position or its regime, or stating another position's, refuses the whole verb with exit 2 rather than being listed. A cold-reading file naming no closed position is not an instrument and is invisible to the field."
impact: fix
---

commands/reading.md describes the bare verb's 'definitions' field as 'the reading definitions present under agents/'. Since itd-184 that field is what the definition locator RESOLVES, not what is present: a cold-reading-<name>.md naming no closed position is invisible to it, and a definition silent about its position or its regime refuses the whole verb with exit 2 rather than being listed. The plugin surface should say resolved rather than present, and say that a malformed definition is a refusal. The itd-184 builder's lane did not include commands/, so the correction was captured rather than taken.

## Grounds

- pursued: we expect the plugin page to describe what the verb does since itd-184, so an operator reading a short definitions list reads it as a repository with fewer definitions rather than as a malformed one being skipped; a page that still describes a listing of what is present, or that omits the refusal, would show it wrong.
