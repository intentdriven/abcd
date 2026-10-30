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
- a contributor's pull request (its author has Read or Triage, or no,
  permission on the repository) arms only once a maintainer's approval of
  its head counts, in either mode;
- `human` arms a maintainer's pull request only once the technical
  facilitator has approved its head, and says the approval is needed until
  then;
- `auto` arms a maintainer's pull request at once by the ruleset mirror's
  method, but only a head that a review receipt names.

While the mode is `human`, the shell guard refuses the merge commands. While
it is `auto`, the status line says merging is autonomous. abcd's own
repository sets `auto`. Every behaviour is test-first, and each forge call
is checked live and read-only before it is relied on: the forge's fields are
pinned in `internal/ghcontract`, and the fake forges refuse what the real one
refuses.

## Footprint

- packages: internal/core/landing/config.go, internal/core/landing/config_test.go, internal/core/layered/layered.go, internal/core/layered/layered_test.go, internal/core/layered/repoat_test.go, internal/core/board/board.go, internal/core/board/board_test.go, internal/surface/cli/cli.go, internal/surface/cli/board.go, internal/surface/cli/board_status.go, internal/surface/cli/board_status_test.go, internal/core/landing/review.go, internal/core/landing/review_test.go, internal/core/landing/primitives.go, internal/surface/cli/receipts.go, internal/core/landing/facilitator.go, internal/core/landing/facilitator_test.go, internal/core/landing/land.go, internal/core/landing/land_test.go, internal/core/landing/ruleset.go, internal/core/implement/loop/land.go, internal/core/implement/loop/land_test.go, internal/core/implement/loop/land_approval_test.go, internal/core/implement/loop/hold_approval_test.go, internal/core/implement/loop/drain_finish_test.go, internal/core/implement/loop/ghcontract_test.go, internal/core/implement/loop/issue_test.go, internal/core/implement/loop/parallel_test.go, internal/core/implement/loop/parked_reserve_test.go, internal/ghcontract/rest.go, internal/ghcontract/rest-fields.txt, internal/ghcontract/ghcontract_test.go, internal/surface/cli/land.go, internal/surface/cli/land_surface_test.go, internal/core/guard/merge.go, internal/core/guard/merge_test.go, internal/core/guard/guard.go, internal/core/guard/speculate.go, internal/core/guard/unknownsites_test.go, internal/core/guard/defaults/guard.json, internal/surface/cli/guard.go, internal/surface/cli/guard_approve.go, internal/surface/cli/guard_merge_test.go, .abcd/config.json, commands/land.md, commands/guard.md, commands/build.md, commands/implement.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/17-guard.md, .abcd/development/brief/04-surfaces/27-implement.md, .abcd/development/brief/04-surfaces/34-build.md, .abcd/development/brief/04-surfaces/39-status.md, internal/core/statusline/compose.go, internal/core/statusline/render.go, internal/core/statusline/defaults/statusline.json, internal/core/statusline/merging_test.go, internal/gitutil/defaultref.go, internal/gitutil/rangebase.go, internal/surface/cli/statusline.go, internal/surface/cli/statusline_merging_test.go, commands/abcd.md, .abcd/development/brief/04-surfaces/26-mode.md
- tests: internal/core/landing, internal/core/layered, internal/core/implement/loop, internal/core/board, internal/core/guard, internal/core/statusline, internal/ghcontract, internal/gitutil, internal/surface/cli
- The packages line names the files the steps change, not their directories,
  so the reservation does not hold unrelated work in the same packages. It
  names every file steps 1 to 7 changed, as each was built, the ones a step
  reached beyond its own packages line included (step 2's
  `internal/surface/cli/cli.go`, step 4's loop and `internal/ghcontract`
  files, step 7's `internal/core/statusline`, `internal/core/layered` and
  `internal/gitutil` files). Generated output (`docs/reference/cli/commands.md`,
  `.abcd/development/release/surface.json` and any other linguist-generated
  file) is reserved by no one: whoever lands second regenerates it.

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
   - needs: 1, 3
   - `mergeGate` becomes the one function `abcd land` and the loop's
     `landArm` ask.
   - Who is a maintainer is one seam: the pull request author's permission
     on the repository as the forge reports it (`role_name`). Write,
     Maintain and Admin are maintainers; Read and Triage are contributors; a
     custom role counts as its base role; a 404 or no role is a
     contributor. A permission that cannot be read, or a custom role whose
     base cannot be read, is unverified: the gate arms nothing and says so.
     Pin the fields in `internal/ghcontract` after a live read-only call.
   - An approval counts only when it is the latest review of a maintainer
     other than the author and approves the head commit, or when the
     technical facilitator has recorded their approval of the head commit
     through abcd's approval act. That act is a front door to decide and
     record in this step (a sub-verb or a flag of `abcd land`), wired to the
     CLI and the plugin page, writing a record for the full head sha in the
     local tier beside the review receipts; the shell guard refuses it to an
     agent (step 5). AM1 is retired as
     the stand-in, but this reading of approvals is kept for the two cases
     below.
   - A contributor's pull request arms nothing, in either mode, until a
     maintainer's approval counts, and says a maintainer's approval is
     needed.
   - `human` arms a maintainer's pull request by the mirror's method once
     the technical facilitator's approval counts, and until then arms
     nothing and says the pull request waits for that approval.
   - `auto` arms by the mirror's method when a review receipt names the head,
     and otherwise arms nothing and names the commit with no recorded review
     pair.
   - With no queue gating the default branch, an armed merge is auto-merge,
     and where the forge refuses that, it says so.
   - abcd adds no forge approval rule or code-owner file.
   - The mode is read from the default branch's committed
     `.abcd/config.json` and the machine layer, never from the checkout of
     the branch being landed, so a branch that commits `auto`, or deletes a
     repository-level `human`, does not decide the gate for itself
     (dcn-2610230058333177). Tests: a lane whose own config says `auto` over a
     default branch at `human` arms nothing; a lane that deletes the
     default branch's `human` arms nothing.
   - A `facilitator.mode` refusal is passed through `termsafe.Sanitize`
     wherever a front door prints it: the board (step 2), `abcd land` and
     the loop (here), the guard (step 5). Test: a hostile key carrying
     terminal control bytes prints escaped.
