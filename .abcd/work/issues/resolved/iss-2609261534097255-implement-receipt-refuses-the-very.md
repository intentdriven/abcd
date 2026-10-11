---
schema_version: 1
id: "iss-2609261534097255"
slug: "implement-receipt-refuses-the-very"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/loop.go"
resolution: "samePath compares real paths, so the awaited receipt named through a symlinked spelling of the checkout is taken"
impact: fix
resolved_by:
  commit: "24d9ab37"
---

implement receipt refuses the very receipt the awaiting lane named when the checkout is reached through a symlinked path: the CLI joins a relative receipt path onto os.Getwd (the symlinked spelling, e.g. /var/... on macOS) while the loop's root is git's resolved toplevel (/private/var/...), and loop.samePath compares the two spellings lexically, so the correct receipt reads as 'not at the path given' and the lane cannot advance. samePath should compare the real paths.

## Grounds

- pursued: we expect comparing real paths to take the awaited receipt under any spelling of the checkout; shown wrong if a receipt at another file resolves equal, or the symlinked spelling is still refused
