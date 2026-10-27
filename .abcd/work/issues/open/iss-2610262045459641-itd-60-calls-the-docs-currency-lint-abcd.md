---
schema_version: 1
id: "iss-2610262045459641"
slug: "itd-60-calls-the-docs-currency-lint-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1295 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:151"
remedy: "none (filed automatically)"
related_intents: [itd-60, itd-2609212130136102]
finding_ref: cfk-d20ba39988a90352
---

itd-60 calls the docs-currency lint abcd docs lint; itd-2609212130136102 moves every check under abcd lint, as lint docs

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:151` — "`abcd docs lint` — change-narration, resolvable cross-links"
- End B: `.abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:41` — "**One lint with targets**: `abcd lint` (all), `lint docs`, `lint outbound`"

itd-60 names the shipped docs-currency lint abcd docs lint. itd-2609212130136102 consolidates the repository checks into one lint with targets, lint docs among them, keeping only the writing docs cite under docs. One check now has two spellings across the record, and the one itd-60 uses is the moved one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1295.
