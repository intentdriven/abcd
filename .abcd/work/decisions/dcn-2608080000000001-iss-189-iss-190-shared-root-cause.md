---
id: dcn-2608080000000001
slug: iss-189-iss-190-shared-root-cause
date: 2026-08-08
---
2026-08-08 — iss-189/iss-190 shared root cause identified, fix designed but not
implemented: bug-hunt loop rounds 6-8 (state issue #197) each attempted a local
patch to scanAllPatterns/probeAt's 512-byte adjacency window
(internal/adapter/scanner/scanner.go) and each was BLOCKed on review for the
same underlying reason — a truncated, possibly-ambiguous window-edge view was
driving a discard/skip decision that reached further than the ambiguity itself
(round 6: discarding a window-edge match lost the forward-chaining offset;
round 7: discarding lost a genuine shorter match, and its boundary classifier
mis-fired on multi-alternation patterns; round 8: an uncertain match end wrongly
suppressed stolenJunctions, a backward search that never depended on that offset
being exact). Recommended structural fix: replace the fixed-window single-shot
probe with a galloping/exponential-doubling probe (`trueMatchEnd`) that grows
the window only while a match keeps running into its edge, and stops the moment
the match ends short of the edge or the window reaches the real end of line —
so `m.end` is always the true end, never an artifact of truncation. This removes
iss-189's boundary-ambiguity question outright (no more guessing whether `\b`
is real, so no classifier to get wrong) and iss-190's "clipped, so skip
chaining" special case (nothing is ever clipped), without reintroducing round
6's cost regression — it only grows for matches that are genuinely still
growing, at the same amortized cost class the top-level unbounded match in the
same function already pays. Not yet implemented or reviewed; captured here so
the next attempt (an autonomous ROOT-CAUSE ESCALATION round, or a human) starts
from this direction instead of re-deriving it from the same three round
post-mortems.
