---
schema_version: 1
id: "iss-2609292057441479"
slug: "itd-111-acceptance-criteria-3-and-5-name"
severity: "nitpick"
category: "drift"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-111 follow-up"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently.md"
resolution: "itd-111's Audit Notes now name the surface each drifted criterion is met by: criterion 3 by abcd --version (and abcd ahoy), criterion 5 by abcd update --check; the criterion text stands as shipped, per ruling H10."
impact: internal
resolved_by:
  commit: "0915d0279"
---

itd-111 acceptance criteria 3 and 5 name command spellings the CLI no longer answers to: AC3 says the vintage report comes from 'abcd version' and AC5 says the explicit release check is 'abcd version --check' (design decision 6 and its open question say the same). Since itd-2609212130136102 consolidated the verbs, the report is 'abcd --version' and the check is 'abcd update --check'; both old spellings refuse with a moved notice naming the new one. The behaviour each criterion asks for is present at the new spelling, so the shipped criteria read true only once the record says which surface meets them. adr-38 tier 2 also cites 'abcd version --check' as an example of an explicit ask; the rule it states still holds.

## Grounds

- pursued: a reader of itd-111 who runs the spelling its criteria name finds the moved notice and, in the intent's Audit Notes, the surface that meets each criterion; shown wrong if either named surface stops rendering the vintage report or the release check
