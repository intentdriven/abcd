---
schema_version: 1
id: "iss-2610262032509493"
slug: "record-families-maps-intent-spec-step"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 324 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:201"
remedy: "none (filed automatically)"
finding_ref: cfk-42abf5288ffdc283
---

record-families maps intent, spec, step, bundle, issue, release and status; embark's 'record families' are ADRs, issues, intents, specs and release retrospectives

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:201` — "the five record families land at their canonical locations under the target"
- End B: `.abcd/development/brief/glossary/core/record-families.md:4` — "The one page that maps abcd's record families (intent, spec, step, bundle, issue, release, status)"

core/record-families is the one map of abcd's record families, and a name with no row on it is one nobody has justified; its rows are intent, spec, step, bundle, issue, release and status. 03-embark uses 'the five record families' for ADRs, issues, intents, specs and release retrospectives, two of which have no row on the map, so the term names a different set in each document.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 324.
