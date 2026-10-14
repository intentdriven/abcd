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
resolution: "Each instance asked the rule's question (would the risk exist without abcd? does the hardening change how the repository works?). The pick commit's hooks off and commit.gpgsign=false, and the sync merge's merge.verifySignatures=false, hooks off, unsigned merge commit and MergeDriverOverrides: every program they kept from running runs on the person's own next commit or merge with no abcd involved, so the risk exists without abcd, and switching them off changed how the repository commits and merges; restored. The pick commit and the sync's merge now go through commitGit, built on gitutil.WorkingTreeCommitCommand, which is WorkingTreeCommand less the core.hooksPath, commit.gpgsign and merge.verifySignatures pins; MergeDriverOverrides and BuiltinMergeDriver are removed, having no other caller. A sync the repository refuses before the merge begins (merge.verifySignatures over an unsigned tip) is now refused with git's reason instead of failing on an abort of a merge that never started. Kept: ExecPins and FilterOverrides on abcd's read-only commands, which are abcd's own route (the person's git would not run those reads), and the core.worktree ancestor refusal, which bounds what abcd reads and changes nothing about the repository. Tests: TestThePickCommitRunsTheRepositorysHooks, TestThePickCommitAndTheSyncMergeAreSignedAsTheRepositorySays, TestTheSyncMergeUsesTheRepositorysMergeDriver, TestASyncTheRepositoryRefusesIsRefusedWithGitsReason, TestWorkingTreeCommitCommandLeavesTheRepositorysCommitSettings. Impact fix, as the comparable restore of the person's own git behaviour (iss-2610111729375108, Git LFS in lanes) was marked."
impact: fix
---

Several hardenings shipped in v0.13.4 switch a repository's own git settings off for abcd's commands, and need re-assessing against the security scope rule adopted on 2026-10-11 (principle abcd-fixes-what-it-adds: abcd never changes how a repository works on its own, and prevents an existing risk only where it can do so without such a change). Known instances: a sync's merge makes every configured merge driver git's built-in merge (gitutil.MergeDriverOverrides), so a repository's own lockfile or union-style driver is ignored on abcd's sync; the pick commit runs with commit.gpgsign=false and hooks off (gitutil.ExecPins via pickGit), so it is unsigned where the repository requires signed commits; the toplevel check refuses a checkout whose core.worktree names an ancestor, which refuses a deliberately configured layout. Reads that blank content filters (gitutil.FilterOverrides) change no repository behaviour and are likely fine.
