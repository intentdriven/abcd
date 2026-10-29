---
schema_version: 1
id: "iss-2610262043202769"
slug: "itd-109-lets-the-maintainer-record-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1131 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-109-a-two-part-verification-suite-for-abcd.md:29"
remedy: "In itd-109 name the role wherever it says the maintainer: the technical facilitator records a disposition over a failed or pending machine check, and the product thinker records one that accepts a criterion as unmet; fix the quote's attribution in the same edit"
related_intents: [itd-109, itd-2609212137129937]
finding_ref: cfk-09229528449657ac
---

itd-109 lets 'the maintainer' record a disposition, where itd-2609212137129937 requires naming the product thinker or the facilitator

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-109-a-two-part-verification-suite-for-abcd.md:29` — "verdict refuses the move unless the maintainer records a disposition"
- End B: `.abcd/development/intents/shipped/itd-2609212137129937-abcd-s-own-text-names-the-product.md:39` — "rewritten to the product thinker (what to build, adoption, rulings) or the technical facilitator (how, gates, mechanics)"

The draft's gate refuses a ship, a plan completion or a cut unless the maintainer records a disposition; the shipped intent rewrites every such use to the product thinker (rulings) or the technical facilitator (gates, mechanics). The draft does not say which of the two may answer its gate, so a builder cannot tell whose act the disposition is.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1131.
