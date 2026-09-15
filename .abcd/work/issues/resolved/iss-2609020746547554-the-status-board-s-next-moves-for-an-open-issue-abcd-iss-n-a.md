---
schema_version: 1
id: "iss-2609020746547554"
slug: "the-status-board-s-next-moves-for-an-open-issue-abcd-iss-n-a"
severity: "major"
category: "ux"
source: "review-followup"
found_during: "release-v0.7.1-docs-currency"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/record/record.go"
resolution: "The status board's next moves for an open issue print the promote and resolve remedies with the --grounds argument both routes require, so each runs as printed; TestDescribeIssueNextMoves refuses any promote or resolve next move that omits --grounds, and fails on the previous render."
impact: fix
---

The status board's next moves for an open issue (abcd <iss-N>, /abcd iss-N, internal/core/record/record.go) print two remedies a user copies and runs, and both refuse: 'abcd capture promote <iss-N>' and 'abcd capture resolve <iss-N> "<note>" --impact <...>' exit 2 with 'requires --grounds' because grounds have been mandatory on both routes since v0.7.0 (88b3da87). The wontfix remedy is correct as printed. The same class was fixed for the orphan-draft remedy in promote.go but this render was missed, and README's first-run section points readers at the board. Found by the v0.7.1 docs-currency pass as its one blocker.

## Grounds

- pursued: a printed remedy is an instruction the user copies, so it is held to the same rule as the orphan-draft remedy: it must run as printed, and the test that pins it reads the rendered text rather than the source
