---
id: dcn-2611070755104151
slug: make-precheck-chooses-audit-tests
date: 2026-11-07
---
2026-11-07 — make precheck chooses audit tests outside the command and site packages by what they do, using the test reuse classifier's reading of what reaches the repository tree, with a stop at any helper that starts a process. A list of names was the smaller edit but needed keeping by hand, and a second detector would have disagreed with the first. The command and site packages stay on the name pattern, because selecting by reach there would run hundreds of behaviour tests. A false positive costs seconds and a miss costs a preflight, so the rule leans towards choosing.
