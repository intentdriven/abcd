---
schema_version: 1
id: "iss-2610262043265040"
slug: "itd-106-gives-the-product-thinker"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1138 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-106, itd-2610090831227812]
finding_ref: cfk-4744d9f16d1a4b45
---

itd-106 gives the product-thinker persona the name Alice; itd-2610090831227812 records the registry's product-thinker persona as Iris

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:30` — "said Alice, a product thinker"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:99` — "the persona registry picks the name from the role"

itd-2610090831227812 records that the persona registry picks the name from the role, and the quote's voice was settled as 'Iris, product thinker'. itd-106 attributes its quote 'said Alice, a product thinker'. One role carries two names, so at most one of the two attributions follows the registry.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1138.
