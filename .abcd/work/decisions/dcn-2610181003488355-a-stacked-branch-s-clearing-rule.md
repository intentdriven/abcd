---
id: dcn-2610181003488355
slug: a-stacked-branch-s-clearing-rule
date: 2026-10-18
---
2026-10-18 — A stacked branch's clearing rule includes review and works without abcd's preflight (facilitator-on-delegation, itd-2610180912189125 review round 2, minor findings; refines dcn-2610180936571036): a stacked branch whose base is still open is also owed while a review of its head is outstanding, that is, while the away listing shows the forge's review decision as CHANGES_REQUESTED or, for a run's lane, its run state records no passing review of its head, since the coordination prep's decision that a declared dependency constrains only the landing says its review runs now. Verified means a preflight receipt naming its head where the repository mints receipts (a receipt folder exists in one of its worktrees); in a repository without them, its pull request's checks passing in the away listing; and with no pull request there either, the branch is judged on being behind and on review alone, and the reason says it was not verified (grounds: plan review round 2, minor findings; the receipt lookup in internal/core/landing)
