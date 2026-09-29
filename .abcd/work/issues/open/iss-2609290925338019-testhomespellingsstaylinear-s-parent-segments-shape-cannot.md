---
schema_version: 1
id: "iss-2609290925338019"
slug: "testhomespellingsstaylinear-s-parent-segments-shape-cannot"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: verify-fix4-guardResid"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/homeresiduals_test.go"
---

TestHomeSpellingsStayLinear's parent-segments shape cannot see the lexical fold: cleanSeparators, foldParents and literalSegment tally no work, so a fold that copies its stack per pop (quadratic) passes the pin whose name says it holds the spelling linear.
