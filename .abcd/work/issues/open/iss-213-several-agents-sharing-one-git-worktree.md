---
schema_version: 1
id: "iss-213"
slug: "several-agents-sharing-one-git-worktree"
severity: "major"
category: "process"
source: "user-observation"
found_during: "install-test round with concurrent agents (2026-08-11)"
found_at: ".abcd/work/CONTEXT.md"
deferred_after: v0.11.1
deferral_reason: "Carried by the planned itd-148, rewritten on 2026-09-29 to the product thinker's rulings and now blocked on the store draft itd-2609091014076309, which owns adding, listing and clearing away worktrees; the per-agent worktree direction this record proposed is the convention in AGENTS.md meanwhile. Owed: the store draft's planning interview with the product thinker, before which nothing from the store draft is built (ruling Q4, 2026-09-29); itd-148 waits on the store's verbs through its blocked_by (ruling Q2)."
---

Several agents sharing ONE git worktree silently invalidated a verification result and came close to losing committed work. Observed repeatedly during the 2026-08-11 install-test round, in a repo that is about to run more agents, not fewer.

What happened, in one session: the working tree was switched under an in-flight agent three times (to main, to a second agent's branch, and to a dependabot branch); a "git rebase origin/main" silently rebased MAIN because the branch had changed underneath between the checkout and the rebase; a "git checkout" failed with "error: could not detach HEAD" as a transient race against another agent's git process, succeeding on retry with no lock present; and a "make preflight" running in the background spanned two branch switches, so its exit 0 described no tree in particular.

The last one is the dangerous member of the set. A long verification (preflight is several minutes here) that reports clean while the tree changes beneath it is a FALSE GREEN of the kind the loud-staging principle exists to refuse — and unlike a red result, nothing about it invites a second look. A change verified that way could merge on the strength of a run that never saw it. In this round the result was noticed and discarded, and CI was used as the authority instead; that was luck and vigilance, not a property of the setup.

The near-miss on work: two commits existed only on a local branch while another agent was pruning branches and worktrees. Pushing early is what protected them, which is a habit rather than a guarantee.

Directions, none adopted. Give each agent its own git worktree (git worktree add), so branch state is per-agent and the whole class disappears — the harness already supports worktree isolation for subagents. Or, if a shared tree is kept, treat any verification longer than a moment as untrustworthy and defer to CI, and have agents assert the expected branch immediately before and after a long-running gate rather than assuming it held. Worth settling before the next multi-agent round rather than after the first bad merge.

## Deferral 2026-09-29

Deferred past v0.11.1: Carried by the planned itd-148, rewritten on 2026-09-29 to the product thinker's rulings and now blocked on the store draft itd-2609091014076309, which owns adding, listing and clearing away worktrees; the per-agent worktree direction this record proposed is the convention in AGENTS.md meanwhile. Owed: the store draft's planning interview with the product thinker, before which nothing from the store draft is built (ruling Q4, 2026-09-29); itd-148 waits on the store's verbs through its blocked_by (ruling Q2).
