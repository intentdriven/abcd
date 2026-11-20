---
schema_version: 1
id: "iss-2611060809158052"
slug: "04-surfaces-05-intent-md-154-says-the"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Say itd-190 has no `## Fit`, or add that section to the record."
refines: [iss-2610262030443173]
resolution: "The intent surface's two-idioms paragraph now says nine of the ten rule-idiom disciplines carry The gate, Fit and Staging, and that itd-190 has no Fit section."
impact: internal
resolved_by:
  commit: "42ee3c9c07aa01a6fe41ea7e3c349ff0423cd99e"
---

`04-surfaces/05-intent.md:154` says the ten disciplines written since itd-190 carry `## The gate`, `## Fit` and `## Staging`. Nine do. `.abcd/development/intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s.md` has `## The rule`, `## The gate` and `## Staging` and no `## Fit`.

## Grounds

- pursued: the brief's claim matches the discipline records; it would be wrong if any other discipline with a The rule heading lacked Fit, or if itd-190 gained a Fit section, which the lane's detector.sh would report
