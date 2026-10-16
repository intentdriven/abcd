---
id: dcn-2607180000000002
slug: acceptance-criteria-sign-off-is-implicit
date: 2026-07-18
---
2026-07-18 — Acceptance-criteria sign-off is implicit in a human running
`abcd intent plan` (itd-94): no `ac_confirmed` frontmatter field, keeping
directory-as-truth pure and adding no forgeable schema surface. The gate
(`abcd intent ready`) distinguishes only drafts vs planned; agents are barred
from unattended planning at the protocol layer (run-protocol step 0 +
`/abcd:intent`'s interview script: `plan` is never run without the human's
explicit in-session confirmation). Escalation path if violations appear: an
`ac_confirmed_by:` field is lint-legal today and slots in as a fifth check.
