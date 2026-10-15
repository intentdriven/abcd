---
id: dcn-2610110000000003
slug: security-work-is-scoped-by-one-rule-the
date: 2026-10-11
---
2026-10-11 — Security work is scoped by one rule, the product thinker's ruling of the day: abcd fixes the security risks it adds, never changes how a repository works on its own (its hooks, content filters, large-file storage, merge tools and remote transport), and helps prevent a risk that exists without abcd only where it can do so without such a change; otherwise the finding is closed with that reason. Recorded as the principle `abcd-fixes-what-it-adds`, injected into security sessions by the `SECURITY` domain in `.abcd/rules.json`, and stated to reporters in `.github/SECURITY.md`. Fixes already shipped that switch a repository's own settings off for abcd are to be reviewed against it separately.
