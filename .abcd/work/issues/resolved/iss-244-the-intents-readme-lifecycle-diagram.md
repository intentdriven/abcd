---
schema_version: 1
id: "iss-244"
slug: "the-intents-readme-lifecycle-diagram"
severity: "major"
category: "process"
source: "user-observation"
found_during: "intent-planning-interview"
found_at: ".abcd/development/intents/README.md:66-120"
resolution: "README now marks the ship design-target explicitly; the overclaim is gone"
impact: internal
---

The intents README lifecycle diagram documents four things the binary does not have: an /abcd:intent ship verb, an /abcd:intent reclassify verb, an intent_lifecycle_hook, and the bundle-member and discipline paths of intent plan. abcd intent plan takes exactly one id and silently defaults kind to standalone with no classifier, no kind confirmation and no plan-review; the shipped verb set is intent create, intent plan, intent ready, intent link, intent review, intent review ingest and spec close. The diagram also frames the lifecycle as 'mostly automatic, with deliberate manual steps' when it is mostly manual with three transition verbs, and its step 1 claims intent create refuses without acceptance criteria when the refusal actually lives in intent plan. A committed record describing an unbuilt future as present reality is the inverse of change-narration and misleads every session that reads it.