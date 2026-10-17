---
schema_version: 1
id: "iss-2610111600048326"
slug: "gitfile-lets-core-worktree-name-parent"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-rh6r-8hjh-rvv3, filed 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Under abcd-fixes-what-it-adds this is not abcd's vulnerability: with the planted gitfile and core.worktree, plain git itself treats the parent as the checkout (rev-parse --show-toplevel prints it, git add -A stages its files; tested 2026-10-11). A narrow refusal is bypassable from a linked worktree (extensions.worktreeConfig plus a per-worktree core.worktree) and refuses a symlinked .git that works today; a strict one refuses deliberate core.worktree setups; both change how repositories work. Prevention: the run tripwire of iss-2610111600058575 also watches core.worktree and extensions.worktreeConfig, per-worktree config included. Close GHSA-rh6r as not a vulnerability in abcd and move this record to wontfix with that reason when the tripwire lands"
blocked_by: [iss-2610111600058575]
wontfix_reason: "Not abcd's vulnerability under abcd-fixes-what-it-adds: with the planted gitfile and core.worktree, plain git itself treats the parent as the checkout (rev-parse --show-toplevel prints it and git add -A stages its files; tested 2026-10-11), so abcd following git's answer adds no route. Both available refusals change how repositories work (one is bypassable from a linked worktree and refuses a symlinked .git that works today; the strict one refuses deliberate core.worktree layouts). The run tripwire watches core.worktree and extensions.worktreeConfig, per-worktree config included, so a change during a run stops abcd's next git step."
---

A planted gitfile defeats the toplevel check that GHSA-6wfm-pfhv-w267 added: with `core.worktree=../..` in a checkout `co` and a one-line `<parent>/.git` reading `gitdir: co/.git`, `holdsGitDir` accepts the parent, so `Toplevel` and `CheckoutRoot` return the parent and every store, the implement run and its `git add -A` address a directory outside the checkout.

Private security advisory GHSA-rh6r-8hjh-rvv3 (draft, severity high). Full text, evidence and reproduction: .abcd/.work.local/scratch/security-drain-2026-10-11/advisories/ in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (main 3d171421a): `holdsGitDir` (internal/gitutil/repo.go) reads the gitfile's `gitdir:` line, joins a relative target onto the toplevel and accepts it on `os.SameFile`, with no check that the target lies inside the toplevel it is vouching for. `CheckoutRoot` returns that toplevel; `implementRun` assigns it to the run's RepoRoot. Confirmed by reading the code at that commit; not re-run.

## Grounds

- declined: Not abcd's vulnerability under abcd-fixes-what-it-adds: with the planted gitfile and core.worktree, plain git itself treats the parent as the checkout (rev-parse --show-toplevel prints it and git add -A stages its files; tested 2026-10-11), so abcd following git's answer adds no route. Both available refusals change how repositories work (one is bypassable from a linked worktree and refuses a symlinked .git that works today; the strict one refuses deliberate core.worktree layouts). The run tripwire watches core.worktree and extensions.worktreeConfig, per-worktree config included, so a change during a run stops abcd's next git step.
