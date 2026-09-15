---
schema_version: 1
id: "iss-2608310954048572"
slug: "spc-62-states-that-internal-core-reading-already-locates-the"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "phase-0 criteria review, cold-reading cycle 1"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open/spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md"
resolution: "spc-62's prose claimed the definition locator already existed; the locator is now built in internal/core/reading and the sentence corrected to describe it"
impact: internal
resolved_by:
  intent: "itd-184"
  spec: "spc-62"
---

spc-62 states that internal/core/reading already locates the four cold-reading definitions and hashes them for spc-63 instrument identity, and it does not: no non-test file in that package references agents/ at all. A builder reading the spec would take the locator as existing and find nothing to call. The locator is unbuilt work that spc-62 must deliver, and the sentence is a prose claim about code behaviour of exactly the shape itd-195 refuses.

## Grounds

- pursued: the locator is spc-62's own delivery rather than a dependency it inherits, so building it here and correcting the sentence removes the trap; what would show this wrong is itd-185 finding the returned shape insufficient to stamp instrument identity
