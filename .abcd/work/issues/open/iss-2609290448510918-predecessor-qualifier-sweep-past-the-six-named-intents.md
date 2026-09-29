---
schema_version: 1
id: "iss-2609290448510918"
slug: "predecessor-qualifier-sweep-past-the-six-named-intents"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents"
deferred_after: "v0.11.1"
deferral_reason: "no ruling owed; carried past v0.11.1 by lane drainDrift3 (run A 2026-09-29) on its size: the 192 sites each need a reading against the live spec their id collides with, which the lane's time box did not hold after qualifying the six intents iss-2609261536147903 named."
---

The predecessor-store qualifier sweep reaches past the six intents iss-2609261536147903 named: outside drafts/ and those six, 192 citation sites across about forty intents name a spc-N at or below spc-70 that is not the citing intent's own spec, with no '(predecessor store)' on the line. Some are live cross-references (the cold-reading family's intents cite each other's live specs), and some are the predecessor store's (itd-4, itd-6, itd-29, itd-47 and itd-49 describe pre-rebuild work in its terms), so each site needs the same reading against the live spec it collides with; the specs charter's Two spc-N Namespaces section is the rule. Found while sweeping the pattern of iss-2609261536147903 in lane drainDrift3; a census script over intents/{planned,shipped,disciplines} reproduces the count.
