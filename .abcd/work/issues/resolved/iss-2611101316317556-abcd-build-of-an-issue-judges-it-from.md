---
schema_version: 1
id: "iss-2611101316317556"
slug: "abcd-build-of-an-issue-judges-it-from"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/check.go"
remedy: "Have build judge an issue from origin's default branch after a fetch, as the drain's own move does, and when the local ledger disagrees say that the checkout is behind and by how many commits; a test builds an issue present only on origin and is not refused."
resolution: "abcd build <iss-N> now fetches origin, with the same bound as the drain move's fetch, and judges the issue from the ledger on origin's default branch (gitutil.DefaultRef), reserving it with that record's found_at. When this checkout's own ledger disagrees, the eligible row says the checkout is N commit(s) behind. Where origin cannot be read, the build judges from the checkout's own ledger as before and says why. A checkout with no remote named origin behaves as before."
impact: fix
resolved_by:
  commit: "2bd2647e43293ee461353ff944e21ec0cfe274c1"
---

abcd build of an issue judges it from the ledger of the checkout it runs in, without fetching and without saying that the checkout is behind origin, so it refuses an issue that origin's default branch holds. issueCheck (internal/core/implement/loop/check.go) plans the drain from the working tree unless a drain handed it origin's snapshot. On 10 Nov at 08:54 build from the main checkout refused iss-2611100548341325 as not an open issue, because that checkout's ledger was behind origin/main and the record had arrived in #1478. The same happened on 7 Nov at 14:25 and on 8 Nov at 04:35, when the main checkout was 104 commits behind and could not be fast-forwarded while drives might run from it. Each time the issue was reserved by hand. iss-2611010953085251 covers the same stale read at the land stage only.

## Grounds

- pursued: a build of an issue that only origin's default branch holds open starts and is reserved with origin's found_at, and a build of an issue origin has resolved is refused at the eligible check with the behind count (TestABuildOfAnIssueOnOriginAloneIsJudgedFromOrigin, TestABuildOfAnIssueFinishedOnOriginIsRefusedSayingTheCheckoutIsBehind). It would be shown wrong by a behind checkout refusing an issue origin holds open, by a build reserving with a found_at origin's record does not name, or by a build hanging or refusing silently when origin cannot be reached (TestABuildThatCannotFetchOriginJudgesFromTheCheckoutAndSaysSo).
