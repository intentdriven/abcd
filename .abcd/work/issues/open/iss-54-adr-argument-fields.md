---
schema_version: 1
id: "iss-54"
slug: "adr-argument-fields"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
found_at: ".abcd/development/decisions/adrs"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): Should ADR frontmatter gain related_principles, confidence and revisit_when fields, filled selectively? The ADR shape abcd decide writes (decisions/adrs/README.md) carries none of the three at v0.11.1."
remedy: "Waits on the ADR-fields ruling: if adopted, add related_principles (prn- handles), confidence (high, medium or low) and revisit_when (the condition that reopens the decision) as optional keys to the ADR frontmatter in decisions/adrs/README.md, written null by abcd decide's skeleton, with record-lint checking that each principle handle resolves and confidence is in the list, and no backfill; if declined, move the record to wontfix. Prove it with a decide skeleton test and a record-lint test on an unresolvable principle handle."
---

ADR frontmatter selectively gains three argument fields: related_principles (which standing principles the decision leans on), a confidence qualifier, and revisit_when — the rebuttal slot naming the conditions under which the decision no longer holds. The convention serves decision durability: an ADR that records only the choice invites re-litigation, while one that records its confidence and its expiry conditions tells a future session exactly when reopening is legitimate. Adopt the fields selectively per the 2026 template-comparison evidence rather than the full heavyweight argumentation set, which adds ceremony without proportionate recall value. Acceptance: new ADRs carry the three fields where they apply, and a revisit_when condition coming true is sufficient grounds to reopen without further debate.

## Remedy grounds (2026-09-29)

- Why: the record asks for three fields, not a heavier template; principles carry prn- handles (adr-2609021016270132), so related_principles is checkable.
- Sources (consulted 2026-09-29): Tyree and Akerman's decision template adds a related-principles field (IEEE Software 22(2), 2005, https://doi.org/10.1109/MS.2005.27); MADR 4.0.0 carries none of the three and puts compliance in an optional Confirmation section (https://adr.github.io/madr/).
- Rejected: MADR's decision-makers, consulted and informed keys, ceremony the record's evidence advises against.
