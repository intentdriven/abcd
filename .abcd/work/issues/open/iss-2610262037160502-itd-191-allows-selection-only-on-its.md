---
schema_version: 1
id: "iss-2610262037160502"
slug: "itd-191-allows-selection-only-on-its"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 711 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-191-the-selection-criteria-are-a-declared.md:18"
remedy: "In itd-191 scope the rule to the comparative reading's selection among conjectures and say the build pick (34-build.md: a readiness score with a computed grounds entry) is a different selection under its own recorded rule."
related_intents: [itd-191]
finding_ref: cfk-147390af5a44ab66
---

itd-191 allows selection only on its declared criteria, with the final choice made by the maintainer; the build pick chooses an intent itself on three computed scores and records that as a pursued: grounds entry.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-191-the-selection-criteria-are-a-declared.md:18` — "Selection at the gate weighs candidates against the criteria this record declares, and only these."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:176` — "three parts of 0 to 100 at equal weight (decision 7, the bundled default): criteria clarity"

itd-191 limits selection to plausibility, generativity, cost, risk, learning value and practical importance, and says combining them into a choice is the maintainer's act, on recorded grounds. 34-build's pick instead scores candidates on criteria clarity, test path and footprint at equal weight, takes the readiest, and appends a 'picked by run' pursued: grounds entry. That is a machine-made selection on criteria the discipline does not declare.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 711.
