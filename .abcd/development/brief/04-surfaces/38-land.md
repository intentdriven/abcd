# `/abcd:land` — Land a Branch Whose Preflight Passed

`/abcd:land` is one landing path for any branch whose work is committed, not
only a build lane's: it gates the branch's head on a preflight receipt, pushes
the branch to `origin`, and queues its open pull request where the merge gate
allows (iss-2610150524235929, iss-2610150524246898, and the queue confirmation
of iss-2610090642376032 for this verb). Before it, a branch landed outside the
implement loop was pushed only if someone remembered to, and ready pull
requests reached the merge queue one at a time. It is listed in the
agents-and-hosts block, and its page is hidden from a person's `/abcd:` menu.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|

## The steps

The verb takes at most one operand, the branch, and lands the checkout's
current branch without one. A named branch is landed from the worktree that
has it checked out, because the preflight runs in the branch's own checkout.
Every step is idempotent, so running the verb again resumes where it stopped.
The preflight is the branch's own `make preflight`, run with the person's
environment and credentials, so the verb is for branches whose content the
person trusts. With `land.push` and `land.enqueue` both off nothing runs. With
`land.push` off and `land.enqueue` on the gate still runs, by design: arming a
pull request says its head is ready to merge, and readiness is the gate.

Before the steps, the origin and the forge's default branch are checked (see
the trust boundaries), and then `origin`'s default branch is fetched. A head
already on it has landed (the implement loop's own ancestor check,
`landing.Landed`). A squash or rebase merge leaves no such ancestry, so when
the ancestor check says no the forge is asked for a merged pull request from
this repository's branch whose head was the head and whose base was the
default branch (the landing package's merged-pull-request check); one there has landed too. Either way the verb reports one step, `land`, with
status `landed` (naming the pull request the forge listed), gating, pushing
and queueing nothing, so a re-run after the merge never recreates the branch
the merge deleted, whichever method the queue merged by. A pull request merged
at another head, from another repository, or into another base (a stacked
pull request, whose work is not on the default branch) proves nothing, and a
forge that cannot be asked is a refusal: no gate and no push. So is a listing
its bound (20) filled with none at the head, since the proof may lie past the
bound: the refusal says so and gives the push to run by hand.

1. **Gate.** When a preflight receipt names the head in any worktree of the
   repository, the lookup the pre-push hook makes, the gate is skipped.
   Otherwise `make preflight` runs in the branch's checkout. It is run
   directly, never wrapped in a second load wait, because the target already
   waits on the machine's load waiter. After it the head must be
   unchanged, a receipt must name it, and `origin`'s fetch and push URLs must
   read exactly as they were checked: the preflight is the branch's own and
   could add a push URL or a `pushInsteadOf`. A repository whose make file declares
   no `preflight` target has no gate that mints a receipt, and gets no push.
   The receipt is a convenience check, not a security boundary: CI re-runs
   every gate on the pushed commit.
2. **Push** (`land.push`). A plain `git push --porcelain origin
   <head>:refs/heads/<branch>` from the branch's checkout, through the
   networked git runner, with the repository's hooks running: never forced,
   never `--no-verify`, never the default branch. The source is the commit the
   gate checked, never the branch's ref and never its configured upstream, so a
   branch that moves after the gate cannot change what is pushed. The push of a branch other
   than the default is not a merge, so it does not wait on the facilitator.
3. **Enqueue** (`land.enqueue`). The branch's one open pull request from this
   repository is armed by the merge gate, pinned to the head
   (`--match-head-commit`), and then confirmed in the merge queue at that
   head, through the forge's GraphQL API (the forge client's pull-request
   view has no merge-queue field), asked of the repository the forge client
   named when `origin` was checked: a pull request the forge reports CLEAN and unqueued has the auto-merge
   land made toggled once, and any other state is named. An arming someone
   else made is left as it is and reported, and a toggle whose re-arming fails
   says the pull request is now disarmed and gives the command that arms it.
   A re-run never toggles an arming again (its own earlier one included, and
   never one land did not make), so a pull request still CLEAN and unqueued is
   reported with the person's way on: wait for the forge, or withdraw and
   re-arm it with the exact `gh pr merge` commands the note gives. With no
   pull request the step stops with the `gh pr create` command; the verb never opens one (ruling R2), and the next run
   queues it.

