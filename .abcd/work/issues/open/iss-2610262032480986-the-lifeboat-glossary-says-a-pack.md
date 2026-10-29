---
schema_version: 1
id: "iss-2610262032480986"
slug: "the-lifeboat-glossary-says-a-pack"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 320 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md:121"
remedy: "In glossary/core/lifeboat.md (intro, line 20) list what a pack writes today (grounded brief, coverage report, graveyard, intent and spec spine, release retrospectives, ADRs, issue ledger copy) and drop 'logbook summary, and configuration'; glossary/core/disembark.md's definition says 'configuration' too."
finding_ref: cfk-f0178017e6ebd8a8
---

the lifeboat glossary says a pack carries a logbook summary and configuration; disembark's output shape carries neither

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/02-disembark.md:121` — "The plan lists the file set a pack writes, and a pack that completes writes that same tree."
- End B: `.abcd/development/brief/glossary/core/lifeboat.md:20` — "packs the grounded brief, a coverage report, a graveyard, key decisions, logbook summary, and configuration"

core/lifeboat says disembark packs the grounded brief, a coverage report, a graveyard, key decisions, a logbook summary and configuration. 02-disembark says the file set a pack writes is its output shape, which holds _provenance.json, coverage, brief/, graveyard/, rescue/ (intents, specs, spine), retrospectives/, docs/adrs/ and activity/issues/, and no logbook summary or configuration. The glossary describes contents the packer does not write.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 320.
