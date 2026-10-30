---
schema_version: 1
id: "iss-2610262031493380"
slug: "the-brief-lifeboat-mapping-grounds-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 182 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:63"
remedy: "none (filed automatically)"
related_intents: [itd-90]
finding_ref: cfk-7b93452db84b1ee1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. mapping.go marks mental-model human-owned while its table row still reads partial at tier 1; the two facts are orthogonal and 00-meta's personas paragraph names the sharpest case without denying the others."
---

The brief-lifeboat mapping grounds the mental model partly from conventions, but itd-90 treats the mental model as a human-owned section.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:63` — "`product/mental-model` | `brief/01-product/03-mental-model.md` | blank | partial | grounded"
- End B: `.abcd/development/intents/drafts/itd-90-brief-interview-for-the-blanks.md:32` — "A `human-owned` section (personas, mental model) is presented as"

00-meta's generated mapping gives `product/mental-model` a partial grounding at tier 1, from docs explanation pages and ADR context sections. It singles out personas as the section that cannot be derived from a repository and is therefore human-owned (adr-36). itd-90 lists the mental model beside personas as a human-owned section to present as 'this is yours to write', a section the mapping says can be extracted.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 182.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. mapping.go marks mental-model human-owned while its table row still reads partial at tier 1; the two facts are orthogonal and 00-meta's personas paragraph names the sharpest case without denying the others.
