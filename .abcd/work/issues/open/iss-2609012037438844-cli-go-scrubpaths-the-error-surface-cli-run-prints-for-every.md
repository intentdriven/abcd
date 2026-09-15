---
schema_version: 1
id: "iss-2609012037438844"
slug: "cli-go-scrubpaths-the-error-surface-cli-run-prints-for-every"
severity: "minor"
category: "security"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

cli.go scrubPaths — the error surface cli.Run prints for every verb — redacts the cwd, the home and the paths embedded in a PathError but applies no termsafe.Sanitize, so an error message that echoes an operand (acquireSource fetch failed for %s with the raw URL, and every other verb whose error text quotes user or repository content) can carry ESC, C1 and bidi runes to stderr raw. Repo-wide, all verbs; found while fixing GHSA-4fmm-95pf-32c6 and deliberately not fixed there. A fix would sanitise once at the print site and pin it with an ESC-bearing operand.
