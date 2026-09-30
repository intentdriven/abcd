---
schema_version: 1
id: "iss-2608271804496010"
slug: "four-pruned-adrs-are-still-cited-by-name-restore-or-rule"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/decisions/adrs"
remedy: "Waits on ruling F: if restored, bring adr-4, adr-8, adr-16 and adr-18 back from their pruning commits with status superseded and both halves of each supersession in frontmatter, as `.abcd/development/decisions/adrs/README.md` retains a still-cited ADR, proven by lint-decisions and links_resolve green; if body-prose citations do not count, add that sentence to the same README's retention paragraph and resolve this record with it."
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: adr-4, adr-8, adr-16 and adr-18 are still pruned and still cited by name (adr-4 in several brief chapters, adr-8 in adr-10 and adr-25). Restore them as superseded records, or rule that body-prose citations do not count for the retention clause. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

eight pruned ADR ids are still cited by later records, and for four of them (adr-4, adr-8, adr-16, adr-18) the citations are live by-name references that the decisions charter's retention clause says keep a record restorable: either restore those four from the pruning commits as status superseded records, or record a maintainer ruling that body-prose citations do not count as cite for the retention clause. Needs the maintainer's call — the two remedies diverge and neither is obviously right.

## Remedy grounds (2026-09-29)

- The charter's retention sentence (a superseded ADR is pruned unless later records still cite it) is the rule both branches must satisfy, so each remedy edits either the records or that sentence and nothing else.
- No outside-practice check: the question is this repository's own retention clause.
- Rejected: rewriting the citing records to drop the names, which erases the trail the retention clause protects.
