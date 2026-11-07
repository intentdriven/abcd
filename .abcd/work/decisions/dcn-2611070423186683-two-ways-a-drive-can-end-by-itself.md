---
id: dcn-2611070423186683
slug: two-ways-a-drive-can-end-by-itself
date: 2026-11-07
---
2026-11-07 — Two ways a drive can end by itself around a resuming start stay in this release, and both are deferred to the follow-up bugfix release. A resuming start with no bound, or one tried again, that meets a launch giving no verdict for more than a few minutes ends with exit 2, where the loop would wait the same outage out. A harness file removed during a limit wait at a resuming proof ends as a wall failure, where the loop waits and re-proves. Both need an outage or a harness update in the minutes around a resuming start, which comes about once per release, and each stop says what ended the run. Changing what a resuming start waits on needs its own review, so it isn't folded in this late. The follow-up: a resuming start waits on no verdict as the loop does, and the limit wait at a proof uses the loop's check.
