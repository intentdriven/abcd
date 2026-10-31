---
schema_version: 1
id: "iss-2610311224596015"
slug: "itd-39-s-open-questions-section-is-one"
severity: "minor"
category: "inconsistency"
source: "agent-observation"
found_during: "selecting intents to build from the board"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md"
remedy: "Rewrite the section as the settled prose marker the reader recognises (_None open; the three questions first left here are settled under Decisions (2026-10-11), and the technical questions the plan leaves open are in the bundle's spec._), with no list item."
resolution: "itd-39's Open Questions section now states it is settled in the italic prose marker the reader recognises, so the build no longer counts a question there"
impact: internal
---

itd-39's ## Open Questions section is one bullet, 'None for the product thinker. The three questions first left here are settled under Decisions (2026-10-11); …', and the fail-closed reader in internal/core/intent/questions.go counts every list item without a resolved or deferred marker as a question, so abcd build itd-39 refuses at open_questions (itd-2610091918433290, which shares spec spc-2610111627213138, reads only its own section and was refused at the reservation instead). The record says it is settled in a form the reader does not recognise.
