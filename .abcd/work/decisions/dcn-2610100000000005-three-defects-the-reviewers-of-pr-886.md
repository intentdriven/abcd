---
id: dcn-2610100000000005
slug: three-defects-the-reviewers-of-pr-886
date: 2026-10-10
---
2026-10-10 — Three defects the reviewers of PR #886 (itd-2609201925079472, run-2610100550297632) noted below their bar are deferred, not fixed in the change that set the bundled pause to zero: iss-2610100913086397 (a rate-limited validator whose partial return cannot be saved aside leaves its lane stuck, its named `--restart` refused), iss-2610100913088696 (the rate-limit checkpoint matches an await by receipt path alone, against its comment) and iss-2610100913096412 (the run-start quota query has no deadline and its error is not home-redacted). Reason: each needs a tested change to the rate-limit and budget code in `internal/core/implement/loop/`, which steps 3 to 5 of spc-2609212015054359 (itd-82, the drain) rework next, and none is reachable in a run today without a second fault: a refused save-aside on a rate limit, an operator restart racing the dispatch, or a quota-reporting runner, of which none ships. They are taken up with the rate limit's own wait (iss-2610100602595553), or by the drain's steps where those touch the same code.
