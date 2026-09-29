---
schema_version: 1
id: "iss-2609292057441479"
slug: "itd-111-acceptance-criteria-3-and-5-name-command-spellings"
severity: "nitpick"
category: "drift"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-111 follow-up"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently-every-surface-that-runs.md"
---

itd-111 acceptance criteria 3 and 5 name command spellings the CLI no longer answers to: AC3 says the vintage report comes from 'abcd version' and AC5 says the explicit release check is 'abcd version --check' (design decision 6 and its open question say the same). Since itd-2609212130136102 consolidated the verbs, the report is 'abcd --version' and the check is 'abcd update --check'; both old spellings refuse with a moved notice naming the new one. The behaviour each criterion asks for is present at the new spelling, so the shipped criteria read true only once the record says which surface meets them. adr-38 tier 2 also cites 'abcd version --check' as an example of an explicit ask; the rule it states still holds.