5. The guard's hard rule
   - packages: internal/core/guard
   - tests: internal/core/guard
   - needs: 1, 4
   - While the mode is `human`, `abcd guard` refuses `gh pr merge --auto`,
     `gh pr merge` aimed at a queue, and a GraphQL `enqueuePullRequest`.
   - It reads the mode through the same resolver.
   - It names the setting that would allow each command.
   - In any mode it refuses the approval act of step 4 when an agent runs
     it: approving is the person's act.
6. abcd's own repository and the docs
   - packages: .abcd/config.json, commands/land.md, .abcd/development/brief/04-surfaces/38-land.md
   - tests: internal/surface/cli
   - needs: 4, 5
   - Set `facilitator.mode: auto` in abcd's committed config, in the same
     change as step 4, so its sessions keep queueing.
   - Make the docs say exactly what each mode does, and who approves.
7. The status line
   - packages: internal/core/statusline/compose.go, internal/core/statusline/render.go, internal/core/statusline/defaults/statusline.json, internal/core/statusline/merging_test.go, internal/core/landing/config.go, internal/core/layered/layered.go, internal/gitutil/defaultref.go, internal/gitutil/rangebase.go, internal/surface/cli/statusline.go, internal/surface/cli/statusline_merging_test.go, commands/abcd.md, .abcd/development/brief/04-surfaces/26-mode.md
   - tests: internal/core/statusline, internal/core/landing, internal/core/layered, internal/gitutil, internal/surface/cli
   - needs: 1
   - While the mode is `auto`, abcd's status-line row says merging is
     autonomous; under `human` the row has no such element. It is switchable
     in `statusline.json` like the row's other elements.
   - It reads the mode through the same resolver, asks the network nothing
     and stays inside the status line's time budget; a refusal drops the
     element and is said on stderr, as the row's other elements do.
   - It reaches the host only through the status line the person consented
     to; nothing else in the host's settings calls abcd.
