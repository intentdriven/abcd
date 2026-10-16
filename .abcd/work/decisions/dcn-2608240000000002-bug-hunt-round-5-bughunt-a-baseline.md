---
id: dcn-2608240000000002
slug: bug-hunt-round-5-bughunt-a-baseline
date: 2026-08-24
---
2026-08-24 — Bug-hunt round 5 (bughunt-a): baseline green; five parallel
hunters (core code, newer code, infrastructure, docs-vs-functionality,
internal record consistency) on this v0.6.3 tree returned four genuine nulls
and one confirmed substantive record finding, adversarially refuter-survived.
Fixed iss-2608240806090544 — planned/README.md categorically claimed every
intent carries a spec_id: spc-N link, but 20 of 43 planned intents legally
hold spec_id: null (the lint-enforced intent_lifecycle state, the surface
contract 05-intent.md, and adr-34's orthogonality); the subheading silently
reintroduced the over-narrow framing adr-34 already corrected in the parent
intents/README (resolved iss-3). Rewritten to the authoritative wording.
drafts/README refuter-cleared (describes the promotion transition, not
current contents) — kept out of scope. Persona-pronoun drift in a frozen
dated implementation-prompt transcript logged as considered-and-rejected
(historical artefact; the shipped intent itd-135 already reads they/them).
