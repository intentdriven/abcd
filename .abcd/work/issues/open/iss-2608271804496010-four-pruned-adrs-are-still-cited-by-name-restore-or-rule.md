---
schema_version: 1
id: "iss-2608271804496010"
slug: "four-pruned-adrs-are-still-cited-by-name-restore-or-rule"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/decisions/adrs"
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: adr-4, adr-8, adr-16 and adr-18 are still pruned and still cited by name (adr-4 in several brief chapters, adr-8 in adr-10 and adr-25). Restore them as superseded records, or rule that body-prose citations do not count for the retention clause. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

eight pruned ADR ids are still cited by later records, and for four of them (adr-4, adr-8, adr-16, adr-18) the citations are live by-name references that the decisions charter's retention clause says keep a record restorable: either restore those four from the pruning commits as status superseded records, or record a maintainer ruling that body-prose citations do not count as cite for the retention clause. Needs the maintainer's call — the two remedies diverge and neither is obviously right.