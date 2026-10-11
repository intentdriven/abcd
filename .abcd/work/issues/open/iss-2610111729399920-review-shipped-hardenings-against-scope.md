---
schema_version: 1
id: "iss-2610111729399920"
slug: "review-shipped-hardenings-against-scope"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security triage and fix run, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/execpins.go"
remedy: "For each instance, ask the rule's question (would the risk exist without abcd? does the hardening change how the repository works?) and record the answer on this issue; restore the repository's own behaviour where the hardening changes it and prevents nothing the person's own git would not run anyway, and keep it where it does; prove each restored behaviour with a test"
---

Several hardenings shipped in v0.13.4 switch a repository's own git settings off for abcd's commands, and need re-assessing against the security scope rule adopted on 2026-10-11 (principle abcd-fixes-what-it-adds: abcd never changes how a repository works on its own, and prevents an existing risk only where it can do so without such a change). Known instances: a sync's merge makes every configured merge driver git's built-in merge (gitutil.MergeDriverOverrides), so a repository's own lockfile or union-style driver is ignored on abcd's sync; the pick commit runs with commit.gpgsign=false and hooks off (gitutil.ExecPins via pickGit), so it is unsigned where the repository requires signed commits; the toplevel check refuses a checkout whose core.worktree names an ancestor, which refuses a deliberately configured layout. Reads that blank content filters (gitutil.FilterOverrides) change no repository behaviour and are likely fine.
