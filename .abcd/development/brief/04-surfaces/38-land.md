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

1. **Gate.** When a preflight receipt names the head in any worktree of the
   repository, the lookup the pre-push hook makes, the gate is skipped.
   Otherwise `make preflight` runs in the branch's checkout. It is run
   directly, never wrapped in a second load wait, because the target already
   waits on the machine's load waiter. After it the head must be
   unchanged and a receipt must name it. A repository whose make file declares
   no `preflight` target has no gate that mints a receipt, and gets no push.
   The receipt is a convenience check, not a security boundary: CI re-runs
   every gate on the pushed commit.
2. **Push** (`land.push`). A plain `git push --porcelain origin
   refs/heads/<branch>:refs/heads/<branch>` from the branch's checkout, through
   the networked git runner, with the repository's hooks running: never forced,
   never `--no-verify`, never the default branch. The push of a branch other
   than the default is not a merge, so it does not wait on the facilitator.
3. **Enqueue** (`land.enqueue`). The branch's one open pull request is armed
   by the merge gate and then confirmed in the merge queue: a pull request the
   forge reports CLEAN and unqueued has its auto-merge toggled once, and any
   other state is named. With no pull request the step stops with the `gh pr
   create` command; the verb never opens one (ruling R2), and the next run
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

- The remote is `origin` alone, and both its fetch and its push URL must be
  the repository the forge client resolves for the checkout.
- A pull request from another repository is refused, and one whose head on
  the forge is not the landed head is not armed.
- The merge rule is read from the ruleset mirror at the merge base with
  `origin`'s default branch, never at the branch's tip, so the branch cannot
  change the rule it lands by.
- The forge client's and git's environments are scrubbed of `GIT_DIR` and
  `GIT_CONFIG_*`, as the networked git runner's are.
- Nothing is written to the repository's or the queue's settings.

## Configuration

`land.push` and `land.enqueue` are booleans, both true by default, read
through the layered configuration reader from `.abcd/config.json` and
`~/.abcd.noindex/config.json`
([`../05-internals/03-configuration.md`](../05-internals/03-configuration.md)).
The precedence is deliberately asymmetric: off wins from any layer, so the
machine's file can switch an outward act off for every repository and no
repository's file switches it back on. A flag turns either act off for one
run (the appendix lists them); no flag turns one on. A step that is off names the layer and
file that turned it off.

## Exits

Exit 0 when every step did its act or said why it stopped or left the merge
to a person. Exit 2 on a refusal, which names the steps done before it. The
JSON document carries `branch`, `head`, `checkout`, `default_branch`, `push`
and `enqueue` (each `key`, `on`, `layer`, `origin`), `pull_request` and
`pull_request_url` when one was found, and `steps`, each with `name`, `status`
(`done`, `skipped`, `off`, `stopped` or `left`), `note` and an optional
`command`.

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
