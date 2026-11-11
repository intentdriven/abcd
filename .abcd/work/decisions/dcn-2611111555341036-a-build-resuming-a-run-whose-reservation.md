---
id: dcn-2611111555341036
slug: a-build-resuming-a-run-whose-reservation
date: 2026-11-11
---
2026-11-11 — A build resuming a run whose reservation is refused because the record already sits finished on origin's default branch goes on to close the run's merged lanes only when closing them all finishes the run, then reports the run finished with nothing more started; otherwise the reservation's refusal stands and nothing is removed; where a later lane's removal is refused, an earlier lane's closing is kept. The route is taken on that one refusal alone, told by the class of the reservation's own refusal (claim.ErrFinished), never by a read of the checkout's refs: any other refusal of the reservation (another holder, origin unreachable, a lock error, an unreadable record) stands and removes nothing. Without this, the last lane's merge always finishes the record, the resume's reservation always refuses, and a single-lane run whose pull request merged could never be closed by the build (iss-2611101317510356).
