---
schema_version: 1
id: "iss-2611260643446976"
slug: "adr-23-s-core-boundary-holds-by"
severity: "minor"
category: "architectural-insight"
source: "agent-observation"
found_during: "an architecture review, 2026-11-26"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core"
remedy: "Add an audit test that walks internal/core's packages with go/packages (or go list -deps) and fails on an import of internal/surface, a reference to os.Stdout or fmt.Print*, and a call to os.Exit, with an allowlist file naming each standing exception and its reason; then decide each embedded front-door string (keep with a reason, or move it to the surface)."
---

adr-23's boundary, that internal/core never imports internal/surface, never writes to stdout and never calls os.Exit, holds by convention only: no test or lint checks it, and some core strings already embed front-door text, such as the regenerate command in internal/core/lint/indexgen.go.
