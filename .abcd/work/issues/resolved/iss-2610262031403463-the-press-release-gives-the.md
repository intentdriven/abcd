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
resolution: "itd-106's product-thinker quote already read 'said Iris' on the base (c4fb3355d, resolving the sibling iss-2610262043135705), agreeing with the press release and personas.json; this commit adds TestTheProductThinkerQuotesNameTheRegisteredPersona so both quotes stay on the registry's name for the role."
impact: internal
resolved_by:
  commit: "d0196a7dfb4edcdb96e468bcfcfd20b625794dc1"
---

The press release gives the product-thinker role to Iris, while itd-106 gives it to Alice

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:7` — "said Iris, a product thinker."
- End B: `.abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:30` — "said Alice, a product thinker."

The press release attributes a quote to Iris, a product thinker. itd-106 attributes one to Alice, a product thinker. Under the personas rule that the role picks the registered name, one role carries two names here, so at most one of the two quotes follows role-first selection.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 161.

## Grounds

- pursued: the press release and itd-106 both attribute their product-thinker quote to Iris, the name personas.json gives the role; a quote in either naming anyone else, or the registry giving the role to another name, would show it wrong, and the new test fails on exactly that (watched failing on the c4fb3355d^ text).
