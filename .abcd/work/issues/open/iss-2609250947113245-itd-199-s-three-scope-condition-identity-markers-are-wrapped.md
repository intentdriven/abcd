---
schema_version: 1
id: "iss-2609250947113245"
slug: "itd-199-s-three-scope-condition-identity-markers-are-wrapped"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-199-a-reading-is-about-something-narrower-than-everything-its.md"
---

itd-199's three scope-condition identity markers are wrapped in backticks in its shipped record, so the claim reader treats each as documentation of the marker grammar rather than an identity: ParseClaims reports zero conditions, the verdict dispositions keyed to cond-2608312031029678, cond-2608312031028702 and cond-2608312031020321 key to nothing the record carries, and abcd intent condition refuses to re-disposition cond-2608312031028702, the worked example its own spec names
