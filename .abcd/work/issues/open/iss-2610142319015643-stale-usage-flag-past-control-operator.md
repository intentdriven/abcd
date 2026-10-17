---
schema_version: 1
id: "iss-2610142319015643"
slug: "stale-usage-flag-past-control-operator"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "Opus+Sonnet review of fix/stale-usage-notes, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
remedy: "Stop the flag scan for an invocation at the first shell control operator after it (|, ||, &&, ;, &), so only that invocation's own arguments count; test that a fenced `abcd lint | jq --frob` does not document lint --frob, watched fail first."
---

synopsisCarriesFlag (internal/surface/cli/staleusage.go) counts a flag anywhere after an invocation on the same logical line, across a shell control operator: a fenced `abcd lint | jq --frob` (or `abcd lint && jq --frob`) reads as documenting `abcd lint --frob`, so the stale-binary note can be granted for a flag that belongs to another command. Found by the Opus+Sonnet review of fix/stale-usage-notes, 2026-10-15.
