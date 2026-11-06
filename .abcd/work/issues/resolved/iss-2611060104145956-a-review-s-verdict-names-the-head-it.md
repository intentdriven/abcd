---
schema_version: 1
id: "iss-2611060104145956"
slug: "a-review-s-verdict-names-the-head-it"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "reading how a review's verdict is recorded"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/review.go"
remedy: "Read every review return by one rule that holds its verdict to naming the head in full, refuse other shapes with instructions, count only receipts read under that rule, and set aside a judge's return when its launch stopped short."
resolution: "a review's verdict names the head it judges in full, read by one rule in abcd land --record-review and the build loop; a return of the older shape is refused with instructions, a receipt recorded before the rule is recorded again, and a judge whose launch stopped short does not count"
impact: breaking
resolved_by:
  commit: "497e0231ac1ad4d3a9d7364faa05ec1e30fac739"
---

A review's verdict names the head it judges, in full. abcd land --record-review and the build loop read a return by one rule: the first line under the ### Verdict heading opens with the verdict and names the head, or a Head: line directly under the heading names it, and a plan review names its draft's digest the same way. A line opening a verdict in any other shape, a qualifier included, refuses the return. A return of the older shape is refused with instructions for writing it again. A receipt recorded before this rule does not count at the merge gate or as a delta's base, so the pair is recorded again. A judge whose launch reports it stopped short, for length among other reasons, does not count, and the step runs again.
