---
schema_version: 1
id: "iss-2611262226253325"
slug: "while-the-person-is-away-the-stop-hook"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy"
remedy: "Have the owed-work reader apply land's stacked check: a pull request whose branch carries commits of another open pull request not on the default branch is owed nothing until that base merges, and the stop check names it as waiting on its base (as it already does for a stacked branch with no pull request); test a stacked pull request reads as waiting, not owed an arm."
---

While the person is away, the stop hook names a stacked pull request as owed an arm ('pull request #1817 (reserve-comma-test-1126) is behind its base, which a merge queue guards, and auto-merge is not armed: arm it'), but abcd land refuses to arm that same pull request because its branch carries commits of another open, armed pull request (#1815) not yet on main. The session can do nothing to move the item until the base merges, so the hook holds every stop on an act land forbids (2026-11-26, 22:35 GMT).
