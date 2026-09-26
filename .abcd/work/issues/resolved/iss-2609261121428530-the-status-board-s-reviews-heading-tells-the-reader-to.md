---
schema_version: 1
id: "iss-2609261121428530"
slug: "the-status-board-s-reviews-heading-tells-the-reader-to"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/board_reviews.go"
resolution: "The board's text render lists the dated reviews alone, asks for a re-run only when a review is marked !, and folds the release receipts into one line naming their count, how far behind the oldest release gated is, the pins this history no longer holds, and that --json lists each; the JSON is unchanged."
impact: fix
resolved_by:
  commit: "c9bbc85e"
---

The status board's reviews heading tells the reader to 're-run those marked !' (internal/surface/cli/board_reviews.go), but on this repository every flagged row is a release receipt, which nobody re-runs: a receipt gates the release it names, RD002 forbids removing it, and every past release's receipt is permanently past the threshold. The imperative is untrue for a receipt, and the text render grows the bare board by one permanently flagged row per release. The receipt's staleness should be worded truthfully (the release it gated is N commits behind), the imperative kept for a dated review, and the text render (not --json) capped so past-release receipts collapse to one summary line.

## Grounds

- pursued: the bare board no longer asks for a re-run of a receipt and no longer grows a row per release; a receipt row or a re-run instruction for a receipt in the text render would show it wrong
