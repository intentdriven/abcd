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
resolution: "The reviews-charter gate refuses a symlinked review folder or 00-summary.md (RD001), a summary past 1 MiB and a NUL byte anywhere in the frontmatter block (RD004), the shapes reviews.Read shows unpinned; nine cases in check-reviews-cases.sh, run under bash 5 and /bin/bash 3.2, and TestReadShowsUnpinnedWhatTheGateRefuses pin both halves."
impact: internal
resolved_by:
  commit: "f8feb0d4"
---

The reviews-charter gate passes three summaries the board reads as unpinned: scripts/check-reviews.sh RD001 tests the summary with [ -f ], which follows a symlink, while reviews.Read maps a symlinked 00-summary.md (fsutil.ErrNotRegular) to no pin; RD004 reads a summary of any size while reviews.Read caps it at maxSummaryBytes (1 MiB) and shows a larger one unpinned; and bash read drops a NUL byte from the frontmatter line, so a pin line carrying one passes RD004 while reviews.Pin refuses it. Each is gate-green and board-unpinned, the looser polarity; the gate should refuse all three so the board and the gate agree about every folder.

## Grounds

- pursued: the gate now refuses every summary the board reads as unpinned; a summary the board shows unpinned that check-reviews.sh passes, under either bash, would show it wrong
