---
schema_version: 1
id: "iss-2609291925149138"
slug: "the-status-line-role-label-waiting-product-thinker-waiting"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "rulings interview with the product thinker, 2026-09-29 (abcd-23 [b1e81b])"
origin: researcher-authored
production_mode: hand-written
remedy: "Before each question, classify its addressee (product thinker vs technical facilitator) and run the mode verb when it differs from the current label; state the rule in the GRILL domain and the interview command pages, and consider letting the question tool call carry the addressee so the label is set with the question itself."
resolution: "GRILL gains the rule: classify each question's addressee first (the product thinker or the technical facilitator), set the mode label with the mode verb when it differs, then ask; a mixed interview re-sets it per question. Pinned by TestGrillDomainCarriesQuestionVisibilityAndAddresseeRules. The planning interview page (commands/intent.md) carries the same line, pinned by TestIntentInterviewPageCarriesGrillVisibilityAndAddresseeLines. Letting the question tool call carry the addressee itself, which the remedy floats, is out of abcd's reach: the question tool belongs to the host."
impact: internal
resolved_by:
  commit: "5357c4c68128e1d656e0cb105391ea3893bcfb8a"
---

The status-line role label (waiting: product thinker / waiting: facilitator) must always name the person the CURRENT question is for. An agent set 'product thinker' at the start of a mixed interview and left it while asking technical-facilitator questions, so the badge was wrong. Assess each question's addressee first, then set the label, then ask.

Delivered: the GRILL rule in 5357c4c68; the interview command page line in 66b5512e9 (commands/intent.md); the question tool carrying the addressee is out of abcd's reach, since the host owns that tool's shape, and the mode verb set before each question is the half abcd holds.

## Grounds

- pursued: the status-line label is only true if it is set per question, from the question's own addressee, before the question shows; a purely internal convention (the mode verb, itd-200), so no outside practice is cited. Shown wrong if a mixed interview run under the rule still shows a label naming the other role.
