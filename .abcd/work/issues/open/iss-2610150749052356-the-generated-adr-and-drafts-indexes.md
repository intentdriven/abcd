---
schema_version: 1
id: "iss-2610150749052356"
slug: "the-generated-adr-and-drafts-indexes"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "iss-2608220150157511 fix review pair"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs/README.md"
remedy: "Render each row's typed links from the record's own frontmatter (supersedes, refines and the other typed-link fields) so the relationship shows in the index without a hand edit; add any missing typed link to the record where the note was the only place it lived."
---

The generated ADR and drafts indexes dropped hand-written relationship notes the old rows carried ('refines adr-50' on adr-55, 'amends adr-19' and 'extends adr-19, adr-20' on adr-31, adr-39/adr-23 links on itd-22, a reframe note on itd-16), because a row is derived from the record's H1 and status only.
