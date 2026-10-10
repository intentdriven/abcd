---
schema_version: 1
id: "iss-2609251606543515"
slug: "the-spec-close-remedy-prints-the-verb"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "The routing resolver strips a leading 'abcd ' from the verb before naming it on its fallback and diagnostic stderr lines, so every caller spelling prints the verb once; TestRouteStderrNamesTheVerbOnce pins it."
impact: fix
resolved_by:
  commit: "94f02fce5"
---

The spec-close remedy prints the verb twice ("abcd abcd spec close"): route.go:259 prefixes an already-prefixed verb string, and the same doubling stands at cli.go:2360, cli.go:2397 and ship.go:331 (review2-tier2 note a).

## Grounds

- pursued: every routed verb's stderr line names the verb exactly once; a line reading 'abcd abcd' from any routed verb would show it wrong
