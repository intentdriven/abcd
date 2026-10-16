---
schema_version: 1
id: "iss-2610160608373524"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/19-identity.md"
remedy: "Say at 19-identity.md:62-63 that warn reports drift as a warning, which exits 1 when no error is found, rather than that it does not fail the run."
---

The brief at .abcd/development/brief/04-surfaces/19-identity.md:62-63 says '`warn`, the default, reports drift as a warning that does not fail the run'. Bare `abcd lint` is tri-state and exits 1 on warnings alone (its help; internal/surface/cli/lint.go:73-75), so warn-tier drift alone makes the run exit 1.
