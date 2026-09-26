---
schema_version: 1
id: "iss-2609261121375292"
slug: "the-reviews-charter-gate-passes-three-summaries-the-board"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd28"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-reviews.sh"
---

The reviews-charter gate passes three summaries the board reads as unpinned: scripts/check-reviews.sh RD001 tests the summary with [ -f ], which follows a symlink, while reviews.Read maps a symlinked 00-summary.md (fsutil.ErrNotRegular) to no pin; RD004 reads a summary of any size while reviews.Read caps it at maxSummaryBytes (1 MiB) and shows a larger one unpinned; and bash read drops a NUL byte from the frontmatter line, so a pin line carrying one passes RD004 while reviews.Pin refuses it. Each is gate-green and board-unpinned, the looser polarity; the gate should refuse all three so the board and the gate agree about every folder.
