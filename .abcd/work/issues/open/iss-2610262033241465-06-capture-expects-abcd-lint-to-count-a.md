---
schema_version: 1
id: "iss-2610262033241465"
slug: "06-capture-expects-abcd-lint-to-count-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 401 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md:570"
remedy: "Reword the acceptance criterion at 06-capture.md lines 570-573 to name the surfaces that carry the count: the bare capture board and the record lint's reading_outstanding report, not abcd lint, whose bare targets do not read the readings store (16-lint.md)."
finding_ref: cfk-0f1a5e6141cf8b27
---

06-capture expects `abcd lint` to count a widening run's admitted, declined and held proposals, but 16-lint's rule set has no such check.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/06-capture.md:570` — "**Given** a run of widening items, **when** the bare board or `abcd lint`"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:28` — "judges the repository, each as one rule of the set below: the docs through"

One of 06-capture's acceptance criteria expects `abcd lint` to count a widening run's admitted, declined and held proposals, and to name each one that has neither an admission nor a declined or held disposition. 16-lint says bare `abcd lint` runs every target that judges the repository, each as one rule of its v1 set: three-tier-layout, conventions-router, decision-durability, docs-currency, privacy-hygiene, site-gates and identity-positioning. None of those reads the readings store or the admission records.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 401.
