---
id: dcn-2611211501156446
slug: decided-by-the-run-for-owner-review
date: 2026-11-21
---
2026-11-21 — Decided by the run for owner review: private advisories are investigated before any verdict is proposed, with an asymmetric review bar. One independent security reviewer checks each advisory against current main (confirmed, already fixed or refuted, with file and line evidence, and whether the route exists without abcd, proved by a scratch test where not obvious). A closing verdict (not-abcds or not-a-vulnerability) is proposed only when a second reviewer, sharing no context with the first and not told who filed the advisory, independently agrees; a split goes to the person as an open question. Grounds: a wrongly closed advisory hides a real hole while a wrongly confirmed one costs only an unneeded fix; the August advisory-handling pilot used one refuting reviewer, and closures carry a justification as VEX not-affected statements do. The run records no verdict: each proposal is queued for the person, who records it with embargo triage. To reverse: drop the second-reviewer requirement, or record verdicts on one reviewer's report.
