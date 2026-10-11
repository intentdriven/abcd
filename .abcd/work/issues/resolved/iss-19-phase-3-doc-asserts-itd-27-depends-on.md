---
schema_version: 1
id: "iss-19"
slug: "phase-3-doc-asserts-itd-27-depends-on"
severity: "major"
impact: internal
category: "inconsistency"
source: "agent-finding"
found_during: "intent-dependency-sweep"
found_at: ".abcd/development/roadmap/phases/phase-3-intent.md"
resolution: "Direction adjudicated from the intents themselves: itd-42 declares blocked_by: [itd-27] and its press release corrects the grill itd-27 built (renaming --with-docs, adding --coherence); itd-27 records 'Extended by: itd-42'. The phase-3 doc's reversed dependency bullet and its open-question ordering line are corrected to itd-27 -> itd-42, citing both intents' declarations."
---

phase-3 doc asserts itd-27 depends on itd-42 while itd-42's own prose asserts the reverse (it extends the grill itd-27 built); adjudicate direction — first live catch for the itd-78 phase-consistency lint