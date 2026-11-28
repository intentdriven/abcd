---
id: dcn-2611271952390075
slug: defer-the-partial-notes-renewal-line-to
date: 2026-11-27
---
2026-11-27 — Defer the partial-notes renewal line to C step 4. When only some holds of a run carry a left-to-lapse note, leaveToLapse reports the run left while leftToLapse needs every hold, so a drive end can say left to lapse for a hold still renewed. Both C step 3 reviews ruled this a note, not a blocker: a named renewer always renews and no hold lapses earlier than said. It predates step 3's last commit, and C step 4 rewrites the same end lines into the drive's state, so the fix is a per-issue clause there. Reverse by fixing it in step 3 before that lane lands.
