---
schema_version: 1
id: "iss-2610262045454143"
slug: "itd-60-attributes-decisions-and-reviews"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1296 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:99"
remedy: "none (filed automatically)"
related_intents: [itd-60, itd-2609212137129937]
finding_ref: cfk-9eea9b1ba64c8354
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The later ruling names where the word is rewritten and shipped intents are not among them; this is a nitpick on historical wording."
---

itd-60 attributes decisions and reviews to the maintainer; itd-2609212137129937 retires the word in favour of the two named roles

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:99` — "**The gate can run fully autonomously (maintainer, 2026-09-02).**"
- End B: `.abcd/development/intents/shipped/itd-2609212137129937-abcd-s-own-text-names-the-product.md:21` — "the word maintainer leaves the vocabulary"

itd-60 records a ruling as the maintainer's and sends applied brief edits for the maintainer's review, while elsewhere naming the product thinker for the same act. itd-2609212137129937 rules that the word maintainer leaves the vocabulary and that each use names the product thinker or the technical facilitator.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1296.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The later ruling names where the word is rewritten and shipped intents are not among them; this is a nitpick on historical wording.
