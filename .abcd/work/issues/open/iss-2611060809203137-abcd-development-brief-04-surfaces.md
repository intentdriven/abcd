---
schema_version: 1
id: "iss-2611060809203137"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Say the refusal states three routes and name re-grading as a fourth outside it, or add re-grading to the refusal's text."
---

`.abcd/development/brief/04-surfaces/04-launch.md:659` gives four routes out of the unfixed-findings refusal, 'in the order the refusal states them'. The refusal (`unfixedReason`, `internal/core/changelog/findings.go:538`) states three: fix and resolve, `abcd capture wontfix`, and the deferral pair. Re-grading the record is not in its text.
