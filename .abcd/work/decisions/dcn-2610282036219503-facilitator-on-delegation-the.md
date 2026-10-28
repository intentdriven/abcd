---
id: dcn-2610282036219503
slug: facilitator-on-delegation-the
date: 2026-10-28
---
2026-10-28 — facilitator-on-delegation: the coordinating session (abcd-f5), on delegation from the technical facilitator, raises the check job's macOS leg ceiling in ci.yml from 70 to 85 minutes and moves its warning line from 60 to 75, keeping it ten minutes ahead; the ubuntu leg stays at 45. On 2026-10-28 merge-group run 40752008734 (PR #1196) had check (macos-latest) cancelled after about 71 minutes, from 19:11:30Z to 20:22:05Z, and the queue dropped the group with failed_checks, while that day's passing merge-group runs took roughly 46 to 68 minutes. 85 stays below the merge queue's 90-minute check response timeout. The lasting remedy is still a faster suite: iss-2610181711260494 named it and was resolved by the raise to 70, and the faster race pass stays open as iss-2610181757540736; this resolves iss-2610282035340363.
