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
remedy: "none (filed automatically)"
related_intents: [itd-2609291923559186]
finding_ref: cfk-41a25c09f3066484
---

The glossary keeps 'product thinker' off personas; the consistency-pass press release uses it as a persona's role hint.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/product-thinker.md:46` — "persona is a modelled archetype in a press release, and its role hint is an outside job title."
- End B: `.abcd/development/intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md:26` — "said Iris, a product thinker"

product-thinker.md says the term is not for a persona: a persona is a modelled archetype whose role hint is an outside job title. itd-2609291923559186's press release gives the persona Iris the role 'a product thinker', mixing the role abcd addresses with a persona's job title.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 974.
