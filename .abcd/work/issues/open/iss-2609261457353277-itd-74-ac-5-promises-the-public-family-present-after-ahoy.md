---
schema_version: 1
id: "iss-2609261457353277"
slug: "itd-74-ac-5-promises-the-public-family-present-after-ahoy"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-74"
origin: researcher-authored
production_mode: hand-written
remedy: "Waits on ruling D3 item 3, which rides with the itd-159 planning of iss-2609100506269348: if (a), move the public banned-names config out of .abcd/; if (b), the public fence carries one un-ignore line for .abcd/docs-lint.json; if (c), add an Audit Notes line to shipped itd-74 saying a public-visibility repository with nothing committed under .abcd/ relies on the private layer, and ahoy's gap report says so. Prove (a) or (b) by turning TestPublicFamilyUnderPublicVisibility into a test that the family is written and not ignored."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (rulings-owed D3, narrowing a shipped promise; run A 2026-09-28): Under visibility public, where no .abcd/ file is committed and the fence ignores the whole namespace, should the public banned-names config move outside .abcd/, should the fence carry one un-ignore for it, or should itd-74 ac-5 be amended so a public-visibility repo relies on the private layer alone?"
---

itd-74 ac-5 promises the public family present after ahoy scaffolding; where the docs-lint config path is gitignored (TestPublicFamilyUnderPublicVisibility) install writes no public family and reports the gap as unresolvable, leaving that repo with no CI-enforced banned-names family

## Remedy grounds (2026-09-29)

- Why: the three answers D3 item 3 lists, each with its change; the ruling is unanswered and none is picked. The fence is the same one iss-2609100506269348 reports, so one planning interview should settle both.
- Rejected: taking the private layer as the whole answer without amending ac-5, which would leave a shipped promise false.
