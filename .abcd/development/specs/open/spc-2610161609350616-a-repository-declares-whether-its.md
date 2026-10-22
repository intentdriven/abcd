---
id: spc-2610161609350616
slug: a-repository-declares-whether-its
intent: itd-2610150602547271
origin: researcher-authored
production_mode: hand-written
---
# The facilitator's mode gates the merge

## Summary

Delivers itd-2610150602547271, the first slice of itd-97. A repository declares
`facilitator.mode` (`human` or `auto`) in `.abcd/config.json`, `human` by
default and `human` from any layer winning. The bare `abcd` board names the
mode and its layer. The merge gate (`mergeGate` in `internal/core/landing`,
today ruling AM1 as a stand-in) reads the mode for `abcd land` and the build
loop alike:
- `human` arms nothing, and a person merges;
- `auto` arms at once by the ruleset mirror's method, but only a head that a
  review receipt names.

While the mode is `human`, the shell guard refuses the merge commands. abcd's
own repository sets `auto`. Every behaviour is test-first, and each forge call
is checked live and read-only before it is relied on: the forge's fields are
pinned in `internal/ghcontract`, and the fake forges refuse what the real one
refuses.

## Footprint

- packages: internal/core/landing/config.go, internal/core/landing/facilitator.go, internal/core/landing/facilitator_test.go, internal/core/landing/review.go, internal/core/landing/review_test.go, internal/core/landing/primitives.go, internal/core/landing/land.go, internal/core/board/board.go, internal/core/board/board_test.go, internal/core/guard/merge.go, internal/core/guard/merge_test.go, internal/core/guard/guard.go, internal/surface/cli/cli.go, internal/surface/cli/board.go, internal/surface/cli/board_status.go, internal/surface/cli/board_status_test.go, internal/surface/cli/land.go, internal/surface/cli/land_surface_test.go, internal/surface/cli/receipts.go, internal/surface/cli/guard.go, .abcd/config.json, commands/land.md, commands/guard.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/17-guard.md, .abcd/development/brief/04-surfaces/39-status.md
- tests: internal/core/landing, internal/core/implement/loop, internal/core/board, internal/core/guard, internal/surface/cli
- The packages line names the files the steps change, not their directories,
  so the reservation does not hold unrelated work in the same packages.
  Generated output (`docs/reference/cli/commands.md`,
  `.abcd/development/release/surface.json` and any other linguist-generated
  file) is reserved by no one: whoever lands second regenerates it.
- Step 4 also changes `internal/core/implement/loop/land.go` and its test.
  They join the packages line when step 4 begins, and `abcd build` then
  reserves the widened footprint (dcn-2610222356532638).

## Steps

1. The declaration: `facilitator.mode` read through `internal/core/layered`
   - packages: internal/core/landing
   - tests: internal/core/landing
   - Claim the `facilitator` namespace with the one key `mode`.
   - The values are `human` and `auto`; any other value, in any layer, is
     refused loudly, naming the file.
   - Bundled default `human`; `human` from any layer wins, read from every
     layer by `Lookup` and `Decode`, as land's switches are.
   - Report the layer that decided.
2. The board line
   - packages: internal/core/board, internal/surface/cli
   - tests: internal/core/board, internal/surface/cli
   - The bare `abcd` board names the mode and the layer that set it.
3. Review receipts
   - packages: internal/core/landing, internal/surface/cli, commands/land.md
   - tests: internal/core/landing, internal/surface/cli
   - Write one receipt per commit, named by its full sha, in the local tier
     beside the preflight receipts. It is found from any worktree the way
     `landing.Receipt` finds those.
   - Write it only when the review pair's verdicts for that commit pass the
     stop rule (no critical or major finding).
   - The front door that records it is a decision to make and record in this
     step, as a sub-verb or a flag on an existing verb, wired to the CLI and
     the plugin page both.
4. The merge gate reads the mode
   - packages: internal/core/landing, internal/core/implement/loop
   - tests: internal/core/landing, internal/core/implement/loop
   - `mergeGate` becomes the one function `abcd land` and the loop's
     `landArm` ask.
   - `human` arms nothing and says the pull request waits for the
     facilitator.
   - `auto` arms by the mirror's method when a review receipt names the head,
     and otherwise arms nothing and names the commit with no recorded review
     pair.
   - With no queue gating the default branch, `auto` arms auto-merge, and
     where the forge refuses that, it says so.
   - AM1's approval reading is retired.
5. The guard's hard rule
   - packages: internal/core/guard
   - tests: internal/core/guard
   - While the mode is `human`, `abcd guard` refuses `gh pr merge --auto`,
     `gh pr merge` aimed at a queue, and a GraphQL `enqueuePullRequest`.
   - It reads the mode through the same resolver.
   - It names the setting that would allow each command.
6. abcd's own repository and the docs
   - packages: .abcd/config.json, commands/land.md, .abcd/development/brief/04-surfaces/38-land.md
   - tests: internal/surface/cli
   - Set `facilitator.mode: auto` in abcd's committed config, in the same
     change as step 4, so its sessions keep queueing.
   - Make the docs say exactly what each mode does.
