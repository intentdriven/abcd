---
schema_version: 1
id: "iss-2611101449088556"
slug: "memory-recall-s-take-can-send-a-smaller"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "lane t review round 3, autonomous run 2026-11-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recall/recall.go"
remedy: "Decide which holds, the ranking invariant or the fuller send, then either close the full phase when an over-budget candidate's line does not fit, or reword the comment and docs; add a test with over-budget pages ranked above a small item whose lines do not fit."
---

Memory recall's take can send a smaller candidate in full below one whose line did not fit. In take.next (internal/core/recall/recall.go) a candidate whose full block is larger than the whole turn budget skips the full phase without closing it; if its line then does not fit either, only the line phase closes, so a smaller candidate ranked below it still comes back in full. The comment above the take says no candidate comes back in a larger form than one ranked above it, which this breaks. It shows in the fresh and moderate rows when pages over the turn budget rank above a small item. No test covers it: the existing test of an over-budget item covers only the case where its line fits. Found by a reviewer of the memory docs on 10 Nov 2026; the docs were worded to match the code as it is.
