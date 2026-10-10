---
schema_version: 1
id: "iss-113"
slug: "convopenquestionssource-s-truncation"
severity: "nitpick"
impact: internal
category: "tech-debt"
source: "impl-review"
found_during: "itd-96 P1 review"
found_at: "internal/core/lifeboat/sources_conventions.go"
resolution: "Fixed on the itd-95 branch alongside the walk-bounding change that caused it: both truncation strings in convOpenQuestionsSource now name the walk cap as '(N entries, M levels deep)' rather than as a file cap alone, so the evidence matches the three dimensions the walk actually bounds."
---

convOpenQuestionsSource's truncation evidence names only the "%d-file walk cap", but WalkFiles now also truncates on its directory cap and its depth cap, so a walk stopped by either reports a cause it did not hit. The three affected strings are the searched note, the blank's qualified question, and the non-blank scan note. convInternalsSource words the same note as "walk cap (N entries, M levels deep)"; the two should say the same thing. Found by an independent security review of the itd-96 diff.