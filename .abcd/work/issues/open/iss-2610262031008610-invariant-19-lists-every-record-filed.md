---
schema_version: 1
id: "iss-2610262031008610"
slug: "invariant-19-lists-every-record-filed"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 72 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:175"
remedy: "In 03-invariants.md invariant 19, replace 'a record filed before that carries none and is listed as ineligible' with 'a record filed before reads its older `suggested_fix:` as its remedy and is ineligible only when it has neither'."
finding_ref: cfk-adfb28697068a717
---

Invariant 19 lists every record filed before the remedy field as ineligible, but the drain reads such a record's suggested_fix as its remedy

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:175` — "a record filed before that carries none and is listed as ineligible"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:75` — "record written before the field existed reads its `suggested_fix:` as its remedy"

Invariant 19 states that a record filed before remedies were required carries none and is listed as ineligible. The drain chapter says such a record reads its suggested_fix: as its remedy, so a legacy record that has one can pass the remedy clause and be taken by a machine. The invariant's fixed disposition and the drain's fallback disagree about what happens to the same records.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 72.
