---
schema_version: 1
id: "iss-2610151855227496"
slug: "abcd-land-s-single-run-push-and-then"
severity: "minor"
category: "bug"
source: "manual-test"
found_during: "abcd land real run on PR #1020, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd land's enqueue step, right after its own push"
remedy: "After abcd land's own push changes the remote head, re-read the pull request's head from the forge a bounded number of times (every 2 s, for up to 30 s) until it equals the pushed commit, and only then decide whether to arm; on timeout keep today's stop and its wording. Test it with a fake forge whose first answers lag, and re-check it live."
resolution: "abcd land now re-reads the pull request's head every 2 s for at most 30 s after its own push moves the branch, and arms once the forge shows the pushed commit; on timeout it keeps today's stop and says how long it waited"
impact: fix
---

abcd land's single run, push and then queue, stops on its most common path. On 2026-10-15 it pushed feat/abcd-land at 2abcab16c (18:54:26Z), then in the same run read pull request #1020 and the forge still reported the old head 572cd8c0f, so the enqueue step stopped with 'wait for the forge to see the push' and nothing was armed. A re-run 12 seconds later armed it. Refusing to arm on a stale head is the safe failure and stays, but since a push that moves the head is the normal case, the person must usually run abcd land twice. Found by a real run; the fake forge answers at once, so no test could show it.

2026-10-16, a second occurrence: abcd land pushed feat/abcd-land at a5e880e4b at 23:43:55Z (UTC, the night of 2026-10-15) and, in the same run, read pull request #1020 at the old head 2abcab16c, so it stopped; the re-run 14 seconds later saw the new head and left the existing arming in place. Two of two head-moving runs have now hit the lag.
