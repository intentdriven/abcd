---
id: dcn-2608220000000006
slug: bug-hunt-round-7-merge-gate-pr-415
date: 2026-08-22
---
2026-08-22 — Bug-hunt round 7 merge gate (PR #415, resumed after the round's
session ended between PR-open and gate). origin/main (the site-build and
itd-112 merges) merged into bughunt-b/round-7 cleanly; baseline green on the
merged tree; all 12 CI checks green. Dual pre-merge review: the Opus 5
reviewer returned NO-SHIP with three remediable blockers, the Fable 5
reviewer SHIP with the same frontmatter residual flagged non-blocking. All
three remediated with tests watched to fail under mutation of each fix: the
update dispatch-refusal receipt now redacts target_path at construction
(refusalReport seam; it had shipped the raw home root beside the redacted
detail), capture list --json now emits [] not null for empty issues/skipped
(the CHANGELOG claim is now true), and the repolint cap+1 grown-file branch
is pinned through a capRead seam (the prior boundary test passed against the
pre-fix read, so nothing guarded the +1 — the round-7 "closed by
construction" exception is retired). Residuals captured open, not widened
into the PR: iss-2608221126066379 the frontmatter BOM tolerance now diverges
from the sibling parsers whose comments promise byte-exact parity
(fail-closed, comment-invariant falsified); iss-2608221126066631 the guard
process-substitution redirection family still allows a glued blocker flag
(pre-existing, same family as the fixed &>). The reviewer's probe-horizon
concern was refuted in-session: isBinary self-caps at 8 KiB, so both
oversize branches agree.