## The merge gate

Whether the verb arms a merge is decided in one function, `mergeGate`, the
seat of the facilitator mode draft itd-97 describes (auto: queue at once;
human: the merge waits for the facilitator), which replaces it when that
toggle is built (ruling R4). Until then every repository runs in human mode,
and in human mode ruling AM1 ([`34-build.md`](34-build.md)) is the behaviour:
auto-merge is armed only where a merge queue gates the default branch and a
ruleset requires a person's approval. abcd and every managed repository take
this one code path (ruling R3).

## Trust boundaries

- The remote is `origin` alone, and its every fetch and push URL must be the
  repository the forge client resolves for the checkout. An SSH host alias (a
  host of one label, no dot) is refused by name, with the `insteadOf` rule
  that maps it in the URL's own form (scp-like or `ssh://`): the verb does not
  read ssh config. A URL on another real host is refused as a different host,
  and the only advice is `git remote set-url`, never a rule that reroutes that
  host's traffic. The URLs are read again after the preflight, and a change
  is refused.
- The default branch the verb never pushes is the forge's own, asked in the
  same call, and it must match the one `origin/HEAD` names locally, which can
  be stale; a forge that cannot be asked gets no gate and no push.
- A pull request from another repository is never armed (it is set aside
  when this repository's own is listed beside it), and one whose head on the
  forge is not the landed head is not armed.
- The merge rule is read from the ruleset mirror at `origin`'s default branch
  as just fetched, never at the branch's tip: equally out of the branch's
  reach as the merge base, and fresher, so the branch cannot change the rule
  it lands by. (The implement loop reads it at the lane's base.)
- The forge client's and git's environments are scrubbed of `GIT_DIR` and
  `GIT_CONFIG_*`, as the networked git runner's are. A local git command (the
  read of `origin`'s URLs) runs through `gitutil.LocalGit`, which adds none of the
  networked runner's settings.
- A credential in a URL that git or the forge client quotes
  (`scheme://user:token@`, `https://token@`) is dropped from every note and
  refusal.
- Nothing is written to the repository's or the queue's settings.

## Configuration

`land.push` and `land.enqueue` are booleans, both true by default, read
through the layered configuration reader from `.abcd/config.json` and
`~/.abcd.noindex/config.json`
([`../05-internals/03-configuration.md`](../05-internals/03-configuration.md)).
The repository layer is read in the working directory's checkout and in the
landed branch's own, and both count. The precedence is deliberately
asymmetric: off wins from any layer, so the machine's file can switch an
outward act off for every repository and no repository's file switches it
back on. A home the reader refuses hides the machine's file, and with it any
off there, so both acts read as off and say why. A flag turns either act off for one
run (the appendix lists them); no flag turns one on. A step that is off names the layer and
file that turned it off.

## Exits

Exit 0 when every step did its act or said why it stopped or left the merge
to a person. Exit 2 on a refusal, which names the steps done before it. The
JSON document carries `branch`, `head`, `checkout`, `default_branch`, `push`
and `enqueue` (each `key`, `on`, `layer`, `origin`), `pull_request` and
`pull_request_url` when one was found (or the forge proved the head merged,
the merged pull request's),
`pushed` when the push sent a commit, `landed` when the head had already
landed, and `steps`, each
with `name`, `status` (`done`, `skipped`, `off`, `stopped`, `left` or
`landed`), `note` and an optional `command`.

## Where this sits

- The records: iss-2610150524235929, iss-2610150524246898,
  iss-2610090642376032.
- The shared primitives: `internal/core/landing`, which the implement loop's
  land stage also reads its receipt lookup, forge client and ruleset mirror
  through.
- The plugin surface: `commands/land.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd land`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--no-enqueue` | bool |
| `--no-push` | bool |

<!-- surface-appendix:end -->
