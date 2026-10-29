---
schema_version: 1
id: "iss-2610262042029426"
slug: "the-product-thinker-entry-says-a-persona"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1041 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/product-thinker.md:46"
remedy: "none (filed automatically)"
related_intents: [itd-2610090831227812]
finding_ref: cfk-94efe17a1c889894
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262041032173. Same End A and the same registry fact; itd-2610090831227812 is another instance of 'Iris, a product thinker'."
duplicates: [iss-2610262041032173]
---

The product-thinker entry says a persona's role hint is an outside job title, not the product thinker; itd-2610090831227812 attributes its quote to 'Iris, a product thinker'.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/product-thinker.md:46` — "persona is a modelled archetype in a press release, and its role hint is an outside job title."
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:23` — "said Iris, a product thinker"

The product-thinker entry says the role is not to be used for a persona, because a persona is a modelled archetype whose role hint is an outside job title. The shipped menu intent attributes its press-release quote to the persona Iris with 'product thinker' as her role. That uses abcd's own role name as a persona's job title and merges the two concepts the glossary keeps apart.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1041.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262041032173. Same End A and the same registry fact; itd-2610090831227812 is another instance of 'Iris, a product thinker'.
