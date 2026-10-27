---
schema_version: 1
id: "iss-2610262030410509"
slug: "invariant-17-makes-the-intent-to-spec"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 31 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:171"
remedy: "none (filed automatically)"
finding_ref: cfk-a5143d198ac1dfdc
---

Invariant 17 makes the intent-to-spec relation 1:n; the intent chapter's kinds table maps a standalone intent to one spec (1:1)

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:171` — "the intent↔spec relation is 1:n: an intent that has been thought through stands as written"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:83` — "One spec (1:1) | itd-3, itd-4, itd-7, most of the corpus"

03-invariants holds that an intent owns one or more specs and ships when its last one closes, while the kinds table in 05-intent states a standalone intent maps to one spec, 1:1. A standalone intent with a remainder spec is legal under the invariant and excluded by the table.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 31.
