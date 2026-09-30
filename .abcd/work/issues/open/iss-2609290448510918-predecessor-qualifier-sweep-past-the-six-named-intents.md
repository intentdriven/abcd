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
remedy: "Read each remaining site against the live spec its id collides with and qualify the predecessor-store ones '(predecessor store)' per the specs charter's Two spc-N Namespaces rule, after the branches carrying itd-82 and itd-130 land (the five sites named in the progress section below); grounds: the charter is the rule, and the census below (ordinal ids at or below spc-70, not the intent's own spec, no qualifier on the line) reproduces the set."
---

The predecessor-store qualifier sweep reaches past the six intents iss-2609261536147903 named: outside drafts/ and those six, 192 citation sites across about forty intents name a spc-N at or below spc-70 that is not the citing intent's own spec, with no '(predecessor store)' on the line. Some are live cross-references (the cold-reading family's intents cite each other's live specs), and some are the predecessor store's (itd-4, itd-6, itd-29, itd-47 and itd-49 describe pre-rebuild work in its terms), so each site needs the same reading against the live spec it collides with; the specs charter's Two spc-N Namespaces section is the rule. Found while sweeping the pattern of iss-2609261536147903 in lane drainDrift3; a census script over intents/{planned,shipped,disciplines} reproduces the count.

## Progress 2026-09-30 (lane drainCitations, run A)

A census at 2b9d52fbb over `intents/planned`, `intents/shipped`, `intents/disciplines` and `intents/superseded` (an ordinal `spc-N` at or below spc-70, not the citing intent's own spec, on a line without the qualifier) counts 186 sites across 42 intents: 125 outside superseded/ and 61 inside it, where itd-29, itd-47 and itd-49 now sit. The first count of 192 was not scripted in the record, so the six-site difference is not traced. Every site was read against the live spec it collides with.

- Predecessor store, now qualified (94 sites, 12 intents): itd-4 (spc-20 to spc-23), itd-6 (spc-2, spc-4, spc-5), itd-36 (spc-38, spc-39), itd-50 (spc-52), itd-65 and itd-66 (spc-64, spc-27), and superseded itd-17, itd-20, itd-27, itd-29, itd-47 and itd-49. Two shipped acceptance criteria gained the qualifier, itd-4's drift criterion and itd-65's fail-closed criterion; each names an owner or a precedent, not a promise, so both are wording clarifications. The Implementing specs sections of itd-36, itd-4 and itd-47 also misstated where those ids stand, captured and fixed as iss-2609300025372991 and iss-2609300032219282.
- Live cross-references, correct as written (72 sites): the cold-reading family's citations of spc-55 to spc-69 (itd-177 to itd-189, itd-2609020625400194, itd-2609020625400445), itd-4's audit notes on spc-24, and itd-3, itd-80, itd-94, itd-101, itd-121, itd-132, itd-133, itd-147, itd-160, itd-161, itd-2609091416295622, itd-2609111003026787 and itd-2609231013154443.
- Data, left as written (15 sites): the `routed_from` frontmatter of itd-48, itd-50 and itd-53; itd-27's dated reclassification_history reason; fixture strings quoted as evidence in itd-4, itd-28, itd-186 and itd-2609111003026787; the example resolution note inside itd-4's resolve criterion; and itd-4's audit-note line that already says "of the retired record system".

Skipped, because other branches carry these intents: itd-82 (one site, spc-24) and itd-130 (four sites, spc-35). Read in place, all five cite live specs (itd-119's promote and itd-132's data directory), so none looks owed a qualifier. The other intents named for skipping (itd-111, itd-148, itd-24, itd-103, itd-2609081951381895, itd-2609211116005482, itd-2609212103568351 and itd-2609212103572513) hold no site in the census. The record stays open until those five sites are confirmed at the merged tip.
