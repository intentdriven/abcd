---
schema_version: 1
id: "iss-2610262031403463"
slug: "the-press-release-gives-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 161 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:7"
remedy: "Change 'Alice' to 'Iris' in itd-106's customer quote: personas.json gives Iris the 'product thinker' role hint and Alice solo founder and product engineer."
related_intents: [itd-106]
finding_ref: cfk-7e540bf8983ab184
---

The press release gives the product-thinker role to Iris, while itd-106 gives it to Alice

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:7` — "said Iris, a product thinker."
- End B: `.abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:30` — "said Alice, a product thinker."

The press release attributes a quote to Iris, a product thinker. itd-106 attributes one to Alice, a product thinker. Under the personas rule that the role picks the registered name, one role carries two names here, so at most one of the two quotes follows role-first selection.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 161.
