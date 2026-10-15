---
id: dcn-2607140000000005
slug: adr-35-s-blast-radius-across-the-record
date: 2026-07-14
---
2026-07-14 — adr-35's blast radius across the record was FAR wider than the plan
anticipated, and the line drawn is: **the brief, glossary and roadmap are
reconciled; the intent corpus is NOT.** An adversarial review (four hostile lenses,
every finding independently verified) found the first pass had rewritten the
vocabulary registry to the new model while ~14 other files still asserted the old
one as fact — including an INVARIANT (`03-invariants.md` #6), the product's own
press release, the verification matrix (which encoded adr-4's `.bak` overwrite as a
TEST GATE), and the lint-enforced glossary SSOT. A registry contradicting an accepted
ADR is drift of exactly the kind iss-35 exists to prevent, so all of it was swept.
The INTENTS (itd-2/8/9/10/13/15/19/22/24) were deliberately left alone and tracked as
iss-94: an intent is a proposal with its own lifecycle, and silently rewriting nine of
them inside an unrelated change is worse than recording the drift — each reconciles
when it is next planned. Where adr-35 genuinely does not settle a question (where
`embark scan` searches now that destinations are operator-chosen; what the `/abcd`
status board reads now that there is no in-tree lifeboat to stat), the text carries an
explicit `Open question (adr-35)` note rather than an invented answer.
