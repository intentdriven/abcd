---
id: spc-2611262159212693
slug: a-person-sets-for-their-own-git-identity
intent: itd-2611252049031692
origin: researcher-authored
production_mode: hand-written
---
# The guard holds the committed mode line and "only your own"

## Summary

itd-2611252049031692's two places decide the mode, and the merge gate
(`internal/core/landing/facilitator.go`) holds both rulings the product
thinker made beside them on 2026-11-26, as built on the branch that planned
this spec: `abcd land` and the build loop arm no pull request whose diff, read
from its merge base with the default branch's tip to its head, changes the
`facilitator.mode` line of `.abcd/config.json`, in any mode, and say the
person merges it on the forge's page; and where `auto` is the person's own
global setting rather than the repository's, the gate arms only a pull request
the forge account abcd runs as opened, judging anyone else's as `human` judges
it (`TestAChangeToTheCommittedModeLineIsNeverArmed`,
`TestAPersonsGlobalAutoArmsOnlyTheirOwnPullRequest` in
`internal/core/landing`).

What is left is two things. First, the gate refuses such a pull request, but
an arming of abcd's own that already stands at its head is left in place, as
every refusal of the gate leaves it today; the intent's criterion asks that it
be taken back off. Second, the same two rules on the commands themselves. The shell
guard's merge rule (`internal/core/guard/merge.go`) judges `gh pr merge`, the
GraphQL merge mutations and the REST merge route by the mode alone, so under
`auto` an agent's own `gh pr merge --auto` of a pull request that changes the
mode line, or of a colleague's pull request under a person's global `auto`,
passes the guard though the gate would refuse it. Closing that needs the guard
to know which pull request a command selects and what that pull request
changes, which it does not read today; recorded as larger than the change that
built the gate's half (the decision that planned this spec).

## Footprint

- packages: internal/core/landing/land.go, internal/core/landing/depends.go, internal/core/landing/gate_ownership_test.go, internal/core/guard/merge.go, internal/core/guard/merge_test.go, internal/core/guard/speculate.go, internal/core/guard/defaults/guard.json, internal/surface/cli/guard.go, internal/surface/cli/guard_merge_test.go, internal/core/landing/facilitator.go, commands/guard.md, .abcd/development/brief/04-surfaces/17-guard.md, .abcd/development/brief/02-constraints/03-invariants.md
- tests: internal/core/guard, internal/surface/cli, internal/core/landing

## Steps

1. The guard refuses a merge command for a pull request the gate would never arm
   - packages: internal/core/guard, internal/surface/cli, internal/core/landing, commands/guard.md, .abcd/development/brief
   - tests: internal/core/guard, internal/surface/cli
   - Hand the guard a second reading beside `guard.MergeMode`, from the front
     door as the mode is handed in (`loadRepoGuard` in
     `internal/surface/cli/guard.go`): for the pull request a merge command
     selects, whether its diff changes the committed `facilitator.mode` line
     and, under a person's global `auto`, whether the account abcd runs as
     opened it, answered by the gate's own readers (`modeLineChange`,
     `readViewer`) so the guard and the gate cannot disagree.
   - Read the selected pull request from each merge form the guard knows: `gh
     pr merge` by number, URL, branch or the current branch; the GraphQL
     mutations by node id; the REST merge route by number. A selection the
     line does not fix, a pull request the forge cannot be asked for, or a head
     the checkout cannot read is refused, as an unreadable mode is today.
   - Refuse in any mode, naming the reason the gate gives: the person merges a
     change to the mode line on the forge's page; a colleague's pull request
     waits for an approval or for its author to land it.
   - Write each refusal's test first, with a `known_bad` line in the registry
     for each form, and hold the work-per-byte bar (`work_test.go`).
   - Then say in invariant 28 that the guard holds both rules, not the gate
     alone.
2. abcd takes its own arming back off a pull request that changes the mode line
   - packages: internal/core/landing, commands/land.md, .abcd/development/brief
   - tests: internal/core/landing
   - Where the gate refuses a pull request for changing the committed
     `facilitator.mode` line and this account's arming or queue entry stands at
     its head, `abcd land`, the build loop and a re-arm take it off
     (`withdrawOwn`, as the dependency check does) and say so; an arming
     another account made is left and named.
   - Write the test first: an arming of this account's at such a head is
     withdrawn, another account's is not.
