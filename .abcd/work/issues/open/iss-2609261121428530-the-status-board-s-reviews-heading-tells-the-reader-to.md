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
---

The status board's reviews heading tells the reader to 're-run those marked !' (internal/surface/cli/board_reviews.go), but on this repository every flagged row is a release receipt, which nobody re-runs: a receipt gates the release it names, RD002 forbids removing it, and every past release's receipt is permanently past the threshold. The imperative is untrue for a receipt, and the text render grows the bare board by one permanently flagged row per release. The receipt's staleness should be worded truthfully (the release it gated is N commits behind), the imperative kept for a dated review, and the text render (not --json) capped so past-release receipts collapse to one summary line.
