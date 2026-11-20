---
schema_version: 1
id: "iss-2610262041032173"
slug: "the-glossary-keeps-product-thinker-off"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 974 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/product-thinker.md:46"
remedy: "In product-thinker.md replace 'its role hint is an outside job title' with a statement that a persona such as Iris may carry 'product thinker' among its role hints (as .abcd/development/personas.json does), while the persona stays a modelled archetype and not the role abcd addresses."
related_intents: [itd-2609291923559186]
finding_ref: cfk-41a25c09f3066484
resolution: "product-thinker.md's When NOT to use no longer calls a persona's role hint an outside job title; it says a persona such as Iris may carry 'product thinker' among its role hints, as personas.json does, while the persona stays a modelled archetype and not the role abcd addresses. TestProductThinkerGlossaryLetsAPersonaCarryTheRoleHint holds the entry to the registry."
impact: internal
resolved_by:
  commit: "ee625b45607399c84c88bc2c002b794f51eed353"
---

The glossary keeps 'product thinker' off personas; the consistency-pass press release uses it as a persona's role hint.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/product-thinker.md:46` — "persona is a modelled archetype in a press release, and its role hint is an outside job title."
- End B: `.abcd/development/intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md:26` — "said Iris, a product thinker"

product-thinker.md says the term is not for a persona: a persona is a modelled archetype whose role hint is an outside job title. itd-2609291923559186's press release gives the persona Iris the role 'a product thinker', mixing the role abcd addresses with a persona's job title.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 974.

## Grounds

- pursued: the glossary entry agrees with personas.json and the itd-2609291923559186 press release; shown wrong if the entry still calls the role hint an outside job title, or if the new test fails against the registry
