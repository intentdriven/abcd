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
resolution: "cleanSeparators, foldParents, literalSegment, collapseStars and dotParents tally their work. On a scratch copy, a fold that calls literalSegment on its whole joined stack per pop fails TestHomeSpellingsStayLinear/parent_segments (211.9 units per byte, growth 14.48x) and passes it with the tallies removed. Two new shapes, glob segments and working-directory parents, hold the new readings linear."
impact: internal
resolved_by:
  commit: "4ef5013bd"
---

TestHomeSpellingsStayLinear's parent-segments shape cannot see the lexical fold: cleanSeparators, foldParents and literalSegment tally no work, so a fold that copies its stack per pop (quadratic) passes the pin whose name says it holds the spelling linear.

## Grounds

- pursued: a quadratic fold fails the pin; a quadratic mutation of the fold that the pin passes would show it wrong
