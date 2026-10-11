---
schema_version: 1
id: "iss-2609252053576085"
slug: "the-build-s-open-questions-check-intent"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "intent.OpenQuestions reads the section opener _All resolved …_ and an item's explicit resolved/deferred marker, and nothing else; the rule is the 2026-09-25 DECISIONS entry. itd-111 and itd-93 now pass; ten planned intents still refuse on items the record does not mark settled."
impact: fix
resolved_by:
  commit: "b321386f"
---

The build's open_questions check (intent.OpenQuestions, read by internal/core/implement/loop/check.go) counts every list item under '## Open Questions' as a question whatever it says, so it refuses records that follow the record's own settled convention: 12 of 50 planned intents carry list items there, and some are settled by their own words, itd-111's section opening '_All resolved or explicitly deferred at planning_' and itd-93's '_All four resolved_', itd-76's item marked '**explicitly deferred**', itd-60's items marked '**Resolved', itd-117's '**Deferred**' items. 'abcd build next' (itd-2609211116005482) would meet refusals nobody planned, and the ruling lives only in a lane report. Keep the check fail-closed, but read the markers the record actually uses.

## Grounds

- pursued: we expect the check to pass exactly the sections and items the record marks settled; shown wrong if an item led Open, a pointer, or a question mentioning deferral passes, or a marked item still refuses
