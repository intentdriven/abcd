---
schema_version: 1
id: "iss-2610262044373028"
slug: "the-memory-notes-promotion-rule-is"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1219 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:61"
remedy: "In the memory-notes intent's promotion decision replace 'more than two maintainers ... another maintainer' with the vocabulary the shipped ruling allows (the technical facilitator, or people with write access to the repository); the promotion criteria moved to itd-2610111655362931, so carry the same wording there."
related_intents: [itd-2610091918433290, itd-2609212137129937]
finding_ref: cfk-246a04d257239063
---

The memory-notes promotion rule is stated in terms of maintainers, a word the shipped vocabulary ruling retired

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:61` — "In a repository with more than two maintainers, at least one other maintainer must agree before a promotion lands."
- End B: `.abcd/development/intents/shipped/itd-2609212137129937-abcd-s-own-text-names-the-product.md:21` — "Every place abcd writes names which of the two people it means, and the word maintainer leaves the vocabulary."

itd-2610091918433290 makes promotion depend on a count of maintainers and on another maintainer agreeing. itd-2609212137129937 rules that abcd's text names the product thinker or the technical facilitator and that the word maintainer leaves the vocabulary. The rule uses the retired word for the people who approve.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1219.
